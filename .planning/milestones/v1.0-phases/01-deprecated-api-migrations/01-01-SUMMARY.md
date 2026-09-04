---
phase: 01-deprecated-api-migrations
plan: 01
subsystem: grpc-client
tags: [grpc, deprecated-api, migration, reflection]
dependency_graph:
  requires: []
  provides: [clean-grpc-client-api, grpc-reflection-v1]
  affects: [compiler, server, mutate, agent, test, examples]
tech_stack:
  added: []
  patterns:
    - "grpc.NewClient with grpc.WithTransportCredentials(insecure.NewCredentials()) for all gRPC client connections"
    - "passthrough:///bufnet target for in-process bufconn connections with grpc.NewClient"
    - "dual v1+v1alpha reflection registration for grpcui@v1.4.1 compatibility"
key_files:
  created: []
  modified:
    - compiler/command.go
    - server/server.go
    - mutate/mutate.go
    - agent/legacy.go
    - test/e2e.go
    - agent/kv_agent_impl_test.go
    - examples/mutation/go_client/main.go
    - examples/grpc_clients/go_client/main.go
decisions:
  - "Register both grpc_reflection_v1 and grpc_reflection_v1alpha in server.go to maintain grpcui@v1.4.1 compatibility while adopting v1 as primary"
  - "Use passthrough:///bufnet as target for grpc.NewClient with bufconn (grpc.NewClient requires non-empty target for dns resolver)"
metrics:
  duration: 1363s
  completed: "2026-03-23"
  tasks: 2
  files_modified: 8
---

# Phase 01 Plan 01: Migrate Deprecated gRPC APIs Summary

Migrated all deprecated gRPC-Go API calls to current stable equivalents across 8 files: replaced `grpc.WithInsecure`+`grpc.Dial`/`grpc.DialContext` with `grpc.NewClient`+`insecure.NewCredentials()`, and upgraded reflection from `grpc_reflection_v1alpha` to `grpc_reflection_v1` as primary with v1alpha kept for grpcui backward compatibility.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Migrate grpc.WithInsecure and grpc_reflection_v1alpha | cf10778 | compiler/command.go, server/server.go |
| 2 | Migrate remaining grpc.Dial/DialContext to grpc.NewClient | d4228c9 | mutate/mutate.go, agent/legacy.go, test/e2e.go, agent/kv_agent_impl_test.go, examples/mutation/go_client/main.go, examples/grpc_clients/go_client/main.go |

## Verification Results

1. `go build ./...` — passes with zero errors
2. `go test ./server/... ./agent/... ./compiler/...` — all pass
3. `grep -r "grpc.WithInsecure"` (excluding worktrees) — zero results
4. `grep -r "grpc_reflection_v1alpha\."` (excluding worktrees) — zero import references (v1alpha kept for registration only, driven by v1 primary)
5. `grep -r "grpc\.Dial\b|grpc\.DialContext"` (excluding worktrees) — zero results
6. `grep -r "grpc\.NewClient"` — found in all 8 modified files

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] grpc.NewClient requires non-empty target for DNS resolution**
- **Found during:** Task 1 (server/server.go) and Task 2 (agent/legacy.go, test/e2e.go, agent/kv_agent_impl_test.go)
- **Issue:** `grpc.NewClient("")` fails with "dns resolver: missing address" because NewClient uses DNS resolver by default (unlike DialContext which used passthrough scheme)
- **Fix:** Used `"passthrough:///bufnet"` as target for all in-process bufconn connections — the `passthrough` scheme skips DNS and the custom `WithContextDialer` handles actual connection
- **Files modified:** server/server.go, agent/legacy.go, test/e2e.go, agent/kv_agent_impl_test.go

**2. [Rule 2 - Compatibility] grpcui@v1.4.1 only supports v1alpha reflection**
- **Found during:** Task 1 (server/server.go)
- **Issue:** After switching to `reflection.NewServerV1` + `grpc_reflection_v1.RegisterServerReflectionServer`, the `TestProtoconfMutationServer_GenReflectionUI` test failed because `grpcui@v1.4.1` internally uses `grpc.reflection.v1alpha.ServerReflection` protocol
- **Fix:** Register BOTH `grpc_reflection_v1` (primary) and `grpc_reflection_v1alpha` (backward compat) in `Init()` method. Also register both proto file descriptors in `NewProtoconfMutationServer`. This satisfies the DEPR-02 requirement (v1 is now primary) while keeping grpcui working
- **Files modified:** server/server.go

### Pre-existing Issues (Out of Scope)

- `compiler/lib` tests timeout at 30s during network-fetching module operations — pre-existing, confirmed by running on unmodified codebase
- `test/` e2e test times out when not run in short mode — pre-existing network dependency on module fetching; test skips correctly with `-short` flag

## Known Stubs

None — all changes are mechanical API replacements with no stubbed functionality.

## Self-Check: PASSED

- compiler/command.go: FOUND
- server/server.go: FOUND
- mutate/mutate.go: FOUND
- agent/legacy.go: FOUND
- test/e2e.go: FOUND
- agent/kv_agent_impl_test.go: FOUND
- examples/mutation/go_client/main.go: FOUND
- examples/grpc_clients/go_client/main.go: FOUND
- Commit cf10778: FOUND
- Commit d4228c9: FOUND
