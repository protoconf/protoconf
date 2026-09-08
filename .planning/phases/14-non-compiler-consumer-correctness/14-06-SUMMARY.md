---
phase: 14-non-compiler-consumer-correctness
plan: 06
subsystem: testing
tags: [race-detector, grpc, bufconn, mutation-server, singleflight, concurrency]

# Dependency graph
requires:
  - phase: 14-03
    provides: "NewProtoconfMutationServer on lib.NewLazyModuleService; MutateConfig's marshal resolver on s.parser.TypeResolver; reflection reads the retained discoveryFiles resolver (D-06)"
  - phase: 14-04
    provides: "GenReflectionUI/collectExamples no longer abort the mutable_config/ walk on a resolution failure"
  - phase: 12
    provides: "the unpaced tight-loop race-test pattern (utils/growable_resolver_race_test.go) and its lock-removal sanity-check precedent"
  - phase: 14-07
    provides: "the agent-side twin proof (bufconn e2e + dedicated tight-loop + lock-removal validation) this plan mirrors for the mutation server"
provides:
  - "SAFE-02 proof for the mutation server: a bufconn e2e test with 16 concurrent MutateConfig clients through the real bearerTokenInterceptor and real gRPC codec, overlapped with a concurrent GenReflectionUI walk, plus a dedicated unpaced tight-loop test shown capable of failing"
affects: [server, utils]

# Actuals (#2632)
actuals:
  tokens: 2366
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "bufconn + grpc.ChainUnaryInterceptor(bearerTokenInterceptor) inline harness (TestAuthFlow precedent) driven by an errgroup of N concurrent clients instead of TestAuthFlow's sequential subtests"
    - "Reflection-walk goroutine overlapping a writer wave: unpaced for { select { case <-done: return; default: GenReflectionUI(...) } } loop started before errgroup.Wait() and stopped after it"
    - "Unpaced tight-loop reader goroutines (for { select { case <-done: return; default: ... } }) alongside an errgroup of writers resolving distinct never-before-seen symbols, adapted from utils/growable_resolver_race_test.go's TestRegisterFileRacesRangeFiles and 14-07's filekv analog"
    - "Failure-detection sanity check: temporarily remove the lock under test, confirm the race detector reports it (3/3 runs), then git checkout to restore before the task's commit"

key-files:
  created:
    - server/mutate_config_race_test.go
  modified: []

key-decisions:
  - "Task 1's request payload wraps test.v1.TestMessage (resolved once via srv.parser.TypeResolver.FindMessageByURL before the writer wave starts) rather than a distinct type per goroutine -- Task 1 proves the real-interceptor/real-codec/concurrent-file-write shape; forcing 16 independent first-time resolutions is Task 2's job (the dedicated tight-loop test), not Task 1's."
  - "GenReflectionUI is passed a bare grpc.NewServer() per plan's literal instruction, which lacks reflection registration -- standalone.HandlerViaReflection fails with an Unimplemented ServerReflection error each call, but this is harmless: collectExamples() (the concurrency-relevant read of mutable_config/) runs and completes before that failure, so the reflection walk still genuinely overlaps the writer wave. The plan's own acceptance criteria anticipate GenReflectionUI returning a non-nil error and require no assertion on it."
  - "Task 2's writers resolve tight.v1..v23 (23 distinct, previously-unparsed custom types, one per goroutine) so ParseOne's singleflight cannot collapse them into one memoised entry -- mirrors 14-07's filekv precedent and utils/growable_resolver_race_test.go's corpus-based approach."

requirements-completed: [SAFE-02]

coverage:
  - id: D1
    description: "16 concurrent gRPC clients calling MutateConfig over one shared bufconn connection, through the real bearerTokenInterceptor and real gRPC codec, against a long-lived mutation server, complete without error and without a data race under go test -race, with a concurrent GenReflectionUI walk overlapping the writer wave (D-07 e2e half)"
    requirement: "SAFE-02"
    verification:
      - kind: unit
        ref: "server/mutate_config_race_test.go#TestMutateConfigConcurrentClientsAreRaceFree"
        status: pass
    human_judgment: false
  - id: D2
    description: "Dedicated unpaced tight-loop test forcing the mutation server's ParseOne read/record interleaving continuously via s.parser.TypeResolver.FindMessageByURL and s.parser.ReadConfig, proven capable of failing via temporary lock removal (3/3 runs reported WARNING: DATA RACE) (D-07 dedicated half)"
    requirement: "SAFE-02"
    verification:
      - kind: unit
        ref: "server/mutate_config_race_test.go#TestMutationServerResolverTightLoopIsRaceFree"
        status: pass
    human_judgment: false

duration: 25min
completed: 2026-09-08
status: complete
---

# Phase 14 Plan 06: Mutation Server Concurrency Race Proofs Summary

**Two new race tests prove SAFE-02 for the mutation server's lazy-registry path (bufconn e2e with a concurrent reflection walk + a dedicated unpaced tight loop), with the tight loop's failure-detection capability verified by a temporary lock removal that reproduced `WARNING: DATA RACE` on 3/3 runs.**

## Performance

- **Duration:** 25 min
- **Started:** 2026-09-08T20:55:00Z (approx)
- **Completed:** 2026-09-08T21:14:30Z (approx)
- **Tasks:** 2
- **Files modified:** 1 (new test file only; no production files changed)

## Accomplishments
- `TestMutateConfigConcurrentClientsAreRaceFree` (server package): 16 concurrent gRPC clients call `MutateConfig` over one shared bufconn connection through the real `bearerTokenInterceptor` and real gRPC codec, against one long-lived `ProtoconfMutationServer` with no compiler attached, so the marshal resolves `in.Value` through `s.parser.TypeResolver` -- the lazy-registry path SAFE-02 is about. A background goroutine runs `GenReflectionUI` in a loop for the duration of the writer wave, overlapping the periodic reflection walk with in-flight mutations. Clean under `-race` and `-count=2`; asserts 16 distinct files exist under `mutable_config/`.
- `TestMutationServerResolverTightLoopIsRaceFree` (server package): two unpaced reader goroutines hammer `s.parser.TypeResolver.FindMessageByURL` (an already-resolved symbol) and `s.parser.ReadConfig` (a materialized config) with no sleep and no pacing, while an errgroup of 23 writer goroutines each resolve one distinct, previously-unparsed custom message type (`tight.v1`..`tight.v23`) concurrently -- forcing `ParseOne`'s read/record interleaving on the shared `*utils.DescriptorRegistry` continuously.
- Failure-detection validation executed as specified: temporarily removed the `d.mu.Lock()`/`d.mu.Unlock()` pair bracketing `ParseOne`'s `d.recordFileLocked(fds[0])` call and the canonical-entry lookup in `utils/utils.go` (lines 395/408). Ran the tight-loop test under `-race` three times; all three runs reported `WARNING: DATA RACE` (4, 15, and 13 race reports respectively) on `recordFileLocked`'s map write (`utils/utils.go:450`, `runtime.mapaccess2_faststr`) racing another writer's identical unsynchronized write, reached via `LoadSymbolByScan` -> `ParseOne` -> `singleflight.Group.Do` -> `recordFileLocked` -- exactly the path the mutation server's resolver reads exercise. First line of the race detector's output: `WARNING: DATA RACE`. Restored via `git checkout -- utils/utils.go`; `git diff --exit-code utils/utils.go` confirms byte-identical restoration.

## Task Commits

Each task was committed atomically:

1. **Task 1: bufconn e2e -- N concurrent MutateConfig clients plus a concurrent reflection walk** - `26f9f97` (test)
2. **Task 2: dedicated unpaced tight-loop resolver race test, validated by temporary lock removal** - `434b4b4` (test)

**Plan metadata:** (this commit)

_Note: Both tasks are proof-only against already-implemented behavior (14-03's lazy-registry flip and 14-04's non-aborting reflection walk), so neither produced a feat/refactor commit -- matching 14-02's and 14-07's precedent for their analogous proof tests._

## Files Created/Modified
- `server/mutate_config_race_test.go` - `TestMutateConfigConcurrentClientsAreRaceFree` (bufconn e2e + concurrent reflection walk), `TestMutationServerResolverTightLoopIsRaceFree` (dedicated tight-loop resolver race test), and their fixture builders (`newMutationServerTightLoopRoot`)

## Decisions Made
- Task 1's 16 clients all wrap the same fixture type (`test.v1.TestMessage`), resolved once before the writer wave starts, rather than 16 distinct types -- Task 1's job is proving the real-interceptor/real-codec/concurrent-distinct-file-write shape; forcing independent first-time resolutions under concurrency is Task 2's dedicated tight-loop job.
- `GenReflectionUI` is invoked with a bare `grpc.NewServer()` per the plan's literal instruction; this has no reflection service registered, so `standalone.HandlerViaReflection` fails with an `Unimplemented ServerReflection` error on every call. This is harmless to the test's purpose: `collectExamples()` -- the actual concurrent read of `mutable_config/` -- runs and completes before that failure occurs, so the reflection walk still genuinely overlaps the writer wave. The plan's own acceptance criteria anticipate a non-nil `GenReflectionUI` error and require no assertion on it.
- Task 2's writers resolve `tight.v1.Msg`..`tight.v23.Msg` (23 distinct, previously-unparsed types, one per goroutine, each in its own package directory) so `ParseOne`'s singleflight cannot collapse them into one memoised entry -- matches 14-07's `filekv` tight-loop precedent and `utils/growable_resolver_race_test.go`'s corpus-based approach.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- SAFE-02 is now proven for both consumers named in the requirement: the agent (14-07) and the mutation server (this plan) -- both via an executed, verified failure-detection check, not just a passing-by-construction test.
- No production files were touched; `utils/utils.go`, `go.mod`, and `go.sum` are all byte-identical to their state at plan start.
- This is the last plan in Phase 14 (all 8 plans now have SUMMARY.md files); the phase is ready for `/gsd-verify-work`.

---
*Phase: 14-non-compiler-consumer-correctness*
*Completed: 2026-09-08*

## Self-Check: PASSED

- FOUND: server/mutate_config_race_test.go
- FOUND commit: 26f9f97 (Task 1)
- FOUND commit: 434b4b4 (Task 2)
- Re-ran plan-level verification: `go test -race -count=2 ./server/... -run 'TestMutateConfigConcurrentClientsAreRaceFree|TestMutationServerResolverTightLoopIsRaceFree'` -> ok, no `WARNING: DATA RACE`
- Re-ran: `go test -race -count=1 ./server/... ./test/...` -> both packages `ok`, no `WARNING: DATA RACE`
- Re-ran: `git diff --exit-code utils/utils.go go.mod go.sum` -> exit 0 (clean)
- plan_head_before: 722c824b5a55c0edbe02a8891f3ee92b3c48c13f, commits (measured via git rev-list --count): 2
