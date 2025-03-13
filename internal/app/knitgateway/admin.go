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
	"fmt"
	"net/http"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

// RegisterAdminHandlers registers "/admin/" HTTP handlers on the given mux, to
// expose admin functions for the given gateway.
func RegisterAdminHandlers(gateway *Gateway, mux *http.ServeMux) {
	handleFunc(mux, "/admin/", http.MethodGet, func(respWriter http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintf(respWriter, `
		<html>
		<head><title>Admin :: knitgateway</title></head>
		<body>
		<h1>Admin for knitgateway</h1>
		<ul>
		<li><a href="/admin/config">View config</a>
		<li><a href="/admin/services/">Service listing</a>
		<li><form style='display:none' id="reload" action="/admin/reload-now" method="post"></form>
			<a href="#" onclick="javascript:document.querySelector('form#reload').submit()">Reload schemas</a>
		</ul>
		</body>
		`)
	})
	handleFunc(mux, "/admin/config", http.MethodGet, func(respWriter http.ResponseWriter, _ *http.Request) {
		respWriter.Header().Set("Content-Type", "text/yaml")
		_, _ = respWriter.Write(gateway.config.RawConfig)
	})
	handleFunc(mux, "/admin/reload-now", http.MethodPost, func(respWriter http.ResponseWriter, _ *http.Request) {
		for _, watcher := range gateway.watchers {
			watcher.ResolveNow()
		}
		_, _ = respWriter.Write(([]byte)("Reloaded"))
	})
	handleFunc(mux, "/admin/services/", http.MethodGet, func(respWriter http.ResponseWriter, req *http.Request) {
		serviceName := strings.TrimPrefix(req.URL.Path, "/admin/services/")
		if strings.ContainsRune(serviceName, '/') {
			http.Error(respWriter, fmt.Sprintf("Not a valid service name: %q", serviceName), http.StatusNotFound)
			return
		}
		currentHandler := gateway.delegate.Load()
		if currentHandler == nil {
			http.Error(respWriter, "gateway not initialized", http.StatusServiceUnavailable)
			return
		}

		if serviceName == "" {
			// Show index of all services
			handleServiceListing(gateway, currentHandler.services, respWriter)
			return
		}

		// Download FileDescriptorSet for named service
		handleServiceSchema(currentHandler.services, serviceName, respWriter)
	})
}

func handleFunc(mux *http.ServeMux, pattern, method string, handlerFunc http.HandlerFunc) {
	handler := func(respWriter http.ResponseWriter, req *http.Request) {
		if req.Method != method {
			respWriter.Header().Set("Allow", method)
			http.Error(respWriter, fmt.Sprintf("unsupported method %q", method), http.StatusMethodNotAllowed)
			return
		}
		handlerFunc(respWriter, req)
	}
	mux.HandleFunc(pattern, handler)
}

func handleServiceListing(gateway *Gateway, services []protoreflect.ServiceDescriptor, respWriter http.ResponseWriter) {
	_, _ = fmt.Fprintf(respWriter, `
	<html>
	<head><title>Admin -- Services :: knitgateway</title></head>
	<body>
	<h1>Configured services</h1>
	<ul>
	`)
	for _, svc := range services {
		svcConf := gateway.config.Services[string(svc.FullName())]
		var fetchMethods, doMethods, listenMethods []string
		methods := svc.Methods()
		for i, length := 0, methods.Len(); i < length; i++ {
			method := methods.Get(i)
			opts, _ := method.Options().(*descriptorpb.MethodOptions)
			switch {
			case method.IsStreamingClient():
				continue
			case method.IsStreamingServer():
				listenMethods = append(listenMethods, string(method.Name()))
			case opts.GetIdempotencyLevel() == descriptorpb.MethodOptions_NO_SIDE_EFFECTS:
				fetchMethods = append(fetchMethods, string(method.Name()))
			default:
				doMethods = append(doMethods, string(method.Name()))
			}
		}
		stats := gateway.pollingStats[svcConf.Descriptors]
		updated := stats.LastUpdated()
		checked, checkErr := stats.LastChecked()
		var errStr string
		if checkErr != nil {
			errStr = " (failed)"
		}
		_, _ = fmt.Fprintf(respWriter, `
		<li><b>%s</b>:<br/>
			Last updated: %s<br/>
			Last checked: %s%s<br/>
			<a href="/admin/services/%s">Download schema</a><br/>
			Methods:
		`, svc.FullName(), formatTime(updated), formatTime(checked), errStr, svc.FullName())
		if len(fetchMethods) == 0 && len(doMethods) == 0 && len(listenMethods) == 0 {
			_, _ = fmt.Fprintf(respWriter, "<ul><li><i>None</i></li></ul>\n")
		} else {
			_, _ = fmt.Fprintf(respWriter, "<ul>\n")
			methodSets := []struct {
				name    string
				methods []string
			}{
				{name: "Fetch", methods: fetchMethods},
				{name: "Do", methods: doMethods},
				{name: "Listen", methods: listenMethods},
			}
			for _, methodSet := range methodSets {
				if len(methodSet.methods) == 0 {
					continue
				}
				_, _ = fmt.Fprintf(respWriter, "<li>%s <ul>\n", methodSet.name)
				for _, methodName := range methodSet.methods {
					_, _ = fmt.Fprintf(respWriter, "<li>%s</li>\n", methodName)
				}
				_, _ = fmt.Fprintf(respWriter, "</ul> </li>\n")
			}
			_, _ = fmt.Fprintf(respWriter, `</ul>`)
		}
		_, _ = fmt.Fprintf(respWriter, `
				</li>`)
	}
	_, _ = respWriter.Write(([]byte)(`
	</ul>
	</body>
	`))
}

func handleServiceSchema(services []protoreflect.ServiceDescriptor, serviceName string, respWriter http.ResponseWriter) {
	var svcDesc protoreflect.ServiceDescriptor
	for _, svc := range services {
		if string(svc.FullName()) == serviceName {
			svcDesc = svc
			break
		}
	}
	if svcDesc == nil {
		http.Error(respWriter, fmt.Sprintf("Unknown service name: %q", serviceName), http.StatusNotFound)
		return
	}
	var files descriptorpb.FileDescriptorSet
	accumulateFiles(svcDesc.ParentFile(), map[string]struct{}{}, &files.File)
	data, err := proto.Marshal(&files)
	if err != nil {
		http.Error(respWriter, fmt.Sprintf("Failed to marshal file descriptor set: %v", err), http.StatusInternalServerError)
		return
	}
	respWriter.Header().Set("Content-Type", `application/x-protobuf; messageType="google.protobuf.FileDescriptorSet""`)
	respWriter.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-schema.bin"`, svcDesc.Name()))
	_, _ = respWriter.Write(data)
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "<i>never</i>"
	}
	timeSince := time.Since(t)
	if timeSince < time.Second {
		return "just now"
	}
	return timeSince.Truncate(time.Second).String() + " ago"
}

func accumulateFiles(fileDesc protoreflect.FileDescriptor, seen map[string]struct{}, target *[]*descriptorpb.FileDescriptorProto) {
	if _, alreadySeen := seen[fileDesc.Path()]; alreadySeen {
		return
	}
	seen[fileDesc.Path()] = struct{}{}
	// We accumulate in topological order, which means dependencies first
	imps := fileDesc.Imports()
	for i, length := 0, imps.Len(); i < length; i++ {
		accumulateFiles(imps.Get(i), seen, target)
	}
	*target = append(*target, protodesc.ToFileDescriptorProto(fileDesc))
}
