---
phase: 11-concurrency-safe-lazy-registry-core
reviewed: 2026-09-07T21:10:00Z
depth: standard
files_reviewed: 14
files_reviewed_list:
  - compiler/lib/eager_fallback_visible_test.go
  - compiler/lib/mod_sync_fds_test.go
  - compiler/lib/module_service.go
  - compiler/lib/module_service_test.go
  - compiler/lib/registry_cache_test.go
  - mod/command.go
  - mod/command_test.go
  - server/init_order_test.go
  - server/init_prohibitions_test.go
  - server/server.go
  - utils/lazy_parse_canonical_test.go
  - utils/lazy_parse_error_test.go
  - utils/parse_all_deadlock_test.go
  - utils/utils.go
findings:
  critical: 0
  warning: 2
  info: 3
  total: 5
status: issues_found
---

# Phase 11: Code Review Report

**Reviewed:** 2026-09-07T21:10:00Z
**Depth:** standard
**Files Reviewed:** 14
**Status:** issues_found

## Summary

This is an incremental review. The prior report (commit `f8022cf`, now overwritten by this
file) covered `e3f8b3f`/`c32b74f` and raised WR-01..WR-05 and IN-01/IN-02. Since then, five
commits landed: `0713f86` (WR-02 fix — `ParseOne` returns the canonical descriptor when
racing `ParseAll`), `84efad0` (WR-04 fix — `GetProtoRegistry`'s cache is now
double-checked-locked under `m.mutex`), `60459de` (Init now logs services it considers but
cannot register, plus an order-independence test), `243ab6f` (G-11-3 — `Init` now loads the
lock file itself and returns its parse error before applying the CONFIGSPACE merge;
`LoadFromLockFile` restores the `Deps` invariant after a resetting unmarshal; `MergeLock` no
longer reloads and discards `Init`'s merge; `walk()`'s per-branch error is now correctly
accumulated instead of discarded on every iteration), and `7ca3a16` (G-11-7 — `GenFileDescriptorSet`
now refuses to `Store()` an empty descriptor set and reports which dependency/paths were
searched).

I re-verified WR-02 and WR-04 are genuinely fixed (code + the new regression tests, which I
ran with `-race`; all pass). I confirmed by tracing every production caller that `registry.Store()`
(the only writer of `.fds` files) is reachable exclusively through `GenFileDescriptorSet`, so the
new G-11-7 guard cannot be bypassed by any other path. `go build ./...` is clean.

WR-01 (`ParseAll` holds `d.mu` across the whole-tree parse) was **not** fixed — it remains
exactly as before, now explicitly accepted as documented risk and backstopped only by a
deadlock-freedom test (`utils/parse_all_deadlock_test.go`), which pins termination, not the
lock-duration/serialization concern the original finding raised. I'm carrying it forward
below. WR-03 (`parser.go`'s `ParseAll()` error swallowed silently) and WR-05 (the concurrent-compile
test's doc comment overstating what it exercises) live in files not in this review's scope
(`compiler/lib/parser/parser.go`, `compiler/lib/concurrent_compile_test.go` are unchanged
since `f8022cf` — confirmed via `git diff f8022cf..HEAD --stat` returning nothing for
either) — noting them here so they aren't silently dropped, but I have not re-verified them
this session.

This pass also surfaced one new, previously-unflagged lock-discipline inconsistency in the
new `LocalFileCount()` accessor (utils/utils.go), and one code-quality inconsistency in
`mod/command.go` where the G-11-3 cleanup was applied to `modInitCommand.Run` but not to the
structurally identical `modTidyCommand.Run`.

## Warnings

### WR-01 (carried forward, still open): `ParseAll` still holds `d.mu` across the entire whole-tree parse

**File:** `utils/utils.go:352-373` (`ParseAll`), compare with `ParseOne`'s documented discipline at `utils/utils.go:265-269`

**Issue:** Unchanged since the last review. `ParseAll` takes `d.mu.Lock()` and holds it via
`defer` across the full `d.Import(d.Parse, ...)` call — a recursive walk-and-parse of every
`.proto` under `ImportPaths`. Every other concurrent caller (`ParseOne`, `FileDescriptor`,
`GetFileDescriptorSet`, `LoadedFileCount`, `LocalFileCount`, `FellBackToEager`) blocks on
`d.mu` for the full duration. This phase's own new test,
`TestParseAllConcurrentWithParseOneDoesNotDeadlock` (`utils/parse_all_deadlock_test.go`),
confirms this doesn't deadlock, but by its own doc comment only pins "the one genuinely
testable half of WR-01" — termination, not the serialization/lock-duration property the
original finding raised. Since `ParseAll` fires from inside `RegistryTypeResolver.FindMessageByURL`/`FindMessageByName`
mid-compile (unchanged, in `compiler/lib/parser/parser.go`, out of this review's scope but
unmodified since `f8022cf`), one goroutine hitting a D-03 type-URL miss still stalls every
other concurrently-compiling file on the same shared registry for the whole-tree parse
duration.

**Fix:** Unchanged from the prior report — release `d.mu` before calling `d.Import`/`d.Parse`
in `ParseAll`, re-acquiring it only to merge results and set `d.eagerFallback = true`, or gate
the parse with a second `sync.Once`/singleflight key instead of the write lock.

---

### WR-02 (new): `LocalFileCount()`'s `d.mu.RLock()` does not synchronize with `localFiles`'s only production writer

**File:** `utils/utils.go:390-394` (`LocalFileCount`), `utils/utils.go:210-222` (`Parse`, the sole writer), `utils/utils.go:59` (the mutex's documented scope)

**Issue:** The struct's own comment on `mu` is explicit about what it protects: *"guards
FileRegistry, lazyLoaded, eagerFallback on the lazy path"* — `localFiles` is deliberately
excluded, and `ParseOne`'s doc comment reinforces why: *"ParseOne never touches localFiles
(LAZY-03)"*. That's consistent for the lazy path. But the new `LocalFileCount()`, added in
this diff to back G-11-7's empty-descriptor-set guard, takes `d.mu.RLock()` around reading
`d.localFiles`:

```go
func (d *DescriptorRegistry) LocalFileCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.localFiles)
}
```

Meanwhile `Parse()` — the only production writer of `localFiles`, invoked via
`registry.Import(registry.Parse, ...)` from `GenFileDescriptorSet` (`compiler/lib/module_service.go:361`)
— writes it with no lock at all:

```go
func (d *DescriptorRegistry) Parse(parser *protoparse.Parser, files []string) error {
	d.localFiles = map[string]struct{}{}          // unguarded write
	descriptors, err := parser.ParseFiles(files...)
	for _, fd := range descriptors {
		...
		d.localFiles[fd.GetName()] = struct{}{}    // unguarded write
	}
	...
}
```

Today this is not a live race: `GenFileDescriptorSet` calls `Import`/`Parse` to completion,
then calls `LocalFileCount()` synchronously afterward on the same goroutine, and `Sync()`'s
walk over dependencies is single-threaded (no goroutines), so there's no actual concurrent
access in the current call graph — this is the same "latent, not yet live" class of issue the
prior review accepted as WR-04 before it was fixed. The `RLock()` in `LocalFileCount()` gives
a false impression that `localFiles` is guarded the same way `FileRegistry`/`lazyLoaded`/`eagerFallback`
are; it isn't, and a future change that calls `GenFileDescriptorSet`/`LocalFileCount` from a
different goroutine than the one running `Parse` (e.g., parallelizing `Sync()`'s dependency
walk, a very plausible follow-up given `ModuleService`'s other concurrency-hardening work in
this same phase) would reintroduce exactly WR-04's failure mode on `localFiles` instead of
`cachedRegistry`, and `-race` would not catch it until that interleaving is actually
exercised.

**Fix:** Either extend `mu`'s documented scope to genuinely cover `localFiles` (take `d.mu.Lock()`
around the writes in `Parse`, matching what `ParseAll` already does for its own call to `Parse`
via `Import`), or drop the `RLock()`/`RUnlock()` in `LocalFileCount()` and add a comment
stating plainly that it is *not* safe to call concurrently with `Import`/`Parse` on the same
registry — so the safety claim in the code matches the safety claim callers can rely on.

## Info

### IN-01 (new): `mod/command.go`'s `modTidyCommand.Run` still performs the redundant `LoadFromLockFile` call the G-11-3 fix removed from `modInitCommand.Run`

**File:** `mod/command.go:149-155` (`modTidyCommand.Run`), compare with `mod/command.go:65-78` (`modInitCommand.Run`)

**Issue:** `243ab6f` removed `modInitCommand.Run`'s own `c.ms.LoadFromLockFile()` call with the
comment *"Init loads the lock file itself now ... so this no longer needs its own unchecked
LoadFromLockFile call"* — but `modTidyCommand.Run`, which calls the identical `Init(ctx,
"CONFIGSPACE")` sequence, still calls `LoadFromLockFile()` explicitly first:

```go
func (c *modTidyCommand) Run(args []string) int {
	c.flag.Parse(args)
	err := c.ms.LoadFromLockFile()   // redundant: Init (below) now does this itself
	if err != nil {
		c.ui.Error(err.Error())
		return 1
	}
	err = c.ms.Init(context.Background(), "CONFIGSPACE")
	...
```

`Init` (`compiler/lib/module_service.go:113-120`) now calls `m.LoadFromLockFile()` itself as
its first step, so `mod tidy` parses `protoconf.lock` twice on every invocation. This is
harmless today — the second read is deterministic given the first succeeded and nothing wrote
to the file in between — but it's dead weight left behind by an incomplete refactor, and there
is no test (`mod/command_test.go` only exercises `modInitCommand`/`modSyncCommand`) that would
catch `modTidyCommand` diverging further from `modInitCommand`'s now-corrected pattern.

**Fix:** Drop the redundant `c.ms.LoadFromLockFile()` call from `modTidyCommand.Run`, mirroring
the `modInitCommand.Run` cleanup; `Init`'s own load-and-return-error-first behavior already
covers it.

---

### IN-02 (carried forward, still open): `GetProtoRegistry`'s lazy-path comment still doesn't mention the remote-dependency `.fds` load

**File:** `compiler/lib/module_service.go:408-464`

**Issue:** Unchanged since the last review. The `m.lazyRegistry` branch's comment describes
"skips the whole-src/ eager parse+link entirely," but the unconditional `m.Walk(...)` loop just
above it still calls `registry.Load(...)` for every remote dependency's cached `.fds` — only
the local `src/` walk is skipped on the lazy path. Not a bug, just a comment that could mislead
a reader into thinking the lazy registry starts fully empty.

**Fix:** No action required; carried forward as documentation debt only.

---

### IN-03 (carried forward, still open): Duplicate whole-`src/` parse on mutation-server startup

**File:** `server/server.go:325-350`

**Issue:** Unchanged since the last review. `Init()`'s `discoveryRegistry` performs its own
full `Import`/`Parse` of `src/` (CONS-01 fix), while `NewProtoconfMutationServer` already built
an eager `ms.GetProtoRegistry()` covering the same tree moments earlier — `s.parser` is never
lazy in the current architecture (`NewProtoconfMutationServer` always calls
`lib.NewModuleService`, never `NewLazyModuleService`), so this defends against a scenario that
cannot yet occur, at the cost of parsing `src/` twice on every server start. Reasonable forward
defense for when `s.parser` does become lazy; currently pure duplicate work.

**Fix:** No action required now; revisit once `server.go` shares a lazy registry with
`s.compiler`.

---

_Reviewed: 2026-09-07T21:10:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
