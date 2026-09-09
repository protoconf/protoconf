# Baseline: Compiler Startup Performance

**Measured:** 2026-09-04
**Corpus:** `~/go/src/github.com/protoconf/protoconf-terraform/example` — 799 `.proto` under `src/`, one `example.mpconf`
**Machine:** darwin/arm64, warm page cache, `CGO_ENABLED` default
**Binary:** `go build ./cmd/protoconf` at `823f546`

## Headline

`protoconf compile .` takes **6.97s**. The target is 200ms.

Essentially all of it is startup. The actual compile is already inside budget.

## Stage breakdown

Measured by instrumenting the `NewCompiler` path directly (not `go test`, to avoid
harness noise). Each stage timed in isolation on the corpus above:

| Stage | Time | Share |
|-------|------|-------|
| `NewModuleService` + `LoadFromLockFile` | 0.03ms | ~0% |
| `utils.NewDescriptorRegistry` (`desc.WrapFiles` over global registry) | 6.2ms | 0.1% |
| **`ms.GetProtoRegistry()` — parse + link all 799 protos** | **4,639ms** | **94%** |
| **`parser.NewParserWithDescriptorRegistry` — build resolvers over 864 files** | **260ms** | **5%** |
| `protovalidate.New(legacy.WithLegacySupport(...))` | 2.7ms | 0.1% |
| `c.CompileFile("example.mpconf")` — Starlark eval, validate, write | 197ms | — |

`NewCompiler` = **5.06s**. `CompileFile` = **197ms**.

**The compile step is already at the 200ms target.** The entire problem is that
`NewCompiler` eagerly materializes the whole proto tree before any config is read.

## What the config actually needs

`src/example.mpconf` opens with six `load()` statements. Five name `.proto` files:

```
//terraform/random/provider/v3/random.proto
//terraform/random/resources/v3/pet.proto
//terraform/null/provider/v3/null.proto
//terraform/null/datasources/v3/data.proto
//protoconf_terraform/config/v1/config.proto
```

Parsing and linking exactly those five, plus their transitive imports:

```
parsed+linked 5 requested -> 7 total files in 2.686ms
```

**7 files needed. 864 loaded. 2.7ms vs 4,639ms — a ~1,700x gap.**

This is the whole finding. The compiler's cost is not proportional to the config
being compiled; it is proportional to the size of the repository the config
happens to live in.

## CPU profile

`go tool pprof` over `GetProtoRegistry()` (5.15s wall, 7.57s samples across ~1.5 cores):

| Node | cum | Notes |
|------|-----|-------|
| `runtime.gcDrain` | 2.80s (37%) | GC pressure |
| `runtime.scanobject` | 2.59s (34%) | — |
| `linker.Link` | 1.94s (26%) | of which `resolveReferences` 1.67s |
| `linker.Files.FindFileByPath` | 1.18s (16%) | reached via `resolveInFile` -> `FindImportByPath` |
| `runtime.madvise` | 1.10s (15%) | heap growth / return |
| `parser.protoParserImpl.Parse` | 0.50s (7%) | — |

Roughly: **~40% garbage collection, ~26% linking, ~7% parsing**, remainder
allocation and I/O.

The GC share is the tell. Materializing 864 fully-linked descriptor trees
allocates enormously, and most of those objects are never read. Cutting the file
count attacks the linking cost and the GC cost together — they are the same
problem measured two ways.

## Negative result: the linker lookup is not a fixable upstream bug

`linker.Files.FindFileByPath` is a linear scan over a slice, and it shows up at
16% of profile time. That looks like an O(n²) defect worth reporting upstream or
patching around. It is not:

- `(*result).FindImportByPath` calls `r.deps.FindFileByPath` — `r.deps` is **that
  file's direct dependencies only**, typically a handful, not all 864.
- The cost is therefore *call volume* (one lookup per symbol reference resolved
  across 799 files), not scan length.
- `protocompile` v0.14.1 has the identical implementation, byte for byte. A
  dependency bump changes nothing here.

Conclusion: there is no lookup to optimize. The only lever on linking cost is
**linking fewer files**. Do not spend time on this path again.

## Related measurements

Numbers that constrain the design space (see OPTIONS.md):

| Operation over all 799 protos | Time |
|-------------------------------|------|
| `ParseFilesButDoNotLink` (parse only, no linking) | 1,286ms |
| Naive single-threaded lexical scan (read + 2 regexes/file) | 623ms |
| Full parse + link (current behaviour) | 4,639ms |

Both eager alternatives blow the 200ms budget on their own. **Any design that
touches all 799 files on the hot path fails, regardless of how cheaply it
touches them.**

## Milestone close (2026-09-09)

**Measured:** 2026-09-09
**Commit:** `cbfe79c66e1d8adbe0042ff41bd1e9b90e4f1d24`
**Machines:** darwin/arm64 (in-process figure, CLI figure) and `ubuntu-latest` via
GitHub Actions (CI-observed figure, both under GATE-04's `-race ./...` run and the
dedicated non-race budget step)

This section is the milestone's "after" for the before-numbers above. It is
appended, not merged into them — this file is a living document per D-09, and the
2026-09-04 measurements above stay exactly as they were measured.

### After-numbers

All four figures below measure the same operation (`NewCompiler` + `CompileFile`,
the same call path `compileCorpus` in `startup_bench_test.go` exercises) over the
same 2400-proto calibrated corpus, unless noted otherwise.

| Figure | Value | Produced by |
|---|---|---|
| In-process `compileCorpus`, local | 70.929333ms | `go test -run '^TestCompilerStartupBudget$' -count=1 -v ./compiler/lib/...` on darwin/arm64, non-race |
| In-process `compileCorpus`, CI | 93.393125ms | GitHub Actions run [34313915733](https://github.com/protoconf/protoconf/actions/runs/34313915733), step "Run startup budget gate", on `ubuntu-latest` |
| Calibrated threshold, asserted in CI | 160ms | `TestCompilerStartupBudget`'s `budget` constant (`compiler/lib/startup_bench_test.go`), calibrated per D-03 from an earlier CI observation (run `34311638861`, `77.581263ms`) |
| Real `protoconf compile` CLI invocation | 0.08s (80ms) | `/usr/bin/time -p <binary> compile <dir> main.mpconf`, best of three runs (1.38s, 0.08s, 0.08s — the first run pays first-run page-cache cost) on darwin/arm64, binary built via `go build -o <scratch>/protoconf ./cmd/protoconf` |

The CLI figure exists because PROJECT.md's goal names the binary — "`protoconf
compile` completes in under 200ms" — which includes Go runtime init, flag parsing
and config loading that the in-process figure excludes (D-02). All three
in-process/CI/CLI runs confirmed from their own `compile finished` log line that
they took the lazy path: `protoFilesLoaded=5` over the 2400-proto tree.

### What the corpus is

The figures above were measured over a corpus generated by
`utils/testdata.GenerateCorpus`, **not** over the real 799-proto
protoconf-terraform tree at `../protoconf-terraform/example/src`. It is sized at
2400 protos to *cost what those 799 real protos cost*, using the ~3x per-file
multiplier GATE-05 states (799 × 3 ≈ 2400, D-07).

That multiplier is unsourced: it appears nowhere in
`.planning/research/compiler-performance/`, and `utils/testdata/corpus.go`'s own
tuning history (earlier cuts 47x then 21x cheaper per file, closed by the current
message count and field options) suggests the generator is closer to parity than
3x (D-08).

This is acceptable rather than merely admitted, because startup cost is now flat
against corpus size — 39ms at n=50, 35ms at n=500 (`BenchmarkCompilerStartup`) —
so a wrong multiplier changes the file count without materially changing the
measured time or the gate's outcome. The cost of being wrong here is a slightly
weaker evidential claim, not a wrong gate.

### GATE-05's requirement-text mismatch

GATE-05, as written in `REQUIREMENTS.md`, asks for: "The real protoconf-terraform
corpus (799 protos, 6.97s at baseline) is measured once at milestone close and the
number recorded as evidence — not a CI gate, since that repo is not checked in and
drifts independently."

D-07 supersedes this wording: the real protoconf-terraform corpus is a sibling
checkout, present on one author's machine, absent on CI and on every other
developer's machine, and drifting independently — a milestone-closing gate cannot
depend on it. The calibrated generated corpus above preserves GATE-05's underlying
intent (a closing claim as strong as one made against a real tree) without that
dependency. Amending GATE-05's text in `REQUIREMENTS.md` to match was offered
during phase discussion and explicitly deferred out of this phase.

### GATE-04 evidence

**Run:** [34313915733](https://github.com/protoconf/protoconf/actions/runs/34313915733), commit `cbfe79c66e1d8adbe0042ff41bd1e9b90e4f1d24`, conclusion: **success**.

The five named cases, all from `compiler/lib/compiler_test.go`'s `TestCompiler_CompileFile` table, observed in that run:

| Fixture | Expected | Observed |
|---|---|---|
| `with_config_rollout_validator.pconf` | `ErrInvalidConfig` | PASS |
| `validator_test.pconf` | `ErrInvalidConfig` | PASS |
| `validator_repeated_test.pconf` | `ErrInvalidConfig` | PASS |
| `validator_map_test.pconf` | `ErrInvalidConfig` | PASS |
| `field_type_any_test.pconf` | `nil` | PASS |

The same run's `TestCompilerStartupScaling` passed, logging `alloc ratio=1.02x
(39298920 -> 39937944 bytes)` — the CI-side confirmation of GATE-01's allocation
gate under `-race`.

**What the two commands cover (D-10).** The pinned local command,
`go test -race -timeout 300s -skip 'Test_cliCommand_Run' ./...`, is the
reviewer-runnable check — it passed with zero failures across all packages. CI is
authoritative for GATE-04 because it runs the full `./...` under `-race` with no
skips, and therefore additionally covers the `Test_cliCommand_Run` cases in
`inserter/`, `server/`, `agent/` and `compiler/` that the local command omits
because they stall on Consul.

`.planning/` edits committed after the cited run — including this section — do
not change test outcomes, so run `34313915733` remains the evidence for the
phase's code state.

### The gates' resting state

- **GATE-01** — asserted by `TestCompilerStartupScaling` (`require.LessOrEqual` on the allocation ratio).
- **GATE-02** — asserted by `TestCompilerStartupBudget` in the "Run startup budget gate" CI step.
- **GATE-03** — documented in `CHANGELOG.md` and `README.md`.
- **GATE-04** — evidenced above: run `34313915733`, conclusion success, all five named cases unchanged in outcome.
- **GATE-05** — evidenced above: the calibrated-corpus figures, with the corpus and the requirement-text mismatch both stated for what they are.

## Reproduction

Harness scripts are inlined in TESTING.md. The corpus is a working tree, not a
fixture — see TESTING.md for how to pin one.
