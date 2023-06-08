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
	"fmt"
	"math"
	"os"
	"testing"
	"time"

	gatewayv1alpha1 "buf.build/gen/go/bufbuild/knit/protocolbuffers/go/buf/knit/gateway/v1alpha1"
	connect "github.com/bufbuild/connect-go"
	"github.com/bufbuild/knit-go/internal/gen/buf/knittest"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestMain(m *testing.M) {
	deterministicPatchOrder = true
	os.Exit(m.Run())
}

func TestFormatMessage(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		msg     proto.Message
		mask    []*gatewayv1alpha1.MaskField
		patches [][]any
	}{
		{
			name: "wkts",
			msg:  getWellKnownTypesExample(t),
			mask: getExhaustiveMaskForWellKnownTypes(),
		},
		{
			name: "scalars",
			msg:  getScalarsExample(),
			mask: getExhaustiveMaskForScalars(),
		},
		{
			name: "foo",
			msg:  getFooExample(),
			mask: getExhaustiveFooMask(),
			patches: [][]any{
				{"style"},
				{"balance_cents"},
				{"bars"},
			},
		},
		{
			name: "fizz",
			msg:  getFizzExample(),
			mask: getExhaustiveFizzMask(),
			patches: [][]any{
				{"bars", 0, "baz"},
				{"bars", 0, "bedazzles"},
				{"bars", 1, "baz"},
				{"bars", 1, "bedazzles"},
				{"bars", 2, "baz"},
				{"bars", 2, "bedazzles"},
				{"bars_by_category", "A1", "baz"},
				{"bars_by_category", "A1", "bedazzles"},
				{"bars_by_category", "B2", "baz"},
				{"bars_by_category", "B2", "bedazzles"},
				{"buzzes"},
			},
		},
	}

	for i := range testCases {
		testCase := testCases[i]
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			info := &resolveMeta{}
			val, patches, err := formatMessage(testCase.msg, info, testCase.mask, []string{"foo"}, nil, false, nil)
			require.NoError(t, err)

			// make sure what we produced is valid and matches the original
			// message definition by doing a round-trip (translation party)
			roundTrip := testCase.msg.ProtoReflect().New().Interface()
			err = parseMessage(val, roundTrip, nil)
			require.NoError(t, err)
			if !proto.Equal(testCase.msg, roundTrip) {
				// we have to guard the diff because the diff always complains
				// about NaN values, but proto.Equal is smart enough not to
				diff := cmp.Diff(testCase.msg, roundTrip, protocmp.Transform())
				require.Fail(t, "incorrect output", "round-trip failed (- expected, + actual):\n%s", diff)
			}

			// check patches
			require.Equal(t, len(testCase.patches), len(patches))
			for i, expectedPatch := range testCase.patches {
				p := buildPatch(t, expectedPatch, testCase.msg, val.GetStructValue(), testCase.mask, []string{"foo"})
				checkPatch(t, patches[i], p.path, p.msg, p.val, p.mask, info, nil, "")
			}
		})
	}
	t.Run("errors", func(t *testing.T) {
		t.Parallel()
		testCases := []struct {
			name          string
			onError       string
			upstream      bool
			fallbackCatch bool
			errTarget     string
		}{
			{
				name:          "fallback-catch",
				fallbackCatch: true,
				errTarget:     "self",
			},
			{
				name:      "upstream",
				upstream:  true,
				errTarget: "upstream",
			},
			{
				name:      "catch",
				onError:   "catch",
				errTarget: "self",
			},
			{
				name:          "throw-fallback",
				onError:       "throw",
				fallbackCatch: true,
			},
			{
				name:      "throw-upstream",
				onError:   "throw",
				upstream:  true,
				errTarget: "upstream",
			},
		}
		for _, testCase := range testCases {
			testCase := testCase
			t.Run(testCase.name, func(t *testing.T) {
				t.Parallel()
				msg := getFooExample()
				mask := []*gatewayv1alpha1.MaskField{
					{
						Name: "style",
					},
				}
				switch testCase.onError {
				case "catch":
					mask[0].OnError = &gatewayv1alpha1.MaskField_Catch{Catch: &gatewayv1alpha1.Catch{}}
				case "throw":
					mask[0].OnError = &gatewayv1alpha1.MaskField_Throw{Throw: &gatewayv1alpha1.Throw{}}
				}
				var upstreamErrTarget structpb.Struct
				upstreamErrPatch := &errorPatch{formatTarget: &upstreamErrTarget, name: "bar"}
				if !testCase.upstream {
					upstreamErrPatch = nil
				}
				resMeta := &resolveMeta{}
				target, patches, err := formatMessage(msg, resMeta, mask, []string{"foo"}, upstreamErrPatch, testCase.fallbackCatch, nil)
				require.NoError(t, err)
				require.NotNil(t, target)
				assert.Len(t, patches, 1)
				switch testCase.errTarget {
				case "upstream":
					checkPatch(t, patches[0], []string{"foo"}, msg, target.GetStructValue(), mask[0], resMeta, &upstreamErrTarget, "bar")
				case "self":
					checkPatch(t, patches[0], []string{"foo"}, msg, target.GetStructValue(), mask[0], resMeta, target.GetStructValue(), "style")
				case "":
					checkPatch(t, patches[0], []string{"foo"}, msg, target.GetStructValue(), mask[0], resMeta, nil, "")
				}
			})
		}
	})
}

func TestFormatError(t *testing.T) {
	t.Parallel()
	relErr := connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("msg"))
	foo := &knittest.Foo{
		Name: "name",
	}
	errDetail, err := connect.NewErrorDetail(foo)
	require.NoError(t, err)
	relErr.AddDetail(errDetail)
	val := formatError(relErr, "foo", protoregistry.GlobalTypes)
	fooBytes, err := proto.Marshal(foo)
	require.NoError(t, err)
	require.Equal(t, val.AsInterface(), map[string]any{
		"[@error]": map[string]any{},
		"code":     gatewayv1alpha1.Error_INVALID_ARGUMENT.String(),
		"message":  "msg",
		"path":     "foo",
		"details": []any{
			map[string]any{
				"type":  "buf.knittest.Foo",
				"value": base64.RawStdEncoding.EncodeToString(fooBytes),
				"debug": map[string]any{
					"name": "name",
				},
			},
		},
	})
}

type patchExpectation struct {
	path []string
	msg  proto.Message
	val  *structpb.Struct
	mask *gatewayv1alpha1.MaskField
}

// build the expected details of a patch from the given path. The path consists of a sequence of
// field names, map keys (when a field value is a map), and indexes (when a field value is a list).
// Traversal of the path recursively traverses into relevant values inside the given msg, val, and mask.
// patchPath accumulates the path of the expected patch as we traverse path.
func buildPatch(t *testing.T, path []any, msg proto.Message, val *structpb.Struct, mask []*gatewayv1alpha1.MaskField, patchPath []string) *patchExpectation {
	t.Helper()
	name := path[0].(string) //nolint:errcheck,forcetypeassert
	maskField := findMask(t, camelCase(name), mask)
	if len(path) == 1 {
		return &patchExpectation{
			path: patchPath,
			msg:  msg,
			val:  val,
			mask: maskField,
		}
	}
	patchPath = append(patchPath, maskField.Name)
	field := msg.ProtoReflect().Descriptor().Fields().ByName(protoreflect.Name(name))
	fieldValue := msg.ProtoReflect().Get(field)
	valFieldValue := val.Fields[field.JSONName()]
	switch {
	case field.IsMap():
		mapKey := protoreflect.ValueOf(path[1]).MapKey()
		valKey, err := formatMapKeyValue(field.MapKey().Kind(), mapKey, nil)
		require.NoError(t, err)
		return buildPatch(t, path[2:], fieldValue.Map().Get(mapKey).Message().Interface(), valFieldValue.GetStructValue().Fields[valKey].GetStructValue(), maskField.Mask, patchPath)
	case field.IsList():
		index := path[1].(int) //nolint:errcheck,forcetypeassert
		return buildPatch(t, path[2:], fieldValue.List().Get(index).Message().Interface(), valFieldValue.GetListValue().Values[index].GetStructValue(), maskField.Mask, patchPath)
	default:
		return buildPatch(t, path[1:], fieldValue.Message().Interface(), valFieldValue.GetStructValue(), maskField.Mask, patchPath)
	}
}

func checkPatch(
	t *testing.T,
	patch *patch,
	path []string,
	msg proto.Message,
	val *structpb.Struct,
	mask *gatewayv1alpha1.MaskField,
	info *resolveMeta,
	errTarget *structpb.Struct,
	errName string,
) {
	t.Helper()
	require.Equal(t, path, patch.path)
	require.Same(t, msg, patch.target)
	require.Same(t, val, patch.formatTarget)
	require.Same(t, mask, patch.mask)
	require.Same(t, info, patch.meta)
	if errTarget == nil {
		require.Nil(t, patch.errPatch)
	} else {
		require.NotNil(t, patch.errPatch)
		require.Same(t, errTarget, patch.errPatch.formatTarget)
		require.Equal(t, errName, patch.errPatch.name)
	}
}

func findMask(t *testing.T, name string, mask []*gatewayv1alpha1.MaskField) *gatewayv1alpha1.MaskField {
	t.Helper()
	for _, field := range mask {
		if field.Name == name {
			return field
		}
	}
	t.Fatalf("could not find mask field named %q", name)
	return nil // unreachable
}

func getWellKnownTypesExample(t *testing.T) *knittest.WellKnownTypes {
	t.Helper()

	anyMsg := &anypb.Any{}
	err := anypb.MarshalFrom(anyMsg, &knittest.Fizz{Color: "beige", Density: 1.048}, proto.MarshalOptions{})
	require.NoError(t, err)

	duration := durationpb.New(20 * time.Second)
	empty := &emptypb.Empty{}
	fieldMask := &fieldmaskpb.FieldMask{
		Paths: []string{"foo_bar.baz", "fizz_buzz", "fizz.buzz._bar._baz._be_dazzle"},
	}
	structVal, err := structpb.NewStruct(map[string]interface{}{"field1": 1, "field2": 2})
	require.NoError(t, err)
	listValue, err := structpb.NewList([]interface{}{"abc", nil, true, 123, []interface{}{0, 1, 2}, map[string]interface{}{"field1": 1, "field2": 2}})
	require.NoError(t, err)
	timestamp := timestamppb.Now()

	wkt := &knittest.WellKnownTypes{
		Any:             anyMsg,
		AnyList:         []*anypb.Any{anyMsg, anyMsg},
		AnyMap:          map[string]*anypb.Any{"abc": anyMsg},
		Duration:        duration,
		DurationList:    []*durationpb.Duration{duration, duration},
		DurationMap:     map[int64]*durationpb.Duration{1010101: duration},
		Empty:           empty,
		EmptyList:       []*emptypb.Empty{empty, empty},
		EmptyMap:        map[string]*emptypb.Empty{"foo": empty},
		FieldMask:       fieldMask,
		FieldMaskList:   []*fieldmaskpb.FieldMask{fieldMask, fieldMask},
		FieldMaskMap:    map[uint32]*fieldmaskpb.FieldMask{123: fieldMask},
		Value:           structpb.NewListValue(listValue),
		ValueList:       []*structpb.Value{structpb.NewNumberValue(101), structpb.NewBoolValue(true), structpb.NewStringValue("abc")},
		ValueMap:        map[string]*structpb.Value{"bar": structpb.NewStructValue(structVal), "baz": structpb.NewNullValue()},
		ListValue:       listValue,
		ListValueList:   []*structpb.ListValue{listValue, listValue},
		ListValueMap:    map[uint64]*structpb.ListValue{202020202: listValue},
		Struct:          structVal,
		StructList:      []*structpb.Struct{structVal, structVal},
		StructMap:       map[string]*structpb.Struct{"abc": structVal},
		NullValue:       structpb.NullValue_NULL_VALUE,
		NullValueList:   []structpb.NullValue{structpb.NullValue_NULL_VALUE, structpb.NullValue_NULL_VALUE},
		NullValueMap:    map[bool]structpb.NullValue{true: structpb.NullValue_NULL_VALUE},
		Timestamp:       timestamp,
		TimestampList:   []*timestamppb.Timestamp{timestamp, timestamp},
		TimestampMap:    map[string]*timestamppb.Timestamp{"foo": timestamp},
		Int32Value:      wrapperspb.Int32(-101),
		Int32ValueList:  []*wrapperspb.Int32Value{wrapperspb.Int32(-1), wrapperspb.Int32(9999)},
		Int32ValueMap:   map[int32]*wrapperspb.Int32Value{-42: wrapperspb.Int32(42)},
		Int64Value:      wrapperspb.Int64(-101),
		Int64ValueList:  []*wrapperspb.Int64Value{wrapperspb.Int64(-1), wrapperspb.Int64(9999)},
		Int64ValueMap:   map[int64]*wrapperspb.Int64Value{-42: wrapperspb.Int64(42)},
		Uint32Value:     wrapperspb.UInt32(101),
		Uint32ValueList: []*wrapperspb.UInt32Value{wrapperspb.UInt32(1), wrapperspb.UInt32(9999)},
		Uint32ValueMap:  map[uint32]*wrapperspb.UInt32Value{42: wrapperspb.UInt32(42)},
		Uint64Value:     wrapperspb.UInt64(101),
		Uint64ValueList: []*wrapperspb.UInt64Value{wrapperspb.UInt64(1), wrapperspb.UInt64(9999)},
		Uint64ValueMap:  map[uint64]*wrapperspb.UInt64Value{42: wrapperspb.UInt64(42)},
		FloatValue:      wrapperspb.Float(123.456),
		FloatValueList:  []*wrapperspb.FloatValue{wrapperspb.Float(123.456), wrapperspb.Float(-987.654)},
		FloatValueMap:   map[int32]*wrapperspb.FloatValue{0: wrapperspb.Float(float32(math.Inf(1)))},
		DoubleValue:     wrapperspb.Double(math.NaN()),
		DoubleValueList: []*wrapperspb.DoubleValue{wrapperspb.Double(123.456), wrapperspb.Double(-987.654)},
		DoubleValueMap:  map[int64]*wrapperspb.DoubleValue{0: wrapperspb.Double(math.Inf(-1))},
		BoolValue:       wrapperspb.Bool(true),
		BoolValueList:   []*wrapperspb.BoolValue{wrapperspb.Bool(true), wrapperspb.Bool(false)},
		BoolValueMap:    map[bool]*wrapperspb.BoolValue{false: wrapperspb.Bool(true)},
		StringValue:     wrapperspb.String("abc"),
		StringValueList: []*wrapperspb.StringValue{wrapperspb.String("a"), wrapperspb.String("b")},
		StringValueMap:  map[string]*wrapperspb.StringValue{"a": wrapperspb.String("b")},
		BytesValue:      wrapperspb.Bytes([]byte("abc")),
		BytesValueList:  []*wrapperspb.BytesValue{wrapperspb.Bytes([]byte{12}), wrapperspb.Bytes([]byte{34})},
		BytesValueMap:   map[string]*wrapperspb.BytesValue{"a": wrapperspb.Bytes([]byte{101, 102, 103})},
	}
	wktClone := proto.Clone(wkt).(*knittest.WellKnownTypes) //nolint:forcetypeassert,errcheck
	wkt.NestedInList = []*knittest.WellKnownTypes{wktClone, wktClone, wktClone}
	wkt.NestedInMap = map[int32]*knittest.WellKnownTypes{101: wktClone, 202: wktClone}

	return wkt
}

func getScalarsExample() *knittest.Scalars {
	return &knittest.Scalars{
		I32:       123,
		I32List:   []int32{123, -234},
		I32Map:    map[int32]int32{-123: 234},
		I64:       12345,
		I64List:   []int64{12345, -23456},
		I64Map:    map[int64]int64{-12345: 23456},
		U32:       123,
		U32List:   []uint32{123, 234},
		U32Map:    map[uint32]uint32{123: 234},
		U64:       12345,
		U64List:   []uint64{12345, 23456},
		U64Map:    map[uint64]uint64{12345: 23456},
		S32:       123,
		S32List:   []int32{123, -234},
		S32Map:    map[int32]int32{-123: 234},
		S64:       12345,
		S64List:   []int64{12345, -23456},
		S64Map:    map[int64]int64{-12345: 23456},
		Sfx32:     123,
		Sfx32List: []int32{123, -234},
		Sfx32Map:  map[int32]int32{-123: 234},
		Sfx64:     12345,
		Sfx64List: []int64{12345, -23456},
		Sfx64Map:  map[int64]int64{-12345: 23456},
		Fx32:      123,
		Fx32List:  []uint32{123, 234},
		Fx32Map:   map[uint32]uint32{123: 234},
		Fx64:      12345,
		Fx64List:  []uint64{12345, 23456},
		Fx64Map:   map[uint64]uint64{12345: 23456},
		F32:       123.123,
		F32List:   []float32{123.123, 234.234},
		F32Map:    map[string]float32{"a": 234.234},
		F64:       12345.12345,
		F64List:   []float64{12345.12345, 23456.23456},
		F64Map:    map[string]float64{"a": 23456.23456},
		B:         true,
		BList:     []bool{true, false},
		BMap:      map[bool]bool{false: true},
		Byt:       []byte{1, 2, 3, 4},
		BytList:   [][]byte{{1, 2, 3, 4}, {9, 8, 7, 6}},
		BytMap:    map[string][]byte{"a": {1, 2, 3, 4}},
		Str:       "abc",
		StrList:   []string{"abc", "def"},
		StrMap:    map[string]string{"abc": "def"},
	}
}

func getFooExample() *knittest.Foo {
	return &knittest.Foo{
		Name:         "abc",
		Id:           123,
		State:        knittest.FooState_FOO_STATE_READY,
		Tags:         []string{"stale", "do-not-use"},
		Attributes:   map[string]string{"validated": "yesterday", "birthday": "today"},
		StateHistory: []knittest.FooState{knittest.FooState_FOO_STATE_INIT, knittest.FooState_FOO_STATE_READY},
		ConstructedWith: &knittest.Foo_Params{
			InitFrom:     &knittest.Foo_Params_Name{Name: "other"},
			RequestAttrs: map[string]string{"birthday": "today"},
			InitState:    knittest.FooState_FOO_STATE_INIT,
		},
	}
}

func getFizzExample() *knittest.Fizz {
	return &knittest.Fizz{
		Color:   "red",
		Density: 1.052,
		Depth:   proto.Int32(3),
		Bars: []*knittest.Bar{
			{
				Uid:       12345,
				Threshold: 9.99,
				Purpose:   "be",
			},
			{
				Uid:       23456,
				Threshold: 8.88,
				Purpose:   "learn",
			},
			{
				Uid:       34567,
				Threshold: 7.77,
				Purpose:   "grow",
			},
		},
		BarsByCategory: map[string]*knittest.Bar{
			"A1": {
				Uid:       45678,
				Threshold: 6.66,
				Purpose:   "live",
			},
			"B2": {
				Uid:       56789,
				Threshold: 5.55,
				Purpose:   "create",
			},
		},
	}
}

func getExhaustiveMaskForWellKnownTypes() []*gatewayv1alpha1.MaskField {
	// generate an exhaustive field mask, to test the serialization for all types
	fields := (*knittest.WellKnownTypes)(nil).ProtoReflect().Descriptor().Fields()
	var mask []*gatewayv1alpha1.MaskField
	for i := 0; i < fields.Len(); i++ {
		if fields.Get(i).Number() <= 2 {
			// skip the nested fields for now
			continue
		}
		mask = append(mask, &gatewayv1alpha1.MaskField{
			Name: camelCase(string(fields.Get(i).Name())),
		})
	}
	return append(mask,
		&gatewayv1alpha1.MaskField{Name: "nestedInList", Mask: mask},
		&gatewayv1alpha1.MaskField{Name: "nestedInMap", Mask: mask},
	)
}

func getExhaustiveMaskForScalars() []*gatewayv1alpha1.MaskField {
	// generate an exhaustive field mask, to test the serialization for all types
	fields := (*knittest.Scalars)(nil).ProtoReflect().Descriptor().Fields()
	var mask []*gatewayv1alpha1.MaskField
	for i := 0; i < fields.Len(); i++ {
		mask = append(mask, &gatewayv1alpha1.MaskField{
			Name: camelCase(string(fields.Get(i).Name())),
		})
	}
	return mask
}

func getExhaustiveFooMask() []*gatewayv1alpha1.MaskField {
	// References all fields and relations of Foo (other than WellKnownTypes)
	return []*gatewayv1alpha1.MaskField{
		{
			Name: "name",
		},
		{
			Name: "id",
		},
		{
			Name: "state",
		},
		{
			Name: "tags",
		},
		{
			Name: "attributes",
		},
		{
			Name: "stateHistory",
		},
		{
			Name: "constructedWith",
			Mask: []*gatewayv1alpha1.MaskField{
				{
					Name: "name",
				},
				{
					Name: "id",
				},
				{
					Name: "requestAttrs",
				},
				{
					Name: "initState",
				},
			},
		},
		{
			Name: "style",
		},
		{
			Name: "balanceCents",
		},
		{
			Name: "bars",
			Mask: getExhaustiveBarMask(),
		},
	}
}

func getExhaustiveFizzMask() []*gatewayv1alpha1.MaskField {
	return []*gatewayv1alpha1.MaskField{
		{
			Name: "color",
		},
		{
			Name: "density",
		},
		{
			Name: "depth",
		},
		{
			Name: "bars",
			Mask: getExhaustiveBarMask(),
		},
		{
			Name: "barsByCategory",
			Mask: getExhaustiveBarMask(),
		},
		{
			Name: "buzzes",
			Mask: []*gatewayv1alpha1.MaskField{
				{
					Name: "volume",
				},
				{
					Name: "clipReference",
				},
				{
					Name: "startAt",
				},
				{
					Name: "endAt",
				},
			},
		},
	}
}

func getExhaustiveBarMask() []*gatewayv1alpha1.MaskField {
	return []*gatewayv1alpha1.MaskField{
		{
			Name: "uid",
		},
		{
			Name: "threshold",
		},
		{
			Name: "purpose",
		},
		{
			Name: "baz",
			Mask: []*gatewayv1alpha1.MaskField{
				{
					Name: "thingies",
				},
				{
					Name: "en",
				},
				{
					Name: "sequence",
				},
			},
		},
		{
			Name: "bedazzles",
			Mask: []*gatewayv1alpha1.MaskField{
				{
					Name: "brightness",
				},
				{
					Name: "pattern",
				},
				{
					Name: "duration",
				},
				{
					Name: "frequency",
				},
			},
		},
	}
}
