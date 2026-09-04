# Compiler Performance Research — Summary

**Date:** 2026-09-04 · **Corpus:** protoconf-terraform (799 protos) · **Target:** 200ms

## The finding in one line

`protoconf compile` costs 6.97s, of which **5.06s is `NewCompiler` eagerly
parsing and linking all 799 protos** — while the config being compiled loads
exactly 5 of them, 7 with transitive imports.

The actual compile step (`CompileFile`) is already **197ms**. It is at target.
There is nothing to optimise there.

| | Now | Needed | Measured achievable |
|---|---|---|---|
| Registry construction | 4,639ms | — | **2.7ms** (parse+link the 7 real files) |
| Resolver construction | 260ms | — | proportional — collapses with the registry |
| CompileFile | 197ms | ≤200ms | already fine |
| **Total** | **6.97s** | **200ms** | **~210ms** |

The 200ms goal is reachable, and the headroom is comfortable — the fix removes
~4.9s and the remainder is already inside budget.

## Why it is slow

Cost is proportional to **repository size**, not to the config being compiled.
`GetProtoRegistry()` walks `src/`, parses everything, links everything, and caches
it; `ParseFilesX` — the function that services `load("//x/y.proto", ...)` — is then
a pure map lookup that never parses. The work is done up front on the assumption
that all of it will be needed. On this corpus ~99% is discarded.

Profile: ~40% garbage collection (materializing 864 descriptor trees), ~26%
linking, ~7% parsing. The GC share and the linking share are the same problem
counted twice — cutting the file count attacks both.

## Two things that look like leads and are not

Recorded so they are not re-investigated:

- **`linker.Files.FindFileByPath` is 16% of profile time and is a linear scan** —
  but it scans a file's *direct deps*, not all 864. The cost is call volume,
  inherent to linking 799 files. `protocompile` v0.14.1 is byte-identical, so a
  dependency bump changes nothing. There is no lookup to fix; only files not to link.
- **Parallelising the eager parse** — divides 4.6s by core count at best, still
  3-4x over budget, and burns every core on ~99% discarded work.

## Recommendation

**Lazy, load-driven resolution.** Parse a proto the first time it is asked for by
path; the `load()` statements already name exactly what is needed and `protoparse`
resolves transitive imports itself. Measured cost of that path: 2.7ms.

Deferred deliberately: an on-disk descriptor cache (optimises 3ms — not where the
budget is, and the `.fds` machinery already half-exists if warm-start ever
justifies it), and the `jhump/protoreflect` → `dynamicpb` migration (already a
recorded v2 item, and the profile does not implicate the wrappers).

## The real work is correctness, not speed

The speedup is close to free. **Lazy loading is a correctness change disguised as
a performance change**, and every risk it carries is a silent-wrong-answer risk:

1. **`loadValidators` walks the registry, not the filesystem** — with 7 files in
   the registry instead of 864, validators on protos not reachable from `load()`
   are silently skipped. Configs that should fail would compile clean. This is a
   blocker, and the fix (walk the filesystem for `*.proto-validator` instead) is
   *also faster* — the terraform corpus has zero validators and still pays 864
   stat syscalls today. **Do this first, on its own merits.**
2. **`Any` resolution is by type URL, not path** — but the envelope already
   carries the answer. `ProtoconfValue.proto_file` (field 1) is populated on every
   write, and every consumer resolving a type URL is reading a materialized config,
   so the lookup is `type URL -> proto_file -> lazy parse by path`. No symbol index.
   *(Corrected 2026-09-04 — the first pass called this the hard part and proposed
   an index; see PITFALLS.md item 2.)* The real work is a pre-pass in
   `parser.ReadConfig` to read `proto_file` before `protojson` needs the `Any`
   type, plus an explicit, **loud** fallback when `proto_file` is absent or stale —
   a silent one lets a repo regress to 6.9s with the benchmark still green.
3. **The resolvers are eager snapshots** of the registry, taken at construction
   and consulted on the load path. Against a lazy registry that is a stale-cache
   bug — "works the first time, wrong the second".
4. **`mod sync` must stay eager** — it serialises the registry to `.fds` cache
   files, and a lazy registry would write a truncated cache that then loads clean
   and is wrong.

## Testing

No compiler performance coverage exists today. Two gaps:

- **Pin a corpus.** Every number here came from a live working tree. Generate
  synthetic protos parameterised on N instead — the property worth asserting is
  *scaling*, and only a generator lets you assert it.
- **Assert the ratio, not the milliseconds.** "Compiling a 5-proto config costs the
  same at N=50 and N=5000" is machine-independent and fails loudly if an eager walk
  returns. An absolute 200ms gate on a CI runner will flake, get bumped, and become
  worthless. Track `allocs/op` too — with 40% of time in GC it moves first and is
  less noisy than `ns/op`.

Speed tests catch none of the four correctness risks. Each needs its own test that
fails before the fix — the validator one (a `.proto-validator` on a proto no config
loads) is the highest-value test in the set.

## Suggested order

1. Fix `loadValidators` — independently correct, independently faster, removes an
   ordering hazard rather than working around it.
2. Pin a benchmark corpus — nothing else is measurable without it.
3. Make the resolvers lazy views — prerequisite for a lazy registry.
4. Lazy-by-path registry — the actual win (~4.9s).
5. Type-URL resolution — `proto_file`-driven; the `ReadConfig` pre-pass and the absent-`proto_file` fallback.
6. Decide and document the error-surface change (a broken proto no config loads is
   no longer reported at compile time — arguably better, but it is a behaviour
   change that will surface as "CI used to catch this").

## Documents

- `BASELINE.md` — measurements, stage table, profile, reproduction
- `OPTIONS.md` — approaches considered, rejected paths with reasons, blast radius
- `PITFALLS.md` — the four correctness risks in detail, with fixes
- `TESTING.md` — corpus strategy, benchmark shape, correctness guards, CI posture
