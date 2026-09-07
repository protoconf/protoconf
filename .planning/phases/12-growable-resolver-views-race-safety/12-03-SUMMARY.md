---
phase: 12-growable-resolver-views-race-safety
plan: 03
subsystem: testing
tags: [protoregistry, race-detector, concurrency, tdd]

requires:
  - phase: 12-growable-resolver-views-race-safety
    provides: "Plan 12-01's growable filesResolver, locked FindFileByPath/RangeFiles/FilesResolverRegistrationCount/FilesResolverRegistrationErrorCount accessors, and plan 12-02's canonical-pointer fix — this plan proves both under concurrency, adding no new production code"
provides:
  - "TestConcurrentCompile now asserts the growable resolver survives 8 concurrent CompileFile calls without divergence from FileRegistry (D-02 under concurrency, production call shape)"
  - "A dedicated utils package test, TestRegisterFileRacesRangeFiles, that forces the RegisterFile-against-RangeFiles/FindFileByPath interleaving continuously via unpaced tight-loop readers, rather than leaving it to scheduler luck"
affects: []

actuals:
  tokens: 1447
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "Unpaced tight-loop reader goroutines (RLock-then-yield, no sleep/ticker) held open for the full duration of concurrent writer goroutines via a done channel + sync.WaitGroup, to raise the probability the race detector observes a narrow read/write interleaving"
    - "Race-test sanity check: temporarily strip the lock under test, confirm go test -race reports WARNING: DATA RACE against the expected write site, then restore the file byte-identical (verified via diff against HEAD) before committing — proves the test can fail, not just that it currently passes"

key-files:
  created:
    - utils/growable_resolver_race_test.go
  modified:
    - compiler/lib/concurrent_compile_test.go

key-decisions:
  - "Resolved Research Open Question 2 as both an extension of TestConcurrentCompile (production call shape) and a dedicated utils-level test (forces the interleaving), exactly as the plan's locked decision specified — neither alone would have been sufficient"
  - "No production code was touched. Both tasks are regression-pinning tests over behavior plan 12-01 already implemented correctly, so there is no RED-to-GREEN transition on an implementation; the RED-phase-equivalent evidence for Task 2 is the sanity check (lock removed -> race fires -> lock restored), documented explicitly below rather than skipped"

requirements-completed: [SAFE-01]

coverage:
  - id: D1
    description: "Two or more concurrent compiles against one shared *lib.Compiler, each reaching a proto the others have not touched, all complete without error and without a data race under go test -race; afterwards every FileRegistry key resolves through FindFileByPath, RangeFiles count is at least len(FileRegistry), registration errors are 0, and registration count is > 0 (SAFE-01, ROADMAP criterion 3, D-02 under concurrency)"
    requirement: "SAFE-01"
    verification:
      - kind: unit
        ref: "compiler/lib/concurrent_compile_test.go#TestConcurrentCompile"
        status: pass
      - kind: unit
        ref: "go test -race ./compiler/..."
        status: pass
    human_judgment: false
  - id: D2
    description: "The RegisterFile-against-RangeFiles/FindFileByPath interleaving is forced continuously (not left to scheduling luck) for the duration of 8 concurrent on-demand parses against distinct paths, and the detector reports nothing"
    requirement: "SAFE-01"
    verification:
      - kind: unit
        ref: "utils/growable_resolver_race_test.go#TestRegisterFileRacesRangeFiles"
        status: pass
      - kind: unit
        ref: "go test -race ./utils/..."
        status: pass
    human_judgment: false
  - id: D3
    description: "The dedicated test is genuinely capable of catching the bug it exists for, not just green by construction — demonstrated by temporarily removing d.mu locking from FindFileByPath/RangeFiles and observing a real WARNING: DATA RACE against registerFileLocked's write, then restoring the accessors byte-identical"
    verification:
      - kind: other
        ref: "manual sanity check, see Issues Encountered / TDD Gate Compliance below"
        status: pass
    human_judgment: true
    rationale: "This is a one-time manual demonstration performed during execution, not a repeatable automated test — a human should confirm the described procedure and output are convincing evidence rather than take the pass/fail label alone"
  - id: D4
    description: "Whole-module go test -race is green with the change in place, using the Phase-11/12-01/12-02-proven agent-package exclusion for the pre-existing, unrelated command_test.go:118 hang"
    verification:
      - kind: unit
        ref: "go test -race -count=1 $(go list ./... | grep -v '/agent$')"
        status: pass
    human_judgment: false

duration: 25min
completed: 2026-09-08
status: complete
---

# Phase 12 Plan 03: Concurrent Resolver Race Tests Summary

**Two new race tests — one extending `TestConcurrentCompile` at the production call shape, one dedicated to forcing `RegisterFile`-vs-`RangeFiles` in `utils` — prove SAFE-01 under `go test -race`, with the dedicated test's failure-detection capability demonstrated by a temporary lock-removal sanity check rather than assumed.**

## Performance

- **Duration:** 25 min (approx, sequential executor, no worktree)
- **Started:** 2026-09-07T18:00:00Z (approx)
- **Completed:** 2026-09-07T18:22:00Z (approx)
- **Tasks:** 2
- **Files modified:** 2 (1 created, 1 modified)

## Accomplishments
- `TestConcurrentCompile` (`compiler/lib/concurrent_compile_test.go`) keeps both of its original `LoadedFileCount` bounds assertions and gains a post-`g.Wait()` block: every `FileRegistry` key resolves through `FindFileByPath`, `RangeFiles`'s count is at least `len(FileRegistry)`, `FilesResolverRegistrationErrorCount()` is 0, and `FilesResolverRegistrationCount()` is greater than 0 — proving D-02 non-divergence survives 8 concurrent `CompileFile` calls at the exact shape `compiler/service.go` and `compiler/command.go` use in production.
- New `utils/growable_resolver_race_test.go`, `TestRegisterFileRacesRangeFiles`: two unpaced tight-loop reader goroutines (`RangeFiles`, `FindFileByPath`) run continuously for the whole duration of 8 concurrent `ParseOne` calls, each demanding a distinct path (`pkg8/msg8.proto`..`pkg15/msg15.proto`) from a 20-file corpus. This makes the `RegisterFile`-vs-enumeration window nearly continuous instead of the rare sliver `TestConcurrentCompile`'s parse-heavy goroutines would otherwise leave for the detector to catch.
- Verified the dedicated test can actually fail: temporarily removed `d.mu` locking from `FindFileByPath` and `RangeFiles`, re-ran the test under `-race`, and observed two distinct `WARNING: DATA RACE` reports — both pointing at `registerFileLocked`'s write inside `protoregistry.(*Files).RegisterFile` racing against the reader's `RangeFiles`/map read — then restored `utils/utils.go` to a byte-identical state (`git diff` against `HEAD` empty) before committing. This is the concrete evidence requested by the plan's critical-correctness note 3, not an assumption.
- No production code was modified. Both tasks consume only the accessors plan `12-01` created; the underlying growable-resolver locking was already correct, so these are regression-pinning tests, not bug fixes.
- All three plan-level `<verification>` commands pass: `go test -race -count=2 ./compiler/lib/... -run TestConcurrentCompile -v`, `go test -race -count=2 ./utils/... -run TestRegisterFileRacesRangeFiles -v`, and `go test -race -count=1 $(go list ./... | grep -v '/agent$')` (the Phase-11-proven exclusion for the pre-existing, unrelated `agent/command_test.go:118` hang).

## Task Commits

1. **Task 1: TestConcurrentCompile also asserts the resolver view survived the concurrency** - `1b33037` (test)
2. **Task 2: Dedicated RegisterFile-against-RangeFiles race test in utils** - `5ea50e0` (test)

**Plan metadata:** (this commit)

## TDD Gate Compliance

Both tasks carry `tdd="true"`, but the plan's own `<artifacts_this_phase_produces>` states "No production source is modified by this plan" — both tasks are regression-pinning tests over behavior plan `12-01` already implemented and proved correct. There is no implementation step to make a failing test pass, so the classic RED-then-GREEN commit pair does not apply here, mirroring 12-01's Task 3 and 12-02's Task 2 (both plain single-commit test additions over already-correct code).

| Task | RED-equivalent evidence | GREEN | REFACTOR | Status |
|------|--------------------------|-------|----------|--------|
| Task 1 | n/a — extends an existing test with assertions against already-implemented accessors; ran once and passed on first try (expected, since 12-01 shipped the locking) | `1b33037` | none needed | Pass |
| Task 2 | Sanity check: temporarily removed `d.mu` from `FindFileByPath`/`RangeFiles`, re-ran under `-race`, observed `WARNING: DATA RACE` against `registerFileLocked`'s write; restored file byte-identical before commit | `5ea50e0` | none needed | Pass |

This is reported plainly per the plan's critical-correctness note 3 ("report honestly ... if you cannot make it fire, say so") — the fire-check succeeded, and the procedure is recorded above rather than just the pass/fail label.

## Files Created/Modified
- `compiler/lib/concurrent_compile_test.go` — `TestConcurrentCompile` gains a post-`g.Wait()` resolver-view assertion block (D-02 under concurrency) and two sentences of doc comment; no new test function, one new import (`protoreflect`, needed for the `RangeFiles` callback signature)
- `utils/growable_resolver_race_test.go` — new, `TestRegisterFileRacesRangeFiles`: two tight-loop readers vs. 8 concurrent `ParseOne` writers, joined via `sync.WaitGroup` after `close(done)`

## Decisions Made
- Confirmed Research Open Question 2's "do both" resolution is correct on evidence: `TestConcurrentCompile`'s goroutines spend the overwhelming majority of their time inside `protoparse.ParseFiles` with no lock held (confirmed by the test's own doc comment and by how narrow the assertion-only failure window would be), so only the dedicated test with continuously-running readers reliably forces the interleaving.
- Kept the reader loops unpaced (`for { select { case <-done: return; default: } }` calling the accessor once per iteration, no `time.Sleep`/ticker) per the plan's explicit instruction — every accessor call already takes and releases `d.mu.RLock()`, which yields, so pacing was unnecessary and would have reintroduced the rare-interleaving problem the test exists to fix.
- Phrased the resolver-vs-registry size assertion in Task 1 as "at least" (`require.GreaterOrEqual`), not exact equality, per the plan's D-02 guidance: the one-time seed build can legitimately pull in a transitive dependency the registry map does not key separately, and an equality assertion would be a spurious future failure waiting to happen.

## Deviations from Plan

None - plan executed exactly as written. No Rule 1-3 auto-fixes were needed; every accessor and counter referenced in the plan's `<behavior>` sections already existed exactly as specified by plan `12-01`.

## Issues Encountered

- **Task 1's first verify command produces a benign "no tests to run" line.** `go test -race -count=2 ./compiler/lib/... -run TestConcurrentCompile -v` recurses into `compiler/lib/parser`, which has no test matching `-run TestConcurrentCompile`, so Go prints `testing: warning: no tests to run` / `ok ... [no tests to run]` for that sibling package. The task's `<verify><fails_when>` text names both substrings as failure signals, and they are literally present in the full output — but they originate from an unrelated package the `/...` wildcard happens to include, not from any failure of `TestConcurrentCompile` itself, which shows a clean `--- PASS: TestConcurrentCompile` line with no race report in both `-count=2` iterations. Read as intended (did the named test run and pass, with no race/fatal signature), the gate is satisfied. Documented here rather than silently reinterpreting the plan's literal text, per the honesty requirement in the plan's critical-correctness notes.
- The known pre-existing, unrelated `agent/command_test.go:118` hang (documented across Phase 11's SUMMARY/UAT and Phase 12's 12-01/12-02 SUMMARYs) still reproduces on an unscoped `go test -race ./...`. Used the same proven workaround: `go test -race -count=1 $(go list ./... | grep -v '/agent$')`, which passed clean across all 27 testable packages in ~130s aggregate CPU time (individual package times up to 128s for `compiler/lib`). Out of scope for this plan.

## Carried-Forward Planning Note (not resolved by this plan)

The plan's `<flagged_planner_assumptions>` section flags that SAFE-01's spec-less `unclassified` edge-coverage probe row is being treated as fully discharged by this plan's two race tests plus plan `12-01`'s locking, with no additional uncovered edge identified. This executor did not have a specific SAFE-01 edge in mind that neither test reaches (e.g., concurrent compiles against *different* registries in one process, or a compile racing `mod sync`'s own eager registry) and did not add a third task speculatively. Surfacing this verbatim for a human to confirm or override, per the plan's own instruction, rather than silently treating it as resolved.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- SAFE-01 (ROADMAP criterion 3) is now proven under `go test -race` at both the production call shape and a dedicated, interleaving-forcing test. Combined with 12-01 (RSLV-01, RSLV-02) and 12-02 (RSLV-03), all four of Phase 12's success criteria are closed.
- Phase 13 (exact symbol index) inherits a `DescriptorRegistry` whose growable resolver view is now proven race-safe under concurrent compiles, not just implemented.
- The `agent` package's pre-existing `command_test.go:118` hang remains open across all three Phase 12 plans (11, 12-01, 12-02, 12-03 now); the cost of re-confirming it every plan continues to add up — still recommended as a dedicated `/gsd-quick` or Phase 14+ cleanup item.
- The carried-forward planning note above (SAFE-01's unclassified edge-coverage row) should be reviewed by the developer before Phase 12 is considered fully closed out.

---
*Phase: 12-growable-resolver-views-race-safety*
*Completed: 2026-09-08*

## Self-Check: PASSED
- `utils/growable_resolver_race_test.go` exists: FOUND
- `compiler/lib/concurrent_compile_test.go` modified (contains `FindFileByPath`, `RangeFiles`, `FilesResolverRegistrationErrorCount`, `FilesResolverRegistrationCount`): FOUND
- Commits `1b33037`, `5ea50e0` present in `git log --oneline --all`: FOUND
- All plan-level `<verification>` commands re-run and green:
  - `go test -race -count=2 ./compiler/lib/... -run TestConcurrentCompile -v`: PASS (see Issues Encountered for the benign "no tests to run" nuance from a sibling package)
  - `go test -race -count=2 ./utils/... -run TestRegisterFileRacesRangeFiles -v`: PASS
  - `go test -race ./compiler/...`: PASS
  - `go test -race ./utils/...`: PASS
  - `go test -race -count=1 $(go list ./... | grep -v '/agent$')`: PASS (agent-package hang pre-existing, documented)
- `git diff` of `utils/utils.go` against `HEAD` after the sanity-check lock removal/restoration: empty (confirmed clean before Task 2's commit)
