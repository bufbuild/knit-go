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

	"connectrpc.com/connect"
)

const buildVersion = "v0.1.0-dev"

//nolint:gochecknoglobals
var (
	// NB: This is a var instead of a const so it can be changed via -X ldflags.
	buildVersionSuffix = ""
)

func Version() string {
	return buildVersion + buildVersionSuffix
}

func UserAgentInterceptor(call connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		// decorate user-agent with the program name and version
		userAgent := fmt.Sprintf("knitgateway/%s %s", Version(), req.Header().Get("User-Agent"))
		req.Header().Set("User-Agent", userAgent)
		return call(ctx, req)
	}
}
