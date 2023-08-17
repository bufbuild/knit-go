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
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"buf.build/gen/go/bufbuild/knit/connectrpc/go/buf/knit/gateway/v1alpha1/gatewayv1alpha1connect"
	gatewayv1alpha1 "buf.build/gen/go/bufbuild/knit/protocolbuffers/go/buf/knit/gateway/v1alpha1"
	connect "connectrpc.com/connect"
	"github.com/bufbuild/knit-go/internal/gen/buf/knittest"
	"github.com/bufbuild/knit-go/internal/gen/buf/knittest/knittestconnect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestEndToEnd(t *testing.T) {
	t.Parallel()

	expectPresentHeaders := map[string]string{
		"foo":           "bar",
		"baz":           "bedazzle",
		"fizz":          "buzz",
		"authorization": "bearer lk1234ljk1ewlkj1",
	}
	expectMissingHeaders := map[string]string{
		"accept-foo":       "bar",
		"content-blah":     "baz",
		"connect-frobnitz": "buzz",
	}
	methodsObserved := &sync.Map{}

	// start backend server
	svrListen, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	mux := http.NewServeMux()
	svrImpl := newServerImpl(expectPresentHeaders, expectMissingHeaders, methodsObserved)
	mux.Handle(knittestconnect.NewFooServiceHandler(svrImpl))
	mux.Handle(knittestconnect.NewFizzServiceHandler(svrImpl))
	mux.Handle(knittestconnect.NewRelationsServiceHandler(svrImpl))
	svr := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: time.Second,
	}
	t.Cleanup(func() {
		_ = svr.Shutdown(context.Background())
	})
	go func() {
		_ = svr.Serve(svrListen)
	}()
	target, err := url.Parse(fmt.Sprintf("http://" + svrListen.Addr().String()))
	require.NoError(t, err)

	// create and configure the knit gateway
	gateway := &Gateway{
		Route: target,
	}
	err = gateway.AddServiceByName(knittestconnect.FooServiceName)
	require.NoError(t, err)
	err = gateway.AddServiceByName(knittestconnect.FizzServiceName)
	require.NoError(t, err)
	err = gateway.AddServiceByName(knittestconnect.RelationsServiceName)
	require.NoError(t, err)

	// Start the knit gateway as a server
	gatewayListen, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	mux = http.NewServeMux()
	mux.Handle(gateway.AsHandler())
	gatewaySvr := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: time.Second,
	}
	t.Cleanup(func() {
		_ = gatewaySvr.Shutdown(context.Background())
	})
	go func() {
		_ = gatewaySvr.Serve(gatewayListen)
	}()

	fooBarParams := &knittest.GetFooBarsRequest{UpToThreshold: 102030405060, Limit: 3}
	barBedazzleParams := &knittest.GetBarBedazzlesRequest{Limit: 2}
	fooMask := getFooMaskWithParams(t, fooBarParams, barBedazzleParams)
	i := 0
	for _, m := range fooMask {
		// remove the two fields for relations that don't have RPCs
		if m.Name == "style" || m.Name == "balanceCents" {
			continue
		}
		fooMask[i] = m
		i++
	}
	fooMask = fooMask[:i]

	// now we can run the tests
	knitClient := gatewayv1alpha1connect.NewKnitServiceClient(http.DefaultClient, fmt.Sprintf("http://"+gatewayListen.Addr().String()))
	ctx := context.Background()

	addHeaders := func(headers http.Header) {
		for k, v := range expectPresentHeaders {
			headers.Set(k, v)
		}
		for k, v := range expectMissingHeaders {
			headers.Set(k, v)
		}
	}

	t.Run("fetch", func(t *testing.T) {
		t.Parallel()

		req := connect.NewRequest(&gatewayv1alpha1.FetchRequest{
			Requests: []*gatewayv1alpha1.Request{
				{
					Method: knittestconnect.FooServiceName + ".GetFoo",
					Body: structpb.NewStructValue(&structpb.Struct{
						Fields: map[string]*structpb.Value{},
					}),
					Mask: []*gatewayv1alpha1.MaskField{
						{
							Name: "results",
							Mask: fooMask,
						},
					},
				},
				{
					Method: knittestconnect.FizzServiceName + ".GetFizz",
					Body: structpb.NewStructValue(&structpb.Struct{
						Fields: map[string]*structpb.Value{},
					}),
					Mask: []*gatewayv1alpha1.MaskField{
						{
							Name: "result",
							Mask: getExhaustiveFizzMask(),
						},
					},
				},
			},
		})
		addHeaders(req.Header())
		resp, err := knitClient.Fetch(ctx, req)
		require.NoError(t, err)
		checkGoldenFile(t, "e2e-fetch.txt", format(t, resp.Msg))
	})
	t.Run("do", func(t *testing.T) {
		t.Parallel()
		throwFooMask := make([]*gatewayv1alpha1.MaskField, 0, len(fooMask))
		for _, m := range fooMask {
			if m.Name == "description" {
				m, _ = proto.Clone(m).(*gatewayv1alpha1.MaskField)
				m.OnError = &gatewayv1alpha1.MaskField_Throw{Throw: &gatewayv1alpha1.Throw{}}
			}
			throwFooMask = append(throwFooMask, m)
		}
		req := connect.NewRequest(&gatewayv1alpha1.DoRequest{
			Requests: []*gatewayv1alpha1.Request{
				{
					Method: knittestconnect.FooServiceName + ".MutateFoo",
					Body: structpb.NewStructValue(&structpb.Struct{
						Fields: map[string]*structpb.Value{},
					}),
					Mask: []*gatewayv1alpha1.MaskField{
						{
							Name: "results",
							Mask: throwFooMask,
						},
					},
				},
				{
					Method: knittestconnect.FooServiceName + ".MutateFoo",
					Body: structpb.NewStructValue(&structpb.Struct{
						Fields: map[string]*structpb.Value{},
					}),
					Mask: []*gatewayv1alpha1.MaskField{
						{
							Name: "results",
							Mask: fooMask,
						},
					},
				},
			},
		})
		addHeaders(req.Header())
		resp, err := knitClient.Do(ctx, req)
		require.NoError(t, err)
		checkGoldenFile(t, "e2e-do.txt", format(t, resp.Msg))
	})
	t.Run("listen", func(t *testing.T) {
		t.Parallel()
		req := connect.NewRequest(&gatewayv1alpha1.ListenRequest{
			Request: &gatewayv1alpha1.Request{
				Method: knittestconnect.FooServiceName + ".QueryFoos",
				Body: structpb.NewStructValue(&structpb.Struct{
					Fields: map[string]*structpb.Value{},
				}),
				Mask: []*gatewayv1alpha1.MaskField{
					{
						Name: "foo",
						Mask: fooMask,
					},
				},
			},
		})
		addHeaders(req.Header())
		stream, err := knitClient.Listen(ctx, req)
		require.NoError(t, err)
		var buf bytes.Buffer
		for {
			if !stream.Receive() {
				break
			}
			buf.WriteString(format(t, stream.Msg()))
			buf.WriteString("\n")
		}
		require.NoError(t, stream.Err())
		checkGoldenFile(t, "e2e-listen.txt", buf.String())
	})

	t.Cleanup(func() {
		var methods []string
		methodsObserved.Range(func(k, v any) bool {
			methods = append(methods, k.(string)) //nolint:forcetypeassert
			return true
		})
		sort.Strings(methods)
		require.Equal(t, []string{
			"buf.knittest.FizzService.GetFizz",
			"buf.knittest.FooService.GetFoo",
			"buf.knittest.FooService.MutateFoo",
			"buf.knittest.FooService.QueryFoos",
			"buf.knittest.RelationsService.GetFooBars",
		}, methods)
	})
}

type serverImpl struct {
	knittestconnect.UnimplementedRelationsServiceHandler
	foos              []*knittest.Foo
	barsByFooID       map[uint64][]*knittest.Bar
	bazesByBarID      map[int64]*knittest.Baz
	bedazzlesByBarID  map[int64][]*knittest.Bedazzle
	fizz              *knittest.Fizz
	buzzes            map[string]*knittest.Buzz
	presentHeaders    map[string]string
	missingHeaders    map[string]string
	observedResolvers *sync.Map
}

var _ knittestconnect.FooServiceHandler = (*serverImpl)(nil)
var _ knittestconnect.FizzServiceHandler = (*serverImpl)(nil)

func newServerImpl(presentHeaders, missingHeaders map[string]string, observedResolvers *sync.Map) *serverImpl {
	svr := &serverImpl{
		barsByFooID:       map[uint64][]*knittest.Bar{},
		bazesByBarID:      map[int64]*knittest.Baz{},
		bedazzlesByBarID:  map[int64][]*knittest.Bedazzle{},
		presentHeaders:    presentHeaders,
		missingHeaders:    missingHeaders,
		observedResolvers: observedResolvers,
	}

	svr.foos = getFooData()

	// The relations descending from Foo are derived in the same way as in stitcher_test.go
	addBarRelations := func(bar *knittest.Bar) {
		// Baz
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
		svr.bazesByBarID[bar.Uid] = &knittest.Baz{
			Thingies: things,
			En:       knittest.Baz_Enum(bar.Uid),
			Sequence: seq,
		}

		// Bedazzles
		num = len(bar.Purpose) / 2
		results := make([]*knittest.Bedazzle, num)
		for i := 0; i < num; i++ {
			results[i] = &knittest.Bedazzle{
				Brightness: float32(bar.Threshold * 10),
				Pattern:    knittest.Bedazzle_Pattern(bar.Uid % 7),
				Duration:   durationpb.New(time.Duration(float64(time.Second) * bar.Threshold)),
				Frequency:  bar.Threshold/2 + 100,
			}
		}
		svr.bedazzlesByBarID[bar.Uid] = results
	}

	for _, foo := range svr.foos {
		num := len(foo.Tags)
		for i := 0; i < num; i++ {
			// Bar
			bar := &knittest.Bar{
				Uid:       int64(foo.Id*100 + uint64(i)),
				Purpose:   foo.Tags[i],
				Threshold: float64(len(foo.Attributes)) / 10,
			}
			svr.barsByFooID[foo.Id] = append(svr.barsByFooID[foo.Id], bar)
			addBarRelations(bar)
		}
	}

	svr.fizz = getFizzExample()
	for _, bar := range svr.fizz.Bars {
		addBarRelations(bar)
	}
	for _, bar := range svr.fizz.BarsByCategory {
		addBarRelations(bar)
	}
	svr.buzzes = map[string]*knittest.Buzz{
		"low": {
			Volume:        2.123,
			ClipReference: "/foo/bar/clip.au",
		},
		"mid": {
			Volume:        3.495,
			ClipReference: "/foo/baz/clip.au",
			StartAt:       23234,
		},
		"high": {
			Volume:        9.999,
			ClipReference: "/foo/buzz/clip.au",
			StartAt:       10234,
			EndAt:         proto.Uint64(4059812),
		},
	}

	return svr
}

func (s *serverImpl) GetFoo(_ context.Context, req *connect.Request[knittest.GetFooRequest]) (*connect.Response[knittest.GetFooResponse], error) {
	if err := checkHeaders(s.presentHeaders, s.missingHeaders, req.Header()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&knittest.GetFooResponse{Results: s.foos}), nil
}

func (s *serverImpl) MutateFoo(_ context.Context, req *connect.Request[knittest.MutateFooRequest]) (*connect.Response[knittest.MutateFooResponse], error) {
	if err := checkHeaders(s.presentHeaders, s.missingHeaders, req.Header()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&knittest.MutateFooResponse{Results: s.foos}), nil
}

func (s *serverImpl) QueryFoos(_ context.Context, req *connect.Request[knittest.QueryFoosRequest], stream *connect.ServerStream[knittest.QueryFoosResponse]) error {
	if err := checkHeaders(s.presentHeaders, s.missingHeaders, req.Header()); err != nil {
		return err
	}
	for _, foo := range s.foos {
		if err := stream.Send(&knittest.QueryFoosResponse{Foo: foo}); err != nil {
			return err
		}
	}
	return nil
}

func (s *serverImpl) GetFizz(_ context.Context, req *connect.Request[knittest.GetFizzRequest]) (*connect.Response[knittest.GetFizzResponse], error) {
	if err := checkHeaders(s.presentHeaders, s.missingHeaders, req.Header()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&knittest.GetFizzResponse{Result: s.fizz}), nil
}

func (s *serverImpl) GetFooBars(_ context.Context, req *connect.Request[knittest.GetFooBarsRequest]) (*connect.Response[knittest.GetFooBarsResponse], error) {
	err := checkExpectedMeta(req.Header(), req.Spec().Procedure, s.presentHeaders, s.missingHeaders, s.observedResolvers)
	if err != nil {
		return nil, err
	}
	bases := req.Msg.Bases
	values := make([]*knittest.GetFooBarsResponse_FooResult, len(bases))
	for i, base := range bases {
		bars := s.barsByFooID[base.Id]
		if len(bars) > int(req.Msg.Limit) {
			bars = bars[:int(req.Msg.Limit)]
		}
		values[i] = &knittest.GetFooBarsResponse_FooResult{
			Bars: bars,
		}
	}
	return connect.NewResponse(&knittest.GetFooBarsResponse{Values: values}), nil
}

func (s *serverImpl) GetFooDescription(_ context.Context, _ *connect.Request[knittest.GetFooDescriptionRequest]) (*connect.Response[knittest.GetFooDescriptionResponse], error) {
	return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("description not available"))
}

func (s *serverImpl) GetBarBaz(_ context.Context, req *connect.Request[knittest.GetBarBazRequest]) (*connect.Response[knittest.GetBarBazResponse], error) {
	err := checkExpectedMeta(req.Header(), req.Spec().Procedure, s.presentHeaders, s.missingHeaders, s.observedResolvers)
	if err != nil {
		return nil, err
	}
	bases := req.Msg.Bases
	values := make([]*knittest.GetBarBazResponse_BarResult, len(bases))
	for i, base := range bases {
		baz := s.bazesByBarID[base.Uid]
		if baz == nil {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("bar with id %d not found", base.Uid))
		}
		values[i] = &knittest.GetBarBazResponse_BarResult{Baz: baz}
	}
	return connect.NewResponse(&knittest.GetBarBazResponse{Values: values}), nil
}

func (s *serverImpl) GetBarBedazzles(_ context.Context, req *connect.Request[knittest.GetBarBedazzlesRequest]) (*connect.Response[knittest.GetBarBedazzlesResponse], error) {
	err := checkExpectedMeta(req.Header(), req.Spec().Procedure, s.presentHeaders, s.missingHeaders, s.observedResolvers)
	if err != nil {
		return nil, err
	}
	bases := req.Msg.Bases
	values := make([]*knittest.GetBarBedazzlesResponse_BarResult, len(bases))
	for i, base := range bases {
		bedazzles, ok := s.bedazzlesByBarID[base.Uid]
		if !ok {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("bar with id %d not found", base.Uid))
		}
		values[i] = &knittest.GetBarBedazzlesResponse_BarResult{Bedazzles: bedazzles}
	}
	return connect.NewResponse(&knittest.GetBarBedazzlesResponse{Values: values}), nil
}

func (s *serverImpl) GetFizzBuzzes(_ context.Context, req *connect.Request[knittest.GetFizzBuzzesRequest]) (*connect.Response[knittest.GetFizzBuzzesResponse], error) {
	err := checkExpectedMeta(req.Header(), req.Spec().Procedure, s.presentHeaders, s.missingHeaders, s.observedResolvers)
	if err != nil {
		return nil, err
	}
	bases := req.Msg.Bases
	values := make([]*knittest.GetFizzBuzzesResponse_FizzResult, len(bases))
	for i := range bases {
		values[i] = &knittest.GetFizzBuzzesResponse_FizzResult{Buzzes: s.buzzes}
	}
	return connect.NewResponse(&knittest.GetFizzBuzzesResponse{Values: values}), nil
}

func checkExpectedMeta(hdrs http.Header, procedure string, presentHeaders, missingHeaders map[string]string, observedMethods *sync.Map) error {
	var errs []string

	ops := hdrs.Values(knitOperationsHeader)
	if len(ops) < 2 {
		errs = []string{fmt.Sprintf("knit-context header should have at least two values; got %v (%d)", ops, len(ops))}
	} else {
		errs = checkOperations(ops, procedure)
	}

	if len(ops) > 0 {
		observedMethods.Store(ops[len(ops)-1], nil)
	}

	if err := checkHeaders(presentHeaders, missingHeaders, hdrs); err != nil {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}

func checkOperations(actualOps []string, procedure string) (errs []string) {
	var expectedOps []string
	currentOp := toMethodName(procedure)
	actualOps = append(actualOps, currentOp)

	// make sure the Knit entry-point is correct based on the top-level method
	topLevelOp := actualOps[1]
	switch topLevelOp {
	case toMethodName(knittestconnect.FizzServiceGetFizzProcedure),
		toMethodName(knittestconnect.FooServiceGetFooProcedure):
		expectedOps = []string{toMethodName(gatewayv1alpha1connect.KnitServiceFetchProcedure), topLevelOp}
	case toMethodName(knittestconnect.FooServiceMutateFooProcedure):
		expectedOps = []string{toMethodName(gatewayv1alpha1connect.KnitServiceDoProcedure), topLevelOp}
	case toMethodName(knittestconnect.FooServiceQueryFoosProcedure):
		expectedOps = []string{toMethodName(gatewayv1alpha1connect.KnitServiceListenProcedure), topLevelOp}
	}

	// check call path for other relations
	if expectedOps != nil {
		switch procedure {
		case knittestconnect.RelationsServiceGetFooBarsProcedure:
			if topLevelOp == toMethodName(knittestconnect.FizzServiceGetFizzProcedure) {
				errs = append(errs, fmt.Sprintf("unexpected method %q in call path for relation %q", topLevelOp, currentOp))
			}
			expectedOps = append(expectedOps, currentOp)
		case knittestconnect.RelationsServiceGetFizzBuzzesProcedure:
			if topLevelOp != toMethodName(knittestconnect.FizzServiceGetFizzProcedure) {
				errs = append(errs, fmt.Sprintf("unexpected method %q in call path for relation %q", topLevelOp, currentOp))
			}
			expectedOps = append(expectedOps, currentOp)
		case knittestconnect.RelationsServiceGetBarBazProcedure,
			knittestconnect.RelationsServiceGetBarBedazzlesProcedure:
			if topLevelOp != toMethodName(knittestconnect.FizzServiceGetFizzProcedure) {
				expectedOps = append(expectedOps, toMethodName(knittestconnect.RelationsServiceGetFooBarsProcedure))
			}
			expectedOps = append(expectedOps, currentOp)
		default:
			errs = append(errs, fmt.Sprintf("unexpected relation: %s", currentOp))
		}
	}

	if expectedOps == nil {
		errs = append(errs, fmt.Sprintf("unexpected operation: %s", topLevelOp))
	} else if !reflect.DeepEqual(expectedOps, actualOps) {
		errs = append(errs, fmt.Sprintf("unexpected operations in resolver metadata: want %v; got %s", expectedOps, actualOps))
	}
	return errs
}

func checkHeaders(presentHeaders, missingHeaders map[string]string, actualHeaders http.Header) error {
	for k, v := range presentHeaders {
		if actualHeaders.Get(k) != v {
			return fmt.Errorf("headers have wrong value for %s: expecting %q, got %q", k, v, actualHeaders.Get(k))
		}
	}
	for k := range missingHeaders {
		if len(actualHeaders.Values(k)) > 0 {
			return fmt.Errorf("headers include %s but should not", k)
		}
	}
	return nil
}
