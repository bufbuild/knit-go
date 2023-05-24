# 🧶 Knit

[![License](https://img.shields.io/github/license/bufbuild/knit-go?color=blue)][badges_license]
[![Slack](https://img.shields.io/badge/slack-buf-%23e01563)][badges_slack]
[![Build](https://github.com/bufbuild/knit-go/actions/workflows/ci.yaml/badge.svg?branch=main)](https://github.com/bufbuild/knit-go/actions/workflows/ci.yaml)
[![Report Card](https://goreportcard.com/badge/github.com/bufbuild/knit-go)](https://goreportcard.com/report/github.com/bufbuild/knit-go)
[![GoDoc](https://pkg.go.dev/badge/github.com/bufbuild/knit-go.svg)](https://pkg.go.dev/github.com/bufbuild/knit-go)

**Knit brings GraphQL-like capabilities to RPCs. Knit has type-safe and
declarative queries that shape the response, batching support to eliminate
the N+1 problem, and first-class support for error handling with partial
responses. It is built on top of Protobuf and Connect.**

**[Knit] is currently in alpha (α), and looking for feedback. Learn more
about it at the [Knit] repo, and learn how to use it with the [Tutorial].**

---

This repo is an implementation in Go of the server-side of the Knit protocol. The result is a
gateway that processes these declarative queries, dispatches the relevant RPCs, and then merges
the results together. The actual service interface is defined in the BSR:
[buf.build/bufbuild/knit](https://buf.build/bufbuild/knit).

For more information on the core concepts of Knit, read the documentation in the [repo that
defines the protocol](https://github.com/bufbuild/knit).

This repo contains two key components:

1. The runtime library used by a Knit gateway: Go package `"github.com/bufbuild/knit-go"`.
2. A standalone server program that can be used as a Knit gateway and configured via YAML file:
   `"github.com/bufbuild/knit-go/cmd/knitgateway`.


## Knit Gateway

The Knit gateway is a Go server that implements the [Knit service](https://buf.build/bufbuild/knit/docs/main:buf.knit.gateway.v1alpha1#buf.knit.gateway.v1alpha1.KnitService).

The process of handling a Knit query consists of the following steps:

1. **Request Validation and Schema Computation**

   The first step is to validate each requested entry point method and its associated mask.
   Validating the mask is done at the same time as producing the response schema, both of
   which involve a traversal of the mask, comparing requested field names against the RPC's
   response schema and the gateway's set of known relations.

2. **Issuing Entry-Point RPCs**

   Once the request is validated, all indicated methods are invoked. These requests are
   sent concurrently (up to a configurable parallelism limit). When the gateway is configured,
   a route is associated with each RPC service, so this dispatch step could end up sending
   multiple requests to the same backend or scattering requests to many backends (depending
   on which methods were in the request and their configured routes).

3. **Response Masking**

   Once an entry-point RPC completes, the response data is filtered according to the mask
   in the request. If the mask indicated any relations that must be resolved, those are
   accumulated in a set of "patches". A patch indicates a piece of data that must first be
   computed by a resolver and then inserted into the response structure.

4. **Stitching**

   Stitching is the iterative process of resolving patches and adding them to the response
   structure. Stitching is complete when there are no patches to resolve.

   So if any patches were identified in the above step, they are aggregated into batches
   and sent to resolvers. Resolvers are functions that know how to compute the values of
   relation fields. To avoid the N+1 problem, resolvers always accept batches. All batches
   are resolved concurrently (up to the same configurable parallelism limit used for dispatching
   entry-point RPCs).

   After a resolver provides results, we go back to step 3: the result data is filtered
   according to the mask in the request and inserted into the response structure. If the
   mask for the resolved data includes more relations, a subsequent set of patches is
   computed, and then the gateway performs another round of stitching.

At the end of this step, the gateway has aggregated the results of all RPCs and can send a
response to the client.

This process occurs for all Knit operations: `Fetch`, `Do`, and `Listen`. That last one is
a server-stream, where the above steps are executed for each response message in the stream.

### Resolvers

When services are registered, if any of the service's methods are annotated as relation
resolvers, then the gateway will use that RPC method to resolve relations that appear in
incoming queries.


## Using the Standalone Server

This repo contains a stand-alone Knit gateway server that can get you up
and going by just writing a YAML config file.

The server is a single statically-linked binary that can be downloaded
from the [_Releases_](https://github.com/bufbuild/knit-go/releases) page
for this repo.

You can also use the Go tool to build and install the server from source:

```bash
go install github.com/bufbuild/knit-go/cmd/knitgateway@latest
```

This builds a binary named `knitgateway` from the latest release.

Running the binary will start the server, which will by default expect a
config file named `knitgateway.yaml` to exist in the current working directory.

The binary accepts the following command-line options:

* **`--conf <filename>`**: Overrides the name and path of the config file to use.
* **`--log-format <format>`**: Configures the log output format. The default is
  "console" format, which emits logs in a simple human-readable line-oriented text
  form. The other option is "json" format, which emits structured data formatted
  as JSON.
* **`--version`**: Prints the version of the gateway program and then immediately
  exits.

### Configuration

In order to configure the server, you need to provide a YAML config file. There
is a an example in the root of this repo named
[`knitgateway.example.yaml`](/knitgateway.example.yaml).
The example file shows all the properties that can be configured. The example
also is a working example if you also run the
[`swapi-server`](https://github.com/bufbuild/knit-demo/blob/main/go/cmd/swapi-server/)
demo server as the backend.

The YAML config format is documented in its entirety in a [separate page](/CONFIG.md).


## Creating a Custom Gateway

You may want a custom gateway if you need the gateway to do something that the standalone
gateway program does not do. This could range from custom observability or alternate logging,
additional endpoints that the environment expects, add features not present in the standalone
gateway (supporting other encodings, compression algorithms, protocols [e.g. HTTP/3],
etc), or even embedding the gateway into the same process as a Connect or gRPC backend.

Creating a custom gateway involves writing a Go HTTP server. This server will install a handler
for the Knit service, which is provided by the `"github.com/bufbuild/knit-go"` package in this
repo.

The main steps to use this package all involve configuring the handler.

### Initial Configuration

First we have to create a gateway. Note that _none_ of the attributes are
required, so it can be as simple as this:

```go
gateway := &knit.Gateway{}
```

This returns a gateway that will:

1. Use `http.DefaultClient` as the transport for outbound RPCs.
2. Use Connect as the protocol (vs. gRPC or gRPC-Web) and use the Protobuf
   binary format as the message encoding.
3. Have no limit on parallelism for outbound RPCs
4. Use `protoregistry.GlobalTypes` for resolving extension names in requests and
   for resolving message names in `google.protobuf.Any` messages.
5. Require that registered services include routing information (so the gateway
   knows where to send outbound RPCs).

You can customize the above behavior by setting various fields on the gateway:

- `Client`: The transport to use for outbound RPCs. (This can also be overridden
  on a per-service basis, if some services require different middleware, such as
  auth, than others).
- `ClientOptions`: The Connect client options to use for outbound RPCs. This
  allows customizing things like interceptors and protocols. If some backends
  only support gRPC, you can configure that with a client option.
- `MaxParallelismPerRequest`: The concurrency limit for handling a single Knit
  request. Note that this controls the parallelism of issuing entry-point RPCs
  and the parallelism of invoking resolvers. This setting cannot be enforced
  inside of resolver implementations: if a resolver implementation starts other
  goroutines to operate with additional parallelism, this limit may be exceeded.
- `Route`: This is a default route. If you have one application that will
  receive most (or all) of the Connect/gRPC traffic, configure it here. Then you
  only need to include routing information when registering services that should
  be routed elsewhere.
- `TypeResolver`: This is an advanced option that is usually only useful or
  necessary when using dynamic RPC schemas. This resolver provides descriptors
  for extensions and messages, in case any requests or responses include
  extensions or `google.protobuf.Any` messages.

> NOTE: If you want to configure a custom codec for outbound RPCs, to customize
> content encoding, you must use `knit.WithCodec` **instead of**
> `connect.WithCodec` when creating the Connect client option.

### Configuring Entry-Point Services

Once the gateway is created with basic configuration, we register RPC services
whose methods can be used as entry points for Knit operations.

The simplest way is to register services is to import the Connect generated code
for these services. This generated code includes a constant for the service name
and will also ensure that the relevant service descriptors are linked into your
program.

```go
package main

// This is the generated package for the Connect demo service: Eliza
import (
	"net/url"

	"buf.build/gen/go/bufbuild/eliza/bufbuild/connect-go/buf/connect/demo/eliza/v1/elizav1connect"
	"github.com/bufbuild/knit-go"
)

func main() {
	gateway := &knit.Gateway{
		Route: &url.URL{
			Scheme: "https",
			Host:   "my.backend.service:8443",
		},
		MaxParallelismPerRequest: 10,
	}
	// Refer to generated constant for service name
	err := gateway.AddServiceByName(elizav1connect.ElizaServiceName)

	// ... more configuration ...
	// ... start server ...
}
```

When you register a service, requests for that service will be routed to the
`gateway.Route` URL. If that field is not set (i.e. there is no route for the
service), the call to `AddServiceByName` will return an error.

You can supply the route (or override the default one in `gateway.Route`) with
an option. There are other options that allow you to provide a different
HTTP client and different Connect client options. These can be used if your
backends are not homogenous: for example, some are Connect and some are gRPC,
some support "[h2c](https://connect.build/docs/go/deployment)" and some do not,
etc.

```go
err := gateway.AddServiceByName(
	elizav1connect.ElizaServiceName,
	knit.WithRoute(elizaBackendURL),
	knit.WithClient(h2cClient),
	knit.WithClientOptions(connect.WithGRPC()),
	)
```

### Starting a Server

The Knit protocol is a Protobuf service, so it can be exposed over HTTP using
the Connect framework, like any other such service.

So now that our gateway is fully configured, we just wire it up as an HTTP
handler:

```go
package main

import (
	"net"
	"net/http"

	"github.com/bufbuild/knit-go"
)

// Example function for starting an HTTP server that exposes a
// configured Knit gateway.
func serveHTTP(bindAddress string, gateway *knit.Gateway) error {
	listener, err := net.Listen("tcp", bindAddress)
	if err != nil {
		return err
	}
	mux := http.NewServeMux()
	mux.Handle(path, gateway.AsHandler())
	// This returns when the server is stopped
	return http.Serve(listener, mux)
}
```

Now a Knit client can send requests to the HTTP server we just started.


## Status: Alpha

Knit is undergoing initial development and is not yet stable.


## Legal

Offered under the [Apache 2 license][badges_license].


[badges_license]: https://github.com/bufbuild/knit-go/blob/main/LICENSE
[badges_slack]: https://buf.build/links/slack
[knit]: https://github.com/bufbuild/knit
[tutorial]: https://github.com/bufbuild/knit/tree/main/tutorial
