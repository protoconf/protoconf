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
