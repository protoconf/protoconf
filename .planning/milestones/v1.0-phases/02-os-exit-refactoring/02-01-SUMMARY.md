---
phase: 02-os-exit-refactoring
plan: 01
subsystem: compiler
tags: [go, error-propagation, os-exit, refactoring, library-design]

# Dependency graph
requires: []
provides:
  - "NewModuleService returns (*ModuleService, error) with filepath.Abs at construction"
  - "NewCompiler returns (*Compiler, error) propagating module service errors"
  - "NewCompilerService returns (*CompilerService, error)"
  - "NewProtoconfMutationServer returns (*ProtoconfMutationServer, error)"
  - "Zero os.Exit calls in compiler/lib/module_service.go"
  - "Zero os.Exit calls in compiler/lib/starlark_loader.go"
affects: [phase-09-unit-tests, phase-10-integration-tests, devserver, server, compiler]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Resolve-at-construction: filepath.Abs called in NewModuleService constructor, stored canonical path eliminates error in helper"
    - "Closure error capture: var walkErr error before RangeFiles callback; return false to stop iteration and check after"
    - "Error propagation chain: library constructor errors propagate to CLI entry points returning int"

key-files:
  created: []
  modified:
    - compiler/lib/module_service.go
    - compiler/lib/starlark_loader.go
    - compiler/lib/compiler.go
    - compiler/command.go
    - compiler/service.go
    - server/server.go
    - devserver/command.go
    - test/e2e_test.go
    - server/server_test.go
    - compiler/lib/compiler_test.go
    - compiler/lib/module_service_test.go

key-decisions:
  - "Resolve filepath.Abs at construction time in NewModuleService rather than per-call in getProtoconfPath - eliminates error propagation through all string-returning helpers"
  - "NewCompiler returns (*Compiler, error) rather than log+panic - callers are CLI commands that can properly handle error with return 1"
  - "NewProtoconfMutationServer returns error rather than ignoring it - library code must not silently fail"

patterns-established:
  - "Resolve-at-construction: immutable derived values resolved once in constructor, stored canonically"
  - "Closure error capture: use var walkErr with return false for RangeFiles-style callbacks that don't accept error returns"
  - "Constructor error propagation: all constructors that can fail return (T, error) and callers handle at CLI boundary"

requirements-completed: [REFC-01, REFC-02, REFC-04]

# Metrics
duration: 15min
completed: 2026-03-24
---

# Phase 02 Plan 01: Compiler Library os.Exit Removal Summary

**os.Exit-free compiler library with error-returning constructors: NewModuleService, NewCompiler, NewCompilerService, and NewProtoconfMutationServer now propagate errors to CLI entry points**

## Performance

- **Duration:** ~15 min
- **Started:** 2026-03-24T05:28:00Z
- **Completed:** 2026-03-24T05:41:00Z
- **Tasks:** 2
- **Files modified:** 11

## Accomplishments

- Removed all os.Exit calls from compiler/lib/module_service.go (1 exit) by resolving filepath.Abs at construction time
- Removed all os.Exit calls from compiler/lib/starlark_loader.go (3 exits) using closure error capture pattern
- Changed NewModuleService, NewCompiler, NewCompilerService, and NewProtoconfMutationServer to return errors
- Updated all 7+ caller files to handle errors appropriately (CLI returns int, tests use require.NoError)

## Task Commits

Each task was committed atomically:

1. **Task 1: Refactor module_service.go and starlark_loader.go** - `d2756aa` (feat)
2. **Task 2: Update NewCompiler and all callers** - `546ae60` (feat, committed by parallel 02-02 agent)

## Files Created/Modified

- `compiler/lib/module_service.go` - NewModuleService now returns (*ModuleService, error); getProtoconfPath() simplified to trivial return
- `compiler/lib/starlark_loader.go` - loadValidators() uses var walkErr error with return false pattern; removed unused os and slog imports
- `compiler/lib/compiler.go` - NewCompiler returns (*Compiler, error)
- `compiler/command.go` - runLocally handles NewCompiler error with slog.Error + return 1
- `compiler/service.go` - NewCompilerService returns (*CompilerService, error)
- `server/server.go` - NewProtoconfMutationServer returns (*ProtoconfMutationServer, error); run method handles error
- `devserver/command.go` - handles errors from NewCompilerService and NewProtoconfMutationServer
- `test/e2e_test.go` - uses require.NoError for NewModuleService and NewCompiler
- `server/server_test.go` - uses require.NoError for all constructor calls; added require import
- `compiler/lib/compiler_test.go` - updated for new NewModuleService and NewCompiler signatures
- `compiler/lib/module_service_test.go` - updated for new NewModuleService signature; added require import

## Decisions Made

- Used resolve-at-construction pattern for NewModuleService to avoid propagating errors through all string-returning helper methods (getCacheDir, getLockFile, etc.)
- Changed NewCompiler signature to return error rather than log+panic, enabling callers to handle initialization failures gracefully
- Kept `os` import in module_service.go since it's still used for os.MkdirAll, os.ReadFile, os.WriteFile, os.ErrNotExist

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed existing test files that needed updating for new signatures**
- **Found during:** Task 2 (updating callers)
- **Issue:** compiler/lib/compiler_test.go and compiler/lib/module_service_test.go also called the old signatures
- **Fix:** Updated test files to use error-returning constructors with require.NoError
- **Files modified:** compiler/lib/compiler_test.go, compiler/lib/module_service_test.go
- **Verification:** go test ./compiler/lib/... passes
- **Committed in:** 546ae60 (Task 2 commit, parallel agent)

---

**Total deviations:** 1 auto-fixed (1 bug — test files missed in plan's files_modified list)
**Impact on plan:** Required for compilation. No scope creep.

## Issues Encountered

- Parallel execution: the 02-02 agent committed Task 2 changes (546ae60) concurrently as part of its own work, so Task 2 changes appear in that commit. The 02-01 Task 1 commit (d2756aa) is separate and clean.

## Next Phase Readiness

- All compiler/lib os.Exit calls removed; error propagation chain established from library to CLI
- Phase 09 (Unit Test Coverage) can now test real error paths without os.Exit interrupting test harness
- No blockers for remaining Phase 02 work (REFC-03 in plan 02-02 for mutate/mutate.go)

---
*Phase: 02-os-exit-refactoring*
*Completed: 2026-03-24*
