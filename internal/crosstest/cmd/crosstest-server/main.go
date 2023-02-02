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

package main

import (
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"

	"github.com/bufbuild/knit-go"
	"github.com/bufbuild/knit-go/internal/crosstest"
	"github.com/bufbuild/knit-go/internal/gen/buf/knit/crosstest/crosstestconnect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func main() {
	mux := http.NewServeMux()
	parentService := crosstest.NewParentService()
	childService := crosstest.NewChildService(parentService)
	resolver := crosstest.NewResolver(parentService, childService)
	mux.Handle(crosstestconnect.NewParentServiceHandler(parentService))
	mux.Handle(crosstestconnect.NewChildServiceHandler(childService))
	mux.Handle(crosstestconnect.NewRelationsServiceHandler(resolver))
	knitGateway := &knit.Gateway{
		Client: &http.Client{
			Transport: handlerTripper{mux},
		},
		Route: &url.URL{
			Scheme: "http",
			Host:   "localhost",
			Path:   "/",
		},
	}
	if err := knitGateway.AddServiceByName(crosstestconnect.ParentServiceName); err != nil {
		log.Fatalln("failed to add parent service to gateway")
	}
	if err := knitGateway.AddServiceByName(crosstestconnect.ChildServiceName); err != nil {
		log.Fatalln("failed to add child service to gateway")
	}
	if err := knitGateway.AddServiceByName(crosstestconnect.RelationsServiceName); err != nil {
		log.Fatalln("failed to add relations service to gateway")
	}
	mux.Handle(knitGateway.AsHandler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	//nolint:gosec
	server := &http.Server{
		Addr:    ":50051",
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}
	if err := server.ListenAndServe(); err != nil {
		log.Fatalln("serve error: %w", err)
	}
}

type handlerTripper struct {
	http.Handler
}

func (t handlerTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	w := httptest.NewRecorder()
	t.Handler.ServeHTTP(w, r)
	return w.Result(), nil
}
