---
phase: 260902-erj
plan: 01
subsystem: infra
tags: [etcd, grpc, kv-store, valkeyrie, agent-startup]

requires: []
provides:
  - "checkStoreAvailable shared helper in agent/kv_agent_impl.go, wired into both agent constructors"
  - "Regression test proving both constructors probe a non-empty key regardless of config.Prefix"
affects: [agent, devserver]

actuals:
  tokens: 1314
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "Shared reachability probe helper (checkStoreAvailable) instead of duplicated inline checks in each constructor"

key-files:
  created:
    - agent/store_probe_test.go
  modified:
    - agent/kv_agent_impl.go
    - agent/kv_agent_rollout_impl.go

key-decisions:
  - "Kept the fail-fast reachability check rather than deleting it (as upstream PR #496 did) - an unreachable store must fail once at boot, not on every client subscription"
  - "Used path.Join(config.GetPrefix(), storeProbeKey) with a dot-prefixed sentinel key so the joined probe key is never empty after the backend's leading-slash normalization, for any prefix value"
  - "Used a local probeStore fake embedding store.Store in the regression test instead of extending agent/dummykv, since dummykv.Exists hardcodes true and dummykv.Get doesn't return store.ErrKeyNotFound"

requirements-completed: [QUICK-260902-erj]

coverage:
  - id: D1
    description: "Agent constructors probe a non-empty key after prefix normalization, so a reachable-but-empty etcd store no longer fails startup"
    requirement: "QUICK-260902-erj"
    verification:
      - kind: unit
        ref: "agent/store_probe_test.go#TestStoreAvailabilityProbe/NewProtoconfKVAgent/probe_key_is_non-empty_after_leading-slash_normalization"
        status: pass
      - kind: unit
        ref: "agent/store_probe_test.go#TestStoreAvailabilityProbe/NewProtoconfKVAgentRollout/probe_key_is_non-empty_after_leading-slash_normalization"
        status: pass
    human_judgment: false
  - id: D2
    description: "Fail-fast behavior preserved: a transport error from the store still aborts agent construction with 'store is not available'"
    requirement: "QUICK-260902-erj"
    verification:
      - kind: unit
        ref: "agent/store_probe_test.go#TestStoreAvailabilityProbe/NewProtoconfKVAgent/transport_error_fails"
        status: pass
      - kind: unit
        ref: "agent/store_probe_test.go#TestStoreAvailabilityProbe/NewProtoconfKVAgentRollout/transport_error_fails"
        status: pass
    human_judgment: false
  - id: D3
    description: "A key that is merely absent (fresh, empty store) is treated as success, not a startup failure"
    requirement: "QUICK-260902-erj"
    verification:
      - kind: unit
        ref: "agent/store_probe_test.go#TestStoreAvailabilityProbe/NewProtoconfKVAgent/absent_key_succeeds"
        status: pass
      - kind: unit
        ref: "agent/store_probe_test.go#TestStoreAvailabilityProbe/NewProtoconfKVAgentRollout/absent_key_succeeds"
        status: pass
    human_judgment: false

duration: 25min
completed: 2026-09-02
status: complete
---

# Quick Task 260902-erj: Fix etcd agent startup invalid store health-check key

**Extracted a shared `checkStoreAvailable` helper that probes `path.Join(config.GetPrefix(), storeProbeKey)` instead of the bare `"/"`, fixing false startup failures against a healthy, empty etcd store while keeping the fail-fast check PR #496 deleted.**

## Performance

- **Duration:** ~25 min
- **Started:** 2026-09-02T10:41:00Z (plan authored) / execution ~10:44-11:09
- **Completed:** 2026-09-02
- **Tasks:** 2
- **Files modified:** 3 (2 modified, 1 created)

## Accomplishments
- Root-caused and fixed the etcd startup failure: `Exists(ctx, "/", nil)` normalizes to the empty string under `kvtools/etcdv3`, which etcd rejects with a transport-level `ErrEmptyKey` even against a perfectly reachable store
- Extracted one shared `checkStoreAvailable(ctx, store, config)` helper used by both `NewProtoconfKVAgent` and `NewProtoconfKVAgentRollout`, eliminating the duplicated byte-identical probe blocks
- Added a table-driven regression test (`TestStoreAvailabilityProbe`) covering both constructors: probe-key-non-empty invariant, absent-key-succeeds, and transport-error-still-fails
- Preserved fail-fast behavior instead of deleting the check (the approach upstream PR #496 took, which trades a false startup failure for a real one)

## Task Commits

Each task was committed atomically (TDD RED/GREEN cycle):

1. **Task 1: Failing regression test for the store-availability probe (RED)** - `9aefb2d` (test)
2. **Task 2: Extract shared checkStoreAvailable helper and wire both constructors (GREEN)** - `db33bbd` (fix)

**Plan metadata:** commit deferred to orchestrator per constraints (this SUMMARY, STATE.md, PLAN.md not committed by executor)

## Files Created/Modified
- `agent/store_probe_test.go` - New regression test; local `probeStore` fake (embeds `store.Store`) records the probed key and returns canned exists/err values; table over both constructors
- `agent/kv_agent_impl.go` - Added `storeProbeKey` const and `checkStoreAvailable` helper; `NewProtoconfKVAgent` now calls it
- `agent/kv_agent_rollout_impl.go` - `NewProtoconfKVAgentRollout` now calls the shared helper instead of its own inline probe

## Decisions Made
- Kept the reachability check (threat model T-260902erj-01 disposition "mitigate") rather than deleting it - an unreachable store must fail once at boot, not surface as a failure on every client subscription
- Sentinel key `.protoconf-agent-healthcheck` chosen dot-prefixed so it won't collide with real config paths; documented via GoDoc comment on `checkStoreAvailable` explaining both the non-empty-key requirement and the absent-key-is-success behavior
- Test fake is a new local `probeStore` type rather than extending `agent/dummykv`, since `dummykv.Exists` hardcodes `(true, nil)` and `dummykv.Get` returns a plain `fmt.Errorf` rather than `store.ErrKeyNotFound` - changing it would have broken every existing dummykv-backed test

## Deviations from Plan

None - plan executed exactly as written. Two pre-existing, out-of-scope environmental issues were discovered and documented (not fixed, per SCOPE BOUNDARY) rather than deviations from the plan itself:

### Deferred (out-of-scope, not auto-fixed)

**1. Pre-existing `go vet ./agent/` findings unrelated to this task**
- `agent/agent_test.go:27` - discarded `context.WithTimeoutCause` cancel func (introduced 2024-03-17, commit f7ba446)
- `agent/legacy.go:97` - unreachable code (introduced 2024-05-27, commit fb46573)
- Neither file is in this task's `files_modified` list; `go build ./...` and `go test ./agent/...` both pass regardless. Logged to `deferred-items.md`, left unfixed per scope discipline.

**2. `Test_cliCommand_Run/run_consul_server` hangs in this specific dev sandbox (environmental)**
- This machine has a real local Consul agent listening on `127.0.0.1:8500` (confirmed via `lsof -i :8500`, unrelated to this repo). The test expects the agent to fail to start (`want: 1`) assuming no Consul is reachable, but here it connects successfully - identical behavior before and after this fix, since a live server responds regardless of the exact probe key. With the store check passing, `agent.RunAgent` (called with `context.Background()`, no timeout) starts serving and blocks forever inside the test.
- Confirmed via `go test ./agent/... -count=1 -timeout 30s` (times out inside `run_consul_server`'s gRPC/OTel goroutines) versus `go test ./agent/... -count=1 -skip Test_cliCommand_Run` (all pass, including the new `TestStoreAvailabilityProbe`). Not caused by this task's changes; logged to `deferred-items.md`.

---

**Total deviations:** 0
**Impact on plan:** None - both deferred items are pre-existing and environment-specific, unrelated to the store-probe fix.

## Issues Encountered
- The plan's sanity-check command `go test ./test/... -count=1 -run TestE2E` doesn't match an actual test name (the e2e test function is named `Test`, not `TestE2E`). Ran `go test ./test/... -count=1` instead, which covers the same intent (agents constructed against filekv/dummykv) - all 4 tests pass.

## Verification Results

```
$ go build ./...
(exit 0, no output)

$ go test ./agent/... -count=1 -skip Test_cliCommand_Run
ok  	github.com/protoconf/protoconf/agent	22.5s
ok  	github.com/protoconf/protoconf/agent/configmaps	1.2s
ok  	github.com/protoconf/protoconf/agent/dummykv	7.6s
ok  	github.com/protoconf/protoconf/agent/filekv	7.3s
ok  	github.com/protoconf/protoconf/agent/otelkv	1.7s
(all PASS, including TestStoreAvailabilityProbe for both constructors)

$ grep -rn 'Exists(context.Background(), "/"' agent/
(no output - old empty-normalizing probe removed from both files)

$ go test ./test/... -count=1
ok  	github.com/protoconf/protoconf/test	4.9s
(TestMutationWithScripts, TestTLSMutation, TestAuthFlow, Test all PASS)
```

`go test ./agent/...` without `-skip` hangs specifically on `Test_cliCommand_Run/run_consul_server` due to the environmental Consul collision described above - not a regression from this change (see Deviations).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Agent starts correctly against a reachable-but-empty etcd store; no further action needed for this fix
- The two deferred items (pre-existing `go vet` findings, environment-specific Consul test hang) remain available in `deferred-items.md` for anyone auditing `agent/command_test.go` or `agent/legacy.go` separately - out of scope for this task

---
*Phase: 260902-erj*
*Completed: 2026-09-02*

## Self-Check: PASSED
- FOUND: agent/store_probe_test.go
- FOUND: agent/kv_agent_impl.go (modified, checkStoreAvailable present)
- FOUND: agent/kv_agent_rollout_impl.go (modified, checkStoreAvailable wired)
- FOUND commit: 9aefb2d (test)
- FOUND commit: db33bbd (fix)
