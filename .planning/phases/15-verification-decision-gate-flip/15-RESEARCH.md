# Phase 15: Verification, Decision & Gate Flip - Research

**Researched:** 2026-09-09
**Domain:** Go testing/benchmarking, GitHub Actions CI wiring, technical documentation (CHANGELOG/README)
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Where the budget is asserted**

- **D-01:** The wall-clock budget gets its own non-race CI step. The existing `Run coverage` step in `.github/workflows/go.yml` runs `go test -race -coverprofile=... ./...`, and the race detector costs roughly 8x: a 400-proto corpus measured 34ms plain and 268ms under `-race`. A 200ms wall-clock assertion cannot live inside that job and mean anything. A second step runs the budget test without `-race` and without coverage, so the asserted number is the one an operator would actually see. Rejected: asserting a race-scaled budget in the existing job.
- **D-02:** The budget test times the in-process operation, not the CLI binary. It measures `NewCompiler` + `CompileFile` — the same operation `compileCorpus` (`compiler/lib/startup_bench_test.go:59`) already measures — so the figure is directly comparable to BASELINE.md's 6.97s breakdown and to the existing scaling test. PROJECT.md's "`protoconf compile` completes in under 200ms" names the CLI and includes Go runtime init/flag parsing/config loading the in-process figure excludes; that gap is closed by evidence rather than a heavier test: **one real `protoconf compile` invocation is timed by hand and its number recorded alongside the GATE-05 figures.** CI stays light.
- **D-03:** The gate threshold is tighter than the stated 200ms, targeting roughly 100ms, and the exact number is calibrated from an observed CI runner measurement rather than a developer laptop. A 200ms gate against a 35ms measurement has ~5.7x headroom and is nearly useless as a regression detector. Calibration is required, not optional: land the test logging its measurement unconditionally, read the real runner number from an actual CI run inside this phase, then set the threshold at roughly 2x that observed value. **A threshold picked from the numbers in this document, without a CI observation, is a planning error.** Any threshold at or below 200ms satisfies GATE-02 strictly. Reversibility: reversible — one constant in one test.

**The broken-proto behavior change**

- **D-04:** Accepted, with buf named as the answer. Before lazy loading, every proto under `src/` was parsed and linked at startup, so a syntax error in an unreferenced proto still failed the compile. Under the lazy registry it is never parsed, so it is never reported. The decision states plainly that whole-tree proto validation is now buf's job, not the compiler's. This is the operator-facing half of Phase 14's D-02 (first-request failure accepted, no startup validation pass). **Honest caveat the write-up must not paper over:** `buf.yaml` configures `lint` but `.github/workflows/lint.yml` runs only `buf breaking` — no `buf lint` step exists today. `buf.yaml` covers protoconf's own protos, not a downstream user's config repository. buf is something an operator runs against their own tree, not something protoconf already runs for them. Reversibility: reversible as a document; the underlying behavior (Phase 13's D-02 deleted the eager fallback) is one-way.
- **D-05:** The write-up lands in two places: the CHANGELOG breaking-changes section and a README section. The `## Unreleased` block in `CHANGELOG.md` already carries two `### ⚠ BREAKING CHANGES` entries from the CLI precedence work, in the same operator-facing voice. The README section catches newcomers who never read a changelog, documenting what `protoconf compile` does and does not validate as standing behavior. Both must stay consistent with each other.
- **D-06 (negative decision):** No test pins the documented behavior. GATE-03 asks for a written decision, not a test. Phase 11's LAZY tests already establish that only reached protos are parsed, so an unreferenced broken proto not failing the compile is a consequence of behavior already under test. Do not add a fixture for it.

**Real-corpus evidence**

- **D-07:** The gate generates its own calibrated corpus. Nothing depends on the sibling checkout. The real 799-proto protoconf-terraform corpus lives at `../protoconf-terraform/example/src` — present on the author's machine only, absent on CI, drifting independently. The gate generates a synthetic corpus at run time using `utils/testdata.GenerateCorpus`, sized so it **costs what the real 799 protos cost**, not merely has 800 files — using the ~3x per-file multiplier stated in GATE-05, roughly **2400 generated protos**. **This supersedes GATE-05's "compiled once at milestone close" wording.** The planner should treat GATE-05 as satisfied by the calibrated generated corpus and should surface the requirement-text mismatch rather than silently reinterpret it. Reversibility: costly — undoing reintroduces a dependency CI cannot satisfy.
- **D-08:** The ~3x multiplier is taken at face value, deliberately, despite being unsourced. GATE-05's "~3x cheaper per file" claim appears nowhere in `.planning/research/compiler-performance/`; the only 3.3x figure (`REQUIREMENTS.md:30`) measures something unrelated (unlinked vs. linked parse). `utils/testdata/corpus.go` documents its own tuning history in the opposite direction (47x, then 21x cheaper, now much closer to parity). Alternatives (size by BASELINE.md's 1,286ms unlinked-parse cost, or measure and correct the figure) were offered and declined. **Exposure is low:** because startup cost is flat against corpus size (39ms at n=50, 35ms at n=500), an inaccurate multiplier changes file count but barely changes measured time or the gate's outcome.
- **D-09:** The closing numbers are appended to `.planning/research/compiler-performance/BASELINE.md` as an after-section — the calibrated-corpus figure and the single real `protoconf compile` CLI timing from D-02. It becomes a living document rather than a frozen snapshot; that is accepted.

**Test-suite evidence**

- **D-10:** Both suites, CI authoritative. The pinned command `go test -race -timeout 300s -skip 'Test_cliCommand_Run' ./...` (`workflow.test_command` in `.planning/config.json`) is the runnable local verify. The green Go workflow run on the phase's own commit is what GATE-04 is closed against, because CI runs the full `./...` under `-race` with no skips, covering the four `Test_cliCommand_Run` cases (`inserter/`, `server/`, `agent/`, `compiler/`) the local command omits (they stall on Consul locally).
- **D-11:** A green run is sufficient proof of unchanged outcome for the validator cases — already encoded as assertions in `compiler/lib/compiler_test.go`: four cases expecting `ErrInvalidConfig` (`with_config_rollout_validator.pconf:41`, `validator_test.pconf:56`, `validator_repeated_test.pconf:57`, `validator_map_test.pconf:58`) and `validator_passing_test.pconf:59` expecting `nil`. No before/after comparison, no new assertions. **Finding recorded for completeness, deliberately not required in the evidence:** `load_remote_with_load_local.pconf` (`compiler/lib/compiler_test.go:47-51`) is skipped for a stale `vizceral_repo` module pin (244 commits back, predates this milestone). The planner should not add it to the evidence, but should not be surprised by it either.

### Claude's Discretion

The user explicitly deferred this. Decide it on evidence from the code and the CI job layout, not by asking again.

- **Where `TestCompilerStartupScaling` lives once the non-race step exists.** Either it stays in the main `-race` job — its allocation ratio measured 0.91x plain and 1.02x under `-race`, so the race detector does not threaten it, and it keeps its coverage contribution — or it moves to the new non-race step so both performance gates sit together and it stops paying the race detector's 8x slowdown (0.98s to 3.3s locally). Both are defensible; pick one and say why. *(This research's recommendation: move it — see Open Questions.)*

### Deferred Ideas (OUT OF SCOPE)

- **Add a `buf lint` step to `.github/workflows/lint.yml`.** `buf.yaml` configures lint but CI runs only `buf breaking`. Deliberately out of scope: it would lint protoconf's own protos, not an operator's config repository, which is the case GATE-03 is actually about. Adding it would not strengthen D-04's claim.
- **Repin `vizceral_repo` in the small testdata lock file** to un-skip `load_remote_with_load_local.pconf`. A stale upstream module pin from 2026-09-01, unrelated to this milestone.
- **Amend GATE-05's text in REQUIREMENTS.md** to match D-07, and correct or source the ~3x multiplier per D-08. Offered during discussion and declined for this phase; the mismatch is recorded in D-07 and D-08 instead.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| GATE-01 | `TestCompilerStartupScaling`'s allocation ratio at n=50 vs n=400 is at or below 2.0x, and its `t.Skipf` branch is replaced by `require.LessOrEqual` | Pattern 1; exact code at `compiler/lib/startup_bench_test.go:119-167`, verified this session |
| GATE-02 | Compiling the in-repo 800-proto synthetic corpus completes end-to-end in under 200ms, asserted in CI | Patterns 2-3, D-01/D-02/D-03/D-07 verified against `compileCorpus`, `GenerateCorpus`, and `go.yml`; Pitfall 1 covers the calibration sequencing risk |
| GATE-03 | The behaviour change to compile-time validation is decided deliberately and documented for operators | D-04/D-05/D-06 verified against `buf.yaml`/`lint.yml`/`CHANGELOG.md`/`README.md` current state; Pitfall 3 |
| GATE-04 | Every pre-existing test stays green, with the four validator cases and `field_type_any_test.pconf` unchanged in outcome | D-10/D-11 verified against `compiler_test.go:41,56-59,66` and `.planning/config.json`; Pitfall 3 |
| GATE-05 | The real protoconf-terraform corpus is measured once and recorded as milestone-close evidence, not a CI gate | D-07/D-08/D-09 verified against `utils/testdata/corpus.go` and `BASELINE.md`; superseded by the calibrated-generated-corpus approach per D-07 |
</phase_requirements>

## Summary

This phase has almost no design surface of its own — 15-CONTEXT.md already carries
locked decisions (D-01 through D-11) with exact file paths and line numbers, and
this research's job is to verify those citations against the actual files on disk
(not to explore alternatives) and surface the mechanical details a planner needs:
exact current test/CI content, what a new test must look like to slot into the
existing pattern, and one operational gap CONTEXT.md does not spell out — how the
D-03 CI-observed threshold actually gets read back into the repo.

All code citations in 15-CONTEXT.md were opened and verified this session; none
were wrong. `TestCompilerStartupScaling`'s `t.Skipf` is confirmed at
`compiler/lib/startup_bench_test.go:161-166`, `compileCorpus` at line 59, the sole
`Run coverage` step at `.github/workflows/go.yml:30-37`, the four validator cases
and `field_type_any_test.pconf` at `compiler/lib/compiler_test.go:41,56-59,66`, and
`buf.yaml`/`lint.yml`'s lint-configured-but-unrun gap.

**Primary recommendation:** Build the phase as five independent, mostly-parallel
tracks matching GATE-01 through GATE-05, plus the two housekeeping decisions
(D-05 write-up, D-09 BASELINE.md append). GATE-01 and GATE-02 touch the same file
(`startup_bench_test.go`) and the same CI file (`go.yml`) so they should land as
one plan; GATE-03/D-04/D-05 (the decision write-up) is pure prose touching
CHANGELOG.md and README.md; GATE-04 is an observation task (watch CI go green,
record it), not a code change; GATE-05/D-07/D-08/D-09 is one more test plus one
BASELINE.md edit. The one sequencing constraint the planner must encode explicitly:
**D-03's threshold cannot be chosen from this document — it requires landing the
test with logging first, triggering a real CI run, reading the observed number
from the run's log via `gh run view --log`, then committing the calibrated
constant as a second, separate change.** This is a two-commit gate, not a
one-shot write.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Allocation-ratio gate (GATE-01) | Test / CI | — | Pure Go test assertion, already exists, only the skip→assert flip changes |
| Wall-clock budget gate (GATE-02) | Test / CI | — | New Go test + new CI step; must NOT share the `-race` job (D-01) |
| Broken-proto decision (GATE-03) | Documentation | — | No code changes; a written decision plus operator-facing docs (D-04, D-05, D-06) |
| Test-suite evidence (GATE-04) | CI (observation) | — | Existing tests, existing pinned command; this phase observes and records, does not author new coverage |
| Real-corpus evidence (GATE-05) | Test / Documentation | — | Calibrated generated corpus (D-07) run once, number appended to BASELINE.md (D-09) — not a CI gate |

## Standard Stack

No new dependencies. This phase is Go stdlib `testing`/`runtime`, `testify`
(already a project dependency), GitHub Actions YAML, and Markdown — all already
in the project's stack per `CLAUDE.md`. No package installs, so the Package
Legitimacy Audit is not applicable.

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| New CI step in existing `build` job | New separate CI job | A separate job re-checks-out and re-builds from scratch, doubling build time for no benefit — CONTEXT.md D-01 says "step," not "job," and the existing job already has the checkout + Go setup a second step can reuse |
| `gh run view --log` for D-03 calibration | Manually reading the Actions web UI | `gh` CLI is installed and authenticated in this environment (verified: `gh version 2.86.0`, `gh auth status` shows logged in as `smintz`) — scriptable, and the number can be piped straight into the commit that sets the threshold |

## Package Legitimacy Audit

Not applicable — this phase installs no packages.

## Architecture Patterns

### System Architecture Diagram

```
   git push (branch)
        |
        v
  .github/workflows/go.yml  (on: push/pull_request)
        |
        +--> Step: Build                         (go build -v ./...)
        |
        +--> Step: Run coverage  [-race, existing]
        |         go test -race -coverprofile=... ./... -json > test_results.json
        |         includes: TestCompilerStartupScaling (allocation gate, GATE-01)
        |                    [Claude's Discretion: may move to the new step below]
        |
        +--> Step: Run startup budget  [NEW, non-race, D-01]
        |         go test -run <NewBudgetTest> ./compiler/lib/...
        |         generates calibrated corpus (D-07, ~2400 protos)
        |         asserts wall-clock <= threshold (D-03, calibrated post-hoc)
        |         --> GATE-02
        |
        +--> Step: Test report / Upload artifacts / Codecov [existing, unchanged]

   Separate, out-of-band (not CI):
   D-02: one manual `time protoconf compile <dir>` invocation
         --> recorded by hand into BASELINE.md's after-section (D-09)
   D-07/GATE-05: calibrated generated-corpus timing
         --> recorded by hand into BASELINE.md's after-section (D-09)

   Documentation-only path (GATE-03):
   CHANGELOG.md ## Unreleased / ### BREAKING CHANGES  <-- D-05 entry
   README.md    <-- D-05 standing-behavior section
   (both must describe the same fact: buf is the operator's tool, not protoconf's)
```

### Recommended Project Structure

No new directories. New/changed files only:

```
compiler/lib/
├── startup_bench_test.go   # GATE-01: delete t.Skipf (line 161-166) -> require.LessOrEqual
│                           # GATE-02: add new budget test calling compileCorpus over
│                           #          a D-07-calibrated corpus, gated on wall-clock
.github/workflows/
└── go.yml                  # D-01: add one new step, non-race, no coverage
CHANGELOG.md                 # D-05: new entry under ## Unreleased / ### BREAKING CHANGES
README.md                    # D-05: new standing-behavior section
.planning/research/compiler-performance/
└── BASELINE.md              # D-09: append after-section with calibrated GATE-02/05 numbers
```

### Pattern 1: Measure-then-log, gate on the deterministic signal

**What:** `TestCompilerStartupScaling` computes both an allocation ratio and a
wall-clock ratio every run, `t.Logf`s both unconditionally, but only asserts on
the allocation ratio — because allocation count is a deterministic function of
file count while wall-clock varies with the machine.

**When to use:** GATE-01's flip keeps this pattern exactly (delete `t.Skipf`,
add `require.LessOrEqual`, nothing else changes). GATE-02 is the one place this
phase deliberately **breaks** the pattern — asserting on wall-clock — which is
exactly why D-03 mandates the CI-observed calibration step: gating on a
non-deterministic signal without machine-specific calibration is the anti-pattern
the rest of the codebase avoids.

**Example (current code, verified this session):**
```go
// Source: compiler/lib/startup_bench_test.go:151-166
allocRatio := float64(alloc400) / float64(alloc50)
wallRatio := float64(elapsed400) / float64(elapsed50)

t.Logf("scaling n=50->400: alloc ratio=%.2fx (%d -> %d bytes), wall-clock ratio=%.2fx (%s -> %s)",
    allocRatio, alloc50, alloc400, wallRatio, elapsed50, elapsed400)

if allocRatio > maxRatio {
    t.Skipf("compiler startup does not yet scale with config size, not repo size: "+
        "alloc ratio=%.2fx exceeds target %.1fx (see .planning/research/compiler-performance/BASELINE.md); "+
        "this Skip is the lazy-loading milestone's definition of done — delete it, turning this into "+
        "require.LessOrEqual(t, allocRatio, maxRatio), once the ratio is in bounds", allocRatio, maxRatio)
}
```
GATE-01's task is literally: delete the `if allocRatio > maxRatio { t.Skipf(...) }`
block and replace it with `require.LessOrEqual(t, allocRatio, maxRatio)`.

### Pattern 2: `compileCorpus` as the single measured operation

**What:** Both `BenchmarkCompilerStartup` and `TestCompilerStartupScaling` call
the same unexported `compileCorpus(dir string) error` helper
(`compiler/lib/startup_bench_test.go:55-68`), which does exactly `NewCompiler` +
`CompileFile("main.mpconf")`. This is the operation BASELINE.md's 6.97s number
and the existing tests are already about.

**When to use:** GATE-02's new budget test and the D-02 manual CLI timing must
both measure something traceable to this same operation — the budget test by
calling `compileCorpus` directly (do not re-implement compile-and-time logic), the
CLI timing by wrapping `time protoconf compile <dir>` around the same underlying
call path (`compiler/command.go`'s `runLocally` → `compilerlib.NewCompiler` +
`compiler.CompileFile`, confirmed at `compiler/command.go:118-119`).

**Example (new test — not yet written, follows the established shape):**
```go
// Illustrative shape for GATE-02, modeled on TestCompilerStartupScaling's
// existing measure closure (startup_bench_test.go:126-144) and D-07's
// calibrated corpus size.
func TestCompilerStartupBudget(t *testing.T) {
    if testing.Short() {
        t.Skip("budget measurement is slow; skipped under -short")
    }

    const budgetMS = /* D-03: filled in AFTER a real CI observation, not now */

    dir := t.TempDir()
    require.NoError(t, testdata.GenerateCorpus(dir, calibratedN)) // D-07: ~2400

    start := time.Now()
    require.NoError(t, compileCorpus(dir))
    elapsed := time.Since(start)

    t.Logf("startup budget: n=%d compiled in %s (budget %dms)", calibratedN, elapsed, budgetMS)
    require.LessOrEqual(t, elapsed, time.Duration(budgetMS)*time.Millisecond)
}
```

### Pattern 3: Deterministic corpus generation, caller owns the directory

**What:** `testdata.GenerateCorpus(dir, n)` (`utils/testdata/corpus.go:26-53`) is
already the single generator used by every scaling/budget test in this file. It
is deterministic (fixed-seed `math/rand`), requires `n >= 5`, needs no lock file,
git repo, or `CONFIGSPACE` marker, and writes exactly `n` files under `src/` plus
a fixed `main.mpconf` that loads 5 of them regardless of `n`.

**When to use:** GATE-02's new test and any D-07 calibration script must call
this exact function, not a new one. The generator's own doc comment
(`utils/testdata/corpus.go:68-77`) already records its own accuracy history: two
earlier tuning passes were 47x then 21x cheaper per file than the real corpus
before the current message/field-option shape closed most of the gap — this is
the primary source for D-08's "~3x is unsourced but low-exposure" framing, not a
new finding.

### Anti-Patterns to Avoid

- **Asserting a race-scaled wall-clock budget in the existing `-race` job:** the
  400-proto corpus measured 34ms plain vs. 268ms under `-race` (an ~8x factor,
  recorded in CONTEXT.md's pre-session measurement table) — a "200ms" assertion
  inside that job would not mean 200ms to anyone reading it later. D-01 rejects
  this explicitly.
- **Choosing the D-03 threshold from a number in this document or in
  CONTEXT.md:** every number recorded so far (39ms/35ms benchmark, 34ms/268ms
  scaling) was measured on the author's laptop, not the CI runner. GitHub-hosted
  `ubuntu-latest` runners are commonly 2-3x slower than a modern laptop for
  CPU-bound Go workloads — using a laptop number here would either be a
  false-negative-prone threshold (too tight, flakes) or too loose to catch
  regressions, and this repo's own convention (`TESTING.md`, cited in
  CONTEXT.md D-01) is that a flaky threshold gets bumped until it is worthless.
  Land the test logging only, observe the real CI number, then set the constant.
- **Re-implementing the "compile and time it" logic instead of calling
  `compileCorpus`:** would silently decouple the new gate from the same
  operation BASELINE.md's numbers describe, breaking D-02's comparability
  requirement.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Synthetic proto corpus for GATE-02/GATE-05 | A new generator, a checked-in fixture tree | `testdata.GenerateCorpus(dir, n)` | Already deterministic, already the shared generator for every startup-cost test in this file; a second generator would immediately diverge in per-file cost from the one BASELINE.md's ~3x multiplier (D-08) is even loosely anchored to |
| Reading a CI run's logged number | Manually opening the Actions web UI and copy-pasting | `gh run view <run-id> --log` (or `gh run watch` while the push is in flight) | `gh` CLI verified installed and authenticated in this environment; scriptable and reproducible, avoids transcription error on the one number D-03 depends on |
| Timing a real `protoconf compile` invocation (D-02) | A new instrumented CLI flag or a wrapper script | Plain shell `time protoconf compile <dir>` after `go build ./cmd/protoconf` | D-02 explicitly wants "one real invocation timed by hand" — building measurement infrastructure for a number that is recorded once and never re-run would be scope creep the decision itself rejects |

**Key insight:** every piece of infrastructure this phase needs (corpus
generator, measured operation, CI job) already exists from Phases 11-14. The
entire phase is wiring and one calibration step, not new mechanism — this
matches CONTEXT.md's own framing ("No new compiler mechanism ships here").

## Common Pitfalls

### Pitfall 1: Setting the D-03 threshold before observing real CI

**What goes wrong:** A planner reads the 39ms/35ms benchmark numbers or the
34ms/268ms scaling numbers in this document or CONTEXT.md and picks "100ms" or
"200ms" directly, skipping the CI-observation step.

**Why it happens:** The numbers are right there, already measured, and look
authoritative — but they were measured on darwin/arm64 hardware, not
`ubuntu-latest`.

**How to avoid:** Structure the GATE-02 work as two commits: (1) land the test
with the assertion either absent or set to an obviously-loose placeholder
(e.g., a comment marking it TODO, or `t.Logf`-only, matching this codebase's own
measure-then-log convention), open a PR / push to trigger `go.yml`, pull the
logged number with `gh run view --log`, then (2) commit the calibrated threshold
as `roughly 2x that observed value` per D-03. Do not let these collapse into one
commit with a guessed number.

**Warning signs:** A PLAN.md task that writes both the test body and a specific
millisecond constant in the same task, with no intervening "read CI" step.

### Pitfall 2: Moving `TestCompilerStartupScaling` without checking coverage impact

**What goes wrong:** The Claude's Discretion item (moving the scaling test out
of the `-race` job) is chosen without checking whether that test's code paths
are still exercised by something else in the `-race` job — if not, Codecov's
project coverage could measurably drop.

**Why it happens:** The test itself doesn't disappear, but the `-race` job's
coverage run (`go test -race -coverprofile=...`) only counts code paths hit by
tests it runs. Moving the test to the non-race step (which per D-01 runs "without
coverage") removes its contribution.

**How to avoid:** `TestGeneratedCorpusCompiles` (same file, `startup_bench_test.go:23-53`)
already exercises `NewCompiler` + `CompileFile` + `LoadedFileCount()` through the
same `compileCorpus`-adjacent code path and stays in the `-race` job either way —
the scaling test's unique contribution is the ratio computation itself, a small,
already-tested code region. This makes moving it low-risk, and is the
recommendation this research offers (see Open Questions) — but the planner should
still state this reasoning explicitly per CONTEXT.md's "pick one and say why."

### Pitfall 3: Forgetting `field_type_any_test.pconf` is a GATE-04 named case, not incidental

**What goes wrong:** GATE-04's evidence write-up lists "the four validator cases"
from memory and treats `field_type_any_test.pconf` as just another one of the ~24
other `compilerTest` cases in the same table, rather than a fifth explicitly-named
case (GATE-04's own text: "the four validator cases and `field_type_any_test.pconf`").

**Why it happens:** It sits in the same `t.Run(...)` table
(`compiler/lib/compiler_test.go:66`) as everything else, with no visual
distinction.

**How to avoid:** When writing the GATE-04 evidence, cite all five cases by name:
`with_config_rollout_validator.pconf:41`, `validator_test.pconf:56`,
`validator_repeated_test.pconf:57`, `validator_map_test.pconf:58` (all expect
`ErrInvalidConfig`), plus `field_type_any_test.pconf:66` (expects `nil`) —
verified present in the file this session, all five currently pass.

## Code Examples

### Current CI step this phase must not disturb
```yaml
# Source: .github/workflows/go.yml:30-41 (verified this session)
      - name: Run coverage
        run: |
          GOROOT="$(go env GOROOT)"
          export GOROOT
          go test -race -coverprofile=coverage.txt -covermode=atomic -v ./... -json > test_results.json

      - name: Test report
        if: always()
        run: go run gotest.tools/gotestsum@v1.13.0 --format testdox --raw-command -- cat test_results.json
```

### New step shape for D-01 (illustrative — exact test name is the planner's/executor's call)
```yaml
      - name: Run startup budget gate
        run: go test -run TestCompilerStartupBudget -v ./compiler/lib/...
```
No `-race`, no `-coverprofile`. Placed as a new step in the existing `build` job,
after `Run coverage` (or before — order does not matter since neither step
depends on the other's output), reusing the same job's checkout and Go setup.

### Reading the observed CI number for D-03
```bash
# After pushing a commit that lands the budget test with logging:
gh run list --branch <branch> --limit 1
gh run view <run-id> --log | grep "startup budget"
```

## State of the Art

Not applicable in the usual sense — this is a project-internal verification
phase, not adopting an external library or framework. The one "state of the art"
fact worth recording: GitHub-hosted `ubuntu-latest` runners are commonly
documented (GitHub's own runner specs) as 2-core/7GB shared VMs, materially
weaker than a modern Apple Silicon laptop for single-threaded CPU-bound Go work
— this is the underlying reason D-03 requires an observed-not-guessed threshold.
`[ASSUMED]` — general knowledge of GitHub Actions runner specs, not verified
against this repo's actual observed runtime this session (that observation is
exactly the task D-03 assigns to plan execution, not to research).

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `ubuntu-latest` GitHub-hosted runners are ~2-3x slower than the author's darwin/arm64 laptop for this workload | State of the Art, Pitfall 1 | Low — D-03 already treats this as a reason to observe rather than guess; if the real ratio differs, the calibration step self-corrects since it reads the actual number, this assumption only motivates *why* the step exists, not the threshold value itself |
| A2 | Moving `TestCompilerStartupScaling` to the non-race step is unlikely to measurably drop Codecov coverage, because `TestGeneratedCorpusCompiles` exercises adjacent code in the same file and stays in the `-race`/coverage job | Pitfall 2 | Medium — if wrong, a coverage regression could trip Codecov's project threshold (currently 1%, per STATE.md quick-task `260902-cov`); the planner should verify with a local coverage diff before/after rather than trust this assumption outright |

## Open Questions

1. **Where should the new GATE-02 budget test physically live — a new file or appended to `startup_bench_test.go`?**
   - What we know: `compileCorpus`, the generator, and the two existing
     tests/benchmark are all in `compiler/lib/startup_bench_test.go`.
   - What's unclear: whether the planner should add a new test function to that
     same file (keeping all startup-cost tests together) or create a sibling
     file (e.g. `startup_budget_test.go`) to keep the "milestone-definition-of-done
     gate" (GATE-01, allocation) separate from the "new CI-gated wall-clock
     budget" (GATE-02).
   - Recommendation: same file. Both tests already share `compileCorpus`; a
     second file would just re-import the same helpers with no functional
     benefit, and CONTEXT.md's canonical-refs section treats
     `startup_bench_test.go` as the one home for this concern.

2. **Should `TestCompilerStartupScaling` move to the non-race step (Claude's Discretion)?**
   - What we know: 0.91x plain / 1.02x under `-race` (no race-detector risk to
     the assertion itself); moving it saves the 8x race-detector slowdown
     (0.98s→3.3s locally per CONTEXT.md); staying keeps its small coverage
     contribution.
   - What's unclear: the exact coverage delta on this specific repo's Codecov
     report — not measured this session.
   - Recommendation: move it to the new non-race step, alongside the new
     GATE-02 budget test — both are now "performance gates that must not pay
     the race detector's cost," and grouping them documents that shared reason
     in one place rather than splitting a single narrative across two CI steps.
     State this reasoning explicitly in the plan per CONTEXT.md's instruction
     to "pick one and say why."

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | All test/build work | ✓ | go1.25.8 darwin/arm64 | — |
| `gh` CLI | D-03 CI-log observation | ✓ | 2.86.0, authenticated as `smintz` | Manual Actions web UI read |
| GitHub Actions (`.github/workflows/go.yml`) | GATE-02, D-01, D-03 | ✓ (repo has working CI) | — | — |
| Real protoconf-terraform corpus (`../protoconf-terraform/example/src`) | GATE-05's literal text (not this phase's chosen approach) | Not checked this session — irrelevant per D-07, which supersedes dependence on it | — | D-07: generated calibrated corpus, no external checkout needed |

**Missing dependencies with no fallback:** none.

**Missing dependencies with fallback:** the real sibling-checkout corpus is
explicitly not required by this phase's chosen approach (D-07); no fallback
needed because nothing depends on it.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` + `github.com/stretchr/testify v1.9.0` |
| Config file | none — no `.golangci.yml`; test behavior controlled by flags |
| Quick run command | `go test -run TestCompilerStartupBudget ./compiler/lib/...` (new test, non-race) |
| Full suite command | `go test -race -timeout 300s -skip 'Test_cliCommand_Run' ./...` (pinned in `.planning/config.json`, `workflow.test_command`) |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| GATE-01 | Allocation ratio n=50 vs n=400 <= 2.0x, hard-asserted | unit | `go test -run TestCompilerStartupScaling ./compiler/lib/...` | ✅ (`startup_bench_test.go:119-167`, flip `t.Skipf`→`require.LessOrEqual`) |
| GATE-02 | Calibrated corpus compiles under CI-observed wall-clock budget | unit (new) | `go test -run TestCompilerStartupBudget ./compiler/lib/...` | ❌ — new test, this phase writes it |
| GATE-03 | Decision documented (not a test — D-06 explicitly rejects adding a fixture) | documentation | n/a | n/a — CHANGELOG.md + README.md prose |
| GATE-04 | Full pre-existing suite green, validator cases + `field_type_any_test.pconf` unchanged | integration (existing) | `.planning/config.json`'s pinned command locally; full `./...` under `-race` in CI is authoritative per D-10 | ✅ (`compiler_test.go:41,56-59,66`, all pre-existing) |
| GATE-05 | Calibrated real-corpus-equivalent evidence recorded, not gated | manual/one-shot | run once, hand-record into BASELINE.md | ❌ — new generation call + BASELINE.md edit |

### Sampling Rate
- **Per task commit:** the pinned local command from `.planning/config.json`
  (`go test -race -timeout 300s -skip 'Test_cliCommand_Run' ./...`) — this
  already excludes the Consul-hanging `Test_cliCommand_Run` cases per
  MEMORY.md's recorded learning.
- **Per wave merge:** same command, plus a manual run of the new
  `TestCompilerStartupBudget` if it was just added/changed (it may be excluded
  from the fast local loop if its corpus generation is slow — verify at
  execution time).
- **Phase gate:** the green `go.yml` Actions run on the phase's own commit is
  what GATE-04 is closed against (D-10) — not the local pinned command alone,
  since only CI runs the full `./...` with no skips and covers
  `Test_cliCommand_Run`.

### Wave 0 Gaps
None — existing test infrastructure (`compileCorpus`, `testdata.GenerateCorpus`,
the `runtime.MemStats` measure pattern) covers everything GATE-01/02/05 need;
GATE-03/04 need no test infrastructure at all.

## Security Domain

Not applicable to this phase — no new attack surface, no new input parsing, no
new authentication/authorization/crypto code. This phase adds a CI step, flips
one assertion, writes documentation, and appends to a Markdown file. Per the
verification protocol, `security_enforcement` in `.planning/config.json` is
absent (default enabled), but there is no ASVS category with a plausible finding
here — omitted rather than padded with N/A rows.

## Sources

### Primary (HIGH confidence — files opened and verified this session)
- `compiler/lib/startup_bench_test.go` — full file read; `TestCompilerStartupScaling`, `compileCorpus`, `BenchmarkCompilerStartup`, `TestGeneratedCorpusCompiles`
- `compiler/lib/compiler_test.go` (lines 1-70) — the five GATE-04 named cases
- `.github/workflows/go.yml` — full file, the single `Run coverage` step
- `.github/workflows/lint.yml` — full file, confirms only `buf breaking` runs, no `buf lint`
- `buf.yaml` — full file, confirms `lint: use: [DEFAULT]` is configured but unrun
- `utils/testdata/corpus.go` — full file, `GenerateCorpus`, shape constants, tuning-history comment (source for D-08's framing)
- `.planning/research/compiler-performance/BASELINE.md` — full file, the 6.97s/4,639ms/197ms breakdown D-09 appends to
- `.planning/config.json` — `workflow.test_command` pinned value
- `CHANGELOG.md` (lines 1-40) — `## Unreleased` / `### BREAKING CHANGES` structure D-05's entry joins
- `README.md` (lines 1-100, headings scan) — no existing "what compile validates" section; D-05's section is new
- `compiler/command.go` (grep) — `runLocally` → `compilerlib.NewCompiler` + `CompileFile`, the D-02 CLI timing's underlying call path
- `utils/utils.go:505-509`, and 15+ test call sites — `LoadedFileCount()` exists and is already the established accessor (relevant background, not itself a GATE)
- Shell verification: `gh version 2.86.0`, `gh auth status` (logged in), `go version go1.25.8 darwin/arm64`

### Secondary (MEDIUM confidence)
- 15-CONTEXT.md itself — all D-01 through D-11 decisions treated as locked per the user-constraints contract; not re-litigated

### Tertiary (LOW confidence)
- GitHub-hosted `ubuntu-latest` runner hardware specs (2 vCPU) — general knowledge, not verified against this repo's actual CI run this session (see Assumptions Log A1)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependencies, all existing project tooling verified by reading source
- Architecture: HIGH — every pattern cited was read from the actual file this session, not recalled
- Pitfalls: HIGH for Pitfalls 1 and 3 (directly derived from verified file contents and CONTEXT.md's explicit calibration requirement); MEDIUM for Pitfall 2 (coverage-impact claim not measured this session, flagged in Assumptions Log)

**Research date:** 2026-09-09
**Valid until:** Effectively the life of this phase — this is a project-internal verification phase with no external-library drift risk; the only perishable fact (CI runner relative speed) is resolved by D-03's own observation step, not by this document.

## Project Constraints (from CLAUDE.md)

- **Tech stack:** Go 1.25.8+; verified this session (`go version go1.25.8 darwin/arm64`)
- **Build:** `CGO_ENABLED=0`, must produce static binaries — unaffected by this phase (no build changes, only test/CI/docs)
- **Testing:** must not break existing CI (GitHub Actions with Codecov) — directly governs D-01's "new step, not a replacement" framing and Pitfall 2's coverage-impact concern
- **Naming:** test files use `_test.go` suffix co-located with source — governs Open Question 1 (new GATE-02 test belongs in `compiler/lib/`, following existing `startup_bench_test.go` conventions)
- **Error handling / logging conventions:** not implicated — this phase adds no new production error paths or log lines
- **GSD Workflow Enforcement:** file-changing work must go through a GSD entry point (`/gsd:execute-phase` for this planned phase work) — the planner should assume execution happens under `/gsd:execute-phase`, not ad hoc edits
