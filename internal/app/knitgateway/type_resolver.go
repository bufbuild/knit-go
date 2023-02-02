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

package knitgateway

import (
	"fmt"
	"strings"

	"github.com/bufbuild/knit-go"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

type typeResolver struct {
	DescriptorSource
}

var _ knit.TypeResolver = typeResolver{}

func (t typeResolver) FindMessageByName(message protoreflect.FullName) (protoreflect.MessageType, error) {
	desc, err := t.FindDescriptorByName(message)
	if err != nil {
		return nil, err
	}
	md, ok := desc.(protoreflect.MessageDescriptor)
	if !ok {
		return nil, fmt.Errorf("expecting message, got %s", descriptorKind(desc))
	}
	return dynamicpb.NewMessageType(md), nil
}

func (t typeResolver) FindMessageByURL(url string) (protoreflect.MessageType, error) {
	pos := strings.LastIndexByte(url, '/')
	typeName := url[pos+1:]
	return t.FindMessageByName(protoreflect.FullName(typeName))
}

func (t typeResolver) FindExtensionByName(field protoreflect.FullName) (protoreflect.ExtensionType, error) {
	desc, err := t.FindDescriptorByName(field)
	if err != nil {
		return nil, err
	}
	extd, ok := desc.(protoreflect.FieldDescriptor)
	if !ok || !extd.IsExtension() {
		return nil, fmt.Errorf("expecting extension, got %s", descriptorKind(desc))
	}
	return dynamicpb.NewExtensionType(extd), nil
}
