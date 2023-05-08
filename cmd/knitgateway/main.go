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
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/bufbuild/knit-go/internal/app/knitgateway"
	"github.com/rs/cors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

const (
	logFormatConsole = "console"
	logFormatJSON    = "json"
)

func main() {
	flagSet := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	conf := flagSet.String("conf", "knitgateway.yaml", "The path to a YAML config file.")
	help := flagSet.Bool("help", false, "Show usage information.")
	logFormat := flagSet.String("log-format", logFormatConsole, fmt.Sprintf("Can be %q or %q.", logFormatConsole, logFormatJSON))
	flagSet.Usage = func() { printUsage(flagSet) }
	_ = flagSet.Parse(os.Args[1:])

	if flag.NArg() > 0 {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %s\nThis command does not accept any positional arguments.\n", flag.Arg(0))
		printUsage(flagSet)
		os.Exit(2)
	}
	if *help {
		printUsage(flagSet)
		return
	}

	if *logFormat != logFormatConsole && *logFormat != logFormatJSON {
		_, _ = fmt.Fprintf(os.Stderr, "-log-format value %q is invalid; must be either %q or %q\n", *logFormat, logFormatConsole, logFormatJSON)
	}
	loggerConfig := zap.NewProductionConfig()
	loggerConfig.Encoding = *logFormat
	loggerConfig.EncoderConfig.TimeKey = "timestamp"
	loggerConfig.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(time.RFC3339)
	logger, err := loggerConfig.Build()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		_ = logger.Sync()
	}()

	fatalf := func(msg string, args ...any) {
		logger.Sugar().Fatalf(msg, args...)
		_ = logger.Sync()
		os.Exit(1)
	}

	ctx := context.Background()
	config, err := knitgateway.LoadConfig(ctx, *conf)
	if err != nil {
		fatalf("failed to load config from %s: %v", *conf, err)
	}
	gateway, err := knitgateway.CreateGateway(config)
	if err != nil {
		fatalf("failed to configure gateway: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle(gateway.AsHandler())
	corsHandler := cors.AllowAll().Handler(mux)
	loggingHandler := knitgateway.NewLoggingHandler(corsHandler, logger)
	svr := http.Server{
		Handler:           h2c.NewHandler(loggingHandler, &http2.Server{}),
		ReadHeaderTimeout: 30 * time.Second,
	}
	l, err := net.Listen("tcp", config.ListenAddress)
	if err != nil {
		fatalf("failed to listen on bind address %s: %v\n", config.ListenAddress, err)
	}
	logger.Sugar().Infof("Listening on %s for HTTP requests...", l.Addr().String())
	if err := svr.Serve(l); err != nil {
		fatalf("HTTP server failed: %v", err)
	}
}

func printUsage(flagSet *flag.FlagSet) {
	w := flagSet.Output()
	_, _ = fmt.Fprint(w, `Runs the Knit gateway.

By default, the gateway will load configuration from a file named "knitgateway.yaml".
You can direct it to load a different file via command-line flag.

The Knit gateway runs an HTTP server that accepts requests from a Knit client using
the Knit protocol. The gateway processes requests by sending RPCs to backends (which
must be defined in the configuration file). A single Knit request can indicate
multiple RPCs; the responses to them will be combined by the gateway and sent back to
the client. Before the response is sent, the response body is filtered according to
the fields requested by the client.

If the gateway is configured for services that contain "relation resolver" RPC methods,
then a Knit query may also include references to these relations. These result in
RPC invocations to "resolve" the relations and join the data retrieved into an entity
message, allowing for complex joining of data from multiple backend services.

The configuration file contains all details for how the gateway operates. For more
details, see the following:
   https://github.com/bufbuild/knit-go#using-the-standalone-server
`)
	flagSet.PrintDefaults()
}