# Phase 15: Verification, Decision & Gate Flip - Context

**Gathered:** 2026-09-09
**Status:** Ready for planning

<domain>
## Phase Boundary

This phase turns the v2.0 milestone's definition of done into assertions that
run, evidence that is written down, and one behavior change that is stated
rather than silently absorbed. It covers GATE-01 through GATE-05.

**No new compiler mechanism ships here.** Phases 11-14 delivered the lazy
registry, growable resolver views, the exact symbol index and non-compiler
consumer correctness. Phase 15 measures that work, asserts it in CI, documents
its one operator-visible consequence, and closes the milestone.

**Measured this session, before any decisions were taken** — both gates are
already met on the mechanism side, which is why this phase is about how they
get asserted, not whether they pass:

| Measurement | Result |
|---|---|
| `TestCompilerStartupScaling` alloc ratio, n=50 to n=400, plain | 0.91x (target ≤ 2.0x) |
| Same, under `-race` | 1.02x |
| `compileCorpus` wall clock, n=400, plain | 34ms |
| Same, under `-race` | 268ms |
| `BenchmarkCompilerStartup` n=50 | 39ms / 595,780 allocs |
| `BenchmarkCompilerStartup` n=500 | 35ms / 572,089 allocs |

The last two rows are the milestone working: startup cost is now flat against
repository size. A consequence the planner must not miss — **corpus size no
longer moves the measured compile time**, so a larger corpus buys evidential
weight for the claim, not a harder gate, and its only real cost is generation
time.

</domain>

<decisions>
## Implementation Decisions

### Where the budget is asserted

- **D-01:** **The wall-clock budget gets its own non-race CI step.** The
  existing `Run coverage` step in `.github/workflows/go.yml` runs
  `go test -race -coverprofile=... ./...`, and the race detector costs roughly
  8x: a 400-proto corpus measured 34ms plain and 268ms under `-race`. A
  200ms wall-clock assertion cannot live inside that job and mean anything.
  A second step runs the budget test without `-race` and without coverage, so
  the asserted number is the one an operator would actually see.

  Rejected: asserting a race-scaled budget in the existing job. It keeps one
  job but the asserted number stops meaning 200ms, and `.planning/codebase/TESTING.md`
  already warns that a threshold which flakes gets bumped until it is worthless.

- **D-02:** **The budget test times the in-process operation, not the CLI
  binary.** It measures `NewCompiler` + `CompileFile` — the same operation
  `compileCorpus` (`compiler/lib/startup_bench_test.go:59`) already measures —
  so the figure is directly comparable to BASELINE.md's 6.97s breakdown and to
  the existing scaling test.

  PROJECT.md states the milestone goal as "`protoconf compile` completes in
  under 200ms", which names the CLI and therefore includes Go runtime init,
  flag parsing and config loading that the in-process figure excludes. That gap
  is closed by evidence rather than by a heavier test: **one real
  `protoconf compile` invocation is timed by hand and its number recorded
  alongside the GATE-05 figures** (see D-06). CI stays light; the goal
  statement still gets an honest measurement behind it.

- **D-03:** **The gate threshold is tighter than the stated 200ms, targeting
  roughly 100ms, and the exact number is calibrated from an observed CI runner
  measurement rather than from a developer laptop.** A 200ms gate against a
  35ms measurement has ~5.7x headroom and will not fail until something
  regresses roughly fivefold — it certifies the milestone but is nearly
  useless as a regression detector.

  Calibration is a required step, not an optional refinement: land the test
  logging its measurement unconditionally, read the real runner number from an
  actual CI run inside this phase, then set the threshold at roughly 2x that
  observed value. GitHub shared runners commonly land 2-3x slower than the
  laptop these numbers came from, which would put an 800-proto compile at
  70-105ms — sitting directly on a naively-chosen 100ms line. **A threshold
  picked from the numbers in this document, without a CI observation, is a
  planning error.**

  Any threshold at or below 200ms satisfies GATE-02 strictly, so this
  tightening does not conflict with the requirement text.

  — **Reversibility:** reversible — the threshold is one constant in one test;
  loosening it later touches nothing else.

### The broken-proto behavior change

- **D-04:** **Accepted, with buf named as the answer.** Before lazy loading,
  every proto under `src/` was parsed and linked at startup, so a syntax error
  in a proto no config referenced still failed the compile. Under the lazy
  registry it is never parsed, so it is never reported. The decision states
  plainly that whole-tree proto validation is now buf's job, not the
  compiler's, and points operators at buf.

  This is the operator-facing half of Phase 14's D-02, which accepted
  first-request failure and explicitly rejected adding a startup validation
  pass. It is consistent with that decision, not a new one.

  **Honest caveat the write-up must not paper over:** `buf.yaml` configures
  `lint` but `.github/workflows/lint.yml` runs only `buf breaking` — there is
  no `buf lint` step in this repository's CI today. And `buf.yaml` covers
  protoconf's own protos, not a downstream user's config repository. buf is
  therefore something an operator runs against their own tree, not something
  protoconf already runs for them. The write-up should say that rather than
  imply coverage that does not exist.

  — **Reversibility:** reversible as a document. The underlying behavior it
  describes is one-way (Phase 13's D-02 deleted the eager fallback), but this
  phase only writes the behavior down.

- **D-05:** **The write-up lands in two places: the CHANGELOG breaking-changes
  section and a README section.** The `## Unreleased` block in `CHANGELOG.md`
  already carries two `### ⚠ BREAKING CHANGES` entries from the CLI precedence
  work, written in the same operator-facing voice — an upgrader reads that
  section by habit. The README section catches newcomers who never read a
  changelog, by documenting what `protoconf compile` does and does not
  validate as standing behavior.

  Both must stay consistent with each other.

- **D-06 (negative decision):** **No test pins the documented behavior.**
  GATE-03 asks for a written decision, not a test. Phase 11's LAZY tests
  already establish that only reached protos are parsed, so an unreferenced
  broken proto not failing the compile is a consequence of behavior that is
  already under test. Do not add a fixture for it.

### Real-corpus evidence

- **D-07:** **The gate generates its own calibrated corpus. Nothing depends on
  the sibling checkout.** The real 799-proto protoconf-terraform corpus lives
  at `../protoconf-terraform/example/src` — a sibling checkout, present on the
  author's machine, absent on CI and on any other developer's machine, and
  drifting independently. Rather than making verification conditional on a tree
  that may not exist, the gate generates a synthetic corpus at run time using
  `utils/testdata.GenerateCorpus`.

  The corpus is **sized so that it costs what the real 799 protos cost**, not
  merely so it has 800 files. Using the ~3x per-file multiplier stated in
  GATE-05, that is roughly **2400 generated protos**.

  **This supersedes GATE-05's "compiled once at milestone close" wording.** The
  requirement's underlying intent — that the closing claim be as strong as one
  made against a real tree — is preserved by the calibration. The planner
  should treat GATE-05 as satisfied by the calibrated generated corpus and
  should surface the requirement-text mismatch rather than silently reinterpret
  it.

  — **Reversibility:** costly — undoing means reintroducing a dependency on an
  external checkout that CI cannot satisfy, which is the problem this avoids.

- **D-08:** **The ~3x multiplier is taken at face value, deliberately, despite
  being unsourced.** Recorded as an accepted risk rather than discovered later:

  GATE-05's claim that "the synthetic corpus is ~3x cheaper per file" appears
  nowhere in `.planning/research/compiler-performance/`. The only 3.3x figure
  in the milestone is at `REQUIREMENTS.md:30` and measures something unrelated
  — unlinked parse against linked parse. Meanwhile `utils/testdata/corpus.go`
  documents its own tuning history in the opposite direction, noting earlier
  cuts were 47x then 21x cheaper and that the current message count and field
  options "close the rest", implying the generator is now much closer to parity
  than 3x.

  The alternatives — sizing the corpus by matching BASELINE.md's measured
  1,286ms unlinked-parse cost over the real tree, or measuring the ratio and
  correcting the figure in REQUIREMENTS.md — were both offered and declined.

  **Exposure is low and that is why this is acceptable:** because startup cost
  is now flat against corpus size (39ms at n=50, 35ms at n=500), an inaccurate
  multiplier changes the file count but barely changes the measured time or the
  gate's outcome. The cost of being wrong is a slightly weaker evidential claim,
  not a wrong gate.

- **D-09:** **The closing numbers are appended to
  `.planning/research/compiler-performance/BASELINE.md` as an after-section.**
  BASELINE.md already holds the 6.97s before-numbers with a per-stage
  breakdown, so an after-section there makes the comparison immediate and keeps
  one document as the milestone's measurement record. It becomes a living
  document rather than a frozen snapshot; that is accepted.

  The section records the calibrated-corpus figure and the single real
  `protoconf compile` CLI timing from D-02.

### Test-suite evidence

- **D-10:** **Both suites, CI authoritative.** The pinned command
  `go test -race -timeout 300s -skip 'Test_cliCommand_Run' ./...` (already set
  as `workflow.test_command` in `.planning/config.json`) is the runnable verify
  a reviewer can execute locally. The green Go workflow run on the phase's own
  commit is what GATE-04 is closed against, because CI runs the full `./...`
  under `-race` with no skips and therefore covers the four
  `Test_cliCommand_Run` cases the local command omits.

  The gap between the two is stated rather than left implicit: `Test_cliCommand_Run`
  exists in `inserter/`, `server/`, `agent/` and `compiler/` and is skipped
  locally because it stalls on Consul.

- **D-11:** **A green run is sufficient proof of unchanged outcome for the
  validator cases.** The expected outcomes are already encoded as assertions in
  `compiler/lib/compiler_test.go` — four cases expecting `ErrInvalidConfig`
  (`with_config_rollout_validator.pconf:41`, `validator_test.pconf:56`,
  `validator_repeated_test.pconf:57`, `validator_map_test.pconf:58`) and
  `validator_passing_test.pconf:59` expecting `nil`. A passing run is by
  definition an unchanged outcome. No before/after checkout comparison, no
  new assertions.

  **Finding recorded for completeness, deliberately not required in the
  evidence:** one pre-existing compiler test is skipped rather than green —
  `load_remote_with_load_local.pconf` (`compiler/lib/compiler_test.go:47-51`),
  skipped in commit `7fdffc4` on 2026-09-01 for a stale `vizceral_repo` module
  pin. It is 244 commits back, predates this milestone's phases, and is
  unrelated to lazy loading. The option to name it as an explicit exception in
  the GATE-04 evidence was offered and declined; a flat "every pre-existing
  test stays green" claim therefore overclaims by exactly this one test. The
  planner should not add it to the evidence, but should not be surprised by it
  either.

### Claude's Discretion

The user explicitly deferred this. Decide it on evidence from the code and the
CI job layout, not by asking again.

- **Where `TestCompilerStartupScaling` lives once the non-race step exists.**
  Either it stays in the main `-race` job — its allocation ratio measured 0.91x
  plain and 1.02x under `-race`, so the race detector does not threaten it, and
  it keeps its coverage contribution — or it moves to the new non-race step so
  both performance gates sit together and it stops paying the race detector's
  8x slowdown (0.98s to 3.3s locally). Both are defensible; pick one and say why.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Milestone measurement record
- `.planning/research/compiler-performance/BASELINE.md` — the 6.97s
  before-numbers with per-stage breakdown, the 1,286ms unlinked-parse figure
  over the real 799-proto tree, and the corpus path. D-09 appends the
  after-section here.
- `.planning/research/compiler-performance/OPTIONS.md` — measured alternatives;
  source of the scan-cost figures Phase 13 built on.
- `.planning/REQUIREMENTS.md` — GATE-01 through GATE-05 verbatim, lines 76-81.
  Note line 30's 3.3x figure is unrelated to GATE-05's ~3x claim (D-08).

### The gates themselves
- `compiler/lib/startup_bench_test.go` — `TestCompilerStartupScaling` (the
  `t.Skipf` at line 163 that GATE-01 deletes), `compileCorpus`, and
  `BenchmarkCompilerStartup`.
- `compiler/lib/compiler_test.go` — the validator cases at lines 41 and 56-59,
  and the pre-existing skip at lines 47-51.
- `utils/testdata/corpus.go` — `GenerateCorpus`, its shape constants, and its
  tuning history documenting the 47x-then-21x per-file cost gap.

### CI wiring
- `.github/workflows/go.yml` — the single `Run coverage` step running
  `go test -race -coverprofile=... ./...`; D-01 adds a step here.
- `.github/workflows/lint.yml` — runs `buf breaking` only, no `buf lint`
  (the caveat in D-04).
- `buf.yaml` — lint is configured but unrun in CI; excludes `utils/testdata`.
- `.planning/config.json` — `workflow.test_command`, the pinned local suite.

### Operator-facing documents
- `CHANGELOG.md` — the `## Unreleased` / `### ⚠ BREAKING CHANGES` block where
  D-05's entry lands, beside the existing CLI precedence entries.
- `README.md` — where D-05's standing-behavior section lands.

### Prior decisions this phase inherits
- `.planning/phases/13-exact-symbol-index-shared-type-url-resolution/13-CONTEXT.md`
  — D-02 (eager fallback deleted, accepted risk stated), D-04 (counters are
  test-only; a timing assertion folded into the scaling test was rejected as
  the flakiest gate shape).
- `.planning/phases/14-non-compiler-consumer-correctness/14-CONTEXT.md` — D-02
  (failure surfaces at first request, loudly; no startup validation pass).
  GATE-03 is that decision's operator-facing half.
- `.planning/codebase/TESTING.md` — the convention that a threshold which
  flakes gets bumped until it is worthless.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `compileCorpus` (`compiler/lib/startup_bench_test.go:59`) — already the
  shared measured operation for the benchmark and the scaling test. The budget
  test in D-02 should measure this same function, not a new one.
- `testdata.GenerateCorpus(dir, n)` (`utils/testdata/corpus.go:26`) —
  deterministic, caller-owns-dir, needs no lock file, git repo or
  `CONFIGSPACE` marker. Two calls with the same n produce byte-identical
  output. This is what D-07's ~2400-proto corpus comes from.
- `runtime.MemStats` before/after pattern in `TestCompilerStartupScaling`'s
  `measure` closure — the established shape for a machine-independent
  measurement in this file.

### Established Patterns
- Gate on allocations, report wall clock. The scaling test logs both
  unconditionally and asserts only on the allocation ratio, with the reasoning
  written inline: allocation tracks file count almost exactly and is
  deterministic, while wall clock on a shared runner is not. D-03's calibration
  requirement exists precisely because the budget test breaks this pattern by
  gating on wall clock.
- Measure-then-log even when not gating. The scaling test's `t.Logf` runs on
  every invocation regardless of outcome. D-03 depends on this: the budget
  test must log its measurement before a threshold is chosen.
- Phase 14 established that a test cited as a control must be demonstrated
  capable of failing before its guard lands. GATE-01's flip from `t.Skipf` to
  `require.LessOrEqual` should show the assertion can fail, not just that it
  passes at 0.91x.

### Integration Points
- `.github/workflows/go.yml` gains one step. It is the only workflow that runs
  Go tests; `lint.yml` runs golangci-lint, buf breaking and actionlint, and
  `release.yml` runs goreleaser.
- `BASELINE.md` gains an after-section; nothing reads it programmatically.
- `CHANGELOG.md` and `README.md` gain prose sections; no tooling parses either.

</code_context>

<specifics>
## Specific Ideas

- The corpus target is "costs what the real 799 protos cost", not "has 800
  files". Roughly 2400 generated protos at the ~3x multiplier. The phrasing
  matters for how the evidence reads.
- The threshold number in D-03 is deliberately left unfilled. It comes from a
  CI observation taken during this phase, not from this document.
- The `protoconf compile` CLI timing in D-02 is one number from one real
  invocation, recorded as evidence. It is not a gate and needs no automation.

</specifics>

<deferred>
## Deferred Ideas

- **Add a `buf lint` step to `.github/workflows/lint.yml`.** `buf.yaml`
  configures lint but CI runs only `buf breaking`. Deliberately out of scope
  here: it would lint protoconf's own protos, not an operator's config
  repository, which is the case GATE-03 is actually about. Adding it would not
  strengthen D-04's claim.
- **Repin `vizceral_repo` in the small testdata lock file** to un-skip
  `load_remote_with_load_local.pconf` (`compiler/lib/compiler_test.go:47-51`).
  A stale upstream module pin from 2026-09-01, unrelated to this milestone.
- **Amend GATE-05's text in REQUIREMENTS.md** to match D-07, and correct or
  source the ~3x multiplier per D-08. Offered during discussion and declined
  for this phase; the mismatch is recorded in D-07 and D-08 instead.

</deferred>

---

*Phase: 15-verification-decision-gate-flip*
*Context gathered: 2026-09-09*
