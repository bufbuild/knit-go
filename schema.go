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
	"fmt"
	"strings"

	gatewayv1alpha1 "buf.build/gen/go/bufbuild/knit/protocolbuffers/go/buf/knit/gateway/v1alpha1"
	connect "github.com/bufbuild/connect-go"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type rpcKind int

const (
	rpcKindUnary = rpcKind(iota)
	rpcKindClientStream
	rpcKindServerStream
	rpcKindBidiStream
)

func (k rpcKind) String() string {
	switch k {
	case rpcKindUnary:
		return "unary"
	case rpcKindClientStream:
		return "client stream"
	case rpcKindServerStream:
		return "server stream"
	case rpcKindBidiStream:
		return "bidi stream"
	default:
		return fmt.Sprintf("?(%d)", k)
	}
}

//nolint:gochecknoglobals
var (
	anyName         = (*anypb.Any)(nil).ProtoReflect().Descriptor().FullName()
	fieldMaskName   = (*fieldmaskpb.FieldMask)(nil).ProtoReflect().Descriptor().FullName()
	timestampName   = (*timestamppb.Timestamp)(nil).ProtoReflect().Descriptor().FullName()
	durationName    = (*durationpb.Duration)(nil).ProtoReflect().Descriptor().FullName()
	valueName       = (*structpb.Value)(nil).ProtoReflect().Descriptor().FullName()
	listValueName   = (*structpb.ListValue)(nil).ProtoReflect().Descriptor().FullName()
	structName      = (*structpb.Struct)(nil).ProtoReflect().Descriptor().FullName()
	nullValueName   = structpb.NullValue(0).Descriptor().FullName()
	boolValueName   = (*wrapperspb.BoolValue)(nil).ProtoReflect().Descriptor().FullName()
	int32ValueName  = (*wrapperspb.Int32Value)(nil).ProtoReflect().Descriptor().FullName()
	int64ValueName  = (*wrapperspb.Int64Value)(nil).ProtoReflect().Descriptor().FullName()
	uint32ValueName = (*wrapperspb.UInt32Value)(nil).ProtoReflect().Descriptor().FullName()
	uint64ValueName = (*wrapperspb.UInt64Value)(nil).ProtoReflect().Descriptor().FullName()
	floatValueName  = (*wrapperspb.FloatValue)(nil).ProtoReflect().Descriptor().FullName()
	doubleValueName = (*wrapperspb.DoubleValue)(nil).ProtoReflect().Descriptor().FullName()
	stringValueName = (*wrapperspb.StringValue)(nil).ProtoReflect().Descriptor().FullName()
	bytesValueName  = (*wrapperspb.BytesValue)(nil).ProtoReflect().Descriptor().FullName()

	wellKnownTypes = map[protoreflect.FullName]struct{}{
		anyName:         {},
		fieldMaskName:   {},
		timestampName:   {},
		durationName:    {},
		valueName:       {},
		listValueName:   {},
		structName:      {},
		boolValueName:   {},
		int32ValueName:  {},
		int64ValueName:  {},
		uint32ValueName: {},
		uint64ValueName: {},
		floatValueName:  {},
		doubleValueName: {},
		stringValueName: {},
		bytesValueName:  {},
	}

	allowedMapKeyTypes = map[gatewayv1alpha1.Schema_Field_Type_ScalarType]struct{}{
		gatewayv1alpha1.Schema_Field_Type_SCALAR_TYPE_BOOL:   {},
		gatewayv1alpha1.Schema_Field_Type_SCALAR_TYPE_INT32:  {},
		gatewayv1alpha1.Schema_Field_Type_SCALAR_TYPE_INT64:  {},
		gatewayv1alpha1.Schema_Field_Type_SCALAR_TYPE_UINT32: {},
		gatewayv1alpha1.Schema_Field_Type_SCALAR_TYPE_UINT64: {},
		gatewayv1alpha1.Schema_Field_Type_SCALAR_TYPE_STRING: {},
	}
)

func computeResponseSchemas(gateway *Gateway, requests []*gatewayv1alpha1.Request, forFetch bool, methodKind rpcKind) ([]*methodConfig, []*gatewayv1alpha1.Schema, error) {
	confs := make([]*methodConfig, len(requests))
	schemas := make([]*gatewayv1alpha1.Schema, len(requests))
	for i, request := range requests {
		conf, schema, err := computeResponseSchema(gateway, request, forFetch, methodKind)
		if err != nil {
			return nil, nil, err
		}
		confs[i], schemas[i] = conf, schema
	}
	return confs, schemas, nil
}

func computeResponseSchema(gateway *Gateway, request *gatewayv1alpha1.Request, forFetch bool, methodKind rpcKind) (*methodConfig, *gatewayv1alpha1.Schema, error) {
	var conf *methodConfig
	methodName := protoreflect.FullName(request.Method)
	if !methodName.IsValid() {
		return conf, nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("%q is not a valid method name", methodName))
	}
	conf, ok := gateway.methods[methodName]
	if !ok {
		return conf, nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("requested endpoint %q is not supported", methodName))
	}
	if kindOf(conf.descriptor) != methodKind {
		return conf, nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("endpoint %q has unexpected kind %v (expecting %v)", methodName, kindOf(conf.descriptor), methodKind))
	}

	if forFetch {
		// Since conf.descriptor is a protoreflect.MethodDescriptor, we know the options
		// type is a *descriptorpb.MethodOptions.
		//nolint:forcetypeassert
		idempotence := conf.descriptor.Options().(*descriptorpb.MethodOptions).GetIdempotencyLevel()
		if idempotence != descriptorpb.MethodOptions_NO_SIDE_EFFECTS {
			return conf, nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("endpoint %q cannot be used with Fetch: idempotency_level = %v", methodName, idempotence))
		}
	}

	schema, err := computeSchema(gateway, conf.descriptor.Output(), request.Mask, forFetch, []string{string(methodName)})
	return conf, schema, err
}

func computeSchema(gateway *Gateway, descriptor protoreflect.MessageDescriptor, mask []*gatewayv1alpha1.MaskField, forFetch bool, path []string) (*gatewayv1alpha1.Schema, error) {
	msgName := descriptor.FullName()
	schema := &gatewayv1alpha1.Schema{Name: string(msgName)}
	if _, ok := wellKnownTypes[msgName]; ok {
		if len(mask) > 0 {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("%v: cannot supply mask for well-known type %q", strings.Join(path, "/"), msgName))
		}
		return schema, nil
	}
	if len(mask) == 0 {
		// nothing else to do
		return schema, nil
	}

	schemaFields := make([]*gatewayv1alpha1.Schema_Field, len(mask))
	names := make(map[string]struct{}, len(mask))
	for i, field := range mask {
		if _, ok := names[field.Name]; ok {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("%v: invalid mask: more than one entry for %q", strings.Join(path, "/"), field.Name))
		}
		names[field.Name] = struct{}{}
		fieldDescriptor := fieldWithCamelCaseName(descriptor.Fields(), field.Name)
		if fieldDescriptor == nil {
			conf, ok := gateway.relations[msgName][field.Name]
			if !ok {
				return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("%v: no such field %q", strings.Join(path, "/"), field.Name))
			}
			if conf.requestType == nil && field.Params != nil {
				return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("%v: relation %q takes no parameters, but parameters were provided", strings.Join(path, "/"), field.Name))
			}
			fieldDescriptor = conf.descriptor
		}
		fieldType, err := computeFieldType(gateway, fieldDescriptor, field, forFetch, path)
		if err != nil {
			return nil, err
		}
		camelCaseName := camelCase(field.Name)
		jsonName := fieldDescriptor.JSONName()
		if jsonName == camelCaseName {
			jsonName = "" // only provide JsonName field when it differs from Name
		}
		schemaFields[i] = &gatewayv1alpha1.Schema_Field{
			Name:     camelCaseName,
			JsonName: jsonName,
			Type:     fieldType,
		}
	}

	schema.Fields = schemaFields
	return schema, nil
}

func computeFieldType(gateway *Gateway, descriptor protoreflect.FieldDescriptor, mask *gatewayv1alpha1.MaskField, forFetch bool, path []string) (*gatewayv1alpha1.Schema_Field_Type, error) {
	switch {
	case descriptor.IsMap():
		key, err := computeScalarFieldType(descriptor.MapKey(), &gatewayv1alpha1.MaskField{}, path)
		if err != nil {
			return nil, err
		}
		if _, ok := allowedMapKeyTypes[key]; !ok {
			// shouldn't be possible
			return nil, fmt.Errorf("%v: map key type %v is not valid", strings.Join(path, "/"), key)
		}
		valueDescriptor := descriptor.MapValue()
		if isMessage(valueDescriptor.Kind()) {
			schema, err := computeSchema(gateway, valueDescriptor.Message(), mask.Mask, forFetch, append(path, mask.Name))
			if err != nil {
				return nil, err
			}
			return &gatewayv1alpha1.Schema_Field_Type{
				Value: &gatewayv1alpha1.Schema_Field_Type_Map{
					Map: &gatewayv1alpha1.Schema_Field_Type_MapType{
						Key:   key,
						Value: &gatewayv1alpha1.Schema_Field_Type_MapType_Message{Message: schema},
					},
				},
			}, nil
		}
		scalar, err := computeScalarFieldType(valueDescriptor, mask, path)
		if err != nil {
			return nil, err
		}
		return &gatewayv1alpha1.Schema_Field_Type{
			Value: &gatewayv1alpha1.Schema_Field_Type_Map{
				Map: &gatewayv1alpha1.Schema_Field_Type_MapType{
					Key:   key,
					Value: &gatewayv1alpha1.Schema_Field_Type_MapType_Scalar{Scalar: scalar},
				},
			},
		}, nil

	case descriptor.IsList():
		if isMessage(descriptor.Kind()) {
			schema, err := computeSchema(gateway, descriptor.Message(), mask.Mask, forFetch, append(path, mask.Name))
			if err != nil {
				return nil, err
			}
			return &gatewayv1alpha1.Schema_Field_Type{
				Value: &gatewayv1alpha1.Schema_Field_Type_Repeated{
					Repeated: &gatewayv1alpha1.Schema_Field_Type_RepeatedType{
						Element: &gatewayv1alpha1.Schema_Field_Type_RepeatedType_Message{Message: schema},
					},
				},
			}, nil
		}
		scalar, err := computeScalarFieldType(descriptor, mask, path)
		if err != nil {
			return nil, err
		}
		return &gatewayv1alpha1.Schema_Field_Type{
			Value: &gatewayv1alpha1.Schema_Field_Type_Repeated{
				Repeated: &gatewayv1alpha1.Schema_Field_Type_RepeatedType{
					Element: &gatewayv1alpha1.Schema_Field_Type_RepeatedType_Scalar{Scalar: scalar},
				},
			},
		}, nil

	case isMessage(descriptor.Kind()):
		schema, err := computeSchema(gateway, descriptor.Message(), mask.Mask, forFetch, append(path, mask.Name))
		if err != nil {
			return nil, err
		}
		return &gatewayv1alpha1.Schema_Field_Type{
			Value: &gatewayv1alpha1.Schema_Field_Type_Message{Message: schema},
		}, nil

	default:
		scalar, err := computeScalarFieldType(descriptor, mask, path)
		if err != nil {
			return nil, err
		}
		return &gatewayv1alpha1.Schema_Field_Type{
			Value: &gatewayv1alpha1.Schema_Field_Type_Scalar{Scalar: scalar},
		}, nil
	}
}

func computeScalarFieldType(descriptor protoreflect.FieldDescriptor, mask *gatewayv1alpha1.MaskField, path []string) (gatewayv1alpha1.Schema_Field_Type_ScalarType, error) {
	var scalar gatewayv1alpha1.Schema_Field_Type_ScalarType
	switch descriptor.Kind() {
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		scalar = gatewayv1alpha1.Schema_Field_Type_SCALAR_TYPE_INT32
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		scalar = gatewayv1alpha1.Schema_Field_Type_SCALAR_TYPE_UINT32
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		scalar = gatewayv1alpha1.Schema_Field_Type_SCALAR_TYPE_INT64
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		scalar = gatewayv1alpha1.Schema_Field_Type_SCALAR_TYPE_UINT64
	case protoreflect.DoubleKind:
		scalar = gatewayv1alpha1.Schema_Field_Type_SCALAR_TYPE_DOUBLE
	case protoreflect.FloatKind:
		scalar = gatewayv1alpha1.Schema_Field_Type_SCALAR_TYPE_FLOAT
	case protoreflect.BoolKind:
		scalar = gatewayv1alpha1.Schema_Field_Type_SCALAR_TYPE_BOOL
	case protoreflect.StringKind:
		scalar = gatewayv1alpha1.Schema_Field_Type_SCALAR_TYPE_STRING
	case protoreflect.BytesKind:
		scalar = gatewayv1alpha1.Schema_Field_Type_SCALAR_TYPE_BYTES
	case protoreflect.EnumKind:
		if descriptor.Enum().FullName() == nullValueName {
			scalar = gatewayv1alpha1.Schema_Field_Type_SCALAR_TYPE_NULL
		} else {
			scalar = gatewayv1alpha1.Schema_Field_Type_SCALAR_TYPE_ENUM
		}
	case protoreflect.MessageKind, protoreflect.GroupKind:
		// these aren't allowed here
		fallthrough
	default:
		return 0, fmt.Errorf("%v: expecting primitive type but instead got %v", strings.Join(path, "/"), descriptor.Kind())
	}
	if len(mask.Mask) > 0 {
		return 0, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("%v: cannot supply mask for scalar type %q", strings.Join(path, "/"), scalar))
	}
	return scalar, nil
}

func isMessage(kind protoreflect.Kind) bool {
	return kind == protoreflect.MessageKind || kind == protoreflect.GroupKind
}

func kindOf(method protoreflect.MethodDescriptor) rpcKind {
	switch {
	case method.IsStreamingClient() && method.IsStreamingServer():
		return rpcKindBidiStream
	case method.IsStreamingServer():
		return rpcKindServerStream
	case method.IsStreamingClient():
		return rpcKindClientStream
	default:
		return rpcKindUnary
	}
}
