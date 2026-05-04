# GRPC-INV-LB — gRPC Client-Side Load Balancing

## Invariant

Every gRPC connection from one Instill backend to another MUST use:

1. The `dns-refresh:///` resolver (not `dns:///`, `passthrough:///`, or bare `host:port`)
2. A `round_robin` load balancing policy via `grpc.WithDefaultServiceConfig`
3. A headless Kubernetes Service as the target (in K8s deployments)

Every HTTP/2 proxy path (e.g. the api-gateway `grpc-proxy` plugin) MUST
create a per-request transport to prevent connection pooling from pinning
all traffic to a single backend pod.

## Why

gRPC multiplexes all RPCs over a single HTTP/2 connection. With a regular
ClusterIP Service, kube-proxy performs connect-time DNAT: the first TCP
connect picks a pod, and that pod handles 100% of traffic for the
connection's lifetime. When HPA scales the backend from 1 to N pods,
existing clients never discover the new pods — their single HTTP/2
connection stays pinned to the original pod.

The same problem applies to raw HTTP/2 proxies (`http2.Transport`): Go's
transport pools connections by authority, so a shared transport pins all
proxied requests to one pod indefinitely.

This defeats HPA entirely: the hot pod stays at 90%+ CPU while new pods
sit idle. Worse, the HPA's average-CPU metric is diluted by the idle pods,
so it thinks the service is healthy and stops scaling. The Cluster
Autoscaler never provisions new nodes because there are no pending pods.

### Why `dns-refresh:///` instead of `dns:///`

The built-in gRPC `dns:///` resolver only re-resolves DNS when a subchannel
enters TRANSIENT_FAILURE — i.e., when an existing connection breaks. It does
**not** periodically poll DNS. This means that when HPA scales from 1 to N
pods, a healthy client with an active connection to the original pod will
**never** discover the new pods. All RPCs remain pinned to the original pod
indefinitely, defeating both HPA and round_robin.

The `dns-refresh:///` resolver (implemented in `x/client/grpc/resolver.go`)
solves this by periodically (every 30s) performing DNS A-record lookups and
pushing the full address list to the gRPC balancer. When HPA adds pods, the
resolver discovers them within one refresh cycle and `round_robin` opens
subchannels to them.

## How it works

1. **Headless Service** (`clusterIP: None`): DNS returns A records for
   every ready pod IP, not a single virtual IP.

2. **`dns-refresh:///` resolver** (`x/client/grpc/resolver.go`): Custom gRPC
   resolver that periodically (every 30s) looks up DNS A records via
   `net.DefaultResolver.LookupHost` and pushes updated addresses to the gRPC
   client connection via `resolver.ClientConn.UpdateState`. Unlike the built-in
   `dns:///` resolver, this discovers newly scaled pods without requiring a
   connection failure.

3. **`round_robin` policy**: gRPC opens a subchannel to each resolved
   address and distributes RPCs across them in round-robin order.

4. **Per-request transport** (HTTP/2 proxy): the `grpc-proxy` plugin
   creates a fresh `http2.Transport` per request, so each request dials
   a new connection. With a headless Service, DNS returns a random pod IP
   on each resolution, distributing traffic across replicas.

Together, these pieces ensure that when HPA adds pods, both gRPC clients
and HTTP/2 proxies distribute traffic to them within 30 seconds.

## Where the fix lives

### Shared library (`x/client/grpc/clients.go`)

The `newConn()` function applies `dns-refresh:///` + `round_robin` to all
connections created via `grpc.NewClient[T]()`. Every backend that uses
`x/client` for inter-service gRPC calls inherits this behavior
automatically — including pipeline-backend, artifact-backend, model-backend,
mgmt-backend, and agent-backend.

### Periodic DNS resolver (`x/client/grpc/resolver.go`)

`PeriodicDNSResolverBuilder` returns a `resolver.Builder` registered under
the `dns-refresh` scheme. Usage:

```go
grpc.NewClient("dns-refresh:///my-headless-svc:8091",
    grpc.WithResolvers(xgrpc.PeriodicDNSResolverBuilder(xgrpc.DefaultDNSRefreshInterval)),
    grpc.WithDefaultServiceConfig(`{"loadBalancingConfig":[{"round_robin":{}}]}`),
)
```

The resolver performs `net.DefaultResolver.LookupHost` on Build (immediate)
and every 30s thereafter. Each lookup pushes the full address set to the
balancer, so `round_robin` opens subchannels to newly HPA-scaled pods.

### Standalone gRPC callers (not using `x/client`)

These must apply `dns-refresh:///` + `round_robin` + `WithResolvers` manually:

| Caller | File |
|--------|------|
| api-gateway `simple-auth` plugin | `plugins/simple-auth/external.go` |
| api-gateway `blob` plugin | `plugins/blob/external.go` |
| api-gateway `registry` plugin | `plugins/registry/external.go` |
| pipeline-backend `instillartifact` component | `pkg/component/data/instillartifact/v0/client.go` |
| pipeline-backend `instillmodel` component | `pkg/component/ai/instillmodel/v0/client.go` |
| agent-backend-ee `llmworker` client | `pkg/grpc/llmworker/client.go` |

Downstream consumers that maintain their own gRPC client construction
outside of `x/client` must also apply the same pattern.

### OpenFGA gRPC client (`x/acl/client.go`)

`InitOpenFGAClient` creates the gRPC connection used by every backend
(artifact, pipeline, model, agent) for authorization checks. It uses
`dns-refresh:///` + `round_robin` targeting the headless OpenFGA Service so
that authorization RPCs are distributed across all OpenFGA pods.

### mgmt-backend OpenFGA HTTP client

mgmt-backend communicates with OpenFGA over HTTP REST (port 8080), not
gRPC. It uses the `openfga/go-sdk` with a custom `http.Client` whose
transport has `DisableKeepAlives: true`, forcing a fresh TCP connection
per request. kube-proxy DNAT distributes these connections across
OpenFGA pods via the regular ClusterIP Service.

### HTTP/2 proxy (`api-gateway/plugins/grpc-proxy/client.go`)

The `grpc-proxy` plugin creates a fresh `http2.Transport` + `http.Client`
inside each request handler invocation. This prevents Go's transport
connection pool from pinning all proxied traffic to a single pod.

### HTTP/1.1 REST proxy (`api-gateway/plugins/http-no-pool/client.go`)

The `http-no-pool` plugin is the HTTP/1.1 counterpart of `grpc-proxy`.
KrakenD's default HTTP backend proxy pools connections per-host
(`http.Transport` with keep-alive), pinning all REST requests to the same
pod for the pool's lifetime. This affects all `http_auth` and `no_auth`
endpoints: knowledge-bases, models, user profiles, health checks, etc.

The `http-no-pool-client` plugin replaces the pooled transport with a
per-request `http.Transport` (`DisableKeepAlives: true`). Each request
opens a fresh TCP connection; kube-proxy routes it to a randomly-selected
pod from the Service's endpoint set — including pods added by HPA
mid-test.

Applied via `"plugin/http-client"` in every `http_auth` and `no_auth`
backend block in the KrakenD templates (same location as `grpc-proxy-client`
in `grpc_auth` / `grpc_no_auth` blocks).

### HTTP/1.1 SSE & WebSocket proxies

The SSE streaming plugins (`pipeline-sse-streaming`, `model-sse-streaming`,
and their EE counterparts `agent-sse-streaming`, `agent-websocket`) use
HTTP/1.1 to proxy to backend pods. Each request handler creates a fresh
`http.Transport` with `DisableKeepAlives: true`. Without this, Go's
default transport pools connections per-host, pinning all SSE/WS traffic
from the gateway to whichever pod was first resolved — the same class of
bug as the HTTP/2 proxy case above.

### Helm charts — Service type split

Each backend (artifact, pipeline, model, mgmt) has both a regular
ClusterIP Service and a headless sibling (`{backend}-headless`).
OpenFGA and llmworker also have headless siblings for gRPC callers.

**The api-gateway uses them differently per protocol:**

| Protocol | Config source | K8s Service type | LB mechanism |
|----------|--------------|-----------------|--------------|
| gRPC (`grpc-proxy-client`) | `plugins.json` → `*_BACKEND_HOST` | Headless | `dns:///` + `round_robin` (client-side) |
| HTTP/1.1 (`http-no-pool-client`) | `backends.json` → `*_BACKEND_HTTP_HOST` | ClusterIP | kube-proxy DNAT (per-connection) |

The `envsubst.sh` script derives `*_BACKEND_HTTP_HOST` by stripping the
`-headless` suffix from `*_BACKEND_HOST`. For Docker Compose (where
hostnames never carry `-headless`), the stripping is a no-op.

## Docker Compose compatibility

`dns:///` works in Docker Compose without changes — Docker DNS resolves
service names to container IPs directly. With `docker compose --scale`,
DNS returns multiple IPs and `round_robin` distributes across them.

## Regression check

To verify no new gRPC client bypasses this contract:

```shell
rg 'grpc\.(NewClient|Dial)\(' --glob '*.go' \
  | rg -v 'dns-refresh:///' \
  | rg -v 'dns:///' \
  | rg -v '_test\.go'
```

Any hit that is a new inter-service call must add `dns-refresh:///` +
`round_robin` + `WithResolvers(PeriodicDNSResolverBuilder(...))`.

## Anchored by 2026-05-03 incident

Added after diagnosing that `artifact-backend` HPA scaled to 2 pods but
only one received traffic (920m CPU vs 3m CPU). Inter-service gRPC calls
and the api-gateway's HTTP/2 proxy were both pinned to the hot pod,
causing cell processing timeouts and 528 orphan-reaped cells in a
production collection.
