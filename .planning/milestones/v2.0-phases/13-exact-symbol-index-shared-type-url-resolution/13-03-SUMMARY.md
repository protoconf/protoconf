---
phase: 13-exact-symbol-index-shared-type-url-resolution
plan: 03
subsystem: compiler
tags: [protobuf, protoreflect, type-url-resolution, symbol-index, error-handling]

requires:
  - phase: 13-exact-symbol-index-shared-type-url-resolution
    plan: 02
    provides: "The exact symbol index (Tier 3) and its IndexBuildCount/IndexCacheHitCount/IndexState observables this plan's hard error and log line consume"
provides:
  - "D-02 executed: the whole-tree eager fallback (DescriptorRegistry.ParseAll), its eagerFallback latch, and FellBackToEager() no longer exist anywhere in the tree"
  - "resolveTiers' exhausted chain returns a NotFound-wrapping hard error naming the unresolved symbol, the import roots searched, and the symbol index's build/cache state"
  - "compile-finished log line replaced in kind: symbolIndexBuilds/symbolIndexCacheHits/scanResolutions reported on every compile"
  - "LoadSymbolByIndex guarded against a fullName with no resolvable package.Message shape (empty, or no PascalCase segment), so an empty/malformed type URL never triggers a whole-tree index build"
affects: [14-consumer-migration-and-repo-wide-grep-clean]

actuals:
  tokens: 12089
  tasks: 3
  commits: 3
  plan_head_before: a199e3b

tech-stack:
  added: []
  patterns:
    - "Hard-error diagnostics wrap the sentinel rather than replace it: fmt.Errorf(\"%w: %s not found in import roots %v (symbol index: %s)\", protoregistry.NotFound, display, r.registry.ImportPaths, r.registry.IndexState()) -- every existing errors.Is(err, protoregistry.NotFound) caller keeps working unchanged"
    - "An expensive tier's adjacency guard (symbolScanCandidates' len(rest)==0 check) is reused, not reinvented, at every entry point that can trigger that tier -- LoadSymbolByIndex now applies the identical splitSymbolPackage check the scan tier already applied, closing a gap where only the scan tier was defended against a malformed symbol name"

key-files:
  created:
    - compiler/lib/parser/hard_error_test.go
  modified:
    - utils/utils.go
    - utils/symbol_index.go
    - compiler/lib/parser/parser.go
    - compiler/lib/compiler.go
    - compiler/lib/lazy_load_count_test.go
    - compiler/lib/tier_observability_test.go (renamed from eager_fallback_visible_test.go)
    - utils/index_build_deadlock_test.go (renamed from parse_all_deadlock_test.go)
    - utils/growable_resolver_test.go
    - utils/lazy_parse_canonical_test.go
    - utils/symbol_index_test.go
    - utils/symbol_scan_test.go

key-decisions:
  - "Task 1 checkpoint resolved: developer selected option 1 (\"Proceed per D-02\") — delete the whole-tree fallback, its latch and its accessor outright. Option 3 (keep the method for a non-resolution caller) was explicitly ruled out on evidence: grep -rn --include='*.go' 'ParseAll|FellBackToEager' found exactly one non-test caller of each (compiler/lib/parser/parser.go:92 inside resolution, and the compile-finished log line compiler/lib/compiler.go:221, which this plan replaces in kind), so no carve-out was needed."
  - "utils/growable_resolver_test.go's TestParseAllRegistersIntoFilesResolver and TestFilesResolverRegistrationErrorsStayZero were deleted rather than re-pointed at Import/Parse driven directly: the registration-diff behavior they pinned lived entirely inside ParseAll's own deleted before/after diff loop. Every production caller of Import/Parse (compiler/lib/module_service.go, server/server.go) always runs on an eager registry (ImportPaths empty), where GetFilesResolver rebuilds fresh from FileRegistry on every call instead of growing incrementally -- the combination these two tests exercised (ImportPaths set AND Import/Parse called directly) has no production caller anywhere in the tree. Re-pointing would only test reimplemented ParseAll glue, not a real code path. TestRegistrationCountIsIncremental in the same file already fully pins the registration-diff invariant via ParseOne, the sole remaining writer."
  - "utils/index_build_deadlock_test.go folds 13-02's already-existing utils/symbol_index_test.go#TestIndexBuildRacesParseOne into the renamed file rather than writing a second, near-duplicate race test: both would have exercised the identical hazard (LoadSymbolByIndex/buildSymbolIndex racing ParseOne under Phase 11 lock discipline) using the same deadlockCorpus helper. The test was moved into the renamed file, which is deadlockCorpus's natural home; TestIndexBuildRacesParseOne's comment/import block was removed from symbol_index_test.go accordingly."
  - "The hard error's diagnostic format is fmt.Errorf(\"%w: %s not found in import roots %v (symbol index: %s)\", protoregistry.NotFound, display, r.registry.ImportPaths, r.registry.IndexState()) -- chosen to keep the existing %w-wrapping convention already used at this exact call site pre-13-03, extended with the two additional pieces of context D-02 requires (import roots, index state) rather than inventing a new error-formatting idiom."

patterns-established:
  - "resolveTiers' final branch is the one and only place a hard NotFound error is constructed for type-URL resolution; both FindMessageByURL and FindMessageByName share it (TYPE-08), so the diagnostic content can never drift between the two entry points."

requirements-completed: [TYPE-08]

coverage:
  - id: D1
    description: "The whole-tree eager fallback (ParseAll), its eagerFallback latch, and FellBackToEager() are deleted; go build ./... is the proof no caller of any of the three survives anywhere in the tree"
    requirement: "TYPE-08"
    verification:
      - kind: unit
        ref: "go build ./... (structural: grep -rn --include='*.go' 'func (d \\*DescriptorRegistry) ParseAll\\(' . and the FellBackToEager equivalent both produce no output)"
        status: pass
    human_judgment: false
  - id: D2
    description: "An exhausted resolveTiers chain returns an error that still satisfies errors.Is(err, protoregistry.NotFound) and names the unresolved type URL, the import roots searched, and the symbol index's build-vs-cache-hit state"
    requirement: "TYPE-08"
    verification:
      - kind: unit
        ref: "compiler/lib/parser/hard_error_test.go#TestUnresolvableTypeURLErrorIsDiagnostic"
        status: pass
      - kind: unit
        ref: "compiler/lib/parser/hard_error_test.go#TestUnresolvableTypeURLReportsIndexState"
        status: pass
    human_judgment: false
  - id: D3
    description: "An empty type URL, and a type URL with no host prefix and no resolvable Message-name segment, each fail with a NotFound-wrapped error without running the scan tier or building the symbol index"
    verification:
      - kind: unit
        ref: "compiler/lib/parser/hard_error_test.go#TestEmptyTypeURLFailsWithoutScanOrIndex"
        status: pass
    human_judgment: false
  - id: D4
    description: "A symbol resolvable at Tier 0 (the construction-time snapshot) never consults the scan or index tiers; a hit at one tier never falls through to the next"
    verification:
      - kind: unit
        ref: "compiler/lib/parser/hard_error_test.go#TestTier0HitConsultsNothingElse"
        status: pass
    human_judgment: false
  - id: D5
    description: "FindMessageByURL and FindMessageByName, asked about the same unresolvable symbol, produce errors carrying identical searched-roots and index-state content, proving one resolveTiers implementation backs both entry points (TYPE-08)"
    requirement: "TYPE-08"
    verification:
      - kind: unit
        ref: "compiler/lib/parser/hard_error_test.go#TestBothEntryPointsShareTheTierChain"
        status: pass
    human_judgment: false
  - id: D6
    description: "The compile-finished log line reports symbolIndexBuilds/symbolIndexCacheHits/scanResolutions on every compile, present and zero when nothing escalated, and the reported values agree with the registry's own counters once a tier has fired"
    verification:
      - kind: unit
        ref: "compiler/lib/tier_observability_test.go#TestCompileFinishedReportsResolutionTiers"
        status: pass
    human_judgment: false
  - id: D7
    description: "The four tests broken by the deletion (eager-fallback visibility, the ParseAll/ParseOne deadlock guard, the growable-resolver registration-diff tests, and the canonical-pointer race stager) are re-pointed at the replacement mechanism or deleted with recorded reasoning -- none silently dropped"
    verification:
      - kind: unit
        ref: "utils/index_build_deadlock_test.go#TestIndexBuildRacesParseOne"
        status: pass
      - kind: unit
        ref: "utils/lazy_parse_canonical_test.go#TestParseOneReturnsCanonicalPointerWhenRacingConcurrentInsert"
        status: pass
    human_judgment: true
    rationale: "The growable-resolver disposition (delete two tests, keep their reasoning) is a judgment call about whether the deleted behavior has a real production analog -- documented in key-decisions, but whether the reasoning itself is sound is worth a human read rather than an automated pass/fail."

duration: 55min
completed: 2026-09-08
status: complete
---

# Phase 13 Plan 3: Delete the Whole-Tree Eager Fallback, Replace With a Hard Error Summary

**D-02 executed: `DescriptorRegistry.ParseAll`, its `eagerFallback` latch, and `FellBackToEager()` are gone from the tree; an exhausted `resolveTiers` chain now fails fast with a `protoregistry.NotFound`-wrapping error naming the unresolved symbol, the import roots searched, and the symbol index's build/cache state, and the compile-finished log line reports the three tier counters (`symbolIndexBuilds`/`symbolIndexCacheHits`/`scanResolutions`) in the old boolean's place.**

## Performance

- **Duration:** ~55 min (continuing from the Task 1 checkpoint's developer response)
- **Started:** 2026-09-08 (approx., continuation agent)
- **Completed:** 2026-09-08
- **Tasks:** 3 (Task 1 resolved by developer decision, no code; Tasks 2-3 executed)
- **Files modified:** 12 (1 created, 2 renamed, 9 modified)

## Accomplishments
- **Task 1 (checkpoint:decision, resolved):** The developer selected option 1, "Proceed per D-02." The orchestrator's own `grep -rn --include='*.go' 'ParseAll|FellBackToEager'` found exactly one non-test caller of each (`compiler/lib/parser/parser.go:92`, inside type-URL resolution; and the compile-finished log line `compiler/lib/compiler.go:221`, which this plan replaces in kind) — no carve-out for a surviving caller was needed, so option 3 did not apply.
- **Task 2:** Deleted `DescriptorRegistry.ParseAll`, the `eagerFallback` boolean latch, and `FellBackToEager()` from `utils/utils.go`. `resolveTiers` in `compiler/lib/parser/parser.go` no longer retries via the whole-tree fallback on exhaustion; it returns a hard error wrapping `protoregistry.NotFound` with the unresolved symbol, `r.registry.ImportPaths`, and `r.registry.IndexState()`. `compiler/lib/compiler.go`'s compile-finished line now reports `symbolIndexBuilds`, `symbolIndexCacheHits`, and `scanResolutions` (all present, all zero on a normal compile) instead of the old `eagerFallback` boolean.
- Re-pointed the four tests the deletion broke:
  - `compiler/lib/eager_fallback_visible_test.go` → `compiler/lib/tier_observability_test.go`: `TestCompileFinishedReportsEagerFallback` → `TestCompileFinishedReportsResolutionTiers`, asserting the three new counters instead of the boolean, and driving an escalation via a direct `registry.SymbolFile(...)` call (the D-02 replacement for the old test's direct `reg.ParseAll()` call) rather than a fallback that no longer exists.
  - `utils/parse_all_deadlock_test.go` → `utils/index_build_deadlock_test.go`: kept `deadlockCorpus`, the 30s `select` guard, the vacuous-pass guard, and the `require.Same` pointer-identity loop; replaced the ParseAll-driving goroutine with one driving `LoadSymbolByIndex`. Folded in 13-02's already-existing near-duplicate `TestIndexBuildRacesParseOne` from `utils/symbol_index_test.go` rather than keeping two copies of the same race test.
  - `utils/lazy_parse_canonical_test.go`: `TestParseOneReturnsCanonicalPointerWhenRacingParseAll` → `TestParseOneReturnsCanonicalPointerWhenRacingConcurrentInsert`. `afterParseHook` now stages the same interleaving by independently re-parsing `canon.proto` and inserting it via a direct, locked `recordFileLocked` call instead of `d.ParseAll()`.
  - `utils/growable_resolver_test.go`: `TestParseAllRegistersIntoFilesResolver` and `TestFilesResolverRegistrationErrorsStayZero` were deleted with reasoning recorded in a comment and in this SUMMARY's key-decisions — the registration-diff behavior they pinned lived entirely inside `ParseAll`'s own deleted diff loop, with no production caller of `Import`/`Parse` on a lazy registry to re-point at.
  - Also fixed `compiler/lib/lazy_load_count_test.go`, a caller of the deleted `FellBackToEager()` accessor that the plan's own `<files>` list did not name (Rule 3: blocking compile issue) — replaced with the equivalent `ScanResolutionCount()`/`IndexBuildCount()` zero-assertions.
- **Task 3 (TDD):** Added `compiler/lib/parser/hard_error_test.go` pinning five behaviors. RED: `TestEmptyTypeURLFailsWithoutScanOrIndex` genuinely failed — `LoadSymbolByIndex` had no guard against a `fullName` with no resolvable `package.Message` shape (empty string, or a bare lowercase word with no PascalCase segment), so it unconditionally built the whole-tree symbol index even for input that could never resolve (`IndexBuildCount()` was 1, expected 0). The other four tests in the file already passed against the Task 2 implementation. GREEN: added the same adjacency guard `symbolScanCandidates` already applies to the scan tier — `splitSymbolPackage(fullName)` yielding no type-name segment — to `LoadSymbolByIndex`.

## TYPE-08 structural check (recorded per Task 3's `<output>` spec)

```
$ grep -rn --include='*.go' 'FindMessageTypeByUrl' compiler/
compiler/lib/config.go:66:              md, err := c.messageRegistry.FindMessageTypeByUrl(result.GetTypeUrl())
compiler/lib/starlark_loader.go:187:    d, err := l.moduleService.GetProtoRegistry().MessageRegistry.FindMessageTypeByUrl(protoconfValue.Value.TypeUrl)
compiler/lib/parser/parser.go:80:      if md, mErr := r.registry.MessageRegistry.FindMessageTypeByUrl(url); mErr == nil && md != nil {
compiler/lib/parser/parser.go:84:              if md, mErr := r.registry.MessageRegistry.FindMessageTypeByUrl(url); mErr == nil && md != nil {
compiler/lib/parser/parser.go:89:              if md, mErr := r.registry.MessageRegistry.FindMessageTypeByUrl(url); mErr == nil && md != nil {
```

The three `parser.go` hits all live inside `RegistryTypeResolver.resolveTiers` itself — the one shared tier chain. `compiler/lib/config.go:66` and `compiler/lib/starlark_loader.go:187` are direct `MessageRegistry.FindMessageTypeByUrl` calls that bypass `TypeResolver` entirely — both are pre-existing, already-flagged divergences (`starlark_loader.go:187` is the CONS-05 divergence 13-CONTEXT.md names explicitly; `config.go:66`'s sibling `ErrUnexpectedType` silent-skip at `:68` is the flagged-assumption item explicitly marked out of scope for this phase, carried into Phase 14 alongside CONS-02/03/04). Neither was introduced or touched by this plan.

**Stating explicitly, per Task 3's action text:** the repository-wide grep-clean state does **not** complete in this phase. `inserter/inserter.go:369`, `server/server.go:607`, and `mutate/mutate.go:76` all still call `LocalResolver`/`anyResolver` directly (confirmed this session: `grep -rn --include='*.go' 'FindMessageByURL\|FindMessageByName' inserter/ server/ mutate/'`), exactly matching 13-CONTEXT.md's D-03 fence. ROADMAP.md success criterion 1's grep clause completes in Phase 14.

## Example of the new hard error's full text

```
proto: not found: type.googleapis.com/ghost.v1.TrulyMissing not found in import roots [/var/folders/.../src] (symbol index: rebuilt)
```

## Task Commits

Task 1 was a `checkpoint:decision` resolved by the developer with no code change (see Decisions Made below) — no commit.

1. **Task 2: Delete the whole-tree fallback and repair every caller and test in one atomic change** — `862410e` (feat)
2. **Task 3: Pin the hard error's diagnostic content and tier ordering** — RED: `29a2d67` (test), GREEN: `684a731` (feat). No REFACTOR commit: the GREEN implementation (a four-line guard reusing an existing helper) needed no follow-up cleanup.

**Plan metadata:** commit to follow (docs: complete plan)

## Files Created/Modified
- `compiler/lib/parser/hard_error_test.go` - Five tests pinning the hard error's diagnostic content and the D-01 tier ordering (Task 3)
- `utils/utils.go` - Removed `ParseAll`, `eagerFallback` field, `FellBackToEager()`; updated stale comments referencing them
- `utils/symbol_index.go` - `LoadSymbolByIndex` now guards against a `fullName` with no resolvable type-name shape (Task 3 GREEN)
- `compiler/lib/parser/parser.go` - `resolveTiers` no longer retries via `ParseAll`; builds the D-02 hard error naming the URL, import roots, and index state
- `compiler/lib/compiler.go` - Compile-finished log line reports `symbolIndexBuilds`/`symbolIndexCacheHits`/`scanResolutions` instead of `eagerFallback`
- `compiler/lib/lazy_load_count_test.go` - Replaced the deleted `FellBackToEager()` assertion with the equivalent tier-counter zero-assertions (Rule 3 fix, caller the plan's file list missed)
- `compiler/lib/tier_observability_test.go` (renamed from `eager_fallback_visible_test.go`) - Re-pointed at the three tier counters
- `utils/index_build_deadlock_test.go` (renamed from `parse_all_deadlock_test.go`) - Re-pointed at `LoadSymbolByIndex`; folds in 13-02's `TestIndexBuildRacesParseOne`
- `utils/growable_resolver_test.go` - Deleted the two ParseAll-specific tests with reasoning recorded
- `utils/lazy_parse_canonical_test.go` - `afterParseHook` now stages a direct concurrent insert instead of calling `ParseAll`
- `utils/symbol_index_test.go` - Removed the now-duplicate `TestIndexBuildRacesParseOne` and its now-unused `fmt`/`time` imports
- `utils/symbol_scan_test.go` - Updated a stale comment reference to the renamed deadlock-test file

## Decisions Made
See `key-decisions` in frontmatter. In prose:
- **Task 1's checkpoint resolved with option 1** ("Proceed per D-02"), on the orchestrator's own grep evidence that no non-resolution caller of `ParseAll`/`FellBackToEager` exists.
- **The two growable-resolver tests were deleted, not re-pointed**, because the behavior they pinned (bulk-Import/Parse registering into a growable resolver via a before/after diff) lived entirely inside `ParseAll`'s own deleted loop and has no production analog — every real `Import`/`Parse` caller runs on an eager registry.
- **`TestIndexBuildRacesParseOne` was folded rather than duplicated**, since 13-02 Task 3 had already added an identical race test to `utils/symbol_index_test.go` before this plan existed.
- **The hard error's format extends the existing `%w`-wrapping convention** already present at this call site pre-13-03, rather than introducing a new error-formatting idiom.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `compiler/lib/lazy_load_count_test.go` called the deleted `FellBackToEager()` accessor**
- **Found during:** Task 2, after removing `FellBackToEager()` from `utils/utils.go`
- **Issue:** This file is not in the plan's `<files>` list for Task 2, but it called `c.ModuleService.GetProtoRegistry().FellBackToEager()` — a compile-blocking reference to a deleted method. The plan's own read of the codebase (and the orchestrator's grep) found only `compiler/lib/compiler.go:221` as the non-test caller of `FellBackToEager`; this file is a *test* caller the grep's own scope (non-test callers) did not surface.
- **Fix:** Replaced the single `require.False(t, ... .FellBackToEager())` assertion with `require.Equal(t, 0, registry.ScanResolutionCount())` and `require.Equal(t, 0, registry.IndexBuildCount())`, preserving the test's original intent (the corpus never resolves an unknown type URL, so no tier past the growable MessageRegistry should ever fire) under the replacement observability contract.
- **Files modified:** `compiler/lib/lazy_load_count_test.go`
- **Verification:** `go test -race ./compiler/lib/... -run TestLoadedFileCount -v` — PASS
- **Committed in:** `862410e` (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking-compile-issue, Rule 3 class)
**Impact on plan:** No scope or correctness impact — a genuine compile-blocking caller the plan's file list missed, fixed in kind with the same replacement counters used everywhere else in this plan.

## Issues Encountered
- `go test -race ./...` (the full repository suite, in this plan's own `<verification>` block) reproduces the same `github.com/protoconf/protoconf/agent` `Conductor.playWithLogger` hang 13-02 already logged — unrelated to any file this plan touches (`agent/*.go` is not in this plan's file list, and neither the deleted `ParseAll`/`FellBackToEager` symbols nor `LoadSymbolByIndex`'s new guard are reachable from `agent`'s orchestra-based process-lifecycle tests). Logged to `.planning/phases/13-exact-symbol-index-shared-type-url-resolution/deferred-items.md` under a new "13-03" entry, matching 13-01/13-02's precedent. The plan-scoped verification this plan's own tasks actually require — `go test -race ./compiler/... ./utils/...` — is fully green: `compiler` 32.5s, `compiler/lib` 161.3s, `compiler/lib/parser` 36.0s, `utils` 15.4s, all `ok`. `go build ./...` and `go vet ./compiler/... ./utils/...` are clean; `go vet ./...` (whole-repo) surfaces the same three pre-existing findings 13-01/13-02 already logged (`test/e2e_test.go`, `agent/agent_test.go`, `agent/legacy.go`), none in this plan's files.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- D-02 is fully executed: the whole-tree eager fallback is gone, and `go build ./...` is the compile-time proof no caller of it survives anywhere in the tree.
- The hard error's diagnostic content (URL, import roots, index state) and the D-01 tier ordering (Tier 0 through Tier 3, hard error last) are pinned by `hard_error_test.go` and will not silently regress.
- `TYPE-08` is now uniquely owned by this plan (13-01 and this plan both declared it; 13-01's SUMMARY already noted the shared-ID gate deferred marking it complete until this plan finished) and is ready to mark complete in `REQUIREMENTS.md`.
- Phase 14's known scope is unchanged and explicitly re-confirmed this session: `inserter/inserter.go:369`, `server/server.go:607`, `mutate/mutate.go:76` (CONS-02/03/04), plus `compiler/lib/config.go:66/68`'s silent-skip validation call site and `compiler/lib/starlark_loader.go:187`'s CONS-05 divergence, all still bypass the shared `RegistryTypeResolver.resolveTiers` chain. ROADMAP.md success criterion 1's grep clause completes in Phase 14, not this phase.

## Self-Check: PASSED

- All key files found on disk: `compiler/lib/parser/hard_error_test.go`, `compiler/lib/tier_observability_test.go`, `utils/index_build_deadlock_test.go` exist; `compiler/lib/eager_fallback_visible_test.go` and `utils/parse_all_deadlock_test.go` do not
- All 3 task commits found in git log (`862410e`, `29a2d67`, `684a731`)
- `commits: 3` matches `git rev-list --count a199e3b..HEAD` measured directly (no narrated count)
- All plan-level `<verification>` commands re-run: `go build ./... && go vet ./compiler/... ./utils/...` clean; `go test -race ./compiler/... ./utils/...` green
- Every task's `<acceptance_criteria>` re-verified passing, including the `grep`-based structural checks for `ParseAll`/`FellBackToEager`/`eagerFallback`/`symbolIndexBuilds` and the renamed-file existence checks

---
*Phase: 13-exact-symbol-index-shared-type-url-resolution*
*Completed: 2026-09-08*
