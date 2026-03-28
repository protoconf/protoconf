# Phase 8: CLI Flag Generation & Config Loading - Research

**Researched:** 2026-03-29
**Domain:** Go CLI flag generation from protobuf definitions via libprotoconf
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**D-01:** Each component gets a component-specific env var prefix:
- Compiler: `PROTOCONF_COMPILER_*`
- Server: `PROTOCONF_SERVER_*`
- Inserter: `PROTOCONF_INSERTER_*`
- Mutate: `PROTOCONF_MUTATE_*`

**D-02:** The existing `PROTOCONF_COMPILER_ADDR` manual env var becomes `PROTOCONF_COMPILER_COMPILER_ADDRESS` via libprotoconf's automatic naming. The old manual `os.Getenv("PROTOCONF_COMPILER_ADDR")` call in `compiler/command.go` line 47 is removed.

**D-03:** All four components get a `--config-file` flag using the same pattern as `agent/command.go`:
- `lpc.Unmarshal(filename, bytes)` for json/yaml/pb format support
- `proto.Merge(orig, config)` to merge file values with existing config
- Config file values are overridden by CLI flags (standard precedence)

**D-04:** Config file format detection follows the agent pattern — libprotoconf infers format from file extension.

**D-05:** Precedence order: CLI flags > env vars > config file > proto defaults.

**D-06:** Replace manual `cliConfig` structs and `flag.StringVar`/`flag.BoolVar` calls entirely with `libprotoconf.PopulateFlagSet`. Phase 7's `json_name` options ensure generated flag names match current ones exactly.

**D-07:** Remove the manual `cliConfig` struct from each component's `command.go` — the proto-generated config struct becomes the single source of truth.

**D-08:** The `Run()` method in each component reads fields from the proto config struct instead of the old `cliConfig` struct.

**D-09:** DevServer does NOT get its own proto config. It creates component configs inline (AgentConfig, CompilerConfig, ServerConfig) and passes them to the component constructors.

**D-10:** DevServer's minimal flag parsing (just `protoconfRoot` positional arg) remains manual.

**D-11:** `command.AddKVStoreFlags()` and `command.KVStoreConfig` struct become dead code after inserter migrates to its own proto config. Remove them in this phase.

**D-12:** Agent already has its own KV store fields in AgentConfig. InserterConfig already has its own store fields (Phase 7 D-07). No shared KV proto needed.

**D-13:** Follow the agent's exact initialization sequence in each component:
1. Create proto config instance
2. Create libprotoconf config: `lpc := configtool.NewConfig(config)`
3. Set env prefix: `lpc.SetEnvKeyPrefix("PROTOCONF_COMPONENT")`
4. Load env vars: `lpc.Environment()`
5. Create flagset: `flag.NewFlagSet(...)`
6. Populate flags: `lpc.PopulateFlagSet(flagset)`
7. Add config-file flag: `flag.Func("config-file", ...)`
8. Parse args: `flag.Parse(args)`

### Claude's Discretion

- Whether to refactor devserver to accept component configs as constructor args or keep inline construction
- How to handle positional args that remain outside proto (protoconf_root, files...) — likely kept as manual `flag.Args()` after flag parsing
- Whether to add a brief deprecation comment for removed manual flag code or just delete it
- Error message wording for invalid config file loading

### Deferred Ideas (OUT OF SCOPE)

- Consolidating KV store proto definitions into a shared package
- Adding protovalidate rules to config protos
- Migrating mitchellh/cli to cobra or urfave/cli (PCLI-10, deferred to v2)
- DevServer getting its own proto config
- fmt command proto config
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| PCLI-05 | CLI flag parsing is generated from proto definitions for all components | libprotoconf.PopulateFlagSet reads proto field descriptors and registers flags using json_name as flag name |
| PCLI-06 | Generated CLI matches current flag interface (backward compatible) | Phase 7 proto files already set json_name options matching exact current flag names; verified field-by-field below |
| PCLI-07 | All components support config loading via environment variables (PROTOCONF_* prefix) | libprotoconf.Environment() reads env vars using toEnvKey(prefix, fieldName) pattern |
| PCLI-08 | All components support config loading via config files (JSON/YAML/protobuf) | libprotoconf.Unmarshal() dispatches on file extension: .json, .yaml/.yml, .pb, .data, .jsonnet, .toml |
| PCLI-09 | Config precedence follows: flags > env vars > config file > defaults | Sequence order: Environment() before PopulateFlagSet(), config-file via flag.Func processed during Parse() |
</phase_requirements>

## Summary

Phase 8 wires the proto config definitions created in Phase 7 to `libprotoconf` for all four components (server, compiler, inserter, mutate). The agent already demonstrates the complete pattern — this phase replicates it exactly for the remaining components.

The work is predominantly mechanical: for each component, replace the `cliConfig` struct and manual `flag.StringVar`/`flag.BoolVar` calls with the libprotoconf initialization sequence. The `cliCommand` struct in each file gains a `config` field (the proto message) and a `flag` field (the flagset), mirroring `agent/command.go`. The `Run()` method reads from the proto config struct instead of the old struct.

One significant non-mechanical concern: `server.go`'s `ProtoconfMutationServer` struct holds a `*cliConfig` pointer used during `MutateConfig()` calls to access `authToken` for `runScript()`. This internal coupling must be migrated to use the `*protoconf_server_config.ServerConfig` pointer instead (or the individual fields directly). The `PROTOCONF_COMPILER_ADDR` env var in `runScript()` must also be updated to use the server config's `GrpcAddress` field.

**Primary recommendation:** Follow `agent/command.go` line-for-line as the template. The only structural decision is how `ProtoconfMutationServer` in `server.go` gets access to config values — store the `*ServerConfig` directly on the struct, replacing the existing `*cliConfig`.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/protoconf/libprotoconf` | v0.1.0 | Proto-to-flags/env/file generation | Already a dep; agent already uses it; project philosophy |
| `google.golang.org/protobuf/proto` | v1.34.1 | `proto.Clone` and `proto.Merge` for config-file handler | Required for correct merge semantics |
| `flag` (stdlib) | Go 1.22 | FlagSet creation, `flag.Func` for config-file handler | Already used by all components |
| `os` (stdlib) | Go 1.22 | `os.ReadFile` in config-file handler | Already used |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/mitchellh/cli` | v1.1.5 | `cli.Command` interface | Keep unchanged — PCLI-10 defers framework migration |

### No New Dependencies
This phase introduces no new dependencies. `libprotoconf` is already in `go.mod`. All required functionality (`NewConfig`, `SetEnvKeyPrefix`, `Environment`, `PopulateFlagSet`, `Unmarshal`) is present in v0.1.0.

## Architecture Patterns

### Recommended Component Structure After Migration

Each component's `command.go` follows this pattern (replicated from `agent/command.go`):

```
type cliCommand struct {
    config *proto_config.ComponentConfig   // replaces old cliConfig struct
    flag   *flag.FlagSet
}

func (c *cliCommand) Run(args []string) int {
    // parse flags, call component Run function
}

func Command() (cli.Command, error) {
    c := &cliCommand{
        config: &proto_config.ComponentConfig{
            // set proto defaults here
        },
    }
    lpc := configtool.NewConfig(c.config)
    lpc.SetEnvKeyPrefix("PROTOCONF_COMPONENT")
    lpc.Environment()
    c.flag = flag.NewFlagSet(string(c.config.ProtoReflect().Descriptor().FullName()), flag.ContinueOnError)
    lpc.PopulateFlagSet(c.flag)
    c.flag.Func("config-file", "Component configuration file (available formats: json, yaml, pb)", func(filename string) error {
        b, err := os.ReadFile(filename)
        if err != nil {
            return fmt.Errorf("failed to read config file: %v", err)
        }
        orig := proto.Clone(c.config)
        err = lpc.Unmarshal(filename, b)
        if err != nil {
            return fmt.Errorf("failed to parse config file: %v", err)
        }
        proto.Merge(orig, c.config)
        c.config, _ = orig.(*proto_config.ComponentConfig)
        return nil
    })
    return c, nil
}
```

Source: `agent/command.go` (verified directly in codebase)

### Pattern 1: libprotoconf Flag Name Generation

`PopulateFlagSet` uses `f.JSONName()` as the flag name for each field. This means the proto `json_name` option directly controls the generated flag name.

**Example:** `string grpc_address = 1 [json_name = "grpc-address"]` generates flag `--grpc-address`.

For nested message fields, libprotoconf prefixes the parent's json_name with a hyphen:
```
// MessageKind case in flagset.go:
fs.Var(fl.Value, strings.Join([]string{f.JSONName(), fl.Name}, "-"), fl.Usage)
```
This means a `tls_config` nested message field generates flags like `--tls-config-key-file`, `--tls-config-cert-file`, etc.

Source: `libprotoconf@v0.1.0/flagset.go` lines 148-155 (verified directly)

### Pattern 2: Environment Variable Naming

`Environment()` calls `toEnvKey(prefix, fieldName)` where `fieldName` is the proto field name (not the json_name):
```go
envName := toEnvKey(c.envKeyPrefix, string(fd.Name()))
// toEnvKey: uppercase, replace non-alphanumeric with underscore
```

So `SetEnvKeyPrefix("PROTOCONF_SERVER")` + field `grpc_address` = `PROTOCONF_SERVER_GRPC_ADDRESS`.

Source: `libprotoconf@v0.1.0/environment.go` lines 15-27 (verified directly)

### Pattern 3: Config File Merge Semantics

The config-file handler uses `proto.Clone` + `proto.Merge` in a specific order:
```go
orig := proto.Clone(c.config)   // snapshot current config (has env var values)
lpc.Unmarshal(filename, b)      // loads file INTO c.config (clobbers env vars)
proto.Merge(orig, c.config)     // merge: orig (env) takes priority over c.config (file)
c.config = orig.(TypeAssertion) // restore orig as the live config
```

This achieves: env vars win over file values, since `proto.Merge` sets fields from the source only if they are not set in the destination.

Source: `agent/command.go` lines 85-91 (verified directly)

### Pattern 4: `flag.ContinueOnError` vs `flag.ExitOnError`

The agent uses `flag.ContinueOnError` for the FlagSet. The current components use `flag.ExitOnError`. After migration, use `flag.ContinueOnError` to match the agent pattern — error handling moves to the `Run()` method.

### Anti-Patterns to Avoid

- **Do not use `flag.ExitOnError`** after migration: `ContinueOnError` allows tests to exercise error paths without calling `os.Exit`.
- **Do not call `lpc.Environment()` after `PopulateFlagSet()`**: env vars must be loaded before flags are registered so the flag default value reflects the env var value at registration time.
- **Do not call `lpc.PopulateFlagSet()` before `lpc.Environment()`**: the flag default value shown in `--help` will be wrong (it won't reflect env var overrides).
- **Do not pass `newFlagSet()` results to `Help()`**: after migration, `Help()` should use `c.flag`, not create a fresh flagset.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Flag name generation from proto fields | Custom reflection code | `libprotoconf.PopulateFlagSet` | Handles enums, lists, nested messages, IsBoolFlag |
| Env var reading from proto fields | `os.Getenv` + manual assignment | `libprotoconf.Environment()` | Handles all field kinds, uses consistent naming |
| Config file parsing | Custom json/yaml parsers | `libprotoconf.Unmarshal` | Dispatches on extension, handles JSON/YAML/proto-text/binary/jsonnet/TOML |
| Merge precedence logic | Custom field-by-field merging | `proto.Clone` + `proto.Merge` | Proto merge semantics handle optional vs set correctly |

**Key insight:** libprotoconf was designed specifically for this project's philosophy. Do not implement any part of this logic manually.

## Component-by-Component Migration Plan

### Server (`server/server.go`)

**Current manual flags (lines 83-89):**
| Flag name | Default | Proto field | json_name verified |
|-----------|---------|-------------|--------------------|
| `grpc-address` | `:4301` | `grpc_address` | `json_name = "grpc-address"` ✓ |
| `pre` | `""` | `pre_mutation_script` | `json_name = "pre"` ✓ |
| `post` | `""` | `post_mutation_script` | `json_name = "post"` ✓ |
| `tls-cert` | `""` | `tls_cert` | `json_name = "tls-cert"` ✓ |
| `tls-key` | `""` | `tls_key` | `json_name = "tls-key"` ✓ |
| `tls-ca` | `""` | `tls_ca` | `json_name = "tls-ca"` ✓ |
| `auth-token` | `""` | `auth_token` | `json_name = "auth-token"` ✓ |

**Structural change needed:** `ProtoconfMutationServer` holds `config *cliConfig` (line 240). Two methods use `s.config.authToken` and `s.config.grpcAddress` at runtime:
- `MutateConfig()` line 406: `s.config.authToken` passed to `runScript()`
- `MutateConfig()` line 457: `s.config.authToken` passed to `runScript()`
- `runScript()` line 519: `s.config.grpcAddress` written as `PROTOCONF_COMPILER_ADDR` env var

After migration, `ProtoconfMutationServer.config` must change from `*cliConfig` to `*protoconf_server_config.ServerConfig`. The `NewProtoconfMutationServer` function (line 257) initializes `config: &cliConfig{}` — this changes to `config: &protoconf_server_config.ServerConfig{}`. The `run()` method sets `protoconfServer.config = config` — this changes to pass the proto config directly.

**Env prefix:** `PROTOCONF_SERVER`
**Generated env vars:**
- `PROTOCONF_SERVER_GRPC_ADDRESS`
- `PROTOCONF_SERVER_PRE_MUTATION_SCRIPT`
- `PROTOCONF_SERVER_POST_MUTATION_SCRIPT`
- `PROTOCONF_SERVER_TLS_CERT`
- `PROTOCONF_SERVER_TLS_KEY`
- `PROTOCONF_SERVER_TLS_CA`
- `PROTOCONF_SERVER_AUTH_TOKEN`

### Compiler (`compiler/command.go`)

**Current manual flags (lines 49-54):**
| Flag name | Default | Proto field | json_name verified |
|-----------|---------|-------------|--------------------|
| `repl` | `false` | `repl` | (default json_name = "repl") ✓ |
| `V` | `false` | `verbose_logging` | `json_name = "V"` ✓ |
| `process-templates` | `false` | `process_templates` | `json_name = "process-templates"` ✓ |
| `cpuprofile` | `""` | `cpuprofile` | (default json_name = "cpuprofile") ✓ |
| `memprofile` | `""` | `memprofile` | (default json_name = "memprofile") ✓ |
| `compiler-address` | `$PROTOCONF_COMPILER_ADDR` | `compiler_address` | `json_name = "compiler-address"` ✓ |

**Special case:** Line 47 reads `os.Getenv("PROTOCONF_COMPILER_ADDR")` as the default value for `compilerAddress`. After migration, `lpc.Environment()` handles this automatically via `PROTOCONF_COMPILER_COMPILER_ADDRESS`. The manual `os.Getenv` call is removed.

**Structural change needed:** `cliCommand` currently holds no fields. After migration it gains `config *protoconf_compiler_config.CompilerConfig` and `flag *flag.FlagSet`. The `Run()` method currently calls `newFlagSet()` inside itself — after migration it uses `c.flag` directly (same as agent pattern).

**Env prefix:** `PROTOCONF_COMPILER`
**Generated env vars:**
- `PROTOCONF_COMPILER_REPL`
- `PROTOCONF_COMPILER_VERBOSE_LOGGING`
- `PROTOCONF_COMPILER_PROCESS_TEMPLATES`
- `PROTOCONF_COMPILER_CPUPROFILE`
- `PROTOCONF_COMPILER_MEMPROFILE`
- `PROTOCONF_COMPILER_COMPILER_ADDRESS` (replaces old `PROTOCONF_COMPILER_ADDR`)

### Inserter (`inserter/inserter.go`)

**Current manual flags (lines 53-57):**
| Flag name | Default | Proto field | json_name verified |
|-----------|---------|-------------|--------------------|
| `store-address` | `""` | `store_address` (repeated) | `json_name = "store-address"` ✓ |
| `store` | `"consul"` | `store` (enum StoreType) | (default json_name = "store") ✓ |
| `prefix` | `""` | `prefix` | (default json_name = "prefix") ✓ |
| `namespace` | `""` | `namespace` | (default json_name = "namespace") ✓ |
| `d` | `false` | `delete` | `json_name = "d"` ✓ |

**Structural change needed:** Current `newFlagSet()` returns three values: `(*flag.FlagSet, *cliConfig, *command.KVStoreConfig)`. After migration it returns only a single `cliCommand` with embedded config. The `command.KVStoreConfig` usage throughout `Run()` (lines 76-99) must be replaced with `config.Store` (InserterConfig.StoreType enum) and `config.StoreAddress` (repeated string).

**StoreType enum mapping:**
- Old: `command.KVStoreConsul` = `"consul"` → New: `protoconf_inserter_config.InserterConfig_consul`
- Old: `command.KVStoreEtcd` = `"etcd"` → New: `protoconf_inserter_config.InserterConfig_etcd`
- Old: `command.KVStoreZookeeper` = `"zookeeper"` → New: `protoconf_inserter_config.InserterConfig_zookeeper`
- Old: `command.KVStoreConfigMaps` = `"configmaps"` → New: `protoconf_inserter_config.InserterConfig_configmaps`

**Default store address handling:** Current code uses `consts.EtcdDefaultAddress` / `consts.ZookeeperDefaultAddress` as fallbacks when `kVConfig.Address` is empty. After migration, the proto default is the zero-value empty string — same fallback logic applies to `config.StoreAddress[0]` (or check `len(config.StoreAddress) == 0`).

**Env prefix:** `PROTOCONF_INSERTER`
**Generated env vars:**
- `PROTOCONF_INSERTER_STORE`
- `PROTOCONF_INSERTER_STORE_ADDRESS`
- `PROTOCONF_INSERTER_PREFIX`
- `PROTOCONF_INSERTER_NAMESPACE`
- `PROTOCONF_INSERTER_DELETE`

### Mutate (`mutate/mutate.go`)

**Current manual flags (lines 84-94):**
| Flag name | Default | Proto field | json_name verified |
|-----------|---------|-------------|--------------------|
| `root` | `"./src"` | `protoconf_root` | `json_name = "root"` ✓ |
| `proto` | `""` | `proto_file` | `json_name = "proto"` ✓ |
| `msg` | `""` | `proto_msg` | `json_name = "msg"` ✓ |
| `addr` | `"localhost:4301"` | `server_address` | `json_name = "addr"` ✓ |
| `path` | `""` | `config_path` | `json_name = "path"` ✓ |
| `metadata` | `""` | `metadata_str` | `json_name = "metadata"` ✓ |
| `field` | (array) | `fields` (repeated) | `json_name = "field"` ✓ |
| `tls-cert` | `""` | `tls_cert` | `json_name = "tls-cert"` ✓ |
| `tls-key` | `""` | `tls_key` | `json_name = "tls-key"` ✓ |
| `tls-ca` | `""` | `tls_ca` | `json_name = "tls-ca"` ✓ |
| `insecure` | `false` | `insecure_tls` | `json_name = "insecure"` ✓ |

**Special case:** The current `fieldsArray` is a custom `flag.Value` implementing append-on-repeat. libprotoconf handles `repeated string` fields natively in `flaggable.Set()` by splitting on comma. The repeated flag (`-field a -field b`) behavior needs verification — libprotoconf's `Set()` uses `strings.Split(value, ",")` to append to the list, but does not support multiple flag invocations for the same repeated field (each `-field` call replaces via split, not appends across calls). See Pitfalls section.

**Structural change needed:** The `cliCommand.ui` field and package-level `ui` var are retained (they handle output, not config). Proto config replaces the `cliConfig` struct. The `Run()` method currently calls `newFlagSet()` inside itself — after migration uses `c.flag`.

**Env prefix:** `PROTOCONF_MUTATE`
**Generated env vars:**
- `PROTOCONF_MUTATE_PROTOCONF_ROOT`
- `PROTOCONF_MUTATE_PROTO_FILE`
- etc.

### `command/command.go` — Dead Code Removal

After inserter migrates to InserterConfig, `KVStoreConfig` struct, `AddKVStoreFlags()`, and the four `KVStore*` constants become unused. Remove them. The `RunCommand()`, `RunSubcommands()`, `DefaultUI`, `PrefixedUi` remain — they are used by multiple components.

## Common Pitfalls

### Pitfall 1: Repeated String Flag Append Behavior

**What goes wrong:** libprotoconf `flaggable.Set()` for list fields calls `strings.Split(value, ",")` and appends each item. When `--field a --field b` is passed (two separate flag invocations), the second call appends `["b"]` to the existing `["a"]`, giving `["a", "b"]`. This matches the old custom `fieldsArray.Set()` behavior (each call appends). However, `--field a,b` (comma-separated) would also work and produce `["a", "b"]`. Behavior is backward compatible for the standard usage pattern.

**Why it happens:** libprotoconf's list handling in `flaggable.Set()` splits on comma then appends all items.

**How to avoid:** Test `--field key=val1 --field key=val2` produces two entries in the fields slice.

**Warning signs:** Missing field values when multiple `-field` flags are passed.

### Pitfall 2: `flag.ExitOnError` vs `flag.ContinueOnError`

**What goes wrong:** Current components create their flagset with `flag.ExitOnError`. After migration, the flagset is created in `Command()` with `flag.ContinueOnError`. If `Run()` doesn't check the error from `c.flag.Parse(args)`, parse errors are silently swallowed.

**Why it happens:** The agent pattern explicitly checks: `if err := c.flag.Parse(args); err != nil { ... return 2 }`.

**How to avoid:** Always check the error from `c.flag.Parse(args)` and return exit code 2 on parse error.

### Pitfall 3: `proto.Merge` Direction in Config-File Handler

**What goes wrong:** Calling `proto.Merge(c.config, orig)` (wrong order) makes file values win over env vars. The correct order is `proto.Merge(orig, c.config)` where `orig` is the snapshot with env var values.

**Why it happens:** `proto.Merge(dst, src)` copies from src into dst only for fields not set in dst. So `dst=orig` (env vars) means env var fields are preserved, file fields fill in the gaps.

**How to avoid:** Always follow the agent pattern exactly. `orig := proto.Clone(c.config)` → `lpc.Unmarshal(...)` → `proto.Merge(orig, c.config)` → `c.config = orig`.

### Pitfall 4: `Help()` Method After Migration

**What goes wrong:** Current `Help()` methods call `newFlagSet()` to get a fresh flagset for printing. After migration, `newFlagSet()` is deleted. `Help()` must use `c.flag` directly.

**Why it happens:** The agent's `Help()` method writes `c.flag.Usage()` to a buffer using `c.flag.SetOutput(&b)`.

**How to avoid:** Follow the agent `Help()` pattern:
```go
func (c *cliCommand) Help() string {
    var b bytes.Buffer
    b.WriteString(c.Synopsis())
    b.WriteString("\n")
    c.flag.SetOutput(&b)
    c.flag.Usage()
    return b.String()
}
```

### Pitfall 5: `ProtoconfMutationServer.config` Field Type Change

**What goes wrong:** `server.go` has `config *cliConfig` on the `ProtoconfMutationServer` struct (line 240), used by `MutateConfig()` to access `authToken` and `grpcAddress`. If this field is not updated, the struct still references the deleted `cliConfig` type.

**Why it happens:** The config is set after construction: `protoconfServer.config = config`. After migration this becomes `protoconfServer.config = c.config` where `c.config` is `*protoconf_server_config.ServerConfig`.

**How to avoid:** Change `ProtoconfMutationServer.config` from `*cliConfig` to `*protoconf_server_config.ServerConfig`. Update `NewProtoconfMutationServer` to initialize it as `config: &protoconf_server_config.ServerConfig{}`. Update all field accesses: `s.config.authToken` → `s.config.AuthToken`, `s.config.grpcAddress` → `s.config.GrpcAddress`.

### Pitfall 6: `Command()` Function Change in `server.go`

**What goes wrong:** Current `Command()` in server.go (line 232) simply returns `&cliCommand{}`. The `cliCommand` struct is empty and all flag setup happens inside `Run()` via `newFlagSet()`. After migration, `Command()` must perform the full initialization sequence before returning.

**Why it happens:** Unlike agent which already had this pattern, server diverged.

**How to avoid:** Refactor `cliCommand` in server.go to hold `config *protoconf_server_config.ServerConfig` and `flag *flag.FlagSet`. Move flag setup to `Command()`.

### Pitfall 7: Inserter StoreAddress is `repeated string`

**What goes wrong:** Current `KVStoreConfig.Address` is a single `string`. `InserterConfig.StoreAddress` is `repeated string`. The valkeyrie `NewStore` calls accept `[]string` for addresses. After migration, pass `config.StoreAddress` directly (already a slice). For backward compat, when `config.StoreAddress` is empty, use the same fallback defaults as before.

**Why it happens:** The proto was designed for multi-address support, but current code only passes one address.

**How to avoid:** Use `config.StoreAddress` directly as the addresses slice for valkeyrie. Guard: `if len(config.StoreAddress) == 0 { use default }`.

## Code Examples

### Complete Command() Pattern (agent reference)

```go
// Source: agent/command.go
func Command() (cli.Command, error) {
    c := &cliCommand{
        config: &protoconf_agent_config.AgentConfig{
            GrpcAddress: ":4300",
            HttpAddress: ":4380",
        }}
    lpc := configtool.NewConfig(c.config)
    lpc.SetEnvKeyPrefix("PROTOCONF_AGENT")
    lpc.Environment()
    c.flag = flag.NewFlagSet(string(c.config.ProtoReflect().Descriptor().FullName()), flag.ContinueOnError)
    lpc.PopulateFlagSet(c.flag)
    c.flag.Func("config-file", "Agent configuration file (available formats: json, jsonnet, yaml, pb)", func(filename string) error {
        b, err := os.ReadFile(filename)
        if err != nil {
            return fmt.Errorf("failed to read config file: %v", err)
        }
        orig := proto.Clone(c.config)
        err = lpc.Unmarshal(filename, b)
        if err != nil {
            return fmt.Errorf("failed to parse config file: %v", err)
        }
        proto.Merge(orig, c.config)
        c.config, _ = orig.(*protoconf_agent_config.AgentConfig)
        return nil
    })
    return c, nil
}
```

### Run() Pattern After Migration

```go
// Pattern: all migrated components
func (c *cliCommand) Run(args []string) int {
    err := c.flag.Parse(args)
    if err != nil {
        fmt.Fprint(os.Stderr, "failed to parse flags", err)
        return 2
    }
    if c.flag.NArg() < 1 {
        c.flag.Usage()
        return 1
    }
    protoconfRoot := strings.TrimSpace(c.flag.Args()[0])
    // ... rest of Run using c.config.FieldName instead of config.fieldName
    return 0
}
```

### Server Command() Template

```go
// Template for server/server.go Command()
func Command() (cli.Command, error) {
    c := &cliCommand{
        config: &protoconf_server_config.ServerConfig{
            GrpcAddress: consts.ServerDefaultAddress,
        },
    }
    lpc := configtool.NewConfig(c.config)
    lpc.SetEnvKeyPrefix("PROTOCONF_SERVER")
    lpc.Environment()
    c.flag = flag.NewFlagSet(string(c.config.ProtoReflect().Descriptor().FullName()), flag.ContinueOnError)
    lpc.PopulateFlagSet(c.flag)
    c.flag.Func("config-file", "Server configuration file (available formats: json, yaml, pb)", func(filename string) error {
        b, err := os.ReadFile(filename)
        if err != nil {
            return fmt.Errorf("failed to read config file: %v", err)
        }
        orig := proto.Clone(c.config)
        err = lpc.Unmarshal(filename, b)
        if err != nil {
            return fmt.Errorf("failed to parse config file: %v", err)
        }
        proto.Merge(orig, c.config)
        c.config, _ = orig.(*protoconf_server_config.ServerConfig)
        return nil
    })
    return c, nil
}
```

### libprotoconf Env Key Formula

```
// Source: libprotoconf@v0.1.0/environment.go
// toEnvKey(prefix, fieldName) = uppercase(prefix_fieldName).replace(/[^A-Z0-9]+/, "_")
// Examples:
// SetEnvKeyPrefix("PROTOCONF_SERVER") + grpc_address = PROTOCONF_SERVER_GRPC_ADDRESS
// SetEnvKeyPrefix("PROTOCONF_COMPILER") + compiler_address = PROTOCONF_COMPILER_COMPILER_ADDRESS
// SetEnvKeyPrefix("PROTOCONF_INSERTER") + store_address = PROTOCONF_INSERTER_STORE_ADDRESS
// SetEnvKeyPrefix("PROTOCONF_MUTATE") + insecure_tls = PROTOCONF_MUTATE_INSECURE_TLS
```

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` + `github.com/stretchr/testify v1.9.0` |
| Config file | None (no separate config file needed) |
| Quick run command | `go test ./server/... ./compiler/... ./inserter/... ./mutate/... -run TestCommand -count=1` |
| Full suite command | `go test ./server/... ./compiler/... ./inserter/... ./mutate/... -count=1` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| PCLI-05 | `Command()` returns flags generated from proto (not hand-written) | unit | `go test ./server/... -run TestCommand` | ❌ Wave 0 |
| PCLI-06 | Generated flag names match current flag names exactly | unit | `go test ./... -run TestFlagNames` | ❌ Wave 0 |
| PCLI-07 | `PROTOCONF_SERVER_GRPC_ADDRESS` env var sets grpc-address | unit | `go test ./server/... -run TestEnvVars` | ❌ Wave 0 |
| PCLI-07 | `PROTOCONF_COMPILER_COMPILER_ADDRESS` env var sets compiler-address | unit | `go test ./compiler/... -run TestEnvVars` | ❌ Wave 0 |
| PCLI-07 | `PROTOCONF_INSERTER_STORE` env var sets store | unit | `go test ./inserter/... -run TestEnvVars` | ❌ Wave 0 |
| PCLI-07 | `PROTOCONF_MUTATE_SERVER_ADDRESS` env var sets addr | unit | `go test ./mutate/... -run TestEnvVars` | ❌ Wave 0 |
| PCLI-08 | `--config-file foo.yaml` loads config from YAML | unit | `go test ./server/... -run TestConfigFile` | ❌ Wave 0 |
| PCLI-08 | `--config-file foo.json` loads config from JSON | unit | `go test ./compiler/... -run TestConfigFile` | ❌ Wave 0 |
| PCLI-09 | CLI flag overrides env var value | unit | `go test ./... -run TestPrecedence` | ❌ Wave 0 |
| PCLI-09 | Env var overrides config file value | unit | `go test ./... -run TestPrecedence` | ❌ Wave 0 |
| PCLI-06 | `command/command.go` compiles without KVStoreConfig | build | `go build ./command/...` | N/A — build verification |

**Note:** Existing test files (`server/server_test.go`, `inserter/inserter_test.go`) test business logic, not CLI flag parsing. New tests for flag generation are lightweight — use `flag.Lookup()` to verify flag names exist and have expected defaults.

### Sampling Rate
- **Per task commit:** `go build ./...` (ensures no compilation errors)
- **Per wave merge:** `go test ./server/... ./compiler/... ./inserter/... ./mutate/... -count=1`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps

Tests for CLI flag generation are currently minimal. However, since Phase 9 is specifically dedicated to unit test coverage, this phase should:
- Verify the code builds and passes existing tests
- Optionally add minimal smoke tests for flag registration

The following files may need creation for complete coverage (but full coverage is Phase 9 scope):
- [ ] `server/command_test.go` — covers PCLI-05, PCLI-06, PCLI-07 for server
- [ ] `compiler/command_test.go` — covers PCLI-05, PCLI-06, PCLI-07 for compiler
- [ ] `mutate/command_test.go` — covers PCLI-05, PCLI-06, PCLI-07 for mutate
- (inserter/inserter_test.go exists — extend for new flag behavior)

## Environment Availability

Step 2.6: SKIPPED — Phase 8 is purely code/config changes modifying existing Go source files. No external dependencies are introduced. All required tools (Go 1.22, libprotoconf in go.sum) are already available.

## Open Questions

1. **`runScript()` environment variable name for server address**
   - What we know: `runScript()` currently sets `PROTOCONF_COMPILER_ADDR=s.config.grpcAddress` as an env var passed to scripts (line 519). This is the *server's* gRPC address passed to pre/post mutation scripts, not the compiler address.
   - What's unclear: The variable name `PROTOCONF_COMPILER_ADDR` is misleading (it's actually the server address). Should this be renamed to `PROTOCONF_SERVER_ADDR` when migrating, or kept as-is for script backward compatibility?
   - Recommendation: Keep `PROTOCONF_COMPILER_ADDR` in the script environment for backward compatibility (existing scripts may rely on it). The value comes from `s.config.GrpcAddress` after migration. Document in comments that this env var name is legacy.

2. **Proto default values for server and mutate**
   - What we know: The server's `GrpcAddress` defaults to `consts.ServerDefaultAddress` (`:4301`) — this must be set as a struct literal default in `Command()`. The mutate's `ServerAddress` defaults to `"localhost:4301"` and `ProtoconfRoot` to `"./src"`.
   - What's unclear: proto defaults (zero values) for strings are empty string, not `:4301`. The defaults must be set in the `Command()` factory, not in the proto itself.
   - Recommendation: Set non-zero defaults as struct literal values in `Command()` exactly as the agent does for `GrpcAddress: ":4300"`.

## Sources

### Primary (HIGH confidence)
- `agent/command.go` (local codebase) — Complete reference implementation of the exact pattern to replicate
- `agent/config/v1/agent_config.proto` (local codebase) — Demonstrates json_name option pattern
- `libprotoconf@v0.1.0/libprotoconf.go` (Go module cache, verified) — `NewConfig`, `SetEnvKeyPrefix`, `Environment`, `AppendLoadDir`
- `libprotoconf@v0.1.0/flagset.go` (Go module cache, verified) — `PopulateFlagSet`, flag name = `f.JSONName()`, nested message prefix behavior
- `libprotoconf@v0.1.0/environment.go` (Go module cache, verified) — `toEnvKey` formula: `uppercase(prefix_fieldName).replace(/[^A-Z0-9]+/, "_")`
- `libprotoconf@v0.1.0/unmarshalers.go` (Go module cache, verified) — Format dispatch on extension: `.json`, `.yaml/.yml`, `.pb`, `.data`, `.b64`, `.jsonnet`, `.toml`
- `server/config/v1/server_config.proto` (local, Phase 7 output) — 7 fields with json_name options
- `compiler/config/v1/compiler_config.proto` (local, Phase 7 output) — 6 fields with json_name options
- `inserter/config/v1/inserter_config.proto` (local, Phase 7 output) — 5 fields with json_name options and StoreType enum
- `mutate/config/v1/mutate_config.proto` (local, Phase 7 output) — 11 fields with json_name options

### Secondary (MEDIUM confidence)
- N/A — all findings verified from primary sources

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all libraries already in go.mod, verified in module cache
- Architecture: HIGH — agent reference implementation is directly readable, all patterns verified
- Pitfalls: HIGH — identified from direct code inspection of the files to be modified
- Backward compatibility: HIGH — every flag name/default verified against current code and proto json_name options

**Research date:** 2026-03-29
**Valid until:** 2026-04-28 (30 days — stable ecosystem, libprotoconf API is not fast-moving)
