# Phase 8: CLI Flag Generation & Config Loading - Context

**Gathered:** 2026-03-29
**Status:** Ready for planning

<domain>
## Phase Boundary

Wire Phase 7's proto config definitions to `libprotoconf.PopulateFlagSet` for all components (server, compiler, inserter, mutate). Add environment variable support via `libprotoconf.SetEnvKeyPrefix` and config file loading via `--config-file` flag. Remove manual `cliConfig` structs and hand-written flag parsing. DevServer inherits proto configs by composing component configs inline.

</domain>

<decisions>
## Implementation Decisions

### Env Var Prefix Naming — PCLI-07
- **D-01 [folded]:** [satisfied by 08-01/08-02] Each component gets a component-specific env var prefix following the agent pattern:
  - Compiler: `PROTOCONF_COMPILER_*`
  - Server: `PROTOCONF_SERVER_*`
  - Inserter: `PROTOCONF_INSERTER_*`
  - Mutate: `PROTOCONF_MUTATE_*`
- **D-02:** The existing `PROTOCONF_COMPILER_ADDR` manual env var becomes `PROTOCONF_COMPILER_COMPILER_ADDRESS` via libprotoconf's automatic naming. The old manual `os.Getenv("PROTOCONF_COMPILER_ADDR")` call is removed.

### Config File Loading — PCLI-08
- **D-03:** All five components (the four migrated plus the pre-existing agent) get a `--config-file` flag:
  - `lpc.Unmarshal(filename, bytes)` for json/yaml/pb format support
  - Layering is performed by the shared `command.LayerConfigFile` helper, which mutates the live config message in place and never reassigns the component's `config` field
  - Config file values are overridden by env vars and by CLI flags

  **Superseded (corrected 2026-09-01 during 08 gap closure):** this decision originally specified
  `proto.Merge(orig, config)` "to merge file values with existing config", copied from
  `agent/command.go`. That mechanism is wrong in two independent ways, both confirmed against
  `libprotoconf@v0.1.0` source:
  1. It merges file values *on top of* env values, producing `flags > config file > env vars`
     — the inversion of PCLI-09 that verification caught as the phase's only gap.
  2. `flaggable.Set` writes through `f.cfg.msg`, the message handed to `NewConfig`, resolved at
     call time. Reassigning `config` to the merged clone unbinds the flag set, so any flag parsed
     *after* `-config-file` was silently discarded.

  See `08-03-PLAN.md` `<why_not_the_one_line_reversal>` for the full derivation.
- **D-04 [folded]:** [satisfied by 08-01/08-02] Config file format detection follows the agent pattern — libprotoconf infers format from file extension.

### Config Precedence — PCLI-09
- **D-05:** Precedence order: CLI flags > env vars > config file > proto defaults, as stated by PCLI-09
  and ROADMAP Success Criterion #5. `lpc.Environment()` is called before `lpc.PopulateFlagSet()`, so
  flags override env vars. The env-over-file half is delivered by `command.LayerConfigFile` (see D-03),
  which takes a `base` snapshot *before* `lpc.Environment()` in order to tell an operator-supplied env
  value apart from a built-in factory default.

  **Correction (2026-09-01, 08 gap closure):** the original wording attributed this ordering to the
  agent's `proto.Merge()` pattern. That attribution was false — that pattern yields
  `flags > config file > env vars`. The precedence goal stated here is unchanged and remains the target;
  only the claimed mechanism was wrong. Implemented by `08-03-PLAN.md` / `08-04-PLAN.md`.

  **Known limitation (tested, not a defect):** proto3 implicit presence means a zero-valued enum set
  via an env var is indistinguishable from unset, so `PROTOCONF_INSERTER_STORE=consul` (where
  `consul = 0`) cannot override a config file. The command-line flag `-store consul` still works as the
  escape hatch. Pinned by `zero_value_enum_from_env_is_indistinguishable_from_unset` and
  `flag_can_still_select_the_zero_value_enum`.

### Backward Compatibility Migration — PCLI-05, PCLI-06
- **D-06:** Replace manual `cliConfig` structs and `flag.StringVar`/`flag.BoolVar` calls entirely with `libprotoconf.PopulateFlagSet`. Phase 7's `json_name` options (D-03 from Phase 7) ensure generated flag names match current ones exactly.
- **D-07:** Remove the manual `cliConfig` struct from each component's `command.go` — the proto-generated config struct becomes the single source of truth.
- **D-08:** The `Run()` method in each component reads fields from the proto config struct instead of the old `cliConfig` struct.

### DevServer Composition
> [informational] DevServer was not in the plan set for this phase — neither the executed plans
> (08-01, 08-02) nor the gap-closure plans (08-03, 08-04) touch it. These two entries record the
> intended shape for a future phase; they are not tracked deliverables of Phase 8.

- **D-09 [informational]:** DevServer does NOT get its own proto config. It creates component configs inline (AgentConfig, CompilerConfig, ServerConfig) and passes them to the component constructors, matching the current pattern where devserver constructs services directly.
- **D-10 [informational]:** DevServer's minimal flag parsing (just `protoconfRoot` positional arg) remains manual — it delegates config to composed components.

### KVStoreConfig Transition
- **D-11:** `command.AddKVStoreFlags()` and `command.KVStoreConfig` struct become dead code after inserter migrates to its own proto config. Remove them in this phase.
- **D-12:** Agent already has its own KV store fields in AgentConfig. InserterConfig already has its own store fields (Phase 7 D-07). No shared KV proto needed.

### Implementation Pattern
- **D-13:** Follow the agent's exact initialization sequence in each component:
  1. Create proto config instance: `config := &proto_config.ComponentConfig{}`
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

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Reference implementation (agent — the pattern to replicate)
- `agent/command.go` — Complete libprotoconf integration: NewConfig, SetEnvKeyPrefix, Environment, PopulateFlagSet, config-file handler. **Note (08 gap closure):** its `proto.Merge(orig, c.config)` config-file handler is the source of the PCLI-09 inversion and is itself rewired by `08-04-PLAN.md`. Treat the initialization sequence (D-13) as the pattern to replicate; do NOT copy its config-file merge — use `command.LayerConfigFile` (D-03).
- `agent/config/v1/agent_config.proto` — Proto config with json_name field options for CLI flag mapping

### Proto definitions to wire (Phase 7 output)
- `server/config/v1/server_config.proto` — ServerConfig message (7 fields: grpc_address, TLS, auth, scripts)
- `compiler/config/v1/compiler_config.proto` — CompilerConfig message (6 fields: repl, verbose, templates, profiling, compiler_address)
- `inserter/config/v1/inserter_config.proto` — InserterConfig message (5 fields: store, store_address, prefix, namespace, delete)
- `mutate/config/v1/mutate_config.proto` — MutateConfig message (11 fields: root, proto file/msg, server, TLS, config_path, metadata, fields)

### Current CLI code to replace
- `server/server.go` lines 62-69 — Manual cliConfig struct with flag.StringVar calls
- `compiler/command.go` lines 31-36 — Manual cliConfig struct with flag parsing
- `inserter/inserter.go` lines 42-44 — Manual cliConfig struct + command.AddKVStoreFlags
- `mutate/mutate.go` lines 58-70 — Manual cliConfig struct with TLS flags
- `command/command.go` — KVStoreConfig struct and AddKVStoreFlags helper (to be removed)

### Requirements
- `.planning/REQUIREMENTS.md` — PCLI-05 through PCLI-09

### libprotoconf library
- `github.com/protoconf/libprotoconf` — NewConfig, PopulateFlagSet, Environment, SetEnvKeyPrefix, Unmarshal

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `libprotoconf` package: Already a dependency, provides the full proto-to-CLI stack (flags, env vars, config files)
- `agent/command.go`: Reference implementation showing exact initialization sequence, config-file handler, and proto.Merge pattern
- Phase 7 proto configs: All four `.proto` files with `json_name` options and generated `.pb.go` files ready to use

### Established Patterns
- Agent's `cliCommand` struct holds both proto config and flag.FlagSet — replicate this pattern
- `configtool.NewConfig(protoMessage)` wraps any proto message for flag/env/file generation
- `proto.Merge(orig, updated)` preserves default values while overlaying file/env values
- `flag.ContinueOnError` used with `flag.NewFlagSet` for non-fatal parse errors
- Positional args accessed via `flagset.Args()` after parsing (protoconf_root, files...)

### Integration Points
- Each component's `Command()` factory function — entry point for wiring
- `command.RunCommand()` and `command.RunSubcommands()` — CLI dispatch remains unchanged
- DevServer constructs agent/compiler/server inline — will need to create proto configs for each
- `generate.go` — No changes needed (Phase 7 already added proto generation)

</code_context>

<specifics>
## Specific Ideas

No specific requirements — replicate the agent pattern consistently across all components.

</specifics>

<deferred>
## Deferred Ideas

- Consolidating KV store proto definitions into a shared package — future concern
- Adding protovalidate rules to config protos — could be added in a future phase
- Migrating mitchellh/cli to cobra or urfave/cli — deferred to v2 (PCLI-10)
- DevServer getting its own proto config — too trivial, composes other configs
- fmt command proto config — only 2 trivial flags, not worth proto-defining

</deferred>

---

*Phase: 08-cli-flag-generation-config-loading*
*Context gathered: 2026-03-29*
