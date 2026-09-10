---
phase: 15-verification-decision-gate-flip
plan: 03
subsystem: testing
tags: [github-actions, evidence, baseline, gate-04, gate-05]

requires:
  - phase: 15-verification-decision-gate-flip (plan 01)
    provides: "TestCompilerStartupBudget with its calibrated 160ms threshold, and the corpusProtos=2400 constant this plan's figures are measured against"
provides:
  - "Milestone-close section in BASELINE.md recording the in-process, CI-observed, calibrated-threshold and hand-timed-CLI figures over the 2400-proto calibrated corpus (GATE-05)"
  - "A named, successful GitHub Actions run (34313915733, commit cbfe79c) with all five GATE-04 fixtures' outcomes and the local-vs-CI test coverage gap stated (GATE-04)"
affects: []

actuals:
  tokens: 1700
  tasks: 3
  commits: 1

tech-stack:
  added: []
  patterns: []

key-files:
  created: []
  modified:
    - .planning/research/compiler-performance/BASELINE.md

key-decisions:
  - "Milestone-close section placed before ## Reproduction (the plan's suggested default), so the reproduction notes remain the file's closer."
  - "CLI figure taken as the best of three /usr/bin/time -p runs (1.38s, 0.08s, 0.08s) — the first run's 1.38s is first-run page-cache/binary-load noise on the freshly built 129MB binary, not a second measurement worth separately recording as the CLI figure."
  - "GATE-04's five named fixtures were all found green (PASS) in run 34313915733's testdox-formatted log after downloading the raw Go-results test_results.json artifact — gotestsum's default testdox rendering does not print t.Log output for passing tests, so the TestCompilerStartupScaling allocation-ratio line (alloc ratio=1.02x) had to be extracted from the downloaded JSON artifact, not the plain gh run view --log text."
  - "The pre-existing load_remote_with_load_local.pconf skip was confirmed present in this run's log (as expected per D-11) and deliberately not carried into BASELINE.md's new section."

requirements-completed: [GATE-04, GATE-05]

coverage:
  - id: D1
    description: "BASELINE.md gains a milestone-close section recording the in-process compileCorpus figure (70.9ms local / 93.4ms CI), the calibrated 160ms threshold, and one hand-timed real protoconf compile CLI invocation (80ms), all over the same 2400-proto calibrated corpus"
    requirement: "GATE-05"
    verification:
      - kind: other
        ref: "go test -run '^TestCompilerStartupBudget$' -count=1 -v ./compiler/lib/... (local, non-race) -- startup budget: n=2400 compiled in 70.929333ms"
        status: pass
      - kind: other
        ref: "/usr/bin/time -p <scratch>/protoconf compile <dir> main.mpconf, best of three: 1.38s/0.08s/0.08s"
        status: pass
    human_judgment: false
  - id: D2
    description: "The section states plainly that the figure was measured over a GenerateCorpus-generated corpus, not the real 799-proto protoconf-terraform tree, and that the ~3x per-file multiplier is unsourced and accepted at face value"
    requirement: "GATE-05"
    verification:
      - kind: other
        ref: "grep -c GenerateCorpus .planning/research/compiler-performance/BASELINE.md == 1; grep -q protoconf-terraform (present, 4 occurrences); grep -q '39ms at n=50, 35ms at n=500'"
        status: pass
    human_judgment: false
  - id: D3
    description: "The GATE-05 requirement-text mismatch is surfaced: GATE-05 as written asks for the real protoconf-terraform corpus, D-07 supersedes it with the calibrated generated corpus"
    requirement: "GATE-05"
    verification:
      - kind: other
        ref: "BASELINE.md '### GATE-05's requirement-text mismatch' section quotes REQUIREMENTS.md's GATE-05 text verbatim and states D-07 supersedes it"
        status: pass
    human_judgment: false
  - id: D4
    description: "GATE-04 evidence names a specific successful GitHub Actions run (34313915733) and commit SHA (cbfe79c), with all five named fixtures' outcomes, and states the exact local-vs-CI gap (Test_cliCommand_Run in inserter/, server/, agent/, compiler/)"
    requirement: "GATE-04"
    verification:
      - kind: e2e
        ref: "GitHub Actions run 34313915733, conclusion success, headSha cbfe79c66e1d8adbe0042ff41bd1e9b90e4f1d24"
        status: pass
      - kind: unit
        ref: "compiler/lib/compiler_test.go#TestCompiler_CompileFile (with_config_rollout_validator.pconf, validator_test.pconf, validator_repeated_test.pconf, validator_map_test.pconf, field_type_any_test.pconf) — all PASS in run 34313915733"
        status: pass
    human_judgment: false
  - id: D5
    description: "A cancelled run or one with no test results is not accepted; when multiple runs exist, the latest completed successful run on the branch head is cited"
    requirement: "GATE-04"
    verification:
      - kind: other
        ref: "gh run list --workflow=go.yml --branch perf/compiler confirmed 34313915733 completed/success on the exact headSha this plan's commits were measured against; an earlier in-progress run (34313304833, older SHA) was superseded by concurrency cancel-in-progress and not cited"
        status: pass
    human_judgment: false
  - id: D6
    description: "BASELINE.md's 2026-09-04 before-numbers (6.97s headline, stage breakdown, CPU profile, negative result, related measurements) are unedited — the new section is appended, not merged in"
    verification:
      - kind: other
        ref: "git diff --stat shows 104 insertions(+), 0 deletions on BASELINE.md; grep confirms 6.97s, 4,639ms, 1,286ms, 'a ~1,700x gap' all still present"
        status: pass
    human_judgment: false

duration: 26min
completed: 2026-09-09
status: complete
---

# Phase 15 Plan 3: Milestone-Close Evidence for GATE-04 and GATE-05 Summary

**Appended a milestone-close section to BASELINE.md recording four labelled after-numbers (70.9ms in-process local, 93.4ms CI, 160ms calibrated threshold, 80ms hand-timed CLI) over the calibrated 2400-proto corpus, and closed GATE-04 against a named green GitHub Actions run (34313915733) with all five validator/Any fixtures unchanged in outcome.**

## Performance

- **Duration:** 26 min
- **Started:** 2026-09-09T05:01:38Z (approx, following 15-02 completion)
- **Completed:** 2026-09-09T05:27:19Z
- **Tasks:** 3
- **Files modified:** 1

## Accomplishments
- **GATE-05 figures collected (Task 1):** local `TestCompilerStartupBudget` run measured `startup budget: n=2400 compiled in 70.929333ms (budget 160ms)`. A real `protoconf compile <dir> main.mpconf` CLI invocation was hand-timed three times (`/usr/bin/time -p`, best of three: 1.38s / 0.08s / 0.08s), against a binary built with `go build -o <scratch>/protoconf ./cmd/protoconf` and a corpus generated outside the repository via a throwaway `go run` invocation of `testdata.GenerateCorpus(dir, 2400)`. Every invocation's `compile finished` log line confirmed `protoFilesLoaded=5` — proof the lazy path, not a whole-tree scan, was exercised over the 2400-proto tree. `git status --porcelain` was empty at the end of the task (measurement artifacts lived entirely under the session scratch directory, never inside the repository).
- **GATE-04 CI evidence obtained (Task 2):** ran the pinned local command `go test -race -timeout 300s -skip 'Test_cliCommand_Run' ./...` (0 failures, all packages ok/cached), then pushed the branch and watched GitHub Actions run `34313915733` to completion — `conclusion: success` on commit `cbfe79c66e1d8adbe0042ff41bd1e9b90e4f1d24`. Extracted from the run's log and downloaded `test_results.json` artifact: all five named fixtures (`with_config_rollout_validator.pconf`, `validator_test.pconf`, `validator_repeated_test.pconf`, `validator_map_test.pconf` — all `ErrInvalidConfig` — and `field_type_any_test.pconf` — `nil`) PASS; `TestCompilerStartupScaling` PASS with `alloc ratio=1.02x`; the CI-side `startup budget: n=2400 compiled in 93.393125ms` line; zero `fail` actions across the entire `-race ./...` JSON stream; and confirmation that the pre-existing `load_remote_with_load_local.pconf` skip is present (as D-11 predicted) but not carried into BASELINE.md.
- **BASELINE.md milestone-close section appended (Task 3):** one new `## Milestone close (2026-09-09)` section placed before `## Reproduction`, containing the after-numbers table, the corpus honesty statement (GenerateCorpus-generated, ~3x multiplier unsourced, exposure low because startup cost is flat at 39ms/n=50 vs 35ms/n=500), the GATE-05 requirement-text-mismatch callout (quoting REQUIREMENTS.md verbatim, naming D-07 as the superseding decision), the GATE-04 evidence table with the run id/SHA/conclusion and the D-10 local-vs-CI coverage-gap statement, and a five-line gate-resting-state summary (GATE-01 through GATE-05). All seven pre-existing `##` headings and their exact figures (`6.97s`, `4,639ms`, `1,286ms`, `a ~1,700x gap`) survive untouched; the diff is 104 insertions, 0 deletions.

## Task Commits

Tasks 1 and 2 produced no repository file changes (measurements and CI evidence only, per plan's own `<files>` annotation — everything they touched lives outside the working tree or in the task output). Task 3 committed the only code change:

1. **Task 1: Measure the two closing figures — in-process and real CLI** - (no commit; no repository files modified)
2. **Task 2: Obtain the authoritative green CI run for GATE-04** - (no commit; no repository files modified)
3. **Task 3: Append the milestone-close section to BASELINE.md** - `435069f` (docs)

## Files Created/Modified
- `.planning/research/compiler-performance/BASELINE.md` - appended a `## Milestone close (2026-09-09)` section (after-numbers table, corpus honesty statement, GATE-05 requirement-text mismatch, GATE-04 evidence, gates' resting state) before the pre-existing `## Reproduction` closer; all prior content unedited

## Decisions Made
- Placed the new section before `## Reproduction` (the plan's suggested default) rather than after, so the reproduction notes remain the file's closer.
- Recorded 0.08s as the CLI figure (best of three), treating the first run's 1.38s as first-run page-cache/binary-load noise on the freshly built binary rather than a competing measurement.
- Extracted the CI-side `TestCompilerStartupScaling` allocation-ratio line from the downloaded `test_results.json` artifact rather than the plain `gh run view --log` text, since gotestsum's testdox format does not print `t.Log` output for passing tests.
- Confirmed but did not name `load_remote_with_load_local.pconf`'s skip in BASELINE.md, per D-11's explicit instruction.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None. The CI run took approximately 12 minutes end-to-end (push at 05:11:55Z, completed by ~05:23:59Z per the log timestamps), consistent with CONTEXT.md's stated ~11.5-12 minute baseline for this job.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- GATE-04 and GATE-05 are now both evidenced in `.planning/research/compiler-performance/BASELINE.md`, closing the last two open items in `.planning/REQUIREMENTS.md`'s Verification section (GATE-01 through GATE-03 were already checked off; GATE-04 and GATE-05 checkboxes still read `[ ]` in REQUIREMENTS.md itself — updating those checkboxes was not in this plan's `files_modified` scope and is a candidate for the milestone-close/transition step).
- This is the last plan in Phase 15 (verification-decision-gate-flip); all three plans (15-01 gates flip, 15-02 operator docs, 15-03 milestone-close evidence) are now complete.
- No blockers for milestone completion.

---
*Phase: 15-verification-decision-gate-flip*
*Completed: 2026-09-09*

## Self-Check: PASSED

- FOUND: `.planning/research/compiler-performance/BASELINE.md` (modified, verified via `[ -f ]`)
- FOUND: commit `435069f` (Task 3) — verified via `git log --oneline --all | grep 435069f`
- Plan-level `<verification>` re-run:
  - `.planning/research/compiler-performance/BASELINE.md` carries the milestone-close section with run id `34313915733`, commit SHA `cbfe79c66e1d8adbe0042ff41bd1e9b90e4f1d24`, four labelled figures, all five GATE-04 fixture names, and the corpus caveats — confirmed present.
  - `git diff cbfe79c66e1d8adbe0042ff41bd1e9b90e4f1d24..435069f -- .planning/research/compiler-performance/BASELINE.md` shows additions only (104 insertions, 0 deletions).
  - GitHub Actions run `34313915733` reports `conclusion: success` (re-confirmed via `gh run view`).
  - `go test -race -timeout 300s -skip 'Test_cliCommand_Run' ./...` passes (0 `--- FAIL` lines, re-run locally after the commit).
