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
	"fmt"
	"net/http"
	"os"
	"strings"

	"buf.build/gen/go/bufbuild/reflect/connectrpc/go/buf/reflect/v1beta1/reflectv1beta1connect"
	connect "connectrpc.com/connect"
	grpcreflect "connectrpc.com/grpcreflect"
	"github.com/bufbuild/prototransform"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

type descriptorSource interface {
	fmt.Stringer

	isCacheable() bool
	newPoller(client connect.HTTPClient, opts []connect.ClientOption) (prototransform.SchemaPoller, error)
}

func newDescriptorSource(config externalDescriptorConfig, targetURL string, h2c bool) (descriptorSource, error) {
	var zero descriptorSource
	var properties []string
	if config.DescriptorSetFile != "" {
		properties = append(properties, "descriptor_set_file")
	}
	if config.GRPCReflection {
		properties = append(properties, "grpc_reflection")
	}
	if config.BufModule != "" {
		properties = append(properties, "buf_module")
	}
	if len(properties) > 1 {
		return zero, fmt.Errorf("descriptor config should have exactly one field set, instead got %d [%v]", len(properties), properties)
	}
	switch {
	case config.DescriptorSetFile != "":
		return descriptorSetFileSource(config.DescriptorSetFile), nil

	case config.BufModule != "":
		return bufModuleSource(config.BufModule), nil

	case config.GRPCReflection:
		if !strings.HasPrefix(targetURL, httpsScheme) && !h2c {
			return zero, fmt.Errorf("cannot use grpc reflection with http scheme without H2C")
		}
		return grpcReflectionSource(targetURL), nil

	default:
		return zero, fmt.Errorf("descriptor config is empty")
	}
}

type descriptorSetFileSource string

func (f descriptorSetFileSource) String() string {
	return fmt.Sprintf("descriptor set file: %q", string(f))
}

func (f descriptorSetFileSource) isCacheable() bool {
	return false // no need to cache descriptors that were read from file
}

func (f descriptorSetFileSource) newPoller(_ connect.HTTPClient, _ []connect.ClientOption) (prototransform.SchemaPoller, error) {
	info, err := os.Stat(string(f))
	if err != nil {
		return nil, fmt.Errorf("failed to load descriptor set %q: %w", f, err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("failed to load descriptor set %q: path is a directory, not a file", f)
	}
	return filePoller(f), nil
}

type bufModuleSource string

func (b bufModuleSource) String() string {
	return fmt.Sprintf("module %s", string(b))
}

func (b bufModuleSource) isCacheable() bool {
	return true
}

func (b bufModuleSource) newPoller(_ connect.HTTPClient, _ []connect.ClientOption) (prototransform.SchemaPoller, error) {
	moduleVersionParts := strings.SplitN(string(b), ":", 2)
	module := moduleVersionParts[0]
	var version string
	if len(moduleVersionParts) > 1 {
		version = moduleVersionParts[1]
	}
	moduleParts := strings.SplitN(module, "/", 2)
	remote := moduleParts[0]
	token, err := prototransform.BufTokenFromEnvironment(string(b))
	if err != nil {
		return nil, err
	}
	reflectClient := reflectv1beta1connect.NewFileDescriptorSetServiceClient(
		http.DefaultClient,
		"https://api."+remote,
		connect.WithInterceptors(prototransform.NewAuthInterceptor(token)),
	)
	return prototransform.NewSchemaPoller(reflectClient, module, version), nil
}

type grpcReflectionSource string

func (g grpcReflectionSource) String() string {
	return fmt.Sprintf("gRPC server reflection %s", string(g))
}

func (g grpcReflectionSource) isCacheable() bool {
	return true
}

func (g grpcReflectionSource) newPoller(client connect.HTTPClient, opts []connect.ClientOption) (prototransform.SchemaPoller, error) {
	reflectClient := grpcreflect.NewClient(client, string(g), opts...)
	return &grpcReflectionPoller{baseURL: string(g), client: reflectClient}, nil
}

type filePoller string

func (f filePoller) GetSchema(_ context.Context, _ []string, _ string) (descriptors *descriptorpb.FileDescriptorSet, version string, err error) {
	data, err := os.ReadFile(string(f))
	if err != nil {
		return nil, "", err
	}
	descriptors = &descriptorpb.FileDescriptorSet{}
	if err := proto.Unmarshal(data, descriptors); err != nil {
		return nil, "", err
	}
	return descriptors, "", nil
}

func (f filePoller) GetSchemaID() string {
	return fmt.Sprintf("file:%s", f)
}

type grpcReflectionPoller struct {
	baseURL string
	client  *grpcreflect.Client
}

func (g *grpcReflectionPoller) GetSchema(ctx context.Context, symbols []string, _ string) (descriptors *descriptorpb.FileDescriptorSet, version string, err error) {
	stream := g.client.NewStream(ctx)
	defer func() {
		_, _ = stream.Close()
	}()

	if len(symbols) == 0 {
		names, err := stream.ListServices()
		if err != nil {
			return nil, "", err
		}
		symbols = make([]string, len(names))
		for i := range names {
			symbols[i] = string(names[i])
		}
	}

	seen := map[string]struct{}{}
	var results []*descriptorpb.FileDescriptorProto
	var queue []string
	handleFiles := func(files []*descriptorpb.FileDescriptorProto) {
		// Add new files to results
		for _, file := range files {
			if _, ok := seen[file.GetName()]; ok {
				continue
			}
			results = append(results, file)
			seen[file.GetName()] = struct{}{}
		}
		// Queue up any other files that we need from dependencies.
		// We do this is a separate pass (instead of combining into
		// one loop over files) so that we don't queue up files that
		// are actually present later in the files slice.
		for _, file := range files {
			if _, ok := seen[file.GetName()]; ok {
				continue
			}
			for _, dep := range file.GetDependency() {
				if _, ok := seen[dep]; ok {
					continue
				}
				seen[dep] = struct{}{}
				queue = append(queue, dep)
			}
		}
	}

	// start with the symbols of interest
	for _, sym := range symbols {
		files, err := stream.FileContainingSymbol(protoreflect.FullName(sym))
		if err != nil {
			return nil, "", err
		}
		handleFiles(files)
	}

	// and now process the queue until we have the whole dependency graph
	for len(queue) > 0 {
		files, err := stream.FileByFilename(queue[0])
		if err != nil {
			return nil, "", err
		}
		queue = queue[1:]
		handleFiles(files)
	}

	// We could topologically sort the files, since that's how descriptor sets
	// look when produced by a compiler. But it's not necessary since the proto
	// runtime (protodesc.NewFiles) doesn't require it.
	return &descriptorpb.FileDescriptorSet{File: results}, "", nil
}

func (g *grpcReflectionPoller) GetSchemaID() string {
	return fmt.Sprintf("grpc reflection %s", g.baseURL)
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
