---
phase: 14-non-compiler-consumer-correctness
reviewed: 2026-09-08T00:00:00Z
depth: standard
files_reviewed: 4
files_reviewed_list:
  - agent/filekv/filekv.go
  - agent/filekv/filekv_test.go
  - server/mutate_config_path_test.go
  - server/server.go
findings:
  critical: 0
  warning: 3
  info: 1
  total: 4
status: issues_found
---

# Phase 14: Code Review Report

**Reviewed:** 2026-09-08T00:00:00Z
**Depth:** standard
**Files Reviewed:** 4
**Status:** issues_found

## Summary

This is an incremental review of gap-closure plan 14-09: `agent/filekv.resolveKeyPath` (shared by `Get`/`Watch`) and a containment check added to `server.MutateConfig` on the caller-supplied `in.Path`. Both checks were traced against the specific failure modes the prior review (`0ebb5b6`, CR-01/WR-03) recorded: prefix-match pitfalls, `..`-prefixed keys that `filepath.Clean` cannot collapse, and whether the guard precedes every filesystem/exec side effect.

**Both fixes are correct for the lexical containment they claim.** `resolveKeyPath`'s `filepath.Rel`-based check (not a naive string-prefix check) correctly rejects sibling-directory-name-prefix escapes, and correctly rejects every traversal key traced through it (`../secret/leak`, `../../x`, a bare `..`, an absolute-looking key) — confirmed by tracing `filepath.Join`/`filepath.Clean`'s actual string-cleaning behavior against each case, not just by reading the code. `server.MutateConfig`'s new check is placed before every side effect on the write path (protojson marshal, pre/post scripts, `MkdirAll`, `WriteFile`) exactly as its comment claims, and the legacy gRPC service (`server/legacy.go`) and the dynamic `Put` handler both route through the same `MutateConfig`, so there is no bypass via an alternate entry point. I did not find a working traversal bypass in either check.

What remains: the two containment checks are two independent, differently-strict implementations of the same pattern (a duplication the `resolveKeyPath` refactor explicitly set out to eliminate at the `Get`/`Watch` level, but not across packages); the symlink caveat that `filekv` documents and deliberately accepts is left undocumented for the higher-stakes write path in `server.go`; and `MutateConfig` silently accepts an empty `in.Path` instead of rejecting it, producing a confusingly-named file rather than a clear validation error.

## Warnings

### WR-01: Containment logic is duplicated across packages with different strictness, reintroducing the drift risk the refactor was meant to close

**File:** `agent/filekv/filekv.go:122-135` vs `server/server.go:517-528`
**Issue:** `resolveKeyPath`'s own doc comment (`filekv.go:101-103`) states its purpose is to be "the single place in this package that turns a caller-supplied key into a filesystem path... so the two call sites cannot drift apart." That's a direct response to the prior review's WR-03, which flagged `Get` and `Watch` drifting apart. But the same shape of check now exists a second time in `server.MutateConfig`, as an inline block with no shared helper, and it is *not* equivalent:

- `resolveKeyPath` rejects the raw key outright unless it already equals `filepath.ToSlash(filepath.Clean(key))` — e.g. `"./x"`, `"a//b"`, `"a/../b"` are rejected even though they'd resolve to a contained path.
- `server.MutateConfig` never checks the raw `in.Path` form at all; it runs `filepath.Clean(in.Path)` unconditionally and only checks the *result* for containment via `filepath.Rel`. The same three examples above are silently accepted and normalized.

Neither is unsafe on its own (both correctly reject real escapes), but there are now two independently-maintained implementations of a security-relevant lexical check with different behavior for the same input class, in two different packages, with no shared test or shared code to keep them in sync. That is exactly the condition that produced the prior CR-01 (a variant of this check drifted in `Get` vs. the fix, and the regression test didn't catch it).
**Fix:** Extract the "clean-and-contain" check into one small shared helper (e.g. a `pathsafe.ResolveContained(base, key string) (string, error)` in a tiny new package, or on `consts`) and have both `filekv.resolveKeyPath` and `server.MutateConfig` call it:
```go
// pathsafe.ResolveContained joins key under base and guarantees the result
// stays inside base, rejecting any non-canonical raw key form as well as
// any lexical escape.
func ResolveContained(base, key string) (string, error) {
	if key == "" || key != filepath.ToSlash(filepath.Clean(key)) {
		return "", fmt.Errorf("%w: path=%s", ErrInvalidPath, key)
	}
	abs := filepath.Join(base, key)
	rel, err := filepath.Rel(base, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: path=%s", ErrInvalidPath, key)
	}
	return abs, nil
}
```
Callers append their own suffix (extension) before or after calling, as needed.

### WR-02: Write-path containment check has no symlink caveat, unlike the read-path check it mirrors

**File:** `server/server.go:517-528`
**Issue:** `filekv.resolveKeyPath` carries an explicit `ponytail:` comment (`filekv.go:116-121`) acknowledging the check is lexical only — a symlink planted inside `protoconfRoot` pointing outside it defeats it — and argues the risk is acceptable because planting such a symlink already requires write access to the config repo. `server.MutateConfig`'s new check is the same lexical pattern (`filepath.Clean` + `filepath.Rel`, no `EvalSymlinks`), applied to the *write* path, but carries no equivalent note.

The asymmetry matters here more than it might first appear: `MutateConfig` itself can only create regular files (`os.WriteFile`) and directories (`os.MkdirAll`), so it can't plant the symlink itself — but if a symlink is ever planted under `mutable_config/` by any other means (a prior less-strict server version, a manual ops action, a bind-mount, an admin script), every subsequent "contained" `MutateConfig` call whose path traverses through that symlinked directory component silently writes outside `mutableConfigBase` while this check reports success. That is a stronger consequence than `filekv`'s read-path case (an unexpected read vs. an unexpected write), so the same accepted-risk reasoning deserves being made explicit here rather than silently inherited.
**Fix:** At minimum, mirror the `ponytail:` comment here with the write-path-specific risk noted explicitly. If the acceptance bar is different for a write path, consider resolving `mutableConfigBase` (and `filepath.Dir(filename)`) via `filepath.EvalSymlinks` once and re-checking containment against the resolved form before `MkdirAll`/`WriteFile` — the cost is one extra syscall on the already-infrequent mutation path, not the hot `Get` path where the `ponytail` comment's cost argument was made.

### WR-03: `MutateConfig` accepts an empty `in.Path` and silently writes a confusingly-named file instead of rejecting it

**File:** `server/server.go:518, 526-528`
**Issue:** `filekv.resolveKeyPath` explicitly rejects `key == ""` (`filekv.go:123`). `server.MutateConfig`'s new containment check has no equivalent: `filepath.Clean("")` returns `"."`, and because the extension is concatenated onto the cleaned string *before* `filepath.Join` (`filename := filepath.Join(mutableConfigBase, filepath.Clean(in.Path)+consts.CompiledConfigExtension)`), an empty `in.Path` produces the literal filename `..materialized_JSON` (two leading dots, not a `..` traversal segment) — which passes the containment check (it's a normal file under `mutableConfigBase`) and is silently written, scripted around, and (when `s.compiler` is set) picked up by the subsequent compile pass over `mutable_config/`. A client bug that leaves `Path` unset produces a working-but-bizarre config entry instead of a clear error.
**Fix:** Reject the degenerate case explicitly, alongside the existing containment check:
```go
if in.Path == "" || filepath.Clean(in.Path) == "." {
    return nil, logError(fmt.Errorf("mutation path must not be empty"))
}
```
placed before `filename` is computed, so the empty-path rejection reads the same way as the containment rejection that follows it.

## Info

### IN-01: Stale test comment describes behavior the test doesn't exercise

**File:** `agent/filekv/filekv_test.go:418-421`
**Issue:** `TestWatch_ContextCancellation`'s body comment says "Let's test invalid path validation instead" and describes a `materialized_config` subdirectory scenario that the test doesn't actually build or assert against — the test just calls `Watch` with a nonexistent path. The comment reads as leftover exploration notes rather than documentation of the test's actual intent, which could mislead a future reader trying to understand what invariant this test protects.
**Fix:** Trim the comment to state what the test actually checks (`Watch` returns an error for a path whose target file doesn't exist), or fold it into the existing `TestWatch_NonExistentFile` test above it, which already covers the same behavior.

---

_Reviewed: 2026-09-08T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
