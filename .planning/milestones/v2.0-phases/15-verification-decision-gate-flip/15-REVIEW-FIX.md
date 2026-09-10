---
phase: 15-verification-decision-gate-flip
fixed_at: 2026-09-09T05:47:33Z
review_path: .planning/phases/15-verification-decision-gate-flip/15-REVIEW.md
iteration: 1
findings_in_scope: 1
fixed: 1
skipped: 0
status: all_fixed
---

# Phase 15: Code Review Fix Report

**Fixed at:** 2026-09-09T05:47:33Z
**Source review:** .planning/phases/15-verification-decision-gate-flip/15-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 1 (fix_scope: critical_warning — CR-*/BL-*/WR-* only; IN-01 excluded by scope)
- Fixed: 1
- Skipped: 0

## Fixed Issues

### WR-01: New CI gate step verifies almost nothing except its own plumbing

**Files modified:** `.github/workflows/go.yml`
**Commit:** 46d7872
**Applied fix:** Replaced the fragile 3-stage pipe (`go test | tee /dev/stderr | grep -q ...`) with a 2-step sequence: `go test ... | tee test-budget.log` followed by a separate `grep -q 'startup budget: n=' test-budget.log`. Under `set -eo pipefail`, `go test`'s real exit code now fails the script immediately and independently — it is no longer at risk of being masked by `grep -q`'s early-exit `SIGPIPE`-ing `tee` before `tee` finishes writing. The grep line now serves as a genuine secondary sanity check ("did the right test actually run and log its result") rather than doubling as the sole pass/fail signal. This is the exact minimal replacement suggested in REVIEW.md's Fix section, applied verbatim since the CI file at the cited lines matched the review's description exactly.

No change to `budget`, `maxRatio`, or gate presence in `.github/workflows/go.yml` — the assertion strength is untouched, only the shell plumbing around it.

**Verification performed:**
- Tier 1: re-read the modified section of `.github/workflows/go.yml`; fix text present, surrounding steps (Build, Run coverage, Test report, Upload steps) intact.
- Tier 2: YAML parses cleanly (`js-yaml` load succeeded). `actionlint` was not available in this environment, so it was skipped per the verification_strategy fallback rule.
- Scratch reproduction (per phase_constraints), run in the isolated worktree and discarded after: built two synthetic scripts (`pass_test.sh` exit 0, `fail_test.sh` exit 1) each printing the `startup budget: n=` marker line before exiting, mimicking `t.Logf` running before `require.LessOrEqual`. Ran both the OLD pipeline and the NEW pipeline as actual `bash -e` scripts (not inline subshells, to match how GitHub Actions executes `run: |` blocks) against 4000 lines of noise plus the marker:
  - OLD pipeline: pass case exited 0, fail case exited 1 (worked at this synthetic volume, but the review's point stands — this depends on pipe-buffer timing, not correctness).
  - NEW pipeline: pass case exited 0, fail case exited 1 — correct and, unlike the old pipeline, not dependent on output volume or pipe-buffer race, since `go test`'s exit code is captured directly by `set -e` before `grep` ever runs.
  - All scratch files (`pass_test.sh`, `fail_test.sh`, `new_pipeline_pass.sh`, `new_pipeline_fail.sh`, `test-budget.log`) were removed after verification; the scratch directory was left clean.
- Verification ran in the isolated worktree (`.claude/worktrees/rf-15-18987-*`, since removed by cleanup), not the main checkout; the working tree there had no other changes beyond this fix, and the fast-forward step in cleanup makes this commit reproducible from the main checkout's `perf/compiler` branch after this run completes.

## Skipped Issues

### IN-01: TestCompilerStartupScaling's doc comment overstates what the test measures

**File:** `compiler/lib/startup_bench_test.go:106-110`
**Reason:** Out of scope — `fix_scope` for this run is `critical_warning`, which excludes Info-severity findings (IN-*). Not attempted.
**Original issue:** The comment claims the fixed bug covers "50 protos" through "5,000," but `TestCompilerStartupScaling` only measures and gates on n=50 vs n=400; n=5,000 is only exercised by the non-gating `BenchmarkCompilerStartup`. Recommend rewording in a future pass (e.g. with `fix_scope: all`) to reference the actual measured points or explicitly note 5,000 is benchmark-only.

---

_Fixed: 2026-09-09T05:47:33Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
