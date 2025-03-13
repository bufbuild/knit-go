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
	"time"

	"go.uber.org/zap"
)

// NewLoggingHandler wraps the given handler so that all incoming requests are logged.
func NewLoggingHandler(handler http.Handler, logger *zap.Logger, suffix string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		intercepted, w := intercept(w)
		handler.ServeHTTP(w, r)
		logRequest(logger, suffix, r, intercepted.status, intercepted.transportFailed, intercepted.size, time.Since(start))
	})
}

func intercept(w http.ResponseWriter) (*interceptWriter, http.ResponseWriter) {
	intercepted := &interceptWriter{w: w, status: 200}
	if f, ok := w.(http.Flusher); ok {
		// make sure we return flusher if input writer can flush
		return intercepted, writerAndFlusher{ResponseWriter: intercepted, Flusher: f}
	}
	return intercepted, intercepted
}

type interceptWriter struct {
	w               http.ResponseWriter
	alreadyWrote    bool
	status          int
	transportFailed bool
	size            int
}

func (i *interceptWriter) Header() http.Header {
	return i.w.Header()
}

func (i *interceptWriter) Write(bytes []byte) (int, error) {
	i.alreadyWrote = true
	n, err := i.w.Write(bytes)
	i.size += n
	if err != nil {
		i.transportFailed = true
	}
	return n, err
}

func (i *interceptWriter) WriteHeader(statusCode int) {
	if !i.alreadyWrote {
		i.status = statusCode
	}
	i.w.WriteHeader(statusCode)
}

type writerAndFlusher struct {
	http.ResponseWriter
	http.Flusher
}

func logRequest(
	logger *zap.Logger,
	suffix string,
	req *http.Request,
	status int,
	transportFailed bool,
	bodySize int,
	latency time.Duration,
) {
	var msg string
	if suffix == "" {
		msg = "handled HTTP request"
	} else {
		msg = fmt.Sprintf("handled HTTP %s request", suffix)
	}
	logger.Info(msg,
		zap.String("method", req.Method),
		zap.String("requestURI", req.RequestURI),
		zap.String("remoteAddr", req.RemoteAddr),
		zap.Int("status", status),
		zap.Bool("transportFailed", transportFailed),
		zap.Int("responseBytes", bodySize),
		zap.Float64("latency_seconds", latency.Seconds()),
	)
}
