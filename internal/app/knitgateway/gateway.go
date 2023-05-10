// Copyright 2023 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package knitgateway

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/bufbuild/connect-go"
	"github.com/bufbuild/knit-go"
	"golang.org/x/net/http2"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	clientIdentityHeader = "Knit-Client-Subject"
)

//nolint:gochecknoglobals
var (
	// defaultH2CClient is like http.DefaultClient except that it will use HTTP/2 over plaintext
	// (aka "H2C") to send requests to servers.
	defaultH2CClient = &http.Client{
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, network, addr)
			},
		},
	}

	defaultDialer = &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}
)

// CreateGateway creates a Knit gateway with the given configuration.
func CreateGateway(config *GatewayConfig) (*knit.Gateway, error) {
	gateway := &knit.Gateway{}
	for svcName, svcConf := range config.Services {
		httpClient, err := makeHTTPClient(svcConf)
		if err != nil {
			return nil, fmt.Errorf("backend #%d: failed to create HTTP client: %w", svcConf.Index+1, err)
		}
		descSource := svcConf.Descriptors
		if deferredSrc, ok := descSource.(*deferredGRPCDescriptorSource); ok {
			descSource = deferredSrc.WithHTTPClient(httpClient)
		}

		desc, err := descSource.FindDescriptorByName(protoreflect.FullName(svcName))
		if err != nil {
			return nil, fmt.Errorf("could not get descriptor for service %s: %w", svcName, err)
		}
		svc, ok := desc.(protoreflect.ServiceDescriptor)
		if !ok {
			return nil, fmt.Errorf("%s is a %s, not a service", svcName, descriptorKind(desc))
		}
		opts := append(svcConf.ConnectOpts, connect.WithInterceptors(&certForwardingInterceptor{})) //nolint:gocritic
		err = gateway.AddService(
			svc,
			knit.WithRoute(svcConf.BaseURL),
			knit.WithClient(httpClient),
			knit.WithClientOptions(opts...),
			knit.WithTypeResolver(typeResolver{DescriptorSource: descSource}),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to register service %s with gateway: %w", svcName, err)
		}
	}
	return gateway, nil
}

type certForwardingInterceptor struct{}

func (c *certForwardingInterceptor) WrapUnary(unaryFunc connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if cert := clientCertFromContext(ctx); cert != nil {
			req.Header().Set(clientIdentityHeader, cert.Subject.String())
		}
		return unaryFunc(ctx, req)
	}
}

func (c *certForwardingInterceptor) WrapStreamingClient(clientFunc connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		stream := clientFunc(ctx, spec)
		if cert := clientCertFromContext(ctx); cert != nil {
			stream.RequestHeader().Set(clientIdentityHeader, cert.Subject.String())
		}
		return stream
	}
}

func (c *certForwardingInterceptor) WrapStreamingHandler(handlerFunc connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	// should never be called
	return handlerFunc
}

func makeHTTPClient(config ServiceConfig) (connect.HTTPClient, error) {
	if config.TLSConfig == nil && config.UnixSocket == "" {
		// can use default clients
		if config.H2C {
			return defaultH2CClient, nil
		}
		return http.DefaultClient, nil
	}

	dial := defaultDialer.DialContext
	if config.UnixSocket != "" {
		if err := checkUnixSocket(config.UnixSocket); err != nil {
			return nil, errorHasFilename(err, config.UnixSocket)
		}
		dial = func(ctx context.Context, _, _ string) (net.Conn, error) {
			return defaultDialer.DialContext(ctx, "unix", config.UnixSocket)
		}
	}

	if config.H2C {
		// This is the same as used above in defaultH2CClient
		transport := &http2.Transport{
			AllowHTTP: true,
			DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
				return dial(ctx, network, addr)
			},
		}
		if config.TLSConfig != nil {
			transport.TLSClientConfig = config.TLSConfig
		}
		return &http.Client{Transport: transport}, nil
	}

	// This is the same as http.DefaultTransport.
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dial,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	if config.TLSConfig != nil {
		transport.TLSClientConfig = config.TLSConfig
	}
	return &http.Client{Transport: transport}, nil
}

func checkUnixSocket(socketPath string) error {
	conn, err := defaultDialer.Dial("unix", socketPath)
	if err != nil {
		return err
	}
	return conn.Close()
}
