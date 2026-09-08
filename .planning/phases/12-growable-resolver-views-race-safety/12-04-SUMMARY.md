---
phase: 12-growable-resolver-views-race-safety
plan: 04
subsystem: compiler
tags: [parser, protoreflect, tdd, race-safety, gap-closure]

# Dependency graph
requires:
  - phase: 12-growable-resolver-views-race-safety plan 01
    provides: on-demand ParseFilesX / ParseOne / ErrNoGrowableResolver sentinel and D-03 eager boundary
  - phase: 12-growable-resolver-views-race-safety plan 02
    provides: ParseFilesX canonical-lookup branch (RSLV-03 pointer-identity fix)
provides:
  - ParseFilesX resolves a file present only in the parser's raw FilesResolver on an eager DescriptorRegistry
  - Regression test proving the resolution path goes through ParseFilesX, not FilesResolver.FindFileByPath directly
affects: [12-growable-resolver-views-race-safety, server-mutation-server]

actuals:
  tokens: 1300
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "ErrNoGrowableResolver as a signal, not a not-found: ParseFilesX retries the raw *protoregistry.Files field on that sentinel before falling into ParseOne"

key-files:
  created:
    - compiler/lib/parser/eager_resolver_fallback_test.go
  modified:
    - compiler/lib/parser/parser.go

key-decisions:
  - "Fallback lives in ParseFilesX, not in DescriptorRegistry.FindFileByPath — keeps FindFileByPath's ErrNoGrowableResolver contract intact for growable_resolver_test.go Test 3 and any other caller asserting ErrorIs against that sentinel."
  - "gsd-core's `check tdd-red-evidence` verb is Node-test-runner/TAP-specific (parses `# tests N` / `ok N - name` lines) and misclassifies a genuinely correct Go RED as INVALID_RED (zero_tests_discovered) — verified directly against this task's actual go test output. Treated as a Rule 3 tooling gap: the plan's own <verify> commands (grep for the exact `--- FAIL:` test line and the two joined sentinel strings) are the authoritative, project-appropriate RED gate here, and both passed against the unmodified parser.go before any fix was written."

requirements-completed: [RSLV-03]

coverage:
  - id: D1
    description: "A file present only in Parser.FilesResolver (hand-registered externally, mirroring server.go's six well-known files) and absent from FileRegistry now resolves through ParseFilesX on an eager registry."
    requirement: "RSLV-03"
    verification:
      - kind: unit
        ref: "compiler/lib/parser/eager_resolver_fallback_test.go#TestParseFilesXResolvesEagerHandRegisteredFile/the_failed_truth:_eager_hand-registered_file_resolves_through_ParseFilesX"
        status: pass
    human_judgment: false
  - id: D2
    description: "RSLV-03 adjacency: a path present in both FileRegistry and the resolver still returns the FileRegistry canonical Go pointer, never a second desc.WrapFile wrapper."
    requirement: "RSLV-03"
    verification:
      - kind: unit
        ref: "compiler/lib/parser/eager_resolver_fallback_test.go#TestParseFilesXResolvesEagerHandRegisteredFile/RSLV-03_adjacency"
        status: pass
      - kind: unit
        ref: "compiler/lib/parser/canonical_identity_test.go#TestReReferencedProtoKeepsPointerIdentity"
        status: pass
    human_judgment: false
  - id: D3
    description: "RSLV-03 ordering and empty/absent: a single ParseFilesX call mixing branches returns results in argument order; no-args returns empty+nil; empty string and absent paths return an ErrLazyParseDisabled-joined error without panicking."
    requirement: "RSLV-03"
    verification:
      - kind: unit
        ref: "compiler/lib/parser/eager_resolver_fallback_test.go#TestParseFilesXResolvesEagerHandRegisteredFile/RSLV-03_ordering"
        status: pass
      - kind: unit
        ref: "compiler/lib/parser/eager_resolver_fallback_test.go#TestParseFilesXResolvesEagerHandRegisteredFile/RSLV-03_empty/absent"
        status: pass
    human_judgment: false
  - id: D4
    description: "No regression: TestReReferencedProtoKeepsPointerIdentity, TestParserFilesResolverGrowsAfterConstruction, and the rest of the module's -race suite stay green after the fix."
    verification:
      - kind: unit
        ref: "go test -race -count=1 $(go list ./... | grep -v '/agent$')"
        status: pass
    human_judgment: false

duration: 25min
completed: 2026-09-08
status: complete
---

# Phase 12 Plan 04: Eager-registry ParseFilesX fallback Summary

**`ParseFilesX` now falls back to the raw `Parser.FilesResolver` snapshot when `FindFileByPath` returns `ErrNoGrowableResolver`, restoring resolution of hand-registered external files (e.g. the mutation server's six well-known protos) on eager `DescriptorRegistry` instances — closing the one failed must-have from `12-VERIFICATION.md`.**

## Performance

- **Duration:** 25 min
- **Completed:** 2026-09-08
- **Tasks:** 2
- **Files modified:** 2 (1 created, 1 modified)

## Accomplishments

- Reproduced the eager-registry break with a committed, red-to-green regression test (`TestParseFilesXResolvesEagerHandRegisteredFile`) that goes exclusively through `ParseFilesX`, never through the raw-field `FilesResolver.FindFileByPath` shortcut that made the prior citation (`TestDiscoveryScanDoesNotBackReflection`) worthless as evidence.
- Restored the `desc.WrapFile` tail's reachability for eager registries: `ParseFilesX` now interprets `utils.ErrNoGrowableResolver` as "this registry has no growable view, read the raw field" rather than "file not found," retrying the same lookup against `p.FilesResolver` before falling into `ParseOne`.
- Preserved every already-verified Phase 12 truth: `FindFileByPath`'s `ErrNoGrowableResolver` contract is unchanged (`growable_resolver_test.go` Test 3 still passes), the canonical-lookup branch (12-02's RSLV-03 pointer-identity fix) still runs unchanged and ahead of the wrap, and D-03 (eager registries keep zero growable state) is untouched.
- Confirmed the full module race gate (`go test -race -count=1 $(go list ./... | grep -v '/agent$')`) stays green.

## Task Commits

Each task was committed atomically:

1. **Task 1: Reproduce the eager-registry break end-to-end** - `3401e6c` (test)
2. **Task 2: ParseFilesX falls back to the raw FilesResolver on ErrNoGrowableResolver** - `f57d3fd` (fix)

**Plan metadata:** committed separately after this SUMMARY.

_TDD gate: RED committed at `3401e6c` (confirmed failing with `registry has no growable files resolver` joined with `on-demand parsing requires import paths`, via the plan's own `<verify>` grep commands), GREEN committed at `f57d3fd`. No REFACTOR commit needed — the fix is a single, already-minimal 11-line insertion._

## Files Created/Modified
- `compiler/lib/parser/eager_resolver_fallback_test.go` - New regression test, `TestParseFilesXResolvesEagerHandRegisteredFile`, proving the eager-registry fallback through `ParseFilesX` (failed truth, RSLV-03 adjacency, ordering, empty/absent sub-cases)
- `compiler/lib/parser/parser.go` - `ParseFilesX` gains a four-line `errors.Is(resolverErr, utils.ErrNoGrowableResolver)` branch (plus a comment) immediately after the `FindFileByPath` call, retrying against `p.FilesResolver` before falling into `ParseOne`

## Decisions Made

- **Fallback placed in `ParseFilesX`, not `DescriptorRegistry.FindFileByPath`.** `FindFileByPath`'s `ErrNoGrowableResolver` contract is asserted directly by `growable_resolver_test.go` Test 3 and other callers; `ParseFilesX` is the one call site holding a raw field worth falling back to, so it is the one place that interprets the sentinel. No change to `utils/utils.go`.
- **`gsd_run check tdd-red-evidence` does not apply to this project.** That verb's TAP parser (`parseNodeTestSummary`/`tapFailedTestNames`) expects Node's `--test` runner output (`# tests N`, `ok N - name` lines) and has no Go-test-format support. Run directly against this task's actual `go test -v` RED output, it returned `INVALID_RED (zero_tests_discovered)` even though the RED was genuine (the plan's own `<verify>` commands — grepping for the exact `--- FAIL: TestParseFilesXResolvesEagerHandRegisteredFile` line and both joined sentinel strings — passed cleanly against the unmodified code). Treated as a Rule 3 tooling gap rather than a blocker: the plan's explicit, project-appropriate `<verify>` commands are the RED gate of record here, and they are satisfied and reproducible from git history (commit `3401e6c` predates the fix in `f57d3fd`).

## Deviations from Plan

None - plan executed exactly as written. (See "Decisions Made" above for the one tooling-applicability note, which changed no code or test behavior and required no auto-fix.)

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `12-VERIFICATION.md`'s failed truth is now true, `missing:` items 1 and 2 are closed, and the miscitation is not inherited — every evidence command in this plan invokes `ParseFilesX` directly.
- Truths 1-9 of `12-VERIFICATION.md` remain verified; D-03 is untouched; WR-01 (public `Parser.FilesResolver`) and WR-02 (unlocked `FileRegistry` mutators) remain explicitly out of scope, unchanged.
- Phase 12 gap closure is complete; ready for `/gsd-verify-work 12` re-verification.

---
*Phase: 12-growable-resolver-views-race-safety*
*Completed: 2026-09-08*

## Self-Check: PASSED

- FOUND: `compiler/lib/parser/eager_resolver_fallback_test.go`
- FOUND: commit `3401e6c` (RED)
- FOUND: commit `f57d3fd` (GREEN)
- `plan_head_before: b6188f6907208ee2d9f53db6715df82b6d1a353b`, `commits: 2` (measured via `git rev-list --count`)
