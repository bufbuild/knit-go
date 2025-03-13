// Copyright 2023-2025 Buf Technologies, Inc.
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
	"io"
	"io/fs"
	"os"
	"strconv"
	"strings"
	"time"

	gomemcache "github.com/bradfitz/gomemcache/memcache"
	"github.com/bufbuild/prototransform"
	"github.com/bufbuild/prototransform/cache/filecache"
	"github.com/bufbuild/prototransform/cache/memcache"
	"github.com/bufbuild/prototransform/cache/rediscache"
	"github.com/gomodule/redigo/redis"
	"go.uber.org/zap"
)

// DescriptorCacheConfig is the configuration for the mechanism used to
// cache schemas, so that a last-known-good cached schema can be used if
// a schema can't otherwise be loaded from a remote source on startup.
type DescriptorCacheConfig interface {
	toCache() (prototransform.Cache, io.Closer, error)
}

func newCacheConfig(conf externalDescriptorCacheConfig, numCacheableSources int) (DescriptorCacheConfig, error) {
	caches := make([]string, 0, 3)
	if conf.FileSystem != nil {
		caches = append(caches, "file_system")
	}
	if conf.Redis != nil {
		caches = append(caches, "redis")
	}
	if conf.Memcached != nil {
		caches = append(caches, "memcached")
	}
	if len(caches) == 0 {
		return nil, nil //nolint:nilnil // no cache, no problem
	}
	if len(caches) > 1 {
		return nil, fmt.Errorf(
			"only one kind of cache should be configured, file_system, redis, or memcache; instead found %d: %s",
			len(caches), strings.Join(caches, ", "))
	}
	if numCacheableSources == 0 {
		// cache defined, but no cacheable sources
		return nil, fmt.Errorf(
			"caching configured via %s, but no descriptor sources to cache (descriptor_set_file is not cached)",
			caches[0])
	}

	if conf.FileSystem != nil {
		return conf.FileSystem, nil
	}
	if conf.Redis != nil {
		return conf.Redis, nil
	}
	// that leaves only memcache
	return conf.Memcached, nil
}

type externalDescriptorCacheConfig struct {
	// At most one of the following may be set
	FileSystem *externalFileSystemCacheConfig `yaml:"file_system"`
	Redis      *externalRedisCacheConfig      `yaml:"redis"`
	Memcached  *externalMemcacheConfig        `yaml:"memcached"`
}

type externalFileSystemCacheConfig struct {
	Directory      string `yaml:"directory"`
	FileNamePrefix string `yaml:"file_name_prefix"`
	FileExtension  string `yaml:"file_extension"`
	FileMode       string `yaml:"file_mode"`
}

func (c *externalFileSystemCacheConfig) toCache() (prototransform.Cache, io.Closer, error) {
	var fileMode fs.FileMode
	if c.FileMode != "" {
		val, err := strconv.ParseInt(c.FileMode, 8, 32)
		if err != nil {
			return nil, nil, fmt.Errorf("file_mode attribute %q is not valid: %w", c.FileMode, err)
		}
		fileMode = fs.FileMode(val)
	}
	cache, err := filecache.New(filecache.Config{
		Path:              c.Directory,
		FilenamePrefix:    c.FileNamePrefix,
		FilenameExtension: c.FileExtension,
		FileMode:          fileMode,
	})
	return cache, nil, err
}

type externalRedisCacheConfig struct {
	Host               string `yaml:"host"`
	RequireAuth        bool   `yaml:"require_auth"`
	IdleTimeoutSeconds int    `yaml:"idle_timeout_seconds"`
	Database           int    `yaml:"database"`

	KeyPrefix     string `yaml:"key_prefix"`
	ExpirySeconds int    `yaml:"expiry_seconds"`
}

func (c *externalRedisCacheConfig) toCache() (prototransform.Cache, io.Closer, error) {
	var user, password string
	if c.RequireAuth {
		password = os.Getenv("REDIS_PASSWORD")
		if password == "" {
			return nil, nil, errors.New("redis.require_auth is true but no REDIS_PASSWORD in environment")
		}
		// user is optional
		user = os.Getenv("REDIS_USER")
	}

	pool := &redis.Pool{
		DialContext: func(ctx context.Context) (redis.Conn, error) {
			conn, err := redis.DialContext(ctx, "tcp", c.Host)
			if err != nil {
				return nil, err
			}
			if c.RequireAuth {
				var args []any
				if user != "" {
					args = []any{user, password}
				} else {
					args = []any{password}
				}
				_, err := redis.DoContext(conn, ctx, "auth", args...)
				if err != nil {
					return nil, fmt.Errorf("failed to auth with redis server: %w", err)
				}
			}
			if c.Database != 0 {
				_, err := redis.DoContext(conn, ctx, "select", c.Database)
				if err != nil {
					return nil, fmt.Errorf("failed to select db %d with redis server: %w", c.Database, err)
				}
			}
			return conn, nil
		},
		MaxIdle:     1,
		IdleTimeout: time.Second * time.Duration(c.IdleTimeoutSeconds),
	}

	cache, err := rediscache.New(rediscache.Config{
		Client:     pool,
		KeyPrefix:  c.KeyPrefix,
		Expiration: time.Second * time.Duration(c.ExpirySeconds),
	})
	return cache, pool, err
}

type externalMemcacheConfig struct {
	Hosts         []string `yaml:"hosts"`
	KeyPrefix     string   `yaml:"key_prefix"`
	ExpirySeconds int32    `yaml:"expiry_seconds"`
}

func (c *externalMemcacheConfig) toCache() (prototransform.Cache, io.Closer, error) {
	client := gomemcache.New(c.Hosts...)
	cache, err := memcache.New(memcache.Config{
		Client:            client,
		KeyPrefix:         c.KeyPrefix,
		ExpirationSeconds: c.ExpirySeconds,
	})
	return cache, client, err
}

type loggingCache struct {
	logger *zap.Logger
	cache  prototransform.Cache
}

func (l *loggingCache) Load(ctx context.Context, key string) ([]byte, error) {
	result, err := l.cache.Load(ctx, key)
	if err == nil {
		l.logger.Info("loaded schema from cache", zap.String("key", key))
	}
	return result, err
}

func (l *loggingCache) Save(ctx context.Context, key string, data []byte) error {
	err := l.cache.Save(ctx, key, data)
	if err != nil {
		l.logger.Warn("failed to save schema to cache", zap.String("key", key), zap.Error(err))
	}
	return err
}
