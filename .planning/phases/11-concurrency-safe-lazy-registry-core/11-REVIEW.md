---
phase: 11-concurrency-safe-lazy-registry-core
reviewed: 2026-09-04T10:29:25Z
depth: standard
files_reviewed: 13
files_reviewed_list:
  - compiler/lib/compiler.go
  - compiler/lib/concurrent_compile_test.go
  - compiler/lib/config.go
  - compiler/lib/lazy_load_count_test.go
  - compiler/lib/mod_sync_fds_test.go
  - compiler/lib/module_service.go
  - compiler/lib/parser/lazy_parse_test.go
  - compiler/lib/parser/parser.go
  - compiler/lib/starlark_loader.go
  - compiler/lib/startup_bench_test.go
  - server/server.go
  - server/server_test.go
  - utils/utils.go
findings:
  critical: 0
  warning: 5
  info: 2
  total: 7
status: issues_found
---

# Phase 11: Code Review Report

**Reviewed:** 2026-09-04T10:29:25Z
**Depth:** standard
**Files Reviewed:** 13
**Status:** issues_found

## Summary

This phase makes `DescriptorRegistry` lazy: `GetProtoRegistry()` skips the whole-`src/` eager parse on the compiler-only path, `ParseOne` parses/links/memoizes a single file on demand (singleflight-collapsed, `d.mu`-guarded), and a one-shot `ParseAll` whole-tree fallback exists for type-URL misses (D-03). The two previously-identified consumer bugs (`server.go` `Init()`'s discovery scan, and `config.messageRegistry`'s by-value-copy race) are both correctly fixed and I could not find a third instance of either pattern; `go vet`'s copylocks result is confirmed clean (`DescriptorRegistry`/`ModuleService` are never copied by value anywhere in the diffed files).

No BLOCKER-level defects were found in the reachable production call paths, but the review surfaced several genuine, provable concurrency-correctness gaps that the current tests do not catch, plus a mismatch between what the new `TestConcurrentCompile` claims to exercise and what it actually exercises given the corpus generator's schema. These are all traced to specific code paths below, not speculative.

## Warnings

### WR-01: `ParseAll` violates `ParseOne`'s own documented lock discipline, serializing the whole registry for the fallback's full duration

**File:** `utils/utils.go:323-344` (compare with the discipline documented at `utils/utils.go:257-261`)

**Issue:** `ParseOne`'s own comment states the rule plainly: *"Lock discipline: never hold d.mu while calling parser.ParseFiles... holding the write lock across the call would self-deadlock [note: would serialize everything]. Read, unlock, parse, then take the write lock only for the inserts."* `ParseAll` does exactly what that rule warns against — it takes `d.mu.Lock()` at the top and holds it, via `defer`, across the entire `d.Import(d.Parse, ...)` call, i.e. across a full recursive walk-and-parse of every `.proto` under `ImportPaths`:

```go
func (d *DescriptorRegistry) ParseAll() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	...
	err := d.Import(d.Parse, []*regexp.Regexp{}, d.ImportPaths...)
	...
}
```

While this doesn't deadlock (verified: `Import`'s `LookupImport`/`Accessor` closures never re-enter `d.mu`), every other concurrent caller — `ParseOne`, `FileDescriptor`, `GetFileDescriptorSet`, `LoadedFileCount`, `FellBackToEager` — blocks on `d.mu.RLock()`/`d.mu.Lock()` for the full duration of the whole-tree parse. Given `ParseAll` fires from inside `RegistryTypeResolver.FindMessageByURL`/`FindMessageByName` (parser.go:77-84, 99-103), which run mid-compile from a normal `Msg.validate()`/`protojson.Marshal` call, one goroutine hitting a type-URL miss stalls every other concurrently-compiling file on the same shared `*lib.Compiler` — exactly the "lock held across a parse" failure mode this phase's own executor was asked to rule out.

**Fix:** Mirror `ParseOne`'s pattern — release `d.mu` before calling `d.Import`/`d.Parse`, then take it again only to merge results and set `d.eagerFallback = true`. This requires either accepting a second wasted whole-tree parse if two callers race into the un-lock window (rare, and still strictly bounded/idempotent since `Parse` overwrites with equivalent descriptors), or gating the parse itself with a `sync.Once`/second singleflight key instead of holding `d.mu` across it.

---

### WR-02: `ParseOne` can hand different callers non-identical `*desc.FileDescriptor` pointers for the same file when `ParseAll` races it for the same path

**File:** `utils/utils.go:228-297` (return path at line 291-296), interacting with `ParseAll` at `utils/utils.go:323-344`

**Issue:** `ParseOne`'s singleflight closure parses the file *outside* `d.mu` (by design, to avoid WR-01's self-serialization), then takes `d.mu.Lock()` only to call `recordFileLocked`, and finally returns its own `fds[0]` regardless of what `recordFileLocked` did:

```go
d.mu.Lock()
d.recordFileLocked(fds[0])
d.mu.Unlock()
return fds[0], nil   // returned to v, and then to every ParseOne caller for this key
```

`recordFileLocked` silently no-ops if the name is already present (`if _, ok := d.FileRegistry[fd.GetName()]; ok { return }`) — which is exactly what happens if `ParseAll` (holding `d.mu` for its entire run, per WR-01) parsed and inserted the same file first while this `ParseOne` call's own parse was in flight. In that interleaving:
1. Goroutine A calls `ParseOne("pkg7/msg7.proto")`; the target isn't yet in `FileRegistry`, so it starts an external `parser.ParseFiles` call (unlocked).
2. Concurrently, goroutine B's `RegistryTypeResolver.FindMessageByURL` misses twice and calls `ParseAll()`, which acquires `d.mu.Lock()`, walks the whole tree (including `pkg7/msg7.proto`, since it's not yet registered), parses its own independent `*desc.FileDescriptor` for it, inserts it, and releases the lock.
3. Goroutine A's closure now acquires `d.mu.Lock()`, calls `recordFileLocked(fds[0])`, which sees the name already present (B's) and returns without touching `FileRegistry`.
4. Goroutine A's `ParseOne` still returns **its own** `fds[0]` — a distinct, structurally-identical but non-`==` `*desc.FileDescriptor` from the one now canonical in `FileRegistry`.

A later caller of `ParseOne("pkg7/msg7.proto")` or `FileDescriptor("pkg7/msg7.proto")` gets B's pointer; goroutine A already has and may propagate A's pointer (e.g. into a `dynamic.Message`/`dynamicpb.MessageType` built from it). This breaks the pointer-identity contract `ParseOne`'s own doc comment promises ("a second request for the same path... is a map lookup [returning the same object]") and that `TestParseMemoization`'s `require.Same` checks — but that test never triggers `ParseAll`, so it can't catch this interleaving.

**Fix:** After `recordFileLocked` (still holding `d.mu`), look the path back up in `FileRegistry` and return *that* value instead of the locally-parsed `fds[0]`, so every caller — winner or loser of the race — observes the one canonical instance:
```go
d.mu.Lock()
d.recordFileLocked(fds[0])
canonical := d.FileRegistry[path]
d.mu.Unlock()
return canonical, nil
```

---

### WR-03: `RegistryTypeResolver`'s D-03 trigger swallows `ParseAll`'s error entirely, silently

**File:** `compiler/lib/parser/parser.go:74-84`, `95-103`

**Issue:** Both `FindMessageByURL` and `FindMessageByName` do `_ = r.registry.ParseAll()` — the return value, which is the only place a real underlying parse failure (a broken `.proto` file elsewhere in the tree) would surface, is discarded outright, with no log line at the point of failure:

```go
// Trigger the D-03 fallback and retry once regardless of ParseAll's own
// error: a partial whole-tree parse may still have registered the
// requested type before hitting an unrelated broken file elsewhere.
_ = r.registry.ParseAll()
if md, mErr := r.registry.MessageRegistry.FindMessageTypeByUrl(url); mErr == nil && md != nil {
	return dynamicpb.NewMessageType(md.UnwrapMessage()), nil
}
return nil, fmt.Errorf("%w: %s", protoregistry.NotFound, url)
```

Retrying despite `ParseAll`'s error is a reasonable design choice (the comment's reasoning holds), but discarding the error with no `slog` call means an operator debugging "why can't protoconf find this type" sees only `protoregistry.NotFound` and has no way to learn that the real cause is a broken proto file elsewhere in the tree that aborted the fallback parse. `CompileFileAsync`'s end-of-compile log (`compiler.go:220-221`) only reports the boolean `FellBackToEager()`, never the error text.

**Fix:** At minimum, `slog.Warn`/`slog.Error` the discarded error before the retry:
```go
if err := r.registry.ParseAll(); err != nil {
	slog.Warn("D-03 eager fallback parse failed", "url", url, "error", err)
}
```

---

### WR-04: `ModuleService.GetProtoRegistry()`'s cache is a check-then-act on an unsynchronized field, relying on an unenforced construction-order invariant

**File:** `compiler/lib/module_service.go:367-400`

**Issue:** `cachedRegistry` is read and written with no lock at all, even though `ModuleService` already has a `mutex sync.RWMutex` field (used elsewhere, e.g. `Init()` at line 145-147, but never for this field):

```go
func (m *ModuleService) GetProtoRegistry() *utils.DescriptorRegistry {
	if m.cachedRegistry != nil {
		return m.cachedRegistry
	}
	registry := utils.NewDescriptorRegistry()
	...
	m.cachedRegistry = registry
	return registry
}
```

In the *current* call graph this is not a live data race: `NewCompiler` (compiler.go:57-65) calls `GetProtoRegistry()` synchronously once, before the constructed `*Compiler` is ever handed to the `errgroup.Go`-per-file fan-out that `compiler/service.go`/`command.go` use in production, so every later concurrent caller only ever reads an already-populated pointer, and Go's happens-before rule for goroutine creation makes that safe. But that safety depends entirely on every caller going through this exact priming sequence — it is not enforced, documented at the call site, or guarded by the mutex that already exists on the struct for this exact purpose. Any future direct construction (e.g., a test, or a new consumer that calls `NewModuleService` + `GetProtoRegistry()` concurrently from multiple goroutines without an equivalent warm-up) reintroduces a genuine unsynchronized read/write race on `cachedRegistry` silently — `go test -race` will not catch it until that exact interleaving is hit, matching this phase's own note that a green `-race` run isn't proof of absence.

**Fix:** Guard the check-then-act with the existing `m.mutex` (double-checked locking, or an idiomatic `sync.Once`):
```go
func (m *ModuleService) GetProtoRegistry() *utils.DescriptorRegistry {
	m.mutex.RLock()
	if m.cachedRegistry != nil {
		defer m.mutex.RUnlock()
		return m.cachedRegistry
	}
	m.mutex.RUnlock()
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.cachedRegistry != nil {
		return m.cachedRegistry
	}
	... // build registry
	m.cachedRegistry = registry
	return registry
}
```

---

### WR-05: `TestConcurrentCompile`'s doc comment overstates what it exercises — the specific bug it names is never triggered by the generated corpus

**File:** `compiler/lib/concurrent_compile_test.go:14-24`

**Issue:** The test's comment says it proves the fix for "config.messageRegistry copied `msgregistry.MessageRegistry` by value in `Compiler.load`... an unsynchronized concurrent map access," and that "both the ParseOne singleflight path and the MessageRegistry AddFile path are exercised concurrently." That's true for the `AddFile` (write) side. But `config.messageRegistry` (compiler/lib/config.go:26, 68) is read *only* from `config.validate`'s `case *anypb.Any:` branch — and `utils/testdata/corpus.go`'s generator never emits a `google.protobuf.Any` field (confirmed: no `Any` reference anywhere in the generator, and `pickDeps`/`protoFile` only wire up plain message-typed fields, which take the `case *dynamic.Message:` branch and never touch `messageRegistry`). So the specific reader-through-a-stale-copy-races-writer-through-the-original interleaving that this test's own comment says it protects against cannot occur here regardless of whether `messageRegistry` is a pointer or a value copy — the read side of that race is structurally unreachable in this corpus. This matches the phase's own disclosed caveat that this test did not reproduce the live race before the pointer fix; concretely, it's because the test never drives the code path where the bug lived.

**Fix:** Either add an `Any`-typed field to the corpus generator (or a small standalone fixture) so `config.validate`'s Any branch — and thus a genuine concurrent read of `config.messageRegistry` — is exercised, or narrow the test's doc comment to stop claiming it protects the by-value-copy regression specifically, since today it only protects the `DescriptorRegistry.MessageRegistry.AddFile` write path.

## Info

### IN-01: `GetProtoRegistry`'s lazy path still eagerly loads every remote dependency's cached `.fds` set

**File:** `compiler/lib/module_service.go:367-400`

**Issue:** The `lazyRegistry` branch's comment describes the result as "a near-empty registry configured for on-demand parsing," but the `m.Walk(...)` loop immediately above it (unconditional for both eager and lazy paths) still calls `registry.Load(...)` for every remote-repo dependency's cached `.fds` file. Only the local `src/` tree walk is actually skipped. Not a bug — remote-dep handling appears intentionally out of this phase's scope — but the doc comment could mislead a future reader into assuming nothing is loaded eagerly on the lazy path.

**Fix:** Tighten the comment to say "skips the local `src/` eager parse" rather than implying the whole registry starts empty.

### IN-02: Duplicate whole-`src/` parse on every mutation-server startup after this change

**File:** `server/server.go:325-350`

**Issue:** `Init()`'s new `discoveryRegistry` performs its own full `Import`/`Parse` of `src/` (T-11 CONS-01 fix), but `NewProtoconfMutationServer` (server.go:285-303) already built an eager `ms.GetProtoRegistry()` covering the same tree moments earlier, and `s.parser` is never lazy in the current architecture (`NewProtoconfMutationServer` always calls `lib.NewModuleService`, never `NewLazyModuleService`) — so today this defends against a scenario (`s.parser` being lazy) that cannot yet occur, at the cost of parsing `src/` twice on every server start. `TestInitRegistersCustomService` has to manually swap in a bare registry to even exercise the code this guards. This is reasonable forward defense for when `s.parser` does become lazy in a later phase, but is currently pure duplicate work with no live consumer.

**Fix:** No action required now; worth revisiting once `server.go` actually shares a lazy registry with `s.compiler`; leaving as documentation for that phase.

---

_Reviewed: 2026-09-04T10:29:25Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
