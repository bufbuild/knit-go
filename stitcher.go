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
	"strings"
	"sync"

	"github.com/bufbuild/connect-go"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

type stitcher struct {
	gateway *Gateway
	sema    sema
}

func newStitcher(gateway *Gateway) *stitcher {
	return &stitcher{
		gateway: gateway,
		sema:    newSema(gateway.MaxParallelismPerRequest),
	}
}

func (s *stitcher) stitch(ctx context.Context, patches [][]*patch, fallbackCatch bool) error {
	// TODO: There's probably a more clever way to do grouping, with a channel as a source,
	//       to reduce latency and allow pipelining. But this implementation does a simpler
	//       breadth-first traversal, waiting for all patches at one depth to complete before
	//       moving to the next depth.
	var patchesMu sync.Mutex
	for len(patches) > 0 {
		type groupKey struct {
			// relying on pointer equality, so all patches that originate from the
			// same unmarshalled mask definition will get grouped together
			params *structpb.Value
			config *relationConfig
			meta   *resolveMeta
		}
		// We group patches into ones with like keys, that can be aggregated into a batch
		groupedPatches := map[groupKey][]*patch{}
		targetLocks := map[*structpb.Struct]*sync.Mutex{}
		for _, patchSet := range patches {
			for _, patch := range patchSet {
				config, ok := s.gateway.relations[patch.target.ProtoReflect().Descriptor().FullName()][patch.mask.Name]
				if !ok {
					// shouldn't be possible since we should have already validated all requested
					// masks before any RPCs were issued
					return fmt.Errorf("%v: unknown relation %q", strings.Join(patch.path, "."), patch.mask.Name)
				}
				key := groupKey{
					params: patch.mask.Params,
					config: config,
					meta:   patch.meta,
				}
				groupedPatches[key] = append(groupedPatches[key], patch)
				if _, ok := targetLocks[patch.formatTarget]; !ok {
					// create a single lock per formatTarget, so concurrent resolvers
					// can patch values into it in a thread-safe way
					targetLocks[patch.formatTarget] = &sync.Mutex{}
				}
				if patch.errPatch != nil {
					if _, ok := targetLocks[patch.errPatch.formatTarget]; !ok {
						targetLocks[patch.errPatch.formatTarget] = &sync.Mutex{}
					}
				}
			}
		}
		if len(groupedPatches) == 0 {
			// in case patches is a slice of empty slices...
			break
		}
		// Now parse the params message for each group
		params := map[groupKey]proto.Message{}
		for key, patchGroup := range groupedPatches {
			msg := key.config.requestType.New()
			if key.params == nil {
				// leave params empty if none provided in mask
				params[key] = msg.Interface()
				continue
			}
			// TODO: We should move the calls to parseMessage to the start of handling, when validating the query.
			// Currently, we could end up doing some work (top-level method names) even though the request is bad
			// and doomed to fail. Also, its currently possible that we might re-parse this message even though
			// we only need to do so once.
			if err := parseMessage(key.params, msg.Interface(), s.gateway.TypeResolver); err != nil {
				return fmt.Errorf("could not unmarshal request type %q for relation %q of type %q: %w",
					msg.Descriptor().FullName(), patchGroup[0].mask.Name, patchGroup[0].target.ProtoReflect().Descriptor().FullName(), err,
				)
			}
			if msg.Has(key.config.baseField) {
				return fmt.Errorf("could not unmarshal request type %q for relation %q of type %q: query params must not include bases field %s",
					msg.Descriptor().FullName(), patchGroup[0].mask.Name, patchGroup[0].target.ProtoReflect().Descriptor().FullName(), camelCase(string(key.config.baseField.Name())),
				)
			}
			params[key] = msg.Interface()
		}

		// reset patches, to serve as queue for possible subsequent round of stitching
		patches = nil

		// Finally. break up groups into batches and execute resolver for each batch
		group, ctx := errgroup.WithContext(ctx)
		for key, patchGroup := range groupedPatches {
			// TODO: break up group into multiple batches, based on request size and/or count
			batches := [][]*patch{patchGroup}
			// now we can do resolution
			for batchNum := range batches {
				key := key
				batch := batches[batchNum]
				group.Go(func() error {
					if err := s.sema.Acquire(ctx); err != nil {
						return err
					}
					defer s.sema.Release()

					entities := make([]proto.Message, len(batch))
					for i := range batch {
						entities[i] = batch[i].target
					}
					vals, err := key.config.resolver(ctx, key.meta, entities, params[key])
					if err == nil && len(vals) != len(entities) {
						err = connect.NewError(
							connect.CodeInternal,
							fmt.Errorf("resolver for relation %q of type %q returned %d results, expected %d",
								batch[0].mask.Name, batch[0].target.ProtoReflect().Descriptor().FullName(), len(vals), len(entities),
							),
						)
					}
					if err != nil && batch[0].errPatch == nil {
						return fmt.Errorf("resolver for relation %q of type %q failed: %w",
							batch[0].mask.Name, batch[0].target.ProtoReflect().Descriptor().FullName(), err,
						)
					}

					// add the name of the resolver RPC to the stack of operations
					meta := *key.meta
					ops := make([]string, len(meta.operations)+1) // defensive copy
					copy(ops, meta.operations)
					ops[len(ops)-1] = string(key.config.method.FullName())
					meta.operations = ops

					for i, patch := range batch {
						if err != nil {
							errVal := formatError(err, strings.Join(append(patch.path, patch.mask.Name), "."), s.gateway.TypeResolver)
							mu := targetLocks[patch.errPatch.formatTarget]
							mu.Lock()
							patch.errPatch.formatTarget.Fields[patch.errPatch.name] = errVal
							mu.Unlock()
							continue
						}
						if !vals[i].IsValid() {
							// relation field is absent; skip
							continue
						}
						path := make([]string, len(patch.path)+1)
						copy(path, patch.path)
						path[len(path)-1] = patch.mask.Name

						fieldVal, fieldPatches, err := formatValue(key.config.descriptor, vals[i], &meta, patch.mask.Mask, path, patch.errPatch, fallbackCatch, s.gateway.TypeResolver)
						if err != nil {
							return err
						}
						mu := targetLocks[patch.formatTarget]
						mu.Lock()
						patch.formatTarget.Fields[patch.mask.Name] = fieldVal
						mu.Unlock()
						if len(fieldPatches) > 0 {
							patchesMu.Lock()
							patches = append(patches, fieldPatches)
							patchesMu.Unlock()
						}
					}

					return nil
				})
			}
		}

		if err := group.Wait(); err != nil {
			return err
		}
	}
	return nil
}

type sema interface {
	Acquire(ctx context.Context) error
	Release()
}

func newSema(limit int) sema {
	if limit > 0 {
		return (*boundedSema)(semaphore.NewWeighted(int64(limit)))
	}
	return unboundedSema{}
}

type boundedSema semaphore.Weighted

func (b *boundedSema) Acquire(ctx context.Context) error {
	return (*semaphore.Weighted)(b).Acquire(ctx, 1)
}

func (b *boundedSema) Release() {
	(*semaphore.Weighted)(b).Release(1)
}

type unboundedSema struct{}

func (unboundedSema) Acquire(context.Context) error {
	return nil
}

func (unboundedSema) Release() {
}
