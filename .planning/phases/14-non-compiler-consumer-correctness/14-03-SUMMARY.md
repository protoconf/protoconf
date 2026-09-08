---
phase: 14-non-compiler-consumer-correctness
plan: 03
subsystem: server
tags: [protoreflect, dynamicpb, protojson, grpc-reflection, type-resolution, lazy-loading]

# Dependency graph
requires:
  - phase: 14-non-compiler-consumer-correctness
    provides: "14-01's verified D-01/D-03 construction-flip pattern (lib.NewLazyModuleService, LocalResolver -> TypeResolver swap)"
provides:
  - "server/server.go's mutation server on the lazy construction path (D-01) with MutateConfig's marshal resolver reading the tiered s.parser.TypeResolver (D-03)"
  - "gRPC reflection wired to the retained discovery-registry files resolver (D-06), proven by a real ServerReflectionInfo round trip against a src/-declared custom service and both current and legacy hand-registered services"
affects: [14-04, 14-05, 14-06, 14-07, 14-08]

# Actuals (#2632)
actuals:
  tokens: 2060
  tasks: 2
  commits: 3

tech-stack:
  added: []
  patterns:
    - "D-01/D-03 construction flip + marshal-resolver swap, same as 14-01/14-02"
    - "Discovery-registry files resolver retained past service registration and wired as reflection's DescriptorResolver, so a client that can list a service can also describe it, without widening it to a general type-URL/Any resolver (D-04)"

key-files:
  created: []
  modified:
    - server/server.go
    - server/server_test.go

key-decisions:
  - "Followed the RED-GREEN TDD cycle for Task 2 since, unlike 14-01/14-02, this task's implementation edit (mirroring six well-known files onto discoveryFiles, swapping DescriptorResolver, rewriting the stale comment) precedes rather than follows the proof: wrote TestReflectionDescribesCustomAndBuiltinServices first, confirmed it failed intentionally on test.v1.TestService with 'proto: not found' (RED, committed as test(14-03)), then implemented the fix and confirmed green (GREEN, committed as feat(14-03))."
  - "Logged the pre-existing go vet ./test/... lostcancel finding (test/e2e_test.go lines 327/411, dating to commit 99e33a44, 2023-12-14) to deferred-items.md instead of fixing it -- confirmed via isolated grep and a stash-free diff that it is unrelated to this plan's server.go changes; go vet ./server/... alone is silent, and go test ./test/... itself is unaffected since its built-in vet subset excludes lostcancel."

requirements-completed: [CONS-04, SAFE-02]

coverage:
  - id: D1
    description: "The mutation server constructs its module service lazily, and MutateConfig still marshals a request value whose custom type was not present at construction"
    requirement: "CONS-04"
    verification:
      - kind: e2e
        ref: "test/e2e_test.go#TestAuthFlow"
        status: pass
    human_judgment: false
  - id: D2
    description: "gRPC reflection describes every service it can list -- a src/-declared custom service, the current mutation service, and the legacy hand-registered mutation service -- proven over a real ServerReflectionInfo round trip"
    requirement: "SAFE-02"
    verification:
      - kind: integration
        ref: "server/server_test.go#TestReflectionDescribesCustomAndBuiltinServices"
        status: pass
    human_judgment: false
  - id: D3
    description: "The retained discovery-registry files resolver gained no reachability beyond reflection -- no new struct field, and the stale deferral comment describing this as a later phase's decision is gone"
    requirement: "SAFE-02"
    verification:
      - kind: unit
        ref: "grep acceptance criteria over server/server.go (DescriptorResolver, RegisterFile, ExtensionResolver, struct-field, 'later phase' counts)"
        status: pass
    human_judgment: false

duration: 12min
completed: 2026-09-08
status: complete
---

# Phase 14 Plan 03: Mutation Server Lazy Construction + Reflection Completeness Summary

**`server/server.go`'s mutation server now constructs lazily via `lib.NewLazyModuleService`, `MutateConfig` marshals through the tiered `TypeResolver`, and gRPC reflection reads the retained discovery-registry resolver so a client that can list a service can also describe it — proven end-to-end by a new `TestReflectionDescribesCustomAndBuiltinServices` round-trip test.**

## Performance

- **Duration:** ~12 min
- **Started:** 2026-09-08T12:48:00Z
- **Completed:** 2026-09-08T13:00:00Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- `NewProtoconfMutationServer` constructs its module service via `lib.NewLazyModuleService` instead of `lib.NewModuleService` (D-01)
- `MutateConfig`'s marshal resolver reads `s.parser.TypeResolver` instead of the construction-time-only `LocalResolver` (D-03) — the pre-existing `TestAuthFlow` e2e test proves a `google.protobuf.Struct` value still marshals correctly through the tiered resolver
- `Init`'s discovery registry now mirrors the six well-known files (reflection v1/v1alpha, health, legacy mutation, `protoconf_pb`, agent service) hand-registered on the serving parser onto `discoveryFiles`, and both `reflection.ServerOptions.DescriptorResolver` fields (v1 and v1alpha) now read `discoveryFiles` instead of `s.parser.FilesResolver` (D-06)
- `ExtensionResolver` on both reflection constructions stays on `s.parser.LocalResolver`, unchanged — `RegistryTypeResolver.FindExtensionByName` delegates to the same construction-time snapshot either way (D-04 exclusion)
- No new struct field on `ProtoconfMutationServer`: `discoveryRegistry`/`discoveryFiles` remain `Init`-local, captured only by the reflection servers' own references for the process lifetime (D-04 narrowness)
- The stale block comment promising reflection-completeness widening to "a later phase's decision" is rewritten to state the actual, now-shipped behavior
- New `TestReflectionDescribesCustomAndBuiltinServices` proves the full round trip: `FileContainingSymbol` returns a non-empty descriptor for `test.v1.TestService` (the src/-declared custom service), `protoconf.v1.ProtoconfMutationService`, and the legacy `v1.ProtoconfMutationService`; `ListServices` returns all three names

## Task Commits

Each task was committed atomically:

1. **Task 1: D-01 lazy construction and D-03 tiered resolver on MutateConfig's marshal** - `aa55135` (feat)
2. **Task 2 (RED): failing test for D-06 reflection completeness** - `7ed3efa` (test)
2. **Task 2 (GREEN): D-06 reflection reads the retained discovery resolver** - `5f7e27a` (feat)

**Plan metadata:** (this commit)

_Note: Task 2 followed a full RED-GREEN cycle (no REFACTOR commit — the GREEN implementation needed no cleanup). Unlike 14-01/14-02's TDD tasks, this one carried real implementation, not just proof of an already-shipped change, so the test was written and confirmed failing before the fix landed._

## Files Created/Modified
- `server/server.go` - `NewProtoconfMutationServer` construction flipped to `lib.NewLazyModuleService`; `MutateConfig`'s `resolver` swapped to `s.parser.TypeResolver`; `Init` mirrors six well-known files onto `discoveryFiles`, swaps both `DescriptorResolver` fields to `discoveryFiles`, and rewrites the stale discovery-registry comment
- `server/server_test.go` - Added `TestReflectionDescribesCustomAndBuiltinServices`

## Decisions Made
- Task 2 used a genuine RED-GREEN cycle (test first, confirmed failing on `test.v1.TestService` with `proto: not found`, then implementation), since this task's fix is real new behavior rather than a proof of an already-shipped one-line flip.
- Logged the pre-existing `go vet ./test/...` `lostcancel` finding (unrelated `context.WithTimeout` discards in `test/e2e_test.go`, dating to 2023-12-14) to `deferred-items.md` rather than fixing it — confirmed out of scope via isolated `go vet ./server/...` (silent) and git blame.

## Deviations from Plan

### Auto-fixed Issues

None — both changes matched 14-PATTERNS.md's pre-verified diff exactly.

### Deferred (Out of Scope)

**1. [Scope boundary] Pre-existing `go vet ./test/...` `lostcancel` findings**
- **Found during:** Task 1's `<verify>` (`go build ./... && go vet ./server/... ./test/...`)
- **Issue:** Two `context.WithTimeout` calls in `test/e2e_test.go` (lines 327, 411) discard their cancel funcs, tripping `go vet`'s `lostcancel` analyzer
- **Why deferred:** `git blame` shows both lines date to commit `99e33a44` (2023-12-14), unrelated to this plan's `server.go` changes; `go vet ./server/...` alone is silent, and `go test ./test/...` (which runs a narrower built-in vet subset excluding `lostcancel`) is unaffected — `TestAuthFlow` passes
- **Logged:** `.planning/phases/14-non-compiler-consumer-correctness/deferred-items.md`

---

**Total deviations:** 0 auto-fixed, 1 deferred (pre-existing, out of scope)
**Impact on plan:** None on this plan's own correctness. The deferred item is a pre-existing lint-only finding unrelated to D-01/D-03/D-06.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `server/server.go` is now fully on the lazy construction path with both D-03 (custom-type marshal) and D-06 (reflection completeness) regressions closed and covered by tests. `go build ./...`, `go vet ./server/...`, and `go test -race -count=1 ./server/... ./test/...` are all green; `go.mod`/`go.sum` are untouched; no `discovery*` struct field exists on `ProtoconfMutationServer`.
- Three of the four D-01/D-03 consumer sites (`inserter`, `agent/filekv`, `server`) are now converted, and `server`'s D-06 reflection-completeness requirement is closed. `mutate/mutate.go` (D-01/D-03) remains for a later plan in this phase.
- No blockers.

---
*Phase: 14-non-compiler-consumer-correctness*
*Completed: 2026-09-08*

## Self-Check: PASSED

- FOUND: server/server.go
- FOUND: server/server_test.go
- FOUND commit: aa55135
- FOUND commit: 7ed3efa
- FOUND commit: 5f7e27a
