# Phase 7: Proto-Defined CLI Configs - Context

**Gathered:** 2026-03-28
**Status:** Ready for planning

<domain>
## Phase Boundary

Define protobuf messages for all component configurations: server, compiler, inserter, and mutate CLI. This phase creates the `.proto` files only — wiring them to CLI flag generation (`libprotoconf.PopulateFlagSet`) and config file loading is Phase 8. The proto definitions must compile with `protoc` and produce valid `.pb.go` files.

</domain>

<decisions>
## Implementation Decisions

### Proto Package Organization — PCLI-01 through PCLI-04
- **D-01:** One proto file per component, following the agent pattern: `{component}/config/v1/{component}_config.proto`. Specifically:
  - `server/config/v1/server_config.proto` — `ServerConfig` message
  - `compiler/config/v1/compiler_config.proto` — `CompilerConfig` message
  - `inserter/config/v1/inserter_config.proto` — `InserterConfig` message
  - `mutate/config/v1/mutate_config.proto` — `MutateConfig` message
- **D-02:** Each proto file uses `option go_package` following existing convention: `github.com/protoconf/protoconf/{component}/config/v1;protoconf_{component}_config`

### Backward Compatibility
- **D-03:** Every proto field MUST have a `json_name` field option that exactly matches the current CLI flag name. Example: `string grpc_address = 1 [json_name = "grpc-address"];` maps to `--grpc-address`. This ensures Phase 8's `PopulateFlagSet` generates identical flag names.
- **D-04:** Default values in proto should match current Go defaults where possible. For strings this means empty string (proto default). For numeric/bool, use proto default (0/false). The Go code sets non-proto defaults (like `:4301`) at construction time, matching the agent pattern.

### Field Mapping — What Goes in Each Config
- **D-05:** `ServerConfig` includes: `grpc_address`, `pre_mutation_script`, `post_mutation_script`, `tls_cert`, `tls_key`, `tls_ca`, `auth_token`. Reuse `TLSConfig` message from agent by importing, OR define TLS fields inline as flat strings (Claude's discretion on which is cleaner).
- **D-06:** `CompilerConfig` includes: `repl` (bool), `verbose_logging` (bool), `process_templates` (bool), `cpuprofile` (string), `memprofile` (string). The positional args (protoconf_root, files...) remain positional — they are NOT proto fields.
- **D-07:** `InserterConfig` includes KV store fields currently in `command.KVStoreConfig`: `store` (enum), `store_address` (repeated string), `prefix` (string), plus inserter-specific: `delete` (bool). Rollout-related fields if they exist as flags.
- **D-08:** `MutateConfig` includes: `protoconf_root`, `proto_file`, `proto_msg`, `server_address`, `config_path`, `metadata_str`, `tls_cert`, `tls_key`, `tls_ca`, `insecure_tls` (bool), plus field/value pairs handled via repeated message or similar.

### Scope
- **D-09:** fmt and devserver do NOT get proto configs in this phase. fmt has only 2 trivial flags (`--write`, `--diff`). devserver composes agent + compiler + server configs and will inherit proto configs from those components.
- **D-10:** The `command/command.go` `KVStoreConfig` shared struct is NOT converted to proto in this phase — inserter's proto will define its own KV fields. Agent already has its own `StoreType` enum and store fields. Consolidating shared KV config is a future concern.

### Proto Generation
- **D-11:** Generated `.pb.go` files are committed to the repo, following existing convention (agent's `.pb.go` files are committed).
- **D-12:** Add proto files to `buf.yaml` module and ensure they pass `buf lint`.
- **D-13:** Add `go:generate` directives for new proto files, following the pattern in `generate.go`.

### Claude's Discretion
- Whether to import and reuse `AgentConfig.TLSConfig` in `ServerConfig`/`MutateConfig` or define TLS fields as flat strings
- Whether `InserterConfig` should use the same `StoreType` enum as `AgentConfig` (via import) or define its own
- Exact field numbering in proto messages
- Whether to add field-level comments mirroring current flag help text

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Existing proto config pattern (reference implementation)
- `agent/config/v1/agent_config.proto` — The established pattern for proto-defined configs with `json_name` field options
- `agent/command.go` — Shows how `libprotoconf.PopulateFlagSet` wires proto to CLI flags (Phase 8 will replicate this)

### Current CLI configs (what proto definitions must capture)
- `server/server.go` lines 62-69 — `cliConfig` struct with grpc_address, pre/post scripts, TLS, auth_token
- `compiler/command.go` lines 31-36 — `cliConfig` struct with repl, verbose, templates, profiling
- `inserter/inserter.go` lines 42-44 — `cliConfig` struct with delete flag
- `mutate/mutate.go` lines 58-70 — `cliConfig` struct with root, proto file/msg, server address, TLS
- `command/command.go` — `KVStoreConfig` shared struct used by inserter and agent

### Requirements
- `.planning/REQUIREMENTS.md` — PCLI-01 through PCLI-04 define proto config requirements

### Proto build infrastructure
- `generate.go` — `go:generate` directives for proto compilation
- `buf.yaml` — Buf module configuration for proto linting

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `agent/config/v1/agent_config.proto` — Reference implementation for proto configs with `json_name`, enums, nested messages
- `libprotoconf` package — Already imported by agent and mod; provides `NewConfig()`, `PopulateFlagSet()`, `Environment()`, `SetEnvKeyPrefix()`
- `AgentConfig.TLSConfig` message — Could be imported by server/mutate protos instead of redefining
- `AgentConfig.StoreType` enum — Could be imported by inserter proto

### Established Patterns
- Proto files live in `{component}/config/v1/` directory
- Generated Go code uses package alias: `protoconf_{component}_config`
- `json_name` field options provide the CLI flag names
- Proto defaults match Go zero values; actual defaults set in Go constructor

### Integration Points
- `generate.go` — Needs new `go:generate` lines for each new proto file
- `buf.yaml` — Needs to include new proto paths for linting
- Phase 8 will wire these protos to `libprotoconf.PopulateFlagSet` in each component's `command.go`

</code_context>

<specifics>
## Specific Ideas

No specific requirements — follow the established `AgentConfig` pattern for consistency.

</specifics>

<deferred>
## Deferred Ideas

- Consolidating `KVStoreConfig` and `AgentConfig.StoreType` into a shared proto package — future concern
- Adding validation rules (protovalidate) to config protos — could be added later
- Config file loading support — Phase 8 scope
- CLI flag generation wiring — Phase 8 scope
- fmt and devserver proto configs — too trivial / composite, not worth proto-defining

</deferred>

---

*Phase: 07-proto-defined-cli-configs*
*Context gathered: 2026-03-28*
