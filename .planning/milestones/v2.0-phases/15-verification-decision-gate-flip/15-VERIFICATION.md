---
phase: 15-verification-decision-gate-flip
verified: 2026-09-09T12:45:00Z
status: passed
score: 5/5 must-haves verified
covered_files: [".github/workflows/go.yml", ".planning/REQUIREMENTS.md", ".planning/phases/15-verification-decision-gate-flip/15-01-PLAN.md", ".planning/phases/15-verification-decision-gate-flip/15-01-SUMMARY.md", ".planning/phases/15-verification-decision-gate-flip/15-02-PLAN.md", ".planning/phases/15-verification-decision-gate-flip/15-02-SUMMARY.md", ".planning/phases/15-verification-decision-gate-flip/15-03-PLAN.md", ".planning/phases/15-verification-decision-gate-flip/15-03-SUMMARY.md", ".planning/research/compiler-performance/BASELINE.md", "CHANGELOG.md", "README.md", "compiler/lib/race_detector_test.go", "compiler/lib/startup_bench_test.go"]
covered_digest: "v1:sha256:0345bc877822f728fa922fd1f792ce0ed1a53afca242dd3e42d2262bdbf8b801"
behavior_unverified: 0
overrides_applied: 0
---

# Phase 15: Verification, Decision & Gate Flip — Verification Report

**Phase Goal:** The milestone's definition of done is measured and asserted in CI, and the one deliberate behavior change is written down rather than silently absorbed.
**Verified:** 2026-09-09
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (ROADMAP Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | `TestCompilerStartupScaling`'s allocation ratio at n=50 vs n=400 is at or below 2.0x, asserted with `require.LessOrEqual` — the `t.Skipf` branch is gone | ✓ VERIFIED | `compiler/lib/startup_bench_test.go:170` reads `require.LessOrEqual(t, allocRatio, maxRatio)`; no `t.Skipf` remains in the function. Re-ran `go test -race -timeout 300s -skip 'Test_cliCommand_Run' ./...` locally — full suite green. CI run 34313915733 (success) logged `alloc ratio=1.02x`. |
| 2 | Compiling the in-repo synthetic corpus completes end-to-end in under 200ms, asserted in CI | ✓ VERIFIED (documented supersession) | `.github/workflows/go.yml:30-33` step `Run startup budget gate` runs `TestCompilerStartupBudget` (corpus size 2400, not 800) with `require.LessOrEqual(t, elapsed, budget)`, `budget = 160ms`. BASELINE.md's "GATE-05's requirement-text mismatch" and the code's own comment ("GATE-02's text names a 200ms budget over an 800-proto corpus... satisfied strictly") state plainly that the 800/200ms wording is superseded by a calibrated, stricter 2400-proto/160ms gate (D-07/D-08). CI-observed figures: 77.58ms and 93.4ms on `ubuntu-latest`, both well under 160ms. |
| 3 | A written decision exists explaining whether `protoconf compile` still reports a broken proto that no config loads, and why | ✓ VERIFIED | `CHANGELOG.md` `## Unreleased`/`### ⚠ BREAKING CHANGES` gained a `**compiler:**` bullet; `README.md` gained `## What \`protoconf compile\` validates` (verified at line 218, between `## Quick start` and `## Production setup`). Both state the behavior, the consequence, the `buf`/`buf lint` remedy, and the `buf breaking`/`buf.yaml` caveat, and agree with each other. |
| 4 | The full pre-existing test suite passes, including the four validator cases and `field_type_any_test.pconf`, with unchanged outcomes | ✓ VERIFIED | Re-ran `go test -race -timeout 300s -skip 'Test_cliCommand_Run' ./...` myself — all packages `ok`, zero `FAIL`. `compiler/lib/compiler_test.go` confirms the five named cases (`with_config_rollout_validator.pconf`, `validator_test.pconf`, `validator_repeated_test.pconf`, `validator_map_test.pconf` → `ErrInvalidConfig`; `field_type_any_test.pconf` → `nil`) are present unchanged. BASELINE.md's GATE-04 table and a live `gh run view 34313915733` (conclusion: success, headSha `cbfe79c...`) corroborate all five PASS in the authoritative `-race ./...` CI run. |
| 5 | The real protoconf-terraform corpus (799 protos, 6.97s at baseline) is compiled once and its end-to-end time is recorded as milestone-close evidence, not a CI gate | ✓ VERIFIED (documented supersession) | BASELINE.md's `## Milestone close (2026-09-09)` section is appended (confirmed: all 7 pre-existing `##` headings and their exact before-figures — `6.97s`, `4,639ms`, `1,286ms`, `a ~1,700x gap` — survive unedited). The section plainly states (in its own "GATE-05's requirement-text mismatch" subsection) that the real 799-proto sibling checkout was not compiled; a calibrated 2400-proto `GenerateCorpus` corpus was used instead per D-07, with the ~3x multiplier disclosed as unsourced (D-08). This is a deliberate, explicitly-stated supersession, not a silent gap. |

**Score:** 5/5 truths verified (0 present-but-behavior-unverified)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `compiler/lib/startup_bench_test.go` | asserting `TestCompilerStartupScaling`, new `TestCompilerStartupBudget` | ✓ VERIFIED | Both present, both use `require.LessOrEqual`; empirically re-ran with a temporarily lowered `budget` (1ms) — pipeline correctly reported `--- FAIL` and non-zero exit; restored cleanly (`git status` clean afterward). |
| `compiler/lib/race_detector_test.go` | `//go:build race` guard | ✓ VERIFIED | File exists, sets `raceEnabled = true` in `init()`. |
| `.github/workflows/go.yml` | new non-race `Run startup budget gate` step | ✓ VERIFIED | Step present between `Build` and `Run coverage`; `Run coverage` step byte-identical to before (one `-coverprofile` occurrence in the file). |
| `CHANGELOG.md` | breaking-change bullet | ✓ VERIFIED | Third bullet under `### ⚠ BREAKING CHANGES`, two pre-existing `**cli:**` bullets untouched. |
| `README.md` | standing-behavior section | ✓ VERIFIED | New `## What \`protoconf compile\` validates` section present and correctly positioned; all 7 pre-existing headings intact. |
| `.planning/research/compiler-performance/BASELINE.md` | milestone-close section | ✓ VERIFIED | `## Milestone close (2026-09-09)` appended before `## Reproduction`; diff is additions-only (git history: 104 insertions, 0 deletions on that commit). |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `Run startup budget gate` CI step | `TestCompilerStartupBudget` | `go test -run '^TestCompilerStartupBudget$' ... \| tee ... \| grep -q 'startup budget: n='` | ✓ WIRED (with caveat, see Anti-Patterns) | Confirmed the pipeline propagates `go test`'s real exit code in both directions: re-ran with `budget` temporarily set to 1ms — pipeline exited non-zero and printed `--- FAIL`; restored to 160ms — clean. |
| `TestCompilerStartupBudget` | `compileCorpus` | same call path as `TestCompilerStartupScaling`/`BenchmarkCompilerStartup` (D-02) | ✓ WIRED | `compileCorpus(dir)` called directly, no reimplementation. |
| CHANGELOG bullet | README section | must describe the same four facts | ✓ WIRED | Cross-checked text: both name `buf`, `buf breaking`, `buf lint`, `buf.yaml`, and agree on behavior/consequence/remedy/caveat. |
| BASELINE.md GATE-04 evidence | GitHub Actions run 34313915733 | cited run id + commit SHA | ✓ WIRED | `gh run view 34313915733 --json conclusion,headSha,status` independently confirms `{"conclusion":"success","headSha":"cbfe79c66e1d8adbe0042ff41bd1e9b90e4f1d24","status":"completed"}`, matching BASELINE.md verbatim. |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Allocation gate has teeth | (from 15-01-SUMMARY, re-verified structurally) | `require.LessOrEqual` present, `t.Skipf` absent | ✓ PASS |
| Wall-clock gate has teeth under the exact CI pipeline shape | `budget` temporarily set to 1ms; ran `go test -run '^TestCompilerStartupBudget$' ... \| tee /dev/stderr \| grep -q 'startup budget: n='` | `--- FAIL: TestCompilerStartupBudget`, pipeline exit code 1 | ✓ PASS |
| Full pre-existing suite green | `go test -race -timeout 300s -skip 'Test_cliCommand_Run' ./...` | All packages `ok`, zero `FAIL` | ✓ PASS |
| Cited CI run is real and successful | `gh run view 34313915733 --json conclusion,headSha,status` and `gh run view 34312220627 --json ...` | Both `conclusion: success`, headSha matches claims | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| GATE-01 | 15-01 | Allocation ratio gate flipped to assertion | ✓ SATISFIED | `require.LessOrEqual` in place, demonstrated capable of failing |
| GATE-02 | 15-01 | Wall-clock budget asserted in CI | ✓ SATISFIED | `TestCompilerStartupBudget` + CI step, calibrated from real `ubuntu-latest` observations |
| GATE-03 | 15-02 | Written decision on broken-proto compile-time validation change | ✓ SATISFIED | CHANGELOG.md + README.md, mutually consistent |
| GATE-04 | 15-03 | Pre-existing test suite green, five named fixtures unchanged | ✓ SATISFIED | Local re-run + cited CI run 34313915733 (independently confirmed) |
| GATE-05 | 15-03 | Real-corpus figure recorded as milestone-close evidence | ✓ SATISFIED (documented supersession) | BASELINE.md milestone-close section, mismatch explicitly stated |

No orphaned requirements: `.planning/REQUIREMENTS.md`'s traceability table maps all five GATE-01..05 to Phase 15 as Complete, and all five appear in at least one plan's `requirements:` frontmatter.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `.github/workflows/go.yml` | 30-33 | `go test \| tee \| grep -q` pipeline whose pass/fail signal formally depends on bash pipefail semantics rather than a directly-checked exit code | ⚠️ Warning (carried from `15-REVIEW.md` WR-01, independently re-verified) | I reproduced the failure scenario directly (temporarily lowered `budget` to 1ms and ran the exact CI command): the pipeline correctly reported `--- FAIL` and exited non-zero. The theoretical SIGPIPE-masking risk the code review flagged only manifests at output volumes far larger (~4000 lines) than this test currently produces (~10-15 lines). Confirmed not a live defect today, but fragile by construction — a future increase in verbosity could silently change this gate's semantics. Not a blocker; the phase's own code-review artifact (`15-REVIEW.md`) already surfaces this with a concrete fix (capture-then-grep instead of pipe-and-grep). |

No `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` markers found in any file modified by this phase.

### Documentation Consistency Note (non-blocking)

`.planning/ROADMAP.md` line 61 still shows Phase 15's top-level checkbox as `[ ]` (unchecked) even though all three of its plans are individually checked `[x]` and `.planning/REQUIREMENTS.md` marks all five GATE requirements `[x]` Complete. This is a bookkeeping item, not a goal-achievement gap — the underlying code, docs, and evidence are all in place. Flagged for whoever runs the milestone-close/phase-completion step.

### Human Verification Required

None. All must-haves resolved to VERIFIED status through direct codebase inspection, independent re-execution of the pinned test command, an independent re-run of the failure-mode demonstration, and independent confirmation (via `gh`) of the two cited GitHub Actions run IDs.

### Gaps Summary

No gaps. All five ROADMAP success criteria are met — three directly, two (GATE-02's 800-proto/200ms wording and GATE-05's real-corpus requirement) via explicitly documented, plainly-stated supersessions (D-07/D-08) rather than silent reinterpretation, exactly as this phase's own goal demands ("the one deliberate behavior change is written down rather than silently absorbed" — and here, the phase applies that same discipline to its own two requirement-text deviations). One pre-existing code-review warning (CI pipe/grep fragility) was independently re-verified as non-blocking today.

---

_Verified: 2026-09-09_
_Verifier: Claude (gsd-verifier)_
