<!-- GSD:project-start source:PROJECT.md -->
## Project

**Protoconf — Quality & Consistency Overhaul**

Protoconf is a configuration management tool that uses Protocol Buffers as schema and Starlark as the configuration language. It compiles Starlark configs into materialized protobuf, distributes them via KV stores (Consul, etcd, ZooKeeper, Kubernetes ConfigMaps), and serves them to applications via gRPC streaming. This milestone focuses on comprehensive quality improvements: testing, consistency, security hardening, and modernizing deprecated patterns across the entire codebase.

**Core Value:** Every component must be testable, consistent, and free of runtime surprises — no panics in production code, no os.Exit in libraries, no deprecated APIs, and proper test coverage across all packages.

### Constraints

- **Tech stack**: Go 1.25.8+, must maintain backward compatibility with existing config repos
- **Proto compatibility**: Cannot break existing protobuf wire formats or gRPC service definitions
- **KV store interface**: Must remain compatible with valkeyrie store.Store interface
- **Build**: CGO_ENABLED=0, must produce static binaries
- **Testing**: Must not break existing CI (GitHub Actions with Codecov)
<!-- GSD:project-end -->

<!-- GSD:stack-start source:codebase/STACK.md -->
## Technology Stack

## Languages
- Go 1.25.8 - All core application code (`go.mod` line 3)
- Protocol Buffers (proto3) - Service and data type definitions (`agent/api/proto/v1/`, `server/api/proto/v1/`, `datatypes/proto/v1/`, `pb/protoconf/v1/`)
- Starlark - Configuration language executed by the compiler (`go.starlark.net v0.0.0-20240314022150`), user-authored `.pconf` and `.mpconf` files
- Python - Client SDK in `python/` (grpclib-based, `python/requirements.txt`)
## Runtime
- Go 1.25.8+ (all three CI workflows use `go-version-file: go.mod` to read the floor directly from the module)
- CGO disabled for production builds (`CGO_ENABLED=0` in `.goreleaser.yaml`)
- Go Modules
- Lockfile: `go.sum` present
## Frameworks
- gRPC (`google.golang.org/grpc v1.64.0`) - Agent and mutation server APIs
- Protocol Buffers (`google.golang.org/protobuf v1.34.1`) - Data serialization, dynamic message handling
- Starlark (`go.starlark.net`) - Configuration compilation engine, Starlark interpreter
- Valkeyrie (`github.com/kvtools/valkeyrie v1.0.0`) - Abstraction layer over KV stores
- `github.com/mitchellh/cli v1.1.5` - Command-line interface framework (`cmd/protoconf/main.go`)
- OpenTelemetry (`go.opentelemetry.io/otel v1.27.0`) - Tracing and metrics
- Prometheus (`github.com/prometheus/client_golang v1.19.1`) - Metrics exposition
- gRPC Prometheus (`github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0`) - gRPC metrics
- `github.com/stretchr/testify v1.9.0` - Assertions and test suites
- Go standard `testing` package
- GoReleaser v2.0 (`.goreleaser.yaml`) - Build, package, and release
- `go generate` - Proto code generation (`generate.go`)
- `protoc` with `--go_out` and `--go-grpc_out` plugins - Proto compilation
- Buf (`buf.yaml`) - Proto linting and breaking change detection
## Key Dependencies
- `go.starlark.net` - Starlark interpreter, the core compilation engine
- `google.golang.org/grpc v1.64.0` - gRPC server/client for agent and mutation APIs
- `google.golang.org/protobuf v1.34.1` - Protobuf runtime, dynamic message support
- `github.com/kvtools/valkeyrie v1.0.0` - KV store abstraction (Consul, etcd, ZooKeeper, ConfigMaps, file)
- `github.com/jhump/protoreflect v1.16.0` - Proto descriptor handling and reflection
- `github.com/bufbuild/protovalidate-go v0.6.2` - Proto message validation
- `github.com/kvtools/consul v1.0.2` - Consul KV backend
- `github.com/kvtools/etcdv3 v1.0.2` - etcd v3 KV backend
- `github.com/kvtools/zookeeper v1.0.2` - ZooKeeper KV backend
- `k8s.io/client-go v0.30.1` - Kubernetes API client for ConfigMaps backend
- `k8s.io/api v0.30.1` - Kubernetes API types
- `k8s.io/apimachinery v0.30.1` - Kubernetes object framework
- `github.com/go-git/go-git/v5 v5.12.0` - Git operations (mutation server commits changes)
- `github.com/hashicorp/go-getter v1.7.4` / `v2 v2.2.2` - Remote module fetching
- `github.com/fsnotify/fsnotify v1.7.0` - File system watching
- `github.com/fullstorydev/grpcui v1.4.1` - gRPC web UI for the dev server
- `github.com/qri-io/starlib v0.5.0` - Starlark standard library extensions
- `github.com/Masterminds/sprig/v3 v3.2.3` - Template functions
- `github.com/stephenafamo/orchestra v0.1.0` - Process orchestration
- `github.com/protoconf/libprotoconf v0.1.0` - Shared protoconf library
- `github.com/ghodss/yaml v1.0.0` - YAML support
- `github.com/pelletier/go-toml v1.9.5` / `v2 v2.2.2` - TOML support
## Configuration
- No `.env` files detected; configuration is done via CLI flags and protobuf-defined config structs
- Agent config defined in `agent/config/v1/agent_config.proto`
- Key settings: gRPC address, HTTP admin address, KV store type, store addresses, TLS config, log level
- `.goreleaser.yaml` - Release builds (linux/darwin, amd64)
- `generate.go` - `go:generate` directive for proto compilation
- `buf.yaml` - Buf module `buf.build/protoconf/protoconf` for proto linting
- `docker/Dockerfile` - Scratch-based container image
## Platform Requirements
- Go 1.25.8+
- `protoc` compiler with Go plugins (`protoc-gen-go`, `protoc-gen-go-grpc`)
- Buf CLI (for linting)
- Static binary (CGO disabled, scratch Docker image)
- One or more KV store backends: Consul, etcd, ZooKeeper, Kubernetes ConfigMaps, or local filesystem
- Target platforms: linux/amd64, darwin/amd64, darwin/arm64
- `protoconf/protoconf:{tag}` on Docker Hub
- `ghcr.io/protoconf/protoconf:{tag}` on GitHub Container Registry
<!-- GSD:stack-end -->

<!-- GSD:conventions-start source:CONVENTIONS.md -->
## Conventions

## Naming Patterns
- Use `snake_case.go` for all Go source files: `kv_agent_impl.go`, `module_service_test.go`
- Test files use `_test.go` suffix co-located with source: `compiler_test.go` next to `compiler.go`
- Proto-generated files use `.pb.go` and `_grpc.pb.go` suffixes: `protoconf.pb.go`, `protoconf_grpc.pb.go`
- CLI entry points use `command.go` in each package: `compiler/command.go`, `server/server.go`, `agent/agent.go`
- Use short, single-word lowercase names: `agent`, `compiler`, `inserter`, `server`, `consts`, `utils`
- Nested packages for versioned protos follow `v1` convention: `agent/config/v1`, `pb/protoconf/v1`
- Sub-packages under a feature use descriptive names: `compiler/lib`, `compiler/lib/parser`, `compiler/starproto`
- Use `PascalCase` for exported functions: `NewCompiler()`, `RunAgent()`, `CompileFile()`
- Constructors follow `New<Type>` pattern: `NewModuleService()`, `NewProtoconfKVAgent()`, `NewProtoconfMutationServer()`
- CLI command factories follow `Command() (cli.Command, error)` signature consistently across all packages
- Use `camelCase` for local variables: `protoconfRoot`, `kvStore`, `mainOutput`
- Package-level sentinel errors use `Err` prefix: `ErrBadConfigExtension`, `ErrStarlarkEval`, `ErrPreMutationScriptError`
- Constants use `PascalCase`: `AgentDefaultAddress`, `CompiledConfigExtension`, `SrcPath`
- Struct types use `PascalCase`: `Compiler`, `ProtoconfKVAgent`, `ProtoconfMutationServer`
- Internal CLI config structs use unexported `cliConfig` and `cliCommand` patterns
- Option functions follow `With<Name>` pattern: `WithCompiler()` in `server/server.go`
## Code Style
- `gofmt` enforced via Trunk (`.trunk/trunk.yaml`)
- `golangci-lint` enabled in Trunk for additional linting
- Trunk CI with `golangci-lint@1.55.2` and `gofmt@1.20.4`
- `buf-lint@1.28.1` for protobuf linting (config: `buf.yaml`)
- No `.golangci.yml` config file detected -- uses default golangci-lint rules
## Import Organization
- Proto-generated packages use descriptive aliases to avoid collisions:
- Side-effect imports for proto registration use blank identifier:
- No Go module path aliases or `replace` directives for internal packages
- All imports use full module path: `github.com/protoconf/protoconf/...`
## Error Handling
- Use sentinel errors defined as package-level `var` with `errors.New()`:
- Wrap errors using `errors.Join()` (Go 1.20+ pattern) throughout:
- Check errors with `errors.Is()`:
- Use `fmt.Errorf()` for context-rich non-sentinel errors:
- CLI commands return integer exit codes (0 = success, 1 = error, 2 = usage error)
## Logging
- Use `slog.Default()` or create a named logger with `slog.New()`:
- Use structured logging with typed fields:
- Package-level logger variable in `server/server.go`: `var logger = slog.Default()`
- Agent supports JSON logging via config: `slog.NewJSONHandler(os.Stderr, loggerHandlerOptions)`
- OpenTelemetry-aware logging via `slog-otel` wrapper in `agent/agent.go`
- Legacy `log.Printf` still appears in some older code paths (e.g., `compiler/command.go` line 176)
## CLI Command Pattern
## gRPC Server Pattern
## Functional Options Pattern
## Proto Code Generation
- Proto definitions live alongside generated code: `pb/protoconf/v1/protoconf.proto` generates `pb/protoconf/v1/protoconf.pb.go`
- Generation triggered via `go:generate` directive in `generate.go`:
- Generated `.pb.go` files excluded from Codecov coverage (`codecov.yml`)
## Constants
- File extensions: `ConfigExtension = ".pconf"`, `MultiConfigExtension = ".mpconf"`, `CompiledConfigExtension = ".materialized_JSON"`
- Directory paths: `SrcPath = "src/"`, `CompiledConfigPath = "materialized_config/"`, `MutableConfigPath = "mutable_config/"`
- Network defaults: `AgentDefaultAddress = ":4300"`, `ServerDefaultAddress = ":4301"`
## Comments
- Exported `Command()` functions get a brief `// Command is a cli.CommandFactory` comment
- TODO comments mark incomplete test cases: `// TODO: Add test cases.`
- TODO with author attribution: `// TODO(smintz): assert the response`
- Inline comments explain non-obvious behavior
## Module Design
- Each package exports a single primary constructor and type
- CLI packages export a `Command()` factory function
- Internal types (`cliCommand`, `cliConfig`) remain unexported
<!-- GSD:conventions-end -->

<!-- GSD:architecture-start source:ARCHITECTURE.md -->
## Architecture

## Pattern Overview
- Configuration-as-code: configs are defined in Starlark (`.pconf`/`.mpconf`) files that produce protobuf messages
- Compilation pipeline: Starlark source configs are compiled ("materialized") into JSON-encoded protobuf values
- Distribution via KV stores: materialized configs are inserted into key-value stores (Consul, etcd, ZooKeeper, Kubernetes ConfigMaps) and served to clients via a gRPC agent
- Mutation support: a mutation server accepts runtime config changes via gRPC, writes them to the mutable config directory, optionally recompiles, and runs pre/post scripts
- Module system: external protobuf dependencies are managed via a lock file and cache, similar to Go modules
## Component Diagram
```
```
## Layers
- Purpose: Parse command-line arguments and dispatch to core logic
- Location: `cmd/protoconf/main.go` (unified binary), `cmd/agent/`, `cmd/compiler/`, `cmd/inserter/`, `cmd/server/` (individual binaries)
- Contains: CLI command factories, flag parsing
- Depends on: `command/`, each component package
- Used by: End users, CI/CD pipelines
- Purpose: Shared CLI plumbing (subcommand routing, KV store flag config, UI helpers)
- Location: `command/command.go`
- Contains: `RunSubcommands()`, `RunCommand()`, `KVStoreConfig`, `DefaultUI`
- Depends on: `github.com/mitchellh/cli`, `consts/`
- Used by: All CLI entry points
- Purpose: Load Starlark config files, execute them, validate output, serialize to materialized JSON
- Location: `compiler/lib/`
- Contains: `Compiler` struct, Starlark loader, config execution, validation, module service
- Key files: `compiler/lib/compiler.go` (main compiler logic), `compiler/lib/starlark_loader.go` (Starlark module loading), `compiler/lib/config.go` (config execution and validation), `compiler/lib/module_service.go` (dependency management), `compiler/lib/starlark_functions.go` (built-in Starlark functions)
- Depends on: `compiler/starproto/`, `compiler/lib/parser/`, `consts/`, `go.starlark.net`, `jhump/protoreflect`
- Used by: `compiler/command.go`, `compiler/service.go`, `server/server.go`, `devserver/`
- Purpose: Bridge between Starlark values and protobuf messages, allowing Starlark code to construct protobuf objects
- Location: `compiler/starproto/`
- Contains: Message wrappers (`message.go`), field accessors, enum handling, map/repeated field support, Any type support
- Depends on: `go.starlark.net`, `jhump/protoreflect`
- Used by: `compiler/lib/`
- Purpose: Parse and resolve protobuf descriptors, read materialized config files
- Location: `compiler/lib/parser/parser.go`
- Contains: `Parser` struct with `FilesResolver` and `LocalResolver`
- Depends on: `google.golang.org/protobuf`
- Used by: `compiler/lib/`, `server/`, `inserter/`, `mutate/`
- Purpose: Serve configs from KV stores to applications via gRPC streaming with live updates
- Location: `agent/`
- Key files: `agent/agent.go` (RunAgent entry point, server setup with OTel), `agent/kv_agent_impl.go` (SubscribeForConfig implementation), `agent/kv_agent_rollout_impl.go` (rollout-aware agent), `agent/command.go` (CLI factory)
- Contains: gRPC server, KV store watching, OTel instrumentation, Prometheus metrics
- Depends on: `kvtools/valkeyrie`, `agent/filekv/`, `agent/otelkv/`, `agent/configmaps/`
- Used by: Client applications via gRPC
- Purpose: Accept config mutations via gRPC, write to mutable_config directory, optionally recompile, run pre/post scripts
- Location: `server/server.go`
- Contains: `ProtoconfMutationServer`, dynamic gRPC service registration based on protobuf descriptors, gRPC UI generation
- Depends on: `compiler/`, `compiler/lib/parser/`, `fullstorydev/grpcui`
- Used by: Mutation clients, `devserver/`
- Purpose: Read materialized configs from disk and insert them into KV stores with metadata and rollout support
- Location: `inserter/inserter.go`
- Contains: `ProtoconfInserter`, git metadata gathering, versioned insertion, rollout stage management
- Depends on: `kvtools/valkeyrie`, `go-git`, `compiler/lib/parser/`
- Used by: CI/CD pipelines
- Purpose: All-in-one development server combining agent, compiler service, and mutation server
- Location: `devserver/command.go`
- Contains: Combined gRPC server with agent + compiler + mutation services + health checks
- Depends on: `agent/`, `compiler/`, `server/`
- Used by: Local development
- Purpose: Manage external protobuf dependencies (download, cache, lock file)
- Location: `mod/command.go` (CLI), `compiler/lib/module_service.go` (core logic), `compiler/module/v1/` (protobuf config types)
- Contains: init/sync/tidy subcommands, dependency resolution via `go-getter`
- Depends on: `hashicorp/go-getter`, `compiler/module/v1/`
- Used by: `compiler/lib/`
## Data Flow
- Config source of truth: Git repository containing `src/` (Starlark configs) and `mutable_config/` (runtime mutations)
- Compiled artifacts: `materialized_config/` directory (derived, can be regenerated)
- Runtime config distribution: KV store (Consul, etcd, ZooKeeper, or Kubernetes ConfigMaps)
- Dependency cache: `.protoconf_cache/` directory with `protoconf.lock` file
## Key Abstractions
- Purpose: The core data envelope wrapping any protobuf message with metadata
- Definition: `pb/protoconf/v1/protoconf.proto`
- Contains: `proto_file` (source .proto path), `value` (google.protobuf.Any), optional `rollout_config`, optional `metadata`
- Pattern: Used as the serialization format for all materialized configs
- Purpose: Represents a loaded and executable Starlark config file
- Location: `compiler/lib/config.go`
- Contains: Starlark locals dict, validators map, protobuf resolver
- Pattern: Must define a `main()` function that returns protobuf message(s)
- Purpose: Manages external protobuf dependency resolution, downloading, and proto registry building
- Location: `compiler/lib/module_service.go`
- Pattern: Lock file based (`protoconf.lock`), cached downloads (`.protoconf_cache/`), builds a `DescriptorRegistry` for proto resolution
- Purpose: Uniform interface to multiple KV store backends
- Implementations: `agent/filekv/` (file-based, dev mode), `agent/otelkv/` (OTel-instrumented wrapper), `agent/configmaps/` (Kubernetes ConfigMaps), `agent/dummykv/` (test stub)
- Pattern: Strategy pattern via `kvtools/valkeyrie/store.Store` interface
## Entry Points
- Location: `cmd/protoconf/main.go`
- Subcommands: `agent`, `compile`, `devserver`, `fmt`, `insert`, `mutate`, `serve`, `mod init`, `mod sync`, `mod tidy`
- This is the primary entry point; individual binaries in `cmd/agent/`, `cmd/compiler/`, `cmd/inserter/`, `cmd/server/` exist as standalone alternatives
- `cmd/agent/main.go`: Runs only the agent
- `cmd/compiler/main.go`: Runs only the compiler
- `cmd/inserter/main.go`: Runs only the inserter
- `cmd/server/main.go`: Runs only the mutation server
- `ProtoconfService.SubscribeForConfig`: Agent serves config subscriptions (streaming)
- `ProtoconfMutationService.MutateConfig`: Server accepts config mutations (unary)
- `ProtoconfMutationReportService.ReportProgress`: Server accepts mutation progress reports (unary)
- `ProtoconfCompile.CompileFiles`: Compiler service (streaming)
## Error Handling
- Sentinel error variables at package level: `ErrBadConfigExtension`, `ErrMainNotFound`, `ErrStarlarkEval`, etc. in `compiler/lib/compiler.go` and `compiler/lib/config.go`
- `errors.Join()` used to combine sentinel errors with contextual details
- CLI commands return integer exit codes (0 for success, 1 for error)
- Starlark evaluation errors are unwrapped to show backtrace: `evalError.Backtrace()`
- gRPC errors are returned directly to callers
## Cross-Cutting Concerns
<!-- GSD:architecture-end -->

<!-- GSD:workflow-start source:GSD defaults -->
## GSD Workflow Enforcement

Before using Edit, Write, or other file-changing tools, start work through a GSD command so planning artifacts and execution context stay in sync.

Use these entry points:
- `/gsd:quick` for small fixes, doc updates, and ad-hoc tasks
- `/gsd:debug` for investigation and bug fixing
- `/gsd:execute-phase` for planned phase work

Do not make direct repo edits outside a GSD workflow unless the user explicitly asks to bypass it.
<!-- GSD:workflow-end -->



<!-- GSD:profile-start -->
## Developer Profile

> Profile not yet configured. Run `/gsd:profile-user` to generate your developer profile.
> This section is managed by `generate-claude-profile` -- do not edit manually.
<!-- GSD:profile-end -->
