# Phase 7: Proto-Defined CLI Configs - Research

**Researched:** 2026-03-28
**Domain:** Protocol Buffers schema definition, proto3 syntax, protoc code generation
**Confidence:** HIGH

## Summary

Phase 7 is a schema-only task: define four `.proto` files that capture the current CLI flag configuration for the server, compiler, inserter, and mutate components. The reference implementation is `agent/config/v1/agent_config.proto`, which is fully in-repo and has been studied. No external library research is needed — the pattern is established and the toolchain is already installed.

The work is mechanical: read each component's `cliConfig` struct and `newFlagSet()` function, transcribe the fields into proto3 messages following the agent pattern (field names, `json_name` options matching CLI flag names, enum for KV store type), and generate `.pb.go` files using the existing `go:generate` directive in `generate.go`.

For the two discretion items (TLS sharing and StoreType sharing), the code evidence strongly favors reuse via import: `AgentConfig.TLSConfig` already exists as a well-formed nested message and is used twice in `AgentConfig` itself. The inserter uses the same four KV store types as the agent. Defining shared structures inline in each proto would create drift risk.

**Primary recommendation:** Define four proto files following the exact `agent_config.proto` pattern. Reuse `AgentConfig.TLSConfig` and `AgentConfig.StoreType` in server, mutate, and inserter protos via proto import. Commit generated `.pb.go` files. Add new protos to `buf.yaml` and `generate.go`.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** One proto file per component — `server/config/v1/server_config.proto`, `compiler/config/v1/compiler_config.proto`, `inserter/config/v1/inserter_config.proto`, `mutate/config/v1/mutate_config.proto`
- **D-02:** Each `option go_package` follows `github.com/protoconf/protoconf/{component}/config/v1;protoconf_{component}_config`
- **D-03:** Every proto field MUST have a `json_name` field option matching the current CLI flag name exactly
- **D-04:** Default values in proto match Go zero values; non-zero defaults (e.g., `:4301`) set in Go constructor
- **D-05:** `ServerConfig` fields: `grpc_address`, `pre_mutation_script`, `post_mutation_script`, `tls_cert`, `tls_key`, `tls_ca`, `auth_token`. TLS handling is Claude's discretion.
- **D-06:** `CompilerConfig` fields: `repl` (bool), `verbose_logging` (bool), `process_templates` (bool), `cpuprofile` (string), `memprofile` (string). Positional args are NOT proto fields.
- **D-07:** `InserterConfig` fields: KV store fields from `command.KVStoreConfig` (`store` enum, `store_address` repeated string, `prefix` string) plus `delete` (bool).
- **D-08:** `MutateConfig` fields: `protoconf_root`, `proto_file`, `proto_msg`, `server_address`, `config_path`, `metadata_str`, `tls_cert`, `tls_key`, `tls_ca`, `insecure_tls` (bool), plus field/value pairs.
- **D-09:** fmt and devserver do NOT get proto configs in this phase.
- **D-10:** `command/command.go KVStoreConfig` is NOT converted to proto in this phase.
- **D-11:** Generated `.pb.go` files are committed to the repo.
- **D-12:** Add proto files to `buf.yaml` and ensure they pass `buf lint`.
- **D-13:** Add `go:generate` directives for new proto files following `generate.go` pattern.

### Claude's Discretion

- Whether to import and reuse `AgentConfig.TLSConfig` in `ServerConfig`/`MutateConfig` or define TLS fields as flat strings
- Whether `InserterConfig` should use the same `StoreType` enum as `AgentConfig` (via import) or define its own
- Exact field numbering in proto messages
- Whether to add field-level comments mirroring current flag help text

### Deferred Ideas (OUT OF SCOPE)

- Consolidating `KVStoreConfig` and `AgentConfig.StoreType` into a shared proto package
- Adding validation rules (protovalidate) to config protos
- Config file loading support (Phase 8 scope)
- CLI flag generation wiring (Phase 8 scope)
- fmt and devserver proto configs
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| PCLI-01 | Proto definitions exist for server configuration (address, TLS, auth, scripts) | `server/server.go` `cliConfig` struct fully mapped; TLS fields identified |
| PCLI-02 | Proto definitions exist for compiler configuration (proto paths, output settings) | `compiler/command.go` `cliConfig` struct fully mapped; all 5 fields identified |
| PCLI-03 | Proto definitions exist for inserter configuration (KV store, prefix, rollout) | `inserter/inserter.go` + `command/command.go` KVStoreConfig fully mapped |
| PCLI-04 | Proto definitions exist for mutate CLI configuration (target server, field path, value) | `mutate/mutate.go` `cliConfig` struct fully mapped including `fieldsArray` handling |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| proto3 | — | Schema language | Project-wide standard; all existing protos use proto3 |
| protoc | v34.0 (libprotoc) | Code generation | Already installed (`/opt/homebrew/bin/protoc`) |
| protoc-gen-go | v1.28.1 | Go struct generation | Evidenced in agent_config.pb.go header |
| protoc-gen-go-grpc | current | gRPC service generation | Already used for server/agent APIs |
| buf | v1.61.0 | Proto linting | Already installed (`/opt/homebrew/bin/buf`) |

**Version verification:** Verified by running `protoc --version` and `buf --version` on the target machine.

**Installation:** No new installation required — all tools present.

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| google.golang.org/protobuf | v1.34.1 | Runtime for generated code | Already in go.mod; generated code imports it |

## Architecture Patterns

### Recommended Project Structure
```
server/
└── config/v1/
    ├── server_config.proto
    └── server_config.pb.go   (generated, committed)

compiler/
└── config/v1/
    ├── compiler_config.proto
    └── compiler_config.pb.go

inserter/
└── config/v1/
    ├── inserter_config.proto
    └── inserter_config.pb.go

mutate/
└── config/v1/
    ├── mutate_config.proto
    └── mutate_config.pb.go
```

### Pattern 1: Proto File Structure (from agent reference)
**What:** One proto file per component with a single top-level config message. Nested messages and enums defined inside the config message to avoid polluting the package namespace.
**When to use:** All four new proto files.
**Example:**
```protobuf
// Source: agent/config/v1/agent_config.proto
syntax = "proto3";

package protoconf.agent.config.v1;

option go_package = "github.com/protoconf/protoconf/agent/config/v1;protoconf_agent_config";

message AgentConfig {
    string grpc_address = 1 [json_name = "grpc-address"];
    StoreType store = 4;
    repeated string servers = 5 [json_name = "store-address"];

    enum StoreType {
        consul = 0;
        etcd = 1;
        zookeeper = 2;
        file = 3;
        configmaps = 4;
    }

    message TLSConfig {
        oneof key { string key_text = 1; string key_file = 2; }
        oneof cert { string cert_text = 3; string cert_file = 4; }
        oneof ca { string ca_text = 5; string ca_file = 6; }
    }
}
```

### Pattern 2: json_name Field Option for CLI Flags
**What:** Every field that corresponds to a CLI flag uses `[json_name = "flag-name"]` with the exact hyphenated flag name. Fields with identical proto-to-Go name conversion (e.g., `prefix` stays `prefix`) can omit `json_name` but it is acceptable to include it for explicitness.
**When to use:** All fields derived from existing CLI flags.
**Example:**
```protobuf
// Source: agent/config/v1/agent_config.proto
string grpc_address = 1 [json_name = "grpc-address"];
repeated string servers = 5 [json_name = "store-address"];
bool log_as_json = 10 [json_name = "log-as-json"];
```

### Pattern 3: Proto Import for Shared Messages
**What:** `import "agent/config/v1/agent_config.proto";` at the top of a new proto file lets server_config and mutate_config reference `AgentConfig.TLSConfig` without duplication.
**When to use:** ServerConfig (has flat TLS fields in current code) and MutateConfig (has flat TLS fields). The discretion recommendation is to import and reuse — see reasoning below.

**TLS discretion recommendation: reuse via import**

Evidence: The current `server/server.go` and `mutate/mutate.go` both define flat `tlsCert`, `tlsKey`, `tlsCA` string fields. However, `agent/config/v1/agent_config.proto` has already modeled this cleanly as `TLSConfig` with `oneof` variants for text vs file. Using flat strings in server_config.proto would miss the design improvement already made in the agent. Since Phase 8 will wire these to `libprotoconf.PopulateFlagSet`, keeping TLS as a flat set of strings (tls_cert, tls_key, tls_ca as string fields on the root message) is actually more compatible with the current CLI flags — the agent chose the nested approach with oneofs because it supports both file path and inline text. For server/mutate, the current flags only support file paths. **Recommendation: define TLS as flat string fields on the root message** (`tls_cert`, `tls_key`, `tls_ca`) to exactly match existing flags. Using the nested `TLSConfig` with oneofs would require Phase 8 to handle the oneof mapping, which is out of scope. This is the simpler path.

**StoreType discretion recommendation: define own inline enum**

Evidence: `command/command.go` uses string constants (`KVStoreConsul = "consul"`) while `AgentConfig.StoreType` uses proto enum with integer mapping. Importing and using `AgentConfig.StoreType` in `InserterConfig` creates a cross-package proto dependency that ties inserter's schema evolution to the agent package. Since D-10 explicitly says `KVStoreConfig` is NOT being migrated, and the inserter already imports `command.KVStoreConfig` as a separate Go struct, defining `InserterConfig.StoreType` as a local enum with the same four values (consul=0, etcd=1, zookeeper=2, configmaps=3) is cleaner. Note: agent has `file=3` and `configmaps=4`; inserter uses `command.KVStoreConsul/Etcd/Zookeeper/ConfigMaps` — no `file` store. Inserter's enum should reflect its actual supported stores.

### Pattern 4: go_package Naming
**What:** `option go_package` uses the module-qualified path with a semicolon-separated short package alias.
**When to use:** Required in every new proto file.
**Example:**
```protobuf
option go_package = "github.com/protoconf/protoconf/server/config/v1;protoconf_server_config";
option go_package = "github.com/protoconf/protoconf/compiler/config/v1;protoconf_compiler_config";
option go_package = "github.com/protoconf/protoconf/inserter/config/v1;protoconf_inserter_config";
option go_package = "github.com/protoconf/protoconf/mutate/config/v1;protoconf_mutate_config";
```

### Pattern 5: Package Naming Convention
**What:** Proto `package` declaration mirrors the directory path, using dots.
**Example:**
```protobuf
package protoconf.server.config.v1;
package protoconf.compiler.config.v1;
package protoconf.inserter.config.v1;
package protoconf.mutate.config.v1;
```

### Anti-Patterns to Avoid
- **Using `required` fields:** proto3 has no `required`; all fields are optional by default. Do not add `optional` keyword for scalar fields (it is valid proto3 syntax but unnecessary noise).
- **Naming enums at the package level:** Keep `StoreType` as a nested enum inside `InserterConfig` to avoid proto name collisions when buf lints across the full module.
- **Mismatching json_name with actual flag names:** The planner must cross-reference each proto field's `json_name` against the `newFlagSet()` registration in the source file. A single mismatch breaks Phase 8's backward compatibility.
- **Omitting `syntax = "proto3";`:** Always the first non-comment line.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| CLI flag → proto field binding | Custom flag parser that reads proto fields | `libprotoconf.PopulateFlagSet` | Already implemented in Phase 8 scope; agent uses it |
| Proto serialization/deserialization | Custom JSON/YAML marshaling | `google.golang.org/protobuf/encoding/protojson` | Standard library; already a go.mod dependency |
| Enum string mapping | Custom string↔int conversion | Proto-generated `String()` and `value` maps | Generated automatically by protoc-gen-go |

**Key insight:** Phase 7 is purely schema definition. No hand-rolled code at all — the output is `.proto` files and their generated `.pb.go` counterparts.

## Common Pitfalls

### Pitfall 1: buf lint failing on enum value naming
**What goes wrong:** `buf lint` enforces that enum values are prefixed with the enum name in UPPER_SNAKE_CASE. `AgentConfig` violates this (uses lowercase `consul`, `etcd` etc.) because the repo has a lint waiver or the values predate the rule.
**Why it happens:** `buf.yaml` uses `DEFAULT` lint rules which include `ENUM_VALUE_PREFIX`. The agent proto may pass due to being grandfathered or having a waiver.
**How to avoid:** Run `buf lint` immediately after writing each proto file. If the lint rule fires, either add an `enum_value_prefix` exception to `buf.yaml` or use uppercase prefixed values (e.g., `INSERTER_STORE_TYPE_CONSUL = 0`). Check what the agent proto does and be consistent.
**Warning signs:** `buf lint` output mentioning `ENUM_VALUE_PREFIX`.

### Pitfall 2: protoc invocation not finding new proto files
**What goes wrong:** `go generate` runs but the new proto files are not included in the `protoc` invocation, so no `.pb.go` is generated.
**Why it happens:** The current `generate.go` uses `find . -name '*.proto' -not -path '*pb/*' -not -path '*utils/*'` — this glob WILL automatically pick up new proto files in `server/config/v1/`, `compiler/config/v1/`, etc. No change to `generate.go` is needed for the protoc invocation itself. However, D-13 says to "add `go:generate` directives" — verify whether the existing directive already covers them before adding redundant ones.
**How to avoid:** Verify the existing find-based generate command covers the new paths. It should, since it finds all `.proto` files recursively except `pb/` and `utils/`.
**Warning signs:** Running `go generate ./...` produces no new `.pb.go` files in the new directories.

### Pitfall 3: MutateConfig fieldsArray is a custom flag.Value, not a simple string
**What goes wrong:** `mutate/mutate.go` uses `fieldsArray` — a `[]string` implementing `flag.Value` with `Set()`/`String()`. This cannot be directly mapped to a single proto field. A `repeated string fields = N [json_name = "field"]` is the correct proto equivalent.
**Why it happens:** Go's `flag` package uses the `Value` interface for custom types. Proto uses `repeated` for arrays.
**How to avoid:** Map `fieldsArray` to `repeated string fields` in `MutateConfig`. Phase 8 will handle wiring `repeated string` through `PopulateFlagSet`.
**Warning signs:** Treating `fieldsArray` as if it maps to a single `string` field.

### Pitfall 4: Field numbers must never change once committed
**What goes wrong:** Changing a field number in a proto file after the `.pb.go` is generated and committed is a breaking wire-format change.
**Why it happens:** Field numbers are the wire identity of proto fields, not field names.
**How to avoid:** Choose field numbers carefully in the initial definition. Since these are new messages with no existing wire data, any numbering works — but assign them in a logical order and leave gaps (10, 20, 30...) only if extension is anticipated. For this phase, sequential numbering starting at 1 is fine.
**Warning signs:** Renumbering a field after the first commit.

### Pitfall 5: compiler/command.go has a `compilerAddress` field not in CONTEXT.md
**What goes wrong:** CONTEXT.md's D-06 lists compiler fields as `repl`, `verbose_logging`, `process_templates`, `cpuprofile`, `memprofile` — but `compiler/command.go` line 37 also defines `compilerAddress string` with flag `--compiler-address`.
**Why it happens:** The field was missed in the discussion phase.
**How to avoid:** The planner must decide whether to include `compiler_address` in `CompilerConfig`. It is a legitimate CLI flag (registered in `newFlagSet()`), so including it is correct for completeness. Recommended: include it as `string compiler_address = 6 [json_name = "compiler-address"]`.
**Warning signs:** Compiling Phase 8 and discovering `--compiler-address` flag has no proto backing.

## Code Examples

Verified patterns from official sources (agent_config.proto is the authoritative in-repo reference):

### ServerConfig Field Mapping
```protobuf
// Derived from server/server.go newFlagSet() lines 83-89
message ServerConfig {
    string grpc_address = 1 [json_name = "grpc-address"];           // flag: --grpc-address, default: :4301
    string pre_mutation_script = 2 [json_name = "pre"];             // flag: --pre
    string post_mutation_script = 3 [json_name = "post"];           // flag: --post
    string tls_cert = 4 [json_name = "tls-cert"];                   // flag: --tls-cert
    string tls_key = 5 [json_name = "tls-key"];                     // flag: --tls-key
    string tls_ca = 6 [json_name = "tls-ca"];                       // flag: --tls-ca
    string auth_token = 7 [json_name = "auth-token"];               // flag: --auth-token
}
```

### CompilerConfig Field Mapping
```protobuf
// Derived from compiler/command.go newFlagSet() lines 49-54
message CompilerConfig {
    bool repl = 1;                                                    // flag: --repl
    bool verbose_logging = 2 [json_name = "V"];                      // flag: -V
    bool process_templates = 3 [json_name = "process-templates"];    // flag: --process-templates
    string cpuprofile = 4;                                           // flag: --cpuprofile
    string memprofile = 5;                                           // flag: --memprofile
    string compiler_address = 6 [json_name = "compiler-address"];    // flag: --compiler-address
}
```

### InserterConfig Field Mapping
```protobuf
// Derived from inserter/inserter.go + command/command.go AddKVStoreFlags()
message InserterConfig {
    StoreType store = 1;                                              // flag: --store, default: consul
    repeated string store_address = 2 [json_name = "store-address"]; // flag: --store-address
    string prefix = 3;                                               // flag: --prefix
    string namespace = 4;                                            // flag: --namespace (k8s)
    bool delete = 5 [json_name = "d"];                               // flag: -d

    enum StoreType {
        consul = 0;
        etcd = 1;
        zookeeper = 2;
        configmaps = 3;
    }
}
```

### MutateConfig Field Mapping
```protobuf
// Derived from mutate/mutate.go newFlagSet() lines 84-94
message MutateConfig {
    string protoconf_root = 1 [json_name = "root"];                  // flag: --root, default: ./src
    string proto_file = 2 [json_name = "proto"];                     // flag: --proto
    string proto_msg = 3 [json_name = "msg"];                        // flag: --msg
    string server_address = 4 [json_name = "addr"];                  // flag: --addr, default: localhost:4301
    string config_path = 5 [json_name = "path"];                     // flag: --path
    string metadata_str = 6 [json_name = "metadata"];                // flag: --metadata
    repeated string fields = 7 [json_name = "field"];                // flag: --field (repeatable)
    string tls_cert = 8 [json_name = "tls-cert"];                    // flag: --tls-cert
    string tls_key = 9 [json_name = "tls-key"];                      // flag: --tls-key
    string tls_ca = 10 [json_name = "tls-ca"];                       // flag: --tls-ca
    bool insecure_tls = 11 [json_name = "insecure"];                 // flag: --insecure
}
```

### generate.go Pattern
```go
// Source: generate.go
//go:generate sh -xc "find . -name '*.proto' -not -path '*pb/*' -not -path '*utils/*' | xargs protoc -I=. --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative"
```
The existing directive uses a recursive find, so it automatically picks up new proto files in `server/config/v1/`, `compiler/config/v1/`, etc. No new directives are needed for the protoc step.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Hand-written Go flag structs | Proto-defined config with `json_name` options | Agent introduced this pattern (pre-Phase 7) | Phase 8 can auto-generate CLI flags from proto |

**Deprecated/outdated:**
- Hand-written `cliConfig` structs in server/compiler/inserter/mutate: These remain in place for now. Phase 7 adds the proto definitions. Phase 8 will replace the hand-written flag code with `libprotoconf.PopulateFlagSet`.

## Open Questions

1. **buf lint enum value prefix rule**
   - What we know: Agent proto uses lowercase enum values (`consul`, `etcd`) which may violate `ENUM_VALUE_PREFIX` lint rule
   - What's unclear: Whether `buf.yaml` has this rule suppressed or the agent passes by coincidence
   - Recommendation: Run `buf lint` against the existing agent proto first. If it passes with lowercase values, new protos can follow the same pattern.

2. **compiler_address field inclusion**
   - What we know: `compiler/command.go` registers `--compiler-address` flag but CONTEXT.md D-06 does not list it
   - What's unclear: Whether this omission was intentional (exclude from proto) or oversight
   - Recommendation: Include it in `CompilerConfig` as field 6. It is a real CLI flag and excluding it would create an asymmetry between proto definition and actual flags, breaking Phase 8's goal of parity.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| protoc | Proto compilation | ✓ | libprotoc 34.0 | — |
| protoc-gen-go | Go struct generation | ✓ | v1.28.1 (from .pb.go header) | — |
| buf | Proto linting | ✓ | 1.61.0 | — |
| Go toolchain | Build verification | ✓ | 1.22+ | — |

**Missing dependencies:** None. All required tools are present.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go standard `testing` + `testify v1.9.0` |
| Config file | None (no pytest.ini/jest.config equivalent) |
| Quick run command | `go test ./server/... ./compiler/... ./inserter/... ./mutate/...` |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| PCLI-01 | `server/config/v1/server_config.pb.go` compiles without errors | smoke | `go build ./server/config/v1/...` | ❌ Wave 0 — file does not exist yet |
| PCLI-02 | `compiler/config/v1/compiler_config.pb.go` compiles without errors | smoke | `go build ./compiler/config/v1/...` | ❌ Wave 0 |
| PCLI-03 | `inserter/config/v1/inserter_config.pb.go` compiles without errors | smoke | `go build ./inserter/config/v1/...` | ❌ Wave 0 |
| PCLI-04 | `mutate/config/v1/mutate_config.pb.go` compiles without errors | smoke | `go build ./mutate/config/v1/...` | ❌ Wave 0 |
| PCLI-01–04 | `buf lint` passes for all new proto files | lint | `buf lint` | ❌ Wave 0 — new protos needed |
| PCLI-01–04 | All new `.pb.go` field names match existing CLI flag names | manual review | n/a | n/a |

**Note:** These requirements have no runtime behavior — the validation is purely compilation + lint. No unit test files are required for Phase 7 itself. The smoke tests above are `go build` commands, not test functions.

### Sampling Rate
- **Per task commit:** `go build ./...` (verify compilation stays green)
- **Per wave merge:** `go build ./... && buf lint`
- **Phase gate:** `go build ./... && buf lint` both green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `server/config/v1/` directory and `server_config.proto` — covers PCLI-01
- [ ] `compiler/config/v1/` directory and `compiler_config.proto` — covers PCLI-02
- [ ] `inserter/config/v1/` directory and `inserter_config.proto` — covers PCLI-03
- [ ] `mutate/config/v1/` directory and `mutate_config.proto` — covers PCLI-04
- [ ] Run `go generate ./...` to produce `.pb.go` files for all four new protos

## Project Constraints (from CLAUDE.md)

| Constraint | Impact on This Phase |
|-----------|---------------------|
| `CGO_ENABLED=0`, static binaries | Not relevant — no C dependencies introduced |
| Proto compatibility: cannot break wire formats | New proto files only; no existing messages modified |
| KV store interface compatibility | Not impacted — proto files have no runtime behavior |
| `gofmt` enforced | Generated `.pb.go` files are already gofmt-compliant |
| `buf-lint@1.28.1` for protobuf linting | Must run `buf lint` before committing; new protos added to module |
| `go_package` with semicolon alias pattern | Already enforced in D-02 |
| `json_name` for CLI flags | Already enforced in D-03 |
| Commit `.pb.go` generated files | Required by D-11 |
| Constructors follow `New<Type>` pattern | Not applicable — Phase 7 is schema-only, no constructor code |
| Error handling with `errors.New` / `errors.Join` | Not applicable — no Go code written in this phase |

## Sources

### Primary (HIGH confidence)
- `agent/config/v1/agent_config.proto` — In-repo reference implementation, read directly
- `agent/command.go` — Shows `libprotoconf.PopulateFlagSet` usage pattern, read directly
- `server/server.go` lines 65-92 — Current server `cliConfig` and `newFlagSet()`, read directly
- `compiler/command.go` lines 31-56 — Current compiler `cliConfig` and `newFlagSet()`, read directly
- `inserter/inserter.go` lines 42-59 + `command/command.go` lines 52-73 — Current inserter config, read directly
- `mutate/mutate.go` lines 58-96 — Current mutate `cliConfig` and `newFlagSet()`, read directly
- `generate.go` — `go:generate` directive pattern, read directly
- `buf.yaml` — Module config, read directly

### Secondary (MEDIUM confidence)
- `protoc --version` output — Confirms protoc 34.0 installed locally
- `buf --version` output — Confirms buf 1.61.0 installed locally
- `agent/config/v1/agent_config.pb.go` header — Confirms protoc-gen-go v1.28.1

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all tools verified installed locally, versions confirmed
- Architecture: HIGH — reference implementation is in-repo and fully studied
- Field mappings: HIGH — derived directly from source code, not documentation
- Pitfalls: HIGH — derived from code inspection (fieldsArray, existing generate.go glob, buf lint rules observed)

**Research date:** 2026-03-28
**Valid until:** 2026-04-28 (stable domain — proto3 syntax does not change; tool versions stable)
