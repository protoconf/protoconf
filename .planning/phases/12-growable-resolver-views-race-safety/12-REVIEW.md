---
phase: 12-growable-resolver-views-race-safety
reviewed: 2026-09-08T00:00:00Z
depth: standard
files_reviewed: 2
files_reviewed_list:
  - compiler/lib/parser/parser.go
  - compiler/lib/parser/eager_resolver_fallback_test.go
findings:
  critical: 0
  warning: 0
  info: 1
  total: 1
status: issues_found
---

# Phase 12: Code Review Report (incremental — 12-04 gap closure)

**Reviewed:** 2026-09-08T00:00:00Z
**Depth:** standard
**Files Reviewed:** 2
**Status:** issues_found (info-only; no blocking or quality-degrading defects found)

## Summary

This is a scoped, incremental review covering only the 12-04 gap-closure commits
(`3401e6c` RED, `f57d3fd` GREEN) applied on top of the prior review baseline
(`a2ebd45`). The prior `12-REVIEW.md` raised **CR-01**: routing `ParseFilesX`'s
resolver read through the locked `p.registry.FindFileByPath` made the
canonical-lookup/`desc.WrapFile` fallback branch unreachable dead code for both
eager and lazy registries, silently regressing resolution of files present only
in a hand-extended `p.FilesResolver` (the mutation server's six well-known
files, `server/server.go:292-297`) — with no test exercising that state.

**CR-01 is fixed correctly by this change.** The new code adds exactly the
fallback the prior review's own fix suggestion specified: on
`errors.Is(resolverErr, utils.ErrNoGrowableResolver)`, fall through to a direct,
unlocked read of `p.FilesResolver.FindFileByPath` — which is provably
race-free for the eager path (D-03: `d.filesResolver` is never armed for an
eager registry, confirmed at `utils/utils.go:174-180`, so nothing ever grows
the object `p.FilesResolver` snapshots at construction). I traced this against
`utils/utils.go`'s `GetFilesResolver`, `FindFileByPath`, `ParseOne`, and
`FileDescriptor` to confirm:

- For an eager registry (`ImportPaths` empty), `d.filesResolver` is permanently
  `nil`, so `p.registry.FindFileByPath` always returns `ErrNoGrowableResolver`,
  and the new branch is the only way that class of registry can ever resolve a
  file that lives only in the resolver — exactly the regression CR-01
  identified.
- For a lazy registry, `NewParserWithDescriptorRegistry` calls
  `registry.GetFilesResolver()` at construction, which arms and caches
  `d.filesResolver` before any `ParseFilesX` call, so `p.registry.FindFileByPath`
  never returns `ErrNoGrowableResolver` on that path and the new branch is a
  no-op for lazy registries — no change to the lazy/compiler behavior the
  prior review found already correct.
- `protoregistry.Files.FindFileByPath` (vendor: `google.golang.org/protobuf@v1.34.1/reflect/protoregistry/registry.go:315-318`)
  explicitly nil-checks its receiver and returns `NotFound` rather than
  panicking, so even a degenerate case where `p.FilesResolver` itself ended up
  `nil` (e.g. a `protodesc.NewFiles` build failure at construction) cannot
  crash this new read — it degrades gracefully into the existing
  `ParseOne`/`ErrLazyParseDisabled` path.
- The new test, `TestParseFilesXResolvesEagerHandRegisteredFile`, builds the
  exact D-03 eager shape, hand-registers a file directly onto
  `p.FilesResolver` (mirroring `server.go`'s pattern) without adding it to
  `FileRegistry`, and asserts `ParseFilesX` resolves it — closing precisely the
  test gap CR-01 called out. Its three companion subtests (RSLV-03 adjacency,
  ordering, empty/absent) re-confirm the canonical-pointer and
  not-found/`ErrLazyParseDisabled` behavior is unchanged. I ran this test file
  and the full `compiler/lib/parser` and `compiler/lib` suites under `-race`;
  all pass.

No new Critical or Warning issues were introduced by this diff. One minor Info
item follows.

Per the phase's own scoping (12-VERIFICATION.md), `Parser.FilesResolver`'s
public/unlocked field exposure (previously WR-01) and `FileRegistry`'s
unlocked non-`ParseOne` mutators (previously WR-02) remain out of scope for
this phase and are not re-raised here as new findings; the D-03 locked
decision (12-04-PLAN.md) explicitly designates the eager, zero-growable-state
raw-field read this diff adds as race-free by construction, which the trace
above corroborates.

## Info

### IN-01: Fallback read discards the original `ErrNoGrowableResolver` context on a genuine not-found

**File:** `compiler/lib/parser/parser.go:134-136`
**Issue:** When the eager fallback also misses (`p.FilesResolver.FindFileByPath`
returns `protoregistry.NotFound`), `resolverErr` is overwritten with that
`NotFound` error, so the final `errors.Join(resolverErr, parseErr)` returned to
the caller no longer carries `utils.ErrNoGrowableResolver`. No current caller
depends on `errors.Is(err, utils.ErrNoGrowableResolver)` against `ParseFilesX`'s
return value (confirmed via repo-wide grep — the only other reference is
`growable_resolver_test.go`, which asserts against `FindFileByPath` directly,
not `ParseFilesX`), so this has no behavioral impact today. It is purely a
minor loss of diagnostic detail: a future debugger inspecting a
`ParseFilesX` not-found error for an eager registry can no longer distinguish
"no growable resolver was ever armed" from "resolver present, file absent"
by unwrapping the returned error.
**Fix:** Optional — join rather than overwrite, if the extra context is ever
wanted:
```go
if errors.Is(resolverErr, utils.ErrNoGrowableResolver) {
    fallbackFd, fallbackErr := p.FilesResolver.FindFileByPath(filename)
    if fallbackErr == nil {
        resolvedFd, resolverErr = fallbackFd, nil
    } else {
        resolverErr = errors.Join(resolverErr, fallbackErr)
    }
}
```
Not required for correctness; leaving as-is is acceptable.

---

_Reviewed: 2026-09-08T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
