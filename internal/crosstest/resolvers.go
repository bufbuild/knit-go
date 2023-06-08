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

package crosstest

import (
	"context"
	"strings"

	connect "github.com/bufbuild/connect-go"
	"github.com/bufbuild/knit-go/internal/gen/buf/knit/crosstest"
	"github.com/bufbuild/knit-go/internal/gen/buf/knit/crosstest/crosstestconnect"
)

type Resolver struct {
	parentClient crosstestconnect.ParentServiceClient
	childClient  crosstestconnect.ChildServiceClient
}

func NewResolver(
	parentClient crosstestconnect.ParentServiceClient,
	childClient crosstestconnect.ChildServiceClient,
) *Resolver {
	return &Resolver{
		parentClient: parentClient,
		childClient:  childClient,
	}
}

func (r *Resolver) GetParentChildren(ctx context.Context, req *connect.Request[crosstest.GetParentChildrenRequest]) (*connect.Response[crosstest.GetParentChildrenResponse], error) {
	bases := req.Msg.Bases
	values := make([]*crosstest.GetParentChildrenResponse_ParentResult, len(bases))
	for i, base := range bases {
		listChildrenResponse, err := r.childClient.ListChildren(ctx, connect.NewRequest(&crosstest.ListChildrenRequest{
			Parent:    base.Id,
			PageSize:  req.Msg.PageSize,
			PageToken: req.Msg.PageToken,
		}))
		if err != nil {
			return nil, err
		}
		values[i] = &crosstest.GetParentChildrenResponse_ParentResult{
			Children: &crosstest.ChildrenPage{
				Nodes:         listChildrenResponse.Msg.Children,
				NextPageToken: listChildrenResponse.Msg.NextPageToken,
			},
		}
	}
	return connect.NewResponse(&crosstest.GetParentChildrenResponse{Values: values}), nil
}

func (r *Resolver) GetChildParent(ctx context.Context, req *connect.Request[crosstest.GetChildParentRequest]) (*connect.Response[crosstest.GetChildParentResponse], error) {
	bases := req.Msg.Bases
	values := make([]*crosstest.GetChildParentResponse_ChildResult, len(bases))
	for i, base := range bases {
		getParentResponse, err := r.parentClient.GetParent(ctx, connect.NewRequest(&crosstest.GetParentRequest{
			Id: strings.Split(base.Id, "/")[0],
		}))
		if err != nil {
			return nil, err
		}
		values[i] = &crosstest.GetChildParentResponse_ChildResult{
			Parent: getParentResponse.Msg,
		}
	}
	return connect.NewResponse(&crosstest.GetChildParentResponse{Values: values}), nil
}
