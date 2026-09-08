## Deferred Items

- `go vet ./test/...` reports two pre-existing `lostcancel` findings in `test/e2e_test.go` (lines 327 and 411: `tCtx, _ := context.WithTimeout(...)` / `newCtx, _ := context.WithTimeout(...)` discard their cancel funcs).
  status: open
  **What:** Blame shows both lines date to commit `99e33a44` (2023-12-14), long before Phase 14. `git stash`-testing 14-03's `server/server.go` diff confirms `go vet ./server/...` alone is silent; the failure is isolated to `test/e2e_test.go` and unrelated to this plan's D-01/D-03/D-06 changes. `go test ./test/...` itself is unaffected (its built-in vet subset excludes `lostcancel`), so `TestAuthFlow` still runs and passes. Out of scope per the executor's scope-boundary rule (pre-existing, unrelated to current task); left unfixed here.
