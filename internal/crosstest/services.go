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

//nolint:forcetypeassert
package crosstest

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"connectrpc.com/connect"
	"github.com/bufbuild/knit-go/internal/gen/buf/knit/crosstest"
	"github.com/bufbuild/knit-go/internal/gen/buf/knit/crosstest/crosstestconnect"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ParentService struct {
	crosstestconnect.UnimplementedParentServiceHandler

	idCounter atomic.Int64
	store     sync.Map
}

func NewParentService() *ParentService {
	return &ParentService{}
}

func (s *ParentService) ListParents(context.Context, *connect.Request[crosstest.ListParentsRequest]) (*connect.Response[crosstest.ListParentsResponse], error) {
	var parents []*crosstest.Parent
	s.store.Range(func(key, value any) bool {
		parents = append(parents, value.(*crosstest.Parent))
		return true
	})
	return connect.NewResponse(&crosstest.ListParentsResponse{
		Parents:       parents,
		NextPageToken: "",
	}), nil
}

func (s *ParentService) GetParent(_ context.Context, req *connect.Request[crosstest.GetParentRequest]) (*connect.Response[crosstest.Parent], error) {
	parentAny, ok := s.store.Load(req.Msg.GetId())
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("parent with id: %q not found", req.Msg.GetId()))
	}
	return connect.NewResponse(parentAny.(*crosstest.Parent)), nil
}

func (s *ParentService) CreateParent(_ context.Context, req *connect.Request[crosstest.CreateParentRequest]) (*connect.Response[crosstest.Parent], error) {
	id := s.idCounter.Add(1)
	parent := proto.Clone(req.Msg.GetParent()).(*crosstest.Parent) //nolint:errcheck
	parent.Id = strconv.FormatInt(id, 10)
	s.store.Store(parent.Id, parent)
	return connect.NewResponse(parent), nil
}

func (s *ParentService) UpdateParent(_ context.Context, req *connect.Request[crosstest.UpdateParentRequest]) (*connect.Response[crosstest.Parent], error) {
	id := req.Msg.GetParent().GetId()
	if _, ok := s.store.Load(id); !ok {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("parent with id: %q not found", id))
	}
	s.store.Store(id, proto.Clone(req.Msg.Parent))
	return connect.NewResponse(req.Msg.Parent), nil
}

func (s *ParentService) DeleteParent(_ context.Context, req *connect.Request[crosstest.DeleteParentRequest]) (*connect.Response[emptypb.Empty], error) {
	s.store.Delete(req.Msg.GetId())
	return connect.NewResponse(&emptypb.Empty{}), nil
}

type ChildService struct {
	crosstestconnect.UnimplementedChildServiceHandler

	idCounter atomic.Int64
	store     sync.Map

	parentClient crosstestconnect.ParentServiceClient
}

func NewChildService(parentClient crosstestconnect.ParentServiceClient) *ChildService {
	return &ChildService{
		parentClient: parentClient,
	}
}

func (s *ChildService) ListChildren(_ context.Context, req *connect.Request[crosstest.ListChildrenRequest]) (*connect.Response[crosstest.ListChildrenResponse], error) {
	var children []*crosstest.Child
	s.store.Range(func(key, value any) bool {
		child := value.(*crosstest.Child) //nolint:errcheck
		if strings.HasPrefix(child.Id, req.Msg.Parent) {
			children = append(children, child)
		}
		return true
	})
	return connect.NewResponse(&crosstest.ListChildrenResponse{
		Children:      children,
		NextPageToken: "",
	}), nil
}

func (s *ChildService) GetChild(_ context.Context, req *connect.Request[crosstest.GetChildRequest]) (*connect.Response[crosstest.Child], error) {
	childAny, ok := s.store.Load(req.Msg.GetId())
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("child with id: %q not found", req.Msg.GetId()))
	}
	return connect.NewResponse(childAny.(*crosstest.Child)), nil
}

func (s *ChildService) CreateChild(ctx context.Context, req *connect.Request[crosstest.CreateChildRequest]) (*connect.Response[crosstest.Child], error) {
	if _, err := s.parentClient.GetParent(ctx, connect.NewRequest(&crosstest.GetParentRequest{
		Id: req.Msg.Parent,
	})); err != nil {
		return nil, err
	}
	id := s.idCounter.Add(1)
	child := proto.Clone(req.Msg.GetChild()).(*crosstest.Child) //nolint:errcheck
	child.Id = req.Msg.Parent + "/" + strconv.FormatInt(id, 10)
	s.store.Store(child.Id, child)
	return connect.NewResponse(child), nil
}

func (s *ChildService) UpdateChild(_ context.Context, req *connect.Request[crosstest.UpdateChildRequest]) (*connect.Response[crosstest.Child], error) {
	id := req.Msg.GetChild().GetId()
	if _, ok := s.store.Load(id); !ok {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("child with id: %q not found", id))
	}
	s.store.Store(id, proto.Clone(req.Msg.Child))
	return connect.NewResponse(req.Msg.Child), nil
}

func (s *ChildService) DeleteChild(_ context.Context, req *connect.Request[crosstest.DeleteChildRequest]) (*connect.Response[emptypb.Empty], error) {
	s.store.Delete(req.Msg.GetId())
	return connect.NewResponse(&emptypb.Empty{}), nil
}
