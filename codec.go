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

	"github.com/bufbuild/connect-go"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type dynamicCodec struct {
	codec connect.Codec
}

func (d *dynamicCodec) Name() string {
	return d.codec.Name()
}

func (d *dynamicCodec) Marshal(a any) ([]byte, error) {
	if im, ok := a.(*indirectMessage); ok {
		a = im.a
	}
	return d.codec.Marshal(a)
}

func (d *dynamicCodec) Unmarshal(bytes []byte, msg any) error {
	if dm, ok := msg.(*deferredMessage); ok {
		// defensive copy since caller may re-use bytes after we return
		dm.bytes = make([]byte, len(bytes))
		copy(dm.bytes, bytes)
		dm.codec = d.codec
		return nil
	}
	return d.codec.Unmarshal(bytes, msg)
}

var _ connect.Codec = (*dynamicCodec)(nil)

type indirectMessage struct {
	a any
}

type deferredMessage struct {
	bytes []byte
	codec connect.Codec
}

func (dm *deferredMessage) unmarshal(a any) error {
	return dm.codec.Unmarshal(dm.bytes, a)
}

type defaultProtoCodec struct{}

func (d defaultProtoCodec) Name() string {
	return "proto"
}

func (d defaultProtoCodec) Marshal(a any) ([]byte, error) {
	msg, ok := a.(proto.Message)
	if !ok {
		return nil, fmt.Errorf("invalid message: %T does not implement proto.Message", a)
	}
	return proto.Marshal(msg)
}

func (d defaultProtoCodec) Unmarshal(bytes []byte, a any) error {
	msg, ok := a.(proto.Message)
	if !ok {
		return fmt.Errorf("invalid message: %T does not implement proto.Message", a)
	}
	return proto.Unmarshal(bytes, msg)
}

type defaultJSONCodec struct{}

func (d defaultJSONCodec) Name() string {
	return "json"
}

func (d defaultJSONCodec) Marshal(a any) ([]byte, error) {
	msg, ok := a.(proto.Message)
	if !ok {
		return nil, fmt.Errorf("invalid message: %T does not implement proto.Message", a)
	}
	return protojson.Marshal(msg)
}

func (d defaultJSONCodec) Unmarshal(bytes []byte, a any) error {
	msg, ok := a.(proto.Message)
	if !ok {
		return fmt.Errorf("invalid message: %T does not implement proto.Message", a)
	}
	return protojson.Unmarshal(bytes, msg)
}

func defaultCodec() connect.ClientOption {
	return WithCodec(defaultProtoCodec{})
}
