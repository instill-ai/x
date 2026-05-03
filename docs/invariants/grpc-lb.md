# GRPC-INV-LB — gRPC Client-Side Load Balancing

## Invariant

Every gRPC connection from one Instill backend to another MUST use:

1. The `dns:///` resolver prefix (not `passthrough:///` or bare `host:port`)
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

## How it works

1. **Headless Service** (`clusterIP: None`): DNS returns A records for
   every ready pod IP, not a single virtual IP.

2. **`dns:///` resolver**: gRPC's built-in DNS resolver performs periodic
   re-resolution (default 30 min, configurable via `dns_min_time_between_resolutions_ms`
   channel arg). On each resolution, it discovers all pod IPs.

3. **`round_robin` policy**: gRPC opens a subchannel to each resolved
   address and distributes RPCs across them in round-robin order.

4. **Per-request transport** (HTTP/2 proxy): the `grpc-proxy` plugin
   creates a fresh `http2.Transport` per request, so each request dials
   a new connection. With a headless Service, DNS returns a random pod IP
   on each resolution, distributing traffic across replicas.

Together, these pieces ensure that when HPA adds pods, both gRPC clients
and HTTP/2 proxies distribute traffic to them.

## Where the fix lives

### Shared library (`x/client/grpc/clients.go`)

The `newConn()` function applies `dns:///` and `round_robin` to all
connections created via `grpc.NewClient[T]()`. Every backend that uses
`x/client` for inter-service gRPC calls inherits this behavior
automatically.

### Standalone gRPC callers (not using `x/client`)

These must apply the same two options manually:

| Caller | File |
|--------|------|
| api-gateway `simple-auth` plugin | `plugins/simple-auth/external.go` |
| api-gateway `blob` plugin | `plugins/blob/external.go` |
| api-gateway `registry` plugin | `plugins/registry/external.go` |
| pipeline-backend `instillartifact` component | `pkg/component/data/instillartifact/v0/client.go` |
| pipeline-backend `instillmodel` component | `pkg/component/ai/instillmodel/v0/client.go` |

Downstream consumers that maintain their own gRPC client construction
outside of `x/client` must also apply `dns:///` + `round_robin` manually.

### HTTP/2 proxy (`api-gateway/plugins/grpc-proxy/client.go`)

The `grpc-proxy` plugin creates a fresh `http2.Transport` + `http.Client`
inside each request handler invocation. This prevents Go's transport
connection pool from pinning all proxied traffic to a single pod.

### Helm charts

Each backend (artifact, pipeline, model, mgmt) has a headless sibling
Service named `{backend}-headless` alongside the existing ClusterIP
Service. Backend configmaps and the api-gateway configmap reference the
headless service name for all gRPC/HTTP2 backend hosts.

The ClusterIP Service is retained for non-gRPC consumers (health checks,
init containers) that only need to reach any healthy pod.

## Docker Compose compatibility

`dns:///` works in Docker Compose without changes — Docker DNS resolves
service names to container IPs directly. With `docker compose --scale`,
DNS returns multiple IPs and `round_robin` distributes across them.

## Regression check

To verify no new gRPC client bypasses this contract:

```shell
rg 'grpc\.(NewClient|Dial)\(' --glob '*.go' \
  | rg -v 'dns:///' \
  | rg -v '_test\.go' \
  | rg -v 'openfga'
```

Any hit that is a new inter-service call must add `dns:///` + `round_robin`.

## Anchored by 2026-05-03 incident

Added after diagnosing that `artifact-backend` HPA scaled to 2 pods but
only one received traffic (920m CPU vs 3m CPU). Inter-service gRPC calls
and the api-gateway's HTTP/2 proxy were both pinned to the hot pod,
causing cell processing timeouts and 528 orphan-reaped cells in a
production collection.
