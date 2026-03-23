# Technology Stack

**Analysis Date:** 2026-03-23

## Languages

**Primary:**
- Go 1.22.4 - All core application code (`go.mod` line 3)

**Secondary:**
- Protocol Buffers (proto3) - Service and data type definitions (`agent/api/proto/v1/`, `server/api/proto/v1/`, `datatypes/proto/v1/`, `pb/protoconf/v1/`)
- Starlark - Configuration language executed by the compiler (`go.starlark.net v0.0.0-20240314022150`), user-authored `.pconf` and `.mpconf` files
- Python - Client SDK in `python/` (grpclib-based, `python/requirements.txt`)

## Runtime

**Environment:**
- Go 1.22+ (CI uses `go-version: 1.22` in `.github/workflows/go.yml`)
- CGO disabled for production builds (`CGO_ENABLED=0` in `.goreleaser.yaml`)

**Package Manager:**
- Go Modules
- Lockfile: `go.sum` present

## Frameworks

**Core:**
- gRPC (`google.golang.org/grpc v1.64.0`) - Agent and mutation server APIs
- Protocol Buffers (`google.golang.org/protobuf v1.34.1`) - Data serialization, dynamic message handling
- Starlark (`go.starlark.net`) - Configuration compilation engine, Starlark interpreter
- Valkeyrie (`github.com/kvtools/valkeyrie v1.0.0`) - Abstraction layer over KV stores

**CLI:**
- `github.com/mitchellh/cli v1.1.5` - Command-line interface framework (`cmd/protoconf/main.go`)

**Observability:**
- OpenTelemetry (`go.opentelemetry.io/otel v1.27.0`) - Tracing and metrics
- Prometheus (`github.com/prometheus/client_golang v1.19.1`) - Metrics exposition
- gRPC Prometheus (`github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0`) - gRPC metrics

**Testing:**
- `github.com/stretchr/testify v1.9.0` - Assertions and test suites
- Go standard `testing` package

**Build/Dev:**
- GoReleaser v2.0 (`.goreleaser.yaml`) - Build, package, and release
- `go generate` - Proto code generation (`generate.go`)
- `protoc` with `--go_out` and `--go-grpc_out` plugins - Proto compilation
- Buf (`buf.yaml`) - Proto linting and breaking change detection

## Key Dependencies

**Critical:**
- `go.starlark.net` - Starlark interpreter, the core compilation engine
- `google.golang.org/grpc v1.64.0` - gRPC server/client for agent and mutation APIs
- `google.golang.org/protobuf v1.34.1` - Protobuf runtime, dynamic message support
- `github.com/kvtools/valkeyrie v1.0.0` - KV store abstraction (Consul, etcd, ZooKeeper, ConfigMaps, file)
- `github.com/jhump/protoreflect v1.16.0` - Proto descriptor handling and reflection
- `github.com/bufbuild/protovalidate-go v0.6.2` - Proto message validation

**KV Store Backends:**
- `github.com/kvtools/consul v1.0.2` - Consul KV backend
- `github.com/kvtools/etcdv3 v1.0.2` - etcd v3 KV backend
- `github.com/kvtools/zookeeper v1.0.2` - ZooKeeper KV backend

**Kubernetes:**
- `k8s.io/client-go v0.30.1` - Kubernetes API client for ConfigMaps backend
- `k8s.io/api v0.30.1` - Kubernetes API types
- `k8s.io/apimachinery v0.30.1` - Kubernetes object framework

**Infrastructure:**
- `github.com/go-git/go-git/v5 v5.12.0` - Git operations (mutation server commits changes)
- `github.com/hashicorp/go-getter v1.7.4` / `v2 v2.2.2` - Remote module fetching
- `github.com/fsnotify/fsnotify v1.7.0` - File system watching
- `github.com/fullstorydev/grpcui v1.4.1` - gRPC web UI for the dev server
- `github.com/qri-io/starlib v0.5.0` - Starlark standard library extensions
- `github.com/Masterminds/sprig/v3 v3.2.3` - Template functions
- `github.com/stephenafamo/orchestra v0.1.0` - Process orchestration
- `github.com/protoconf/libprotoconf v0.1.0` - Shared protoconf library

**Serialization:**
- `github.com/ghodss/yaml v1.0.0` - YAML support
- `github.com/pelletier/go-toml v1.9.5` / `v2 v2.2.2` - TOML support

## Configuration

**Environment:**
- No `.env` files detected; configuration is done via CLI flags and protobuf-defined config structs
- Agent config defined in `agent/config/v1/agent_config.proto`
- Key settings: gRPC address, HTTP admin address, KV store type, store addresses, TLS config, log level

**Build:**
- `.goreleaser.yaml` - Release builds (linux/darwin, amd64)
- `generate.go` - `go:generate` directive for proto compilation
- `buf.yaml` - Buf module `buf.build/protoconf/protoconf` for proto linting
- `docker/Dockerfile` - Scratch-based container image

## Platform Requirements

**Development:**
- Go 1.22+
- `protoc` compiler with Go plugins (`protoc-gen-go`, `protoc-gen-go-grpc`)
- Buf CLI (for linting)

**Production:**
- Static binary (CGO disabled, scratch Docker image)
- One or more KV store backends: Consul, etcd, ZooKeeper, Kubernetes ConfigMaps, or local filesystem
- Target platforms: linux/amd64, darwin/amd64, darwin/arm64

**Container Images:**
- `protoconf/protoconf:{tag}` on Docker Hub
- `ghcr.io/protoconf/protoconf:{tag}` on GitHub Container Registry

---

*Stack analysis: 2026-03-23*
