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
	"testing"

	gatewayv1alpha1 "buf.build/gen/go/bufbuild/knit/protocolbuffers/go/buf/knit/gateway/v1alpha1"
	connect "github.com/bufbuild/connect-go"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestDynamicCodec(t *testing.T) {
	t.Parallel()

	val, err := structpb.NewStruct(map[string]interface{}{
		"name": "Bob Loblaw",
		"age":  42,
	})
	require.NoError(t, err)
	msg := &gatewayv1alpha1.Request{
		Method: "foo.bar.Baz.Query",
		Body:   structpb.NewStructValue(val),
	}

	testCases := []connect.Codec{defaultProtoCodec{}, defaultJSONCodec{}}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.Name(), func(t *testing.T) {
			t.Parallel()

			codec := dynamicCodec{testCase}
			data, err := codec.Marshal(msg)
			require.NoError(t, err)
			var target deferredMessage
			err = codec.Unmarshal(data, &target)
			require.NoError(t, err)

			var roundTrip gatewayv1alpha1.Request
			err = target.unmarshal(&roundTrip)
			require.NoError(t, err)
			require.True(t, proto.Equal(msg, &roundTrip))
		})
	}
}
