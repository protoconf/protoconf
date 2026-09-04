---
phase: 09-unit-test-coverage-infrastructure
plan: 02
subsystem: agent/kv-stores
tags: [testing, kv-store, dummykv, filekv, otelkv, configmaps, unit-tests]
dependency_graph:
  requires: []
  provides: [kv-store-test-coverage]
  affects: [agent/dummykv, agent/filekv, agent/otelkv, agent/configmaps]
tech_stack:
  added: [k8s.io/client-go/kubernetes/fake]
  patterns: [mock-store, fake-k8s-client, table-driven-tests, testify-assert]
key_files:
  created:
    - agent/dummykv/dummykv_test.go
    - agent/filekv/filekv_test.go
    - agent/otelkv/otelkv_test.go
    - agent/configmaps/configmaps_test.go
  modified:
    - agent/configmaps/configmaps.go
    - go.mod
    - go.sum
decisions:
  - "Use kubernetes.Interface instead of *kubernetes.Clientset to enable fake client injection"
  - "filekv tests focus on error paths and valid construction since Watch requires proto-parsed materialized JSON"
  - "dummykv Watch test uses non-deterministic delivery approach due to concurrent goroutine sends"
metrics:
  duration_seconds: 900
  completed_date: "2026-03-31"
  tasks_completed: 2
  files_created: 4
  files_modified: 3
---

# Phase 09 Plan 02: KV Store Unit Tests Summary

**One-liner:** Dedicated test files for dummykv, filekv, otelkv, and configmaps with fake Kubernetes client injection and OTel delegation verification.

## What Was Built

Added dedicated test files for all four KV store implementations in the `agent/` package. Each package had zero test files before this plan.

### dummykv (12 test functions)
- `TestNew`, `TestNew_NilOptions` — construction with nil/empty config
- `TestPutAndGet` — table-driven with simple, path, and empty value cases
- `TestGet_NotFound` — error path: key not found returns error
- `TestExists` — key existence check
- `TestDelete` — delete then Get returns error
- `TestWatch` — watch on non-existing key receives Put event
- `TestWatch_ExistingKey` — watch after Put delivers value(s); handles non-deterministic goroutine ordering
- `TestWatch_NewKey` — watch fires when key is first written
- `TestDeleteTree`, `TestPut_MultipleTimes`, `TestGet_WithWriteOptions`

### filekv (10 test functions)
- `TestNew` — construction with SmallTestDir (valid protoconf root)
- `TestNew_InvalidRoot` — handles nonexistent path gracefully
- `TestPut`, `TestPut_WithWriteOptions` — no-op Put returns nil
- `TestExists` — always returns true per implementation
- `TestClose` — Close returns nil
- `TestWatch_InvalidPath`, `TestWatch_InvalidPath_Dots` — error paths for invalid paths
- `TestWatch_NonExistentFile`, `TestWatch_ContextCancellation` — error paths via fsnotify

### otelkv (16 test functions)
- `TestNew` — tracer initialized, inner store assigned
- Per-method delegation tests: `TestPut_Delegation`, `TestGet_Delegation`, `TestDelete_Delegation`, `TestExists_Delegation`, `TestWatch_Delegation`, `TestWatchTree_Delegation`, `TestNewLock_Delegation`, `TestList_Delegation`, `TestDeleteTree_Delegation`, `TestAtomicPut_Delegation`, `TestAtomicDelete_Delegation`, `TestClose_Delegation`
- `TestPut_PropagatesError`, `TestGet_PropagatesError` — error path propagation
- `TestDelegation_AllMethods` — table-driven comprehensive delegation check

### configmaps (11 test functions)
- `TestNew_WithFakeClient` — fake clientset injection
- `TestPut_CreatesConfigMap` — verifies ConfigMap created in fake k8s store
- `TestPutAndGet` — round-trip Put+Get with fake client
- `TestGet_NotFound`, `TestGet_KeyNotInConfigMap` — error paths
- `TestExists` — always returns true per implementation
- `TestDelete` — delete removes key from ConfigMap data
- `TestDelete_NotFound` — delete of nonexistent configmap returns error
- `TestPut_MultipleKeys` — multiple keys in same ConfigMap directory
- `TestWatch_ReturnsChannel` — Watch returns channel and delivers initial value
- `TestPut_KeyNameMapping` — path separator to ConfigMap name mapping

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed nil pointer dereference in configmaps.Put**
- **Found during:** Task 2 test execution
- **Issue:** When `cmClient.Get()` returns an error (ConfigMap doesn't exist), `cm` is nil. The code immediately executed `cm.ObjectMeta.Name = configMapName` on the nil pointer, causing panic.
- **Fix:** Added `cm = &cv1.ConfigMap{}` before setting ObjectMeta.Name, creating a new ConfigMap object to populate.
- **Files modified:** `agent/configmaps/configmaps.go`
- **Commit:** dce95b1

**2. [Rule 2 - Testability] Changed clientset field to kubernetes.Interface**
- **Found during:** Task 2, when writing configmaps_test.go
- **Issue:** `Store.clientset` was typed as `*kubernetes.Clientset` (concrete type). The fake client `fake.NewSimpleClientset()` returns `*fake.Clientset` which implements `kubernetes.Interface` but is a different concrete type — cannot be injected.
- **Fix:** Changed `clientset *kubernetes.Clientset` to `clientset kubernetes.Interface` and updated `getClientset()` return type accordingly. Behavior identical at runtime.
- **Files modified:** `agent/configmaps/configmaps.go`
- **Commit:** dce95b1

**3. [Rule 3 - Missing Dependency] Added go.sum entry for k8s.io/client-go/testing**
- **Found during:** Task 2, first test run
- **Issue:** `fake.NewSimpleClientset()` transitively imports `github.com/evanphx/json-patch` which was missing from go.sum.
- **Fix:** Ran `go get k8s.io/client-go/testing@v0.30.1` to add the dependency.
- **Files modified:** `go.mod`, `go.sum`
- **Commit:** dce95b1

### Out-of-Scope Pre-existing Issues (Deferred)

`agent/filekv/filekv.go` has 10 `go vet` warnings about value receiver methods on a struct containing `sync.Mutex` (e.g., `Put passes lock by value`). These are pre-existing issues not caused by this plan's changes. Deferred for a future cleanup plan.

## Known Stubs

- `dummykv.Exists` always returns `true` regardless of whether the key exists — stub behavior intentional per D-11 plan
- `dummykv.List` still panics (not implemented) — test file avoids calling it
- `dummykv.Close` still panics (not implemented) — test file avoids calling it
- `configmaps.Exists` always returns `true` — stub behavior per current implementation
- `filekv.Get` panics (unimplemented) — test file avoids calling it

## Self-Check: PASSED

- agent/dummykv/dummykv_test.go: exists, 12 test functions, uses testify
- agent/filekv/filekv_test.go: exists, 10 test functions, uses testify
- agent/otelkv/otelkv_test.go: exists, 16 test functions, uses testify
- agent/configmaps/configmaps_test.go: exists, 11 test functions, uses fake k8s client
- All tests pass: `go test -v -count=1 -timeout=30s ./agent/dummykv/ ./agent/filekv/ ./agent/otelkv/ ./agent/configmaps/`
- Commits: eea8610 (Task 1), dce95b1 (Task 2)
