---
phase: quick-260902-eie
plan: 01
subsystem: infra
tags: [protobuf, protojson, dynamicpb, kvstore, inserter]

# Dependency graph
requires: []
provides:
  - "Any-typed materialized configs (any.new() output) now insert successfully into the KV store"
  - "Regression test locking in Any resolution on the inserter's config.json write path"
affects: [inserter, compiler-any-fields]

# Actuals (#2632)
actuals:
  tokens: 542
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns: ["config.json write path now shares the same LocalResolver as the read path (parser.ReadConfig), closing the last resolver gap identified in the repo-wide audit"]

key-files:
  created: []
  modified:
    - inserter/inserter.go
    - inserter/inserter_test.go

key-decisions:
  - "Backported only the Any-resolution one-line fix from upstream PR #496 -- left the two sibling protojson.MarshalOptions marshals (rollout, metadata) untouched since they marshal generated protoconf-owned types with no reachable Any field"
  - "Wrote the regression test as a dedicated top-level function rather than a row in the existing table-driven test, since the table's strings.HasPrefix(\"{\") assertion can't distinguish a resolved Any from any other JSON and protojson randomizes indentation whitespace"

requirements-completed: [QUICK-260902-EIE]

coverage:
  - id: D1
    description: "Inserting a materialized config with populated singular/repeated/map google.protobuf.Any fields of a user-defined (LocalResolver-only) type succeeds and writes the resolved payload to config.json"
    requirement: QUICK-260902-EIE
    verification:
      - kind: unit
        ref: "inserter/inserter_test.go#TestProtoconfInserter_InsertConfig_AnyResolution"
        status: pass
    human_judgment: false

duration: 12min
completed: 2026-09-02
status: complete
---

# Quick Task 260902-eie: Fix Any Resolution in Inserter config.json Marshal Summary

**One-line fix: added `Resolver: i.parser.LocalResolver` to the config.json `protojson.MarshalOptions` in `XXXinsertVersion`, so materialized configs containing `google.protobuf.Any` of user-defined types (built via the `any.new()` Starlark builtin) insert successfully instead of aborting.**

## Performance

- **Duration:** 12 min
- **Started:** 2026-09-02T03:23:00Z (approx)
- **Completed:** 2026-09-02T03:35:31Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- Fixed the missing `Resolver` on `(*ProtoconfInserter).XXXinsertVersion`'s config.json `protojson.Marshal`, matching the resolver already used on the read side (`parser.ReadConfig`) and the compiler's write side (`compiler/lib/compiler.go`)
- Added `TestProtoconfInserter_InsertConfig_AnyResolution`, a dedicated regression test that inserts the existing `field_type_any_test.materialized_JSON` fixture and asserts the written `config.json` contains the resolved Any payload for all three shapes (singular, repeated, map)
- Confirmed the test is a genuine regression test: manually reverted the fix, ran `go test ./inserter/... -run AnyResolution`, observed the exact `unable to resolve "type.googleapis.com/test.v1.TestMessage": not found` failure from the bug report, then restored the fix and confirmed the test passes

## Task Commits

Each task was committed atomically:

1. **Task 1: Pass LocalResolver to the config.json protojson.Marshal** - `f65bdd7` (fix)
2. **Task 2: Regression test proving Any values survive the config.json write** - `841ca1b` (test)

**Plan metadata:** committed separately by the orchestrator (docs commit not made by this executor per constraints)

## Files Created/Modified
- `inserter/inserter.go` - Added `Resolver: i.parser.LocalResolver` to the config.json marshal's `protojson.MarshalOptions` (one line changed)
- `inserter/inserter_test.go` - Added `TestProtoconfInserter_InsertConfig_AnyResolution`

## Decisions Made
- Did not touch the two sibling `protojson.MarshalOptions` marshals in the same file (rollout config at line ~308, metadata at line ~390) -- both marshal generated protoconf-owned Go types registered in `protoregistry.GlobalTypes` with no field path that can reach a `google.protobuf.Any`, per the plan's repo-wide sibling-call-site audit
- Kept the regression test as a standalone top-level function instead of extending the existing table-driven `TestProtoconfInserter_InsertConfig`, to avoid a flaky/imprecise `strings.HasPrefix` assertion

## Deviations from Plan

None - plan executed exactly as written. No new imports, no new fixture files, existing test functions unmodified.

## Issues Encountered

None. The existing `field_type_any_test.materialized_JSON` fixture reproduced the bug exactly as documented in the plan's `<prior_investigation>` section, and the fix + test worked on the first attempt.

## Red/Green Verification (Task 2)

- **RED (fix manually reverted):** `go test ./inserter/... -count=1 -run AnyResolution -v` -> `FAIL`, error: `proto: google.protobuf.Any: unable to resolve "type.googleapis.com/test.v1.TestMessage": not found`
- **GREEN (fix restored):** `go test ./inserter/... -count=1 -run AnyResolution -v` -> `PASS`

## Final Verification

```
$ go build ./...
(clean, no output)

$ go test ./inserter/... -count=1
ok  	github.com/protoconf/protoconf/inserter	17.086s
?   	github.com/protoconf/protoconf/inserter/config/v1	[no test files]

$ git diff --stat a659cd34f3cec45086f1f94b1bd9d074cee04d55 HEAD -- inserter/inserter.go inserter/inserter_test.go
 inserter/inserter.go      |  2 +-
 inserter/inserter_test.go | 24 ++++++++++++++++++++++++
 2 files changed, 25 insertions(+), 1 deletion(-)
```

Exactly two files touched; production diff is a single line change; no `.pb.go` file touched; no `go generate` run.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- The inserter's write path now matches the compiler and parser's Any-resolution behavior, closing the last gap identified in the upstream PR #496 audit.
- No blockers. The remaining parts of PR #496 (etcd health check, GetConfig RPC, TLS, OTEL toggles) remain out of scope for this quick task, as intended.

---
*Phase: quick-260902-eie*
*Completed: 2026-09-02*
