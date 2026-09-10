# Deferred Items — 260901-vaj

Out-of-scope discoveries logged per deviation-rules SCOPE BOUNDARY. Not fixed
in this plan; pre-existing and unrelated to the five dependency bumps.

## go vet pre-existing findings (unchanged before/after the bump)

Confirmed identical on both baseline go.mod (go 1.22.4, pre-bump) and the
bumped go.mod (go 1.25.8) — byte-for-byte the same set of findings, so the
dependency bump introduces zero new vet regressions:

- `agent/filekv/filekv.go` (10 findings): `Store` methods pass `sync.Mutex` by value (Put, Get, Delete, Exists, WatchTree, NewLock, List, DeleteTree, AtomicPut, AtomicDelete)
- `compiler/lib/compiler.go:355`: literal copies lock value from `MessageRegistry` (contains `sync.RWMutex`)
- `test/e2e_test.go:319,403`: `context.WithTimeout` cancel func discarded (context leak)
- `agent/agent_test.go:27`: `context.WithTimeoutCause` cancel func discarded (context leak)
- `agent/legacy.go:97`: unreachable code

These make `go vet ./...` exit 1 regardless of the dependency bump. The
plan's Task 2 `<verify>` line includes a bare `go vet ./...` which will
therefore also exit 1 — this is a pre-existing condition, not something this
plan introduced or is scoped to fix (scope_authority: only files a compile
error or a *newly failing* test names are in scope).

## go test -race pre-existing failure: TestWatch_ContextCancellation

`agent/filekv` `TestWatch_ContextCancellation` fails under `go test -race`
with "race detected during execution of test", flagged inside
`go-git/v5`'s worktree/dotgit internals invoked transitively by
`utils/testdata.TestDir()` -> `PlainInit` -> `AddGlob` -> `Status` ->
`Worktree.copyFileToStorage`/`ObjectStorage.SetEncodedObject`.

Confirmed pre-existing: reproduces identically with go-git/v5 reverted to
the baseline v5.12.0 (pre-bump). Not introduced by the go-git v5.12.0 ->
v5.19.2 bump. Out of scope per SCOPE BOUNDARY; not fixed here.

## Flaky test: TestProtoconfKVAgentRollout_SubscribeForConfig

`agent/kv_agent_rollout_impl_test.go` fails intermittently with
"timeout waiting for update" at line 174.

Measured failure rate over 10 consecutive runs of the isolated test:

| Commit | Fails / 10 |
|---|---|
| 7fdffc4 (pre-bump) | 2 |
| post-bump | 5 |

Pre-existing — it fails at the pre-bump commit too, so the dependency bump did
not introduce it. The apparent rate increase is not statistically meaningful at
n=10 and is consistent with the bump perturbing goroutine/gRPC-dial scheduling.

Root cause is a test-design race, not a product bug: the subtest goroutines
(`go func(want *want)`) call `SubscribeForConfig` concurrently with the main
goroutine's `inserter.InsertConfig` loop, and nothing synchronises "all
subscribers registered" against "start inserting". A subscriber that registers
after its update has already been published waits out its 5s budget and times
out.

Fix (deferred, out of scope here): signal readiness from each subscriber
goroutine — a `sync.WaitGroup` or ready-channel awaited before the insert loop
begins — instead of relying on the `time.Sleep(2s)` between inserts.

Impact if left: CI on this branch will go red randomly at roughly a 1-in-5 to
1-in-2 rate for the `agent` package.
