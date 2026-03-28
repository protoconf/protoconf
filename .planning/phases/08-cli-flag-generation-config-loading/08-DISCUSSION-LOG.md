# Phase 8: CLI Flag Generation & Config Loading - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-03-29
**Phase:** 08-cli-flag-generation-config-loading
**Areas discussed:** Env var prefix naming, Config file loading, Backward compatibility migration, DevServer composition, KVStoreConfig transition
**Mode:** Auto (--auto flag)

---

## Env Var Prefix Naming

| Option | Description | Selected |
|--------|-------------|----------|
| Component-specific prefix | PROTOCONF_COMPILER_*, PROTOCONF_SERVER_*, etc. matching agent pattern | [auto] |
| Shared PROTOCONF_ prefix | Single prefix for all components — simpler but collision risk | |
| No env var support | Skip env vars, flags and config files only | |

**User's choice:** [auto] Component-specific prefix matching agent pattern (recommended default)
**Notes:** Follows established agent convention. Each component gets isolated namespace.

---

## Config File Loading

| Option | Description | Selected |
|--------|-------------|----------|
| Same --config-file as agent | json/yaml/pb via lpc.Unmarshal, proto.Merge pattern | [auto] |
| Separate --config-json/--config-yaml flags | Explicit format per flag | |
| No config file support | Env vars and flags only | |

**User's choice:** [auto] Same --config-file pattern as agent (recommended default)
**Notes:** Reuses proven pattern. Format auto-detected from extension.

---

## Backward Compatibility Migration

| Option | Description | Selected |
|--------|-------------|----------|
| Replace entirely | Remove manual cliConfig, use proto config exclusively | [auto] |
| Keep both temporarily | Dual flag parsing during transition | |
| Wrapper approach | Manual flags delegate to proto config fields | |

**User's choice:** [auto] Replace entirely with libprotoconf-generated flags (recommended default)
**Notes:** Phase 7's json_name options guarantee identical flag names. No transition period needed.

---

## DevServer Composition

| Option | Description | Selected |
|--------|-------------|----------|
| Inline component configs | DevServer creates AgentConfig/CompilerConfig/ServerConfig inline | [auto] |
| Own DevServerConfig proto | Separate proto composing other configs | |
| Config passthrough | Accept --agent-config-file, --server-config-file flags | |

**User's choice:** [auto] DevServer creates component configs inline (recommended default)
**Notes:** Matches current pattern. DevServer is thin orchestrator, not a configurable component.

---

## KVStoreConfig Transition

| Option | Description | Selected |
|--------|-------------|----------|
| Remove shared helper | Delete command.AddKVStoreFlags and KVStoreConfig | [auto] |
| Keep as convenience | Maintain shared helper alongside proto configs | |
| Refactor to proto helper | Convert KVStoreConfig to wrap proto fields | |

**User's choice:** [auto] Deprecate shared helper — inserter uses its own proto config (recommended default)
**Notes:** Both agent and inserter have their own proto-defined store fields. Shared helper becomes dead code.

---

## Claude's Discretion

- Positional arg handling (protoconf_root, files...) — kept as manual flag.Args()
- DevServer constructor refactoring — flexibility on approach
- Error message wording for config file errors
- Deprecation comments vs clean deletion of manual flag code

## Deferred Ideas

- Shared KV store proto package — future concern
- protovalidate rules on config protos — future phase
- mitchellh/cli migration to cobra — v2 (PCLI-10)
