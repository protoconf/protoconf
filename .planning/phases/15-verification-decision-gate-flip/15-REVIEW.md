---
phase: 15-verification-decision-gate-flip
reviewed: 2026-09-09T00:00:00Z
depth: standard
files_reviewed: 5
files_reviewed_list:
  - .github/workflows/go.yml
  - CHANGELOG.md
  - README.md
  - compiler/lib/race_detector_test.go
  - compiler/lib/startup_bench_test.go
findings:
  critical: 0
  warning: 1
  info: 1
  total: 2
status: issues_found
---

# Phase 15: Code Review Report

**Reviewed:** 2026-09-09
**Depth:** standard
**Files Reviewed:** 5
**Status:** issues_found

## Summary

This phase flips two soft measurement gates into hard CI-blocking assertions: `TestCompilerStartupScaling` now `require.LessOrEqual`s an allocation ratio instead of `t.Skipf`-ing, and a new `TestCompilerStartupBudget` asserts a wall-clock budget, self-skipping under `-race`/`-short` via a build-tag-gated `raceEnabled` flip in `race_detector_test.go`. A new CI step wires the budget test into `.github/workflows/go.yml`, and `README.md`/`CHANGELOG.md` document the underlying lazy-proto-loading behavior change accurately (cross-checked against `.github/workflows/lint.yml`, which does run `buf breaking` and does not run `buf lint`, matching the docs' claim).

The `raceEnabled` build-tag mechanism, the calibration rationale for `budget`/`corpusProtos`, and the flip from `t.Skipf` to `require.LessOrEqual` are all sound and well-documented in code comments. The one substantive issue is in the new CI step's shell plumbing, which is fragile by construction even though it happens not to misfire at today's output volume (verified empirically below).

## Warnings

### WR-01: New CI gate step verifies almost nothing except its own plumbing

**File:** `.github/workflows/go.yml:30-33`
**Issue:**
```yaml
- name: Run startup budget gate
  run: |
    set -eo pipefail
    go test -run '^TestCompilerStartupBudget$' -count=1 -v ./compiler/lib/... | tee /dev/stderr | grep -q 'startup budget: n='
```
Two compounding problems:

1. The string this step greps for is printed unconditionally. In `compiler/lib/startup_bench_test.go:224-230`, `t.Logf("startup budget: n=%d ...")` executes *before* `require.LessOrEqual(t, elapsed, budget)`. So `grep -q 'startup budget: n='` matches whether the test **passes or fails** — the marker line is emitted either way. The only thing that actually distinguishes pass from fail here is `go test`'s own process exit code propagating correctly through a 3-stage pipe (`go test | tee | grep -q`) under `pipefail`.
2. `grep -q` exits as soon as it finds the first match, closing its end of the pipe while `tee` (and potentially `go test`) may still be writing. Under `set -o pipefail`, bash reports the exit status of the *rightmost pipeline stage that exited non-zero* — so if `tee` gets `SIGPIPE`'d by grep's early exit, the pipeline reports failure (141) via `tee`, not via `go test`, even when `go test` itself passed. I verified this empirically:
   - With ~4000 lines of output before/after the match, `grep -q` closing early reliably makes the pipeline exit 141 even though the pattern was found (i.e., a *passing* condition reported as a shell failure).
   - With the actual small output volume this test produces (~5-6 lines under `-run` filtering), the pipeline consistently exits 0, because `tee` finishes writing before `grep` has a chance to close the pipe early.

At today's output volume this happens to work, but it works by accident of pipe-buffer sizing, not by design. The check as written proves "the marker line appeared" (true on both pass and fail) plus "the pipe didn't get SIGPIPE'd" — it does not independently prove `TestCompilerStartupBudget` passed. A later change that increases verbosity (e.g. broadening `-run`, adding more `t.Logf` calls, or someone removing the `tee`/`pipefail` without noticing why they're there) can silently turn this gate into a no-op or into a source of CI flakes unrelated to the compiler's actual performance.

**Fix:** Don't rely on the exit code of a multi-stage pipe with an early-exiting `grep -q`. Capture output first, then assert on it, so `go test`'s exit code is checked directly and independently of the marker-line grep:
```bash
set -eo pipefail
go test -run '^TestCompilerStartupBudget$' -count=1 -v ./compiler/lib/... | tee test-budget.log
grep -q 'startup budget: n=' test-budget.log
```
Here `go test`'s failure fails the script immediately via `set -e` (it's the first statement in a normal pipe with no other consumer, and `tee` alone never induces `SIGPIPE` since it always finishes reading its input). The second line becomes a genuine independent sanity check ("did the right test actually run and log its result") rather than the sole signal doubling as a fragile pass/fail proxy.

## Info

### IN-01: TestCompilerStartupScaling's doc comment overstates what the test measures

**File:** `compiler/lib/startup_bench_test.go:106-110`
**Issue:** The comment states the fixed bug is "compiling a config that loads 5 protos should cost the same whether the repo contains 50 protos or 5,000," but the very next paragraph (and the code) only measures and gates on `n=50` vs `n=400` — 5,000 is never exercised by this test (only by `BenchmarkCompilerStartup`, which doesn't gate anything). This wording was carried over from before Phase 15 and only lightly reworded here, but since this diff touched the whole comment block, it's worth tightening: a future reader could believe `TestCompilerStartupScaling` provides coverage at n=5,000 that it doesn't.
**Fix:** Reword the opening sentence to reference the actual measured points (50/400), or explicitly note that 5,000 is only exercised by `BenchmarkCompilerStartup`, not this gating test.

---

_Reviewed: 2026-09-09_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
