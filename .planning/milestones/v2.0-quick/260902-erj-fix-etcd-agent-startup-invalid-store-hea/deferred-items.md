# Deferred Items — 260902-erj

Out-of-scope pre-existing issues discovered during execution (SCOPE BOUNDARY rule: only auto-fix issues directly caused by this task's changes).

## `go vet ./agent/` failures (pre-existing, unrelated to this task)

1. `agent/agent_test.go:27` — `context.WithTimeoutCause` cancel func discarded (context leak). Introduced in commit f7ba446 (2024-03-17).
2. `agent/legacy.go:97` — unreachable code. Introduced in commit fb46573 (2024-05-27).

Neither file is in this task's `files_modified` list. `go build ./...` and `go test ./agent/...` both pass; only whole-package `go vet` trips on these two pre-existing findings. Left unfixed per scope discipline.

## `Test_cliCommand_Run/run_consul_server` hangs in this dev sandbox (pre-existing, environmental)

`go test ./agent/...` (top-level `agent` package, unbounded) times out because
`Test_cliCommand_Run/run_consul_server` in `agent/command_test.go` (want: 1,
i.e. expects the agent to fail to start) actually succeeds at connecting: this
specific machine has a real local Consul agent listening on `127.0.0.1:8500`
(confirmed via `lsof -i :8500`, unrelated to this repo). `checkStoreAvailable`
reaches that real server and returns nil either way -- this is identical
before and after the fix, since the pre-fix code's probe key `"/"` also
normalizes to a real, reachable request against a live Consul. With the store
check passing, `agent.RunAgent` (called with `context.Background()`, no
timeout) starts serving and blocks forever inside the test, since nothing
ever cancels it.

This is a pre-existing test/environment collision, not caused by this task's
changes -- confirmed with `go test ./agent/... -count=1 -timeout 30s`, which
times out inside `run_consul_server`'s gRPC/OTel goroutines, while
`go test ./agent/... -count=1 -skip Test_cliCommand_Run` and
`go test ./agent/configmaps/... ./agent/dummykv/... ./agent/filekv/... ./agent/otelkv/...`
all pass cleanly, including the new `TestStoreAvailabilityProbe`. Not fixed
per scope discipline (`agent/command_test.go` is not in `files_modified`).
