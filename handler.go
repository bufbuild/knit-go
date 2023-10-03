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
	"context"
	"fmt"
	"net/http"
	"strings"

	"buf.build/gen/go/bufbuild/knit/connectrpc/go/buf/knit/gateway/v1alpha1/gatewayv1alpha1connect"
	gatewayv1alpha1 "buf.build/gen/go/bufbuild/knit/protocolbuffers/go/buf/knit/gateway/v1alpha1"
	"connectrpc.com/connect"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/structpb"
)

// NB: these keys must all be lower-case.
//
//nolint:gochecknoglobals
var forbiddenHeaders = map[string]struct{}{
	"accept":            {},
	"connect":           {},
	"connection":        {},
	"expect":            {},
	"host":              {},
	"http2-settings":    {},
	"keep-alive":        {},
	"origin":            {},
	"proxy-connection":  {},
	"te":                {},
	"trailer":           {},
	"transfer-encoding": {},
	"upgrade":           {},
	// User-Agent is not forbidden, but we intentionally do NOT forward
	// it since the outbound requests are sufficiently distinct from
	// the original request. Instead, the gateway presents itself as
	// the user agent to RPC backends.
	"user-agent": {},
}

type handler struct {
	gatewayv1alpha1connect.UnimplementedKnitServiceHandler
	gateway *Gateway
}

var _ gatewayv1alpha1connect.KnitServiceHandler = (*handler)(nil)

func (h *handler) Fetch(ctx context.Context, req *connect.Request[gatewayv1alpha1.FetchRequest]) (*connect.Response[gatewayv1alpha1.FetchResponse], error) {
	resps, err := h.do(ctx, req.Msg.Requests, req, true)
	if err != nil {
		return nil, err
	}
	// TODO: caching headers (in case GET is used)
	return connect.NewResponse(&gatewayv1alpha1.FetchResponse{Responses: resps}), nil
}

func (h *handler) Do(ctx context.Context, req *connect.Request[gatewayv1alpha1.DoRequest]) (*connect.Response[gatewayv1alpha1.DoResponse], error) {
	resps, err := h.do(ctx, req.Msg.Requests, req, false)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&gatewayv1alpha1.DoResponse{Responses: resps}), nil
}

func (h *handler) do(ctx context.Context, requests []*gatewayv1alpha1.Request, inboundReq connect.AnyRequest, forFetch bool) ([]*gatewayv1alpha1.Response, error) {
	confs, schemas, err := computeResponseSchemas(h.gateway, requests, forFetch, rpcKindUnary)
	if err != nil {
		return nil, err
	}
	dynamicRequests := make([]*dynamicpb.Message, len(requests))
	for i := range requests {
		msg := dynamicpb.NewMessage(confs[i].descriptor.Input())
		if err := parseMessage(requests[i].Body, msg, confs[i].typeResolver); err != nil {
			return nil, fmt.Errorf("could not unmarshal request type %q for method %q: %w",
				msg.Descriptor().FullName(), confs[i].descriptor.FullName(), err)
		}
		dynamicRequests[i] = msg
	}
	responses := make([]*gatewayv1alpha1.Response, len(requests))
	errPatches := make([]*errorPatch, len(requests))
	allPatches := make([][]*patch, len(requests))
	origCtx := ctx
	group, ctx := errgroup.WithContext(ctx)
	stitcher := newStitcher(h.gateway)
	fallbackCatch := !forFetch
	for i := range requests {
		i := i
		req := requests[i]
		conf := confs[i]
		schema := schemas[i]
		dynamicRequest := dynamicRequests[i]
		group.Go(func() error {
			if err := stitcher.sema.Acquire(ctx); err != nil {
				return err
			}
			defer stitcher.sema.Release()

			methodName := conf.descriptor.FullName()
			outboundReq := connect.NewRequest(dynamicRequest)
			propagateHeaders(inboundReq.Header(), outboundReq.Header())
			catch := fallbackCatch
			switch req.OnError.(type) {
			case *gatewayv1alpha1.Request_Catch:
				catch = true
			case *gatewayv1alpha1.Request_Throw:
				catch = false
			}
			resp, err := conf.connectClient.CallUnary(ctx, outboundReq)
			if err != nil {
				if !catch {
					return fmt.Errorf("failed to invoke method %q: %w", methodName, err)
				}
				responses[i] = &gatewayv1alpha1.Response{
					Method: req.Method,
					Body:   formatError(err, string(methodName), conf.typeResolver),
					Schema: schema,
				}
				return nil
			}
			responseMessage := conf.responseType.New().Interface()
			if err := resp.Msg.unmarshal(responseMessage); err != nil {
				return fmt.Errorf("failed to unmarshal response from method %q: %w", methodName, err)
			}

			info := &resolveMeta{
				operations: []string{toMethodName(inboundReq.Spec().Procedure), toMethodName(outboundReq.Spec().Procedure)},
				headers:    http.Header{},
			}
			if catch {
				errPatches[i] = &errorPatch{
					formatTarget: &structpb.Struct{
						Fields: map[string]*structpb.Value{},
					},
					// Can be anything as it is set on the above formatTarget.
					name: "error",
				}
			}
			body, patches, err := formatMessage(responseMessage, info, req.Mask, []string{string(methodName)}, errPatches[i], fallbackCatch, conf.typeResolver)
			if err != nil {
				return fmt.Errorf("failed to format response from method %q: %w", methodName, err)
			}
			responses[i] = &gatewayv1alpha1.Response{
				Method: req.Method,
				Body:   body,
				Schema: schema,
			}
			if len(patches) > 0 {
				propagateHeaders(inboundReq.Header(), info.headers)
				allPatches[i] = patches
			}
			return nil
		})
	}
	err = group.Wait()
	if err != nil {
		return nil, err
	}
	if err := stitcher.stitch(origCtx, allPatches, fallbackCatch); err != nil {
		return nil, err
	}
	for i, errPatch := range errPatches {
		if errPatch == nil {
			continue
		}
		errBody, ok := errPatch.formatTarget.Fields[errPatch.name]
		if !ok {
			continue
		}
		responses[i].Body = errBody
	}
	return responses, err
}

func (h *handler) Listen(ctx context.Context, inboundReq *connect.Request[gatewayv1alpha1.ListenRequest], stream *connect.ServerStream[gatewayv1alpha1.ListenResponse]) error {
	conf, schema, err := computeResponseSchema(h.gateway, inboundReq.Msg.Request, false, rpcKindServerStream)
	if err != nil {
		return err
	}
	dynamicRequest := dynamicpb.NewMessage(conf.descriptor.Input())
	if err := parseMessage(inboundReq.Msg.Request.Body, dynamicRequest, conf.typeResolver); err != nil {
		return fmt.Errorf("could not unmarshal request type %q for method %q: %w",
			dynamicRequest.Descriptor().FullName(), conf.descriptor.FullName(), err)
	}
	methodName := conf.descriptor.FullName()
	outboundReq := connect.NewRequest(dynamicRequest)
	propagateHeaders(inboundReq.Header(), outboundReq.Header())
	outboundStream, err := conf.connectClient.CallServerStream(ctx, outboundReq)
	if err != nil {
		return fmt.Errorf("failed to invoke method %q: %w", methodName, err)
	}
	responseMessage := conf.responseType.New().Interface()
	request := inboundReq.Msg.Request
	method, mask := request.Method, request.Mask
	info := &resolveMeta{
		operations: []string{toMethodName(inboundReq.Spec().Procedure), toMethodName(outboundReq.Spec().Procedure)},
		headers:    http.Header{},
	}
	propagateHeaders(inboundReq.Header(), info.headers)
	stitcher := newStitcher(h.gateway)
	sentSchema := false
	var errPatch *errorPatch
	if request.GetCatch() != nil {
		errPatch = &errorPatch{
			formatTarget: &structpb.Struct{
				Fields: map[string]*structpb.Value{},
			},
			// Can be anything as it is set on the above formatTarget.
			name: "error",
		}
	}
	const fallbackCatch = false
	for outboundStream.Receive() {
		if err := outboundStream.Msg().unmarshal(responseMessage); err != nil {
			return fmt.Errorf("failed to unmarshal response from method %q: %w", methodName, err)
		}
		body, patches, err := formatMessage(responseMessage, info, mask, []string{string(methodName)}, errPatch, fallbackCatch, conf.typeResolver)
		if err != nil {
			return fmt.Errorf("failed to format response from method %q: %w", methodName, err)
		}
		if err := stitcher.stitch(ctx, [][]*patch{patches}, fallbackCatch); err != nil {
			return fmt.Errorf("failed to add relations to response from method %q: %w", methodName, err)
		}
		if errPatch != nil {
			if errBody, ok := errPatch.formatTarget.Fields[errPatch.name]; ok {
				body = errBody
			}
		}
		responseMessage := &gatewayv1alpha1.ListenResponse{
			Response: &gatewayv1alpha1.Response{
				Method: method,
				Body:   body,
			},
		}
		if !sentSchema {
			responseMessage.Response.Schema = schema
			sentSchema = true
			schema = nil // allow it to be GC'ed
		}
		err = stream.Send(responseMessage)
		if err != nil {
			return err
		}
	}
	if err := outboundStream.Err(); err != nil {
		return fmt.Errorf("received error from method %q: %w", methodName, err)
	}
	return nil
}

func propagateHeaders(src http.Header, dest http.Header) {
	for headerName, headerVals := range src {
		if len(dest[headerName]) > 0 {
			// don't overwrite headers that are already set
			// (shouldn't really be necessary since the exceptions
			// below should already exclude everything that might
			// already be set, but just in case...)
			continue
		}
		lowerName := strings.ToLower(headerName)
		if _, ok := forbiddenHeaders[lowerName]; ok {
			// skip these
			continue
		}
		switch {
		case strings.HasPrefix(lowerName, ":"),
			strings.HasPrefix(lowerName, "accept-"),
			strings.HasPrefix(lowerName, "connect-"),
			strings.HasPrefix(lowerName, "content-"),
			strings.HasPrefix(lowerName, "grpc-"),
			strings.HasPrefix(lowerName, "if-"):
			// skip these, too
			continue
		}
		dest[headerName] = headerVals
	}
	// TODO: Append to Forwarded, X-Forwarded-For, X-Forwarded-Host, X-Forwarded-Proto
	//       as is customary of gateway servers.
}

func toMethodName(procedureName string) string {
	return strings.ReplaceAll(strings.TrimLeft(procedureName, "/"), "/", ".")
}
