---
phase: 14-non-compiler-consumer-correctness
plan: 05
subsystem: mutate
tags: [protoreflect, dynamicpb, protojson, type-resolution, lazy-loading, mutate-cli, legacy-grpc]

# Dependency graph
requires:
  - phase: 14-non-compiler-consumer-correctness
    provides: "14-01/14-02/14-03's verified D-01/D-03 construction-flip pattern (lib.NewLazyModuleService, LocalResolver -> TypeResolver swap), and 14-04's GenReflectionUI aggregate-and-continue rewrite"
provides:
  - "mutate/mutate.go on the lazy construction path (D-01) with its resolver reassigned from parser.TypeResolver (D-03), exit-1 abort on an unresolvable message unchanged and pinned by a test"
  - "compiler/lib/module_service.go's NewLazyModuleService doc comment corrected to match the post-phase consumer set (compiler, server, inserter, agent/filekv, mutate all lazy; mod sync is the sole remaining eager consumer)"
  - "A verified, runnable end-state gate proving zero non-comment references to the construction-time snapshot field survive in inserter/, mutate/, agent/, server/ outside the two deliberately-excluded server.go ExtensionResolver fields"
  - "A fixed pre-existing production bug in server/legacy.go's MutateConfig cross-package proto conversion (proto.Merge -> Marshal/Unmarshal), discovered because this plan's Task 2 test is the first end-to-end round trip through mutate's actual gRPC client"
affects: [14-06, 14-07, 14-08]

# Actuals (#2632)
actuals:
  tokens: 2440
  tasks: 2
  commits: 2
  plan_head_before: 4ca0f0d29ebbbde138c13fdd02133cb103f786a4

tech-stack:
  added: []
  patterns:
    - "D-01/D-03 construction flip + resolver reassignment, same mechanical pattern as 14-01/14-02/14-03, applied to the fourth and final consumer"
    - "Repo-wide end-state gate: grep for the construction-time snapshot field name outside compiler/lib/parser/, filtering full-line comments, the two allowed ExtensionResolver assignments, and (Rule 1 fix) legitimate test-file snapshot-vs-tiered contrast probes"

key-files:
  created: []
  modified:
    - mutate/mutate.go
    - mutate/mutate_test.go
    - compiler/lib/module_service.go
    - server/legacy.go

key-decisions:
  - "[Rule 1 - Bug] Fixed server/legacy.go's legacyProtoconfMutationServer.MutateConfig: it used proto.Merge to copy between two wire-compatible-but-distinct proto message types (v1.ConfigMutationRequest vs protoconf.v1.ConfigMutationRequest), which panics with 'descriptor mismatch' since proto.Merge requires identical descriptors. Switched to Marshal/Unmarshal, the correct idiom for cross-package wire-compatible copies. This is the exact code path every real `protoconf mutate` invocation exercises (mutate.go's client is the legacy `pc` package) -- the bug predates Phase 14 (commit fb46573) and was simply never reached end-to-end by any existing test before this plan's Task 2 test drove a real client through a real server."
  - "[Rule 1 - Bug] Corrected the literal end-state gate command from Task 2's <verify> block: as written, it flags inserter/lazy_resolution_test.go and agent/filekv/filekv_test.go's intentional snapshot-vs-tiered contrast assertions (i.parser.LocalResolver.FindMessageByURL / s.parser.LocalResolver.FindMessageByURL) as offenders. Those are verification code proving the frozen snapshot stays empty -- not a second resolution source in the CONS-02/03/04 sense the gate's must_haves truth #3 actually targets ('no construction-time snapshot remains as a type-URL or nested-Any resolution source'). The gate's own worked example ('At HEAD the gate reports four offending lines') was computed before 14-01/14-02 added these tests, so the literal script under-specifies its own documented intent. Added a `_test.go` exclusion to the filter; re-ran and confirmed zero offenders remain in production code across inserter/, mutate/, agent/, server/."

requirements-completed: [CONS-02, CONS-03, CONS-04]

coverage:
  - id: D1
    description: "The mutate CLI constructs its module service lazily and resolves its target message name through the shared tiered resolver, so a type absent from the construction snapshot still resolves, and the resulting mutation reaches a real mutation server end-to-end"
    requirement: "CONS-02"
    verification:
      - kind: unit
        ref: "mutate/mutate_test.go#TestMutateResolvesMessageAbsentFromConstructionSnapshot"
        status: pass
    human_judgment: false
  - id: D2
    description: "The mutate CLI still returns exit code 1 (not unified with the inserter's exit-0 skip-and-continue) when its target message cannot be found anywhere under src/"
    requirement: "CONS-02"
    verification:
      - kind: unit
        ref: "mutate/mutate_test.go#TestMutateExitsOneOnUnresolvableMessage"
        status: pass
    human_judgment: false
  - id: D3
    description: "No construction-time snapshot remains as a type-URL or nested-Any resolution source anywhere in inserter, mutate, agent or server outside the two deliberately-excluded server.go ExtensionResolver fields"
    requirement: "CONS-04"
    verification:
      - kind: other
        ref: "corrected end-state gate: grep -rn --include='*.go' 'LocalResolver' inserter/ mutate/ agent/ server/ | grep -vE '^[^:]+:[0-9]+:[[:space:]]*//' | grep -vE 'ExtensionResolver:[[:space:]]*s\\.parser\\.LocalResolver' | grep -vE '_test\\.go:' -- exits with zero output"
        status: pass
    human_judgment: false
  - id: D4
    description: "NewLazyModuleService's doc comment states the truth after this phase: mod sync is the only remaining NewModuleService consumer, and the other four (compiler, server, inserter, agent/filekv, mutate) construct lazily"
    requirement: "CONS-03"
    verification:
      - kind: unit
        ref: "grep acceptance criteria over compiler/lib/module_service.go and mod/command.go (must-keep-using count 0, mod-sync mention present, mod/command.go still calls lib.NewModuleService)"
        status: pass
    human_judgment: false

duration: 30min
completed: 2026-09-08
status: complete
---

# Phase 14 Plan 05: Mutate CLI Lazy Construction + Repo-Wide Single-Resolution-Path End-State Summary

**`mutate/mutate.go` now constructs lazily and resolves through the tiered `TypeResolver` (the fourth and final D-01/D-03 consumer), a pre-existing panic in the legacy gRPC mutation adapter is fixed, and a repo-wide gate proves zero surviving construction-time-snapshot resolution sources remain outside the two deliberately-excluded reflection extension-resolver fields.**

## Performance

- **Duration:** ~30 min
- **Tasks:** 2
- **Files modified:** 4 (mutate/mutate.go, mutate/mutate_test.go, compiler/lib/module_service.go, server/legacy.go)

## Accomplishments
- `mutate/mutate.go`'s `Run` constructs its module service via `lib.NewLazyModuleService` instead of `lib.NewModuleService` (D-01); `anyResolver` is reassigned from `parser.TypeResolver` instead of the construction-time-only `LocalResolver` (D-03) -- a two-line diff, exactly as `14-PATTERNS.md` predicted
- The `-msg`-not-found abort (`return 1`) is byte-for-byte unchanged and now pinned by `TestMutateExitsOneOnUnresolvableMessage`, which asserts the exact integer `1` -- guarding against accidental unification with the inserter's skip-and-continue exit `0`
- `TestMutateResolvesMessageAbsentFromConstructionSnapshot` proves the full, real end-to-end path: a message declared only under `src/ondemand/v1/thing.proto` resolves through the scan tier, gets set via a `-field` argument, is dialed to a real TCP-listening mutation server, and the mutable config file is written to disk -- exit code exactly `0`
- Discovered and fixed a pre-existing production bug this test's real round trip exposed: `server/legacy.go`'s `legacyProtoconfMutationServer.MutateConfig` used `proto.Merge` to copy between two different-but-wire-compatible proto message types, which panics with "descriptor mismatch" because `proto.Merge` requires identical descriptors. Every real `protoconf mutate` CLI invocation goes through this exact legacy adapter (mutate.go's gRPC client is the legacy `pc` package), so this was a live, unreachable-until-now bug in the actual mutate flow, dating to commit `fb46573` -- long before Phase 14
- `compiler/lib/module_service.go`'s `NewLazyModuleService` doc comment rewritten: it claimed every other consumer "must keep using `NewModuleService`", which is now false. The corrected comment states the compiler, mutation server, inserter, `agent/filekv` and `mutate` all construct through it, and `mod sync` (verified live at `mod/command.go:81`, still `lib.NewModuleService(".")`) is the sole remaining eager consumer
- Ran the repo-wide end-state gate (corrected -- see Deviations): zero non-comment references to the construction-time snapshot field (`LocalResolver`) survive in production code across `inserter/`, `mutate/`, `agent/`, `server/`, outside the two deliberately-excluded `server.go` `ExtensionResolver: s.parser.LocalResolver` assignments (D-04)

## Task Commits

Each task was committed atomically:

1. **Task 1: mutate CLI lazy construction, tiered resolution, exit-1 abort preserved** - `07ab5f9` (feat)
2. **Task 2: doc-comment truth + repo-wide single-resolution-path end-state gate** - `7431153` (docs)

**Plan metadata:** (this commit)

## Files Created/Modified
- `mutate/mutate.go` - `lib.NewModuleService` -> `lib.NewLazyModuleService` (D-01); `anyResolver` reassigned from `parser.TypeResolver` (D-03)
- `mutate/mutate_test.go` - Added `TestMutateExitsOneOnUnresolvableMessage`, `newOndemandMutateFixture`, `TestMutateResolvesMessageAbsentFromConstructionSnapshot`
- `compiler/lib/module_service.go` - Rewrote `NewLazyModuleService`'s doc comment to match the post-phase consumer set
- `server/legacy.go` - [Rule 1] `legacyProtoconfMutationServer.MutateConfig` switched from `proto.Merge` to `proto.Marshal`/`proto.Unmarshal` for cross-package wire-compatible request/response conversion

## Decisions Made
- See `key-decisions` in frontmatter: the `server/legacy.go` bug fix and the end-state gate's `_test.go` exclusion are both Rule 1 auto-fixes, documented in full below under Deviations.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed `server/legacy.go`'s `proto.Merge` panic on cross-package conversion**
- **Found during:** Task 1's `<verify>` (`TestMutateResolvesMessageAbsentFromConstructionSnapshot`, the first test in the repo to drive a real mutate-CLI client through a real TCP-listening mutation server via the legacy gRPC service)
- **Issue:** `legacyProtoconfMutationServer.MutateConfig` called `proto.Merge(next, in)` to copy `v1.ConfigMutationRequest` (mutate's client type) into `protoconf.v1.ConfigMutationRequest` (the server's native type). `proto.Merge` requires `dst` and `src` to share a descriptor; these are two distinct, wire-compatible-but-differently-packaged proto messages, so it panicked: `descriptor mismatch: protoconf.v1.ConfigMutationRequest != v1.ConfigMutationRequest`. Same bug on the response leg.
- **Fix:** Replaced both `proto.Merge` calls with `proto.Marshal(src)` followed by `proto.Unmarshal(bytes, dst)` -- the standard idiom for copying between two wire-compatible-but-distinct proto message types.
- **Files modified:** `server/legacy.go`
- **Verification:** `TestMutateResolvesMessageAbsentFromConstructionSnapshot` passes end-to-end (exit 0, mutable config file written); `go test -race -count=1 ./server/... ./mutate/...` green
- **Committed in:** `07ab5f9` (Task 1 commit)

**2. [Rule 1 - Bug] Corrected the end-state gate to exclude legitimate test-file snapshot probes**
- **Found during:** Task 2's `<verify>` (running the literal gate command from the plan)
- **Issue:** The gate as literally written in the plan text flags `inserter/lazy_resolution_test.go:36,45` and `agent/filekv/filekv_test.go:249,259` -- intentional "snapshot-vs-tiered contrast" test assertions (`i.parser.LocalResolver.FindMessageByURL(url)` / `s.parser.LocalResolver.FindMessageByURL(url)`) that prove the frozen construction-time snapshot stays empty, added by 14-01 and 14-02. These are verification code, not a second production resolution source -- the exact distinction must_haves truth #3 draws ("no construction-time snapshot remains as a *resolution source*"). The plan's own worked example ("At HEAD the gate reports four offending lines") was computed before 14-01/14-02 introduced this test pattern, so the literal script doesn't yet express its own documented intent.
- **Fix:** Added a `_test.go` exclusion (`grep -vE '_test\.go:'`) to the gate's filter chain, alongside the existing full-line-comment and `ExtensionResolver:` exclusions.
- **Files modified:** none (verification-only; the gate is inline shell run by the executor, not a persisted script)
- **Verification:** Corrected gate exits 0 with zero output across `inserter/`, `mutate/`, `agent/`, `server/`
- **Committed in:** n/a (documented here; no source change)

### Deferred (Out of Scope)

**3. [Scope boundary] Two more pre-existing, unrelated `go vet ./...` findings**
- **Found during:** Task 2's `<verify>` (`go build ./... && go vet ./...`)
- **Issue:** Unscoped `go vet ./...` is not silent: in addition to the already-logged `test/e2e_test.go:327,411` `lostcancel` findings (14-03), it also reports `agent/agent_test.go:27` (`lostcancel` on a `context.WithTimeoutCause` discard, blamed to commit `f7ba446e`, 2024-03-17) and `agent/legacy.go:97` (`unreachable code`, blamed to commit `fb465735`, 2024-05-27) -- both roughly two years old, both outside this plan's diff.
- **Why deferred:** 14-05 is the first plan in this phase whose `<verify>` text calls unscoped `go vet ./...`; prior plans (14-02/14-03/14-04) scoped `go vet` to the packages they touched and never surfaced these. Confirmed via `go vet ./mutate/... ./compiler/... ./server/... ./inserter/... ./agent/filekv/...` (every package this plan's diff touches or is adjacent to) being silent.
- **Logged:** `.planning/phases/14-non-compiler-consumer-correctness/deferred-items.md` and `.planning/WINDOWS.md` (entries 3 and 4, kind `lint-warning`)

---

**Total deviations:** 2 auto-fixed (both Rule 1 -- bugs), 1 deferred (pre-existing, out of scope)
**Impact on plan:** The `server/legacy.go` fix is necessary for correctness -- without it, every real `protoconf mutate` invocation would panic the server. The gate correction preserves the plan's documented intent (a production-code-only resolution-source check) rather than weakening it. No scope creep beyond what was required to make Task 1's own required end-to-end proof possible.

## Issues Encountered

None beyond the two Rule 1 items above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- All four D-01/D-03 consumer sites (`inserter`, `agent/filekv`, `server`, `mutate`) are now on the lazy construction path with tiered resolution, closing the phase's core CONS-02/03/04 requirement set. The repo-wide end-state gate (corrected for legitimate test probes) confirms zero surviving construction-time-snapshot resolution sources in production code.
- `go build ./...` succeeds; `go vet` is silent for every package this plan touches or is adjacent to (four pre-existing, unrelated findings remain elsewhere and are logged); `go test -race -count=1 -skip 'Test_cliCommand_Run' ./...` is fully green across all 20 tested packages; `go.mod`/`go.sum` are untouched.
- No blockers for 14-06/14-07/14-08.

---
*Phase: 14-non-compiler-consumer-correctness*
*Completed: 2026-09-08*

## Self-Check: PASSED

- FOUND: mutate/mutate.go
- FOUND: mutate/mutate_test.go
- FOUND: compiler/lib/module_service.go
- FOUND: server/legacy.go
- FOUND commit: 07ab5f9
- FOUND commit: 7431153
