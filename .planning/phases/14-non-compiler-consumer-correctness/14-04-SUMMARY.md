---
phase: 14-non-compiler-consumer-correctness
plan: 04
subsystem: server
tags: [grpcui, reflection, error-aggregation, filepath-walkdir, mutation-server]

# Dependency graph
requires:
  - phase: 14-non-compiler-consumer-correctness
    provides: "14-03's D-01/D-03/D-06 mutation-server lazy construction and reflection-completeness wiring, which this plan builds on unchanged"
provides:
  - "server/server.go's GenReflectionUI walks all of mutable_config/ to completion, aggregating every unresolvable config's path/type URL/error instead of aborting on the first one (CONS-04, D-05)"
  - "Change-detection state (reflectionMu + lastReflectionFingerprint) so the aggregate is logged only when the failure set actually changes between 5-second ticker passes"
  - "All four GenReflectionUI call sites (server.go ticker + inline, devserver/command.go ticker + inline) assign the returned error explicitly instead of a bare statement"
affects: [14-05, 14-06, 14-07]

# Actuals (#2632)
actuals:
  tokens: 4074
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "errors.Join aggregation over a per-failure fmt.Errorf, matching this file's existing idiom (inserter.go, server.go)"
    - "Order-independent, reason-sensitive fingerprint (sorted 'path|typeURL|err.Error()' lines) as the log-on-change comparison key"

key-files:
  created:
    - server/gen_reflection_ui_test.go
  modified:
    - server/server.go
    - server/server_test.go
    - devserver/command.go

key-decisions:
  - "Fingerprinting and log-on-change comparison live inside collectExamples (not a separate step in GenReflectionUI), since collectExamples is the only place holding the raw []reflectionFailure slice and the required signature is ([]standalone.Example, error) — two return values, no third slice. collectExamples is only called from GenReflectionUI in production, so this still satisfies 'GenReflectionUI owns the log-on-change reporting' in spirit."
  - "Updated TestProtoconfMutationServer_GenReflectionUI (server_test.go) to expect a non-nil error naming SmallTestDir's bad_json/bad_proto_file fixtures, since D-05 correctly surfaces those ReadConfig failures now instead of silently swallowing them — the old assertion (err must be nil) was testing the exact bug this plan fixes, and Task 1's own <verify> runs this test."
  - "An fs.ErrNotExist on the walk root (absent mutable_config/) is excluded from the failure aggregate entirely (nil error, zero examples); every other directory-level WalkDir error is collected like a per-file failure and does not abort the walk."

requirements-completed: [CONS-04]

coverage:
  - id: D1
    description: "GenReflectionUI's mutable_config/ walk completes past an unresolvable config instead of aborting, and returns a non-nil aggregate naming every failure"
    requirement: "CONS-04"
    verification:
      - kind: unit
        ref: "server/gen_reflection_ui_test.go#TestCollectExamplesContinuesPastUnresolvableConfig"
        status: pass
      - kind: unit
        ref: "server/gen_reflection_ui_test.go#TestCollectExamplesAggregateNamesEveryFailure"
        status: pass
      - kind: unit
        ref: "server/server_test.go#TestProtoconfMutationServer_GenReflectionUI"
        status: pass
    human_judgment: false
  - id: D2
    description: "Empty and single-element mutable_config/ inputs behave defined: absent/empty directory yields zero examples and nil error; a single unresolvable config yields zero examples and an error naming it"
    requirement: "CONS-04"
    verification:
      - kind: unit
        ref: "server/gen_reflection_ui_test.go#TestGenReflectionUIEmptyMutableConfig"
        status: pass
      - kind: unit
        ref: "server/gen_reflection_ui_test.go#TestGenReflectionUISingleUnresolvableConfig"
        status: pass
    human_judgment: false
  - id: D3
    description: "The aggregated failure set is logged only when it changes between passes: order-independent comparison, but a changed failure reason for the same path re-logs"
    requirement: "CONS-04"
    verification:
      - kind: unit
        ref: "server/gen_reflection_ui_test.go#TestReflectionFailureSetIsOrderIndependent"
        status: pass
      - kind: unit
        ref: "server/gen_reflection_ui_test.go#TestReflectionFailureReasonChangeRelogs"
        status: pass
    human_judgment: false
  - id: D4
    description: "GenReflectionUI's shared change-detection state is race-free under concurrent calls from the ticker goroutine and the inline caller"
    requirement: "CONS-04"
    verification:
      - kind: unit
        ref: "server/gen_reflection_ui_test.go#TestGenReflectionUIConcurrentCallsAreRaceFree"
        status: pass
    human_judgment: false

duration: 15min
completed: 2026-09-08
status: complete
---

# Phase 14 Plan 04: GenReflectionUI Aggregate-and-Continue Summary

**`GenReflectionUI`'s mutable_config/ walk now runs to completion via an extracted `collectExamples` method that aggregates every unresolvable config with `errors.Join` and logs the aggregate only when the failure set actually changes, instead of aborting `filepath.WalkDir` on the first resolution failure and discarding its own return value.**

## Performance

- **Duration:** ~15 min
- **Tasks:** 2
- **Files modified:** 4 (1 created, 3 modified)

## Accomplishments
- Extracted `collectExamples() ([]standalone.Example, error)` from `GenReflectionUI`: every branch inside the `filepath.WalkDir` closure now appends to a `reflectionFailure` slice and returns `nil`, so one unresolvable config never truncates the examples that would have followed it
- `filepath.WalkDir`'s own return value is now captured and folded into the aggregate; an absent `mutable_config/` directory (`fs.ErrNotExist` on the walk root) is excluded from the aggregate entirely — zero examples, nil error — while every other directory-level I/O error is collected
- Added `reflectionMu sync.Mutex` + `lastReflectionFingerprint string` to `ProtoconfMutationServer`, plus an order-independent, reason-sensitive `reflectionFailureFingerprint` helper, so the aggregate is logged with `logger.Error` only when the failure set changes between the 5-second ticker's passes (and a recovery line logs when a non-empty failure set clears)
- All four call sites (`server.go`'s ticker goroutine + inline call, `devserver/command.go`'s inline call + ticker goroutine) now assign `GenReflectionUI`'s return explicitly (`_ = ...GenReflectionUI(...)`) with a comment noting `GenReflectionUI` owns the reporting; none of them stop the ticker on a failed pass
- New `server/gen_reflection_ui_test.go` covers all four CONS-04 probe-surfaced edges: aggregate-and-continue (with a fixture whose broken config sorts before the resolvable ones), empty/absent/single-element inputs, order-independent-but-reason-sensitive fingerprinting, and race-free concurrent calls

## Task Commits

Each task was committed atomically:

1. **Task 1: Rewrite GenReflectionUI to complete the walk, aggregate, return, and log on change** - `51930ba` (feat)
2. **Task 2: CONS-04 test file — aggregate-and-continue, change detection, empty, ordering, concurrency** - `d7d4461` (test)

**Plan metadata:** (this commit)

## Files Created/Modified
- `server/server.go` - Added `sort` import, `reflectionFailure` struct, `reflectionFailureFingerprint` helper, `reflectionMu`/`lastReflectionFingerprint` fields, extracted `collectExamples`, rewrote `GenReflectionUI` to call it and return its aggregate, updated all four call sites to explicit assignment
- `server/server_test.go` - Updated `TestProtoconfMutationServer_GenReflectionUI` to expect the (now correct) non-nil error naming `bad_json`/`bad_proto_file`
- `devserver/command.go` - Updated both `GenReflectionUI` call sites to explicit assignment with a comment
- `server/gen_reflection_ui_test.go` - New file: seven tests covering aggregate-and-continue, empty/single-element edges, fingerprint ordering/reason-sensitivity, and concurrency

## Decisions Made
- Fingerprinting and the log-on-change comparison live inside `collectExamples` rather than `GenReflectionUI` itself, since `collectExamples`'s mandated two-value return signature (`[]standalone.Example, error`) has no room for a third `[]reflectionFailure` return, and `collectExamples` is the only place holding that raw slice. `collectExamples` is production-only-called-from `GenReflectionUI`, so this preserves "GenReflectionUI owns log-on-change reporting" without widening the return signature.
- `TestProtoconfMutationServer_GenReflectionUI`'s assertion was updated (Rule 1 — test now correctly reflects fixed behavior): before this plan, `SmallTestDir`'s `bad_json`/`bad_proto_file` ReadConfig failures were silently swallowed (logged, `nil` returned); CONS-04 requires reporting them, so a non-nil error naming both is now the correct, not-a-failure outcome. Task 1's own `<verify>` block runs this exact test, confirming the plan authors intended this update.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Updated pre-existing test assertion to match the now-correct behavior**
- **Found during:** Task 1's `<verify>` (`go test -race ... -run 'TestProtoconfMutationServer_GenReflectionUI|...'`)
- **Issue:** `TestProtoconfMutationServer_GenReflectionUI` asserted `GenReflectionUI` returns `nil`, which was only true because the pre-fix code silently swallowed `SmallTestDir`'s two intentionally-broken fixtures (`bad_json`, `bad_proto_file`). Once D-05 correctly surfaces those failures, the test would flag correct behavior as a bug.
- **Fix:** Changed the assertion to `require.Error(t, err)` plus `assert.Contains` on both broken fixture names, matching CONS-04's report-don't-skip contract.
- **Files modified:** `server/server_test.go`
- **Verification:** `go test -race ./server/... -run TestProtoconfMutationServer_GenReflectionUI -v` passes
- **Committed in:** `51930ba` (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (Rule 1, pre-existing test assertion)
**Impact on plan:** None on scope — the fix updates a test to match the exact behavior this plan was written to correct, per Task 1's own `<verify>` block.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `GenReflectionUI` now reports every unresolvable mutable config rather than silently truncating the example list; `go build ./...`, `go vet ./server/... ./devserver/...`, and `go test -race -count=1 ./server/... ./devserver/... ./test/...` are all green; `go.mod`/`go.sum` are untouched.
- No new struct field beyond the two change-detection fields on `ProtoconfMutationServer`; no widening of the resolution path.
- No blockers for 14-05/14-06/14-07.

---
*Phase: 14-non-compiler-consumer-correctness*
*Completed: 2026-09-08*

## Self-Check: PASSED

- FOUND: server/server.go
- FOUND: server/server_test.go
- FOUND: devserver/command.go
- FOUND: server/gen_reflection_ui_test.go
- FOUND commit: 51930ba
- FOUND commit: d7d4461
