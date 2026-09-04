# Protoconf

## What This Is

Protoconf is a configuration management tool that uses Protocol Buffers as schema and Starlark as the configuration language. It compiles Starlark configs into materialized protobuf, distributes them via KV stores (Consul, etcd, ZooKeeper, Kubernetes ConfigMaps), and serves them to applications via gRPC streaming. Milestone v1.0 delivered a quality and consistency overhaul: testing, security hardening, proto-defined CLI configuration, and removal of deprecated patterns. Milestone v2.0 targets compiler startup performance — making compile time proportional to the config being compiled rather than to the size of the repository it lives in.

## Core Value

Every component must be testable, consistent, and free of runtime surprises — no panics in production code, no os.Exit in libraries, no deprecated APIs, and proper test coverage across all packages.

## Current Milestone

**v2.0 — Compiler Startup Performance**

**Goal:** `protoconf compile` completes in under 200ms on a repository of any size, by parsing and linking only the protos a config actually reaches.

**The problem, measured** (800-proto corpus, `.planning/research/compiler-performance/BASELINE.md`):

| Stage | Time |
|---|---|
| `GetProtoRegistry()` — parse + link all 799 protos | 4,639ms |
| Resolver construction over 864 files | 260ms |
| `CompileFile` — Starlark eval, validate, write | 184ms (already at target) |
| **Total** | **6.97s** |

Cost scales with repository size, not with the config. The config `load()`s 5 protos, 7 transitively, which cost 2.7ms to parse in isolation — a ~1,700x gap.

**Scope:** all six `GetProtoRegistry()` consumers — compiler (`load()` + `loadMutable`), mutation server (including its startup service enumeration), inserter, agent filekv, `mutate`, and `GenReflectionUI`'s periodic walk.

**Key design decision:** type URLs resolve through an exact symbol index built by parsing without linking (1.41s vs 4.64s linked on the 799-proto corpus; 138,554 symbols, nested types included), persisted under `.protoconf_cache`. `proto_file` is superseded as a resolution mechanism. Lazy loading was protoconf's original design and `proto_file` exists because of it; it failed when `Any` arrived because lazy-by-path cannot answer a symbol it was never asked to load. The index is the missing piece, not a repeat of that mistake.

**Definition of done:** `TestCompilerStartupScaling`'s alloc ratio at n=50→400 drops from 7.24x to ≤2.0x, and its `t.Skipf` branch is deleted, becoming a `require.LessOrEqual`.

**Already shipped toward this** (v1.0 tail): `loadValidators` filesystem walk (quick 260904-f5j) and the synthetic corpus generator + scaling gate (quick 260904-fwk).

## Requirements

### Validated

- ✓ Starlark-to-protobuf compilation pipeline — existing
- ✓ Multi-backend KV store distribution (Consul, etcd, ZooKeeper, ConfigMaps, file) — existing
- ✓ gRPC agent with streaming config subscriptions — existing
- ✓ Mutation server with pre/post script support — existing
- ✓ Module system for external protobuf dependencies — existing
- ✓ Dev server combining agent + compiler + mutation — existing
- ✓ OpenTelemetry + Prometheus observability — existing
- ✓ Starlark format command (`protoconf fmt`) — existing
- ✓ Rollout-aware config insertion — existing
- ✓ Proto validation (protovalidate + Starlark validators) — existing

### Active

- [x] Unit test coverage for previously-untested packages — Validated in Phase 9: Unit Test Coverage & Infrastructure (mutate, fmt, command, devserver, KV stores, starproto + shared testutil)
- [x] Refactor os.Exit calls in library code to error returns — Validated in Phase 2: os.Exit Refactoring
- [x] Extract shared OTel bootstrap to common package — Validated in Phase 3: Observability & Global State Cleanup
- [x] Migrate all deprecated gRPC APIs (WithInsecure, v1alpha reflection) — Validated in Phase 1: Deprecated API Migrations
- [ ] Migrate from jhump/protoreflect/dynamic to dynamicpb
- [ ] Lazy, load-driven proto resolution — parse and link only what a config reaches (v2.0)
- [ ] Resolvers as lazy views over the registry, replacing eager snapshots (v2.0)
- [ ] Exact symbol index (parse without linking) resolving type URLs, including nested Any, across all six registry consumers (v2.0)
- [ ] Symbol index persisted under `.protoconf_cache`, content-keyed and invalidated on change (v2.0)
- [ ] Loud (never silent) fallback when a type URL cannot be resolved (v2.0)
- [x] Add TLS support for gRPC connections — Validated in Phase 5: TLS Support
- [x] Token-based auth with credential forwarding to pre/post scripts — Validated in Phase 6: Token Auth & Script Security
- [x] Proto-defined CLI configuration (definitions + flag generation) — Validated in Phase 7: Proto-Defined CLI Configs and Phase 8: CLI Flag Generation & Config Loading
- [x] Env var and config file loading for all components, precedence flags > env > config file > proto defaults — Validated in Phase 8: CLI Flag Generation & Config Loading
- [ ] Fix known bugs (dummykv.Exists always true) — PROTOCONF_COMPILER_ADDR `=` bug fixed in Phase 6; `mutate/mutate.go` TYPE_SINT32 converts to uint32 instead of int32 (found by Phase 8 code review, 08-REVIEW.md WR-02)
- [x] Remove dead code and unnecessary init functions — Validated in Phase 4: Dead Code Removal
- [x] Fix global mutable state issues (Starlark resolver settings, mutate package) — Validated in Phase 3: Observability & Global State Cleanup

### Out of Scope

- KV store unimplemented method implementations — panics are intentional interface stubs; they signal gaps if methods become needed
- Full CLI framework migration (mitchellh/cli to cobra) — the real problem is configs not being proto-defined, which is in scope
- Mobile or web client SDKs — focus is backend quality
- New feature development — v1.0 was purely quality/consistency; v2.0 is purely performance
- Persisted linked-descriptor cache — optimises the ~3ms lazy path, not the 4.6s eager one; the `.fds` machinery already half-exists if warm-start ever justifies it
- Parallelising the eager parse — divides 4.6s by core count at best, still 3-4x over budget, and burns every core on ~99% discarded work
- Patching or bumping protocompile for the linker lookup — `linker.Files.FindFileByPath` scans a file's direct deps, not all 864, and v0.14.1 is byte-identical; there is no lookup to fix
- Making `mod sync` lazy — it serialises the registry to `.fds`, and a lazy registry would write a truncated cache that then loads clean and is wrong

## Context

- Brownfield Go project with ~15 packages, serving as a configuration management platform
- Codebase has grown organically with inconsistencies: mixed error handling (os.Exit vs error returns), duplicate code (OTel setup), deprecated APIs still in use
- Several packages have zero test files: mutate/, devserver/, fmt/, command/, KV stores
- ~~Existing tests have placeholder assertions and TODO comments providing false coverage confidence~~ — Resolved in Phase 10: all placeholders replaced with real assertions, e2e tests added for mutation/TLS/auth
- ~~The project uses mitchellh/cli (maintenance mode) but the deeper issue is that CLI configurations should be defined as protobuf messages and CLI flags generated from those definitions~~ — Resolved across Phases 7-8: all five components define config in proto and generate flags from it via libprotoconf; mitchellh/cli remains only as the subcommand router
- Agent already self-configures via protobuf (agent/config/v1/agent_config.proto) — this pattern should be extended to all components
- Pre/post mutation scripts need auth credentials forwarded as environment variables for git operations

## Constraints

- **Tech stack**: Go 1.22+, must maintain backward compatibility with existing config repos
- **Proto compatibility**: Cannot break existing protobuf wire formats or gRPC service definitions
- **KV store interface**: Must remain compatible with valkeyrie store.Store interface
- **Build**: CGO_ENABLED=0, must produce static binaries
- **Testing**: Must not break existing CI (GitHub Actions with Codecov)

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Keep KV store panic stubs | Intentional interface satisfaction; panics signal future needs | — Pending |
| Token-based auth over mTLS | Simpler to implement and forward to scripts as env vars | — Pending |
| Proto-defined CLI configs | Consistency with protoconf's own philosophy; agent already does this | ✓ Shipped — all five components (agent, serve, compile, insert, mutate) in Phases 7-8 |
| Research proto-to-CLI generation | Need to find the right approach before committing to implementation | ✓ Resolved — libprotoconf `PopulateFlagSet`/`Environment` adopted in Phase 8 |
| Track flag/env provenance from the `flag.FlagSet`, not by comparing values against defaults | Value comparison cannot distinguish "explicitly set to the default" from "unset", and loses an env var to a config file that coincidentally matches it | ✓ Shipped in Phase 8 — `command.ConfigLayerer`, after two rounds of gap closure |
| Migrate jhump/protoreflect to dynamicpb | Official package is the recommended replacement | — Pending |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd:transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd:complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-09-01 after Phase 8 completion*
