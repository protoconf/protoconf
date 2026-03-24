---
phase: 02-os-exit-refactoring
plan: 02
subsystem: mutate
tags: [os-exit-refactoring, cli, error-propagation]
requirements: [REFC-03, REFC-04]
dependency_graph:
  requires: [02-01]
  provides: [os-exit-free-mutate-cli]
  affects: [mutate, compiler, agent/filekv, inserter, mod, server, devserver]
tech_stack:
  added: []
  patterns: [return-1-from-run, error-return-helpers, propagate-constructor-errors]
key_files:
  created: []
  modified:
    - mutate/mutate.go
    - compiler/lib/compiler.go
    - compiler/command.go
    - compiler/command_test.go
    - compiler/service.go
    - agent/filekv/filekv.go
    - inserter/inserter.go
    - mod/command.go
    - server/server.go
    - server/server_test.go
    - devserver/command.go
    - test/e2e_test.go
    - compiler/lib/compiler_test.go
    - compiler/lib/module_service_test.go
key_decisions:
  - "Fix all NewModuleService/NewCompiler caller sites across the codebase as a Rule 3 deviation to unblock the build"
metrics:
  duration_seconds: 588
  completed_date: "2026-03-24"
  tasks_completed: 1
  files_modified: 14
---

# Phase 02 Plan 02: Remove os.Exit from mutate/mutate.go Summary

Replace all 10 os.Exit(1) calls in mutate/mutate.go Run() method with return 1, and convert setNumeric/setFloat helpers to return errors with 12 call-site error checks.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Replace all os.Exit calls in mutate/mutate.go | 546ae60 | mutate/mutate.go + 13 caller fixups |

## What Was Done

**Primary change (REFC-03, REFC-04):**

`mutate/mutate.go` had 10 `os.Exit(1)` calls:
- 8 inside the `Run(args []string) int` method — all replaced with `return 1`
- 2 in helper functions `setNumeric` and `setFloat` — both helpers converted to return `error`

The `setNumeric` and `setFloat` helpers had their signatures changed:
- Before: `func setNumeric(msg *dynamic.Message, key, val string, typer typerFunc)` (calls `os.Exit`)
- After: `func setNumeric(msg *dynamic.Message, key, val string, typer typerFunc) error` (returns error)

All 12 call sites in the `switch` statement were wrapped with `if err := ...; err != nil { return 1 }`.

Additionally, the call to `lib.NewModuleService(root)` in `mutate/mutate.go` was updated to handle the new `(*ModuleService, error)` return that plan 02-01 introduced.

**Deviation fixups (Rule 3 — blocking issues):**

Plan 02-01 (`d2756aa`) changed `NewModuleService` to return `(*ModuleService, error)` and the linter cascaded these signature changes to `NewCompiler` and `NewProtoconfMutationServer` and `NewCompilerService`. However, not all callers were updated, causing build failures. These were fixed as Rule 3 deviations:

| File | Fix |
|------|-----|
| `compiler/lib/compiler.go` | Handle `(*ModuleService, error)` from `NewModuleService` |
| `compiler/command.go` | Fix `err :=` shadowing after `NewCompiler` already used `err` |
| `compiler/command_test.go` | Handle `(*CompilerService, error)` from `NewCompilerService` |
| `agent/filekv/filekv.go` | Handle `(*ModuleService, error)` from `NewModuleService` |
| `inserter/inserter.go` | Handle `(*ModuleService, error)` from `NewModuleService` |
| `mod/command.go` | Handle `(*ModuleService, error)` from `NewModuleService` |
| `server/server.go` | Already fixed by linter; `NewProtoconfMutationServer` returns error |
| `server/server_test.go` | Handle `(*ProtoconfMutationServer, error)` return |
| `devserver/command.go` | Handle errors from `NewCompilerService` and `NewProtoconfMutationServer` |
| `test/e2e_test.go` | Handle updated constructor signatures |
| `compiler/lib/compiler_test.go` | Already correctly handled |
| `compiler/lib/module_service_test.go` | Already correctly handled |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Fix NewModuleService/NewCompiler callers across 10+ files**
- **Found during:** Task 1, immediately after writing mutate/mutate.go
- **Issue:** Plan 02-01 changed `NewModuleService` to return `(*ModuleService, error)` and the Go toolchain cascaded signature changes to `NewCompiler`, `NewCompilerService`, and `NewProtoconfMutationServer`. Multiple callers still used the single-return form, causing `go build ./...` to fail.
- **Fix:** Updated all callers to handle the error return, adding `if err != nil { ... return/log }` blocks throughout.
- **Files modified:** compiler/lib/compiler.go, compiler/command.go, compiler/command_test.go, agent/filekv/filekv.go, inserter/inserter.go, mod/command.go, server/server.go, server/server_test.go, devserver/command.go, test/e2e_test.go
- **Commit:** 546ae60

## Out-of-Scope Deferred Issues

- Pre-existing `go vet` warning in `compiler/lib/compiler.go:331` about copying a lock value from `MessageRegistry` — pre-dates plan 02-02, introduced in plan 02-01 commit `d2756aa`
- Pre-existing `go vet` warnings in `agent/filekv/filekv.go` about passing Store by value (contains sync.Mutex) — pre-existing before this phase

## Verification Results

```
grep -c 'os\.Exit' mutate/mutate.go  => 0
go build ./mutate/...                 => OK
go vet ./mutate/...                   => OK
func setNumeric signature             => error return type confirmed
func setFloat signature               => error return type confirmed
```

## Known Stubs

None.

## Self-Check: PASSED

- mutate/mutate.go exists and has 0 os.Exit calls: FOUND
- Commit 546ae60 exists: FOUND
- go build ./mutate/... passes: CONFIRMED
- go vet ./mutate/... passes: CONFIRMED
