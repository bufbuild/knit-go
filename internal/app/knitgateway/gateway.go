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
	"io"
	"math"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/bufbuild/connect-go"
	"github.com/bufbuild/knit-go"
	"github.com/bufbuild/prototransform"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	clientIdentityHeader = "Knit-Client-Subject"
)

//nolint:gochecknoglobals
var (
	defaultDialer = &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	// defaultH2CClient is like http.DefaultClient except that it will use HTTP/2 over plaintext
	// (aka "H2C") to send requests to servers.
	defaultH2CClient = &http.Client{
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
				return defaultDialer.DialContext(ctx, network, addr)
			},
		},
	}
)

// Gateway is a hot-swappable Knit gateway. It watches schemas for updates and
// then replaces the underlying HTTP handler with one that has new configuration
// when an update is observed.
type Gateway struct {
	watchers   map[prototransform.SchemaPoller]*prototransform.SchemaWatcher
	config     *GatewayConfig
	updateChan <-chan struct{}
	delegate   atomic.Pointer[handlerWrapper]
	pattern    string
}

// CreateGateway creates a new gateway with the given configuration. This
// starts background goroutines which will terminate when the given context
// is cancelled.
func CreateGateway(ctx context.Context, logger *zap.Logger, config *GatewayConfig) (g *Gateway, err error) {
	type pollerDetails struct {
		symbols   []string
		cacheable bool
	}
	pollerInfo := map[prototransform.SchemaPoller]*pollerDetails{}
	for svcName, svcConf := range config.Services {
		details := pollerInfo[svcConf.Descriptors]
		if details == nil {
			details = &pollerDetails{cacheable: svcConf.Cacheable}
			pollerInfo[svcConf.Descriptors] = details
		}
		details.symbols = append(details.symbols, svcName)
	}

	closers := make([]io.Closer, 0, len(pollerInfo)+1)
	closeAll := func() {
		for _, closer := range closers {
			_ = closer.Close()
		}
	}
	defer func() {
		// if we're returning an error, not the gateway,
		// then go ahead and shutdown down anything started
		if err != nil {
			closeAll()
		}
	}()

	var schemaCache prototransform.Cache
	if config.Cache != nil {
		var err error
		var closer io.Closer
		schemaCache, closer, err = config.Cache.toCache()
		if closer != nil {
			closers = append(closers, closer)
		}
		if err != nil {
			return nil, fmt.Errorf("could not create cache: %w", err)
		}
		schemaCache = &loggingCache{logger: logger, cache: schemaCache}
	}

	updateChan := make(chan struct{}, 1)
	gateway := &Gateway{
		watchers:   map[prototransform.SchemaPoller]*prototransform.SchemaWatcher{},
		config:     config,
		updateChan: updateChan,
	}
	period, jitter := config.PollingPeriod, config.PollingJitter
	for poller := range pollerInfo {
		poller := poller
		details := pollerInfo[poller]
		watcherConfig := &prototransform.SchemaWatcherConfig{
			SchemaPoller:   poller,
			IncludeSymbols: details.symbols,
			PollingPeriod:  period,
			Jitter:         jitter,
		}
		var sync func()
		if jitter != 0 {
			// Instead of letting everything jitter independently,
			// we let the first one handle the jittered scheduling
			// and then sync the rest to it.
			period = math.MaxInt64
			jitter = 0
			if len(pollerInfo) > 1 {
				sync = func() {
					// sync all other watchers to this one's schedule
					for i, watcher := range gateway.watchers {
						if i == poller {
							continue // don't reload ourself
						}
						watcher.ResolveNow()
					}
				}
			}
		}
		if details.cacheable {
			watcherConfig.Cache = schemaCache
		}
		watcherConfig.OnUpdate = func() {
			if sync != nil {
				sync()
			}
			logger.Info("received schema update", zap.String("source", poller.GetSchemaID()))
			if ctx.Err() != nil {
				return // don't bother sending update signal
			}
			select {
			case updateChan <- struct{}{}:
			default:
				// if updateChan buffer is full, update signal already pending
			}
		}
		watcherConfig.OnError = func(err error) {
			logger.Error("error updating schema",
				zap.String("source", poller.GetSchemaID()),
				zap.Error(err),
			)
		}
		watcher, err := prototransform.NewSchemaWatcher(ctx, watcherConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create schema watcher for %s: %w", poller.GetSchemaID(), err)
		}
		gateway.watchers[poller] = watcher
		closers = append(closers, closerFunc(func() error {
			watcher.Stop()
			return nil
		}))
	}

	go gateway.handleUpdates(ctx, logger)
	go func() {
		<-ctx.Done()
		// make sure everything is shutdown when context is cancelled
		closeAll()
	}()

	return gateway, nil
}

// AwaitReady returns nil when the gateway is ready and all schemas have been downloaded.
// It returns an error if the given context is cancelled or if the configured startup
// max wait period elapses.
func (g *Gateway) AwaitReady(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, g.config.StartupMaxWait)
	defer cancel()
	for _, watcher := range g.watchers {
		if err := watcher.AwaitReady(ctx); err != nil {
			return err
		}
	}
	return nil
}

// CreateHandler creates the underlying HTTP handler. This must be called explicitly
// once by the application. Thereafter, it is called automatically when schemas are
// updated.
func (g *Gateway) CreateHandler() error {
	knitGateway := &knit.Gateway{MaxParallelismPerRequest: g.config.MaxParallelismPerRequest}
	for svcName, svcConf := range g.config.Services {
		watcher := g.watchers[svcConf.Descriptors]
		desc, err := watcher.FindDescriptorByName(protoreflect.FullName(svcName))
		if err != nil {
			return fmt.Errorf("could not get descriptor for service %s: %w", svcName, err)
		}
		svc, ok := desc.(protoreflect.ServiceDescriptor)
		if !ok {
			return fmt.Errorf("%s is a %s, not a service", svcName, descriptorKind(desc))
		}
		opts := append(svcConf.ConnectOpts, connect.WithInterceptors(&certForwardingInterceptor{})) //nolint:gocritic
		err = knitGateway.AddService(
			svc,
			knit.WithRoute(svcConf.BaseURL),
			knit.WithClient(svcConf.HTTPClient),
			knit.WithClientOptions(opts...),
			// TODO: The watcher resolves the requested symbols and then answers all requests from
			// that. So if the initial set of symbols didn't transitively include all needed items
			// (like other messages used in Any messages or extensions), they won't be present.
			// Ideally, we could dynamically fetch add'l messages/extensions as needed. This isn't
			// too tricky with gRPC reflection, but the BSR reflection endpoint doesn't have a way
			// to resolve extensions by extendee+tag number.
			knit.WithTypeResolver(watcher),
		)
		if err != nil {
			return fmt.Errorf("failed to register service %s with knitGateway: %w", svcName, err)
		}
	}
	pattern, handler := knitGateway.AsHandler()
	newDelegate := &handlerWrapper{handler}
	for {
		existing := g.delegate.Load()
		if existing != nil {
			g.delegate.Store(newDelegate)
			return nil
		}
		// Make sure there are no pending update signals, otherwise it's possible that the
		// updater goroutine could think it needs to immediately re-create the handler after
		// we set this to non-nil.
		select {
		case <-g.updateChan:
		default:
		}
		if g.delegate.CompareAndSwap(nil, newDelegate) {
			// to avoid data race, we only set pattern the first time we initialize delegate
			g.pattern = pattern
			return nil
		}
	}
}

// Pattern returns the URI pattern that this gateeway handles. This
// method should only be invoked after first calling CreateHandler.
func (g *Gateway) Pattern() string {
	return g.pattern
}

// ServeHTTP implements http.Handler. If the gateway is not yet ready
// (schemas have not yet been successfully loaded; AwaitReady returns an
// error), this will return 503 Service Unavailable.
func (g *Gateway) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	handler := g.delegate.Load()
	if handler == nil {
		http.Error(responseWriter, "gateway not initialized", http.StatusServiceUnavailable)
		return
	}
	handler.ServeHTTP(responseWriter, request)
}

// handleUpdates handles updating the gateway's underlying configuration and
// associated HTTP handler when schema updates are observed.
func (g *Gateway) handleUpdates(ctx context.Context, logger *zap.Logger) {
	updateChan := g.updateChan
	debounce := g.config.PollingDebounce
	var timer *time.Timer
	for {
		select {
		case <-ctx.Done():
			return
		case <-updateChan:
			if g.delegate.Load() == nil {
				// handler not initialized yet
				continue
			}
			if debounce > 0 {
				if timer == nil {
					timer = time.NewTimer(debounce)
				} else {
					timer.Reset(debounce)
				}
				select {
				case <-ctx.Done():
					timer.Stop()
					return
				case <-timer.C:
					// clear any pending update signal that arrived
					// during debounce period
					select {
					case <-updateChan:
					default:
					}
				}
			}
			if err := g.CreateHandler(); err != nil {
				logger.Error("failed to create new handler for updated schemas", zap.Error(err))
			}
		}
	}
}

type closerFunc func() error

func (fn closerFunc) Close() error {
	return fn()
}

type handlerWrapper struct {
	http.Handler
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
