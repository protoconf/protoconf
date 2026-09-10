---
phase: 15-verification-decision-gate-flip
plan: 01
subsystem: testing
tags: [go-test, ci, github-actions, performance-gate]

requires:
  - phase: 11-14 (compiler startup performance)
    provides: lazy registry, growable resolver views, exact symbol index — the mechanism these gates assert against
provides:
  - GATE-01 as a real assertion (require.LessOrEqual) instead of a t.Skipf, demonstrated capable of failing
  - GATE-02 as a new CI-calibrated wall-clock assertion (TestCompilerStartupBudget), demonstrated capable of failing
  - a race-detector guard (raceEnabled) so the wall-clock gate never asserts against a race-scaled number
  - a new "Run startup budget gate" CI step, non-race/non-coverage, with an anti-vacuity grep guard
affects: [15-02, 15-03]

actuals:
  tokens: 2100
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "//go:build race init() flag (raceEnabled) to let a wall-clock test self-skip under the race detector"
    - "pipefail + tee + grep anti-vacuity guard for a CI step whose go test -run could otherwise match nothing and exit 0"

key-files:
  created:
    - compiler/lib/race_detector_test.go
  modified:
    - compiler/lib/startup_bench_test.go
    - .github/workflows/go.yml

key-decisions:
  - "TestCompilerStartupScaling stays in the existing -race Run coverage step rather than moving to the new non-race step (Claude's Discretion, resolved in PLAN.md): its allocation ratio is race-insensitive (0.91x plain, 1.02x under -race), the saving is under 0.5% of the ~11.5-12min job, moving it adds bookkeeping (-skip on one side, -run on the other) that can silently rot into running twice or nowhere, and it keeps its Codecov contribution."
  - "Budget threshold calibrated at 160ms — roughly 2x the first observed ubuntu-latest CI figure (77.58ms, run 34311638861) — per D-03, not from any planning-document number."
  - "A second CI observation on the calibrated run measured 150.2ms against the 160ms threshold, only ~6.5% headroom. This is a real finding, not a defect: it shows ubuntu-latest run-to-run variance is larger than the first single sample suggested. Recorded here rather than silently re-tuned — CLAUDE.md's own prohibition on raising a gate in response to a run that went red does not apply (this run went green), but a maintainer tightening or loosening this constant later should know the true variance band is closer to 77-150ms, not a single point estimate."

requirements-completed: [GATE-01, GATE-02]

coverage:
  - id: D1
    description: "TestCompilerStartupScaling asserts allocRatio <= 2.0 via require.LessOrEqual (GATE-01), replacing the t.Skipf that was the milestone's placeholder definition of done"
    requirement: "GATE-01"
    verification:
      - kind: unit
        ref: "compiler/lib/startup_bench_test.go#TestCompilerStartupScaling"
        status: pass
    human_judgment: false
  - id: D2
    description: "TestCompilerStartupScaling's guard demonstrated capable of failing: maxRatio temporarily lowered to 0.5 produced --- FAIL, restoring to 2.0 produced --- PASS"
    requirement: "GATE-01"
    verification:
      - kind: unit
        ref: "compiler/lib/startup_bench_test.go#TestCompilerStartupScaling (manual maxRatio=0.5 run, not committed)"
        status: pass
    human_judgment: false
  - id: D3
    description: "TestCompilerStartupBudget asserts a CI-calibrated 160ms wall-clock budget over a 2400-proto generated corpus (GATE-02), self-skipping under -race and -short"
    requirement: "GATE-02"
    verification:
      - kind: unit
        ref: "compiler/lib/startup_bench_test.go#TestCompilerStartupBudget"
        status: pass
      - kind: e2e
        ref: "GitHub Actions run 34312220627, step 'Run startup budget gate'"
        status: pass
    human_judgment: false
  - id: D4
    description: "TestCompilerStartupBudget's guard demonstrated capable of failing: budget temporarily lowered to 1ms produced --- FAIL, restoring to 160ms produced --- PASS"
    requirement: "GATE-02"
    verification:
      - kind: unit
        ref: "compiler/lib/startup_bench_test.go#TestCompilerStartupBudget (manual budget=1ms run, not committed)"
        status: pass
    human_judgment: false
  - id: D5
    description: "New 'Run startup budget gate' CI step runs between Build and Run coverage, without -race/-coverprofile, with a pipefail+grep anti-vacuity guard; the existing Run coverage step is byte-unchanged"
    verification:
      - kind: e2e
        ref: "GitHub Actions run 34312220627 (both steps green)"
        status: pass
      - kind: other
        ref: "grep -c '\\-coverprofile' .github/workflows/go.yml == 1"
        status: pass
    human_judgment: false

duration: 29min
completed: 2026-09-09
status: complete
---

# Phase 15 Plan 1: Startup Performance Gates Flip Summary

**GATE-01 and GATE-02 turned from a measurement-only skip and a missing test into real CI-enforced assertions: TestCompilerStartupScaling now hard-asserts its allocation ratio, and a new TestCompilerStartupBudget asserts a 160ms wall-clock budget calibrated from two live GitHub Actions observations (77.58ms and 150.2ms on ubuntu-latest), running in a dedicated non-race CI step that self-skips under the race detector.**

## Performance

- **Duration:** 29 min
- **Started:** 2026-09-09T04:31:38Z
- **Completed:** 2026-09-09T05:00:28Z
- **Tasks:** 3
- **Files modified:** 3 (1 created, 2 modified)

## Accomplishments
- GATE-01: `TestCompilerStartupScaling`'s `t.Skipf` branch is deleted; it now hard-asserts `require.LessOrEqual(t, allocRatio, maxRatio)` with `maxRatio = 2.0`, demonstrated capable of failing (temporarily set to 0.5, observed `--- FAIL`; restored, observed `--- PASS` at 0.91x).
- GATE-02: new `TestCompilerStartupBudget` measures `compileCorpus` over a calibrated 2400-proto generated corpus (D-07/D-08) and asserts `require.LessOrEqual(t, elapsed, budget)` with `budget = 160ms`, calibrated from a real `ubuntu-latest` GitHub Actions observation (run `34311638861`, `startup budget: n=2400 compiled in 77.581263ms`) rather than any laptop or planning-document number (D-03). Demonstrated capable of failing (temporarily set to 1ms, observed `--- FAIL`; restored, observed `--- PASS`).
- `compiler/lib/race_detector_test.go` (`//go:build race`) sets `raceEnabled = true` so the wall-clock gate self-skips under `-race` — confirmed `--- SKIP: TestCompilerStartupBudget` under `go test -race`, and confirmed `go test -race -count=1 ./compiler/lib/...` still exits 0 overall.
- New `Run startup budget gate` CI step lands between `Build` and `Run coverage` in the existing `build` job, running `go test -run '^TestCompilerStartupBudget$' -count=1 -v ./compiler/lib/...` with neither `-race` nor `-coverprofile`, piped through `tee`+`grep -q 'startup budget: n='` as an anti-vacuity guard. The existing `Run coverage` step is byte-unchanged (verified: exactly one `-coverprofile` occurrence in the file, and the exact prior command line still present).
- A full GitHub Actions run on the branch (`34312220627`) shows the whole `build` job green, including both the new step and the pre-existing `Run coverage` step.
- The pinned project command `go test -race -timeout 300s -skip 'Test_cliCommand_Run' ./...` passes (20/20 packages `ok`, 0 failures).

## Task Commits

Each task was committed atomically:

1. **Task 1: End-to-end startup budget — one measurement, from test through CI, logged only** - `4111293` (feat)
2. **Task 2: Flip the allocation gate — replace the skip with an assertion (GATE-01)** - `2055877` (feat)
3. **Task 3: Calibrate the budget threshold from the observed CI run (GATE-02)** - `0c5a27a` (feat)

_Task 1 pushed immediately to dispatch the CI run Task 3 needed; Task 2 was committed but deliberately not pushed (the workflow's `cancel-in-progress: true` would have cancelled the in-flight run); Task 3 pushed once calibration was complete._

## Files Created/Modified
- `compiler/lib/race_detector_test.go` - `//go:build race` file; `init()` sets `raceEnabled = true` so the wall-clock budget test knows it is running under the race detector
- `compiler/lib/startup_bench_test.go` - added `var raceEnabled bool`, `const corpusProtos = 2400`, `const budget = 160 * time.Millisecond`, `TestCompilerStartupBudget`; flipped `TestCompilerStartupScaling`'s skip to an assertion and rewrote its doc comment
- `.github/workflows/go.yml` - added the `Run startup budget gate` step between `Build` and `Run coverage`

## Decisions Made
- `TestCompilerStartupScaling` stays in the existing `-race` `Run coverage` step rather than moving to the new non-race step (Claude's Discretion, resolved in PLAN.md before execution) — see frontmatter `key-decisions` for the full rationale.
- Budget threshold set to 160ms: roughly 2x the first observed CI figure (77.58ms), per D-03's calibration rule, not from any number in CONTEXT.md/RESEARCH.md/BASELINE.md.
- Recorded but not acted on: a second CI observation on the calibrated run measured 150.2ms (only ~6.5% headroom under 160ms), showing ubuntu-latest run-to-run variance is wider than the single first sample suggested. The run still passed; the constant was not touched in response, consistent with the plan's prohibition against loosening a gate in reaction to observed numbers. Flagged here as a fact a future maintainer should have, not as a defect requiring a fix.

## Deviations from Plan

None - plan executed exactly as written. The `Run coverage` step failed on the first CI run (`34311638861`, exit 143, a timeout/kill unrelated to this plan's changes — it occurred in the pre-existing `-race -coverprofile ./...` step, not in the new `Run startup budget gate` step, which passed on both runs) and passed cleanly on the second (calibrated) run `34312220627`. This is pre-existing flakiness the plan's own CONTEXT.md (D-10) already names, out of scope for this plan, and not attributed to it.

## Issues Encountered
None beyond the pre-existing `Run coverage` flakiness noted above, which resolved itself on the next run without any change on my part.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- GATE-01 and GATE-02 are now real, CI-enforced assertions with demonstrated failure modes — both close cleanly for whatever the next plan in this phase (GATE-03/04/05, decision write-ups) builds on.
- The calibrated 160ms budget has real but not generous headroom against observed CI variance (77-150ms band on two samples). A future regression that pushes wall-clock time up by roughly 5-10% could flip this gate red on a slow runner even with no real regression — worth keeping in mind if this test starts flaking, per TESTING.md's convention that a flaking threshold should be examined rather than mechanically bumped.
- No blockers for the remaining Phase 15 plans (GATE-03 broken-proto decision write-up, GATE-04 test-suite evidence, GATE-05 real-corpus evidence/BASELINE.md after-section).

---
*Phase: 15-verification-decision-gate-flip*
*Completed: 2026-09-09*

## Self-Check: PASSED

- FOUND: compiler/lib/race_detector_test.go
- FOUND: compiler/lib/startup_bench_test.go
- FOUND: .github/workflows/go.yml
- FOUND: commit 4111293 (Task 1)
- FOUND: commit 2055877 (Task 2)
- FOUND: commit 0c5a27a (Task 3)
- Plan-level `<verification>` re-run: `TestCompilerStartupScaling` and `TestCompilerStartupBudget` both PASS locally; `go test -race -count=1 ./compiler/lib/...` passes with the budget test SKIPped; `go test -race -timeout 300s -skip 'Test_cliCommand_Run' ./...` passes (20/20 packages ok); GitHub Actions run 34312220627 is green end-to-end including `Run startup budget gate` and `Run coverage`.
