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
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	gatewayv1alpha1 "buf.build/gen/go/bufbuild/knit/protocolbuffers/go/buf/knit/gateway/v1alpha1"
	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// This is set to true from tests.
var deterministicPatchOrder = false //nolint:gochecknoglobals

type patch struct {
	// path in the request to the mask that corresponds to this patch
	path []string
	// The message whose mask includes a relation that needs to be patched in
	target proto.Message
	// The relation that needs to be patched in.
	mask *gatewayv1alpha1.MaskField
	// The formatted version of target, into which the JSON format of the relation
	// will be added.
	formatTarget *structpb.Struct
	// metadata about resolve operation that generated this patch
	meta *resolveMeta
	// The nearest patch up the query that has a catch clause. If this patch
	// has a catch clause, it will be used to handle errors from this patch.
	errPatch *errorPatch
}

type errorPatch struct {
	// The formatted version of target, into which the JSON format of the error
	// will be added.
	formatTarget *structpb.Struct
	// The name of the field in formatTarget that the error should be added to.
	name string
}

type patchesByKey struct {
	patches []*patch
	keys    []string
}

func (p patchesByKey) Len() int {
	return len(p.patches)
}

func (p patchesByKey) Less(i, j int) bool {
	return p.keys[i] < p.keys[j]
}

func (p patchesByKey) Swap(i, j int) {
	p.patches[i], p.patches[j] = p.patches[j], p.patches[i]
	p.keys[i], p.keys[j] = p.keys[j], p.keys[i]
}

// pointerMessage is a pointer type that implements proto.Message.
type pointerMessage[T any] interface {
	*T
	proto.Message
}

// as returns the given message with the concrete type of *T. This is useful for
// recovering a concrete, generated message type from a value that could possibly
// be a dynamic message. If the given message is a dynamic message, then it will
// be converted by serializing to bytes and then de-serializing into a *T. An
// error is returned if the actual message type of *T is not the same as that of
// the given msg.
func as[T any, P pointerMessage[T]](msg proto.Message) (P, error) {
	if p, ok := msg.(P); ok {
		return p, nil
	}
	// Not the same type. There might be a clever way to use.ProtoReflect().Range(...)
	// to clone the data from msg into the target type. But there are many complicated
	// edge cases, like if the message descriptors are somehow incompatible. So we'll
	// just do the inefficient thing and marshal to intermediate bytes. Hopefully, this
	// is rare/unlikely, since service registration and resolver registration both let
	// caller provide a particular protoreflect.MessageType (which will be associated
	// with a particular runtime Go type, so we can instantiate the right type to start).
	ptr := P(new(T))
	if msg.ProtoReflect().Descriptor().FullName() != ptr.ProtoReflect().Descriptor().FullName() {
		return nil, fmt.Errorf("cannot convert %T to %T because they are different message types: %s != %s",
			msg, ptr, msg.ProtoReflect().Descriptor().FullName(), ptr.ProtoReflect().Descriptor().FullName())
	}
	data, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}
	if err := proto.Unmarshal(data, ptr); err != nil {
		return nil, err
	}
	return ptr, nil
}

func parseMessage(value *structpb.Value, msg proto.Message, res TypeResolver) error {
	// TODO: We could be much more efficient here if we directly converted from google.protobuf.Value.
	//       But that's a whole lot more code to write and test. So we do the easy (but less efficient)
	//       thing for now...
	data, err := protojson.MarshalOptions{Resolver: res}.Marshal(value)
	if err != nil {
		return err
	}
	if err := (protojson.UnmarshalOptions{Resolver: res}).Unmarshal(data, msg); err != nil {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	return nil
}

//nolint:gocyclo
func formatMessage(msg proto.Message, resMeta *resolveMeta, mask []*gatewayv1alpha1.MaskField, path []string, upstreamErrPatch *errorPatch, fallbackCatch bool, res TypeResolver) (*structpb.Value, []*patch, error) {
	switch msg.ProtoReflect().Descriptor().FullName() {
	case anyName:
		msg, err := as[anypb.Any](msg)
		if err != nil {
			return nil, nil, err
		}
		return formatAny(msg, res)

	case fieldMaskName:
		msg, err := as[fieldmaskpb.FieldMask](msg)
		if err != nil {
			return nil, nil, err
		}
		return formatFieldMask(msg), nil, nil

	case timestampName:
		msg, err := as[timestamppb.Timestamp](msg)
		if err != nil {
			return nil, nil, err
		}
		str := msg.AsTime().UTC().Format(time.RFC3339Nano)
		return structpb.NewStringValue(str), nil, nil

	case durationName:
		msg, err := as[durationpb.Duration](msg)
		if err != nil {
			return nil, nil, err
		}
		return structpb.NewStringValue(fmt.Sprintf("%01.9fs", msg.AsDuration().Seconds())), nil, nil

	case valueName:
		msg, err := as[structpb.Value](msg)
		return msg, nil, err

	case listValueName:
		msg, err := as[structpb.ListValue](msg)
		if err != nil {
			return nil, nil, err
		}
		return structpb.NewListValue(msg), nil, nil

	case structName:
		msg, err := as[structpb.Struct](msg)
		if err != nil {
			return nil, nil, err
		}
		return structpb.NewStructValue(msg), nil, nil

	case boolValueName:
		msg, err := as[wrapperspb.BoolValue](msg)
		if err != nil {
			return nil, nil, err
		}
		return structpb.NewBoolValue(msg.GetValue()), nil, nil

	case int32ValueName:
		msg, err := as[wrapperspb.Int32Value](msg)
		if err != nil {
			return nil, nil, err
		}
		return structpb.NewNumberValue(float64(msg.GetValue())), nil, nil

	case int64ValueName:
		msg, err := as[wrapperspb.Int64Value](msg)
		if err != nil {
			return nil, nil, err
		}
		return structpb.NewStringValue(strconv.FormatInt(msg.GetValue(), 10)), nil, nil

	case uint32ValueName:
		msg, err := as[wrapperspb.UInt32Value](msg)
		if err != nil {
			return nil, nil, err
		}
		return structpb.NewNumberValue(float64(msg.GetValue())), nil, nil

	case uint64ValueName:
		msg, err := as[wrapperspb.UInt64Value](msg)
		if err != nil {
			return nil, nil, err
		}
		return structpb.NewStringValue(strconv.FormatUint(msg.GetValue(), 10)), nil, nil

	case floatValueName:
		msg, err := as[wrapperspb.FloatValue](msg)
		if err != nil {
			return nil, nil, err
		}
		return formatFloat(float64(msg.GetValue())), nil, nil

	case doubleValueName:
		msg, err := as[wrapperspb.DoubleValue](msg)
		if err != nil {
			return nil, nil, err
		}
		return formatFloat(msg.GetValue()), nil, nil

	case stringValueName:
		msg, err := as[wrapperspb.StringValue](msg)
		if err != nil {
			return nil, nil, err
		}
		return structpb.NewStringValue(msg.GetValue()), nil, nil

	case bytesValueName:
		msg, err := as[wrapperspb.BytesValue](msg)
		if err != nil {
			return nil, nil, err
		}
		return structpb.NewStringValue(base64.StdEncoding.EncodeToString(msg.GetValue())), nil, nil

	default:
		return formatNormalMessage(msg, resMeta, mask, path, upstreamErrPatch, fallbackCatch, res)
	}
}

func formatNormalMessage(msg proto.Message, resMeta *resolveMeta, mask []*gatewayv1alpha1.MaskField, path []string, upstreamErrPatch *errorPatch, fallbackCatch bool, res TypeResolver) (*structpb.Value, []*patch, error) {
	val := &structpb.Struct{Fields: map[string]*structpb.Value{}}
	var patches []*patch
	msgRef := msg.ProtoReflect()
	msgFields := msgRef.Descriptor().Fields()
	for _, maskField := range mask {
		field := fieldWithCamelCaseName(msgFields, maskField.Name)
		if field == nil {
			errPatch := upstreamErrPatch
			if shouldCatch(maskField, fallbackCatch) {
				errPatch = &errorPatch{
					formatTarget: val,
					name:         maskField.Name,
				}
			}
			// this is a relation that needs to be patched in
			patches = append(patches, &patch{
				path:         path,
				target:       msg,
				mask:         maskField,
				formatTarget: val,
				meta:         resMeta,
				errPatch:     errPatch,
			})
			continue
		}
		if field.HasPresence() && !msgRef.Has(field) {
			continue
		}
		path := append(path, maskField.Name)
		fieldVal, fieldPatches, err := formatValue(field, msgRef.Get(field), resMeta, maskField.Mask, path, upstreamErrPatch, fallbackCatch, res)
		if err != nil {
			return nil, nil, err
		}
		val.Fields[maskField.Name] = fieldVal
		patches = append(patches, fieldPatches...)
	}
	return structpb.NewStructValue(val), patches, nil
}

func formatValue(field protoreflect.FieldDescriptor, val protoreflect.Value, resMeta *resolveMeta, mask []*gatewayv1alpha1.MaskField, path []string, upstreamErrPatch *errorPatch, fallbackCatch bool, res TypeResolver) (*structpb.Value, []*patch, error) {
	switch {
	case field.IsMap():
		mapKey := field.MapKey().Kind()
		mapVal := field.MapValue()
		obj := map[string]*structpb.Value{}
		var patches []*patch
		var keys []string
		var rangeErr error
		val.Map().Range(func(k protoreflect.MapKey, val protoreflect.Value) bool {
			keyStr, err := formatMapKeyValue(mapKey, k, path)
			if err != nil {
				rangeErr = err
				return false
			}
			entryVal, entryPatches, err := formatSingularValue(mapVal, val, resMeta, mask, path, upstreamErrPatch, fallbackCatch, res)
			if err != nil {
				rangeErr = err
				return false
			}
			obj[keyStr] = entryVal
			patches = append(patches, entryPatches...)
			if deterministicPatchOrder {
				for i := 0; i < len(entryPatches); i++ {
					keys = append(keys, keyStr)
				}
			}
			return true
		})
		if rangeErr != nil {
			return nil, nil, rangeErr
		}
		if deterministicPatchOrder {
			sort.Stable(patchesByKey{patches, keys})
		}
		return structpb.NewStructValue(&structpb.Struct{Fields: obj}), patches, nil

	case field.IsList():
		listVal := val.List()
		vals := make([]*structpb.Value, listVal.Len())
		var patches []*patch
		for i, l := 0, listVal.Len(); i < l; i++ {
			elementVal, elementPatches, err := formatSingularValue(field, listVal.Get(i), resMeta, mask, path, upstreamErrPatch, fallbackCatch, res)
			if err != nil {
				return nil, nil, err
			}
			vals[i] = elementVal
			patches = append(patches, elementPatches...)
		}
		return structpb.NewListValue(&structpb.ListValue{Values: vals}), patches, nil

	default:
		return formatSingularValue(field, val, resMeta, mask, path, upstreamErrPatch, fallbackCatch, res)
	}
}

func formatSingularValue(field protoreflect.FieldDescriptor, val protoreflect.Value, resMeta *resolveMeta, mask []*gatewayv1alpha1.MaskField, path []string, upstreamErrPatch *errorPatch, fallbackCatch bool, res TypeResolver) (*structpb.Value, []*patch, error) {
	switch field.Kind() {
	case protoreflect.MessageKind, protoreflect.GroupKind:
		return formatMessage(val.Message().Interface(), resMeta, mask, path, upstreamErrPatch, fallbackCatch, res)
	case protoreflect.EnumKind:
		if field.Enum().FullName() == nullValueName {
			return structpb.NewNullValue(), nil, nil
		}
		enumNum := val.Enum()
		enumVal := field.Enum().Values().ByNumber(enumNum)
		if enumVal == nil {
			// oof, not a known value
			return structpb.NewNumberValue(float64(enumNum)), nil, nil
		}
		return structpb.NewStringValue(string(enumVal.Name())), nil, nil
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return structpb.NewNumberValue(float64(val.Int())), nil, nil
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return structpb.NewStringValue(strconv.FormatInt(val.Int(), 10)), nil, nil
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return structpb.NewNumberValue(float64(val.Uint())), nil, nil
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return structpb.NewStringValue(strconv.FormatUint(val.Uint(), 10)), nil, nil
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		return formatFloat(val.Float()), nil, nil
	case protoreflect.BoolKind:
		return structpb.NewBoolValue(val.Bool()), nil, nil
	case protoreflect.StringKind:
		return structpb.NewStringValue(val.String()), nil, nil
	case protoreflect.BytesKind:
		return structpb.NewStringValue(base64.StdEncoding.EncodeToString(val.Bytes())), nil, nil
	default:
		return nil, nil, fmt.Errorf("%v: unrecognized value kind: %v", strings.Join(path, "."), field.Kind())
	}
}

func formatFloat(floatVal float64) *structpb.Value {
	switch {
	case math.IsNaN(floatVal):
		return structpb.NewStringValue("NaN")
	case math.IsInf(floatVal, 1):
		return structpb.NewStringValue("Infinity")
	case math.IsInf(floatVal, -1):
		return structpb.NewStringValue("-Infinity")
	default:
		return structpb.NewNumberValue(floatVal)
	}
}

func formatMapKeyValue(keyKind protoreflect.Kind, key protoreflect.MapKey, path []string) (string, error) {
	// We can't use an exhaustive switch here because map keys simply don't support all kinds
	//nolint:exhaustive
	switch keyKind {
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
		protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return strconv.FormatInt(key.Int(), 10), nil
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind,
		protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return strconv.FormatUint(key.Uint(), 10), nil
	case protoreflect.BoolKind:
		return strconv.FormatBool(key.Bool()), nil
	case protoreflect.StringKind:
		return key.String(), nil
	default:
		return "", fmt.Errorf("%v: unrecognized map key kind: %v", strings.Join(path, "."), keyKind)
	}
}

func formatAny(anyMsg *anypb.Any, res TypeResolver) (*structpb.Value, []*patch, error) {
	containedMsg, err := anypb.UnmarshalNew(anyMsg, proto.UnmarshalOptions{Resolver: res})
	if err != nil {
		return nil, nil, err
	}

	// We don't apply a mask to messages inside an Any. So we can just use
	// the protojson package to convert to JSON and then to google.protobuf.Value.
	data, err := protojson.MarshalOptions{Resolver: res}.Marshal(containedMsg)
	if err != nil {
		return nil, nil, err
	}
	val := &structpb.Value{}
	err = protojson.UnmarshalOptions{Resolver: res}.Unmarshal(data, val)
	if err != nil {
		return nil, nil, err
	}

	if strct, ok := val.Kind.(*structpb.Value_StructValue); ok {
		strct.StructValue.Fields["@type"] = structpb.NewStringValue(anyMsg.TypeUrl)
	} else {
		// Not a struct? Then contained message has custom JSON format
		// which goes into a "value" property.
		val = structpb.NewStructValue(&structpb.Struct{
			Fields: map[string]*structpb.Value{
				"@type": structpb.NewStringValue(anyMsg.TypeUrl),
				"value": val,
			},
		})
	}
	return val, nil, nil
}

func formatFieldMask(msg *fieldmaskpb.FieldMask) *structpb.Value {
	masks := make([]string, len(msg.Paths))
	for i, path := range msg.Paths {
		parts := strings.Split(path, ".")
		for j := range parts {
			parts[j] = camelCase(parts[j])
		}
		masks[i] = strings.Join(parts, ".")
	}
	return structpb.NewStringValue(strings.Join(masks, ","))
}

func formatError(err error, path string, res TypeResolver) *structpb.Value {
	connectErr := new(connect.Error)
	if !errors.As(err, &connectErr) {
		connectErr = connect.NewError(connect.CodeUnknown, err)
	}
	details := make([]*structpb.Value, 0, len(connectErr.Details()))
	for _, detail := range connectErr.Details() {
		detailStruct := &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"type":  structpb.NewStringValue(detail.Type()),
				"value": structpb.NewStringValue(base64.RawStdEncoding.EncodeToString(detail.Bytes())),
			},
		}
		debug, err := formatDetail(detail, res)
		// Err is ignored here as debug is optional and most likely
		// the error is due to an unknown type.
		if err == nil {
			detailStruct.Fields["debug"] = debug
		}
		details = append(details, structpb.NewStructValue(detailStruct))
	}
	return structpb.NewStructValue(&structpb.Struct{
		Fields: map[string]*structpb.Value{
			"[@error]": structpb.NewStructValue(&structpb.Struct{}),
			"code":     structpb.NewStringValue(gatewayv1alpha1.Error_Code(connectErr.Code()).String()),
			"message":  structpb.NewStringValue(connectErr.Message()),
			"details":  structpb.NewListValue(&structpb.ListValue{Values: details}),
			"path":     structpb.NewStringValue(path),
		},
	})
}

func formatDetail(detail *connect.ErrorDetail, res TypeResolver) (*structpb.Value, error) {
	containedMsg, err := anypb.UnmarshalNew(
		&anypb.Any{
			TypeUrl: "type.googleapis.com/" + detail.Type(),
			Value:   detail.Bytes(),
		},
		proto.UnmarshalOptions{Resolver: res},
	)
	if err != nil {
		return nil, err
	}
	// We don't apply a mask to messages inside an Any. So we can just use
	// the protojson package to convert to JSON and then to google.protobuf.Value.
	data, err := protojson.MarshalOptions{Resolver: res}.Marshal(containedMsg)
	if err != nil {
		return nil, err
	}
	val := &structpb.Value{}
	if err := (protojson.UnmarshalOptions{Resolver: res}).Unmarshal(data, val); err != nil {
		return nil, err
	}
	return val, nil
}

func shouldCatch(maskField *gatewayv1alpha1.MaskField, fallbackCatch bool) bool {
	return maskField.GetCatch() != nil || (fallbackCatch && maskField.GetThrow() == nil)
}

// camelCase returns the default JSON name for a field with the given name.
//
// This function was copied from google.golang.org/protobuf/internal/strs.JSONCamelCase
//
//nolint:varnamelen
func camelCase(s string) string {
	var b []byte
	var wasUnderscore bool
	for i := 0; i < len(s); i++ { // proto identifiers are always ASCII
		c := s[i]
		if c != '_' {
			if wasUnderscore && isASCIILower(c) {
				c -= 'a' - 'A' // convert to uppercase
			}
			b = append(b, c)
		}
		wasUnderscore = c == '_'
	}
	return string(b)
}

// snakeCase is the inverse of camelCase.
//
// This function was copied from google.golang.org/protobuf/internal/strs.JSONSnakeCase
//
//nolint:varnamelen
func snakeCase(s string) string {
	var b []byte
	for i := 0; i < len(s); i++ { // proto identifiers are always ASCII
		c := s[i]
		if isASCIIUpper(c) {
			b = append(b, '_')
			c += 'a' - 'A' // convert to lowercase
		}
		b = append(b, c)
	}
	return string(b)
}

func isASCIILower(c byte) bool {
	return 'a' <= c && c <= 'z'
}

func isASCIIUpper(c byte) bool {
	return 'A' <= c && c <= 'Z'
}

func fieldWithCamelCaseName(fields protoreflect.FieldDescriptors, name string) protoreflect.FieldDescriptor {
	expectedName := snakeCase(name)
	field := fields.ByName(protoreflect.Name(expectedName))
	if field != nil {
		// found it!
		return field
	}
	// fallback to slow search
	for i, l := 0, fields.Len(); i < l; i++ {
		field := fields.Get(i)
		if camelCase(string(field.Name())) == name {
			return field
		}
	}
	return nil // couldn't find it
}
