---
phase: 13-exact-symbol-index-shared-type-url-resolution
plan: 02
subsystem: compiler
tags: [protobuf, protoreflect, protoparse, symbol-index, dirhash, type-url-resolution]

requires:
  - phase: 13-exact-symbol-index-shared-type-url-resolution
    plan: 01
    provides: "The scoped lexical scan tier (Tier 2) and the shared resolveTiers miss-fallthrough chain this plan extends with Tier 3"
provides:
  - "utils/symbol_index.go: an exact symbol -> declaring-file-path index built by parsing every .proto under src/ without linking (ParseFilesButDoNotLink), covering messages and enums at any nesting depth"
  - "Content-keyed persistence under .protoconf_cache/symbol_index.v1, keyed by dirhash.HashDir over the import roots, validated before trust and refused-and-rebuilt on any header/key/entry-count mismatch"
  - "Tier 3 (LoadSymbolByIndex) wired into RegistryTypeResolver.resolveTiers behind the D-01 scan tier"
  - "Proof that a google.protobuf.Value-only mutable compile never builds or reads either new tier (TYPE-07, ROADMAP criterion 3)"
affects: [13-03-delete-parseall-fallback, 13-04-mutable-config-shared-path]

actuals:
  tokens: 9553
  tasks: 3
  commits: 6
  plan_head_before: a3c8dfb101a8d15ce9a5bcb9c34f57d689949e46

tech-stack:
  added: []
  patterns:
    - "Index build as a candidate-of-last-resort tier: buildSymbolIndex never links (ParseFilesButDoNotLink), never touches FileRegistry/lazyLoaded, and is only ever consulted after the scan tier misses -- LoadSymbolByIndex converts a path hit into a real answer only via ParseOne plus a MessageRegistry re-check, exactly mirroring the scan tier's D-05 non-negotiable"
    - "registry.Load's checksum-gate idiom re-applied to a line-oriented text artifact: a versioned header carrying a content key and entry count, refuse-and-rebuild on any mismatch, atomic temp-file-and-rename writes"

key-files:
  created:
    - utils/symbol_index.go
    - utils/symbol_index_test.go
    - utils/symbol_index_cache_test.go
    - compiler/lib/index_not_built_test.go
  modified:
    - utils/utils.go
    - compiler/lib/module_service.go
    - compiler/lib/parser/parser.go
    - compiler/lib/parser/nested_any_test.go

key-decisions:
  - "Symbol-kind coverage: messages AND enums (CONTEXT.md discretion item 1) -- a no-link parse yields enum descriptors for free and D-02 turns any miss into a hard failure, so breadth is cheap insurance against a one-way decision."
  - "Cache key: dirhash.HashDir(root, \"\", dirhash.Hash1) per import root, joined with the root path (CONTEXT.md discretion item 2) -- measured ~30ms warm on the 799-proto corpus, reusing an already-direct dependency and the module-repo cache-key idiom verbatim."
  - "On-disk format: line-oriented text, symbolIndexHeader + content key + entry count on line 1, sorted <symbol>\\t<path> entries thereafter, atomic temp-file-and-rename write (CONTEXT.md discretion item 4) -- needs no protoc regeneration and the entry count covers the truncation failure mode a proto's length-delimited framing would have covered."
  - "Singleflight: reused d.group with the fixed NUL-prefixed key \"\\x00symbol-index\" (CONTEXT.md discretion item 5) -- no second singleflight.Group, and the NUL prefix cannot collide with a real root-relative path since ParseOne's filepath.IsLocal gate rejects it."
  - "A persistence failure (bad content-key hash, unwritable cache dir) sets IndexState() to \"unavailable: <reason>\" but never fails the build -- an in-memory index is still a correct index, matching the plan's action text and D-02's spirit that a degraded path must never become a hard failure."

patterns-established:
  - "buildSymbolIndex/indexFileDescriptorProto: accumulate the FULL scoped name (not the package) as the recursion prefix through DescriptorProto.GetNestedType(), which is what correctly answers Pitfall 3's exact failure shape -- any future nested-symbol walk in this codebase should recurse the same way."

requirements-completed: [TYPE-01, TYPE-02, TYPE-04, TYPE-05, TYPE-06, TYPE-07]

coverage:
  - id: D1
    description: "The symbol index maps every message and enum symbol under src/, nested types included, to its declaring file's root-relative path, built by an unlinked parse (ParseFilesButDoNotLink) that leaves LoadedFileCount() at 0 and skips broken files without failing the whole build"
    requirement: "TYPE-01"
    verification:
      - kind: unit
        ref: "utils/symbol_index_test.go#TestSymbolIndexNestedType"
        status: pass
      - kind: unit
        ref: "utils/symbol_index_test.go#TestSymbolIndexIncludesEnums"
        status: pass
      - kind: unit
        ref: "utils/symbol_index_test.go#TestIndexBuildSkipsBrokenFile"
        status: pass
      - kind: unit
        ref: "utils/symbol_index_test.go#TestIndexDuplicateSymbolPicksFirstPath"
        status: pass
      - kind: unit
        ref: "utils/symbol_index_test.go#TestIndexEmptyTree"
        status: pass
      - kind: unit
        ref: "utils/symbol_index_test.go#TestIndexLookupIsExactBytes"
        status: pass
    human_judgment: false
  - id: D2
    description: "The index build parses without linking (structurally: ParseFilesButDoNotLink) and links zero files -- LoadedFileCount() and FileRegistry are unaffected by the build, and N concurrent callers pay exactly one build via singleflight"
    requirement: "TYPE-02"
    verification:
      - kind: unit
        ref: "utils/symbol_index_test.go#TestIndexBuildLinksZeroFiles"
        status: pass
      - kind: unit
        ref: "utils/symbol_index_test.go#TestIndexBuildIsSingleflighted"
        status: pass
    human_judgment: true
    rationale: "TYPE-02's cost-proportionality claim ('cost is proportional to parsing alone') is a timing property that D-04 explicitly rejects as a gate shape; the structural half (parses without linking) is fully proven, the cost half is measured (see Accomplishments) but not asserted by a test."
  - id: D3
    description: "The index build never links any file (TYPE-04): a fresh lazy registry's LoadedFileCount() is 0 both before and after ensureSymbolIndex, and FileRegistry's size is unchanged"
    requirement: "TYPE-04"
    verification:
      - kind: unit
        ref: "utils/symbol_index_test.go#TestIndexBuildLinksZeroFiles"
        status: pass
    human_judgment: false
  - id: D4
    description: "The index persists under .protoconf_cache, content-keyed by dirhash.HashDir over the import roots: a second registry over an unchanged tree serves the index from disk (IndexBuildCount 0, IndexCacheHitCount 1, IndexState \"cache hit\") instead of rebuilding, and two builds over the same tree write byte-identical cache files"
    requirement: "TYPE-05"
    verification:
      - kind: unit
        ref: "utils/symbol_index_cache_test.go#TestIndexCacheReusedOnUnchangedTree"
        status: pass
      - kind: unit
        ref: "utils/symbol_index_cache_test.go#TestIndexCacheIsByteIdenticalAcrossBuilds"
        status: pass
      - kind: unit
        ref: "utils/symbol_index_cache_test.go#TestIndexCacheDisabledWithoutCacheDir"
        status: pass
    human_judgment: false
  - id: D5
    description: "Any .proto edit under an import root changes the content key so the next registry rebuilds rather than serving stale data, and a cache file with a mismatched key, a mismatched entry count (truncation), or an unrecognised header is refused and rebuilt rather than served"
    requirement: "TYPE-06"
    verification:
      - kind: unit
        ref: "utils/symbol_index_cache_test.go#TestIndexCacheInvalidatedOnProtoEdit"
        status: pass
      - kind: unit
        ref: "utils/symbol_index_cache_test.go#TestIndexCacheRefusedOnKeyMismatch"
        status: pass
      - kind: unit
        ref: "utils/symbol_index_cache_test.go#TestIndexCacheRefusedOnTruncation"
        status: pass
      - kind: unit
        ref: "utils/symbol_index_cache_test.go#TestIndexCacheRefusedOnBadHeader"
        status: pass
    human_judgment: false
  - id: D6
    description: "The index is built on first escalated need, not eagerly: a compile whose only mutable value is a google.protobuf.Value never builds or reads the index (or the scan tier), since the unconditional globalRegexMatcher Tier-0 seed answers it first; a symbol the scan cannot narrow to a candidate still resolves by escalating to the index tier"
    requirement: "TYPE-07"
    verification:
      - kind: unit
        ref: "compiler/lib/index_not_built_test.go#TestValueOnlyMutableNeverBuildsIndex"
        status: pass
      - kind: unit
        ref: "compiler/lib/parser/nested_any_test.go#TestResolveEscalatesToIndex"
        status: pass
    human_judgment: false
  - id: D7
    description: "Racing the index build against several concurrent ParseOne calls over the same import root is deadlock-free and race-clean, preserving ParseOne's canonical-pointer contract -- the Phase 11 lock-discipline invariant extended to Tier 3"
    verification:
      - kind: unit
        ref: "utils/symbol_index_test.go#TestIndexBuildRacesParseOne"
        status: pass
    human_judgment: false

duration: 55min
completed: 2026-09-08
status: complete
---

# Phase 13 Plan 2: Exact Symbol Index & Tier 3 Wiring Summary

**An exact `symbol -> declaring-file` index, built by an unlinked `ParseFilesButDoNotLink` walk and persisted content-keyed under `.protoconf_cache`, now backs Tier 3 of `resolveTiers` behind 13-01's scan tier — and a `google.protobuf.Value`-only mutable compile provably never builds or reads either new tier.**

## Performance

- **Duration:** ~55 min
- **Started:** 2026-09-08 (approx., continuing directly from 13-01)
- **Completed:** 2026-09-08
- **Tasks:** 3
- **Files modified:** 8 (4 modified, 4 created)

## Accomplishments
- `utils/symbol_index.go`: `buildSymbolIndex` parses every `.proto` under each import root with `protoparse.ParseFilesButDoNotLink` and walks `GetMessageType()`/`GetEnumType()` recursively, accumulating the FULL scoped name (not the package) as the recursion prefix — the exact fix Pitfall 3 exists to force — so `nested.v1.Outer.Middle.Inner` indexes correctly and `nested.v1.Inner` never appears. A single broken file falls back to a per-file loop rather than failing the whole build (verified_facts fact 4). Duplicate symbols keep the lexicographically-smaller path, making the map — and its serialization — deterministic.
- Persistence: `symbolIndexContentKey` hashes each import root with `dirhash.HashDir(root, "", dirhash.Hash1)`; `writeSymbolIndexCache`/`loadSymbolIndexCache` implement `registry.Load`'s checksum-gate idiom for a versioned, line-oriented text artifact under `.protoconf_cache/symbol_index.v1` — refuse-and-rebuild on any header, content-key, or entry-count mismatch, atomic temp-file-and-rename write. `compiler/lib/module_service.go`'s lazy `GetProtoRegistry` branch now sets `registry.CacheDir = m.getCacheDir()`, arming persistence for every real compile.
- Tier 3 (`LoadSymbolByIndex`) is wired into `RegistryTypeResolver.resolveTiers` between the D-01 scan tier and the existing `ParseAll` retry: on an index hit it `ParseOne`s the indexed path and re-checks `MessageRegistry` before answering true — the index never itself returns a `MessageType` or `MessageDescriptor` (RSLV-03).
- Proved TYPE-07/ROADMAP criterion 3 end-to-end: compiling a config whose only mutable value is a `google.protobuf.Value` leaves `IndexBuildCount()`, `IndexCacheHitCount()`, AND `ScanResolutionCount()` all at 0 — the unconditional `globalRegexMatcher` Tier-0 seed answers `google.protobuf.Value` before either new tier is ever reached.
- Proved the escalation path fires correctly: a symbol whose scan candidates exceed `scanCandidateLimit` (32) still resolves, provably through the index tier (`IndexBuildCount() == 1`, `ScanResolutionCount() == 0`).
- **Measured, as required by this plan's `<output>` spec, on the 799-proto/37.8MB corpus at `../protoconf-terraform/example/src`:**
  - `buildSymbolIndex` (unlinked parse of the whole tree): **~1.2s** (1.200650292s measured, 138,555 symbols indexed), consistent with the ~1,286ms `ParseFilesButDoNotLink` figure OPTIONS.md recorded.
  - `symbolIndexContentKey` (`dirhash.HashDir` warm): **~30.2ms**, matching 13-01's `<verified_facts>` fact 5 (28.6–34.3ms across three runs).
  - `loadSymbolIndexCache` warm-path read of the persisted ~138K-entry artifact: **~13.6ms**. This is the previously-unmeasured component 13-01's `scanCandidateLimit` comment flagged: the warm break-even floor of `30/5 = ~6` candidates becomes a real number now that the read cost is known — `(30 + 13.6) / 5 ≈ 8.7` candidates, still comfortably below `scanCandidateLimit = 32`, so the constant needs no revision.
  - **Escalation-rule visibility (CONTEXT.md's explicit requirement):** outside the tests written specifically to force it (`TestResolveEscalatesToIndex`'s 33-file over-limit fixture, and `symbol_index_test.go`/`symbol_index_cache_test.go`'s direct `SymbolFile`/`ensureSymbolIndex` calls), the scan-to-index escalation never fired in this plan's own test run — every other scan-tier-exercising test in the phase (13-01's `TestReadConfigNestedAny`, `TestScanNeverAnswersWithoutParse`, etc.) produces 1-2 candidates after package-directory narrowing, consistent with 13-01's own recorded finding that `scanCandidateLimit = 32` needed no adjustment. This is visible evidence, not a silent outcome: without the two tests built to force it, TYPE-01/02/05/06/07 would be structurally correct but never actually invoked by a real workload in this codebase's current test corpus.

## Task Commits

Each task was committed atomically, following the RED-GREEN cycle for its `tdd="true"` attribute (no task needed a REFACTOR commit — the GREEN implementation needed no follow-up cleanup):

1. **Task 1: Build the index — parse without linking, recurse nested types, link nothing** — RED: `dc0a7eb` (test), GREEN: `3b8a6e9` (feat)
2. **Task 2: Persist the index under .protoconf_cache, content-keyed and validated before trust** — RED: `aa62562` (test), GREEN: `65968f6` (feat)
3. **Task 3: Wire the index in as Tier 3, and prove the Value-only compile never touches it** — RED: `25951b6` (test), GREEN: `807b236` (feat)

**Plan metadata:** commit to follow (docs: complete plan)

## Files Created/Modified
- `utils/symbol_index.go` - `buildSymbolIndex`, `indexFileDescriptorProto`, `joinSymbol`, `parseUnlinked` (Task 1); `symbolIndexContentKey`, `writeSymbolIndexCache`, `loadSymbolIndexCache` (Task 2); `LoadSymbolByIndex` (Task 3); `ensureSymbolIndex`, `SymbolFile`, `IndexBuildCount`, `IndexCacheHitCount`, `IndexState` span all three tasks
- `utils/symbol_index_test.go` - 8 Task 1 behaviors (nested types, enums, zero-link, broken-file resilience, duplicate tie-break, empty tree, exact-byte lookup, singleflighted build) plus Task 3's `TestIndexBuildRacesParseOne`
- `utils/symbol_index_cache_test.go` - 7 Task 2 persistence behaviors (reuse, invalidation, refuse-on-key-mismatch/truncation/bad-header, byte-identical writes, CacheDir-disabled path)
- `compiler/lib/index_not_built_test.go` - `TestValueOnlyMutableNeverBuildsIndex` (TYPE-07/criterion 3)
- `utils/utils.go` - Added `CacheDir`, `symbolIndex`, `indexBuilds`, `indexCacheHits`, `indexState` fields to `DescriptorRegistry`, guarded by the existing `d.mu`
- `compiler/lib/module_service.go` - `GetProtoRegistry`'s lazy branch now sets `registry.CacheDir = m.getCacheDir()`
- `compiler/lib/parser/parser.go` - `resolveTiers` inserts Tier 3 (`LoadSymbolByIndex`) between the scan tier and the `ParseAll` retry
- `compiler/lib/parser/nested_any_test.go` - Added `TestResolveEscalatesToIndex` (Task 3)

## Decisions Made
See `key-decisions` in frontmatter for the five CONTEXT.md discretion items this plan resolved (symbol-kind coverage, cache key, on-disk format, singleflight key, persistence-failure handling). All five follow the plan's action text and CONTEXT.md's evidence-based constraints exactly as written — none required deviating from the plan.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - scope clarification, no code change] `TestResolveEscalatesToIndex` and `TestIndexBuildRacesParseOne` placed in existing files, not new ones**
- **Found during:** Task 3
- **Issue:** The plan's `<files>` list for Task 3 names `compiler/lib/index_not_built_test.go` for the new compiler/lib test, but `TestResolveEscalatesToIndex`'s package is `compiler/lib/parser`, and no `compiler/lib/parser` test file is listed in `<files>`. Similarly, `TestIndexBuildRacesParseOne`'s target file `utils/symbol_index_test.go` already exists from Task 1 (not a new file).
- **Resolution:** Added `TestResolveEscalatesToIndex` to the existing `compiler/lib/parser/nested_any_test.go` (13-01's file for exactly this package, reusing its `newLazyParser` helper) rather than creating a new file, and added `TestIndexBuildRacesParseOne` to the existing `utils/symbol_index_test.go`. Both placements match 13-PATTERNS.md's "extend existing files, no new packages" constraint and RESEARCH.md's "Recommended Project Structure." No production code was affected; this is a test-file-location clarification, not a Rule 1-3 code fix.
- **Files modified:** `compiler/lib/parser/nested_any_test.go`, `utils/symbol_index_test.go`
- **Verification:** Both tests pass; `go vet ./compiler/... ./utils/...` clean.
- **Committed in:** `25951b6` (Task 3 RED commit)

---

**Total deviations:** 1 auto-fixed (1 scope clarification, no production-code impact)
**Impact on plan:** None on correctness or scope. All of this plan's `must_haves.truths`, `must_haves.prohibitions`, and task-level `<acceptance_criteria>` are met exactly as written.

## Issues Encountered
- `go test -race ./...` (the full repository suite) reported `FAIL github.com/protoconf/protoconf/agent 602.107s` — a 9-minute goroutine hang inside `github.com/stephenafamo/orchestra`'s process-orchestration `Conductor.playWithLogger`/`conductPlayer`, unrelated to any file this plan touches (`agent/*.go` is not in this plan's `files` list, and neither `utils/symbol_index.go` nor the `resolveTiers` change is reachable from `agent`'s orchestra-based process-lifecycle tests). Confirmed and logged to `.planning/phases/13-exact-symbol-index-shared-type-url-resolution/deferred-items.md`, matching 13-01's precedent for the impractical-whole-repo-race-run finding. The plan-scoped verification this plan's own tasks actually require — `go test -race ./compiler/... ./utils/...`, run to completion multiple times including two dedicated `-count=2` race-stress passes on the singleflight and lock-discipline tests — is fully green with zero regressions. `go build ./...` and `go vet ./compiler/... ./utils/...` are clean; `go vet ./...` (whole-repo) surfaces the same three pre-existing findings 13-01 already logged (`test/e2e_test.go`, `agent/agent_test.go`, `agent/legacy.go`), none of them in this plan's files either.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All three tiers (MessageRegistry, scan, index) are now wired behind `resolveTiers`, with `ParseAll` still in place as the final retry — 13-03 has a clean, unchanged deletion point exactly as 13-01 predicted.
- The symbol index's persistence (`CacheDir`, content-keying, refuse-and-rebuild) is fully proven independent of any particular consumer, so 13-04's mutable-config shared-path work can rely on it without further plumbing.
- TYPE-01, TYPE-02, TYPE-04, TYPE-05, TYPE-06, TYPE-07 are uniquely owned by this plan (verified via `requirements.ready-ids` against sibling plans' `requirements` frontmatter) and are marked complete in `REQUIREMENTS.md`.
- `.protoconf_cache/symbol_index.v1` is a new on-disk artifact alongside the existing `.fds` caches, already covered by `.gitignore`'s existing `.protoconf_cache` entry — no new ignore rule needed.

## Self-Check: PASSED

- All 8 key files found on disk (`utils/symbol_index.go`, `utils/symbol_index_test.go`, `utils/symbol_index_cache_test.go`, `compiler/lib/index_not_built_test.go`, `utils/utils.go`, `compiler/lib/module_service.go`, `compiler/lib/parser/parser.go`, `compiler/lib/parser/nested_any_test.go`)
- All 6 task commits found in git log (`dc0a7eb`, `3b8a6e9`, `aa62562`, `65968f6`, `25951b6`, `807b236`)
- `commits: 6` matches `git rev-list --count a3c8dfb..HEAD` measured directly (no narrated count)
- All plan-level `<verification>` commands re-run: `go build ./...` clean, `go vet ./compiler/... ./utils/...` clean, `go test -race ./compiler/... ./utils/...` green (cached and fresh runs)
- Every task's `<acceptance_criteria>` re-verified passing, including the `grep`-based structural checks (`ParseFilesButDoNotLink(`, zero `slog.` calls, `dirhash.HashDir(`, `os.Rename(`, `registry.CacheDir = m.getCacheDir()`, `LoadSymbolByIndex(` ordering after `LoadSymbolByScan`)

---
*Phase: 13-exact-symbol-index-shared-type-url-resolution*
*Completed: 2026-09-08*
