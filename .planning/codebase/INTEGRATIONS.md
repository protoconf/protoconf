# External Integrations

**Analysis Date:** 2026-03-23

## APIs & External Services

**gRPC Services (self-hosted):**
- ProtoconfService (Agent) - Streams config updates to subscribers
  - Proto: `agent/api/proto/v1/protoconf_service.proto`
  - RPC: `SubscribeForConfig` (server-streaming)
  - Implementation: `agent/kv_agent_impl.go`
- ProtoconfMutationService (Server) - Accepts config mutations via RPC
  - Proto: `server/api/proto/v1/protoconf_mutation.proto`
  - RPC: `MutateConfig`
  - Implementation: `server/server.go`

**gRPC UI:**
- `github.com/fullstorydev/grpcui` - Embedded gRPC web UI in the dev server (`server/server.go`)

## Data Storage

**Key-Value Stores (via Valkeyrie abstraction):**
- HashiCorp Consul
  - Client: `github.com/kvtools/consul v1.0.2`
  - Default address: `127.0.0.1:8500`
  - Agent config enum: `consul` (0)
- etcd v3
  - Client: `github.com/kvtools/etcdv3 v1.0.2`
  - Default address: `127.0.0.1:2379` (defined in `consts/consts.go`)
  - Agent config enum: `etcd` (1)
- Apache ZooKeeper
  - Client: `github.com/kvtools/zookeeper v1.0.2`
  - Default address: `127.0.0.1:2181` (defined in `consts/consts.go`)
  - Agent config enum: `zookeeper` (2)
- Local Filesystem
  - Implementation: `agent/filekv/`
  - Agent config enum: `file` (3)
- Kubernetes ConfigMaps
  - Implementation: `agent/configmaps/configmaps.go`
  - Client: `k8s.io/client-go v0.30.1`
  - Agent config enum: `configmaps` (4)
  - Uses in-cluster config or kubeconfig

**Git (for mutation server):**
- `github.com/go-git/go-git/v5 v5.12.0` - The mutation server commits config changes to a Git repository
- Implementation: `server/server.go`

**File Storage:**
- Local filesystem for compiled configs (`materialized_config/` directory)
- Mutable configs stored in `mutable_config/` directory
- Source protos in `src/` directory

**Caching:**
- None (KV stores serve as the config distribution layer)

## Authentication & Identity

**TLS:**
- Agent supports mutual TLS for gRPC connections
  - Config: `TLSConfig` message in `agent/config/v1/agent_config.proto`
  - Supports key/cert/CA as file paths or inline text
- Store connections support TLS via `store_tls` config field

**Auth Provider:**
- No built-in authentication beyond TLS
- Relies on KV store native auth mechanisms

## Monitoring & Observability

**Tracing:**
- OpenTelemetry with OTLP/gRPC exporter
  - `go.opentelemetry.io/otel v1.27.0`
  - `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.27.0`
  - gRPC instrumentation: `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.52.0`
  - Used in both agent (`agent/agent.go`) and server (`server/server.go`)

**Metrics:**
- Prometheus metrics via `/metrics` HTTP endpoint on the agent admin server
  - `github.com/prometheus/client_golang v1.19.1`
  - `github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0`
- OpenTelemetry metrics with OTLP/gRPC exporter
  - `go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.27.0`

**Logging:**
- `log/slog` (Go standard library structured logging)
- `github.com/remychantenay/slog-otel v1.3.0` - Bridges slog to OpenTelemetry
- Configurable: JSON format, log level (DEBUG/INFO/WARN/ERROR), source location

**Error Tracking:**
- None detected (no Sentry, Bugsnag, etc.)

**Profiling:**
- `net/http/pprof` imported in `agent/agent.go` - Go pprof available on admin HTTP server

## CI/CD & Deployment

**CI Pipeline (GitHub Actions):**
- `.github/workflows/go.yml` - Build and test on push/PR to main
  - Go 1.22, `go build`, `go test` with race detector and coverage
  - Coverage uploaded to Codecov (`codecov/codecov-action@v4.5.0`)
  - Test report via `gotest.tools/gotestsum`
- `.github/workflows/codeql-analysis.yml` - CodeQL security scanning (Go)
  - Runs on push/PR to main and weekly schedule
- `.github/workflows/release.yml` - GoReleaser on tag push
  - Builds binaries for linux and darwin
  - Publishes Docker images to Docker Hub and GHCR
  - Publishes Homebrew formula to `protoconf/homebrew-tap`

**Release Tooling:**
- GoReleaser v2.0 (`.goreleaser.yaml`)
  - Builds: `cmd/protoconf` main binary
  - Docker: scratch-based images, linux/amd64
  - Homebrew: `protoconf/homebrew-tap` repository
  - Checksums and changelog generation

**Container Registries:**
- Docker Hub: `protoconf/protoconf`
- GitHub Container Registry: `ghcr.io/protoconf/protoconf`

**Dependency Management:**
- Renovate (`renovate.json`) - Automated dependency updates

**Code Coverage:**
- Codecov (`codecov.yml`) - Ignores `**/*.pb.go` generated files

## Remote Module Fetching

**go-getter:**
- `github.com/hashicorp/go-getter v1.7.4` and `v2 v2.2.2`
- Used for fetching remote proto modules (supports Git, HTTP, S3, GCS URLs)
- Module management: `mod/` directory (`mod init`, `mod sync`, `mod tidy` commands)

## Kubernetes Integration

**ConfigMaps Backend:**
- `agent/configmaps/configmaps.go` - Implements Valkeyrie store interface over Kubernetes ConfigMaps
- Uses `k8s.io/client-go` for in-cluster and kubeconfig-based auth
- Supports watch for real-time config updates

## Documentation

**MkDocs:**
- `mkdocs.yml` - Material for MkDocs theme
- Hosted at `https://protoconf.github.io/protoconf`
- Google Analytics: `G-H3NLBNNQD7`

## Proto Ecosystem

**Buf:**
- Module: `buf.build/protoconf/protoconf` (`buf.yaml`)
- Linting: DEFAULT rules
- Breaking change detection: FILE-level

**Proto Validation:**
- `github.com/bufbuild/protovalidate-go v0.6.2` - Runtime proto validation

## Client SDKs

**Python:**
- Located in `python/`
- Uses `grpclib==0.4.7` (`python/requirements.txt`)
- Provides gRPC client for ProtoconfService

## Webhooks & Callbacks

**Incoming:**
- None detected

**Outgoing:**
- None detected

## Environment Configuration

**Required env vars (CI only):**
- `CODECOV_TOKEN` - Codecov upload token
- `DOCKERHUB_USER` / `DOCKERHUB_TOKEN` - Docker Hub publish credentials
- `DEPLOY_GITHUB_TOKEN` - Homebrew tap publish token
- `GITHUB_TOKEN` - GitHub releases and GHCR

**Runtime configuration:**
- All runtime config via CLI flags parsed into protobuf config structs
- No environment variable configuration for the application itself

---

*Integration audit: 2026-03-23*
