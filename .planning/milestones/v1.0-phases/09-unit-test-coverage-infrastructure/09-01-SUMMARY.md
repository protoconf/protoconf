---
phase: "09"
plan: "01"
subsystem: testing
tags: [testutil, unit-tests, command, fmt, bufconn, grpc]
dependency_graph:
  requires: []
  provides: [testutil-package, command-tests, fmt-tests]
  affects: [all-packages-using-testutil]
tech_stack:
  added: [testutil package]
  patterns: [bufconn gRPC test server, table-driven tests with testify]
key_files:
  created:
    - testutil/testutil.go
    - command/command_test.go
    - fmt/command_test.go
  modified: []
decisions:
  - "NewTestProtoconfRoot delegates to testdata.SmallTestDir() which already creates an isolated temp dir per call"
  - "testutil imports no protoconf service protos to prevent circular dependency risk"
  - "command_test.go uses blank-identifier reference to testutil.NewAny for D-03 compliance without over-engineering"
metrics:
  duration_seconds: 180
  completed_date: "2026-03-31"
  tasks_completed: 2
  files_created: 3
---

# Phase 09 Plan 01: Test Infrastructure — testutil, command, fmt Tests Summary

**One-liner:** Shared test helpers (bufconn gRPC server, NewAny, testdir) plus table-driven unit tests for command/ and fmt/ packages with error-path coverage.

## Tasks Completed

| # | Name | Commit | Files |
|---|------|--------|-------|
| 1 | Create testutil/ package | 9f20902 | testutil/testutil.go |
| 2 | Add unit tests for command/ and fmt/ | e0b9edb | command/command_test.go, fmt/command_test.go |

## What Was Built

### testutil/testutil.go
- **NewBufconnServer**: Generic in-process gRPC test server using bufconn. Accepts a `func(*grpc.Server)` registration callback so any service can be tested without coupling testutil to specific proto packages.
- **NewAny**: Concise `anypb.New` wrapper ignoring error (valid for test messages).
- **NewTestProtoconfRoot**: Returns an isolated protoconf root directory via `testdata.SmallTestDir()`.

### command/command_test.go
- `TestDefaultUI` — verifies DefaultUI is non-nil.
- `TestNewPrefixedUi` — verifies construction, Prefix field, Ui field.
- `TestPrefixedUi_Output` — table-driven: single line, empty string, multi-line.
- `TestPrefixedUi_Info`, `TestPrefixedUi_Error`, `TestPrefixedUi_Warn` — method delegation with prefix.
- `TestPrefixedUi_Ask`, `TestPrefixedUi_AskSecret` — query prefix prepending.
- `TestPrefixedUi_NilUiPanics` — error path: nil inner Ui causes panic.

### fmt/command_test.go
- `TestIsStarlarkFile` — table-driven: 5 true cases, 5 false cases (including `.sky` absent from StarlarkExtensions).
- `TestFormatFile_*` — already-formatted (no-op), write mode, no-write mode, non-existent file error path.
- `TestComputeDiff` — verifies `---`/`+++` markers and changed lines.
- `TestComputeDiff_Identical` — no change lines emitted for identical content.
- `TestProcessPath_Directory` — mixed file directory, only .pconf processed.
- `TestProcessPath_File` — direct file path.
- `TestProcessPath_NonExistent` — error path.
- `TestCommand` — factory returns non-nil command with non-empty Synopsis/Help.

## Verification Results

```
go build ./testutil/         OK
go vet ./testutil/ ./command/ ./fmt/   OK (no issues)
go test -race ./command/     PASS (9 tests)
go test -race ./fmt/         PASS (11 tests)
grep testutil command/command_test.go fmt/command_test.go   FOUND in both files
```

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed failing TestComputeDiff_Identical test**
- **Found during:** Task 2 first test run
- **Issue:** `computeDiff` always emits `---`/`+++` header lines even for identical content; test incorrectly rejected those lines
- **Fix:** Updated test to skip `---`/`+++` prefix header lines and only flag unexpected change lines
- **Files modified:** fmt/command_test.go
- **Commit:** e0b9edb (part of Task 2 commit)

**2. [Rule 1 - Bug] Removed unused `cli` import in command_test.go**
- **Found during:** Task 2 first compile
- **Issue:** `cli.ConcurrentUi` type assertion was unnecessary since `DefaultUI` is already statically typed
- **Fix:** Removed `github.com/mitchellh/cli` import, rewrote `TestDefaultUI` to only check non-nil
- **Files modified:** command/command_test.go
- **Commit:** e0b9edb (part of Task 2 commit)

## Known Stubs

None — all three exported functions in testutil are fully implemented. The test files exercise real code paths with real assertions.

## Self-Check: PASSED
