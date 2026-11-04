---
phase: quick-261104-i9e
plan: 01
status: complete
commits: 2
plan_head_before: a5a7d208
actuals:
  tokens: 4000
  tasks: 1
  commits: 2
---

# Quick 261104-i9e: -starlark-profile flag for protoconf compile

`protoconf compile -starlark-profile <file>` (env `PROTOCONF_COMPILER_STARLARK_PROFILE`) writes a gzipped pprof Starlark profile via `starlark.StartProfile`/`StopProfile`.

## Commits
- 402028e4 test: failing `Test_cliCommand_StarlarkProfile` (RED, Run returned 2)
- 9f7c8f0e feat: proto field `starlark_profile = 8`, regenerated pb.go, wiring in `cliCommand.Run`

## Implementation
- Field number 8 (7 left unused) because unpushed local `main` (quick 260925-k4y) uses `log_level = 7`.
- Create/Start failures log and `return 1` (no os.Exit). StopProfile defer is registered only after StartProfile succeeds and runs before the file close (LIFO).
- Header of regenerated pb.go: single-file protoc, as planned.

## Verification
- Targeted test passes under -race; pinned `./compiler/...` suite green; `go vet` clean.
- Built binary: `compile -h` lists `-starlark-profile` with `PROTOCONF_COMPILER_STARLARK_PROFILE`; bad path logs "Could not create Starlark profile" and exits 1.

## Deviations from Plan
None in code. Note: `gofmt -l compiler/` lists two pre-existing unformatted files (`compiler/lib/compiler.go`, `compiler/lib/module_service.go`), untouched by this task; the files changed here are gofmt-clean.

## Follow-up
Field 7 was skipped on purpose. Merging with local `main` (log_level = 7) will give a textual conflict in compiler_config.proto and .pb.go; keep both fields and rerun the single-file protoc command.

## Self-Check: PASSED
