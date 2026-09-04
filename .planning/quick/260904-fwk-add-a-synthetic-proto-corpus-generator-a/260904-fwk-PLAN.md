---
phase: quick-260904-fwk
plan: 01
type: execute
wave: 1
depends_on: []
autonomous: true
requirements: [QUICK-260904-fwk]

files_modified:
  - utils/testdata/corpus.go
  - compiler/lib/startup_bench_test.go

estimate:
  tokens: 34000
  raw_tokens: 22000
  tasks: 2
  confidence: low

must_haves:
  truths:
    - "A generated corpus of N protos compiles end-to-end through `lib.NewCompiler(root,false)` + `c.CompileFile(\"main.mpconf\")` with no lock file, no git repo, no CONFIGSPACE marker, and no `ms.Init`/`ms.Sync` — and writes a real `.materialized_JSON` under `<root>/materialized_config/main/`."
    - "The generated `main.mpconf` names exactly 5 `.proto` files in its `load()` statements at every N — N=50 and N=5000 differ only in how many protos are PRESENT, never in how many are DEMANDED."
    - "Generated protos import and structurally reference earlier generated protos, so `linker.Link`/`resolveReferences` has real cross-file symbol work to do — a corpus of zero-import protos would understate the 26% linking share in BASELINE.md."
    - "The generator is deterministic: two calls with the same N produce byte-identical files."
    - "`BenchmarkCompilerStartup` calls `b.ReportAllocs()` and excludes corpus generation from the timed region."
    - "`TestCompilerStartupScaling` measures the scaling ratio, LOGS the measured number on every run, and today ends in `t.Skipf` (not a pass and not a failure) because the ratio is out of bounds by construction. Deleting that skip branch is the lazy-loading milestone's definition of done."
    - "`go test ./compiler/lib/ -count=1` and `go test ./utils/... -count=1` stay green, and `go vet ./...` is clean."
  artifacts:
    - utils/testdata/corpus.go
    - compiler/lib/startup_bench_test.go
  key_links:
    - "`ms.GetProtoRegistry()` (compiler/lib/module_service.go:366) ends with `registry.Import(registry.Parse, nil-excludes, filepath.Join(root, consts.SrcPath))`, and `Import` calls `find(path, \".proto\")` (utils/utils.go:110-131). That single line is the whole eager walk: dropping N `.proto` files under `<root>/src/` is sufficient and necessary to reproduce the cost. Nothing else in the corpus affects it."
    - "`load(\"//pkg0/msg0.proto\", \"Msg0\")` resolves as follows: `loadMatcherRegex` (module_service.go:469) yields Repo==\"\" for a `//`-prefixed path, so `Load` falls through to `toCanonicalPath` -> `pkg0/msg0.proto` -> `loadInner`, which dispatches on the `.proto` suffix (starlark_loader.go:154) to `loadProto` -> `l.parser.ParseFilesX`. The path handed to the parser is src-relative with forward slashes. A relative `load(\"pkg0/msg0.proto\", ...)` also works but resolves against the CALLER's directory; use the `//` form so the mpconf's own location cannot change the answer."
    - "`NewModuleService(root)` needs only an absolute path (module_service.go:43-64) — it does not read the filesystem. `LoadFromLockFile` logs via `slog.Error` and returns rather than failing when `protoconf.lock` is absent (compiler.go:60-63), so the corpus needs no lock file. Expect one `error loading from lock file` line per NewCompiler in test output; that is not a failure."
    - "`writeConfig` does its own `mkdirAll(filepath.Dir(filename))` (compiler.go:301), so the corpus must NOT pre-create `materialized_config/`."
    - "Alloc-bytes are the scaling gate, wall-clock is the reported number. Per BASELINE.md ~40% of profile time is GC and allocation tracks file count almost exactly, so the alloc ratio is deterministic and machine-independent while a wall-clock ratio on a shared CI runner is not. This is the same posture the project took on codecov thresholds (quick 260902-cov)."
---

<objective>
Build the measurement apparatus for the compiler-startup milestone: a deterministic synthetic proto corpus generator parameterised on N, plus a benchmark and a scaling test that consume it.

Purpose: every number in BASELINE.md came from `~/go/src/github.com/protoconf/protoconf-terraform/example` — a live working tree outside this repo, free to drift. A benchmark against an unpinned corpus measures the corpus. The bug is "cost scales with repository size, not config size" (799 protos linked, 7 needed, 4,639ms vs 2.7ms), and *scaling* is the only assertion worth making about it. Only a generator parameterised on N lets you assert scaling. Nothing in the milestone is measurable until this exists.

Output: `utils/testdata/corpus.go` (generator) and `compiler/lib/startup_bench_test.go` (smoke test, benchmark, scaling test). No production code is touched.
</objective>

<execution_context>
@/Users/smintz/go/src/github.com/protoconf/protoconf/.claude/gsd-core/workflows/execute-plan.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.claude/gsd-core/templates/summary.md
</execution_context>

<context>
@.planning/research/compiler-performance/BASELINE.md
@.planning/research/compiler-performance/TESTING.md
@utils/testdata/embed.go
@compiler/lib/compiler.go
</context>

<scope_boundary>
Harness only. Do NOT touch `GetProtoRegistry`, `utils.DescriptorRegistry`, `compiler/lib/parser`, the starlark loader, or any of the five registry consumers. Do NOT implement or prototype lazy loading. Do NOT modify `utils/testdata/embed.go` or any file under `utils/testdata/small/`.
</scope_boundary>

<honesty_note>
What each check actually proves, stated up front so no one later mistakes one for another:

- **`TestGeneratedCorpusCompiles`** proves the generator emits a corpus the real compiler accepts. It is a fixture test. It passes before and after the lazy milestone and is NOT a performance regression guard.
- **`BenchmarkCompilerStartup`** proves nothing on its own — benchmarks do not fail. It is a reported number, and the thing that makes the milestone's improvement legible.
- **`TestCompilerStartupScaling`** is the only check with teeth, and it has none *yet*: today it measures, logs, and skips, because the ratio is ~N/50 by construction. It becomes a real regression guard the moment the skip branch is deleted, which is the milestone's DoD.

Nothing in this plan would fail if lazy loading were reverted. That is expected and is the reason the skip is written to auto-green.
</honesty_note>

<tasks>

<task type="tracer">
  <name>Task 1: Corpus generator wired end-to-end through the real compiler</name>
  <files>utils/testdata/corpus.go, compiler/lib/startup_bench_test.go</files>
  <read_first>
    - utils/testdata/embed.go (sibling conventions; note it uses `os.MkdirTemp` + `os.Exit` — do NOT copy the `os.Exit`, CLAUDE.md forbids it in libraries)
    - compiler/lib/compiler.go:52-80 (`NewCompiler`) and :288-312 (`writeConfig`)
    - compiler/lib/module_service.go:348-373 (`GetProtoRegistry`) and :469-497 (`loadMatcherRegex`, `ParseModulePath`)
    - utils/testdata/small/src/multioutputs_test.mpconf (mpconf shape: `load(...)` then `def main()` returning a dict)
  </read_first>
  <action>
Create `utils/testdata/corpus.go`, package `testdata`, exporting one function:

`func GenerateCorpus(dir string, n int) error`

`dir` is a caller-owned directory (tests pass `t.TempDir()` / `b.TempDir()`, which gives free cleanup — do not call `os.MkdirTemp` inside, and do not `git.PlainInit`; `NewCompiler` needs neither). Return errors wrapped with `fmt.Errorf("...: %w", err)` per CLAUDE.md; never `os.Exit`, never `panic`.

It writes, under `dir`:

1. `src/pkg{i}/msg{i}.proto` for i in [0,n), each proto3, `package corpus.pkg{i};`, one message `Msg{i}` with a `string name = 1;` field.

2. Import fan-out: each proto imports up to 3 EARLIER protos (index j < i) and, for each import, declares a field of that imported message type (`corpus.pkg{j}.Msg{j} dep_{j} = <tag>;`). Declaring the field is the load-bearing part — an unreferenced import gives the linker nothing to resolve, and per BASELINE.md linking is 26% of the profile. Pick the j values with `math/rand.New(rand.NewSource(1))` seeded once per call so output is deterministic across runs; de-duplicate the picks so a proto never imports the same file twice (a duplicate import is a parse error) and never self-imports.

3. `src/main.mpconf` that loads EXACTLY 5 protos at every N, using the fixed lowest indices so the transitive closure stays bounded and independent of N (BASELINE's real config demanded 5 -> 7 transitively; pkg0..pkg4 gives a closure of the same order):

   `load("//pkg0/msg0.proto", "Msg0")` ... through pkg4, then a `def main():` returning a dict of a handful of those messages keyed by output name (e.g. `{"out": Msg0(name="x"), ...}`). Use the `//`-rooted form — see the key_link on load resolution. Requires n >= 5; return an error if n < 5.

Do NOT create `materialized_config/`, `protoconf.lock`, `CONFIGSPACE`, or a git repo.

Then create `compiler/lib/startup_bench_test.go`, `package lib`, with `TestGeneratedCorpusCompiles`: generate N=50 into `t.TempDir()`, `require.NoError` on `GenerateCorpus`, build with `lib.NewCompiler(dir,false)` (in-package, so just `NewCompiler`), `require.NoError(t, c.CompileFile("main.mpconf"))`, and assert a materialized output file exists under `<dir>/materialized_config/main/`. Then sanity-check the generator against BASELINE: `t.Logf` the elapsed `NewCompiler` time and the registry file count (`c.ModuleService.GetProtoRegistry().FileRegistry` length) so a wildly unrepresentative corpus is visible immediately rather than after the milestone plans against it.
  </action>
  <verify>
    <automated>cd /Users/smintz/go/src/github.com/protoconf/protoconf && go test ./compiler/lib/ -run TestGeneratedCorpusCompiles -count=1 -v 2>&1 | tail -20</automated>
  </verify>
  <done>`TestGeneratedCorpusCompiles` passes; its log line shows a non-zero NewCompiler duration and a FileRegistry count of at least 50. `go build ./...` and `go vet ./...` clean.</done>
</task>

<task type="auto">
  <name>Task 2: Scaling test and allocation-reporting benchmark</name>
  <files>compiler/lib/startup_bench_test.go</files>
  <action>
Extend `compiler/lib/startup_bench_test.go` (no new files).

Factor a small unexported helper that both new functions share — generate a corpus of size n into a caller-supplied dir, then `NewCompiler` + `CompileFile("main.mpconf")`, returning an error. Generation stays OUTSIDE the measured/timed region in both callers.

**`BenchmarkCompilerStartup(b *testing.B)`** — sub-benchmarks over n in {50, 500, 5000}, named `protos=%d`. Generate the corpus once per sub-benchmark into `b.TempDir()`, then `b.ReportAllocs()` and `b.ResetTimer()` before the `b.N` loop; `b.Fatal` on error. `allocs/op` is not optional here: BASELINE.md attributes ~40% of profile time to GC, so allocations move before `ns/op` does and are far less noisy. Skip the n=5000 case when `testing.Short()` — at today's eager cost one iteration there is minutes, and note in a comment that `-benchtime=1x` is the appropriate way to run this.

**`TestCompilerStartupScaling(t *testing.T)`** — the assertion that matters. Skip immediately under `testing.Short()`. Measure the helper at n=50 and n=400 (NOT 5000: at ~4.6s per construction for 799 protos, larger N makes a plain `go test ./...` intolerable, and the ratio is already unambiguous at 400).

For each N capture BOTH:
  - wall-clock elapsed, and
  - bytes allocated, via a `runtime.ReadMemStats` `TotalAlloc` delta bracketing the call with a `runtime.GC()` before the first read.

Gate on the **allocation ratio**, report the wall-clock. Rationale to put in a comment: allocation tracks file count almost exactly and is deterministic, so the alloc ratio is machine-independent; a wall-clock ratio on a shared CI runner is not, and per TESTING.md a threshold that flakes gets bumped until it is worthless.

`t.Logf` both ratios unconditionally, so the number is visible on every run. Then:

    if allocRatio > maxRatio { t.Skipf(...) }

with `maxRatio = 2.0` and a message carrying the measured ratio, the target, and the pointer `.planning/research/compiler-performance/BASELINE.md`. Add a comment stating that this skip branch is the lazy-loading milestone's definition of done and must be DELETED (turning it into a `require.LessOrEqual`) when the ratio comes into bounds.

Measure-then-skip is deliberately chosen over an unconditional `t.Skip` at the top of the function: the body still executes, so it cannot silently rot into non-compiling or wrong code; it prints a live number today that doubles as the generator's sanity check; and it auto-greens the instant the fix lands. It is still not a pass — a skipped test is reported as skipped. It is not red either, so CI stays green.
  </action>
  <verify>
    <automated>cd /Users/smintz/go/src/github.com/protoconf/protoconf && go test ./compiler/lib/ -run TestCompilerStartupScaling -count=1 -v 2>&1 | tail -15 && go test ./compiler/lib/ -count=1 && go test ./utils/... -count=1 && go vet ./... && go test ./compiler/lib/ -run XXX -bench BenchmarkCompilerStartup/protos=50 -benchtime=1x -benchmem 2>&1 | tail -8</automated>
  </verify>
  <done>`TestCompilerStartupScaling` reports SKIP with a logged alloc ratio well above 2.0 and a message naming BASELINE.md; the full `go test ./compiler/lib/ -count=1` suite is still green; the benchmark at protos=50 emits an `allocs/op` column.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| (none) | Test-only code. No new network, filesystem-outside-tempdir, or untrusted input paths. Generated protos are produced from an in-process `fmt.Sprintf` template with integer parameters only. |

## STRIDE Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation Plan |
|-----------|----------|-----------|----------|-------------|-----------------|
| T-fwk-01 | Denial of Service | `GenerateCorpus` writing N files | low | mitigate | Callers pass `t.TempDir()`/`b.TempDir()`, so the runtime bounds cleanup; N=5000 is gated behind `!testing.Short()` and only reachable under `-bench`. |
| T-fwk-02 | Tampering | new dependencies | low | accept | No new modules added — `math/rand`, `runtime`, `os`, `fmt`, `path/filepath` are stdlib, `testify` is already a direct dependency. No package-manager install task in this plan. |
</threat_model>

<verification>
- `go test ./compiler/lib/ -count=1` green (baseline at HEAD 78706d3 was green; keep it so).
- `go test ./utils/... -count=1` green.
- `go build ./...` and `go vet ./...` clean.
- `TestCompilerStartupScaling` reports SKIP, not PASS and not FAIL.
- Two consecutive `GenerateCorpus(dir, 50)` runs into different dirs produce identical trees (spot-check with `diff -r`).
</verification>

<success_criteria>
The milestone can now state a target and measure against it without leaving the repository: `go test ./compiler/lib/ -bench BenchmarkCompilerStartup -benchtime=1x -benchmem` produces a scaling table, and `TestCompilerStartupScaling` prints today's out-of-bounds ratio while leaving CI green. The lazy-loading milestone's DoD is reduced to a concrete, reviewable diff: delete the `t.Skipf` branch.
</success_criteria>

<output>
Create `.planning/quick/260904-fwk-add-a-synthetic-proto-corpus-generator-a/260904-fwk-SUMMARY.md` when done.
</output>
