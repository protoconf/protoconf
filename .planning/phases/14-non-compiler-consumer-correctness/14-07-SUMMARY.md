---
phase: 14-non-compiler-consumer-correctness
plan: 07
subsystem: testing
tags: [race-detector, grpc, bufconn, filekv, agent, singleflight, concurrency]

# Dependency graph
requires:
  - phase: 14-02
    provides: "agent/filekv's New constructs through lib.NewLazyModuleService, so Get resolves types on demand through the shared tiered resolver"
  - phase: 12
    provides: "the unpaced tight-loop race-test pattern (utils/growable_resolver_race_test.go) and its lock-removal sanity-check precedent"
provides:
  - "SAFE-02 proof for the agent: a bufconn e2e test over the real lazy filekv-backed agent, plus a dedicated unpaced tight-loop test shown capable of failing"
  - "CONS-03 concurrency-edge proof: parallel Get calls for distinct keys never cross one key's value onto another's response"
affects: [agent, agent/filekv, utils]

# Actuals (#2632)
actuals:
  tokens: 2188
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Package-local many-type temp-root fixture (one package+message per index) forces genuinely independent on-demand resolutions instead of one memoised parse shared by every goroutine"
    - "Unpaced tight-loop reader goroutines (for { select { case <-done: return; default: ... } }) alongside an errgroup of writers, adapted from utils/growable_resolver_race_test.go's TestRegisterFileRacesRangeFiles precedent"
    - "Failure-detection sanity check: temporarily remove the lock under test, confirm the race detector (or a runtime fatal) reports it, then git checkout to restore before the task's commit"

key-files:
  created:
    - agent/kv_agent_race_test.go
    - agent/filekv/filekv_race_test.go
  modified: []

key-decisions:
  - "Task 1 is proof-only, no implementation commit: 14-02 already flipped filekv to lib.NewLazyModuleService, so there is no new GREEN to make pass -- matches 14-02's own precedent for its four proof tests."
  - "Reader goroutines in TestFileKVGetTightLoopIsRaceFree hammer an already-resolved key (Tier 0 msgregistry hit, no d.mu contention on their own); the race is proven by the writer side alone -- concurrent ParseOne calls on distinct, non-singleflight-collapsed keys writing to the shared d.FileRegistry map without the lock. Confirmed empirically: 3/3 lock-removed runs reported WARNING: DATA RACE."

requirements-completed: [SAFE-02, CONS-03]

coverage:
  - id: D1
    description: "12 concurrent gRPC clients streaming SubscribeForConfig against one long-lived agent backed by the lazy filekv store, each receiving their own config with no data race (D-07 e2e half)"
    requirement: "SAFE-02"
    verification:
      - kind: unit
        ref: "agent/kv_agent_race_test.go#TestSubscribeForConfigConcurrentClientsAreRaceFree"
        status: pass
    human_judgment: false
  - id: D2
    description: "Dedicated unpaced tight-loop test forcing the filekv read/record interleaving continuously, proven capable of failing via temporary lock removal (D-07 dedicated half)"
    requirement: "SAFE-02"
    verification:
      - kind: unit
        ref: "agent/filekv/filekv_race_test.go#TestFileKVGetTightLoopIsRaceFree"
        status: pass
    human_judgment: false
  - id: D3
    description: "Concurrent Get calls for distinct keys each return the correct config bytes for the key requested, never crossing one key's value onto another's response"
    requirement: "CONS-03"
    verification:
      - kind: unit
        ref: "agent/filekv/filekv_race_test.go#TestFileKVConcurrentGetReturnsCorrectValuePerKey"
        status: pass
    human_judgment: false

duration: 35min
completed: 2026-09-08
status: complete
---

# Phase 14 Plan 07: Agent Concurrency Race Proofs Summary

**Two new race tests prove SAFE-02 for the agent over the real lazy filekv path (bufconn e2e + a dedicated unpaced tight loop), with the tight loop's failure-detection capability verified by a temporary lock removal that reproduced `WARNING: DATA RACE` on 3/3 runs.**

## Performance

- **Duration:** 35 min
- **Started:** 2026-09-08T20:22:00Z (approx)
- **Completed:** 2026-09-08T20:57:00Z (approx)
- **Tasks:** 2
- **Files modified:** 2 (both new test files; no production files changed)

## Accomplishments
- `TestSubscribeForConfigConcurrentClientsAreRaceFree` (agent package): 12 concurrent gRPC clients stream `SubscribeForConfig` against one `ProtoconfKVAgent` backed by `filekv.New` over a 12-type fixture root, each asserting its own type URL. Clean under `-race` and `-count=2`.
- `TestFileKVGetTightLoopIsRaceFree` (filekv package): two unpaced reader goroutines hammer an already-resolved key while an errgroup of writers resolves 23 distinct not-yet-parsed keys concurrently, forcing the `ParseOne` read/record interleaving on the shared `*utils.DescriptorRegistry` continuously.
- `TestFileKVConcurrentGetReturnsCorrectValuePerKey` (filekv package): 24 keys resolved concurrently, each asserted to carry its own type URL -- proves CONS-03's no-crossed-response guarantee.
- Failure-detection validation executed as specified: temporarily removed the `d.mu.Lock()`/`d.mu.Unlock()` pair bracketing `ParseOne`'s `d.recordFileLocked(fds[0])` call and the canonical-entry lookup in `utils/utils.go`, ran the tight-loop test under `-race` three times, and all three runs reported `WARNING: DATA RACE` on `recordFileLocked`'s map write (`utils/utils.go:450`, `runtime.mapaccess2_faststr`) racing another writer's identical unsynchronized write -- reached via `LoadSymbolByScan` -> `ParseOne` -> `singleflight.Group.Do` -> `recordFileLocked`, exactly the code path Get exercises. Restored via `git checkout -- utils/utils.go`; `git diff --exit-code utils/utils.go` confirms byte-identical restoration.

## Task Commits

Each task was committed atomically:

1. **Task 1: bufconn e2e over lazy filekv-backed agent** - `b200b94` (test) -- no feat/refactor commit; 14-02 already shipped the only implementation change this test proves.
2. **Task 2: dedicated tight-loop filekv race test + lock-removal validation** - `2bae1b8` (test)

**Plan metadata:** (this commit)

_Note: Both tasks are proof-only against already-implemented behavior (14-02's lazy filekv flip), so neither produced a feat/refactor commit -- matching 14-02's own precedent for its analogous proof tests._

## Files Created/Modified
- `agent/kv_agent_race_test.go` - `TestSubscribeForConfigConcurrentClientsAreRaceFree` and its 12-type temp-root fixture builder (`newRaceTestRoot`)
- `agent/filekv/filekv_race_test.go` - `TestFileKVGetTightLoopIsRaceFree`, `TestFileKVConcurrentGetReturnsCorrectValuePerKey`, and their 24-type temp-root fixture builder (`newTightLoopRoot`)

## Decisions Made
- Task 1 committed as a single `test(14-07)` commit with no accompanying `feat`/`refactor` -- the plan explicitly forbids modifying `agent/filekv/filekv.go` or `agent/kv_agent_impl.go` in this task, and 14-02 already shipped the lazy-registry flip that makes the test pass. This mirrors 14-02's own "TDD task 2's four proof tests" precedent recorded in STATE.md.
- The tight-loop test's reader goroutines target an already-resolved key (Tier 0 `MessageRegistry` hit), which does not itself contend on `d.mu` -- the race is proven entirely by the writer side: `errgroup` goroutines resolving distinct, previously-unparsed keys concurrently, each independently reaching `ParseOne`'s `d.group.Do` (no singleflight collapse since the keys differ) and racing on `recordFileLocked`'s map writes once the lock is removed. Verified empirically (3/3 `WARNING: DATA RACE` reports) before committing the test as-is.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- `go test -race -count=1 ./agent/...` (the plan-level `<verify>` command for Task 2) hangs for ~10 minutes on the pre-existing `Test_cliCommand_Run/run_consul_server` case, which requires a live Consul at `127.0.0.1:8500` -- documented as out-of-scope in the plan's `<known_environment_note>`. Bounded every run with `-timeout 300s -skip 'Test_cliCommand_Run'` per that note; all runs (including `-count=2` across `./agent/...`) passed clean with no `WARNING: DATA RACE`.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- SAFE-02 and CONS-03 are both proven for the agent's real serving path (`filekv.Get` -> `parser.ReadConfig` -> shared tiered resolver), backed by an executed, verified failure-detection check -- not just a passing-by-construction test.
- No production files were touched; `agent/filekv/filekv.go`, `agent/kv_agent_impl.go`, `utils/utils.go`, `go.mod`, and `go.sum` are all byte-identical to their state at plan start.
- Ready for the next plan in Phase 14 (wave 3/4 as applicable).

---
*Phase: 14-non-compiler-consumer-correctness*
*Completed: 2026-09-08*

## Self-Check: PASSED

- FOUND: agent/kv_agent_race_test.go
- FOUND: agent/filekv/filekv_race_test.go
- FOUND: .planning/phases/14-non-compiler-consumer-correctness/14-07-SUMMARY.md
- FOUND commit: b200b94 (Task 1)
- FOUND commit: 2bae1b8 (Task 2)
- Re-ran plan-level verification: `go test -race -timeout 300s -skip 'Test_cliCommand_Run' -count=2 ./agent/...` -> all packages `ok`, no `WARNING: DATA RACE`
- Re-ran: `git diff --exit-code utils/utils.go agent/filekv/filekv.go agent/kv_agent_impl.go go.mod go.sum` -> exit 0 (clean)
