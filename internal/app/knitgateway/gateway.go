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

	"github.com/bufbuild/knit-go"
	"golang.org/x/net/http2"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// DefaultH2CClient is like http.DefaultClient except that it will use HTTP/2 over plaintext
// (aka "H2C") to send requests to servers.
var DefaultH2CClient = &http.Client{ //nolint:gochecknoglobals
	Transport: &http2.Transport{
		AllowHTTP: true,
		DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, network, addr)
		},
	},
}

// CreateGateway creates a Knit gateway with the given configuration.
func CreateGateway(config *GatewayConfig) (*knit.Gateway, error) {
	gateway := &knit.Gateway{}
	for svcName, svcConf := range config.Services {
		desc, err := svcConf.Descriptors.FindDescriptorByName(protoreflect.FullName(svcName))
		if err != nil {
			return nil, fmt.Errorf("could not get descriptor for service %s: %w", svcName, err)
		}
		svc, ok := desc.(protoreflect.ServiceDescriptor)
		if !ok {
			return nil, fmt.Errorf("%s is a %s, not a service", svcName, descriptorKind(desc))
		}
		httpClient := http.DefaultClient
		if svcConf.H2C {
			httpClient = DefaultH2CClient
		}
		err = gateway.AddService(
			svc,
			knit.WithRoute(svcConf.BaseURL),
			knit.WithClient(httpClient),
			knit.WithClientOptions(svcConf.ConnectOpts...),
			knit.WithTypeResolver(typeResolver{DescriptorSource: svcConf.Descriptors}),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to register service %s with gateway: %w", svcName, err)
		}
	}
	return gateway, nil
}
