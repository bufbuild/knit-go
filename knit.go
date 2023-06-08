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

package knit

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"buf.build/gen/go/bufbuild/knit/bufbuild/connect-go/buf/knit/gateway/v1alpha1/gatewayv1alpha1connect"
	knitv1alpha1 "buf.build/gen/go/bufbuild/knit/protocolbuffers/go/buf/knit/v1alpha1"
	connect "github.com/bufbuild/connect-go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
)

// Gateway is an HTTP API gateway that accepts requests in the Knit protocol
// and dispatches Connect RPCs to configured backends. It is not a simple
// reverse gateway, like some API gateways, as it handles batching and joining
// of multiple RPCs, grouping the results into a single HTTP response.
//
// Callers must use [Gateway.AddService] or [Gateway.AddServiceByName] to
// configure the supported services and routes. If any of the services added
// contain methods that resolve relations, then the gateway will support the
// corresponding relation in incoming Knit queries.
//
// Once configured, the gateway is thread-safe. But configuration methods
// are not thread-safe. So configuration should be done from a single
// goroutine before sharing the instance with other goroutines, which
// includes registering it with an [http.Server] via [Gateway.AsHandler].
type Gateway struct {
	// Client is the HTTP client to use for outbound Connect RPCs
	// for methods that do not have a custom client configured. If this
	// field is nil and a method has no custom client configured then
	// [http.Client] will be used.
	Client connect.HTTPClient
	// Route is the default base URL for routing outbound Connect RPCs.
	// It must have an "http" or "https" schema. If this field is nil.
	// nil, [WithRoute] must be used to supply routes during configuration.
	Route *url.URL
	// ClientOptions are the Connect client options used for all outbound
	// Connect RPCs. Note that use of [connect.WithCodec] is not allowed as
	// an option. To configure a custom codec, you must use [knit.WithCodec]
	// instead.
	ClientOptions []connect.ClientOption
	// TypeResolver is used to resolve extensions and types when unmarshalling
	// JSON-formatted messages. If not provided/nil, protoregistry.GlobalTypes
	// is used. This can be overridden per outbound service via a
	// [knit.WithTypeResolver] option. Even if overridden for a service,
	// this type resolver will be used to process resolved relations on
	// responses from said service.
	TypeResolver TypeResolver
	// MaxParallelismPerRequest is the maximum number of concurrent goroutines
	// that will be executing relation resolvers during the course of handling
	// a single call to the knit service. If a knit request requires more RPCs
	// than this for a single phase/depth of resolution, they will be queued.
	MaxParallelismPerRequest int

	// TODO: default size limit for batch resolver requests
	// TODO: bounds on query complexity?

	methods   map[protoreflect.FullName]*methodConfig
	relations map[protoreflect.FullName]map[string]*relationConfig
}

// AsHandler returns an HTTP handler for g as well as the URI path that the
// handler expects to handle. The returned values can be passed directly to
// an [*http.ServeMux]'s Handle method:
//
//	mux.Handle(gateway.AsHandler())
func (g *Gateway) AsHandler(handlerOptions ...connect.HandlerOption) (path string, h http.Handler) {
	return gatewayv1alpha1connect.NewKnitServiceHandler(&handler{gateway: g}, handlerOptions...)
}

// AddService adds the given service's methods as available outbound RPCs.
func (g *Gateway) AddService(svc protoreflect.ServiceDescriptor, opts ...ServiceOption) error {
	return g.addService(svc, false, opts)
}

// AddServiceByName adds the given service's methods as available outbound RPCs.
// This function uses [protoregistry.GlobalFiles] to resolve the name. So the given
// service name must be defined in a file whose corresponding generated Go code is
// linked into the current program.
func (g *Gateway) AddServiceByName(svcName protoreflect.FullName, opts ...ServiceOption) error {
	desc, err := protoregistry.GlobalFiles.FindDescriptorByName(svcName)
	if err != nil {
		return err
	}
	svcDesc, ok := desc.(protoreflect.ServiceDescriptor)
	if !ok {
		return fmt.Errorf("named element %q is a %s, not a service", svcName, descType(desc))
	}
	return g.addService(svcDesc, true, opts)
}

func (g *Gateway) getSvcOpts(svcOptions *svcOpts, opts []ServiceOption) (*svcOpts, error) {
	if svcOptions == nil {
		svcOptions = &svcOpts{
			baseURL:      g.Route,
			client:       g.Client,
			opts:         append([]connect.ClientOption{defaultCodec()}, g.ClientOptions...),
			typeResolver: g.TypeResolver,
		}
	}
	for _, opt := range opts {
		opt(svcOptions)
	}
	if svcOptions.client == nil {
		svcOptions.client = http.DefaultClient
	}
	if svcOptions.baseURL == nil {
		return nil, errors.New("no route configured")
	}
	return svcOptions, nil
}

func (g *Gateway) addService(svc protoreflect.ServiceDescriptor, knownMessageTypes bool, opts []ServiceOption) error {
	svcOptions, err := g.getSvcOpts(nil, opts)
	if err != nil {
		return err
	}
	if g.methods == nil {
		g.methods = map[protoreflect.FullName]*methodConfig{}
	}
	methods := svc.Methods()
	for i, length := 0, svc.Methods().Len(); i < length; i++ {
		method := methods.Get(i)
		methodName := method.FullName()
		if _, ok := g.methods[methodName]; ok {
			return fmt.Errorf("method %q has already been registered", methodName)
		}
		client := connect.NewClient[dynamicpb.Message, deferredMessage](svcOptions.client, endpointURL(svcOptions.baseURL, method), svcOptions.opts...)
		var responseType protoreflect.MessageType
		if knownMessageTypes {
			var err error
			responseType, err = protoregistry.GlobalTypes.FindMessageByName(method.Output().FullName())
			if err != nil {
				return fmt.Errorf("could not resolve message type %q (response for method %q): %w", method.Output().FullName(), method.FullName(), err)
			}
		} else {
			responseType = dynamicpb.NewMessageType(method.Output())
		}
		g.methods[methodName] = &methodConfig{
			descriptor:    method,
			responseType:  responseType,
			connectClient: client,
			typeResolver:  svcOptions.typeResolver,
		}
		relationDetails, _ := proto.GetExtension(method.Options(), knitv1alpha1.E_Relation).(*knitv1alpha1.RelationConfig)
		if relationDetails == nil {
			continue
		}
		if relationDetails.Name == "" {
			return fmt.Errorf("method %q is annotated as a relation resolver, but no relation name is configured", method.FullName())
		}
		relationConfig, err := getRelationConfig(method, relationDetails.Name, client)
		if err != nil {
			return err
		}
		if err := g.addRelation(relationConfig); err != nil {
			return err
		}
	}
	return nil
}

func (g *Gateway) addRelation(config *relationConfig) error {
	relationName := camelCase(config.name)
	baseMsg := config.baseField.Message()
	msgName := baseMsg.FullName()
	if field := fieldWithCamelCaseName(baseMsg.Fields(), relationName); field != nil {
		return fmt.Errorf("message %q has a field named %q (%d) so cannot also have a relation with same name",
			msgName, relationName, field.Number())
	}

	if g.relations == nil {
		g.relations = map[protoreflect.FullName]map[string]*relationConfig{}
	}
	relations := g.relations[msgName]
	if relations == nil {
		relations = map[string]*relationConfig{}
		g.relations[msgName] = relations
	}

	if cfg, ok := relations[relationName]; ok {
		return fmt.Errorf("relation named %q for message %q already defined by method %q",
			relationName, msgName, cfg.method.FullName())
	}

	relations[relationName] = config
	return nil
}

// TypeResolver is capable of resolving messages and extensions. This is needed
// for JSON serialization.
type TypeResolver interface {
	protoregistry.MessageTypeResolver
	protoregistry.ExtensionTypeResolver
}

// ServiceOption is an option for configuring an outbound service.
type ServiceOption func(*svcOpts)

// WithRoute is a service option that indicates the base URL that should be
// used when routing RPCs for a particular service.
func WithRoute(baseURL *url.URL) ServiceOption {
	return func(o *svcOpts) {
		o.baseURL = baseURL
	}
}

// WithClient is a service option that indicates the HTTP client that should
// be used when routing RPCs for a particular service.
func WithClient(client connect.HTTPClient) ServiceOption {
	return func(o *svcOpts) {
		o.client = client
	}
}

// WithClientOptions is a service option that indicates the Connect client options
// that should be used when routing RPCs for a particular service.
//
// Note: [connect.WithCodec] should not be used directly and provided to this
// function. Instead, use [knit.WithCodec] to configure custom codecs.
func WithClientOptions(opts ...connect.ClientOption) ServiceOption {
	return func(o *svcOpts) {
		o.opts = append(o.opts, opts...)
	}
}

// WithTypeResolver is a service option that indicates a resolver to use to resolve
// extensions and the contents of google.protobuf.Any messages when serializing and
// de-serializing requests and responses.
func WithTypeResolver(res TypeResolver) ServiceOption {
	return func(o *svcOpts) {
		o.typeResolver = res
	}
}

// WithCodec returns a Connect client option that can be used to customize codecs
// used in outbound RPCs from a [knit.Gateway]. Do not directly use [connect.WithCodec]
// for such configuration as that will result in errors when the gateway needs to
// unmarshal responses from downstream Connect servers.
func WithCodec(codec connect.Codec) connect.ClientOption {
	return connect.WithCodec(&dynamicCodec{codec: codec})
}

// WithProtoJSON returns a Connect client option that indicates that the JSON
// format should be used.
func WithProtoJSON() connect.ClientOption {
	return WithCodec(defaultJSONCodec{})
}

// TODO: We will also likely want options for the following:
//   * Ability to exclude relation resolver methods? (Separately, we'd want a way to
//     add a single method to the gateway, instead of always adding entire services.)
//   * Ability to configure batch size limits for relation resolver methods.
//   * Ability to configure multiple backends with weights (for traffic shaping, like
//     to support migrations).
//   * Ability to configure weights or other means for disambiguation/routing for
//     the case where the same relation is defined by more than one method.

type methodConfig struct {
	descriptor    protoreflect.MethodDescriptor
	responseType  protoreflect.MessageType
	connectClient *connect.Client[dynamicpb.Message, deferredMessage]
	typeResolver  TypeResolver
}

type svcOpts struct {
	baseURL      *url.URL
	client       connect.HTTPClient
	opts         []connect.ClientOption
	typeResolver TypeResolver
}

func endpointURL(baseURL *url.URL, method protoreflect.MethodDescriptor) string {
	baseURLstr := baseURL.String()
	var sep string
	if !strings.HasSuffix(baseURLstr, "/") {
		sep = "/"
	}
	return baseURLstr + sep + string(method.Parent().FullName()) + "/" + string(method.Name())
}

func descType(desc protoreflect.Descriptor) string {
	switch desc := desc.(type) {
	case protoreflect.FileDescriptor:
		return "file"
	case protoreflect.MessageDescriptor:
		return "message"
	case protoreflect.FieldDescriptor:
		if desc.IsExtension() {
			return "extension"
		}
		return "field"
	case protoreflect.OneofDescriptor:
		return "oneof"
	case protoreflect.EnumDescriptor:
		return "enum"
	case protoreflect.EnumValueDescriptor:
		return "enum value"
	case protoreflect.ServiceDescriptor:
		return "service"
	case protoreflect.MethodDescriptor:
		return "method"
	default:
		return fmt.Sprintf("%T", desc)
	}
}
