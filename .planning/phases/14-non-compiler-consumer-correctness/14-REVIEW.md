---
phase: 14-non-compiler-consumer-correctness
reviewed: 2026-09-08T00:00:00Z
depth: standard
files_reviewed: 16
files_reviewed_list:
  - agent/filekv/filekv.go
  - agent/filekv/filekv_race_test.go
  - agent/filekv/filekv_test.go
  - agent/kv_agent_race_test.go
  - compiler/lib/module_service.go
  - compiler/lib/parser/loaded_file_count_test.go
  - devserver/command.go
  - inserter/inserter.go
  - inserter/lazy_resolution_test.go
  - mutate/mutate.go
  - mutate/mutate_test.go
  - server/gen_reflection_ui_test.go
  - server/legacy.go
  - server/mutate_config_race_test.go
  - server/server.go
  - server/server_test.go
findings:
  critical: 1
  warning: 3
  info: 2
  total: 6
status: issues_found
---

# Phase 14: Code Review Report

**Reviewed:** 2026-09-08T00:00:00Z
**Depth:** standard
**Files Reviewed:** 16
**Status:** issues_found

## Summary

Phase 14 moves the inserter, agent/filekv, the mutation server, and the mutate CLI onto `lib.NewLazyModuleService` and the shared tiered `parser.TypeResolver`, and adds a substantial race/concurrency test suite plus a real fix in `server/legacy.go` (Marshal/Unmarshal replacing a `proto.Merge` across two wire-compatible-but-distinct message types). The `NewLazyModuleService` cutover itself is applied consistently — every non-`mod-sync` consumer now goes through it, and every place a resolver needed to see lazily-parsed types was switched from `LocalResolver` to `TypeResolver`. `server/legacy.go`'s fix is sound: both directions (request in→next, response result→out) are marshaled/unmarshaled with error checks on every step, and no path silently drops an error. `server/server.go`'s `collectExamples`/`reflectionFailure` refactor correctly continues past per-file failures instead of aborting the whole reflection walk, and the fingerprint-based log-on-change guard is order-independent and reason-sensitive as documented, verified with dedicated unit tests.

The one blocking issue is in `agent/filekv/filekv.go`'s `Get`/`Watch` path-traversal guard, which the phase's own new test (`TestGetRejectsTraversalKey`) asserts protects against directory traversal but which a live reproduction shows does not: it does not reject a caller-supplied key that already contains an unresolved `..` segment before the first real path component, so a key like `../etc/passwd` or `../secret/leak` sails through the guard and reaches `os.Stat`/`ReadConfig` for a path outside `protoconfRoot`. This guard predates this phase's diff, but the phase both retained the vulnerable code unchanged and added a new "regression" test that gives false confidence it is fixed, because the test only asserts "an error occurred" rather than "no file outside protoconfRoot was read."

## Critical Issues

### CR-01: filekv path-traversal guard does not stop `../`-prefixed keys; new regression test is a false negative

**File:** `agent/filekv/filekv.go:96-100` (also `agent/filekv/filekv.go:138-140` for `Watch`)
**Issue:** `Get` and `Watch` both gate on:
```go
if key != filepath.ToSlash(filepath.Clean(key)) || key == "" {
    return nil, fmt.Errorf("invalid path to get, path=%s", key)
}
```
`filepath.Clean` only collapses lexical redundancy (`./`, `//`, trailing `/`, resolvable `a/../b`); it cannot remove a **leading** `..` because there is no preceding path segment to cancel it against. So for `key = "../etc/passwd"` or `key = "../secret/leak"`, `filepath.Clean(key) == key`, the guard's inequality is false, and the key passes straight through to:
```go
absPath := filepath.Join(s.protoconfRoot, key+consts.CompiledConfigExtension)
```
which resolves to a path *outside* `protoconfRoot`. Reproduced directly:
```go
// root = <tmp>/protoconfRoot, secret file at <tmp>/secret/leak.materialized_JSON
key := "../secret/leak"
key == filepath.ToSlash(filepath.Clean(key)) // true -- guard does NOT reject it
absPath := filepath.Join(root, key+".materialized_JSON")
// => <tmp>/secret/leak.materialized_JSON  (outside root)
os.ReadFile(absPath) // succeeds, returns "SECRET_DATA"
```
This is a genuine path-traversal vulnerability in a KV store `Get`/`Watch` implementation that is reachable from any gRPC client that can pick the `path` for `SubscribeForConfig` (agent's public API surface).

The phase's own new test, `TestGetRejectsTraversalKey` (`agent/filekv/filekv_test.go:306-320`), calls this "the existing traversal guard as a regression" but only asserts `require.Error(t, err)`. In the test's temp-dir fixture, `../etc/passwd.materialized_JSON` simply doesn't exist, so `os.Stat` returns `ErrNotExist` → `store.ErrKeyNotFound`, which is non-nil and passes the assertion — without the guard itself ever having fired. The test therefore cannot fail even if an attacker-reachable file did exist at the traversed location; it validates the wrong thing.

**Fix:** Reject any key containing a `..` path element explicitly (mirroring `validateScriptPath`'s `strings.Contains(path, "..")` in `server/server.go`), or resolve the joined path and verify it remains under `protoconfRoot` via `filepath.Rel`/prefix check:
```go
cleaned := filepath.ToSlash(filepath.Clean(key))
if key != cleaned || key == "" || strings.HasPrefix(cleaned, "../") || cleaned == ".." {
    return nil, fmt.Errorf("invalid path to get, path=%s", key)
}
```
and strengthen `TestGetRejectsTraversalKey` to place a real file outside `protoconfRoot` at the traversal target and assert it is *not* returned (or that the specific "invalid path" error is produced), not merely that some error occurred.

## Warnings

### WR-01: `filekv.readEvents` can panic on send-to-closed-channel under concurrent `Close`

**File:** `agent/filekv/filekv.go:252-281`
**Issue:** `readEvents` snapshots the channel slice for a path under `w.lock`, releases the lock, then sends on each channel outside the lock:
```go
w.lock.Lock()
channels := append([]chan struct{}(nil), w.watches[event.Name]...)
w.lock.Unlock()
for _, channel := range channels {
    channel <- struct{}{}
}
```
If `Store.Close()` (→ `closeWatchers`) runs concurrently between the unlock and the send, it closes the same channel that `readEvents` is about to write to, causing a "send on closed channel" panic. The code already carries a `ponytail:` comment acknowledging this as a known ceiling ("fine under normal operation... Close() is typically called once at shutdown"), but `Close()` is a public method with no documented single-caller contract, and `devserver`/tests call it from `t.Cleanup` while `readEvents` keeps running until the fsnotify watcher itself closes. This is a real crash path, not merely a style nit.
**Fix:** Follow the upgrade path already named in the comment: replace per-watch channel-close-as-signal with one shared `done` channel closed exactly once by `Close`, and have `Watch`'s select read from `done` instead of relying on the closed per-watch channel; or send under the same lock that governs `closeWatchers`.

### WR-02: `GenReflectionUI` spins up a new bufconn listener + `grpc.Server.Serve` goroutine on every call, permanently retained until `ctx` cancellation

**File:** `server/server.go:751-794` (unchanged by this phase's diff, but exercised much harder by the new 5s-ticker `_ = ...GenReflectionUI(...)` call sites in `server/server.go:199`, `devserver/command.go:100`, and the new `TestMutateConfigConcurrentClientsAreRaceFree`/`TestGenReflectionUIConcurrentCallsAreRaceFree` tests that call it in a tight loop / on every tick)
**Issue:** Each `GenReflectionUI` call does `bufconn.Listen`, spawns a goroutine calling `rpcServer.Serve(lis)`, and registers a `context.AfterFunc(ctx, func() { rpcServer.GracefulStop() })`. None of these are cleaned up between calls — they all live until the *outer* `ctx` (the process lifetime context) is cancelled. In production this call happens every 5 seconds for the life of the server, so every tick permanently adds one more listener, one more goroutine, and one more `AfterFunc` registration that is never released early. This is a resource/goroutine leak; flagged as a warning rather than blocker per the review's performance/out-of-scope carve-out, but it is now exercised far more aggressively by phase 14's new tests and ticker call sites, so its severity in a long-running mutation server grows accordingly.
**Fix:** Track and explicitly stop/close the previous cycle's `lis`/goroutine before starting a new one, or restructure `GenReflectionUI` to reuse one long-lived bufconn/gRPC listener across calls instead of creating a fresh one per invocation.

### WR-03: `Watch`'s traversal guard shares the same flaw as `Get`'s (see CR-01) but is harder to exploit for read, still enables watching outside-root files

**File:** `agent/filekv/filekv.go:138-140`
**Issue:** `Watch` uses the identical `key != filepath.ToSlash(filepath.Clean(key))` guard as `Get`. While `fsnotify.Add` requires the target file to already exist (limiting blind exploitation), a caller who knows or can guess an absolute or `../`-reachable materialized-JSON path outside `protoconfRoot` can still register a watch and stream its contents via the returned channel (which itself calls the vulnerable `Get`).
**Fix:** Same fix as CR-01, applied to both `Get` and `Watch` (ideally factored into one shared `validateKey` helper so the two call sites cannot drift).

## Info

### IN-01: `reflectionFailure.path` field is inconsistently absolute vs. relative depending on failure site

**File:** `server/server.go:301-305`, `667-687`
**Issue:** For a directory-level walk error (`err != nil` in the `WalkDir` callback), `failures` gets the raw (root-relative-or-absolute, whichever `WalkDir` handed in) `path`. For every other failure branch, `failures` gets `relPath` (computed via `filepath.Rel(root, path)`). The struct comment documents this ("mutable_config/-relative path, or the walk root on a directory-level error"), so it's intentional, but it means `reflectionFailureFingerprint`'s per-line format (`path|typeURL|err`) mixes two different path conventions depending on failure class, which could be surprising to a future maintainer diffing fingerprints across releases.
**Fix:** Consider normalizing both branches to root-relative paths (falling back to the raw path only if `filepath.Rel` itself fails) for a more uniform fingerprint format. Low priority — no functional impact today.

### IN-02: `compiler/lib/module_service.go` `DownloadDeps` retains dead-code error check

**File:** `compiler/lib/module_service.go:512-531` (pre-existing, not modified by this phase's diff, but in the reviewed file set)
**Issue:** `if errors.Is(err, os.ErrNotExist) || errors.Is(err, &os.PathError{})` — the second disjunct can never be true: `errors.Is` on a freshly-constructed `&os.PathError{}` target compares by identity/`Is()` method, and `*os.PathError` implements neither in a way that makes an unrelated `*os.PathError` instance match. The first disjunct (`os.ErrNotExist`) already does the real work, so this is harmless but dead.
**Fix:** Drop the redundant `errors.Is(err, &os.PathError{})` disjunct.

---

_Reviewed: 2026-09-08T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
