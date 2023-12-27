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

// Code generated by protoc-gen-connect-go. DO NOT EDIT.
//
// Source: buf/knittest/services.proto

package knittestconnect

import (
	connect "connectrpc.com/connect"
	context "context"
	errors "errors"
	knittest "github.com/bufbuild/knit-go/internal/gen/buf/knittest"
	http "net/http"
	strings "strings"
)

// This is a compile-time assertion to ensure that this generated file and the connect package are
// compatible. If you get a compiler error that this constant is not defined, this code was
// generated with a version of connect newer than the one compiled into your binary. You can fix the
// problem by either regenerating this code with an older version of connect or updating the connect
// version compiled into your binary.
const _ = connect.IsAtLeastVersion1_13_0

const (
	// FooServiceName is the fully-qualified name of the FooService service.
	FooServiceName = "buf.knittest.FooService"
	// FizzServiceName is the fully-qualified name of the FizzService service.
	FizzServiceName = "buf.knittest.FizzService"
)

// These constants are the fully-qualified names of the RPCs defined in this package. They're
// exposed at runtime as Spec.Procedure and as the final two segments of the HTTP route.
//
// Note that these are different from the fully-qualified method names used by
// google.golang.org/protobuf/reflect/protoreflect. To convert from these constants to
// reflection-formatted method names, remove the leading slash and convert the remaining slash to a
// period.
const (
	// FooServiceGetFooProcedure is the fully-qualified name of the FooService's GetFoo RPC.
	FooServiceGetFooProcedure = "/buf.knittest.FooService/GetFoo"
	// FooServiceMutateFooProcedure is the fully-qualified name of the FooService's MutateFoo RPC.
	FooServiceMutateFooProcedure = "/buf.knittest.FooService/MutateFoo"
	// FooServiceQueryFoosProcedure is the fully-qualified name of the FooService's QueryFoos RPC.
	FooServiceQueryFoosProcedure = "/buf.knittest.FooService/QueryFoos"
	// FizzServiceGetFizzProcedure is the fully-qualified name of the FizzService's GetFizz RPC.
	FizzServiceGetFizzProcedure = "/buf.knittest.FizzService/GetFizz"
)

// These variables are the protoreflect.Descriptor objects for the RPCs defined in this package.
var (
	fooServiceServiceDescriptor         = knittest.File_buf_knittest_services_proto.Services().ByName("FooService")
	fooServiceGetFooMethodDescriptor    = fooServiceServiceDescriptor.Methods().ByName("GetFoo")
	fooServiceMutateFooMethodDescriptor = fooServiceServiceDescriptor.Methods().ByName("MutateFoo")
	fooServiceQueryFoosMethodDescriptor = fooServiceServiceDescriptor.Methods().ByName("QueryFoos")
	fizzServiceServiceDescriptor        = knittest.File_buf_knittest_services_proto.Services().ByName("FizzService")
	fizzServiceGetFizzMethodDescriptor  = fizzServiceServiceDescriptor.Methods().ByName("GetFizz")
)

// FooServiceClient is a client for the buf.knittest.FooService service.
type FooServiceClient interface {
	GetFoo(context.Context, *connect.Request[knittest.GetFooRequest]) (*connect.Response[knittest.GetFooResponse], error)
	MutateFoo(context.Context, *connect.Request[knittest.MutateFooRequest]) (*connect.Response[knittest.MutateFooResponse], error)
	QueryFoos(context.Context, *connect.Request[knittest.QueryFoosRequest]) (*connect.ServerStreamForClient[knittest.QueryFoosResponse], error)
}

// NewFooServiceClient constructs a client for the buf.knittest.FooService service. By default, it
// uses the Connect protocol with the binary Protobuf Codec, asks for gzipped responses, and sends
// uncompressed requests. To use the gRPC or gRPC-Web protocols, supply the connect.WithGRPC() or
// connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewFooServiceClient(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption) FooServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &fooServiceClient{
		getFoo: connect.NewClient[knittest.GetFooRequest, knittest.GetFooResponse](
			httpClient,
			baseURL+FooServiceGetFooProcedure,
			connect.WithSchema(fooServiceGetFooMethodDescriptor),
			connect.WithIdempotency(connect.IdempotencyNoSideEffects),
			connect.WithClientOptions(opts...),
		),
		mutateFoo: connect.NewClient[knittest.MutateFooRequest, knittest.MutateFooResponse](
			httpClient,
			baseURL+FooServiceMutateFooProcedure,
			connect.WithSchema(fooServiceMutateFooMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
		queryFoos: connect.NewClient[knittest.QueryFoosRequest, knittest.QueryFoosResponse](
			httpClient,
			baseURL+FooServiceQueryFoosProcedure,
			connect.WithSchema(fooServiceQueryFoosMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
	}
}

// fooServiceClient implements FooServiceClient.
type fooServiceClient struct {
	getFoo    *connect.Client[knittest.GetFooRequest, knittest.GetFooResponse]
	mutateFoo *connect.Client[knittest.MutateFooRequest, knittest.MutateFooResponse]
	queryFoos *connect.Client[knittest.QueryFoosRequest, knittest.QueryFoosResponse]
}

// GetFoo calls buf.knittest.FooService.GetFoo.
func (c *fooServiceClient) GetFoo(ctx context.Context, req *connect.Request[knittest.GetFooRequest]) (*connect.Response[knittest.GetFooResponse], error) {
	return c.getFoo.CallUnary(ctx, req)
}

// MutateFoo calls buf.knittest.FooService.MutateFoo.
func (c *fooServiceClient) MutateFoo(ctx context.Context, req *connect.Request[knittest.MutateFooRequest]) (*connect.Response[knittest.MutateFooResponse], error) {
	return c.mutateFoo.CallUnary(ctx, req)
}

// QueryFoos calls buf.knittest.FooService.QueryFoos.
func (c *fooServiceClient) QueryFoos(ctx context.Context, req *connect.Request[knittest.QueryFoosRequest]) (*connect.ServerStreamForClient[knittest.QueryFoosResponse], error) {
	return c.queryFoos.CallServerStream(ctx, req)
}

// FooServiceHandler is an implementation of the buf.knittest.FooService service.
type FooServiceHandler interface {
	GetFoo(context.Context, *connect.Request[knittest.GetFooRequest]) (*connect.Response[knittest.GetFooResponse], error)
	MutateFoo(context.Context, *connect.Request[knittest.MutateFooRequest]) (*connect.Response[knittest.MutateFooResponse], error)
	QueryFoos(context.Context, *connect.Request[knittest.QueryFoosRequest], *connect.ServerStream[knittest.QueryFoosResponse]) error
}

// NewFooServiceHandler builds an HTTP handler from the service implementation. It returns the path
// on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewFooServiceHandler(svc FooServiceHandler, opts ...connect.HandlerOption) (string, http.Handler) {
	fooServiceGetFooHandler := connect.NewUnaryHandler(
		FooServiceGetFooProcedure,
		svc.GetFoo,
		connect.WithSchema(fooServiceGetFooMethodDescriptor),
		connect.WithIdempotency(connect.IdempotencyNoSideEffects),
		connect.WithHandlerOptions(opts...),
	)
	fooServiceMutateFooHandler := connect.NewUnaryHandler(
		FooServiceMutateFooProcedure,
		svc.MutateFoo,
		connect.WithSchema(fooServiceMutateFooMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	fooServiceQueryFoosHandler := connect.NewServerStreamHandler(
		FooServiceQueryFoosProcedure,
		svc.QueryFoos,
		connect.WithSchema(fooServiceQueryFoosMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	return "/buf.knittest.FooService/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case FooServiceGetFooProcedure:
			fooServiceGetFooHandler.ServeHTTP(w, r)
		case FooServiceMutateFooProcedure:
			fooServiceMutateFooHandler.ServeHTTP(w, r)
		case FooServiceQueryFoosProcedure:
			fooServiceQueryFoosHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

// UnimplementedFooServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedFooServiceHandler struct{}

func (UnimplementedFooServiceHandler) GetFoo(context.Context, *connect.Request[knittest.GetFooRequest]) (*connect.Response[knittest.GetFooResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.knittest.FooService.GetFoo is not implemented"))
}

func (UnimplementedFooServiceHandler) MutateFoo(context.Context, *connect.Request[knittest.MutateFooRequest]) (*connect.Response[knittest.MutateFooResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.knittest.FooService.MutateFoo is not implemented"))
}

func (UnimplementedFooServiceHandler) QueryFoos(context.Context, *connect.Request[knittest.QueryFoosRequest], *connect.ServerStream[knittest.QueryFoosResponse]) error {
	return connect.NewError(connect.CodeUnimplemented, errors.New("buf.knittest.FooService.QueryFoos is not implemented"))
}

// FizzServiceClient is a client for the buf.knittest.FizzService service.
type FizzServiceClient interface {
	GetFizz(context.Context, *connect.Request[knittest.GetFizzRequest]) (*connect.Response[knittest.GetFizzResponse], error)
}

// NewFizzServiceClient constructs a client for the buf.knittest.FizzService service. By default, it
// uses the Connect protocol with the binary Protobuf Codec, asks for gzipped responses, and sends
// uncompressed requests. To use the gRPC or gRPC-Web protocols, supply the connect.WithGRPC() or
// connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewFizzServiceClient(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption) FizzServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &fizzServiceClient{
		getFizz: connect.NewClient[knittest.GetFizzRequest, knittest.GetFizzResponse](
			httpClient,
			baseURL+FizzServiceGetFizzProcedure,
			connect.WithSchema(fizzServiceGetFizzMethodDescriptor),
			connect.WithIdempotency(connect.IdempotencyNoSideEffects),
			connect.WithClientOptions(opts...),
		),
	}
}

// fizzServiceClient implements FizzServiceClient.
type fizzServiceClient struct {
	getFizz *connect.Client[knittest.GetFizzRequest, knittest.GetFizzResponse]
}

// GetFizz calls buf.knittest.FizzService.GetFizz.
func (c *fizzServiceClient) GetFizz(ctx context.Context, req *connect.Request[knittest.GetFizzRequest]) (*connect.Response[knittest.GetFizzResponse], error) {
	return c.getFizz.CallUnary(ctx, req)
}

// FizzServiceHandler is an implementation of the buf.knittest.FizzService service.
type FizzServiceHandler interface {
	GetFizz(context.Context, *connect.Request[knittest.GetFizzRequest]) (*connect.Response[knittest.GetFizzResponse], error)
}

// NewFizzServiceHandler builds an HTTP handler from the service implementation. It returns the path
// on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewFizzServiceHandler(svc FizzServiceHandler, opts ...connect.HandlerOption) (string, http.Handler) {
	fizzServiceGetFizzHandler := connect.NewUnaryHandler(
		FizzServiceGetFizzProcedure,
		svc.GetFizz,
		connect.WithSchema(fizzServiceGetFizzMethodDescriptor),
		connect.WithIdempotency(connect.IdempotencyNoSideEffects),
		connect.WithHandlerOptions(opts...),
	)
	return "/buf.knittest.FizzService/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case FizzServiceGetFizzProcedure:
			fizzServiceGetFizzHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

// UnimplementedFizzServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedFizzServiceHandler struct{}

func (UnimplementedFizzServiceHandler) GetFizz(context.Context, *connect.Request[knittest.GetFizzRequest]) (*connect.Response[knittest.GetFizzResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.knittest.FizzService.GetFizz is not implemented"))
}
