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
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"buf.build/gen/go/bufbuild/reflect/bufbuild/connect-go/buf/reflect/v1beta1/reflectv1beta1connect"
	"buf.build/gen/go/bufbuild/reflect/protocolbuffers/go/buf/reflect/v1beta1"
	"github.com/bufbuild/connect-go"
	"github.com/bufbuild/knit-go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"gopkg.in/yaml.v3"
)

const (
	httpScheme  = "http"
	httpsScheme = "https"

	connectProtocol = "connect"
	grpcProtocol    = "grpc"
	grpcWebProtocol = "grpcweb"

	protoEncoding = "proto"
	jsonEncoding  = "json"

	tlsVersion10 = "1.0"
	tlsVersion11 = "1.1"
	tlsVersion12 = "1.2"
	tlsVersion13 = "1.3"
)

// GatewayConfig is the configuration for a Knit gateway.
type GatewayConfig struct {
	ListenAddress string
	UnixSocket    string
	TLSConfig     *tls.Config
	Services      map[string]ServiceConfig
}

// ServiceConfig is the configuration for a single RPC service.
type ServiceConfig struct {
	BaseURL     *url.URL
	UnixSocket  string
	TLSConfig   *tls.Config
	ConnectOpts []connect.ClientOption
	H2C         bool
	Descriptors DescriptorSource
	Index       int
}

// LoadConfig reads the config file at the given path and returns the resulting
// GatewayConfig on success.
func LoadConfig(ctx context.Context, path string) (*GatewayConfig, error) {
	configInput, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = configInput.Close()
	}()
	dec := yaml.NewDecoder(configInput)
	dec.KnownFields(true)
	var extConf externalGatewayConfig
	if err := dec.Decode(&extConf); err != nil {
		return nil, err
	}
	var dummy any
	if err := dec.Decode(&dummy); !errors.Is(err, io.EOF) {
		return nil, errors.New("config file contains multiple documents, should only contain one")
	}

	if extConf.Listen.BindAddress == "" {
		extConf.Listen.BindAddress = "0.0.0.0"
	}
	if extConf.Listen.Port == nil && extConf.Listen.UnixSocket == "" {
		return nil, errors.New("listen config must include at least one of 'port' or 'unix_socket'")
	}

	descriptorSources := map[externalDescriptorConfig]DescriptorSource{}
	backendConfigIndexByService := map[string]int{}
	services := map[string]ServiceConfig{}

	serverTLS, err := configureServerTLS(extConf.Listen.TLS)
	if err != nil {
		return nil, fmt.Errorf("listen.tls: invalid configuration: %w", err)
	}
	// we use this to validate the default_tls config (and can discard the result)
	_, err = configureClientTLS(extConf.DefaultTLS)
	if err != nil {
		return nil, fmt.Errorf("default_tls: invalid configuration: %w", err)
	}

	for i, backendConf := range extConf.Backends {
		if backendConf.RouteTo == "" {
			return nil, fmt.Errorf("backend config #%d: missing 'route_to' property", i+1)
		}
		routeURL, err := url.Parse(backendConf.RouteTo)
		if err != nil {
			return nil, fmt.Errorf("backend config #%d: %q is not a valid URL: %w", i+1, backendConf.RouteTo, err)
		}
		if routeURL.Scheme != httpScheme && routeURL.Scheme != httpsScheme {
			return nil, fmt.Errorf("backend config #%d: %q is not a valid URL: schema should be '%s' or '%s'",
				i+1, backendConf.RouteTo, httpScheme, httpsScheme)
		}
		var options []connect.ClientOption
		switch backendConf.Encoding {
		case "", protoEncoding:
			// nothing to do; this is the default
		case jsonEncoding:
			options = append(options, knit.WithProtoJSON())
		default:
			return nil, fmt.Errorf("backend config #%d: %q is not a valid encoding; should be '%s' or '%s'",
				i+1, backendConf.Encoding, protoEncoding, jsonEncoding)
		}
		h2c := backendConf.H2C != nil && *backendConf.H2C
		if h2c && routeURL.Scheme == httpsScheme {
			return nil, fmt.Errorf("backend config #%d: 'h2c' is enabled, but URL scheme is %q; H2C is for plaintext connections", i+1, routeURL.Scheme)
		}
		switch backendConf.Protocol {
		case "", connectProtocol:
			// nothing to do; this is the default
		case grpcProtocol:
			options = append(options, connect.WithGRPC())
			// for grpc over plaintext, we must use H2C
			if backendConf.H2C == nil {
				h2c = routeURL.Scheme == httpScheme
			} else if !h2c {
				// H2C is explicitly disabled, which is not valid configuration
				return nil, fmt.Errorf("backend config #%d: cannot use %s protocol with %s schema w/out H2C",
					i+1, grpcProtocol, httpScheme)
			}
		case grpcWebProtocol:
			options = append(options, connect.WithGRPCWeb())
		default:
			return nil, fmt.Errorf("backend config #%d: %q is not a valid protocol; should be '%s', '%s', or '%s'",
				i+1, backendConf.Protocol, connectProtocol, grpcProtocol, grpcWebProtocol)
		}
		var clientTLS *tls.Config
		if routeURL.Scheme != httpsScheme {
			if backendConf.TLS != nil {
				return nil, fmt.Errorf("backend config #%d: tls stanza present but URL scheme is not https", i+1)
			}
		} else {
			clientTLS, err = configureClientTLS(mergeClientTLSConfig(&backendConf.TLS.externalClientTLSConfig, extConf.DefaultTLS))
			if err != nil {
				return nil, fmt.Errorf("backend config #%d: tls: invalid configuration: %w", i+1, err)
			}
			if clientTLS != nil && backendConf.TLS.ServerName != "" {
				clientTLS.ServerName = backendConf.TLS.ServerName
			}
		}

		svcConf := ServiceConfig{
			BaseURL:     routeURL,
			UnixSocket:  backendConf.UnixSocket,
			TLSConfig:   clientTLS,
			ConnectOpts: options,
			H2C:         h2c,
			Index:       i,
		}
		if !backendConf.Descriptors.GRPCReflection {
			svcConf.Descriptors = descriptorSources[backendConf.Descriptors]
		}
		if svcConf.Descriptors == nil {
			descSrc, err := newDescriptorSource(ctx, &svcConf, backendConf.Descriptors)
			if err != nil {
				return nil, fmt.Errorf("backend config #%d: %w", i+1, err)
			}
			svcConf.Descriptors = descSrc
			descriptorSources[backendConf.Descriptors] = descSrc
		}

		for _, svc := range backendConf.Services {
			if otherIndex, ok := backendConfigIndexByService[svc]; ok {
				return nil, fmt.Errorf("backend config #%d: service %q is already configured under backend config #%d", i+1, svc, otherIndex+1)
			}
			backendConfigIndexByService[svc] = i
			services[svc] = svcConf
		}
	}

	var listenAddress string
	if extConf.Listen.Port != nil {
		listenAddress = fmt.Sprintf("%s:%d", extConf.Listen.BindAddress, extConf.Listen.Port)
	}

	return &GatewayConfig{
		ListenAddress: listenAddress,
		UnixSocket:    extConf.Listen.UnixSocket,
		TLSConfig:     serverTLS,
		Services:      services,
	}, nil
}

type externalGatewayConfig struct {
	Listen     externalListenConfig     `yaml:"listen"`
	Backends   []externalBackendConfig  `yaml:"backends"`
	DefaultTLS *externalClientTLSConfig `yaml:"default_tls"`
}

type externalListenConfig struct {
	BindAddress string `yaml:"bind_address"`
	Port        *int   `yaml:"port"`
	// If present, gateway will listen for requests on unix domain socket with given path.
	// This is in addition to TCP listener (unless port is missing, in which case only the
	// unix socket listener is used).
	UnixSocket string `yaml:"unix_socket"`
	// If 'tls' section is present, clients must use TLS when connecting.
	TLS *externalServerTLSConfig `yaml:"tls"`
}

type externalServerTLSConfig struct {
	// Cert and Key are both required.
	Cert string `yaml:"cert"`
	Key  string `yaml:"key"`
	// If absent, defaults to "1.0". Allowed values are "1.0", "1.1", "1.2", and "1.3".
	MinVersion string `yaml:"min_version"`
	// If absent, a default set of secure cipher suites is used.
	Ciphers *externalCiphersConfig `yaml:"ciphers"`
	// If 'client_certs' key is absent, client TLS certs will be ignored and not validated.
	ClientCerts *externalClientCertsConfig `yaml:"client_certs"`
}

type externalCiphersConfig struct {
	// Exactly one of the following fields may be used, not both.

	// Enumerates the allowed cipher suite names.
	Allow []string `yaml:"allow"`
	// Enumerates the disallowed cipher suite names. The default cipher suites will be used
	// but items named here will be excluded.
	Disallow []string `yaml:"disallow"`
}

type externalClientCertsConfig struct {
	Require bool   `yaml:"require"`
	CACert  string `yaml:"cacert"`
}

type externalBackendConfig struct {
	RouteTo string `yaml:"route_to"`

	// If present, requests will be sent via unix socket instead of TCP connection.
	UnixSocket string `yaml:"unix_socket"`
	// Should not be present unless 'route_to' indicates "https" scheme. Only
	// required if client certs are to be used, if a custom CA is needed for
	// verifying backend cert, a custom server-name is needed (for SNI), or if
	// top-level 'default_tls' config needs to be overridden.
	TLS *externalBackendClientTLSConfig `yaml:"tls"`

	H2C         *bool                    `yaml:"h2c"`
	Protocol    string                   `yaml:"protocol"`
	Encoding    string                   `yaml:"encoding"`
	Descriptors externalDescriptorConfig `yaml:"descriptors"`
	Services    []string                 `yaml:"services"`
}

type externalBackendClientTLSConfig struct {
	externalClientTLSConfig
	// Override server name, for SNI.
	ServerName string `yaml:"server_name"`
}

type externalClientTLSConfig struct {
	// If absent, defaults to "1.2". Allowed values are "1.0", "1.1", "1.2", and "1.3".
	MinVersion *string `yaml:"min_version"`
	// If absent, a default set of secure cipher suites is used.
	Ciphers *externalCiphersConfig `yaml:"ciphers"`

	CACert *string `yaml:"cacert"`
	// If true, the backend's certificate is not verified. For testing only.
	// It is STRONGLY discouraged to use this flag in real production environments.
	// If present, any value for 'cacert' is ignored.
	SkipVerify *bool `yaml:"skip_verify"`

	// If one of the following is present, then both must be present. When present,
	// they will be used as the client certificated presented to the backend server
	// during TLS handshake.
	Cert *string `yaml:"cert"`
	Key  *string `yaml:"ley"`

	// !!NOTE: If any other fields are added, also update mergeClientTLSConfig in tls.go!!!
}

type externalDescriptorConfig struct {
	DescriptorSetFile string `yaml:"descriptor_set_file"`
	BufModule         string `yaml:"buf_module"`
	GRPCReflection    bool   `yaml:"grpc_reflection"`
}

func newDescriptorSource(ctx context.Context, svcConf *ServiceConfig, config externalDescriptorConfig) (DescriptorSource, error) {
	var properties []string
	if config.DescriptorSetFile != "" {
		properties = append(properties, "descriptor_set_file")
	}
	if config.GRPCReflection {
		properties = append(properties, "grpc_reflection")
	}
	if config.BufModule != "" {
		properties = append(properties, "buf_module")
	}
	if len(properties) > 1 {
		return nil, fmt.Errorf("descriptor config should have exactly one field set, instead got %d [%v]", len(properties), properties)
	}
	switch {
	case config.DescriptorSetFile != "":
		data, err := os.ReadFile(config.DescriptorSetFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load descriptor set %q: %w", config.DescriptorSetFile, err)
		}
		var files descriptorpb.FileDescriptorSet
		if err := proto.Unmarshal(data, &files); err != nil {
			return nil, fmt.Errorf("failed to unmarshal Descriptors from %s: %w", config.DescriptorSetFile, err)
		}
		descSrc, err := newFileDescriptorSetSource(&files)
		if err != nil {
			return nil, fmt.Errorf("failed to process Descriptors read from %s: %w", config.DescriptorSetFile, err)
		}
		return descSrc, nil

	case config.BufModule != "":
		parts := strings.Split(config.BufModule, "/")
		if len(parts) != 3 {
			return nil, fmt.Errorf("descriptor config should have exactly one field set, instead got %d [%v]", len(properties), properties)
		}
		envBufToken := os.Getenv("BUF_TOKEN")
		if envBufToken == "" {
			return nil, fmt.Errorf("cannot download module %q: no BUF_TOKEN environment variable set", config.BufModule)
		}
		tok := parseBufToken(envBufToken, parts[0])
		if tok == "" {
			return nil, fmt.Errorf("cannot download module %q: BUF_TOKEN environment variable did not include a token for remote %q", config.BufModule, parts[0])
		}
		req := connect.NewRequest(&reflectv1beta1.GetFileDescriptorSetRequest{
			Module: config.BufModule,
		})
		req.Header().Set("Authorization", fmt.Sprintf("Bearer %s", tok))
		reflectClient := reflectv1beta1connect.NewFileDescriptorSetServiceClient(http.DefaultClient, "https://api."+parts[0])
		resp, err := reflectClient.GetFileDescriptorSet(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("failed to download Descriptors from %s: %w", parts[0], err)
		}
		descSrc, err := newFileDescriptorSetSource(resp.Msg.FileDescriptorSet)
		if err != nil {
			return nil, fmt.Errorf("failed to process Descriptors downloaded from %s: %w", parts[0], err)
		}
		return descSrc, nil

	case config.GRPCReflection:
		if svcConf.BaseURL.Scheme == httpScheme && !svcConf.H2C {
			return nil, fmt.Errorf("cannot use grpc reflection with http scheme without H2C")
		}
		return &deferredGRPCDescriptorSource{
			ctx:     ctx,
			opts:    svcConf.ConnectOpts,
			baseURL: svcConf.BaseURL.String(),
		}, nil

	default:
		return nil, fmt.Errorf("descriptor config is empty")
	}
}

func parseBufToken(envVar, remote string) string {
	isMultiToken := strings.ContainsAny(envVar, "@,")
	if !isMultiToken {
		return envVar
	}
	tokenConfigs := strings.Split(envVar, ",")
	suffix := "@" + remote
	for _, tokenConfig := range tokenConfigs {
		token := strings.TrimSuffix(tokenConfig, suffix)
		if token == tokenConfig {
			// did not have the right suffix
			continue
		}
		return token
	}
	return ""
}
