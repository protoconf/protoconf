---
phase: 14-non-compiler-consumer-correctness
plan: 01
subsystem: inserter
tags: [protoreflect, dynamicpb, protojson, type-resolution, lazy-loading]

# Dependency graph
requires:
  - phase: 13-exact-symbol-index-shared-type-url-resolution
    provides: "RegistryTypeResolver / parser.TypeResolver tiered chain (snapshot -> growable registry -> scan -> symbol index -> hard NotFound), lib.NewLazyModuleService"
provides:
  - "inserter/inserter.go on the lazy construction path (D-01) with both type-URL/Any resolution call sites reading the tiered i.parser.TypeResolver instead of the construction-time-only LocalResolver (D-03)"
  - "A verified template (this plan's diff) for the identical D-01/D-03 edit pattern in server.go, agent/filekv/filekv.go, and mutate/mutate.go"
affects: [14-02, 14-03, 14-04, 14-05, 14-06, 14-07, 14-08]

# Actuals (#2632)
actuals:
  tokens: 1500
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "D-01/D-03 rewiring: swap lib.NewModuleService -> lib.NewLazyModuleService at construction, then swap every LocalResolver read on the write/marshal path to TypeResolver"
    - "Snapshot-vs-tiered contrast test: assert protoregistry.NotFound on the fixed construction-time *protoregistry.Types both before and after the operation that resolves the type through the tiered resolver, proving the resolution was on-demand rather than pre-seeded"

key-files:
  created:
    - inserter/lazy_resolution_test.go
  modified:
    - inserter/inserter.go

key-decisions:
  - "TestInserterUnresolvableTypeReturnsDiagnostic's fixture needed a git commit inside the temp testdata.SmallTestDir() repo before InsertConfigFile could reach the resolver error at all — GatherMetadata's `git log` call fails first (EOF) on an uncommitted file, so the D-02 diagnostic assertion required committing the new fixture via go-git, matching how the rest of utils/testdata/small's fixtures are provided."
  - "The D-02 diagnostic actually surfaces from parser.ReadConfig's protojson unmarshal of the Any field (which uses TypeResolver as its resolver), not from XXXinsertVersion's later FindMessageByURL call — both route through the same resolveTiers chain, so the error text (not found / symbol name / symbol index:) is identical regardless of which call site produces it."

requirements-completed: [CONS-02]

coverage:
  - id: D1
    description: "The inserter constructs its module service lazily and still writes a config.json whose nested google.protobuf.Any renders the wrapped message's fields"
    requirement: "CONS-02"
    verification:
      - kind: unit
        ref: "inserter/inserter_test.go#TestProtoconfInserter_InsertConfig_AnyResolution"
        status: pass
    human_judgment: false
  - id: D2
    description: "A materialized config whose message type is absent from the inserter's construction-time snapshot is still inserted correctly via the shared tiered resolver"
    requirement: "CONS-02"
    verification:
      - kind: unit
        ref: "inserter/lazy_resolution_test.go#TestInserterResolvesTypeAbsentFromConstructionSnapshot"
        status: pass
    human_judgment: false
  - id: D3
    description: "The inserter CLI exits 0 when one file among several fails to insert, logging that failure per file"
    requirement: "CONS-02"
    verification:
      - kind: unit
        ref: "inserter/lazy_resolution_test.go#TestInserterCLIExitsZeroOnPerFileFailure"
        status: pass
    human_judgment: false
  - id: D4
    description: "An unresolvable type URL on the inserter's write path returns the Phase 13 D-02 diagnostic naming the symbol, import roots, and index state, never swallowed into a successful insert"
    requirement: "CONS-02"
    verification:
      - kind: unit
        ref: "inserter/lazy_resolution_test.go#TestInserterUnresolvableTypeReturnsDiagnostic"
        status: pass
    human_judgment: false

duration: 15min
completed: 2026-09-08
status: complete
---

# Phase 14 Plan 01: Inserter Lazy Construction + Tiered Type-URL Resolution Summary

**Inserter now constructs its module service lazily via `lib.NewLazyModuleService` and resolves nested `google.protobuf.Any` types through the shared tiered `RegistryTypeResolver` instead of a frozen construction-time snapshot, with three new tests proving the resolution is genuinely on-demand.**

## Performance

- **Duration:** ~15 min
- **Started:** 2026-09-08T12:15:00Z
- **Completed:** 2026-09-08T12:23:53Z
- **Tasks:** 2
- **Files modified:** 2 (1 modified, 1 created)

## Accomplishments
- `NewProtoconfInserter` now uses `lib.NewLazyModuleService` (D-01) instead of the eager `lib.NewModuleService`
- Both type-URL/`Any` resolution sites in `XXXinsertVersion`'s config.json write block now read `i.parser.TypeResolver` (D-03) instead of the construction-time-only `LocalResolver`
- Pre-existing regression test `TestProtoconfInserter_InsertConfig_AnyResolution` stays green under `-race`, now proving on-demand resolution rather than trivial eager resolution
- New `inserter/lazy_resolution_test.go` proves CONS-02 as an on-demand claim: the resolved type is demonstrably absent from the fixed construction snapshot (both before and after the insert that resolves it), an unresolvable symbol surfaces the Phase 13 D-02 diagnostic rather than being swallowed, and the CLI's per-file skip-and-log-continue exit-code contract (`0`) is pinned by a dedicated test

## Task Commits

Each task was committed atomically:

1. **Task 1: End-to-end — the inserter constructs lazily and inserts a nested-Any config through the tiered resolver** - `36ba014` (feat)
2. **Task 2: CONS-02 proof — the resolved type was genuinely absent from the construction snapshot** - `939d714` (test)

**Plan metadata:** (this commit)

## Files Created/Modified
- `inserter/inserter.go` - `NewProtoconfInserter` construction flipped to `lib.NewLazyModuleService`; both `LocalResolver` reads in the config.json write block swapped to `i.parser.TypeResolver`
- `inserter/lazy_resolution_test.go` - Three new tests: `TestInserterResolvesTypeAbsentFromConstructionSnapshot`, `TestInserterUnresolvableTypeReturnsDiagnostic`, `TestInserterCLIExitsZeroOnPerFileFailure`

## Decisions Made
- Committed the purpose-built unresolvable-type fixture into the temp git repo (via go-git, matching `testdata.SmallTestDir()`'s own commit idiom) so `GatherMetadata`'s `git log` lookup succeeds and the test reaches the actual resolver error rather than an unrelated git error.
- Verified experimentally (scratch tests, discarded) that the D-02 diagnostic text surfaces from `parser.ReadConfig`'s `protojson.Unmarshal` of the `Any` field before `XXXinsertVersion`'s own `FindMessageByURL` call is ever reached — both paths share the same `resolveTiers` chain, so the assertion holds regardless of which call site actually returns the error.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Task 1's exact D-01/D-03 edit pattern (construction swap + `LocalResolver` -> `TypeResolver`) is now proven end-to-end on the consumer with the strongest pre-existing regression test. Plans 14-02 through 14-05 apply the identical mechanical pattern to `server/server.go`, `agent/filekv/filekv.go`, and `mutate/mutate.go`.
- No blockers. `go build ./...`, `go vet ./inserter/...`, and `go test -race ./inserter/...` are all clean; `go.mod`/`go.sum` are untouched.

---
*Phase: 14-non-compiler-consumer-correctness*
*Completed: 2026-09-08*

## Self-Check: PASSED

- FOUND: inserter/lazy_resolution_test.go
- FOUND: inserter/inserter.go
- FOUND commit: 36ba014
- FOUND commit: 939d714
