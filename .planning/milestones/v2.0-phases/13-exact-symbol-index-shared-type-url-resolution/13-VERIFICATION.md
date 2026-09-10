---
phase: 13-exact-symbol-index-shared-type-url-resolution
verified: 2026-09-08T14:20:00Z
status: passed
score: 4/4 roadmap success criteria verified, 10/10 requirement IDs satisfied
covered_files:
  - .planning/phases/13-exact-symbol-index-shared-type-url-resolution/13-01-PLAN.md
  - .planning/phases/13-exact-symbol-index-shared-type-url-resolution/13-01-SUMMARY.md
  - .planning/phases/13-exact-symbol-index-shared-type-url-resolution/13-02-PLAN.md
  - .planning/phases/13-exact-symbol-index-shared-type-url-resolution/13-02-SUMMARY.md
  - .planning/phases/13-exact-symbol-index-shared-type-url-resolution/13-03-PLAN.md
  - .planning/phases/13-exact-symbol-index-shared-type-url-resolution/13-03-SUMMARY.md
  - .planning/phases/13-exact-symbol-index-shared-type-url-resolution/13-04-PLAN.md
  - .planning/phases/13-exact-symbol-index-shared-type-url-resolution/13-04-SUMMARY.md
  - .planning/phases/13-exact-symbol-index-shared-type-url-resolution/13-REVIEW.md
  - compiler/lib/compiler.go
  - compiler/lib/index_not_built_test.go
  - compiler/lib/lazy_load_count_test.go
  - compiler/lib/load_mutable_nested_any_test.go
  - compiler/lib/module_service.go
  - compiler/lib/parser/hard_error_test.go
  - compiler/lib/parser/nested_any_test.go
  - compiler/lib/parser/parser.go
  - compiler/lib/starlark_loader.go
  - compiler/lib/tier_observability_test.go
  - server/server.go
  - utils/growable_resolver_test.go
  - utils/index_build_deadlock_test.go
  - utils/lazy_parse_canonical_test.go
  - utils/symbol_index.go
  - utils/symbol_index_cache_test.go
  - utils/symbol_index_test.go
  - utils/symbol_scan.go
  - utils/symbol_scan_test.go
  - utils/testdata/small/mutable_config/nested_any_mutation.materialized_JSON
  - utils/testdata/small/src/load_mutable_nested_any_test.pconf
  - utils/utils.go
covered_digest: "v1:sha256:077d05ab9a0370428eb87499827621484c8587d3e0de033a38c9b4335242aaab"
behavior_unverified: 0
overrides_applied: 0
---

# Phase 13: Exact Symbol Index & Shared Type-URL Resolution Verification Report

**Phase Goal:** Any message symbol declared anywhere under `src/`, including
nested types, resolves to its declaring file without linking the whole
repository, and every nested `google.protobuf.Any` in a config — including
one reached only through a mutable config load — resolves through one
shared code path backed by that index, not a per-consumer implementation.

**Verified:** 2026-09-08T14:20:00Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (ROADMAP success criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | A materialized config carrying a nested `Any` at any depth — including one reached only through a mutable config load — round-trips correctly through one shared resolution path; grepping finds that single implementation for the compiler | ✓ VERIFIED | `resolveTiers` in `compiler/lib/parser/parser.go` is the single tier chain both `FindMessageByURL`/`FindMessageByName` delegate to. `TestReadConfigNestedAny`, `TestResolveTiersAgreeAcrossEntryPoints`, `TestBothEntryPointsShareTheTierChain` all PASS (re-run live). `loadMutable` (CONS-05) now resolves exclusively through `l.parser.TypeResolver.FindMessageByURL` — confirmed by `grep -n FindMessageTypeByUrl compiler/lib/starlark_loader.go` returning no output, and `TestLoadMutableResolvesNestedAny`/`TestLoadMutableEmptyValueFailsLoudly` PASS (re-run live). **Scoped caveat, explicit in D-03 and confirmed unchanged:** `inserter/inserter.go:369`, `mutate/mutate.go:76`, and three remaining `server/server.go` call sites (`:436`, `:443`, `:466`) still call `LocalResolver` directly and are out of this phase's scope by design — the repo-wide grep-clean state completes in Phase 14. This is a scoped, documented deferral, not a silent gap. |
| 2 | A repeat compile against an unchanged `src/` tree reuses the persisted, content-keyed index under `.protoconf_cache`; any `.proto` edit invalidates it; a stale index is never served | ✓ VERIFIED | `TestIndexCacheReusedOnUnchangedTree` and `TestIndexCacheInvalidatedOnProtoEdit` PASS (re-run live). Cache validated via `dirhash.HashDir` content-key gate, refused-and-rebuilt on key/header/count mismatch (`TestIndexCacheRefusedOnKeyMismatch/OnTruncation/OnBadHeader`, all pass per 13-02-SUMMARY and code inspection of `loadSymbolIndexCache`). **Flagged, not blocking:** code review WR-01 (`utils/symbol_index.go:207-215`) — the entry-count check compares raw *line* count to the header's declared count, not the number of *distinct keys* actually installed into the map (`index[parts[0]] = parts[1]` has no duplicate-key detection). A cache file corrupted with a duplicate-key line pair would pass validation while under-populating the index. This is a narrower defect than criterion 2's stated claim (edit invalidation, which is the tested and proven mechanism); it applies only to a specific corruption shape of an already-untrusted file. Bounded blast radius per reviewer: a wrong/missing path from an under-populated index still fails safely through `ParseOne`'s `filepath.IsLocal` gate, so this degrades to a slow re-resolution, not silent data corruption reaching a caller. Unfixed at verification time — recommend tracking as a follow-up (see Anti-Patterns / Code Review section). |
| 3 | Compiling a config whose only mutable value is a `google.protobuf.Value` resolves via the existing unconditional global-registry seed and never triggers index construction | ✓ VERIFIED | `TestValueOnlyMutableNeverBuildsIndex` PASS (re-run live): `symbolIndexBuilds=0 symbolIndexCacheHits=0 scanResolutions=0` in the compile-finished log line. |
| 4 | An unknown, custom-type symbol resolves via the index to the file that declares it, including nested types; only files actually referenced by resolved symbols get linked; building the index parses every file under `src/` but links none | ✓ VERIFIED | `TestSymbolIndexNestedType` PASS (re-run live) — `nested.v1.Outer.Middle.Inner` maps correctly, `nested.v1.Inner`/`nested.v1.Middle` absent (Pitfall-3 regression). `TestIndexBuildLinksZeroFiles` and `TestResolveEscalatesToIndex` PASS — `LoadedFileCount()` stays 0 across a build, and `LoadSymbolByIndex` only links the one `ParseOne`-confirmed path after a hit (per 13-02-SUMMARY, structurally: `buildSymbolIndex` uses `ParseFilesButDoNotLink` only). |

**Score:** 4/4 roadmap success criteria verified; 0 present-but-behavior-unverified.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|---|---|---|---|---|
| TYPE-01 | 13-02 | Symbol index maps every message symbol under `src/`, nested included, to its declaring file | ✓ SATISFIED | `TestSymbolIndexNestedType`, `TestSymbolIndexIncludesEnums` pass |
| TYPE-02 | 13-02 | Index built by parsing without linking | ✓ SATISFIED | `buildSymbolIndex` uses `ParseFilesButDoNotLink`; `TestIndexBuildLinksZeroFiles` pass (cost-proportionality half is a documented assumption, not a timing-asserted claim — consistent with D-04's rejection of timing gates) |
| TYPE-03 | 13-01 | Nested `Any` at any depth resolves | ✓ SATISFIED | `TestReadConfigNestedAny` (depth-2) pass |
| TYPE-04 | 13-02 | Only referenced files link; index build links nothing | ✓ SATISFIED | `TestIndexBuildLinksZeroFiles` pass |
| TYPE-05 | 13-02 | Index persisted under `.protoconf_cache`, content-keyed | ✓ SATISFIED | `TestIndexCacheReusedOnUnchangedTree` pass |
| TYPE-06 | 13-02 | `.proto` edit invalidates cache; stale never served | ✓ SATISFIED (see criterion 2's WR-01 caveat above) | `TestIndexCacheInvalidatedOnProtoEdit` pass |
| TYPE-07 | 13-02 | Index built on first need only | ✓ SATISFIED | `TestValueOnlyMutableNeverBuildsIndex` pass |
| TYPE-08 | 13-01 + 13-03 | Single shared code path per consumer | ✓ SATISFIED for the compiler (in-scope consumer, D-03) | `resolveTiers` is structurally the one chain; `TestBothEntryPointsShareTheTierChain` pass. Repo-wide grep-clean (all 4 consumers) is explicitly deferred to Phase 14 by D-03 — a scoped, documented boundary, not a shortfall against this phase's own scope. |
| TYPE-09 | 13-01 | `parser.ReadConfig` resolves nested `@type` at any depth | ✓ SATISFIED | `TestReadConfigNestedAny`, `TestReadConfigTwoSiblingAnysNamesTheFailingOne` pass; zero changes needed to `ReadConfig` (verified via `protojson`'s own recursive resolver consultation) |
| CONS-05 | 13-04 | Mutable config load resolves value type + nested `Any` correctly | ✓ SATISFIED | `TestLoadMutableResolvesNestedAny`, `TestLoadMutableEmptyValueFailsLoudly` pass; `grep FindMessageTypeByUrl compiler/lib/starlark_loader.go` empty |

No orphaned requirements: `REQUIREMENTS.md`'s Phase 13 traceability table lists exactly TYPE-01..09 and CONS-05, and every one of those IDs appears in a plan's frontmatter `requirements` field (13-01: TYPE-03/08/09; 13-02: TYPE-01/02/04/05/06/07; 13-03: TYPE-08; 13-04: CONS-05).

### Cross-Phase Regression (orchestrator note 1) — Confirmed Coherent

13-04's fixture (`nested_any_mutation.materialized_JSON`, outer type
`test.v1.TestMessage`) lives in the shared `utils/testdata/small/mutable_config/`
tree that `server.GenReflectionUI` walks. `standalone.ExampleRequest.MarshalJSON`
(vendored `grpcui`) calls a resolver-less `protojson.Marshal` for any
`proto.Message` value, which cannot resolve a locally-parsed nested `Any`.

Verified end-to-end at HEAD:
- `server/server.go:601-640` (`GenReflectionUI`) now resolves via
  `s.parser.TypeResolver.FindMessageByURL` and pre-marshals each example with
  `protojson.MarshalOptions{Resolver: s.parser.TypeResolver}.Marshal(dynamic)`,
  handing `grpcui` a `json.RawMessage` — confirmed by reading the vendored
  `marshalData`'s `default: return json.Marshal(data)` branch, which passes a
  `json.RawMessage` through verbatim without needing a resolver (commit `3b4db71`).
- `698956a` (13-04's own workaround, retargeting the fixture at a non-RPC-input
  type) was reverted by `2e722fb`, with the reasoning recorded in `10f7dcb`:
  keeping the retarget would have shipped `3b4db71` with **zero** test coverage,
  since the retargeted type has no `exampleMaker` entry and the marshal path is
  never reached.
- Confirmed live: `nested_any_mutation.materialized_JSON`'s outer type is
  `test.v1.TestMessage` again (an RPC input for `TestService.PutTestMessage`,
  so `exampleMaker` registers it and `GenReflectionUI` does exercise the
  nested-Any marshal path). `go test -race ./compiler/... ./utils/... ./server/...`
  is green (re-run live, all `ok`), including
  `TestProtoconfMutationServer_GenReflectionUI`.

**Conclusion: the end state is coherent, and the coverage genuinely exists** —
the regression is fixed at its root cause, and the fixture that would have
caught a recurrence is restored to a shape that actually exercises it.

### Prohibition Assessment (orchestrator note 2) — No Real Coverage Lost

13-03's plan prohibits deleting a test merely because its symbol is gone,
requiring re-pointing at the replacement mechanism instead. Two tests in
`utils/growable_resolver_test.go` (`TestParseAllRegistersIntoFilesResolver`,
`TestFilesResolverRegistrationErrorsStayZero`) were deleted rather than
re-pointed.

Independently verified the executor's and code reviewer's reasoning:
- `grep -rn '\.Import(\|\.Parse(' --include='*.go' .` (excluding tests) shows
  exactly two production call sites of `registry.Import`:
  `server/server.go:356` (a fresh, throwaway `discoveryRegistry` with
  `ImportPaths` never set — eager) and
  `compiler/lib/module_service.go:361` (`GenFileDescriptorSet`, module-fetch
  registries) and `:460`, which is explicitly inside the `else` (non-lazy)
  branch of `GetProtoRegistry` — `registry.ImportPaths` is set only in the
  `if m.lazyRegistry` branch, which calls neither `Import` nor `Parse`.
- Confirmed `GetFilesResolver` (`utils/utils.go:203-226`): for
  `len(d.ImportPaths) == 0` (every caller of `Import`/`Parse` above), it
  rebuilds a fresh `*protoregistry.Files` from `FileRegistry` on every call
  rather than growing incrementally — so the "bulk path registers into the
  growable resolver via a before/after diff" behavior the deleted tests
  pinned genuinely has no production code path to exercise: the combination
  those tests required (`ImportPaths` set AND `Import`/`Parse` called
  directly) only ever existed inside `ParseAll`'s own now-deleted loop.
- `TestRegistrationCountIsIncremental` (retained, re-run live: PASS) already
  fully covers the registration-diff invariant via `ParseOne`, the sole
  remaining writer on the lazy/growable path.

**Conclusion: the prohibition was not meaningfully violated.** The deleted
tests' scenario had no production analog; deleting them (with reasoning
recorded in a code comment and the SUMMARY) rather than re-pointing at
synthetic-only glue was the correct call, independently confirmed rather
than taken on faith from the SUMMARY/REVIEW narrative.

### Behavioral Spot-Checks (live re-run, not taken from SUMMARY claims)

| Behavior | Command | Result | Status |
|---|---|---|---|
| Nested Any (depth 2) resolves via scan tier through `ReadConfig` | `go test -race ./compiler/lib/parser/... -run TestReadConfigNestedAny -v` | `--- PASS` | ✓ PASS |
| Both entry points share one tier chain | `go test -race ./compiler/lib/parser/... -run TestBothEntryPointsShareTheTierChain -v` | `--- PASS` | ✓ PASS |
| Hard error is diagnostic, wraps `protoregistry.NotFound` | `go test -race ./compiler/lib/parser/... -run TestUnresolvableTypeURLErrorIsDiagnostic -v` | `--- PASS` | ✓ PASS |
| Tier 0 hit consults nothing else | `go test -race ./compiler/lib/parser/... -run TestTier0HitConsultsNothingElse -v` | `--- PASS` | ✓ PASS |
| Escalation to index tier when scan can't narrow | `go test -race ./compiler/lib/parser/... -run TestResolveEscalatesToIndex -v` | `--- PASS` | ✓ PASS |
| Cache reused on unchanged tree / invalidated on edit | `go test -race ./utils/... -run 'TestIndexCacheReusedOnUnchangedTree|TestIndexCacheInvalidatedOnProtoEdit'` | `--- PASS` x2 | ✓ PASS |
| Nested symbol maps correctly, Pitfall-3 regression guarded | `go test -race ./utils/... -run TestSymbolIndexNestedType -v` | `--- PASS` | ✓ PASS |
| `google.protobuf.Value`-only compile never builds/reads index | `go test -race ./compiler/... -run TestValueOnlyMutableNeverBuildsIndex -v` | `--- PASS`, log line shows all-zero tier counters | ✓ PASS |
| Mutable config with nested Any resolves through shared chain only | `go test -race ./compiler/lib/ -run 'TestLoadMutableResolvesNestedAny|TestLoadMutableEmptyValueFailsLoudly' -v` | `--- PASS` x2 | ✓ PASS |
| Compile-finished log reports all three tier counters | `go test -race ./compiler/lib/... -run TestCompileFinishedReportsResolutionTiers -v` | `--- PASS` x2 subtests | ✓ PASS |
| Whole-tree fallback fully removed | `grep -rn --include='*.go' 'func (d \*DescriptorRegistry) ParseAll(\|func (d \*DescriptorRegistry) FellBackToEager('` | no output | ✓ PASS |
| Full regression suite (phase-touched packages + server) | `go test -race ./compiler/... ./utils/... ./server/...` | all `ok` | ✓ PASS |
| `go build ./...` | | clean, exit 0 | ✓ PASS |

### Known Pre-Existing Failure (orchestrator note 3) — Not a Regression

`go test -race ./...` hangs in `github.com/protoconf/protoconf/agent`
(`Test_cliCommand_Run/run_consul_server`, `orchestra.Conductor.playWithLogger`).
Logged in `deferred-items.md` for 13-01/02/03/04 with per-plan confirmation
that no file in the hanging package is touched by this phase. Not re-litigated
here; accepted per orchestrator's independent reproduction at `ad6ceee`
(pre-phase-13) in a clean worktree. `go vet ./...`'s 3 pre-existing findings
(`test/e2e_test.go`, `agent/agent_test.go`, `agent/legacy.go`) are likewise
outside every plan's file list and predate this phase.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|---|---|---|---|---|
| `utils/symbol_index.go` | 207-215 | Cache entry-count check validates raw line count, not distinct map keys installed (WR-01, from `13-REVIEW.md`) | Warning | A cache file with a duplicate-key corruption passes validation while under-populating the index. Bounded blast radius (fails safe via `ParseOne`'s path guard). Unfixed at verification time. |
| `utils/symbol_index.go` | 41-64, 271-274 | `buildSymbolIndex` always returns nil error; `ensureSymbolIndex`'s `buildErr != nil` branch is dead code, so a fully-unreadable root reports `IndexState()=="rebuilt"` indistinguishably from "tree legitimately has no matching symbol" (WR-02, from `13-REVIEW.md`) | Warning | Diagnostic-quality gap, not a correctness gap — the compile still fails loudly (hard `NotFound` error), just with a less actionable `IndexState()` string. Unfixed at verification time. |
| — | — | No `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` markers found in any phase-touched file | Info | Clean |

Both warnings are pre-existing, already-documented findings from `13-REVIEW.md`
(0 Critical, 2 Warning, 1 Info), independently re-confirmed by direct code
reading during this verification. Neither breaks a roadmap success criterion
as literally stated (see per-criterion evidence above); both are real,
unfixed quality gaps worth a deliberate human decision on whether to track as
follow-up work before Phase 14, rather than silently absorbed.

### Data-Flow Trace / Key Link Verification

| Link | Status | Evidence |
|---|---|---|
| `RegistryTypeResolver.resolveTiers` → `LoadSymbolByScan` → `ParseOne` → `MessageRegistry` re-check | ✓ WIRED | Code inspection of `compiler/lib/parser/parser.go:80-95`; `TestReadConfigNestedAny` proves the scan tier (not the fallback) answers, via `ScanResolutionCount()==1` |
| `RegistryTypeResolver.resolveTiers` → `LoadSymbolByIndex` → `ParseOne` → `MessageRegistry` re-check | ✓ WIRED | `TestResolveEscalatesToIndex` proves the index tier answers via `IndexBuildCount()==1, ScanResolutionCount()==0` |
| `ModuleService.GetProtoRegistry` (lazy branch) → `registry.CacheDir` → index persistence | ✓ WIRED | `compiler/lib/module_service.go:456-458`, confirmed by grep and `TestIndexCacheReusedOnUnchangedTree` |
| `starlarkLoader.loadMutable` → `l.parser.TypeResolver.FindMessageByURL` → `desc.WrapMessage` (single resolution) | ✓ WIRED | `compiler/lib/starlark_loader.go:178-195`, confirmed by grep (zero direct `MessageRegistry` calls) and passing regression test |
| `server.GenReflectionUI` → `s.parser.TypeResolver` (concurrent fix, outside phase scope but load-bearing for coverage) | ✓ WIRED | `server/server.go:601-640`, confirmed live |

### Human Verification Required

None. All roadmap success criteria and requirement IDs are backed by passing,
live-re-run automated tests plus direct code inspection; no behavior-dependent
truth was left unexercised.

### Gaps Summary

No blocking gaps. Two pre-existing code-review WARNING findings (WR-01, WR-02
in `13-REVIEW.md`, both in `utils/symbol_index.go`) remain unfixed and are
carried forward here for visibility — neither breaks a roadmap success
criterion as stated, both have bounded/diagnostic-only blast radius, but both
are genuine, currently-untested integrity/observability gaps worth a
deliberate decision (fix now, or explicitly defer to Phase 14 alongside the
already-known CONS-02/03/04 consumer migration) rather than letting them age
silently.

The D-03 scope fence (compiler-only; `inserter/`, `mutate/`, and three
remaining `server.go` call sites still bypass the shared chain) is not a
gap — it is CONTEXT.md's explicitly stated, evidence-based phase boundary,
consistently documented across all four plan SUMMARYs and re-confirmed live
by grep during this verification. ROADMAP success criterion 1's "single
implementation" clause is satisfied for the compiler (the consumer this
phase targets); its repo-wide completion is Phase 14's stated job.

---

_Verified: 2026-09-08T14:20:00Z_
_Verifier: Claude (gsd-verifier)_
