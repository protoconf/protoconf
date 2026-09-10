# Phase 15: Verification, Decision & Gate Flip - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-09-09
**Phase:** 15-verification-decision-gate-flip
**Areas discussed:** Where the 200ms budget is asserted, What the broken-proto decision decides, Shape of the real-corpus evidence, What "unchanged in outcome" proves

---

## Where the 200ms budget is asserted

### Q1 — Where should the wall-clock budget run?

Context given: CI runs `go test -race` with coverage, where a 400-proto corpus already takes 268ms locally against 34ms plain.

| Option | Description | Selected |
|--------|-------------|----------|
| Own non-race CI step | Second step in go.yml without -race and without coverage; the asserted number is the honest one. Costs a step and a second corpus generation. | ✓ |
| Proxy in CI, wall clock local | Race job asserts protos loaded or allocations; 200ms figure recorded as evidence rather than gated. Follows Phase 13's rejection of timing assertions. | |
| Wall clock with a race multiplier | One job, budget scaled for the race detector. Simplest wiring, but the number stops meaning 200ms. | |

**User's choice:** Own non-race CI step

### Q2 — Where should `TestCompilerStartupScaling` live once the non-race step exists?

Context given: its allocation ratio measured 0.91x plain and 1.02x under -race, so the race detector does not threaten it.

| Option | Description | Selected |
|--------|-------------|----------|
| Stays in the main race job | Race-tolerant and machine-independent; keeps its coverage contribution. New step stays narrowly the budget's home. | |
| Moves to the non-race step | Both perf gates in one findable place; stops paying the race detector's 8x slowdown (0.98s to 3.3s). Loses coverage contribution. | |
| You decide | Planner chooses on evidence from the test's runtime and the CI job layout. | ✓ |

**User's choice:** You decide — recorded as Claude's Discretion in CONTEXT.md

### Q3 — Should the budget test time the CLI binary or the in-process call?

Context given: PROJECT.md states the goal as "`protoconf compile` completes in under 200ms", which names the CLI, while the 35ms figure is the in-process library call.

| Option | Description | Selected |
|--------|-------------|----------|
| The compile binary end-to-end | Times a real `protoconf compile` invocation. Matches the goal statement, catches CLI-layer regressions. Costs a build step. | |
| In-process, matching the benchmark | Times NewCompiler + CompileFile, what compileCorpus already measures. Cheap and comparable to BASELINE.md. CLI wrapper cost unmeasured. | |
| In-process gate, CLI number recorded | Gate in-process in CI; record one real CLI timing as milestone evidence alongside GATE-05. Keeps CI light, goal statement still measured. | ✓ |

**User's choice:** In-process gate, CLI number recorded

### Q4 — What threshold should the CI assertion use?

Context given: a 200ms gate against a 35ms measurement has 5.7x headroom and will not fail until something regresses roughly fivefold.

| Option | Description | Selected |
|--------|-------------|----------|
| 200ms, the stated budget | Assert exactly what the milestone promised. Never flakes. Accepts that a 3x regression ships silently. | |
| 200ms gate plus a logged number | Hard-fail at 200ms and log the measurement unconditionally, so drift is visible before it breaches. No new flake surface. | |
| Tighter budget, say 100ms | Halve the gate to keep regression-detecting teeth. Still 3x over local, but a slow runner gets close to the line. | ✓ |

**User's choice:** Tighter budget, around 100ms

### Q5 — How to resolve the shared-runner flake risk at 100ms?

Context given: GitHub runners commonly land 2-3x slower, putting an 800-proto compile at 70-105ms — right on a 100ms line, which is the flake shape TESTING.md warns about.

| Option | Description | Selected |
|--------|-------------|----------|
| Calibrate on CI, then set it | Land the test logging its measurement, read the real runner number from an actual CI run, set the threshold at ~2x that. Costs one CI round-trip inside the phase. | ✓ |
| Take the minimum of three runs | Measure three times, assert the fastest, so one noisy hiccup cannot fail the build. Triples test runtime to ~100ms of work. | |
| Keep 100ms as a single measurement | Accept the risk. Simplest test; the phase's own CI run will show flakiness before merge. | |

**User's choice:** Calibrate on CI, then set it

---

## What the broken-proto decision decides

### Q1 — What is the disposition?

Context given: before lazy loading a broken proto no config referenced still failed the compile; under the lazy registry it is never parsed, so never reported.

| Option | Description | Selected |
|--------|-------------|----------|
| Accepted, no recovery path | Compiler reports on what a config reaches, full stop. Consistent with Phase 14's D-02. | |
| Accepted, with a named lever | Same default, but the write-up names a concrete way to check the whole tree on demand. | |
| Accepted, buf named as the answer | States that whole-tree proto validation is buf's job now, pointing at the existing lint workflow. No new code. | ✓ |

**User's choice:** Accepted, buf named as the answer

**Notes:** After the choice, a caveat was surfaced and carried into CONTEXT.md D-04: `buf.yaml` configures lint but `lint.yml` runs only `buf breaking`, and `buf.yaml` covers protoconf's own protos rather than an operator's config repository. buf is something an operator runs on their own tree, not something protoconf runs for them.

### Q2 — Where should the write-up live?

| Option | Description | Selected |
|--------|-------------|----------|
| CHANGELOG breaking changes | The Unreleased section already carries two BREAKING CHANGES entries in the same voice; upgraders read it by habit. | |
| CHANGELOG plus README | CHANGELOG catches upgraders, README catches newcomers who never read a changelog. Two places to keep consistent. | ✓ |
| README only | Document as standing behavior rather than a migration note. Upgraders get no signal. | |

**User's choice:** CHANGELOG plus README

### Q3 — Should a test pin the documented behavior?

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, a test pins both halves | One test for the unreferenced-broken case, one confirming a loaded broken proto still fails loudly. Follows the Phase 14 precedent on demonstrable controls. | |
| Yes, the negative half only | Pin just the changed behavior; the positive case is already covered. | |
| No, prose is enough | GATE-03 asks for a written decision, not a test. Phase 11's LAZY tests already establish that only reached protos are parsed. | ✓ |

**User's choice:** No, prose is enough

---

## Shape of the real-corpus evidence

### Q1 — Where should the milestone-close numbers land?

| Option | Description | Selected |
|--------|-------------|----------|
| Append to BASELINE.md | Already holds the 6.97s before-numbers with the same per-stage breakdown; makes the comparison immediate. Becomes a living doc. | ✓ |
| New evidence doc in the phase | 15-EVIDENCE.md citing BASELINE.md, leaving the baseline frozen as history. | |
| Both, with BASELINE.md pointing over | Full write-up in the phase doc plus an after-row and pointer in BASELINE.md. Two files, one thin. | |

**User's choice:** Append to BASELINE.md

### Q2 — What should verification check, given the sibling corpus may be absent?

| Option | Description | Selected |
|--------|-------------|----------|
| Check the recorded numbers | Assert BASELINE.md contains the after-section with a measured time, corpus path and proto count. Measurement is a one-time human run. | |
| Command that skips when absent | Measures when the sibling tree is present, exits cleanly with a message when not. A skip that always skips looks like a pass. | |
| Command that fails when absent | Corpus treated as required input. Makes the phase unverifiable on CI. | |
| **Other (free text)** | **Generate a large (~1000 proto files) corpus when running the gate.** | ✓ |

**User's choice:** Other — generate a corpus at gate time rather than depending on the sibling checkout.

**Notes:** The free-text answer conflicted with GATE-05's stated rationale, which names the real corpus precisely because the synthetic one is cheaper per file. Three resolutions were offered in plain text:

1. Generated corpus replaces it; GATE-05 amended to drop the real-corpus requirement.
2. Generated corpus is the gate, real corpus compiled once by hand as evidence.
3. Generated corpus only, sized so it costs what 799 real protos did — roughly 2400 files at the 3x gap.

**User selected 3.** Also surfaced at this point: under lazy loading, corpus size no longer moves the measured compile time (39ms at n=50, 35ms at n=500), so the calibration buys evidential weight rather than a harder gate, and its only real cost is generation time.

### Q3 — How should the calibrated corpus size be derived?

Context given: GATE-05's "~3x cheaper per file" appears nowhere in the research docs; the only 3.3x in the milestone measures unlinked-vs-linked parse, and `corpus.go` documents its tuning in the opposite direction.

| Option | Description | Selected |
|--------|-------------|----------|
| Match measured parse cost | Size so an unlinked parse over the synthetic tree costs BASELINE.md's recorded 1,286ms. No multiplier to trust. | |
| Measure the ratio, then apply it | Derive the real per-file multiplier, correct GATE-05's ~3x in REQUIREMENTS.md, size from that. | |
| Take ~3x at face value | Generate roughly 2400 protos and move on. Leaves an unsourced figure load-bearing. | ✓ |

**User's choice:** Take ~3x at face value

**Notes:** Recorded in CONTEXT.md D-08 as a deliberate accepted risk, with the reason exposure is low: because startup cost is flat against corpus size, an inaccurate multiplier changes the file count but barely changes the measured time or the gate's outcome.

---

## What "unchanged in outcome" proves

### Q1 — What is the suite of record for GATE-04?

Context given: CI runs the full `go test -race ./...` with no skip, while the GSD-local command skips `Test_cliCommand_Run`, which exists in four packages and hangs locally on Consul.

| Option | Description | Selected |
|--------|-------------|----------|
| The CI run | The green Go workflow on the phase's commit; full ./... under -race, covering the four cases local runs cannot. | |
| The pinned local command | The config.json test gate. Runnable by hand, but omits four tests from a gate claiming completeness. | |
| Both, CI authoritative | Pinned command as the runnable local verify, CI run as what the requirement closes against, gap stated explicitly. | ✓ |

**User's choice:** Both, CI authoritative

### Q2 — What proves the validator cases are unchanged in outcome?

Context given: one pre-existing compiler test is skipped rather than green — `load_remote_with_load_local.pconf`, skipped 2026-09-01 in commit 7fdffc4 for a stale vizceral_repo module pin, 244 commits back and unrelated to lazy loading. A flat "every pre-existing test stays green" claim overclaims by one.

| Option | Description | Selected |
|--------|-------------|----------|
| Green plus named exception | Suite passing is the proof, and the evidence names the one pre-existing skip with cause and date, so the claim is exact. | |
| Before and after comparison | Run the named tests at the pre-milestone commit and at HEAD, record both. Strongest evidence, costs a checkout dance. | |
| Green is enough | Assertions encode the expected outcomes, so passing is by definition unchanged. Skipped test stays unmentioned in the evidence. | ✓ |

**User's choice:** Green is enough

**Notes:** The skipped test is recorded in CONTEXT.md D-11 as context for the planner, explicitly not as a required element of the GATE-04 evidence.

---

## Claude's Discretion

- Where `TestCompilerStartupScaling` lives once the non-race CI step exists — main race job or the new step. Both defensible; decide on evidence from the test's runtime and the job layout.

## Deferred Ideas

- Add a `buf lint` step to `.github/workflows/lint.yml`. Out of scope: it lints protoconf's own protos, not an operator's config repository, which is the case GATE-03 is about.
- Repin `vizceral_repo` in the small testdata lock file to un-skip `load_remote_with_load_local.pconf`.
- Amend GATE-05's text in REQUIREMENTS.md to match D-07, and correct or source the ~3x multiplier per D-08. Offered and declined for this phase.
