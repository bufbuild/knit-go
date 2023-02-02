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
	"encoding/json"
	"net/url"
	"testing"

	"buf.build/gen/go/bufbuild/knit/protocolbuffers/go/buf/knit/gateway/v1alpha1"
	"github.com/bufbuild/knit-go/internal/gen/buf/knittest/knittestconnect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func TestComputeResponseSchema(t *testing.T) {
	t.Parallel()

	route, err := url.Parse("https://foo.bar.baz/")
	require.NoError(t, err)
	gateway := &Gateway{Route: route}
	err = gateway.AddServiceByName(knittestconnect.FooServiceName)
	require.NoError(t, err)
	err = gateway.AddServiceByName(knittestconnect.FizzServiceName)
	require.NoError(t, err)
	err = gateway.addRelation(newRelationFooStyle(t, nil))
	require.NoError(t, err)
	err = gateway.addRelation(newRelationFooBalanceCents(t, nil))
	require.NoError(t, err)
	err = gateway.addRelation(newRelationFooBars(t, nil))
	require.NoError(t, err)
	err = gateway.addRelation(newRelationBarBaz(t, nil))
	require.NoError(t, err)
	err = gateway.addRelation(newRelationBarBedazzles(t, nil))
	require.NoError(t, err)
	err = gateway.addRelation(newRelationFizzBuzzes(t, nil))
	require.NoError(t, err)

	wktMask := getExhaustiveMaskForWellKnownTypes()
	scalarMask := getExhaustiveMaskForScalars()
	fooMask := getExhaustiveFooMask()
	fizzMask := getExhaustiveFizzMask()

	testCases := []struct {
		name string
		req  *gatewayv1alpha1.Request
	}{
		{
			name: "wkts",
			req: &gatewayv1alpha1.Request{
				Method: knittestconnect.FooServiceName + ".GetFoo",
				Mask: []*gatewayv1alpha1.MaskField{
					{
						Name: "results",
						Mask: []*gatewayv1alpha1.MaskField{
							{
								Name: "wkts",
								Mask: wktMask,
							},
						},
					},
				},
			},
		},
		{
			name: "scalars",
			req: &gatewayv1alpha1.Request{
				Method: knittestconnect.FizzServiceName + ".GetFizz",
				Mask: []*gatewayv1alpha1.MaskField{
					{
						Name: "result",
						Mask: []*gatewayv1alpha1.MaskField{
							{
								Name: "scalars",
								Mask: scalarMask,
							},
						},
					},
				},
			},
		},
		{
			name: "foo",
			req: &gatewayv1alpha1.Request{
				Method: knittestconnect.FooServiceName + ".GetFoo",
				Mask: []*gatewayv1alpha1.MaskField{
					{
						Name: "results",
						Mask: fooMask,
					},
				},
			},
		},
		{
			name: "fizz",
			req: &gatewayv1alpha1.Request{
				Method: knittestconnect.FizzServiceName + ".GetFizz",
				Mask: []*gatewayv1alpha1.MaskField{
					{
						Name: "result",
						Mask: fizzMask,
					},
				},
			},
		},
	}

	for i := range testCases {
		testCase := testCases[i]
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			_, schema, err := computeResponseSchema(gateway, testCase.req, true, rpcKindUnary)
			require.NoError(t, err)
			checkGoldenFile(t, testCase.name+"-schema.txt", format(t, schema))
		})
	}
}

func format(t *testing.T, msg proto.Message) string {
	t.Helper()
	data, err := protojson.Marshal(msg)
	require.NoError(t, err)
	var buf bytes.Buffer
	err = json.Indent(&buf, data, "", "  ")
	require.NoError(t, err)
	return buf.String()
}
