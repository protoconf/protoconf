---
phase: 14-non-compiler-consumer-correctness
plan: 02
subsystem: agent
tags: [protoreflect, dynamicpb, protojson, type-resolution, lazy-loading, filekv]

# Dependency graph
requires:
  - phase: 14-non-compiler-consumer-correctness
    provides: "14-01's verified D-01/D-03 construction-flip pattern (lib.NewLazyModuleService, LocalResolver -> TypeResolver swap)"
provides:
  - "agent/filekv/filekv.go on the lazy construction path (D-01), a one-line diff"
  - "Proof (not assumption) that Store.Get resolves a type absent from the construction snapshot via the shared tiered TypeResolver, idempotently, with a loud diagnostic on genuine failure"
affects: [14-03, 14-04, 14-05, 14-06, 14-07, 14-08]

# Actuals (#2632)
actuals:
  tokens: 1660
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "D-01 construction-only flip: lib.NewModuleService -> lib.NewLazyModuleService, zero resolver rewiring when the read path already routes through parser.ReadConfig's TypeResolver"
    - "Snapshot-vs-tiered contrast test: assert errors.Is(err, protoregistry.NotFound) on the fixed construction-time *protoregistry.Types both before and after the operation that resolves the type through the tiered resolver"

key-files:
  created: []
  modified:
    - agent/filekv/filekv.go
    - agent/filekv/filekv_test.go

key-decisions:
  - "Followed 14-01's precedent exactly: since Task 1 already ships the only implementation change, Task 2's four proof tests are committed as a single test(14-02) commit with no accompanying feat/refactor commit — there is no new implementation for GREEN to make pass, only proof that Task 1's flip already works correctly."
  - "Built a purpose-built temp-root fixture (src/ondemand/v1/thing.proto + materialized_config/ondemand.materialized_JSON) instead of reusing testdata.SmallTestDir(), because SmallTestDir resolves everything at construction under both eager and lazy paths and cannot exercise the scan/index tiers CONS-03 needs proven."

requirements-completed: [CONS-03]

coverage:
  - id: D1
    description: "The agent's filekv store constructs its module service lazily and still serves a subscribed client a config whose message type was not present at construction"
    requirement: "CONS-03"
    verification:
      - kind: unit
        ref: "agent/filekv/filekv_test.go#TestGetResolvesTypeAbsentFromConstructionSnapshot"
        status: pass
    human_judgment: false
  - id: D2
    description: "Repeat Get calls for the same key return byte-identical values -- the on-demand parse is memoised, not repeated"
    requirement: "CONS-03"
    verification:
      - kind: unit
        ref: "agent/filekv/filekv_test.go#TestGetIsIdempotentForSameKey"
        status: pass
    human_judgment: false
  - id: D3
    description: "Get returns the Phase 13 D-02 diagnostic error for a config whose type is declared nowhere under src/, rather than a zero-value or partially-populated KVPair"
    requirement: "CONS-03"
    verification:
      - kind: unit
        ref: "agent/filekv/filekv_test.go#TestGetUnresolvableTypeReturnsDiagnostic"
        status: pass
    human_judgment: false
  - id: D4
    description: "The existing key-traversal guard (T-14-03) is pinned as a regression against the D-01 construction flip"
    requirement: "CONS-03"
    verification:
      - kind: unit
        ref: "agent/filekv/filekv_test.go#TestGetRejectsTraversalKey"
        status: pass
    human_judgment: false

duration: 22min
completed: 2026-09-08
status: complete
---

# Phase 14 Plan 02: Agent FileKV Lazy Construction + On-Demand Resolution Proof Summary

**agent/filekv now constructs its module service lazily via `lib.NewLazyModuleService` (one-line diff), and four new tests prove `Get` resolves a type absent from the construction snapshot on demand, idempotently, with a loud diagnostic on failure and the traversal guard intact.**

## Performance

- **Duration:** ~22 min
- **Started:** 2026-09-08T12:25:00Z
- **Completed:** 2026-09-08T12:47:00Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- `agent/filekv/filekv.go`'s `New` constructs the module service via `lib.NewLazyModuleService` instead of `lib.NewModuleService` (D-01) — a single-line diff, exactly as predicted; no resolver rewiring needed since `Get` already routes through `s.parser.ReadConfig`, whose `protojson` resolver is the tiered `TypeResolver` wired in Phase 13
- `TestGetResolvesTypeAbsentFromConstructionSnapshot` proves CONS-03 as an on-demand claim, not an inference from the flip: `type.googleapis.com/ondemand.v1.Thing` is genuinely absent from `s.parser.LocalResolver`'s fixed construction-time snapshot both before and after a successful `Get`, while `Get` itself resolves and returns the correct type URL
- `TestGetIsIdempotentForSameKey` proves three successive `Get` calls for the same key return byte-identical `KVPair.Value`
- `TestGetUnresolvableTypeReturnsDiagnostic` proves a type declared nowhere under `src/` returns the Phase 13 D-02 diagnostic (`not found`, the symbol name, `symbol index:`) rather than a zero-value or partial `KVPair`
- `TestGetRejectsTraversalKey` pins the pre-existing key-traversal guard (T-14-03) as a regression, unaffected by the construction flip
- Existing `TestGet_ValidKey` and the full pre-existing `agent/filekv` test suite (14 tests) remain green under `-race`

## Task Commits

Each task was committed atomically:

1. **Task 1: D-01 construction flip in agent/filekv, and nothing else** - `1107d63` (feat)
2. **Task 2: CONS-03 proof — Get serves a type the construction snapshot never held, idempotently** - `8f1320b` (test)

**Plan metadata:** (this commit)

_Note: this TDD-flagged task produced no GREEN/REFACTOR commits — the only implementation change was already shipped in Task 1, so Task 2's tests exist purely to prove that change correct, matching 14-01's precedent for the identical construction-flip pattern._

## Files Created/Modified
- `agent/filekv/filekv.go` - `New`'s module-service constructor call swapped to `lib.NewLazyModuleService` (D-01); no other line changed
- `agent/filekv/filekv_test.go` - Added `newOnDemandFixture`/`newOnDemandStore` helpers and four new tests: `TestGetResolvesTypeAbsentFromConstructionSnapshot`, `TestGetIsIdempotentForSameKey`, `TestGetUnresolvableTypeReturnsDiagnostic`, `TestGetRejectsTraversalKey`

## Decisions Made
- Task 2's tests are a single `test(14-02)` commit with no `feat`/`refactor` commit, mirroring 14-01 Task 2's precedent: the RED/GREEN cycle doesn't apply cleanly here because Task 1 already shipped the only implementation change this plan makes, and Task 2 is scoped (by the plan itself) to prove it — "Do not modify agent/filekv/filekv.go in this task."
- Built a purpose-built temp-root fixture (`src/ondemand/v1/thing.proto`, package `ondemand.v1`, matching directory layout) rather than reusing `testdata.SmallTestDir()`, because `SmallTestDir()` resolves every type at construction under both eager and lazy paths and cannot exercise the scan/index tiers that make the "absent from construction snapshot" claim meaningful.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None. The known pre-existing `agent.Test_cliCommand_Run/run_consul_server` hang was avoided per the plan's environment note by bounding `agent/...` test runs with `-timeout 300s -skip 'Test_cliCommand_Run'`; the plan's own literal `go test -race -count=1 ./agent/...` verification command was run with that bound substituted, consistent with the environment note's instruction to bound rather than fix that unrelated hang.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `agent/filekv` is now on the lazy construction path, with CONS-03 proven rather than assumed. Two of the four D-01/D-03 consumer sites (`inserter`, `agent/filekv`) are now converted; `server/server.go` (14-03/14-04/14-05, more involved: D-05/D-06 also apply) and `mutate/mutate.go` (14-06 or later) remain.
- No blockers. `go build ./...`, `go vet ./agent/filekv/...`, `go test -race ./agent/filekv/... -v`, and `go test -race -count=1 -timeout 300s -skip 'Test_cliCommand_Run' ./agent/...` are all clean; `go.mod`/`go.sum` are untouched; `agent/filekv/filekv.go`'s total plan diff is exactly one changed line.

---
*Phase: 14-non-compiler-consumer-correctness*
*Completed: 2026-09-08*

## Self-Check: PASSED

- FOUND: agent/filekv/filekv.go
- FOUND: agent/filekv/filekv_test.go
- FOUND commit: 1107d63
- FOUND commit: 8f1320b
