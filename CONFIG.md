# Configuration for `knitgateway`

This document describes the configuration options for the `knitgateway`
program.

There is a an example config file in the root of this repo named
[`knitgateway.example.yaml`](/knitgateway.example.yaml).
The example file shows all the properties that can be configured (though
most are commented out). It includes many comments to describe each
property.

The YAML file must contain only a single-document. If it contains any
unrecognized properties, the configuration will be rejected and the
gateway will not start.

There are several top-level keys in the file, each of which contains
properties for a different category of configuration.
1. **`listen`**: This section configures the listener for the gateway. This
   contains details about the server like what interface to bind to, what
   port to listen on, and whether TLS should be used.
2. **`limits`**: This section configures limits, to aid with operations.
3. **`backends`**: This section configures all of the backends to which the
   gateway can send requests when processing a Knit query. This section
   configures the available RPC services that can be used in a Knit query
   and how to route them to backends. It includes connectivity details but
   also details on how the gateway can access the schemata for the backend's
   exposed RPC services.
3. **`backend_tls`**: This section allows cross-cutting TLS configuration.
   Each backend can be separately configured in the `backends` section
   above. But if all or most backends use similar TLS configuration, it
   can be consolidated in this top-level section. When there is configuration
   in both this section and for a specific backend in `backends`, the values
   in `backends` _override_ the values here. So this section effectively
   defines the default TLS settings.
4. **`descriptors`**: This section controls the polling behavior of the
   gateway, for periodically reloading schemata. This allows the gateway
   to reconfigure itself at runtime as the schemas change. The schemas
   can also be cached, so a cached last-known-good schema can be used if
   the source of the schema is otherwise unavailable when the gateway
   starts up.
5. **`cors`**: This section controls how the gateway handles cross-origin
   requests and replies to CORS pre-flight requests.

Each of the above config stanzas is described thoroughly in the sections below.

## Listen Config

The top-level `listen` property is a _map_ with the following keys:

- **`bind_address`**: The address on which to listen. This defaults to 0.0.0.0,
  which means it will accept requests on _all_ network interfaces. You can
  specify a specific address to limit what interfaces are used. For example,
  setting this to 127.0.0.1 means that requests are only accepted on the
  loopback interface (i.e. from the local host).
- **`port`**: The port number on which to listen. There is no default: in order for
  the gateway to use a TCP listener, a port must be configured. If the port is
  configured as zero, an ephemeral port is used. In this case, the actual port
  in use will be logged when the gateway starts.
- **`unix_socket`**: The path to a Unix domain socket on which to listen. If the
  socket file already exists, the gateway will not use it and will not start. There
  is no default: in order for the gateway to listen on a Unix socket, this property
  must be configured.
- **`tls`**: If this section is present, the server will require clients to use
  TLS (transport-level security, sometimes called SSL) when connecting. This means
  that connections are secure: both parties can be authenticated via TLS certificates
  and all traffic is encrypted.

  This section is a map with the following sub-keys:
  - **`cert`**: This is the path to a PEM-encoded X509 certificate. This is required
    and configures the public key and certificate chain for the gateway's server
    certificate.
  - **`key`**: This is the path to a PEM-encoded X509 private key. This is required
    and configures the private key for the gateway's server certificate.
  - **`min_version`**: The minimum TLS version that the server will accept. This
    defaults to 1.0. Other allowed settings are 1.1, 1.2, or 1.3.
  - **`ciphers`**: The cipher suites to allow or disallow, for TLS 1.2 and below. (For
    TLS 1.3, the supported cipher suites are not configurable and are always the
    following: TLS_AES_128_GCM_SHA256, TLS_AES_256_GCM_SHA384, TLS_CHACHA20_POLY1305_SHA256.)

    This section is a map with possible keys of `allow` or `disallow`. Only one of these
    keys may be present, depending on whether the configuration is using an allow-list of
    cipher suites or a block-list. See [below](#tls-cipher-suites) for more details.
  - **`client_certs`**: If this section is present, the gateway will request TLS certificates
    from clients during the TLS handshake. This property is a map with the following allowed
    keys which further control the gateway's behavior.
    - **`require`**: If false or absent, client certs are verified if given, but are not
      required. If true, connections will be terminated if the client does not provide a
      valid cert.

      When a client cert is present and valid, the authenticated identity ("subject" field
      of the cert) will be added to HTTP headers for all requests to backends. It will be
      set in a header named `Knit-Client-Subject` and will be in RFC 2253 Distinguished Names
      syntax.
    - **`cacert`**: This is the path to a PEM-encoded X509 certificate pool file that contains
      certs for CAs (certificate authorities/issuers). These are used to verify client certs.

This property is required as the config _must_ define either a unix socket path or a
port (or both).

### TLS Cipher Suites

TLS cipher suites can be configured in one of two ways:
1. An allow-list is defined using the `allow` property. In this mode, only the suites
   listed in the config will be allowed.
2. A block-list is defined using the `disallow` property. In this mode, only the suites
   listed in the config will be blocked, and all others will be allowed.

The following table shows all supported cipher suites. The table also shows which suites
are allowed and disallowed by default, when no cipher suite configuration is provided,

| Cipher Suite                                  | TLS Versions | Disposition |
|-----------------------------------------------|--------------|-------------|
| TLS_RSA_WITH_AES_128_CBC_SHA                  |              | Allowed     |
| TLS_RSA_WITH_AES_256_CBC_SHA                  |              | Allowed     |
| TLS_RSA_WITH_AES_128_GCM_SHA256               | TLS 1.2 only | Allowed     |
| TLS_RSA_WITH_AES_256_GCM_SHA384               | TLS 1.2 only | Allowed     |
| TLS_RSA_WITH_AES_256_GCM_SHA384               | TLS 1.2 only | Allowed     |
| TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA          |              | Allowed     |
| TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA          |              | Allowed     |
| TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA            |              | Allowed     |
| TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA            |              | Allowed     |
| TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256       | TLS 1.2 only | Allowed     |
| TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384       | TLS 1.2 only | Allowed     |
| TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256         | TLS 1.2 only | Allowed     |
| TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384         | TLS 1.2 only | Allowed     |
| TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256   | TLS 1.2 only | Allowed     |
| TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256 | TLS 1.2 only | Allowed     |
| TLS_RSA_WITH_RC4_128_SHA                      |              | Disallowed  |
| TLS_RSA_WITH_3DES_EDE_CBC_SHA                 |              | Disallowed  |
| TLS_RSA_WITH_AES_128_CBC_SHA256               | TLS 1.2 only | Disallowed  |
| TLS_ECDHE_ECDSA_WITH_RC4_128_SHA              |              | Disallowed  |
| TLS_ECDHE_RSA_WITH_RC4_128_SHA                |              | Disallowed  |
| TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA           |              | Disallowed  |
| TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256       | TLS 1.2 only | Disallowed  |
| TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256         | TLS 1.2 only | Disallowed  |

## Limits Config

The top-level `limits` property is a map. At the moment, there is just one key
that is allowed.

- **`per_request_parallelism`**: Tte maximum parallelism per request. This is
  the maximum number of concurrent RPCs that can be made on behalf of a single
  Knit query.

  If not specified, there is no limit, and all RPCs needed to evaluate a query
  will all be made in parallel.

As other kinds of controls are implemented, configuration for them will be added
to this section.

## Backends Config

The top-level `backends` property is an _array_ of backend configuration maps. Each
map in the array has the following keys:

- **`route_to`**: The base URL for this backend. This key must be provided.
  The URL may use either "http" or "https" scheme. Note that no custom
  TLS properties (such as custom root CA certificate or client certificates)
  are currently supported for use with "https" URLs.
- **`unix_socket`**: If the backend is listening on a Unix domain socket and
  not a TCP socket, configure the path to the socket here. By default, TCP
  connections are used, but when this property is present and non-empty it
  is the path to a Unix socket that will be used instead.
- **`h2c`**: If the `routeTo` property uses a plaintext "http" scheme, but
  HTTP/2 should be used, set this property to true. It defaults to false.
- **`protocol`**: This configured the protocol that will be used to communicate
  with this backend. The default protocol is "connect". Other allowed options
  are "grpc" or "grpcweb".
- **`encoding`**: This configures the message encoding for sending requests to
  this backend. The default is "proto", which uses the Protobuf binary format.
  The other allowed option is "json".
- **`services`**: This key is required. The value is a list of fully-qualified
  service names that will be routed to this backend. If these services contain
  methods that can resolve relations, then those relations are automatically
  supported by the server.
- **`tls`**: This key configures TLS settings for the backend. This key may
  only appear when the `route_to` URL has a scheme of "https", indicating
  secure connections are used. This property is a map whose contents are used
  to configure a TLS client for successfully connecting to and verifying the
  backend. See [below](#tls-client-config) for more details about the format
  of this property.
- **`descriptors`**: This key is required. This configuration indicates how
  the gateway will find descriptors for the above named services. The value is
  a map with the following keys, of which _only one may be set_:
    - **`descriptor_set_file`**: The value for this key is the path to a file
      that is an encoded [file descriptor set](https://github.com/protocolbuffers/protobuf/blob/v22.0/src/google/protobuf/descriptor.proto#L54-L58).
      Both `buf` and `protoc` can produce such files. (See more [below](#descriptor-set-examples).)
    - **`buf_module`**: The value for this key is the name of a module that has
      been pushed to a Buf Schema Registry (BSR). The module name must be in the
      format "&lt;remote>/&lt;owner>/&lt;repo>". The first part defines the host name for
      the BSR, for example `buf.build`. When using this option, you must provide
      an environment variable named [`BUF_TOKEN`](https://docs.buf.build/bsr/authentication#buf_token)
      that the gateway will use to authenticate with the BSR in order to download
      the module's descriptors.
    - **`grpc_reflection`**: The value for this key is a boolean. If true, then
      the [gRPC Server Reflection](https://github.com/grpc/grpc/blob/master/doc/server-reflection.md)
      protocol will be used to download the descriptors from the
      backend server itself.

### TLS Client Config

Each backend configured in the `backends` stanza can configure TLS client settings
via a `tls` property. Default TLS settings for all secure backends can be defined
in the top-level `backend_tls` stanza. No TLS settings are required; reasonable
defaults will be used for all settings.

The property is a map with the following keys:

- **`min_version`**: This minimum version of TLS to accept. Defaults to 1.2. Other
  allowed values are 1.1, 1.2, and 1.3. Note that the default here is different than
  for TLS _server_ settings, in the `tls` property of the [`listen`](#listen-config)
  stanza. The listener default is more lenient, to support external clients that may
  be using older browser or mobile OS software.
- **`ciphers`**: The cipher suites to allow or disallow, for TLS 1.2 and below. (For
  TLS 1.3, the supported cipher suites are not configurable and are always the
  following: TLS_AES_128_GCM_SHA256, TLS_AES_256_GCM_SHA384, TLS_CHACHA20_POLY1305_SHA256.)

  This section is a map with possible keys of `allow` or `disallow`. Only one of these
  keys may be present, depending on whether the configuration is using an allow-list of
  cipher suites or a block-list. See [below](#tls-cipher-suites) for more details.
- **`cacert`**: This is the path to a PEM-encoded X509 certificate pool file that
  contains certs for CAs (certificate authorities/issuers). These are used to verify
  server certs of the backends. If not present or blank, system defaults for the set
  of trusted root CAs will be used.
- **`skip_verify`**: This flag **disables** verification of server certs. Its use is
  strongly discouraged. It is present primarily to aid with testing.
- **`cert`**: This is the path to a PEM-encoded X509 certificate. This configures
  the public key and certificate chain for the client certificate to use. This should
  only be present if the server expects clients to provide a TLS certificate. If this
  property is present, `key` must also be present.
- **`key`**: This is the path to a PEM-encoded X509 private key. This configures
  the private key for a client certificate to use. This should only be present if the
  server expects clients to provide a TLS certificate. If this property is present,
  `cert` must also be present.

### Descriptor Set Examples

If using the `descriptor_set_file` option for the `descriptors` key, you can
use `buf` or `protoc` to generate a file in the correct format.

The following example uses `buf` to build proto sources in the current
directory. The `-o` flag indicates the path to the output file that will
contain the descriptors:

```shell
buf build . -o ../my-services.protoset
```

Here's another example using `buf`, this time to build the
[buf.build/bufbuild/knit-demo](https://buf.build/bufbuild/knit-demo) module.
This module contains service definitions that describe the
_Star Wars API_ ([swapi.dev](https://swapi.dev/)).

```shell
buf build buf.build/bufbuild/knit-demo -o swapi.protoset
```

Finally, here's an example that uses `protoc`. This compiles a
hypothetical file at path `foo/bar/services.proto`, where some of
its imports are defined in a `../../others/proto` directory.

Here too, the `-o` flag indicates the path to the output file that will
contain the descriptors. Most importantly, you must also include the
`--include_imports` flag, or else the resulting file may be incomplete
and unusable by the Knit gateway.

```shell
protoc -I ../../others/proto foo/bar/services.proto \
    --include_imports -o ../my-services.protoset
```

Note that using `protoc` may involve specifying multiple input files
and multiple `-I` include path options. Refer to your existing scripts
that invoke `protoc` for code generation.

## Backend TLS Config

The top-level `backend_tls` property is a map that defines default TLS
settings. These settings are used with any backend with a secure URL
(i.e. scheme is "https"). These settings may be overridden on a per-backend
basis via the `tls` property for that backend.

The allowed keys in this map are the same as for the `tls` property for a
backend, [as described above](#tls-client-config). There is one exception:
the `backend_tls` map may _not_ include a `server_name` key. An override
server name can only be configured on a per-backend basis and cannot be
set in the defaults.

## Descriptors Config

The top-level `descriptors` property is a map that defines default settings
for polling and caching of descriptors, which define the schemas for the
supported RPC services. It allows the following keys:

- **`startup_max_wait_seconds`**: This is the maximum time to wait at startup
  for schemas to be resolved and all descriptors to be downloaded. If it takes
  longer than this to resolve schemas, the process will exit with an error
  instead of continuing to wait.

  If unset or zero, then a default value of 15 seconds is used.
- **`polling_period_seconds`**: The time to wait in between attempts to
  re-download descriptors. The gateway continually re-downloads schemas in case
  they change over time.

  If unset or set to zero, a default value of 15 minutes is used (which is 900
  seconds).
- **`polling_jitter`**: A value between zero and one for the amount of random
  jitter to use when scheduling a polling attempt. This is used to prevent
  multiple processes from inadvertently self-synchonizing and turning into a
  thundering herd. A typical value for this purpose is 0.1 to 0.3.

  A value of zero means no jitter. A value of 1 means 100% jitter, which means
  the polling period can be perturbed up to 100% (so it could be as low as zero
  as high as double the configured period). The default is 0.25.
- **`polling_debounce_seconds`**: The number of seconds to wait after a schema
  update to "debounce" updates from multiple sources.

  A smaller value means the gateway's internals are re-created more frequently
  when updates are frequent. A higher value can be more efficient as it re-creates
  the internals only once for a sequence of rapid updates, but it may slow down
  the reaction time from receiving a new schema and serving the corresponding new
  configuration. Note that the server could benefit from debouncing even when the
  polling period is high because there could be multiple sources of descriptors,
  so multiple updates could be arriving, all from a single scheduled re-polling of
  descriptors.

  The default is zero, which is appropriate for development. But production
  deployments with multiple schema sources should consider setting it to a value
  to prevent too much CPU time being used by re-creating configuration. Between
  5 ad 30 seconds is a reasonable range of values.
- **`cache`**: This section configures a cache for resolved schemas. For Buf BSR
  modules and gRPC reflection as descriptor sources, there is a possibility that
  a network partition could prevent the gateway from downloading a schema at
  startup. In this case, the cache can be used to fetch a last-known-good schema.
  The cache is updated whenever a new schema is successfully downloaded. It is
  used for loading the schemas if polling a backend source fails. This improves
  resilience of the gateway.

  Note that when loading schemas from local files, caching is not used. It is
  assumed that the local file will be at least as reliable as a cache source, so
  it's unnecessary.

  This property is a map with three possible keys: `file_system`, `redis`, or
  `memcache`. Only one of the three keys can be present since only a single
  cache source can be active. See [here](#descriptor-caching) for more details
  about configuring caches.

### Descriptor Caching

Descriptors that define the schemata of the gateway's supported RPC services
can be cached, to improve resilience in the face of a descriptor source being
unavailable.

The `cache` property of the `descriptors` top-level key may contain one of
the following:

- **`file_system`**: This caches the results on the file system. This is not
  a particularly good fit if the gateway is deployed as a workload where the
  storage is ephemeral (such that the cache will disappear when the workload
  restarts). But if it is deployed with a persistent disk or has a network
  filesystem mounted then this can work well. If the gateway workload will
  be auto-scaled horizontally (e.g. more replicas created on demand), then
  a network filesystem (where all replicas can share the files) works better
  and is may be easier to configure than a persistent volume.

  This property is a map that contains settings for caching via files.
  - **`directory`**: This first property is required and has no default. You
    must tell the gateway where to store cached data on the filesystem.
  - **`file_name_prefix`**: This is a prefix used in names of files that
    represent cache entries in the configured directory. The rest of the
    filename is a cache key. The default prefix is `"cache_"`. The trailing
    underscore is optional and will be automatically added if needed.
  - **`file_extension`**: This is the extension of cache files created. The
    default is `".bin"`. The leading dot is optional and will be automatically
    added if needed.
  - **`file_mode`**: The mode used to create the files. This will be combined
    with the gateway process's _umask_ to determine the actual permissions of
    created files. If unspecified or zero, defaults to 0600 (readable and
    writable by owner). This value must be in octal; the leading zero is
    optional.

- **`redis`**:  This caches the results in a Redis server. Redis servers
  typically are fast and have high up-time, making them suitable for use as
  a distributed/shared cache.

  This property is a map that contains settings for caching via Redis.
  - **`host`**: The only required property is the address of the Redis host.
    This should include both the host (domain name or IP address) and port.
  - **`require_auth`**: If auth is required by the Redis server, set this to
    true and also set `REDIS_USER` (optional) and `REDIS_PASSWORD` environment
    variables. If only a `REDIS_PASSWORD` is provided, the gateway will issue
    the `auth` command with only a password, for servers using the `requirepass`
    configuration option. For servers using the Redis ACL system (as of Redis
    6.0), both should be supplied.

    This setting defaults to false.
  - **`idle_timeout_seconds`**:  The idle timeout is used to close idle
    connections before they are closed by the server. To that end, this should
    be a value that is less than the server's timeout. If unspecified or zero,
    idle connections will not be closed.
  - **`database`**: If the gateway should store cache entries in a numbered
    database, indicate the database number here. By default, no `select` command
    is issued, so the default database (zero) is used.
  - **`key_prefix`**: The key prefix can be used to namespace keys, in case other
    workloads use the same Redis server to store data. The default value is empty.
  - **`expiry_seconds`**: An expiry may be applied to each cache entry. The entry
    will be auto-removed after this number of seconds elapses. By default, entries
    will be created without expiry (and never be deleted).

- **`memcache`**: This last option is for using memcached as a distributed/shared
  cache. This is similar in many regards to using Redis.

  This property is a map that contains settings for caching via memcached.
  - **`hosts`**: This first value is the only required value. The value must be
    an array of strings that define one or more hosts. If multiple hosts are
    provided, the cache entries will be distributed across them. This allows
    some cache entries to survive and be available, even if a single memcached
    server instance becomes unavailable or is reset.
  - **`key_prefix`**:  The key prefix can be used to namespace keys, in case other
    workloads use the same memcached servers to store data. The default value is
    empty.
  - **`expiry_seconds`**:  An expiry may be applied to each cache entry. The entry
    will be auto-removed after this number of seconds elapses. By default, entries
    will be created without expiry (and never be deleted).

## CORS Config

The top-level `cors` property is a map that defines default settings for how CORS
pre-flight requests are handled and what cross-origin requests are allowed. If
this section is not defined, all CORS pre-flight requests will get a negative
response (i.e. origin not allowed). This section allows the following keys:

- **`allowed_origins`**: This is an array of strings that define the list of
  allowed origins. Specifying a wildcard `"*"` means all origins are allowed.
  An entry in the list can include a single wildcard as a domain component. For
  example, `"https://*.foo.com"` allows all immediate sub-domains of foo.com.
  The default value is empty, which does not allow any origins.
- **`allowed_headers`**:  This is a list of allowed headers. The special wildcard
  entry `"*"` means all headers are allowed. The default value is empty, which
  does not allow any headers.
- **`allow_credentials`**: When true, the browser will be allowed to use
  credentials (such as client TLS certs or cookies) with cross-origin requests.
  The default value is false.
- **`allow_private_networks`**:  When true, the browser will be allowed to send
  cross-origin requests using a private network. The default value is false.
- **`max_age_seconds`**: This value allows the browser to cache the results of a
  pre-flight request, resulting in potentially fewer pre-flight requests to
  authorize future cross-origin requests. If this field is omitted or zero, no
  such header is sent to the browser. If not specified in a response header, the
  default for browsers is typically 5 seconds.
