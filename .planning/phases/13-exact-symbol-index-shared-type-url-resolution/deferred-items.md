# Deferred Items — Phase 13

Out-of-scope discoveries logged during execution, per the executor's scope
boundary rule (do not auto-fix pre-existing issues unrelated to the current
task's files).

## 13-01: pre-existing `go vet ./...` findings outside this plan's files

`go vet ./...` (the plan-level `<verification>` block) reports findings in
files this plan never touches:

- `test/e2e_test.go:327,411` — `context.WithTimeout` cancel func discarded
- `agent/agent_test.go:27` — `context.WithTimeoutCause` cancel func discarded
- `agent/legacy.go:97` — unreachable code

Confirmed pre-existing: none of these files are in this plan's `files`
list (`utils/utils.go`, `utils/symbol_scan.go`, `utils/symbol_scan_test.go`,
`compiler/lib/parser/parser.go`, `compiler/lib/parser/nested_any_test.go`),
and `git log -1 -- agent/legacy.go` shows the offending commit
(`ac77547`) predates this phase entirely. `go vet ./compiler/... ./utils/...`
— the scope this plan's own task-level acceptance criteria require — is
clean.

## 13-02: `go test -race ./...` hang in `github.com/protoconf/protoconf/agent`

A whole-repo `go test -race ./...` run (this plan's own `<verification>`
block) reports `FAIL github.com/protoconf/protoconf/agent 602.107s` with a
goroutine dump rooted in `github.com/stephenafamo/orchestra`'s
`Conductor.playWithLogger`/`conductPlayer` (`sync.WaitGroup.Wait` stuck for
9 minutes) — a hang in mutation-server process orchestration, not in any
proto-parsing or type-resolution path this plan touches.

Confirmed out of scope: `agent/*.go` is not in this plan's `files` list
(`utils/utils.go`, `utils/symbol_index.go`, `utils/symbol_index_test.go`,
`utils/symbol_index_cache_test.go`, `compiler/lib/module_service.go`,
`compiler/lib/parser/parser.go`, `compiler/lib/parser/nested_any_test.go`,
`compiler/lib/index_not_built_test.go`), and neither `utils/symbol_index.go`
nor the `resolveTiers` change is reachable from `agent/agent_test.go`'s
orchestra-based process-lifecycle tests. `go test -race
./compiler/... ./utils/...` — the scope this plan's own task-level
acceptance criteria require, matching 13-01's precedent for the
impractical-whole-repo-race-run finding — is fully green.

## 13-03: same `go test -race ./...` `agent` hang, still out of scope

The same `github.com/protoconf/protoconf/agent` `Conductor.playWithLogger`
hang 13-02 logged reproduces on a whole-repo `go test -race ./...` run of
this plan's own `<verification>` block. Confirmed still out of scope: this
plan's `files_modified` list (`utils/utils.go`,
`compiler/lib/parser/parser.go`, `compiler/lib/compiler.go`, plus the
renamed/re-pointed test files and `compiler/lib/parser/hard_error_test.go`)
touches none of `agent/*.go`, and the deleted `ParseAll`/`FellBackToEager`
symbols had no callers there either (confirmed by grep during Task 2). `go
test -race ./compiler/... ./utils/...` — the scope this plan's own
task-level acceptance criteria require — is fully green.
