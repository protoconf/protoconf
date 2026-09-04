# Phase 7: Proto-Defined CLI Configs - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-03-28
**Phase:** 07-proto-defined-cli-configs
**Areas discussed:** Proto package organization, Backward compatibility, Config field mapping, Scope
**Mode:** Auto (--auto flag, all recommended defaults selected)

---

## Proto Package Organization

| Option | Description | Selected |
|--------|-------------|----------|
| One proto per component in config/v1/ | Follows AgentConfig pattern, clear ownership | :heavy_check_mark: |
| Single shared proto file | All configs in one file — simpler but less modular | |
| Proto in component root | server/server_config.proto — flatter but inconsistent with agent | |

**User's choice:** [auto] One proto per component in config/v1/
**Notes:** Follows established pattern from agent/config/v1/agent_config.proto.

---

## Backward Compatibility

| Option | Description | Selected |
|--------|-------------|----------|
| Exact flag name match via json_name | json_name field options ensure identical CLI flags | :heavy_check_mark: |
| Allow flag name changes | Risk breaking existing scripts/CI | |

**User's choice:** [auto] Exact flag name match via json_name
**Notes:** PROJECT.md constraint: must maintain backward compatibility with existing config repos.

---

## Config Field Mapping

| Option | Description | Selected |
|--------|-------------|----------|
| Per-component protos with inline fields | Each config defines its own fields, TLS can reuse or inline | :heavy_check_mark: |
| Shared KV/TLS proto package | Extract common fields — more DRY but adds coupling | |

**User's choice:** [auto] Per-component protos with inline fields
**Notes:** Claude's discretion whether to import AgentConfig.TLSConfig or define flat TLS strings.

---

## Scope

| Option | Description | Selected |
|--------|-------------|----------|
| Server, compiler, inserter, mutate only | Matches PCLI-01 through PCLI-04 exactly | :heavy_check_mark: |
| Include fmt and devserver | fmt has 2 flags; devserver composes others — not worth it | |

**User's choice:** [auto] Server, compiler, inserter, mutate only
**Notes:** fmt and devserver excluded per requirements mapping.

---

## Claude's Discretion

- TLS field reuse vs inline definition
- StoreType enum sharing
- Field numbering
- Help text as proto comments

## Deferred Ideas

- Shared KV config proto — future
- Config validation rules — future
- CLI flag wiring — Phase 8
- Config file loading — Phase 8
