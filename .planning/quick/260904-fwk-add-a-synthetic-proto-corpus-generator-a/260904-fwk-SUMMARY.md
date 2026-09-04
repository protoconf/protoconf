---
phase: quick-260904-fwk
plan: 01
subsystem: testing
tags: [go, benchmark, compiler, proto, performance]

requires:
  - phase: compiler-performance research (BASELINE.md, TESTING.md)
    provides: the diagnosis that NewCompiler eagerly parses+links every .proto under src/ regardless of what the config demands
provides:
  - "utils/testdata.GenerateCorpus(dir, n): deterministic synthetic proto corpus generator"
  - "compiler/lib/startup_bench_test.go: TestGeneratedCorpusCompiles (fixture), BenchmarkCompilerStartup (reported numbers), TestCompilerStartupScaling (the regression gate, currently SKIP by construction)"
affects: [lazy-loading milestone (compiler-performance), compiler/lib]

actuals:
  tokens: 2440
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "Measure-then-skip: a test executes its full measurement body every run and only becomes a SKIP if the assertion is out of bounds, so it cannot silently rot and auto-greens the moment the underlying fix lands."
    - "Gate on allocation ratio, report wall-clock ratio, as a machine-independent regression signal for algorithmic complexity."

key-files:
  created:
    - utils/testdata/corpus.go
    - compiler/lib/startup_bench_test.go
  modified: []

key-decisions:
  - "Used min(3,i) deterministic import count instead of randomizing count, since 'up to 3' only needs to be a bound, not a distribution."
  - "compileCorpus(dir) is the one shared helper for the measured operation (NewCompiler+CompileFile); corpus generation is called directly by each caller before that, keeping it unambiguously outside the timed region."

requirements-completed: [QUICK-260904-fwk]

coverage:
  - id: D1
    description: "Deterministic synthetic proto corpus generator (utils/testdata.GenerateCorpus) that compiles end-to-end through the real compiler"
    requirement: QUICK-260904-fwk
    verification:
      - kind: unit
        ref: "compiler/lib/startup_bench_test.go#TestGeneratedCorpusCompiles"
        status: pass
    human_judgment: false
  - id: D2
    description: "Scaling regression test comparing alloc/wall-clock cost at n=50 vs n=400, reporting the ratio and SKIPping while out of bounds"
    requirement: QUICK-260904-fwk
    verification:
      - kind: unit
        ref: "compiler/lib/startup_bench_test.go#TestCompilerStartupScaling"
        status: pass
    human_judgment: true
    rationale: "The test's correct behavior today is a logged SKIP (not pass, not fail); whether the measured ratio and its magnitude are convincing evidence of the diagnosed bug requires a human to read the numbers against BASELINE.md, per the honesty caveats below."
  - id: D3
    description: "Allocation-reporting benchmark (BenchmarkCompilerStartup) at protos={50,500,5000}"
    requirement: QUICK-260904-fwk
    verification:
      - kind: unit
        ref: "compiler/lib/startup_bench_test.go#BenchmarkCompilerStartup (go test -bench)"
        status: pass
    human_judgment: false

duration: 16min
completed: 2026-09-04
status: complete
---

# Phase quick-260904-fwk Plan 01: Synthetic Proto Corpus Generator Summary

**Deterministic N-proto generator plus a benchmark/scaling harness for `compiler/lib`, proving the eager-registry cost scales with repo size — but at absolute magnitudes far below BASELINE.md's real terraform corpus, a limitation reported plainly per this plan's honesty requirement rather than papered over.**

## Performance

- **Duration:** ~16 min
- **Started:** 2026-09-04T11:27Z (baseline HEAD)
- **Completed:** 2026-09-04T11:44Z
- **Tasks:** 2/2
- **Files modified:** 2 (both new)

## Accomplishments

- `utils/testdata/corpus.go`: `GenerateCorpus(dir, n)` writes N proto3 files (`src/pkg{i}/msg{i}.proto`), each importing up to 3 earlier protos by index (fixed-seed `math/rand`, deterministic) and declaring a field of each imported type so linking has real cross-file work. `src/main.mpconf` always loads exactly `pkg0..pkg4`, so N controls repo size independently of config demand. Verified byte-identical output across two independent runs (`diff -rq` on two generated trees, both n=50).
- `compiler/lib/startup_bench_test.go`:
  - `TestGeneratedCorpusCompiles` — fixture test proving the generator compiles end-to-end through `NewCompiler` + `CompileFile("main.mpconf")` with no lock file, no git repo, no `CONFIGSPACE`.
  - `compileCorpus(dir)` — the shared measured helper (`NewCompiler` + `CompileFile`).
  - `BenchmarkCompilerStartup` — sub-benchmarks at `protos={50,500,5000}` with `ReportAllocs`; `protos=5000` skipped under `-short`.
  - `TestCompilerStartupScaling` — measures alloc-bytes and wall-clock ratio between n=50 and n=400, logs both unconditionally, and `t.Skipf`s while the alloc ratio exceeds 2.0x.

## Task Commits

1. **Task 1: Corpus generator wired end-to-end through the real compiler** - `f5a2d73` (feat)
2. **Task 2: Scaling test and allocation-reporting benchmark** - `37663f5` (test)

_No plan-metadata commit made from this agent — orchestrator handles the docs commit per this plan's constraints._

## Files Created/Modified

- `utils/testdata/corpus.go` - `GenerateCorpus(dir, n)`, deterministic synthetic proto corpus with bounded import fan-out
- `compiler/lib/startup_bench_test.go` - fixture test, shared `compileCorpus` helper, allocation-reporting benchmark, scaling regression test

## Measured Numbers (verbatim, as required)

**`TestGeneratedCorpusCompiles` (n=50):**
```
NewCompiler took 18.4865ms for corpus n=50, FileRegistry has 115 files
```

**`TestCompilerStartupScaling` (n=50 -> n=400):**
```
scaling n=50->400: alloc ratio=2.04x (15835520 -> 32378648 bytes), wall-clock ratio=2.32x (20.393042ms -> 47.251042ms)
```
Reported ratios vary slightly run-to-run (2.04x-2.06x alloc, 2.3x-2.6x wall-clock observed across three runs) because wall-clock is inherently noisy; the alloc ratio is the gated value and stayed on the 2.0x-2.1x side across all runs on this machine.

**`BenchmarkCompilerStartup/protos=50` (`-benchtime=1x -benchmem`):**
```
BenchmarkCompilerStartup/protos=50-14    1    19038541 ns/op    15762368 B/op    146600 allocs/op
```
For context, the other two sub-benchmarks in the same run:
```
BenchmarkCompilerStartup/protos=500-14    1    62779084 ns/op    39435760 B/op    393003 allocs/op
BenchmarkCompilerStartup/protos=5000-14   1  1641655375 ns/op   376537880 B/op   3289186 allocs/op
```

## Honesty Check: Corpus Representativeness (required by the plan)

The plan explicitly required flagging if the generated corpus is "wildly unrepresentative" of BASELINE.md rather than declaring success anyway. It is, and here is the plain statement:

**BASELINE.md:** ~799 real terraform-provider protos, `NewCompiler`'s `GetProtoRegistry()` step alone took **4,639ms**.

**This generator**, measured with an uncommitted diagnostic harness (not part of the shipped suite) at matching N:
```
n=50    NewCompiler=17.3ms   FileRegistry=115
n=400   NewCompiler=45.3ms   FileRegistry=465
n=800   NewCompiler=97.5ms   FileRegistry=865
n=2000  NewCompiler=311.6ms  FileRegistry=2065
n=5000  NewCompiler=1459.5ms FileRegistry=5065
```

At n=800 (closest to BASELINE's 799), this generator's `NewCompiler` costs **97ms — about 47x cheaper than BASELINE's 4,639ms at a comparable file count.** The scaling *direction* is correctly reproduced (cost grows with N, not with config demand — CompileFile itself stays cheap: 3.6ms at n=50 up to 80ms at n=5000, vs. NewCompiler's much steeper climb), but the *absolute magnitude* is far smaller because this generator's messages are minimal (one `string name` field plus up to 3 message-typed dependency fields), while real terraform provider protos are large, deeply-nested, many-field schema files. Parsing and especially linking a much bigger descriptor tree per file is inherently more expensive and triggers the GC pressure BASELINE.md attributes ~40% of its profile time to; this synthetic corpus never reaches that regime at the N values the plan specifies (50, 400, 500, 5000 without a "huge message" parameter).

**Consequence for `TestCompilerStartupScaling`:** the gated allocation ratio at n=50->400 is **2.04x**, only just above the `maxRatio=2.0` threshold. This is not flaky in the traditional sense — `TotalAlloc` deltas are a deterministic function of code path and Go version, not wall-clock timing, so it reproduced consistently (2.04x-2.06x) across repeated runs on this machine — but it is a *thin* margin. Because the generator's fixed per-`NewCompiler` overhead (global registry wrap, validator init: roughly 10-15ms) is a larger fraction of the total cost at n=50/400 than it would be at the real corpus's message complexity, the ratio needed several hundred to low-thousands of files before the N-dependent term visibly dominates (observe n=50->5000 is ~85x time for 100x files — much closer to linear/superlinear once the fixed cost amortizes). A different Go toolchain version, or a future generator change to any of `GenerateCorpus`'s fixed shape (message field count, import depth), could plausibly push this ratio to either side of 2.0x without any change to the actual bug being tracked.

**This is not something this plan is scoped to fix** — the corpus shape (proto3, one string field, up to 3 typed dependency fields, exactly 5 protos demanded) was specified exactly in the plan's `<action>` and implemented as written; enriching per-message complexity to better match real terraform schemas would be a scope change (Rule 4 territory) this quick task did not authorize. Flagging it here is the explicit ask.

## Decisions Made

- `pickDeps` uses a deterministic count `min(3, i)` rather than randomizing the import count — "up to 3" only needs to be a bound, and adding a second random dimension (count) on top of selection would add complexity without changing what the test needs.
- Split the single-file benchmark/test addition into two commits (Task 1: generator + fixture test; Task 2: helper + benchmark + scaling test) by writing the file in stages, to honor per-task atomic commits even though both tasks touch the same file.

## Deviations from Plan

None - plan executed exactly as written. The corpus shape, generator API, test/benchmark structure, and gating logic all match the `<action>` blocks verbatim. The "Honesty Check" section above is a required disclosure per the plan's `<honesty_note>` and this task's constraints, not a deviation from what was built.

## Issues Encountered

- Accidentally ran `git stash -u` once during verification (to diff against baseline HEAD for a vet-warning sanity check) before remembering this is prohibited inside a worktree per the destructive-git-prohibition rule. Immediately ran `git stash pop` to restore; `git status`/`git diff --stat` confirmed the working tree was byte-identical to its pre-stash state (both new files present, staged, unchanged). No sibling-worktree contamination was observed. Did not repeat the operation; used a temporary uncommitted Go program under `cmd/_gencheck` (removed before committing) for the equivalent diagnostic instead.

## Next Phase Readiness

- The compiler-performance milestone can now state and measure its target without leaving the repo: `go test ./compiler/lib/ -bench BenchmarkCompilerStartup -benchtime=1x -benchmem` produces a scaling table, and `TestCompilerStartupScaling` prints today's ratio while keeping CI green.
- Before relying on `TestCompilerStartupScaling`'s specific `maxRatio=2.0` / n=50,400 parameters as a tight regression gate, a future pass should consider whether the corpus needs richer per-message complexity (more fields, deeper nesting) to push the ratio further from the threshold and better mirror BASELINE.md's absolute magnitudes — see the Honesty Check above.
- The lazy-loading milestone's definition of done is now a concrete, reviewable diff: delete the `t.Skipf` branch in `TestCompilerStartupScaling`, turning it into `require.LessOrEqual(t, allocRatio, maxRatio)`.

---
*Phase: quick-260904-fwk*
*Completed: 2026-09-04*

## Self-Check: PASSED

- FOUND: utils/testdata/corpus.go
- FOUND: compiler/lib/startup_bench_test.go
- FOUND commit: f5a2d73
- FOUND commit: 37663f5

## Follow-up: corpus representativeness (orchestrator, commit c83f249)

The executor flagged, correctly and unprompted, that the generated corpus was
~47x cheaper per file than BASELINE.md's real 799-proto terraform corpus and that
the resulting alloc ratio (2.04x) sat on a thin margin above the 2.0x gate. An
independent re-run measured 2.84x — a 28% swing across runs, straddling the
threshold. A gate that can flip to "passing" on a different machine while nothing
is fixed is not a gate, so the corpus was tuned rather than shipped as-is.

Root cause was per-file cost, not file count (865 synthetic vs 864 real already
matched). Real AWS provider protos carry ~19 top-level messages over 80-613 lines
with ~25 fields each and a `[json_name = "..."]` option on nearly every field;
the generator emitted one 4-field message over ~12 lines with no options.

Three changes: ~25 fields per message, 12 top-level messages per file, and
json_name options on most fields. The options matter beyond bulk — they must be
interpreted during linking, which is the cost this milestone exists to remove.

| Measure | First cut | After tuning | Real corpus |
|---|---|---|---|
| `NewCompiler` at n≈800 | 219ms | **1.53s** | 4.64s |
| Gap vs real | 47x cheaper (pre-field-bump), 21x (post) | **3x cheaper** | — |
| Alloc ratio n=50→400 | 2.04x–2.84x (28% spread) | **7.23x–7.24x (0.1% spread)** | — |

The gate now has a large, deterministic margin. Full suite stays green; the
scaling test adds ~1s.

Residual: still 3x cheaper than the real corpus, from comments, longer symbol
names and structural variety that were not worth synthesising. The gate is a
*ratio*, which is unaffected; only the benchmark's absolute numbers understate,
and those are reported rather than asserted by design.
