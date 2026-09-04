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

## Reproduction

Harness scripts are inlined in TESTING.md. The corpus is a working tree, not a
fixture — see TESTING.md for how to pin one.
