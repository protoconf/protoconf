# Protoconf

## What This Is

Protoconf is a configuration management tool that uses Protocol Buffers as schema and Starlark as the configuration language. It compiles Starlark configs into materialized protobuf, distributes them via KV stores (Consul, etcd, ZooKeeper, Kubernetes ConfigMaps), and serves them to applications via gRPC streaming. Milestone v1.0 delivered a quality and consistency overhaul: testing, security hardening, proto-defined CLI configuration, and removal of deprecated patterns. Milestone v2.0 delivered compiler startup performance: compile time is now proportional to the config being compiled rather than to the size of the repository it lives in, enforced by two CI gates.

## Core Value

Every component must be testable, consistent, and free of runtime surprises — no panics in production code, no os.Exit in libraries, no deprecated APIs, and proper test coverage across all packages.

## Current State

**Shipped: v2.0 — Compiler Startup Performance** (2026-09-10)

Compile cost is now driven by the config, not the repository. `GetProtoRegistry()` no longer bulk-parses `src/`; protos are parsed on demand behind a singleflight guard, resolvers grow in place instead of snapshotting, and type URLs resolve through an exact symbol index built by parsing without linking and persisted content-keyed under `.protoconf_cache`. All six registry consumers — compiler, mutation server, inserter, agent filekv, `mutate`, and the reflection UI walk — construct lazily and resolve through one shared tiered chain.

**Numbers at close** (calibrated 2,400-proto corpus):

| Measurement | Value |
|---|---|
| Allocation ratio, n=50 to n=400 | 1.02x (was 7.24x; gate asserts 2.0x) |
| Startup, in-process local | 70.9ms |
| Startup, CI | 93.4ms |
| Wall-clock budget asserted in CI | 160ms |

Both gates run in CI as hard assertions, not measurements. The scaling gate's `t.Skipf` branch is gone.

**Deliberate behavior change:** a `.proto` no config reaches is no longer parsed, so it is no longer reported at compile time. The changelog and readme name `buf` as the operator's remedy and state plainly that this repository's own CI runs `buf breaking` over protoconf's own protos only.

**Known verification overrides:** 3 newly acknowledged, 0 carried forward from a prior close (see STATE.md Deferred Items). All five phases closed with stale verification digests, invalidated in bulk by post-verification lint and deprecation cleanup across 38 files; whole-repo `go vet` and the full test suite were green at close.

## Next Milestone Goals

Not yet scoped. Two requirements carried forward from v1.0 remain open, and the v2.0 work surfaced no new blocking debt.

- Migrate from `jhump/protoreflect/dynamic` to `dynamicpb`
- Fix the `mutate/mutate.go` TYPE_SINT32 conversion bug (converts to uint32 instead of int32; found in 08-REVIEW.md WR-02) and `dummykv.Exists` always returning true

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
- ✓ Migrate all deprecated gRPC APIs (WithInsecure, v1alpha reflection) — v1.0 (Phase 1)
- ✓ Refactor os.Exit calls in library code to error returns — v1.0 (Phase 2)
- ✓ Extract shared OTel bootstrap; fix global mutable state — v1.0 (Phase 3)
- ✓ Remove dead code and unnecessary init functions — v1.0 (Phase 4)
- ✓ TLS support for gRPC connections — v1.0 (Phase 5)
- ✓ Token-based auth with credential forwarding to pre/post scripts — v1.0 (Phase 6)
- ✓ Proto-defined CLI configuration with generated flags — v1.0 (Phases 7 and 8)
- ✓ Env var and config file loading, precedence flags > env > config file > proto defaults — v1.0 (Phase 8)
- ✓ Unit test coverage for previously-untested packages — v1.0 (Phase 9)
- ✓ Lazy, load-driven proto resolution — parse and link only what a config reaches — v2.0 (Phase 11, LAZY-01..05)
- ✓ Resolvers as lazy views over the registry, replacing eager snapshots — v2.0 (Phase 12, RSLV-01..03, SAFE-01)
- ✓ Exact symbol index (parse without linking) resolving type URLs, including nested Any, across all six registry consumers — v2.0 (Phases 13 and 14, TYPE-01..09, CONS-02..05, SAFE-02, SAFE-03)
- ✓ Symbol index persisted under `.protoconf_cache`, content-keyed and invalidated on change — v2.0 (Phase 13, TYPE-04..06)
- ✓ Loud (never silent) fallback when a type URL cannot be resolved — v2.0 (Phase 13, TYPE-08; the whole-tree eager fallback is deleted)
- ✓ Startup performance asserted in CI, not merely measured, with the milestone's closing numbers recorded as evidence — v2.0 (Phase 15, GATE-01..05)

### Active

- [ ] Migrate from jhump/protoreflect/dynamic to dynamicpb
- [ ] Fix known bugs (dummykv.Exists always true) — PROTOCONF_COMPILER_ADDR `=` bug fixed in Phase 6; `mutate/mutate.go` TYPE_SINT32 converts to uint32 instead of int32 (found by Phase 8 code review, 08-REVIEW.md WR-02)

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
- After v2.0: ~6,450 lines of Go added across 73 files over 25 plans; whole-repo `go vet ./...` is silent and the full race-enabled suite is green (excluding `Test_cliCommand_Run`, which hangs without a live Consul or etcd backend)
- The synthetic proto corpus generator and its 2,400-proto calibrated corpus now back both CI startup gates; the real 799-proto corpus lives in a sibling checkout that CI cannot fetch

## Constraints

- **Tech stack**: Go 1.25.8+, must maintain backward compatibility with existing config repos
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
| Migrate jhump/protoreflect to dynamicpb | Official package is the recommended replacement | — Pending, carried into the next milestone |
| Delete the whole-tree eager fallback outright rather than keep it behind a warning (D-02) | Leaving the 4.6s path reachable is what this milestone exists to prevent; a scan and index blind spot should fail loudly, not silently cost 4.6s | ✓ Shipped in Phase 13 — confirmed at a one-way checkpoint after 13-01/13-02 evidence showed no uncovered resolution case |
| Resolve every in-compiler type URL through one shared `RegistryTypeResolver` chokepoint, not per-consumer lookups | A per-consumer lookup is how `loadMutable` silently bypassed the chain (CONS-05); one chain means one place to fix and one place to observe | ✓ Shipped in Phase 13 — `resolveTiers` is the single chain both `FindMessageByURL` and `FindMessageByName` delegate to |
| Marshal gRPC-UI examples with the shared resolver instead of handing grpcui a proto message | `standalone.ExampleRequest.MarshalJSON` hardcodes a resolver-less `protojson.Marshal` against `protoregistry.GlobalTypes`, which can never hold a type declared only in the user's `.proto` tree | ✓ Shipped in Phase 13 — caught by the regression gate; narrows CONS-04's Phase 14 surface |
| Fix the `MutateConfig` write-path escape inside Phase 14 rather than deferring it | The gap-closure plan's sibling audit found the same defect class, in a weaker form, on a write path; deferring a known unguarded `os.WriteFile` of caller-controlled content to ship a narrower fix trades a real hole for schedule neatness | ✓ Shipped in Phase 14 (14-09) — containment check precedes marshal, pre-mutation script, `MkdirAll` and `WriteFile` |
| A test cited as a security control must be demonstrated capable of failing before its guard lands | Phase 14 shipped a traversal test that could not fail, and two threat-model rows plus a docstring then cited it as a mitigation — false assurance is worse than a known gap | ✓ Shipped in Phase 14 (14-09) — RED `--- FAIL:` output recorded verbatim in each commit body; verification independently reproduced it against the pre-fix commits |
| Calibrate the wall-clock startup budget from an observed GitHub Actions run, never from a laptop figure (D-03) | A threshold set on developer hardware either never fires on CI or fires constantly; only the runner's own numbers bound the runner | ✓ Shipped in Phase 15 — 160ms, roughly 2x the first CI observation of 77.6ms; a later run measured 150.2ms, so the real band is wider than one sample showed |
| Run the wall-clock gate in its own CI step without `-race` or coverage (D-01) | The race detector costs roughly 8x, which would make a wall-clock assertion measure the detector rather than the compiler | ✓ Shipped in Phase 15 — a `//go:build race` guard also skips the gate whenever it is reached under `-race` |
| Accept that an unreferenced broken proto is no longer reported at compile time, and say so in the changelog and readme (D-04) | Lazy loading never parses a proto no config reaches; the honest answer is to name `buf` for whole-tree validation rather than quietly drop a guarantee operators relied on | ✓ Shipped in Phase 15 — both documents carry the caveat that protoconf's own CI runs `buf breaking` only, over protoconf's own protos |
| Gate on a calibrated 2,400-proto generated corpus instead of the real 799-proto sibling checkout (D-07/D-08) | CI cannot fetch a sibling repository, so a gate depending on one is a gate that does not run; the substitution is recorded rather than presented as the real-corpus number | ✓ Shipped in Phase 15 — the baseline record states plainly that the ~3x per-file multiplier sizing the corpus is unsourced and accepted at face value |

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
*Last updated: 2026-09-10 after v2.0 milestone*
