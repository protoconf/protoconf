---
phase: 11-concurrency-safe-lazy-registry-core
plan: 02
subsystem: server
tags: [grpc, service-discovery, mutation-server, protoreflect]

requires:
  - phase: 11
    provides: "plan 11-01's on-demand DescriptorRegistry.ParseOne/singleflight parse primitive — this plan is structurally independent of it (D-01 inherited) but shares the phase's motivation: a lazy registry makes s.parser's construction-time snapshot near-empty at Init() time"
provides:
  - "ProtoconfMutationServer.Init() discovers custom gRPC mutation services via its own one-time eager filesystem scan of src/, independent of s.parser's construction-time resolver snapshot"
  - "TestInitRegistersCustomService / TestInitWithNoCustomServices — regression coverage for CONS-01 (service discovery must not depend on what the compiler's registry happens to contain at startup)"
affects: [12, 13]

actuals:
  tokens: 1200
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "fresh throwaway utils.NewDescriptorRegistry() + eager Import for a one-time 'everything under this tree, right now' scan — the same shape module_service.go's Sync() already uses, applied at a second call site (Init's service discovery)"

key-files:
  created: []
  modified:
    - server/server.go
    - server/server_test.go

key-decisions:
  - "D-02 (locked, followed as written): the new discovery scan is scoped to service registration only. The two reflection.NewServerV1/NewServer calls in Init keep reading s.parser.FilesResolver/LocalResolver unchanged — widening the scan to also back reflection is a later phase's decision, not this plan's."
  - "D-01 (inherited, followed as written): this plan does not depend on 11-01 and runs independently — the Init discovery fix is structural (a second, separate registry instance), not a reaction to GetProtoRegistry() becoming lazy."
  - "The discovery registry is never stored on ProtoconfMutationServer and is garbage after Init returns — it exists solely to answer 'what services does src/ declare right now', not as a second long-lived resolver."

requirements-completed: [CONS-01]

coverage:
  - id: D1
    description: "Init() registers a custom gRPC mutation service (test.v1.TestService) declared under src/, discovered via Init's own eager scan, even when s.parser's construction-time snapshot is empty and no config has been compiled"
    requirement: CONS-01
    verification:
      - kind: unit
        ref: "server/server_test.go#TestInitRegistersCustomService"
        status: pass
    human_judgment: false
  - id: D2
    description: "Init() against a protoconf root with no custom services under src/ registers only the built-in services and does not panic or error"
    requirement: CONS-01
    verification:
      - kind: unit
        ref: "server/server_test.go#TestInitWithNoCustomServices"
        status: pass
    human_judgment: false
  - id: D3
    description: "Duplicate-registration guard: the existing protoregistry.GlobalFiles skip in the discovery loop prevents grpc.Server.RegisterService from panicking now that the scan sees more files than the old snapshot did"
    requirement: CONS-01
    verification:
      - kind: unit
        ref: "server/server_test.go#TestInitRegistersCustomService (require.NotPanics + exactly one test.v1.TestService entry)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Every pre-existing test in ./server/... stays green under -race, including TestProtoconfMutationServer_GenReflectionUI, Test_server_MutateConfig, TestProtoconfMutationServer_ReportProgress"
    requirement: CONS-01
    verification:
      - kind: integration
        ref: "go test -race ./server/..."
        status: pass
    human_judgment: false
  - id: D5
    description: "No new regression introduced repo-wide by this plan's change"
    requirement: CONS-01
    verification:
      - kind: integration
        ref: "go test -race $(go list ./... minus ./agent)"
        status: pass
    human_judgment: false

duration: 8min
completed: 2026-09-04
status: complete
---

# Phase 11 Plan 02: Concurrency-Safe Lazy Registry Core — Mutation Server Init Discovery Fix Summary

**`ProtoconfMutationServer.Init()` now discovers custom gRPC mutation services via its own eager one-time scan of `src/`, instead of ranging the parser's construction-time resolver snapshot — closing the CONS-01 gap where a lazy registry would silently drop every custom service for the process lifetime.**

## Performance

- **Duration:** 8 min
- **Started:** 2026-09-04T09:07:59Z (approx, following 11-01's completion)
- **Completed:** 2026-09-04T09:16:16Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- Added `TestInitRegistersCustomService`, a regression test that replaces the mutation server's construction-time parser resolver with a bare (well-known-types-only) registry — standing in for a future lazy registry's near-empty startup state — calls `Init` with no prior compile, and asserts `test.v1.TestService` and both its methods (`PutTestMessage`, `PutValidateMe`) are registered. Confirmed RED against the pre-fix `Init` (test failed, ranging the emptied snapshot found nothing).
- Added `TestInitWithNoCustomServices` for the empty edge: a protoconf root with nothing under `src/` still registers only the built-in services and `Init` does not panic.
- Fixed `Init()` to build a fresh, throwaway `utils.NewDescriptorRegistry()`, eagerly import every `.proto` under `filepath.Join(s.protoconfRoot, consts.SrcPath)`, and range that registry's files resolver for service discovery — the same shape `module_service.go`'s `Sync()` already uses for its own "everything under this tree, right now" need. A failed import logs and continues rather than aborting startup.
- Per locked decision D-02, left both `reflection.NewServerV1`/`reflection.NewServer` calls reading `s.parser.FilesResolver`/`LocalResolver` unchanged — the discovery scan backs service registration only.

## Task Commits

Each task was committed atomically:

1. **Task 1: Prove the gap — service registration test that fails against the snapshot-based discovery** - `418d00d` (test)
2. **Task 2: Init discovers services with its own eager scan of src/** - `c32b74f` (fix)

## Files Created/Modified

- `server/server_test.go` - Added `TestInitRegistersCustomService` (CONS-01 regression, RED then GREEN) and `TestInitWithNoCustomServices` (empty-src/ edge case)
- `server/server.go` - `Init` now builds `discoveryRegistry := utils.NewDescriptorRegistry()`, imports `src/` into it, and ranges `discoveryRegistry.GetFilesResolver()` instead of `s.parser.FilesResolver` for the service-registration loop; `regexp` added to the import block

## Decisions Made

- Followed D-02 and D-01 exactly as locked in the plan — no deviation from either. See `key-decisions` above.
- Used `protoconfparser` as the import alias for `compiler/lib/parser` in the test file (rather than a bare `parser` import) to avoid shadowing the local variable name `parser` already used inside `NewProtoconfMutationServer` in `server.go`, per the plan's explicit note on this collision risk.

## Deviations from Plan

None — plan executed exactly as written. Both tasks matched their `<action>`/`<behavior>` specs; no Rule 1-4 triggers encountered.

## Issues Encountered

- The plan's repo-wide verification command (`go test -race ./...`, matching CI) was run and, as anticipated in the phase notes, hangs on the pre-existing, unrelated `agent` package Consul-integration test (`Test_cliCommand_Run/run_consul_server`) that requires a live Consul instance unavailable in this sandbox. This was independently confirmed pre-existing by plan 11-01 (reproduced against the unmodified base commit via `git stash`). To verify this plan introduced no regression, `go test -race` was run against every package except `agent` (`go list ./... | grep -v '^github.com/protoconf/protoconf/agent$'`) and passed cleanly, and `go test -race ./server/...` (the package this plan actually touches) passed cleanly on its own, including all pre-existing tests (`TestProtoconfMutationServer_GenReflectionUI`, `Test_server_MutateConfig`, `TestProtoconfMutationServer_ReportProgress`). Not auto-fixed — out of scope per the deviation rules' scope boundary (pre-existing, unrelated to this plan's files).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- CONS-01 is closed: the mutation server's gRPC service catalog is now independent of what the compiler's registry happens to contain at `Init()` time, which was the hard blocking co-requirement for making that registry lazy (per STATE.md's Blockers/Concerns entry for Phase 11).
- `go test -race ./server/...` and `go test -race` on every other package (excluding the pre-existing, unrelated `agent` Consul-integration hang) are green.
- No blockers for plan `11-03` or subsequent phases.

---
*Phase: 11-concurrency-safe-lazy-registry-core*
*Completed: 2026-09-04*

## Self-Check: PASSED

- Both modified files confirmed present on disk: `server/server.go`, `server/server_test.go`.
- Both commit hashes (`418d00d`, `c32b74f`) confirmed in `git log`.
- All plan-level `<acceptance_criteria>` re-verified: `go test ./server/... -run TestInitWithNoCustomServices -v` → `--- PASS`; `go test ./server/... -run TestInitRegistersCustomService -v` → `--- PASS` (GREEN after Task 2, was `--- FAIL` at end of Task 1 as required); `go test ./server/... -run TestProtoconfMutationServer_GenReflectionUI -v` → `--- PASS`; `server/server.go` unchanged after Task 1 (`git diff --name-only` showed only `server_test.go`).
- Plan-level `<verification>` re-run: `go test -race ./server/...` green; `go test -race` on all non-`agent` packages green; `go vet ./server/...` clean.
