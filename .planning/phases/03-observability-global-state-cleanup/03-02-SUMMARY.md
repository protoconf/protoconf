---
phase: 03
plan: 02
subsystem: compiler/lib, mutate
tags: [global-state, sync.Once, race-condition, grpc, refactor]
dependency_graph:
  requires: []
  provides: [REFC-07, REFC-08]
  affects: [compiler/lib/compiler.go, mutate/mutate.go]
tech_stack:
  added: [sync]
  patterns: [sync.Once guard for package-level initialization]
key_files:
  created: []
  modified:
    - compiler/lib/compiler.go
    - mutate/mutate.go
decisions:
  - "sync.Once guards all six resolve.Allow* assignments so concurrent NewCompiler calls are race-free"
  - "grpc.ClientConn localized to Run() — no package-level mutable connection state needed"
metrics:
  duration: "~5 minutes"
  completed_date: "2026-03-27T06:36:25Z"
  tasks: 2
  files: 2
---

# Phase 3 Plan 2: Global State Cleanup Summary

**One-liner:** sync.Once guard for Starlark resolver globals and grpc.ClientConn localized to Run() in the mutate package.

## What Was Done

### Task 1: Move Starlark resolve settings to sync.Once guard (compiler/lib/compiler.go)

Added `var initResolveOnce sync.Once` and `initResolveSettings()` function that wraps all six `resolve.Allow*` assignments inside an `initResolveOnce.Do(func() {...})` callback. The `NewCompiler` function now calls `initResolveSettings()` instead of directly setting all six globals on every invocation.

This eliminates a data race: when multiple goroutines call `NewCompiler` concurrently, the resolve globals (which are package-level vars in go.starlark.net/resolve) could be written from multiple goroutines simultaneously. The `sync.Once` ensures the assignments happen exactly once.

### Task 2: Localize grpc.ClientConn to Run() in mutate package (mutate/mutate.go)

Removed `var conn *grpc.ClientConn` from package scope. Changed `conn, err =` to `conn, err :=` inside `Run()` so that `conn` is declared as a local variable. The `defer conn.Close()` line remains and now closes the locally-scoped variable.

The global `var ui = &cli.BasicUi{...}` was intentionally left in place per D-10.

## Commits

| Task | Commit | Description |
|------|--------|-------------|
| 1 | f6b569d | feat(03-02): guard Starlark resolve settings with sync.Once |
| 2 | 453b285 | fix(03-02): localize grpc.ClientConn to Run() in mutate package |

## Verification

- `go build ./compiler/lib/... ./mutate/...` — passes
- `go vet ./mutate/...` — passes
- `grep -n "var conn" mutate/mutate.go` — returns nothing (no package-level conn)
- `grep -n "initResolveOnce" compiler/lib/compiler.go` — returns declaration and Do usage
- Note: `TestCompiler_CompileFile/load_remote_with_load_local.pconf` fails in the full test suite — confirmed pre-existing (same failure on original code, unrelated to these changes — it's a test fixture file path issue in the test environment)

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None.

## Self-Check: PASSED

- compiler/lib/compiler.go modified and committed at f6b569d
- mutate/mutate.go modified and committed at 453b285
- Both files verified with grep checks
- Build passes for both packages
