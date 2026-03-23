# Codebase Structure

**Analysis Date:** 2026-03-23

## Directory Layout

```
protoconf/
├── agent/                  # gRPC agent serving configs from KV stores
├── cmd/                    # Standalone binary entry points
│   ├── agent/              # Standalone agent binary
│   ├── compiler/           # Standalone compiler binary
│   ├── inserter/           # Standalone inserter binary
│   ├── protoconf/          # Unified protoconf binary (all subcommands)
│   └── server/             # Standalone mutation server binary
├── command/                # Shared CLI infrastructure
├── compiler/               # Starlark-to-protobuf compilation engine
│   ├── lib/                # Core compiler library
│   │   ├── outputs/        # Output format handling
│   │   └── parser/         # Protobuf descriptor parsing
│   ├── module/v1/          # Module config protobuf types
│   └── starproto/          # Starlark-protobuf bridge
├── consts/                 # Global constants
├── datatypes/proto/v1/     # Legacy protobuf datatypes
├── devserver/              # All-in-one development server
├── docker/                 # Docker build files
├── examples/               # Example protoconf projects and clients
│   ├── grpc_clients/       # Client examples (Go, Ruby, Rust, Java, Python)
│   ├── mutation/           # Mutation example
│   ├── protoconf/          # Example protoconf workspace
│   ├── python_client/      # Python client example
│   └── webhooks/           # Webhook examples
├── fmt/                    # Starlark formatter command
├── importers/              # Proto import resolution
├── inserter/               # KV store config inserter
├── mod/                    # Module management commands (init/sync/tidy)
├── mutate/                 # CLI mutation client
├── pb/protoconf/v1/        # Core protobuf definitions and generated Go code
├── python/                 # Python support files
├── server/                 # Mutation server (gRPC)
│   └── api/proto/v1/       # Legacy mutation API protobuf
├── test/                   # Test utilities
├── utils/                  # Shared utilities (descriptor registry)
├── web_ide/                # WebAssembly-based IDE prototype
├── go.mod                  # Go module definition
├── go.sum                  # Go dependency checksums
├── generate.go             # Go generate directives
├── buf.yaml                # Buf protobuf tool config
├── .goreleaser.yaml        # Release automation config
└── mkdocs.yml              # Documentation site config
```

## Directory Purposes

**`agent/`:**
- Purpose: gRPC agent that watches KV stores and streams config updates to clients
- Contains: Agent server setup, KV store implementations, command factory
- Key files:
  - `agent/agent.go`: `RunAgent()` function, OTel setup, KV store selection, gRPC server lifecycle
  - `agent/command.go`: CLI command factory with `libprotoconf`-based config
  - `agent/kv_agent_impl.go`: `ProtoconfKVAgent` implementing `SubscribeForConfig`
  - `agent/kv_agent_rollout_impl.go`: Rollout-aware agent variant
  - `agent/legacy.go`: Legacy API compatibility wrapper
  - `agent/filekv/filekv.go`: File-system based KV store (dev mode, uses `fsnotify`)
  - `agent/otelkv/otelkv.go`: OpenTelemetry instrumented KV store wrapper
  - `agent/configmaps/configmaps.go`: Kubernetes ConfigMaps KV store backend
  - `agent/dummykv/dummykv.go`: Test/dummy KV store
  - `agent/config/v1/`: Protobuf-defined agent configuration schema
  - `agent/api/proto/v1/`: Legacy gRPC service definition

**`cmd/`:**
- Purpose: Binary entry points
- Contains: `main.go` files that wire up command factories
- Key files:
  - `cmd/protoconf/main.go`: Unified binary registering all subcommands (agent, compile, devserver, fmt, insert, mutate, serve, mod init/sync/tidy)
  - `cmd/agent/main.go`, `cmd/compiler/main.go`, `cmd/inserter/main.go`, `cmd/server/main.go`: Standalone single-command binaries

**`command/`:**
- Purpose: Shared CLI plumbing
- Contains: Subcommand routing, KV store config flags, UI helpers
- Key files:
  - `command/command.go`: `RunSubcommands()`, `RunCommand()`, `KVStoreConfig`, `AddKVStoreFlags()`, `DefaultUI`, `PrefixedUi`

**`compiler/`:**
- Purpose: The Starlark compilation engine
- Contains: CLI command, gRPC compiler service, core library
- Key files:
  - `compiler/command.go`: CLI command factory, `GetAllConfigs()`, local/remote compilation, REPL mode
  - `compiler/service.go`: `CompilerService` implementing `ProtoconfCompile` gRPC service
  - `compiler/template.go`: Template file processing

**`compiler/lib/`:**
- Purpose: Core compiler logic
- Contains: Starlark execution, config validation, module management, output writing
- Key files:
  - `compiler/lib/compiler.go`: `Compiler` struct, `CompileFile()`, `CompileFileAsync()`, `writeConfig()`, `writeOutput()`
  - `compiler/lib/config.go`: `config` struct, `main()` execution, `validate()` with protovalidate + custom validators
  - `compiler/lib/starlark_loader.go`: `starlarkLoader` handling `.proto`, `.pconf`, `.pinc`, `mutable:` imports
  - `compiler/lib/starlark_functions.go`: Built-in Starlark functions (`fail`, `struct`, `module`, `object`)
  - `compiler/lib/module_service.go`: `ModuleService` for dependency download, caching, lock file management
  - `compiler/lib/filesystem.go`: File system abstraction (separate `filesystem_js.go` for WASM)
  - `compiler/lib/progress_bar.go`: Progress bar for compilation

**`compiler/lib/parser/`:**
- Purpose: Protobuf descriptor parsing and resolution
- Key files: `compiler/lib/parser/parser.go`

**`compiler/starproto/`:**
- Purpose: Bridge Starlark values to/from protobuf messages
- Key files: `compiler/starproto/message.go`, `compiler/starproto/field.go`, `compiler/starproto/enum.go`, `compiler/starproto/any.go`, `compiler/starproto/map.go`, `compiler/starproto/repeated.go`

**`consts/`:**
- Purpose: Global constants shared across all packages
- Key files: `consts/consts.go`
- Important values: `SrcPath` = `"src/"`, `CompiledConfigPath` = `"materialized_config/"`, `MutableConfigPath` = `"mutable_config/"`, `ConfigExtension` = `".pconf"`, `MultiConfigExtension` = `".mpconf"`, `CompiledConfigExtension` = `".materialized_JSON"`

**`devserver/`:**
- Purpose: All-in-one development server combining agent, compiler, and mutation server
- Key files: `devserver/command.go`

**`inserter/`:**
- Purpose: Insert materialized configs into KV stores with git metadata and rollout support
- Key files: `inserter/inserter.go`

**`mod/`:**
- Purpose: Module management CLI commands (init, sync, tidy)
- Key files: `mod/command.go`

**`mutate/`:**
- Purpose: CLI client for sending mutations to the mutation server
- Key files: `mutate/mutate.go`

**`server/`:**
- Purpose: Mutation server accepting config changes via gRPC
- Key files:
  - `server/server.go`: `ProtoconfMutationServer`, dynamic gRPC service registration, gRPC UI generation
  - `server/legacy.go`: Legacy API compatibility
  - `server/api/proto/v1/`: Legacy mutation API protobuf types

**`pb/protoconf/v1/`:**
- Purpose: Core protobuf definitions and generated Go code
- Key files:
  - `pb/protoconf/v1/protoconf.proto`: Defines `ProtoconfValue`, `ConfigSubscriptionRequest`, `ConfigUpdate`, `ConfigMutationRequest`, `ConfigMutationResponse`, `CompileRequest`, `CompileResponse`, and all gRPC services
  - `pb/protoconf/v1/protoconf.pb.go`: Generated protobuf Go code
  - `pb/protoconf/v1/protoconf_grpc.pb.go`: Generated gRPC Go code

**`utils/`:**
- Purpose: Shared utility code
- Key files: `utils/utils.go` (contains `DescriptorRegistry`)

**`importers/`:**
- Purpose: Proto import resolution with well-known type builders
- Key files: `importers/importers.go`, `importers/wktbuilders/`

**`fmt/`:**
- Purpose: Starlark file formatter command
- Key files: `fmt/command.go`

## Key File Locations

**Entry Points:**
- `cmd/protoconf/main.go`: Primary unified binary
- `cmd/agent/main.go`: Standalone agent binary
- `cmd/compiler/main.go`: Standalone compiler binary
- `cmd/inserter/main.go`: Standalone inserter binary
- `cmd/server/main.go`: Standalone server binary

**Configuration:**
- `consts/consts.go`: All path constants and default addresses
- `agent/config/v1/`: Agent configuration protobuf schema
- `compiler/module/v1/`: Module system configuration protobuf schema
- `.goreleaser.yaml`: Release configuration
- `buf.yaml`: Protobuf tooling configuration

**Core Logic:**
- `compiler/lib/compiler.go`: Main compilation pipeline
- `compiler/lib/starlark_loader.go`: Starlark module resolution and loading
- `compiler/lib/module_service.go`: Dependency management
- `agent/agent.go`: Agent server setup
- `agent/kv_agent_impl.go`: Config subscription implementation
- `server/server.go`: Mutation server and dynamic service registration
- `inserter/inserter.go`: KV store insertion with rollout support

**Protobuf Definitions:**
- `pb/protoconf/v1/protoconf.proto`: Core API types and services
- `agent/api/proto/v1/`: Legacy agent API
- `server/api/proto/v1/`: Legacy mutation API
- `datatypes/proto/v1/`: Legacy datatypes
- `agent/config/v1/`: Agent configuration schema

**Testing:**
- `compiler/command_test.go`: Compiler command tests
- `compiler/lib/compiler_test.go`: Compiler core tests
- `compiler/lib/module_service_test.go`: Module service tests
- `compiler/lib/parser/parser_test.go`: Parser tests
- `agent/agent_test.go`, `agent/command_test.go`, `agent/kv_agent_impl_test.go`, `agent/kv_agent_rollout_impl_test.go`: Agent tests
- `inserter/inserter_test.go`: Inserter tests
- `server/server_test.go`: Server tests
- `importers/importers_test.go`: Importer tests
- `utils/utils_test.go`: Utility tests
- `test/`: Test utilities directory
- `utils/testdata/`: Test data directory

## Naming Conventions

**Files:**
- Snake_case for Go files: `kv_agent_impl.go`, `module_service.go`, `starlark_loader.go`
- Command files named `command.go` in each package
- Test files use `_test.go` suffix: `compiler_test.go`, `agent_test.go`
- Protobuf files use snake_case: `protoconf.proto`
- Generated protobuf: `.pb.go` and `_grpc.pb.go` suffixes

**Directories:**
- Lowercase, single words where possible: `agent/`, `compiler/`, `inserter/`
- Protobuf versioning via `v1/` subdirectories: `pb/protoconf/v1/`, `agent/api/proto/v1/`
- Prefixed KV store implementations: `filekv/`, `otelkv/`, `dummykv/`

**Packages:**
- Package name matches directory name
- CLI command packages export a `Command()` factory function returning `(cli.Command, error)`

## Where to Add New Code

**New CLI Subcommand:**
- Create a new package directory at root level (e.g., `mycommand/`)
- Implement `command.go` with a `Command() (cli.Command, error)` factory function
- Implement the `cli.Command` interface (`Run`, `Help`, `Synopsis`)
- Register in `cmd/protoconf/main.go`

**New KV Store Backend:**
- Create `agent/<storename>/` directory
- Implement the `kvtools/valkeyrie/store.Store` interface
- Add initialization case in `agent/agent.go` `RunAgent()` switch statement
- Add initialization case in `inserter/inserter.go` if needed

**New Starlark Built-in Function:**
- Add to `compiler/lib/starlark_functions.go` in the `getModules()` function
- Or for proto-specific builtins, extend `compiler/starproto/`

**New Protobuf Type for Core API:**
- Add message to `pb/protoconf/v1/protoconf.proto`
- Regenerate with `buf generate` or protoc
- Generated files go to `pb/protoconf/v1/`

**New Compiler Output Format:**
- Add handling in `compiler/lib/compiler.go` `writeOutput()` method
- Follow the existing pattern for JSON/YAML/TOML detection via file suffix

**New Tests:**
- Place test files alongside the code they test (co-located pattern)
- Name with `_test.go` suffix
- Test data goes in `testdata/` subdirectories (see `utils/testdata/`)

## Special Directories

**`examples/protoconf/`:**
- Purpose: Example protoconf workspace demonstrating the directory layout
- Structure mirrors a real protoconf project: `src/` (Starlark configs), `materialized_config/` (compiled output), `mutable_config/` (runtime mutations)
- Generated: Partially (materialized_config is compiled output)
- Committed: Yes

**`.protoconf_cache/`:**
- Purpose: Downloaded external protobuf dependencies
- Generated: Yes (by `mod sync`)
- Committed: No (gitignored)

**`materialized_config/`:**
- Purpose: Compiled config output directory within a protoconf workspace
- Generated: Yes (by `protoconf compile`)
- Committed: Typically yes (for `protoconf insert` to read)

**`mutable_config/`:**
- Purpose: Runtime-mutated configs within a protoconf workspace
- Generated: Yes (by mutation server)
- Committed: Typically yes

**`outputs/`:**
- Purpose: Non-protobuf output files (JSON, YAML, TOML) written by compiler
- Generated: Yes
- Committed: Depends on workflow

**`.evolver/`:**
- Purpose: Unknown/experimental tooling (worktrees)
- Generated: Unknown
- Committed: No (untracked)

**`web_ide/`:**
- Purpose: Prototype WebAssembly-based IDE
- Contains: WASM build script, HTML, JS loader
- Note: Uses `compiler/lib/filesystem_js.go` for WASM filesystem abstraction

## Protoconf Workspace Layout

A protoconf workspace (the `protoconfRoot` parameter used throughout) follows this structure:

```
<workspace>/
├── src/                        # Starlark source configs
│   ├── *.proto                 # Protobuf schema definitions
│   ├── *.pconf                 # Single config files (must define main())
│   ├── *.mpconf                # Multi config files (main() returns dict)
│   ├── *.pinc                  # Starlark include/library files
│   └── *.proto-validator       # Custom validator functions
├── materialized_config/        # Compiled output (JSON protobuf)
│   └── **/*.materialized_JSON
├── mutable_config/             # Runtime mutations
│   └── **/*.materialized_JSON
├── outputs/                    # Non-protobuf outputs (JSON/YAML/TOML)
├── .protoconf_cache/           # Downloaded dependencies
└── protoconf.lock              # Dependency lock file
```

---

*Structure analysis: 2026-03-23*
