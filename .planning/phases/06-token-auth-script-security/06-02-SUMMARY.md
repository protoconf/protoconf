---
phase: 06-token-auth-script-security
plan: 02
subsystem: server
tags: [security, scripts, env-vars, bug-fix, validation]
dependency_graph:
  requires: ["06-01"]
  provides: ["SECR-05", "SECR-07"]
  affects: ["server/server.go", "server/server_test.go"]
tech_stack:
  added: []
  patterns: ["defense-in-depth file existence check", "startup validation"]
key_files:
  created: []
  modified:
    - server/server.go
    - server/server_test.go
decisions:
  - "validateScriptPath rejects bare command names (no '/' path sep) implicitly via existence check, enforcing production convention of absolute paths"
  - "Defense-in-depth os.Stat in runScript handles TOCTOU between startup validation and execution"
  - "Updated test cases use real temp executables instead of bare PATH commands to be compatible with the stat pre-check"
metrics:
  duration_seconds: 183
  completed_date: "2026-03-28"
  tasks_completed: 2
  files_modified: 2
requirements:
  - SECR-05
  - SECR-07
---

# Phase 06 Plan 02: Script Security & Credential Forwarding Summary

**One-liner:** Script path validation at startup blocks traversal/non-executable paths; auth token and metadata env vars forwarded to pre/post scripts via runScript; PROTOCONF_COMPILER_ADDR= bug fixed.

## Tasks Completed

| # | Name | Commit | Files |
|---|------|--------|-------|
| 1 (RED) | Add failing tests for validateScriptPath | 0c64009 | server/server_test.go |
| 1 (GREEN) | Add validateScriptPath and wire into server startup | 546db3a | server/server.go |
| 2 | Add credential forwarding to runScript and fix env var bug | 12bcd1f | server/server.go, server/server_test.go |

## What Was Built

### validateScriptPath function

New function in `server/server.go` that validates a script path at server startup:
- Returns nil for empty paths (no script configured — backward compatible)
- Rejects paths containing `..` (path traversal prevention)
- Rejects non-existent paths (catches misconfiguration at startup)
- Rejects directory paths
- Rejects non-executable files (mode & 0111 == 0)

Wired into `run()` method: both `--pre` and `--post` are validated before the gRPC server starts.

### runScript defense-in-depth

Added `os.Stat(filename)` check before `exec.Command` in `runScript`. Guards against TOCTOU race where a script file is deleted after startup validation.

### Credential forwarding and bug fix

Updated `runScript` signature from `(filename, uuid string)` to `(filename, uuid, authToken, scriptMetadata string)`.

Updated `cmd.Env` in `runScript`:
- `PROTOCONF_COMPILER_ADDR=` — fixed bug: was missing `=` separator, was string concatenation not env var assignment
- `PROTOCONF_AUTH_TOKEN=` — new: forwards bearer token for downstream auth
- `PROTOCONF_SCRIPT_METADATA=` — new: forwards request script metadata

Both `runScript` call sites in `MutateConfig` pass `s.config.authToken` and `in.GetScriptMetadata()`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Updated existing test cases to use real temp executables**

- **Found during:** Task 2 execution — running all server tests
- **Issue:** Existing `Test_server_MutateConfig` subtests used bare command names (`"true"`, `"false"`) as script paths. The new defense-in-depth `os.Stat` check in `runScript` calls `stat "true"` which fails with "no such file" since `true` is a shell builtin/PATH command, not an absolute path.
- **Fix:** Rewrote `Test_server_MutateConfig` from table-driven to subtests. Added `makeTempScript` helper to create real temp executable scripts (using `#!/bin/sh\nexit 0` or `exit 1`). Also added the `"scripts receive auth credentials"` test case per the plan.
- **Files modified:** server/server_test.go
- **Commit:** 12bcd1f

## Known Stubs

None — all functionality fully wired with real tests.

## Self-Check: PASSED

- server/server.go contains `func validateScriptPath(path string) error` — confirmed
- server/server.go contains `strings.Contains(path, "..")` — confirmed
- server/server.go contains `info.Mode()&0111 == 0` — confirmed
- server/server.go contains `validateScriptPath(config.preMutationScript)` in run() — confirmed
- server/server.go contains `validateScriptPath(config.postMutationScript)` in run() — confirmed
- server/server.go contains `os.Stat(filename)` before exec.Command in runScript — confirmed
- server/server.go contains `func (s *ProtoconfMutationServer) runScript(filename, uuid, authToken, scriptMetadata string) error` — confirmed
- server/server.go contains `"PROTOCONF_COMPILER_ADDR="+s.config.grpcAddress` — confirmed
- server/server.go does NOT contain `"PROTOCONF_COMPILER_ADDR"+s.config.grpcAddress` — confirmed (bug fixed)
- server/server.go contains `"PROTOCONF_AUTH_TOKEN="+authToken` — confirmed
- server/server.go contains `"PROTOCONF_SCRIPT_METADATA="+scriptMetadata` — confirmed
- server/server_test.go contains `func Test_validateScriptPath` — confirmed
- `go test ./server/ -v` exits 0 — PASSED (all tests pass)
- `go build ./...` exits 0 — PASSED
