// Copyright 2023-2025 Buf Technologies, Inc.
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
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"connectrpc.com/connect"
	knit "github.com/bufbuild/knit-go"
	"github.com/bufbuild/prototransform"
	"github.com/rs/cors"
	"golang.org/x/net/http2"
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

	defaultPollingPeriod  = 15 * time.Minute
	defaultStartupMaxWait = 15 * time.Second
)

// GatewayConfig is the configuration for a Knit gateway.
type GatewayConfig struct {
	ListenAddress            string
	UnixSocket               string
	TLSConfig                *tls.Config
	CORSConfig               cors.Options
	Services                 map[string]ServiceConfig
	MaxParallelismPerRequest int
	StartupMaxWait           time.Duration
	PollingPeriod            time.Duration
	PollingJitter            float64
	PollingDebounce          time.Duration
	Cache                    DescriptorCacheConfig
	Admin                    *AdminConfig
	RawConfig                []byte
}

// ServiceConfig is the configuration for a single RPC service.
type ServiceConfig struct {
	BaseURL     *url.URL
	HTTPClient  connect.HTTPClient
	ConnectOpts []connect.ClientOption
	Descriptors prototransform.SchemaPoller
	Cacheable   bool
}

// AdminConfig is the configuration for the gateway's admin endpoints.
type AdminConfig struct {
	// If true, admin endpoints are served via the same listeners/ports
	// as other traffic. If false, the following two properties are
	// used to construct a separate listener.
	UseMainListeners bool
	ListenAddress    string
	UseTLS           bool
}

// LoadConfig reads the config file at the given path and returns the resulting
// GatewayConfig on success.
func LoadConfig(path string) (*GatewayConfig, error) { //nolint:gocyclo
	rawConfig, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	dec := yaml.NewDecoder(bytes.NewReader(rawConfig))
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

	if len(extConf.Backends) == 0 {
		return nil, errors.New("backends config is empty")
	}

	adminConf, err := loadAdminConfig(&extConf)
	if err != nil {
		return nil, err
	}

	pollers := map[descriptorSource]prototransform.SchemaPoller{}
	numSourcesCacheable := 0
	backendConfigIndexByService := map[string]int{}
	services := map[string]ServiceConfig{}

	serverTLS, err := configureServerTLS(extConf.Listen.TLS)
	if err != nil {
		return nil, fmt.Errorf("listen.tls: invalid configuration: %w", err)
	}
	// we use this to validate the default_tls config (and can discard the result)
	_, err = configureClientTLS(extConf.DefaultBackendTLS)
	if err != nil {
		return nil, fmt.Errorf("default_tls: invalid configuration: %w", err)
	}
	if extConf.DefaultBackendTLS != nil && extConf.DefaultBackendTLS.ServerName != "" {
		return nil, errors.New("default_tls: invalid configuration: server_name property cannot be used in default settings, only in per-backend config")
	}

	for i, backendConf := range extConf.Backends {
		if backendConf.RouteTo == "" {
			return nil, fmt.Errorf("backend config #%d: missing 'route_to' property", i+1)
		}
		if len(backendConf.Services) == 0 {
			return nil, fmt.Errorf("backend config #%d: 'services' property should not be empty", i+1)
		}
		routeURL, err := url.Parse(backendConf.RouteTo)
		if err != nil {
			return nil, fmt.Errorf("backend config #%d: %q is not a valid URL: %w", i+1, backendConf.RouteTo, err)
		}
		if routeURL.Scheme != httpScheme && routeURL.Scheme != httpsScheme {
			return nil, fmt.Errorf("backend config #%d: %q is not a valid URL: schema should be '%s' or '%s'",
				i+1, backendConf.RouteTo, httpScheme, httpsScheme)
		}
		options := []connect.ClientOption{
			connect.WithInterceptors(connect.UnaryInterceptorFunc(UserAgentInterceptor)),
		}
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
			clientTLS, err = configureClientTLS(mergeClientTLSConfig(backendConf.TLS, extConf.DefaultBackendTLS))
			if err != nil {
				return nil, fmt.Errorf("backend config #%d: tls: invalid configuration: %w", i+1, err)
			}
			if clientTLS != nil && backendConf.TLS != nil && backendConf.TLS.ServerName != "" {
				clientTLS.ServerName = backendConf.TLS.ServerName
			}
		}

		httpClient, err := makeHTTPClient(clientTLS, backendConf.UnixSocket, h2c)
		if err != nil {
			return nil, fmt.Errorf("backend config #%d: failed to create HTTP client: %w", i+1, err)
		}
		svcConf := ServiceConfig{
			BaseURL:     routeURL,
			HTTPClient:  httpClient,
			ConnectOpts: options,
		}
		descSrc, err := newDescriptorSource(backendConf.Descriptors, backendConf.RouteTo, h2c)
		if err != nil {
			return nil, fmt.Errorf("backend config #%d: %w", i+1, err)
		}
		if descSrc.isCacheable() {
			numSourcesCacheable++
			svcConf.Cacheable = true
		}
		poller := pollers[descSrc]
		if poller == nil {
			poller, err = descSrc.newPoller(httpClient, options)
			if err != nil {
				return nil, fmt.Errorf("backend config #%d: %w", i+1, err)
			}
			pollers[descSrc] = poller
		}
		svcConf.Descriptors = poller

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
		listenAddress = fmt.Sprintf("%s:%d", extConf.Listen.BindAddress, *extConf.Listen.Port)
	}

	pollingPeriod := defaultPollingPeriod
	if extConf.Descriptors.PollingPeriodSeconds < 0 {
		return nil, fmt.Errorf("descriptors.polling_period_seconds may not be negative: %d", extConf.Descriptors.PollingPeriodSeconds)
	}
	if extConf.Descriptors.PollingPeriodSeconds > 0 {
		pollingPeriod = time.Second * time.Duration(extConf.Descriptors.PollingPeriodSeconds)
	}

	jitter := 0.25
	if extConf.Descriptors.PollingJitter != nil {
		jitter = *extConf.Descriptors.PollingJitter
		if jitter < 0 {
			return nil, fmt.Errorf("descriptors.polling_jitter may not be negative: %f", jitter)
		}
		if jitter > 1 {
			return nil, fmt.Errorf("descriptors.polling_jitter may not be greater than 1.0: %f", jitter)
		}
	}

	if extConf.Descriptors.PollingDebounceSeconds < 0 {
		return nil, fmt.Errorf("descriptors.polling_debounce_seconds may not be negative: %d", extConf.Descriptors.PollingDebounceSeconds)
	}

	startupMaxWait := defaultStartupMaxWait
	if extConf.Descriptors.StartupMaxWaitSeconds < 0 {
		return nil, fmt.Errorf("descriptors.startup_max_wait_seconds may not be negative: %d", extConf.Descriptors.StartupMaxWaitSeconds)
	}
	if extConf.Descriptors.StartupMaxWaitSeconds > 0 {
		startupMaxWait = time.Second * time.Duration(extConf.Descriptors.StartupMaxWaitSeconds)
	}

	cacheConf, err := newCacheConfig(extConf.Descriptors.Cache, numSourcesCacheable)
	if err != nil {
		return nil, err
	}

	corsConf := cors.Options{
		AllowedOrigins:      extConf.CORS.AllowedOrigins,
		AllowedHeaders:      extConf.CORS.AllowedHeaders,
		AllowCredentials:    extConf.CORS.AllowCredentials,
		AllowPrivateNetwork: extConf.CORS.AllowPrivateNetworks,
		MaxAge:              extConf.CORS.MaxAgeSeconds,
	}
	if len(corsConf.AllowedOrigins) == 0 {
		// If allowed origina is empty, the cors library defaults to allowing all '*'.
		// But we want users to opt into that behavior. So if the list is empty, we
		// provide a function that will be called (instead of using the empty list
		// of origins).
		corsConf.AllowOriginFunc = func(string) bool { return false }
	}

	return &GatewayConfig{
		ListenAddress:            listenAddress,
		UnixSocket:               extConf.Listen.UnixSocket,
		TLSConfig:                serverTLS,
		CORSConfig:               corsConf,
		Services:                 services,
		MaxParallelismPerRequest: extConf.Limits.PerRequestParallelism,
		StartupMaxWait:           startupMaxWait,
		PollingPeriod:            pollingPeriod,
		PollingJitter:            jitter,
		PollingDebounce:          time.Second * time.Duration(extConf.Descriptors.PollingDebounceSeconds),
		Cache:                    cacheConf,
		Admin:                    adminConf,
		RawConfig:                rawConfig,
	}, nil
}

type externalGatewayConfig struct {
	Listen            externalListenConfig            `yaml:"listen"`
	Backends          []externalBackendConfig         `yaml:"backends"`
	Limits            externalLimitsConfig            `yaml:"limits"`
	DefaultBackendTLS *externalClientTLSConfig        `yaml:"backend_tls"`
	CORS              externalCORSConfig              `yaml:"cors"`
	Descriptors       externalDescriptorPollingConfig `yaml:"descriptors"`
	Admin             externalAdminConfig             `yaml:"admin"`
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
	TLS *externalClientTLSConfig `yaml:"tls"`

	H2C         *bool                    `yaml:"h2c"`
	Protocol    string                   `yaml:"protocol"`
	Encoding    string                   `yaml:"encoding"`
	Descriptors externalDescriptorConfig `yaml:"descriptors"`
	Services    []string                 `yaml:"services"`
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
	Key  *string `yaml:"key"`

	// Override server name, for SNI.
	ServerName string `yaml:"server_name"`

	// !!NOTE: If any other fields are added, also update mergeClientTLSConfig in tls.go!!!
}

type externalDescriptorConfig struct {
	DescriptorSetFile string `yaml:"descriptor_set_file"`
	BufModule         string `yaml:"buf_module"`
	// TODO: Headers/authn for reflection requests.
	GRPCReflection bool `yaml:"grpc_reflection"`
}

type externalDescriptorPollingConfig struct {
	StartupMaxWaitSeconds  int                           `yaml:"startup_max_wait_seconds"`
	PollingPeriodSeconds   int                           `yaml:"polling_period_seconds"`
	PollingJitter          *float64                      `yaml:"polling_jitter"`
	PollingDebounceSeconds int                           `yaml:"polling_debounce_seconds"`
	Cache                  externalDescriptorCacheConfig `yaml:"cache"`
}

type externalLimitsConfig struct {
	PerRequestParallelism int `yaml:"per_request_parallelism"`
	// TODO: add other limits like message read size limits, resolver batch size limits,
	//       max total parallelism, max concurrent requests, rate limits, etc.
}

type externalCORSConfig struct {
	AllowedOrigins       []string `yaml:"allowed_origins"`
	AllowedHeaders       []string `yaml:"allowed_headers"`
	AllowCredentials     bool     `yaml:"allow_credentials"`
	AllowPrivateNetworks bool     `yaml:"allow_private_networks"`
	MaxAgeSeconds        int      `yaml:"max_age_seconds"`
}

type externalAdminConfig struct {
	Enabled *bool `yaml:"enabled"`
	// If true, the admin endpoints are exposed using the same listeners as used
	// for accepting requests for Knit traffic. If configured this way, caution
	// should be used to protect these endpoints from external access, such as
	// via ingress or reverse proxy that refuses to forward requests for all URI
	// paths that start with "/admin/".
	UseMainListeners bool `yaml:"use_main_listeners"`
	// The bind address for the listener used to serve admin endpoints. If absent
	// or empty string, defaults to 127.0.0.1 (loopback-only).
	BindAddress string `yaml:"bind_address"`
	// The port on which to listen for admin requests. If absent, defaults to
	// zero, which will pick an ephemeral port. In non-development environments,
	// this effectively makes the admin endpoints inaccessible.
	Port int `yaml:"port"`
	// If true, the TLS settings under "listen" will be applied to the admin port.
	// It is an error to configure this setting if the admin endpoints are served
	// from the same listeners as other traffic.
	//
	// If set to true, but "listen" section does not configure TLS, the config is
	// invalid. If not present, it defaults to true if TLS is configured in "listen"
	// but false otherwise. If false, then the admin port will use plaintext, even
	// if the "listen" section configures TLS for the main traffic port. If false,
	// it is recommended that "bin_address" be loopback-only.
	UseTLS *bool `yaml:"use_tls"`
}

func loadAdminConfig(extConf *externalGatewayConfig) (*AdminConfig, error) {
	adminEnabled := extConf.Admin.Enabled == nil || *extConf.Admin.Enabled
	if !adminEnabled {
		if extConf.Admin.UseMainListeners || extConf.Admin.BindAddress != "" ||
			extConf.Admin.Port != 0 || extConf.Admin.UseTLS != nil {
			return nil, errors.New("admin 'enabled' is false, but other properties for configuring admin endpoints also present")
		}
		return nil, nil //nolint:nilnil
	}

	var adminConf AdminConfig
	if extConf.Admin.UseMainListeners {
		adminConf.UseMainListeners = true
		if extConf.Admin.BindAddress != "" || extConf.Admin.Port != 0 || extConf.Admin.UseTLS != nil {
			return nil, errors.New("admin 'use_main_listeners' is true, but other properties for configuring separate listener are also present")
		}
		return &adminConf, nil
	}

	if extConf.Admin.UseTLS != nil && *extConf.Admin.UseTLS && extConf.Listen.TLS == nil {
		return nil, errors.New("admin config 'use_tls' is true, but no server TLS configuration present")
	}
	if extConf.Listen.Port != nil && extConf.Admin.Port == *extConf.Listen.Port {
		return nil, errors.New("admin port cannot be the same as main listen port (did you mean to instead set 'use_main_listeners' to true?)")
	}
	if extConf.Admin.UseTLS != nil {
		adminConf.UseTLS = *extConf.Admin.UseTLS
	} else {
		adminConf.UseTLS = extConf.Listen.TLS != nil
	}
	if extConf.Admin.BindAddress == "" {
		extConf.Admin.BindAddress = "127.0.0.1"
	}
	adminConf.ListenAddress = fmt.Sprintf("%s:%d", extConf.Admin.BindAddress, extConf.Admin.Port)

	return &adminConf, nil
}

func makeHTTPClient(tlsConfig *tls.Config, unixSocket string, h2c bool) (connect.HTTPClient, error) {
	if tlsConfig == nil && unixSocket == "" {
		// can use default clients
		if h2c {
			return defaultH2CClient, nil
		}
		return http.DefaultClient, nil
	}

	dial := defaultDialer.DialContext
	if unixSocket != "" {
		if err := checkUnixSocket(unixSocket); err != nil {
			return nil, errorHasFilename(err, unixSocket)
		}
		dial = func(ctx context.Context, _, _ string) (net.Conn, error) {
			return defaultDialer.DialContext(ctx, "unix", unixSocket)
		}
	}

	if h2c {
		// This is the same as used above in defaultH2CClient
		transport := &http2.Transport{
			AllowHTTP: true,
			DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
				return dial(ctx, network, addr)
			},
		}
		if tlsConfig != nil {
			transport.TLSClientConfig = tlsConfig
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
	if tlsConfig != nil {
		transport.TLSClientConfig = tlsConfig
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
