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

	connect "github.com/bufbuild/connect-go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

const knitOperationsHeader = "Knit-Operations"

type relationResolver func(context.Context, *resolveMeta, []proto.Message, proto.Message) ([]protoreflect.Value, error)

type resolveMeta struct {
	operations []string
	headers    http.Header
}

func (r *resolveMeta) toHeaders(hdrs http.Header) {
	for k, vv := range r.headers {
		if _, ok := hdrs[k]; ok {
			// header already set
			continue
		}
		hdrs[k] = vv
	}
	hdrs[knitOperationsHeader] = r.operations
}

type relationConfig struct {
	name        string
	method      protoreflect.MethodDescriptor
	descriptor  protoreflect.FieldDescriptor
	requestType protoreflect.MessageType
	baseField   protoreflect.FieldDescriptor
	resolver    relationResolver
}

func getRelationConfig(method protoreflect.MethodDescriptor, name string, client *connect.Client[dynamicpb.Message, deferredMessage]) (*relationConfig, error) {
	if method.IsStreamingClient() || method.IsStreamingServer() {
		return nil, fmt.Errorf("method %s is not a valid resolver: it must be unary, not streamining", method.FullName())
	}
	methodOptions, _ := method.Options().(*descriptorpb.MethodOptions)
	if methodOptions.GetIdempotencyLevel() != descriptorpb.MethodOptions_NO_SIDE_EFFECTS {
		return nil, fmt.Errorf("method %s is not a valid resolver: it must have no side effects, but idempotency level is instead %s",
			method.FullName(), methodOptions.GetIdempotencyLevel())
	}

	requestMsg := method.Input()
	requestFields := requestMsg.Fields()
	if requestFields.Len() < 1 {
		return nil, fmt.Errorf("request message should have at least one field for the base entities")
	}
	baseField := requestFields.ByNumber(1)
	switch {
	case baseField == nil:
		return nil, fmt.Errorf("request message should have a field with tag number 1 for the base entities")
	case baseField.Name() != "bases":
		return nil, fmt.Errorf("request message should have a field named 'bases' for the base entities")
	case !baseField.IsList():
		return nil, fmt.Errorf("request message field named 'bases' should be repeated (to accept a batch of entities to resolve)")
	case baseField.Kind() != protoreflect.MessageKind:
		return nil, fmt.Errorf("request message field named 'bases' should have element type that is a message")
	}

	responseMsg := method.Output()
	responseFields := responseMsg.Fields()
	if responseFields.Len() != 1 {
		return nil, fmt.Errorf("response message should have exactly one field for the resolved relation values")
	}
	resultField := responseFields.ByNumber(1)
	switch {
	case resultField == nil:
		return nil, fmt.Errorf("response message should have a field with tag number 1 for the resolved relation values")
	case resultField.Name() != "values":
		return nil, fmt.Errorf("response message should have a field named 'values' for the resolved relation values")
	case !resultField.IsList():
		return nil, fmt.Errorf("response message field named 'values' should be repeated (one value for each requested base)")
	case resultField.Kind() != protoreflect.MessageKind:
		return nil, fmt.Errorf("response message field named 'values' should have element type that is a single-field message")
	}
	relationWrapperMsg := resultField.Message()
	relationWrapperFields := relationWrapperMsg.Fields()
	if relationWrapperFields.Len() != 1 {
		return nil, fmt.Errorf("message %s should have exactly one field for the resolved relation value", relationWrapperMsg.Name())
	}
	relationField := relationWrapperFields.ByNumber(1)
	switch {
	case relationField == nil:
		return nil, fmt.Errorf("message %s should have a field with tag number 1 for the resolved relation value", relationWrapperMsg.Name())
	case relationField.Name() != protoreflect.Name(name):
		return nil, fmt.Errorf("message %s field named %s should have same name as relation: %s", relationWrapperMsg.Name(), relationField.Name(), name)
	}

	cfg := &relationConfig{
		name:        name,
		method:      method,
		descriptor:  relationField,
		requestType: dynamicpb.NewMessageType(requestMsg),
		baseField:   baseField,
	}
	cfg.resolver = newRemoteResolver(cfg, resultField, dynamicpb.NewMessageType(responseMsg), client)
	return cfg, nil
}

func newRemoteResolver(
	relation *relationConfig,
	resultField protoreflect.FieldDescriptor,
	responseType protoreflect.MessageType,
	client *connect.Client[dynamicpb.Message, deferredMessage],
) relationResolver {
	return func(ctx context.Context, resMeta *resolveMeta, bases []proto.Message, params proto.Message) ([]protoreflect.Value, error) {
		clone := proto.Clone(params)
		request, ok := clone.(*dynamicpb.Message)
		if !ok {
			return nil, fmt.Errorf("cloning request resulting value of type %T instead of %T", clone, request)
		}
		requestRef := request.ProtoReflect()
		basesList := requestRef.NewField(relation.baseField).List()
		for _, base := range bases {
			basesList.Append(protoreflect.ValueOfMessage(base.ProtoReflect()))
		}
		requestRef.Set(relation.baseField, protoreflect.ValueOfList(basesList))

		connReq := connect.NewRequest(request)
		resMeta.toHeaders(connReq.Header())
		connResp, err := client.CallUnary(ctx, connReq)
		if err != nil {
			return nil, fmt.Errorf("resolving relation %q of %q: %w",
				camelCase(relation.name), relation.baseField.Message().FullName(), err)
		}
		respRef := responseType.New()
		resp := respRef.Interface()
		if err := connResp.Msg.unmarshal(resp); err != nil {
			return nil, fmt.Errorf("resolving relation %q of %q: failed to unmarshal response: %w",
				camelCase(relation.name), relation.baseField.Message().FullName(), err)
		}

		valuesList := respRef.Get(resultField).List()
		length := valuesList.Len()
		if length != len(bases) {
			return nil, fmt.Errorf("resolving relation %q of %q: wrong number of results returned: expecting %d, fot %d",
				camelCase(relation.name), relation.baseField.Message().FullName(), len(bases), length)
		}
		results := make([]protoreflect.Value, length)
		for i := 0; i < length; i++ {
			results[i] = valuesList.Get(i).Message().Get(relation.descriptor)
		}

		return results, nil
	}
}
