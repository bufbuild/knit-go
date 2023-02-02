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
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/bufbuild/connect-go"
	"github.com/bufbuild/connect-grpcreflect-go"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

// DescriptorSource provides descriptors, allowing a Knit gateway to process
// a schema dynamically.
type DescriptorSource interface {
	FindDescriptorByName(protoreflect.FullName) (protoreflect.Descriptor, error)
	FindExtensionByNumber(message protoreflect.FullName, field protoreflect.FieldNumber) (protoreflect.ExtensionType, error)
}

type fileDescriptorSetSource struct {
	files protoregistry.Files
	types protoregistry.Types
}

func newFileDescriptorSetSource(fileDescriptorSet *descriptorpb.FileDescriptorSet) (*fileDescriptorSetSource, error) {
	files, err := protodesc.NewFiles(fileDescriptorSet)
	if err != nil {
		return nil, err
	}
	var result fileDescriptorSetSource
	files.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		registerExtensions(&result.types, fd)
		return true
	})
	result.files = *files
	return &result, nil
}

func (f *fileDescriptorSetSource) FindDescriptorByName(name protoreflect.FullName) (protoreflect.Descriptor, error) {
	return f.files.FindDescriptorByName(name)
}

func (f *fileDescriptorSetSource) FindExtensionByNumber(message protoreflect.FullName, field protoreflect.FieldNumber) (protoreflect.ExtensionType, error) {
	return f.types.FindExtensionByNumber(message, field)
}

// The following code was copied from buf curl's implementation.
// See github.com/bufbuild/buf/private/buf/bufcurl.NewServerReflectionResolver

type grpcDescriptorSource struct {
	ctx    context.Context //nolint:containedctx
	client *grpcreflect.Client

	mu               sync.Mutex
	stream           *grpcreflect.ClientStream
	downloadedProtos map[string]*descriptorpb.FileDescriptorProto
	cachedFiles      protoregistry.Files
	cachedExts       protoregistry.Types
}

func newGRPCDescriptorSource(
	ctx context.Context,
	httpClient connect.HTTPClient,
	opts []connect.ClientOption,
	baseURL string,
) *grpcDescriptorSource {
	baseURL = strings.TrimSuffix(baseURL, "/")
	res := &grpcDescriptorSource{
		ctx:              ctx,
		client:           grpcreflect.NewClient(httpClient, baseURL, opts...),
		downloadedProtos: map[string]*descriptorpb.FileDescriptorProto{},
	}
	return res
}

func (r *grpcDescriptorSource) FindDescriptorByName(name protoreflect.FullName) (protoreflect.Descriptor, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	d, err := r.cachedFiles.FindDescriptorByName(name)
	if d != nil {
		return d, nil
	}
	if !errors.Is(err, protoregistry.NotFound) {
		return nil, err
	}
	// if not found in existing files, fetch more
	fileDescriptorProtos, err := r.fileContainingSymbolLocked(name)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve symbol %q: %w", name, err)
	}
	if err := r.cacheFilesLocked(fileDescriptorProtos); err != nil {
		return nil, err
	}
	// now it should definitely be in there!
	return r.cachedFiles.FindDescriptorByName(name)
}

func (r *grpcDescriptorSource) FindExtensionByNumber(message protoreflect.FullName, field protoreflect.FieldNumber) (protoreflect.ExtensionType, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	ext, err := r.cachedExts.FindExtensionByNumber(message, field)
	if ext != nil {
		return ext, nil
	}
	if !errors.Is(err, protoregistry.NotFound) {
		return nil, err
	}
	// if not found in existing files, fetch more
	fileDescriptorProtos, err := r.fileContainingExtensionLocked(message, field)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve extension %d for %q: %w", field, message, err)
	}
	if err := r.cacheFilesLocked(fileDescriptorProtos); err != nil {
		return nil, err
	}
	// now it should definitely be in there!
	return r.cachedExts.FindExtensionByNumber(message, field)
}

func (r *grpcDescriptorSource) fileContainingSymbolLocked(name protoreflect.FullName) ([]*descriptorpb.FileDescriptorProto, error) {
	return r.doLocked(func(stream *grpcreflect.ClientStream) ([]*descriptorpb.FileDescriptorProto, error) {
		return stream.FileContainingSymbol(name)
	})
}

func (r *grpcDescriptorSource) fileContainingExtensionLocked(message protoreflect.FullName, field protoreflect.FieldNumber) ([]*descriptorpb.FileDescriptorProto, error) {
	return r.doLocked(func(stream *grpcreflect.ClientStream) ([]*descriptorpb.FileDescriptorProto, error) {
		return stream.FileContainingExtension(message, field)
	})
}

func (r *grpcDescriptorSource) fileByNameLocked(name string) ([]*descriptorpb.FileDescriptorProto, error) {
	return r.doLocked(func(stream *grpcreflect.ClientStream) ([]*descriptorpb.FileDescriptorProto, error) {
		return stream.FileByFilename(name)
	})
}

func (r *grpcDescriptorSource) cacheFilesLocked(files []*descriptorpb.FileDescriptorProto) error {
	for _, file := range files {
		if _, ok := r.downloadedProtos[file.GetName()]; ok {
			continue // already downloaded, don't bother overwriting
		}
		r.downloadedProtos[file.GetName()] = file
	}
	for _, file := range files {
		if err := r.cacheFileLocked(file.GetName(), nil); err != nil {
			return err
		}
	}
	return nil
}

func (r *grpcDescriptorSource) cacheFileLocked(name string, seen []string) error {
	if _, err := r.cachedFiles.FindFileByPath(name); err == nil {
		return nil // already processed this file
	}
	for i, alreadySeen := range seen {
		if name == alreadySeen {
			// we've seen this file already which means malformed
			// file descriptor protos that have an import cycle
			cycle := append(seen[i:], name) //nolint:gocritic
			return fmt.Errorf("downloaded files contain an import cycle: %s", strings.Join(cycle, " -> "))
		}
	}

	file := r.downloadedProtos[name]
	if file == nil {
		// download missing file(s)
		moreFiles, err := r.fileByNameLocked(name)
		if err != nil {
			return err
		}
		for _, newFile := range moreFiles {
			r.downloadedProtos[newFile.GetName()] = newFile
			if newFile.GetName() == name {
				file = newFile
			}
		}
		if file == nil {
			return fmt.Errorf("requested file %q but response did not contain it", name)
		}
	}

	// make sure imports have been downloaded and cached
	for _, dep := range file.Dependency {
		if err := r.cacheFileLocked(dep, append(seen, name)); err != nil {
			return err
		}
	}

	// now we can create and cache this file
	fileDescriptor, err := protodesc.NewFile(file, &r.cachedFiles)
	if err != nil {
		return err
	}
	if err := r.cachedFiles.RegisterFile(fileDescriptor); err != nil {
		return err
	}
	registerExtensions(&r.cachedExts, fileDescriptor)
	return nil
}

func (r *grpcDescriptorSource) doLocked(send func(*grpcreflect.ClientStream) ([]*descriptorpb.FileDescriptorProto, error)) ([]*descriptorpb.FileDescriptorProto, error) {
	stream := r.getStreamLocked()
	descs, err := send(stream)
	if grpcreflect.IsReflectionStreamBroken(err) {
		// recreate stream and try again
		r.resetLocked()
		stream := r.getStreamLocked()
		descs, err = send(stream)
		if grpcreflect.IsReflectionStreamBroken(err) {
			// again? reset the stream so next call can try again... but we won't try again
			r.resetLocked()
		}
	}
	return descs, err
}

func (r *grpcDescriptorSource) getStreamLocked() *grpcreflect.ClientStream {
	if r.stream == nil {
		r.stream = r.client.NewStream(r.ctx)
	}
	return r.stream
}

func (r *grpcDescriptorSource) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.resetLocked()
}

func (r *grpcDescriptorSource) resetLocked() {
	if r.stream == nil {
		// nothing to do
		return
	}
	_, _ = r.stream.Close()
	r.stream = nil
}

type extensionContainer interface {
	Messages() protoreflect.MessageDescriptors
	Extensions() protoreflect.ExtensionDescriptors
}

func registerExtensions(reg *protoregistry.Types, descriptor extensionContainer) {
	exts := descriptor.Extensions()
	for i := 0; i < exts.Len(); i++ {
		extType := dynamicpb.NewExtensionType(exts.Get(i))
		_ = reg.RegisterExtension(extType)
	}
	msgs := descriptor.Messages()
	for i := 0; i < msgs.Len(); i++ {
		registerExtensions(reg, msgs.Get(i))
	}
}

func descriptorKind(desc protoreflect.Descriptor) string {
	switch desc := desc.(type) {
	case protoreflect.FileDescriptor:
		return "file"
	case protoreflect.MessageDescriptor:
		return "message"
	case protoreflect.FieldDescriptor:
		if desc.IsExtension() {
			return "extension"
		}
		return "field"
	case protoreflect.OneofDescriptor:
		return "oneof"
	case protoreflect.EnumDescriptor:
		return "enum"
	case protoreflect.EnumValueDescriptor:
		return "enum value"
	case protoreflect.ServiceDescriptor:
		return "service"
	case protoreflect.MethodDescriptor:
		return "method"
	default:
		return fmt.Sprintf("%T", desc)
	}
}
