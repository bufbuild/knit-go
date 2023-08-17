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
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"sync/atomic"
	"testing"
	"time"

	gatewayv1alpha1 "buf.build/gen/go/bufbuild/knit/protocolbuffers/go/buf/knit/gateway/v1alpha1"
	connect "connectrpc.com/connect"
	"github.com/bufbuild/knit-go/internal/gen/buf/knittest"
	"github.com/bufbuild/knit-go/internal/gen/buf/knittest/knittestconnect"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	// Set this to true to re-generate golden outputs. One can then diff the
	// changes to the outputs to make sure new logic is as expected.
	regenerateGoldenFiles = false
)

func TestStitcher(t *testing.T) {
	t.Parallel()

	route, err := url.Parse("https://foo.bar.baz/")
	require.NoError(t, err)
	gateway := &Gateway{Route: route, MaxParallelismPerRequest: 2}
	err = gateway.AddServiceByName(knittestconnect.FooServiceName)
	require.NoError(t, err)
	var peak peakMonitor

	// Configure some resolvers. They must be deterministic functions.
	var styleCalls atomic.Int32
	conf := newRelationFooStyle(t,
		func(ctx context.Context, resCtx *resolveMeta, foo *knittest.Foo, _ *knittest.GetFooStyleRequest) (*string, error) {
			peak.Increment()
			defer peak.Decrement()
			styleCalls.Add(1)

			// Artificial delay to max out parallelism
			time.Sleep(100 * time.Millisecond)

			if foo.Id%2 == 0 {
				return nil, nil //nolint:nilnil
			}
			return proto.String("style:" + foo.Name), nil
		},
	)
	styleBatches := measureBatchSizes(conf)
	err = gateway.addRelation(conf)
	require.NoError(t, err)

	var balanceCentsCalls atomic.Int32
	conf = newRelationFooBalanceCents(t,
		func(ctx context.Context, resCtx *resolveMeta, foo *knittest.Foo, _ *knittest.GetFooBalanceCentsRequest) (uint64, error) {
			peak.Increment()
			defer peak.Decrement()
			balanceCentsCalls.Add(1)

			return foo.Id/10 + 123, nil
		},
	)
	balanceCentsBatches := measureBatchSizes(conf)
	err = gateway.addRelation(conf)
	require.NoError(t, err)

	fooBarParams := &knittest.GetFooBarsRequest{UpToThreshold: 102030405060, Limit: 3}
	var barCalls atomic.Int32
	conf = newRelationFooBars(t,
		func(ctx context.Context, resCtx *resolveMeta, foo *knittest.Foo, params *knittest.GetFooBarsRequest) ([]*knittest.Bar, error) {
			peak.Increment()
			defer peak.Decrement()
			barCalls.Add(1)
			diff := cmp.Diff(fooBarParams, params, protocmp.Transform())
			assert.Empty(t, diff)

			num := len(foo.Tags)
			if num > int(params.Limit) {
				num = int(params.Limit)
			}
			results := make([]*knittest.Bar, num)
			for i := 0; i < num; i++ {
				results[i] = &knittest.Bar{
					Uid:       int64(foo.Id*100 + uint64(i)),
					Purpose:   foo.Tags[i],
					Threshold: float64(len(foo.Attributes)) / 10,
				}
			}
			return results, nil
		},
	)
	barBatches := measureBatchSizes(conf)
	err = gateway.addRelation(conf)
	require.NoError(t, err)

	var descriptionCalls atomic.Int32
	conf = newRelationFooDescription(t,
		func(_ context.Context, _ *resolveMeta, _ *knittest.Foo, _ *knittest.GetFooDescriptionRequest) (string, error) {
			peak.Increment()
			defer peak.Decrement()
			descriptionCalls.Add(1)

			return "", connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("description not available"))
		},
	)
	descriptionBatches := measureBatchSizes(conf)
	err = gateway.addRelation(conf)
	require.NoError(t, err)

	var bazCalls atomic.Int32
	conf = newRelationBarBaz(t,
		func(ctx context.Context, resCtx *resolveMeta, bar *knittest.Bar, _ *knittest.GetBarBazRequest) (*knittest.Baz, error) {
			peak.Increment()
			defer peak.Decrement()
			bazCalls.Add(1)

			num := int(bar.Uid) % 10
			things := make([]string, num)
			for j := 0; j < num; j++ {
				things[j] = bar.Purpose
			}
			seq := map[string]knittest.Baz_Enum{}
			if bar.Uid%2 == 0 {
				seq["abc"] = knittest.Baz_UNO
				seq["def"] = knittest.Baz_DOS
			} else {
				seq["xyz"] = knittest.Baz_UNO
				seq["uvw"] = knittest.Baz_DOS
			}
			return &knittest.Baz{
				Thingies: things,
				En:       knittest.Baz_Enum(bar.Uid),
				Sequence: seq,
			}, nil
		},
	)
	bazBatches := measureBatchSizes(conf)
	err = gateway.addRelation(conf)
	require.NoError(t, err)

	barBedazzleParams := &knittest.GetBarBedazzlesRequest{Limit: 2}
	var bedazzleCalls atomic.Int32
	conf = newRelationBarBedazzles(t,
		func(ctx context.Context, resCtx *resolveMeta, bar *knittest.Bar, params *knittest.GetBarBedazzlesRequest) ([]*knittest.Bedazzle, error) {
			peak.Increment()
			defer peak.Decrement()
			bedazzleCalls.Add(1)
			diff := cmp.Diff(barBedazzleParams, params, protocmp.Transform())
			assert.Empty(t, diff)

			num := len(bar.Purpose) / 2
			if num > int(params.Limit) {
				num = int(params.Limit)
			}
			results := make([]*knittest.Bedazzle, num)
			for i := 0; i < num; i++ {
				results[i] = &knittest.Bedazzle{
					Brightness: float32(bar.Threshold * 10),
					Pattern:    knittest.Bedazzle_Pattern(bar.Uid % 7),
					Duration:   durationpb.New(time.Duration(float64(time.Second) * bar.Threshold)),
					Frequency:  bar.Threshold/2 + 100,
				}
			}
			return results, nil
		},
	)
	bedazzleBatches := measureBatchSizes(conf)
	err = gateway.addRelation(conf)
	require.NoError(t, err)

	// 10 results, alternating even/odd IDs, varying numbers of tags and attributes
	resp := &knittest.GetFooResponse{
		Results: getFooData(),
	}

	mask := []*gatewayv1alpha1.MaskField{
		{
			Name: "results",
			Mask: getFooMaskWithParams(t, fooBarParams, barBedazzleParams),
		},
	}
	val, patches, err := formatMessage(resp, &resolveMeta{}, mask, []string{"buf.knittest.FooService.GetFoo"}, nil, false, nil)
	require.NoError(t, err)
	stitcher := newStitcher(gateway)
	err = stitcher.stitch(context.Background(), [][]*patch{patches}, false)
	require.NoError(t, err)

	checkGoldenFile(t, "stitcher.txt", format(t, val))

	require.Equal(t, int32(gateway.MaxParallelismPerRequest), peak.Peak())

	// each relation is resolved with a single batch
	require.Equal(t, []int{10}, balanceCentsBatches.AsSlice())
	require.Equal(t, 10, int(balanceCentsCalls.Load()))
	require.Equal(t, []int{10}, styleBatches.AsSlice())
	require.Equal(t, 10, int(styleCalls.Load()))
	require.Equal(t, []int{10}, barBatches.AsSlice())
	require.Equal(t, 10, int(barCalls.Load()))
	require.Equal(t, []int{10}, descriptionBatches.AsSlice())
	require.Equal(t, 1, int(descriptionCalls.Load())) // only called once, which fails, so rest in batch fail also
	require.Equal(t, []int{24}, bazBatches.AsSlice())
	require.Equal(t, 24, int(bazCalls.Load()))
	require.Equal(t, []int{24}, bedazzleBatches.AsSlice())
	require.Equal(t, 24, int(bedazzleCalls.Load()))
}

type peakMonitor struct {
	val, peak atomic.Int32
}

func (m *peakMonitor) Increment() {
	current := m.val.Add(1)
	for {
		peak := m.peak.Load()
		if current <= peak {
			return
		}
		if m.peak.CompareAndSwap(peak, current) {
			return
		}
	}
}

func (m *peakMonitor) Decrement() {
	m.val.Add(-1)
}

func (m *peakMonitor) Peak() int32 {
	return m.peak.Load()
}

type batchSizes struct {
	head atomic.Pointer[batchSizeEntry]
}

func measureBatchSizes(conf *relationConfig) *batchSizes {
	var batches batchSizes
	fn := conf.resolver
	conf.resolver = func(ctx context.Context, resMeta *resolveMeta, bases []proto.Message, params proto.Message) ([]protoreflect.Value, error) {
		batches.Add(len(bases))
		return fn(ctx, resMeta, bases, params)
	}
	return &batches
}

type batchSizeEntry struct {
	size int
	next atomic.Pointer[batchSizeEntry]
}

func (s *batchSizes) Add(size int) {
	entry := &batchSizeEntry{size: size}
	for {
		head := s.head.Load()
		entry.next.Store(head)
		if s.head.CompareAndSwap(head, entry) {
			return
		}
	}
}

func (s *batchSizes) AsSlice() []int {
	slice := s.head.Load().AsSlice(nil)
	// Batches added concurrently so could be in any order. So
	// sort the list descending: should usually be all the same
	// size but the last, which may be smaller.
	sort.Slice(slice, func(i, j int) bool {
		return slice[i] > slice[j]
	})
	return slice
}

func (s *batchSizeEntry) AsSlice(soFar []int) []int {
	soFar = append(soFar, s.size)
	next := s.next.Load()
	if next != nil {
		return next.AsSlice(soFar)
	}
	return soFar
}

func checkGoldenFile(t *testing.T, goldenFileName, fileContents string) {
	t.Helper()
	path := filepath.Join("internal/testdata", goldenFileName)
	if regenerateGoldenFiles {
		err := os.WriteFile(path, []byte(fileContents), 0666) //nolint:gosec
		require.NoError(t, err)
		return
	}
	actual, err := os.ReadFile(path)
	require.NoError(t, err)
	diff := cmp.Diff(fileContents, string(actual))
	require.True(t, diff == "", "difference in output for %q:\n%s", path, diff)
}

func getFooData() []*knittest.Foo {
	// 10 results, alternating even/odd IDs, varying numbers of tags and attributes
	return []*knittest.Foo{
		{
			Name:       "abc",
			Id:         123,
			Tags:       []string{"abc"},
			Attributes: map[string]string{"abc": "def"},
		},
		{
			Name:       "def",
			Id:         234,
			Tags:       []string{"abc", "defdef"},
			Attributes: map[string]string{"abc": "def", "def": "ghi"},
		},
		{
			Name:       "ghi",
			Id:         345,
			Tags:       []string{"abc", "defdef", "ghighighi"},
			Attributes: map[string]string{"abc": "def", "def": "ghi", "jkl": "mno"},
		},
		{
			Name:       "jkl",
			Id:         456,
			Tags:       []string{"abc", "defdef", "ghighighi", "jkljkljkljkl"},
			Attributes: map[string]string{"abc": "def", "def": "ghi", "jkl": "mno", "pqr": "stu"},
		},
		{
			Name:       "mno",
			Id:         567,
			Tags:       []string{"abc", "defdef", "ghighighi", "jkljkljkljkl", "mnomnomnomnomno"},
			Attributes: map[string]string{"abc": "def", "def": "ghi", "jkl": "mno", "pqr": "stu", "vwx": "yz"},
		},
		{
			Name:       "pqr",
			Id:         678,
			Tags:       []string{"abc"},
			Attributes: map[string]string{"abc": "def"},
		},
		{
			Name:       "stu",
			Id:         789,
			Tags:       []string{"abc", "def"},
			Attributes: map[string]string{"abc": "def", "def": "ghi"},
		},
		{
			Name:       "vwx",
			Id:         890,
			Tags:       []string{"abc", "def", "ghi"},
			Attributes: map[string]string{"abc": "def", "def": "ghi", "jkl": "mno"},
		},
		{
			Name:       "yz",
			Id:         901,
			Tags:       []string{"abc", "def", "ghi", "jkl"},
			Attributes: map[string]string{"abc": "def", "def": "ghi", "jkl": "mno", "pqr": "stu"},
		},
		{
			Name:       "123",
			Id:         12,
			Tags:       []string{"abc", "def", "ghi", "jkl", "mno"},
			Attributes: map[string]string{"abc": "def", "def": "ghi", "jkl": "mno", "pqr": "stu", "vwx": "yz"},
		},
	}
}

func getFooMaskWithParams(t *testing.T, fooBarParams *knittest.GetFooBarsRequest, barBedazzleParams *knittest.GetBarBedazzlesRequest) []*gatewayv1alpha1.MaskField {
	t.Helper()

	fooBarParamsAsJSON := format(t, fooBarParams)
	fooBarParamsAsValue := &structpb.Value{}
	err := protojson.Unmarshal([]byte(fooBarParamsAsJSON), fooBarParamsAsValue)
	require.NoError(t, err)
	barBedazzleParamsAsJSON := format(t, barBedazzleParams)
	barBedazzleParamsAsValue := &structpb.Value{}
	err = protojson.Unmarshal([]byte(barBedazzleParamsAsJSON), barBedazzleParamsAsValue)
	require.NoError(t, err)

	return []*gatewayv1alpha1.MaskField{
		{Name: "name"},
		{Name: "id"},
		{Name: "state"},
		{Name: "tags"},
		{Name: "attributes"},
		{Name: "style"},
		{Name: "balanceCents"},
		{Name: "description", OnError: &gatewayv1alpha1.MaskField_Catch{Catch: &gatewayv1alpha1.Catch{}}},
		{
			Name:   "bars",
			Params: fooBarParamsAsValue,
			Mask: []*gatewayv1alpha1.MaskField{
				{Name: "uid"},
				{Name: "threshold"},
				{Name: "purpose"},
				{
					Name: "baz",
					Mask: []*gatewayv1alpha1.MaskField{
						{Name: "thingies"},
						{Name: "en"},
						{Name: "sequence"},
					},
				},
				{
					Name:   "bedazzles",
					Params: barBedazzleParamsAsValue,
					Mask: []*gatewayv1alpha1.MaskField{
						{Name: "brightness"},
						{Name: "pattern"},
						{Name: "duration"},
						{Name: "frequency"},
					},
				},
			},
		},
	}
}

func newRelationFooStyle(
	t *testing.T,
	resolve func(context.Context, *resolveMeta, *knittest.Foo, *knittest.GetFooStyleRequest) (*string, error),
) *relationConfig {
	t.Helper()
	descriptor, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(toMethodName(knittestconnect.RelationsServiceGetFooStyleProcedure)))
	require.NoError(t, err)
	mtd, ok := descriptor.(protoreflect.MethodDescriptor)
	require.True(t, ok)
	relConfig, err := getRelationConfig(mtd, "style", nil)
	require.NoError(t, err)
	relConfig.resolver = asResolver(resolve, func(resp *knittest.GetFooStyleResponse_FooResult, val *string) {
		resp.Style = val
	})
	return relConfig
}

func newRelationFooBalanceCents(
	t *testing.T,
	resolve func(context.Context, *resolveMeta, *knittest.Foo, *knittest.GetFooBalanceCentsRequest) (uint64, error),
) *relationConfig {
	t.Helper()
	descriptor, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(toMethodName(knittestconnect.RelationsServiceGetFooBalanceCentsProcedure)))
	require.NoError(t, err)
	mtd, ok := descriptor.(protoreflect.MethodDescriptor)
	require.True(t, ok)
	relConfig, err := getRelationConfig(mtd, "balance_cents", nil)
	require.NoError(t, err)
	relConfig.resolver = asResolver(resolve, func(resp *knittest.GetFooBalanceCentsResponse_FooResult, val uint64) {
		resp.BalanceCents = val
	})
	return relConfig
}

func newRelationFooBars(
	t *testing.T,
	resolve func(context.Context, *resolveMeta, *knittest.Foo, *knittest.GetFooBarsRequest) ([]*knittest.Bar, error),
) *relationConfig {
	t.Helper()
	descriptor, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(toMethodName(knittestconnect.RelationsServiceGetFooBarsProcedure)))
	require.NoError(t, err)
	mtd, ok := descriptor.(protoreflect.MethodDescriptor)
	require.True(t, ok)
	relConfig, err := getRelationConfig(mtd, "bars", nil)
	require.NoError(t, err)
	relConfig.resolver = asResolver(resolve, func(resp *knittest.GetFooBarsResponse_FooResult, val []*knittest.Bar) {
		resp.Bars = val
	})
	return relConfig
}

func newRelationFooDescription(
	t *testing.T,
	resolve func(context.Context, *resolveMeta, *knittest.Foo, *knittest.GetFooDescriptionRequest) (string, error),
) *relationConfig {
	t.Helper()
	descriptor, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(toMethodName(knittestconnect.RelationsServiceGetFooDescriptionProcedure)))
	require.NoError(t, err)
	mtd, ok := descriptor.(protoreflect.MethodDescriptor)
	require.True(t, ok)
	relConfig, err := getRelationConfig(mtd, "description", nil)
	require.NoError(t, err)
	relConfig.resolver = asResolver(resolve, func(resp *knittest.GetFooDescriptionResponse_FooResult, val string) {
		resp.Description = val
	})
	return relConfig
}

func newRelationBarBaz(
	t *testing.T,
	resolve func(context.Context, *resolveMeta, *knittest.Bar, *knittest.GetBarBazRequest) (*knittest.Baz, error),
) *relationConfig {
	t.Helper()
	descriptor, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(toMethodName(knittestconnect.RelationsServiceGetBarBazProcedure)))
	require.NoError(t, err)
	mtd, ok := descriptor.(protoreflect.MethodDescriptor)
	require.True(t, ok)
	relConfig, err := getRelationConfig(mtd, "baz", nil)
	require.NoError(t, err)
	relConfig.resolver = asResolver(resolve, func(resp *knittest.GetBarBazResponse_BarResult, val *knittest.Baz) {
		resp.Baz = val
	})
	return relConfig
}

func newRelationBarBedazzles(
	t *testing.T,
	resolve func(context.Context, *resolveMeta, *knittest.Bar, *knittest.GetBarBedazzlesRequest) ([]*knittest.Bedazzle, error),
) *relationConfig {
	t.Helper()
	descriptor, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(toMethodName(knittestconnect.RelationsServiceGetBarBedazzlesProcedure)))
	require.NoError(t, err)
	mtd, ok := descriptor.(protoreflect.MethodDescriptor)
	require.True(t, ok)
	relConfig, err := getRelationConfig(mtd, "bedazzles", nil)
	require.NoError(t, err)
	relConfig.resolver = asResolver(resolve, func(resp *knittest.GetBarBedazzlesResponse_BarResult, val []*knittest.Bedazzle) {
		resp.Bedazzles = val
	})
	return relConfig
}

func newRelationFizzBuzzes(
	t *testing.T,
	resolve func(context.Context, *resolveMeta, *knittest.Fizz, *knittest.GetFizzBuzzesRequest) (map[string]*knittest.Buzz, error),
) *relationConfig {
	t.Helper()
	descriptor, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(toMethodName(knittestconnect.RelationsServiceGetFizzBuzzesProcedure)))
	require.NoError(t, err)
	mtd, ok := descriptor.(protoreflect.MethodDescriptor)
	require.True(t, ok)
	relConfig, err := getRelationConfig(mtd, "buzzes", nil)
	require.NoError(t, err)
	relConfig.resolver = asResolver(resolve, func(resp *knittest.GetFizzBuzzesResponse_FizzResult, val map[string]*knittest.Buzz) {
		resp.Buzzes = val
	})
	return relConfig
}

//nolint:errcheck,forcetypeassert
func asResolver[B, P, R proto.Message, T any](
	resolve func(ctx context.Context, resCtx *resolveMeta, base B, params P) (T, error),
	setResult func(R, T),
) relationResolver {
	return func(ctx context.Context, resCtx *resolveMeta, bases []proto.Message, params proto.Message) ([]protoreflect.Value, error) {
		results := make([]protoreflect.Value, len(bases))
		var typedParams P
		typedParams = typedParams.ProtoReflect().New().Interface().(P)
		if err := convert(params, typedParams); err != nil {
			return nil, err
		}
		for i, base := range bases {
			var typedBase B
			typedBase = typedBase.ProtoReflect().New().Interface().(B)
			if err := convert(base, typedBase); err != nil {
				return nil, err
			}
			val, err := resolve(ctx, resCtx, typedBase, typedParams)
			if err != nil {
				return nil, err
			}
			var typedResp R
			typedRespRef := typedResp.ProtoReflect().New()
			typedResp = typedRespRef.Interface().(R)
			setResult(typedResp, val)
			field := typedRespRef.Descriptor().Fields().ByNumber(1)
			if !field.HasPresence() || typedRespRef.Has(field) {
				results[i] = typedRespRef.Get(field)
			}
		}
		return results, nil
	}
}

func convert(src, dest proto.Message) error {
	data, err := proto.Marshal(src)
	if err != nil {
		return err
	}
	return proto.Unmarshal(data, dest)
}
