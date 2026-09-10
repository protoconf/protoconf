---
phase: 13-exact-symbol-index-shared-type-url-resolution
reviewed: 2026-09-08T06:46:32Z
depth: standard
files_reviewed: 21
files_reviewed_list:
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
  - utils/growable_resolver_test.go
  - utils/index_build_deadlock_test.go
  - utils/lazy_parse_canonical_test.go
  - utils/symbol_index_cache_test.go
  - utils/symbol_index_test.go
  - utils/symbol_index.go
  - utils/symbol_scan_test.go
  - utils/symbol_scan.go
  - utils/testdata/small/mutable_config/nested_any_mutation.materialized_JSON
  - utils/testdata/small/src/load_mutable_nested_any_test.pconf
  - utils/utils.go
findings:
  critical: 0
  warning: 2
  info: 1
  total: 3
status: issues_found
---

# Phase 13: Code Review Report

**Reviewed:** 2026-09-08T06:46:32Z
**Depth:** standard
**Files Reviewed:** 21
**Status:** issues_found

## Summary

Phase 13 replaces the compiler's whole-tree eager-fallback (`ParseAll`, which held `d.mu` across an entire `protoparse` call — a lock-discipline defect in itself) with a three-tier chokepoint (`resolveTiers`): growable `MessageRegistry`, then a scoped lexical scan (`LoadSymbolByScan`), then an exact symbol index (`LoadSymbolByIndex`), finally a hard `protoregistry.NotFound`-wrapping error. I traced every new lock path in `utils/utils.go` and `utils/symbol_index.go` and did not find a case where `d.mu` is held while calling into `protoparse` — `ParseOne`, `buildSymbolIndex`, and `symbolScanCandidates` all release the lock before doing filesystem/parse work, matching the documented contract and backed by `utils/index_build_deadlock_test.go` and `utils/symbol_scan_test.go`'s race tests.

The hard-error path in `compiler/lib/parser/parser.go` correctly wraps `protoregistry.NotFound` via `%w`, verified by `errors.Is` in `hard_error_test.go`. `starlark_loader.go`'s `loadMutable` is nil-safe end to end (`GetValue().GetTypeUrl()`) and resolves entirely through the shared `TypeResolver`, matching the CONS-05 intent and the added regression test. The two deleted `growable_resolver_test.go` cases pinned behavior that lived entirely inside the now-deleted `ParseAll`; I confirmed by grep that `ParseAll`/`FellBackToEager` no longer exist anywhere in the tree and that `TestRegistrationCountIsIncremental` (retained) already covers the registration-diff invariant via `ParseOne`, the sole remaining writer on the lazy/growable path — the deletion is not a coverage loss.

One real gap: the symbol index cache's entry-count validation counts raw lines, not the number of distinct keys actually installed into the resulting map, so a cache file with a duplicate-key line pair can pass validation while silently producing fewer effective entries than the header claims. See WR-01.

## Warnings

### WR-01: Symbol index cache entry-count check validates line count, not resulting map size

**File:** `utils/symbol_index.go:211-224`
**Issue:** `loadSymbolIndexCache` checks `len(entries) != wantCount` (a count of raw lines) and then builds `index` via `index[parts[0]] = parts[1]` in a loop with no duplicate-key detection. If a cache file contains two lines for the same symbol (e.g. a hand-corrupted file, a downgraded/older writer, or partial disk corruption that duplicates a line rather than truncating it), the line count still matches the header's declared count, but the resulting map has fewer distinct entries than declared — silently. This violates the stated contract in the function's own doc comment: "refusing it ... on any of: ... an entry count that differs from the number of parsed entries." The check as written compares against the number of *lines*, not the number of *entries actually present in the map after parsing*.

The blast radius is bounded (a downstream `ParseOne` on the wrong/missing path still fails safely via `filepath.IsLocal`'s path-escape guard), so this does not create a crash or a path-traversal issue — but it is a genuine, currently-untested integrity gap in the exact place the phase's cache-integrity requirement targets, and `writeSymbolIndexCache`'s own line-oriented format has no way to distinguish "duplicate key, benign" from "duplicate key, corruption."

**Fix:**
```go
index := make(map[string]string, wantCount)
for _, line := range entries {
    parts := strings.Split(line, "\t")
    if len(parts) != 2 {
        return nil, fmt.Errorf("symbol index cache %s: malformed entry line %q", path, line)
    }
    if _, dup := index[parts[0]]; dup {
        return nil, fmt.Errorf("symbol index cache %s: duplicate symbol %q", path, parts[0])
    }
    index[parts[0]] = parts[1]
}
if len(index) != wantCount {
    return nil, fmt.Errorf("symbol index cache %s: entry count mismatch: header says %d, found %d distinct entries", path, wantCount, len(index))
}
return index, nil
```

### WR-02: `symbolIndexContentKey` errors and `buildSymbolIndex` errors are structurally unreachable/silently absorbed, hiding a fully-unreadable-root case as a normal "rebuilt" empty index

**File:** `utils/symbol_index.go:41-64`, `utils/symbol_index.go:271-274`
**Issue:** `buildSymbolIndex` always returns a `nil` error (line 63: `return index, nil`), even when `parseUnlinked`'s batch call fails for every file and the per-file fallback also fails for every file (all `oneErr != nil`, `continue`d). The only visible symptom is an empty (or partially empty) index reported as `indexState = "rebuilt"` — indistinguishable from "the tree legitimately has no matching symbol yet." Consequently, `ensureSymbolIndex`'s `if buildErr != nil { return nil, buildErr }` branch (line 272-274) is dead code under the current implementation: nothing can ever populate `buildErr`. If the import root becomes fully unreadable (e.g. a permissions problem, or a root that silently stopped existing after `ImportPaths` was set), the operator gets a hard `NotFound` error whose `IndexState()` says "rebuilt" — which reads as "the index tried and the symbol just isn't there" rather than "the index build could not read anything." This is consistent with the documented D-02 philosophy of degrading rather than hard-failing on a broken file nobody asked about, but a **fully** unreadable root is a different failure class than "one broken file among many," and today it is silently folded into the same "rebuilt but empty" bucket with no diagnostic signal.
**Fix:** At minimum, track whether every file across every root failed to parse (batch and per-file) and surface that in `indexState` (e.g. `"rebuilt (0 files parsed of N found)"`) so the hard error's `IndexState()` string gives an operator something actionable when `LoadSymbolByIndex` misses. If `buildSymbolIndex`'s error return is intentionally vestigial, consider dropping it (or documenting explicitly why it can never fire) so a future reader does not assume `ensureSymbolIndex`'s error branch is live.

## Info

### IN-01: `writeSymbolIndexCache`/cache directory share `.protoconf_cache` with module-dependency artifacts without a dedicated subpath

**File:** `compiler/lib/module_service.go:456-458`, `utils/symbol_index.go:19`
**Issue:** `registry.CacheDir = m.getCacheDir()` points the symbol index cache (`symbol_index.v1`) at the same top-level `.protoconf_cache/` directory used for downloaded-module `.fds` files and repo checkouts (`repoCacheDir`, `<label>.fds`). The name `symbol_index.v1` is unlikely to collide with a repo label or `.fds` file today, but there's no directory-level isolation (e.g. a `symbol-index/` subdirectory) between "content-addressed derived cache Phase 13 owns" and "external download cache other subsystems own," which makes it easy for a future addition on either side to introduce an actual collision.
**Fix:** Not urgent; consider `filepath.Join(m.getCacheDir(), "symbol-index", symbolIndexCacheFile)` or similar namespacing if another cache-file-producing feature is added to the same directory.

---

_Reviewed: 2026-09-08T06:46:32Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
