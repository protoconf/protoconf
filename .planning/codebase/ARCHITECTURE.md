# Architecture

**Analysis Date:** 2026-03-23

## Pattern Overview

**Overall:** Multi-component configuration management system built around Protocol Buffers and Starlark scripting

**Key Characteristics:**
- Configuration-as-code: configs are defined in Starlark (`.pconf`/`.mpconf`) files that produce protobuf messages
- Compilation pipeline: Starlark source configs are compiled ("materialized") into JSON-encoded protobuf values
- Distribution via KV stores: materialized configs are inserted into key-value stores (Consul, etcd, ZooKeeper, Kubernetes ConfigMaps) and served to clients via a gRPC agent
- Mutation support: a mutation server accepts runtime config changes via gRPC, writes them to the mutable config directory, optionally recompiles, and runs pre/post scripts
- Module system: external protobuf dependencies are managed via a lock file and cache, similar to Go modules

**System Purpose:**
Protoconf is a configuration management tool that uses Protocol Buffers as the schema for configuration and Starlark as the language for generating configuration values. It provides a full lifecycle: authoring configs in Starlark, compiling them to materialized protobuf, inserting them into distributed KV stores, and serving them to applications via a gRPC streaming API with live updates.

## Component Diagram

```
                    +------------------+
                    |   User / CI/CD   |
                    +--------+---------+
                             |
              +--------------+--------------+
              |              |              |
              v              v              v
       +------+------+ +----+-----+ +------+------+
       |  protoconf  | | protoconf| | protoconf   |
       |   compile   | |  mutate  | |  mod init/  |
       |             | |          | |  sync/tidy  |
       +------+------+ +----+-----+ +------+------+
              |              |              |
              v              v              v
       +------+------+ +----+--------+ +---+----------+
       |  Compiler   | |  Mutation   | | Module       |
       |  (Starlark  | |  Server     | | Service      |
       |   Engine)   | |  (gRPC)     | | (Dependency  |
       +------+------+ +----+--------+ |  Resolver)   |
              |              |          +--------------+
              v              |
   +----------+-----------+  |
   | materialized_config/ |<-+
   | (JSON protobuf)      |
   +----------+-----------+
              |
              v
       +------+------+
       |  protoconf  |
       |   insert    |
       +------+------+
              |
              v
     +--------+--------+
     |   KV Store       |
     | (Consul/etcd/    |
     |  ZooKeeper/      |
     |  ConfigMaps)     |
     +--------+---------+
              |
              v
       +------+------+
       |  protoconf  |
       |   agent     |
       |  (gRPC)     |
       +------+------+
              |
              v
     +--------+---------+
     |  Client Apps     |
     |  (gRPC streaming |
     |   subscribers)   |
     +------------------+
```

## Layers

**CLI Layer:**
- Purpose: Parse command-line arguments and dispatch to core logic
- Location: `cmd/protoconf/main.go` (unified binary), `cmd/agent/`, `cmd/compiler/`, `cmd/inserter/`, `cmd/server/` (individual binaries)
- Contains: CLI command factories, flag parsing
- Depends on: `command/`, each component package
- Used by: End users, CI/CD pipelines

**Command Infrastructure:**
- Purpose: Shared CLI plumbing (subcommand routing, KV store flag config, UI helpers)
- Location: `command/command.go`
- Contains: `RunSubcommands()`, `RunCommand()`, `KVStoreConfig`, `DefaultUI`
- Depends on: `github.com/mitchellh/cli`, `consts/`
- Used by: All CLI entry points

**Compiler Core:**
- Purpose: Load Starlark config files, execute them, validate output, serialize to materialized JSON
- Location: `compiler/lib/`
- Contains: `Compiler` struct, Starlark loader, config execution, validation, module service
- Key files: `compiler/lib/compiler.go` (main compiler logic), `compiler/lib/starlark_loader.go` (Starlark module loading), `compiler/lib/config.go` (config execution and validation), `compiler/lib/module_service.go` (dependency management), `compiler/lib/starlark_functions.go` (built-in Starlark functions)
- Depends on: `compiler/starproto/`, `compiler/lib/parser/`, `consts/`, `go.starlark.net`, `jhump/protoreflect`
- Used by: `compiler/command.go`, `compiler/service.go`, `server/server.go`, `devserver/`

**Starlark-Protobuf Bridge:**
- Purpose: Bridge between Starlark values and protobuf messages, allowing Starlark code to construct protobuf objects
- Location: `compiler/starproto/`
- Contains: Message wrappers (`message.go`), field accessors, enum handling, map/repeated field support, Any type support
- Depends on: `go.starlark.net`, `jhump/protoreflect`
- Used by: `compiler/lib/`

**Proto Parser:**
- Purpose: Parse and resolve protobuf descriptors, read materialized config files
- Location: `compiler/lib/parser/parser.go`
- Contains: `Parser` struct with `FilesResolver` and `LocalResolver`
- Depends on: `google.golang.org/protobuf`
- Used by: `compiler/lib/`, `server/`, `inserter/`, `mutate/`

**Agent:**
- Purpose: Serve configs from KV stores to applications via gRPC streaming with live updates
- Location: `agent/`
- Key files: `agent/agent.go` (RunAgent entry point, server setup with OTel), `agent/kv_agent_impl.go` (SubscribeForConfig implementation), `agent/kv_agent_rollout_impl.go` (rollout-aware agent), `agent/command.go` (CLI factory)
- Contains: gRPC server, KV store watching, OTel instrumentation, Prometheus metrics
- Depends on: `kvtools/valkeyrie`, `agent/filekv/`, `agent/otelkv/`, `agent/configmaps/`
- Used by: Client applications via gRPC

**Mutation Server:**
- Purpose: Accept config mutations via gRPC, write to mutable_config directory, optionally recompile, run pre/post scripts
- Location: `server/server.go`
- Contains: `ProtoconfMutationServer`, dynamic gRPC service registration based on protobuf descriptors, gRPC UI generation
- Depends on: `compiler/`, `compiler/lib/parser/`, `fullstorydev/grpcui`
- Used by: Mutation clients, `devserver/`

**Inserter:**
- Purpose: Read materialized configs from disk and insert them into KV stores with metadata and rollout support
- Location: `inserter/inserter.go`
- Contains: `ProtoconfInserter`, git metadata gathering, versioned insertion, rollout stage management
- Depends on: `kvtools/valkeyrie`, `go-git`, `compiler/lib/parser/`
- Used by: CI/CD pipelines

**Dev Server:**
- Purpose: All-in-one development server combining agent, compiler service, and mutation server
- Location: `devserver/command.go`
- Contains: Combined gRPC server with agent + compiler + mutation services + health checks
- Depends on: `agent/`, `compiler/`, `server/`
- Used by: Local development

**Module System:**
- Purpose: Manage external protobuf dependencies (download, cache, lock file)
- Location: `mod/command.go` (CLI), `compiler/lib/module_service.go` (core logic), `compiler/module/v1/` (protobuf config types)
- Contains: init/sync/tidy subcommands, dependency resolution via `go-getter`
- Depends on: `hashicorp/go-getter`, `compiler/module/v1/`
- Used by: `compiler/lib/`

## Data Flow

**Config Compilation Flow:**

1. User writes `.pconf` (single config) or `.mpconf` (multi config) Starlark files in `src/` directory
2. `protoconf compile <root>` invokes `compiler/lib.Compiler.CompileFile()`
3. Compiler creates a `starlarkLoader` which resolves imports: `.proto` files become Starlark-accessible protobuf constructors via `starproto/`, `.pinc` files are included Starlark libraries, `mutable:` prefix reads from `mutable_config/`
4. Starlark `main()` function is called and must return a protobuf message (single config) or dict of string->protobuf (multi config)
5. Output is validated using both `protovalidate` and optional Starlark validators (`.proto-validator` files)
6. Materialized output is written as JSON to `materialized_config/` with `.materialized_JSON` extension

**Config Distribution Flow:**

1. `protoconf insert <root> <config>` reads materialized JSON from `materialized_config/`
2. Inserter gathers git metadata (commit hash, author, timestamps)
3. Config is serialized as both JSON and binary protobuf, then written to KV store paths: `<prefix>/<config>/config.json`, `<prefix>/<config>/config.data`, `<prefix>/<config>/metadata.json`
4. If rollout config is present, stages are applied sequentially with cooldown periods
5. `protoconf agent` watches KV store paths and streams updates to subscribed clients via `ProtoconfService.SubscribeForConfig` gRPC streaming RPC

**Mutation Flow:**

1. Client calls `ProtoconfMutationService.MutateConfig` gRPC endpoint on mutation server
2. Server marshals the protobuf value to JSON and writes it to `mutable_config/<path>.materialized_JSON`
3. Optional pre-mutation script is executed
4. If an embedded compiler is configured (as in devserver), all configs are recompiled
5. Optional post-mutation script is executed
6. Response includes timing data for each phase

**State Management:**
- Config source of truth: Git repository containing `src/` (Starlark configs) and `mutable_config/` (runtime mutations)
- Compiled artifacts: `materialized_config/` directory (derived, can be regenerated)
- Runtime config distribution: KV store (Consul, etcd, ZooKeeper, or Kubernetes ConfigMaps)
- Dependency cache: `.protoconf_cache/` directory with `protoconf.lock` file

## Key Abstractions

**ProtoconfValue:**
- Purpose: The core data envelope wrapping any protobuf message with metadata
- Definition: `pb/protoconf/v1/protoconf.proto`
- Contains: `proto_file` (source .proto path), `value` (google.protobuf.Any), optional `rollout_config`, optional `metadata`
- Pattern: Used as the serialization format for all materialized configs

**Starlark Config (`config` struct):**
- Purpose: Represents a loaded and executable Starlark config file
- Location: `compiler/lib/config.go`
- Contains: Starlark locals dict, validators map, protobuf resolver
- Pattern: Must define a `main()` function that returns protobuf message(s)

**ModuleService:**
- Purpose: Manages external protobuf dependency resolution, downloading, and proto registry building
- Location: `compiler/lib/module_service.go`
- Pattern: Lock file based (`protoconf.lock`), cached downloads (`.protoconf_cache/`), builds a `DescriptorRegistry` for proto resolution

**KV Store Abstraction (valkeyrie):**
- Purpose: Uniform interface to multiple KV store backends
- Implementations: `agent/filekv/` (file-based, dev mode), `agent/otelkv/` (OTel-instrumented wrapper), `agent/configmaps/` (Kubernetes ConfigMaps), `agent/dummykv/` (test stub)
- Pattern: Strategy pattern via `kvtools/valkeyrie/store.Store` interface

## Entry Points

**Unified Binary (`protoconf`):**
- Location: `cmd/protoconf/main.go`
- Subcommands: `agent`, `compile`, `devserver`, `fmt`, `insert`, `mutate`, `serve`, `mod init`, `mod sync`, `mod tidy`
- This is the primary entry point; individual binaries in `cmd/agent/`, `cmd/compiler/`, `cmd/inserter/`, `cmd/server/` exist as standalone alternatives

**Standalone Binaries:**
- `cmd/agent/main.go`: Runs only the agent
- `cmd/compiler/main.go`: Runs only the compiler
- `cmd/inserter/main.go`: Runs only the inserter
- `cmd/server/main.go`: Runs only the mutation server

**gRPC Services (defined in `pb/protoconf/v1/protoconf.proto`):**
- `ProtoconfService.SubscribeForConfig`: Agent serves config subscriptions (streaming)
- `ProtoconfMutationService.MutateConfig`: Server accepts config mutations (unary)
- `ProtoconfMutationReportService.ReportProgress`: Server accepts mutation progress reports (unary)
- `ProtoconfCompile.CompileFiles`: Compiler service (streaming)

## Error Handling

**Strategy:** Sentinel errors with `errors.Join` for context wrapping

**Patterns:**
- Sentinel error variables at package level: `ErrBadConfigExtension`, `ErrMainNotFound`, `ErrStarlarkEval`, etc. in `compiler/lib/compiler.go` and `compiler/lib/config.go`
- `errors.Join()` used to combine sentinel errors with contextual details
- CLI commands return integer exit codes (0 for success, 1 for error)
- Starlark evaluation errors are unwrapped to show backtrace: `evalError.Backtrace()`
- gRPC errors are returned directly to callers

## Cross-Cutting Concerns

**Logging:** `log/slog` (structured logging) used throughout. Agent supports JSON or text output controlled by config. OTel-enriched logging via `slog-otel` handler.

**Observability:** Full OpenTelemetry integration in agent (`agent/agent.go`) and server (`server/server.go`) with tracing (OTLP exporter), metrics (OTLP exporter), and Prometheus metrics endpoint. Agent wraps KV store with `agent/otelkv/` for trace propagation.

**Validation:** Two-tier validation: (1) `protovalidate` for proto-level constraints, (2) Optional Starlark validator functions (`.proto-validator` files) for custom business logic. Both run in `compiler/lib/config.go`.

**Authentication:** Not built-in. gRPC servers use insecure transport by default. Agent config has an `insecure` flag placeholder.

**Configuration of the tool itself:** Agent uses `libprotoconf` to self-configure from environment variables (`PROTOCONF_AGENT_*` prefix), config files (JSON/YAML/protobuf), and CLI flags. Config schema defined in `agent/config/v1/`.

---

*Architecture analysis: 2026-03-23*
