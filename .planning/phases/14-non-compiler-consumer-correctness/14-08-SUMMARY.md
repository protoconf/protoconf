---
phase: 14-non-compiler-consumer-correctness
plan: 08
subsystem: testing
tags: [descriptor-registry, symbol-index, symbol-scan, lazy-loading, denial-of-service]

# Dependency graph
requires:
  - phase: 14-non-compiler-consumer-correctness
    provides: "14-01's verified lazy-registry construction pattern (utils.NewDescriptorRegistry with ImportPaths/CacheDir set, NewParserWithDescriptorRegistry) that this plan's fixtures reuse"
provides:
  - "compiler/lib/parser/loaded_file_count_test.go proving SAFE-03's escalation guard (D-08a) and sequence guard (D-08b), plus the adjacency/empty/ordering edges, all measured on the serving *utils.DescriptorRegistry handle"
affects: []

# Actuals (#2632)
actuals:
  tokens: 2361
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "Serving-registry-only measurement: every count assertion reads the *utils.DescriptorRegistry passed to NewParserWithDescriptorRegistry, captured as a local variable and never re-derived from a process-wide total or a retained discovery registry (D-06 scoping)"
    - "Baseline-before, delta-after: every test captures `before := d.LoadedFileCount()` immediately after construction rather than assuming zero, since well-known types are seeded into FileRegistry (not lazyLoaded) at construction"

key-files:
  created:
    - compiler/lib/parser/loaded_file_count_test.go
  modified: []

key-decisions:
  - "Treated as proof-only tests with no accompanying implementation change, matching 14-02's precedent: D-08's escalation and sequence guards are properties of already-shipped Phase 11-13 code (LoadSymbolByScan's candidate-limit bail, LoadSymbolByIndex's tiered escalation, recordFileLocked's closure recording), so both tasks' tests passed on first run and were committed as test(14-08) commits with no feat/refactor follow-up"
  - "TestDuplicateSymbolResolutionIsStable resolves through the scan tier (2 candidates, well under scanCandidateLimit=32) rather than forcing an index-tier escalation, since the plan's <behavior> section only requires resolution to be deterministic and stable across ten calls -- it does not require D-08(a)'s escalation fixture to be reused here, and sort.Strings' lexicographic candidate ordering gives the same kind of deterministic tie-break the index's own duplicate-symbol handling provides"

requirements-completed: [SAFE-03]

coverage:
  - id: D1
    description: "D-08(a) escalation guard: an index-tier-only resolution grows the loaded-file count by exactly the resolved file's closure, never by the number of scan candidates considered, and the index build itself registers nothing beyond that"
    requirement: "SAFE-03"
    verification:
      - kind: unit
        ref: "compiler/lib/parser/loaded_file_count_test.go#TestLoadedFileCountEscalationGuard"
        status: pass
      - kind: unit
        ref: "compiler/lib/parser/loaded_file_count_test.go#TestIndexBuildDoesNotRegisterTheTree"
        status: pass
    human_judgment: false
  - id: D2
    description: "SAFE-03 empty edge: an empty type URL and a name with no resolvable package-and-message shape both resolve nothing, cost zero, and never trigger a symbol-index build"
    requirement: "SAFE-03"
    verification:
      - kind: unit
        ref: "compiler/lib/parser/loaded_file_count_test.go#TestEmptyTypeURLResolvesNothingAndBuildsNoIndex"
        status: pass
    human_judgment: false
  - id: D3
    description: "SAFE-03 ordering edge: two files declaring the same fully-qualified symbol resolve deterministically and stably across repeated calls, growing the loaded-file count only once"
    requirement: "SAFE-03"
    verification:
      - kind: unit
        ref: "compiler/lib/parser/loaded_file_count_test.go#TestDuplicateSymbolResolutionIsStable"
        status: pass
    human_judgment: false
  - id: D4
    description: "D-08(b) sequence guard: eight distinct configs resolved in sequence against one long-lived registry grow the loaded set by a bounded per-symbol delta and end far below corpus size"
    requirement: "SAFE-03"
    verification:
      - kind: unit
        ref: "compiler/lib/parser/loaded_file_count_test.go#TestLoadedFileCountSequenceGuard"
        status: pass
    human_judgment: false
  - id: D5
    description: "SAFE-03 adjacency edge: repeated resolution of the same symbol, and two distinct symbols declared in the same file, both cost the registry only once"
    requirement: "SAFE-03"
    verification:
      - kind: unit
        ref: "compiler/lib/parser/loaded_file_count_test.go#TestRepeatedResolutionOfSameSymbolCostsNothingExtra"
        status: pass
      - kind: unit
        ref: "compiler/lib/parser/loaded_file_count_test.go#TestTwoSymbolsInOneFileShareOneLoad"
        status: pass
    human_judgment: false

duration: 15min
completed: 2026-09-08
status: complete
---

# Phase 14 Plan 08: SAFE-03 Loaded-File-Count Escalation and Sequence Guards Summary

**New `compiler/lib/parser/loaded_file_count_test.go` proves both SAFE-03 clauses -- a single unusual request never jumps to corpus size, and many requests over one long-lived registry stay proportional -- with all seven tests reading the exact serving `*utils.DescriptorRegistry` handle, never a process-wide total or the mutation server's retained discovery registry.**

## Performance

- **Duration:** ~15 min
- **Started:** 2026-09-08T12:53:00Z
- **Completed:** 2026-09-08T13:08:00Z
- **Tasks:** 2
- **Files modified:** 1 (new)

## Accomplishments
- `TestLoadedFileCountEscalationGuard`: a 33-file candidate-limit-busting fixture forces the scan tier to bail (`ScanResolutionCount() == 0`) and the index tier to answer (`IndexBuildCount() >= 1`); `LoadedFileCount()` grows by exactly one file's closure, not the 33 candidates considered
- `TestIndexBuildDoesNotRegisterTheTree`: the index build (parse-without-link over the whole root) registers nothing beyond the one resolved file -- `LoadedFileCount()` stays far below the actual `.proto` file count, computed in the test
- `TestEmptyTypeURLResolvesNothingAndBuildsNoIndex`: an empty type URL and a name with no resolvable package-and-message shape both fail fast at zero cost and never call `ensureSymbolIndex`
- `TestDuplicateSymbolResolutionIsStable`: two files declaring the same fully-qualified symbol resolve to a stable file path across ten calls, growing the loaded-file count only once
- `TestLoadedFileCountSequenceGuard`: resolving `corpus.pkg0`..`corpus.pkg7` in sequence against one long-lived registry asserts both clauses separately -- a per-resolution delta bounded by `{pkg0..pkgK}`'s closure size, and a final total strictly below the 60-file corpus
- `TestRepeatedResolutionOfSameSymbolCostsNothingExtra` and `TestTwoSymbolsInOneFileShareOneLoad`: repeat resolutions and same-file sibling messages both cost the registry nothing beyond the first load

## Task Commits

Each task was committed atomically:

1. **Task 1: D-08(a) escalation guard** - `280b1ee` (test)
2. **Task 2: D-08(b) sequence guard** - `541fa6f` (test)

**Plan metadata:** (this commit)

_Note: Both tasks carried `tdd="true"` but proved already-shipped Phase 11-13 behavior with no accompanying implementation change -- both test batches passed on first run, so each was committed as a single `test(14-08)` commit with no `feat`/`refactor` follow-up, matching 14-02's precedent._

## Files Created/Modified
- `compiler/lib/parser/loaded_file_count_test.go` - Seven tests proving SAFE-03's escalation guard, sequence guard, and adjacency/empty/ordering edges, all measured on the serving registry handle; three local fixture/helper functions (`writeLoadedFileCountProto`, `newLoadedFileCountRegistry`, `writeCandidateLimitFixture`)

## Decisions Made
- Verified via direct source reading (not assumption) that `splitSymbolPackage("")` and `splitSymbolPackage("all.lowercase.name")` both yield an empty `rest`, which is what makes the empty-edge test's zero-cost assertion correct rather than coincidental
- Confirmed `Msg5Part1`'s exact spelling against `utils/testdata/corpus.go`'s `protoFile` function before using it in `TestTwoSymbolsInOneFileShareOneLoad`, per the plan's explicit instruction not to trust the description
- Did not force `TestDuplicateSymbolResolutionIsStable` through the index tier (see key-decisions above) -- the test's actual requirement (stability across ten calls, one-time cost) holds regardless of which tier answers it, and reusing the 33-file escalation fixture here would have conflated two different fixture designs for no additional proof value

## Deviations from Plan

None - plan executed exactly as written. All seven tests passed on first run against the existing Phase 11-13 implementation; no bugs found, no missing functionality, no blocking issues.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- SAFE-03 (ROADMAP criterion 5) is now proven on the correct handle: a long-running process serving many different configs keeps its loaded-file count proportional to what was actually demanded, and a single unusual request never jumps to the full repository count
- `compiler/lib/parser` package fully covers D-08's two clauses plus the adjacency/empty/ordering edges named in this plan's `must_haves`
- No blockers for remaining Phase 14 plans (14-04 through 14-07, if not yet executed) or for phase-level verification

---
*Phase: 14-non-compiler-consumer-correctness*
*Completed: 2026-09-08*
