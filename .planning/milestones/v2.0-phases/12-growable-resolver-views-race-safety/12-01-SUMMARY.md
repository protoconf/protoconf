---
phase: 12-growable-resolver-views-race-safety
plan: 01
subsystem: compiler
tags: [protoregistry, descriptor-registry, race-safety, tdd]

requires:
  - phase: 11-concurrency-safe-lazy-registry-core
    provides: "ParseOne, recordFileLocked, ParseAll, LoadedFileCount, RegistryTypeResolver — the locked single-writer DescriptorRegistry this plan grows a resolver view on top of"
provides:
  - "A growable, d.mu-guarded *protoregistry.Files on DescriptorRegistry that grows in place via RegisterFile instead of being rebuilt"
  - "Locked FindFileByPath/RangeFiles accessors, routed to from ParseFilesX's second branch"
  - "Two D-05 test-only observables: FilesResolverRegistrationCount, FilesResolverRegistrationErrorCount"
affects: [12-02, phase-13-symbol-index]

actuals:
  tokens: 4561
  tasks: 3
  commits: 5

tech-stack:
  added: []
  patterns:
    - "Growable *protoregistry.Files cached on first GetFilesResolver call for a lazy registry (ImportPaths non-empty), grown in place by registerFileLocked; eager registries keep the pre-existing fresh-build-per-call behavior byte-identical"
    - "Single insert point (registerFileLocked) called from both recordFileLocked (ParseOne path) and ParseAll's existing before/after diff loop — no second, independently-dedup'd registration path"
    - "Paired success/error counters (registrationCount, registrationErrors) as the RSLV-02 observable, so a duplicate-registration error that is only logged cannot mask a FileRegistry/filesResolver divergence"

key-files:
  created:
    - compiler/lib/parser/growable_resolver_test.go
    - utils/growable_resolver_test.go
  modified:
    - utils/utils.go
    - compiler/lib/parser/parser.go

key-decisions:
  - "D-01 hybrid confirmed as implemented: only FilesResolver grows by real RegisterFile calls; RegistryTypeResolver/LocalResolver untouched"
  - "No second lock object for the growable resolver — filesResolver/registrationCount/registrationErrors are new fields on the existing DescriptorRegistry, guarded by the pre-existing d.mu"
  - "registerFileLocked registers the whole transitive dependency closure, reusing recordFileLocked's existing recursion and idempotency guard, so FileRegistry and filesResolver can never diverge on the ParseOne path"
  - "ParseAll's eager-fallback files register through the SAME before/after diff loop that already writes lazyLoaded (one insert point, one key set) rather than a second pass over FileRegistry"

requirements-completed: [RSLV-01, RSLV-02, SAFE-01]

coverage:
  - id: D1
    description: "A proto parsed by ParseFilesX after NewParserWithDescriptorRegistry returned is findable through the parser's own FilesResolver field, re-read after the parse (RSLV-01)"
    requirement: "RSLV-01"
    verification:
      - kind: unit
        ref: "compiler/lib/parser/growable_resolver_test.go#TestParserFilesResolverGrowsAfterConstruction"
        status: pass
    human_judgment: false
  - id: D2
    description: "Registration is incremental, never a rebuild: cumulative registrations track cumulative distinct files exactly, the error tally stays zero, and the resolver pointer is never replaced (RSLV-02)"
    requirement: "RSLV-02"
    verification:
      - kind: unit
        ref: "utils/growable_resolver_test.go#TestRegistrationCountIsIncremental"
        status: pass
    human_judgment: false
  - id: D3
    description: "ParseAll's whole-tree eager-fallback files register into the same growable view through the same before/after diff that drives lazyLoaded — no divergence (D-02)"
    verification:
      - kind: unit
        ref: "utils/growable_resolver_test.go#TestParseAllRegistersIntoFilesResolver"
        status: pass
      - kind: unit
        ref: "utils/growable_resolver_test.go#TestFilesResolverRegistrationErrorsStayZero"
        status: pass
    human_judgment: false
  - id: D4
    description: "The only reader that could observe the growing resolver mid-write (ParseFilesX branch 2) is rerouted through the locked accessor; every eager consumer's resolver behavior stays byte-identical; the whole suite is race-clean with growth switched on (SAFE-01, this plan's share)"
    requirement: "SAFE-01"
    verification:
      - kind: unit
        ref: "go test -race ./compiler/... ./utils/..."
        status: pass
      - kind: unit
        ref: "go test -race ./server/... ./inserter/... ./mutate/... ./agent/filekv/..."
        status: pass
      - kind: unit
        ref: "go vet ./compiler/... ./utils/..."
        status: pass
    human_judgment: false

duration: 35min
completed: 2026-09-07
status: complete
---

# Phase 12 Plan 01: Growable FilesResolver Under d.mu Summary

**`DescriptorRegistry.filesResolver` now grows in place via `RegisterFile` — one call per newly recorded file — instead of the read-only, construction-time snapshot Phase 11 left behind, closing the RSLV-01 staleness gap without reintroducing the O(everything-loaded-so-far) rebuild this milestone deleted.**

## Performance

- **Duration:** 35 min
- **Started:** 2026-09-07T17:13:00Z
- **Completed:** 2026-09-07T17:48:46Z
- **Tasks:** 3
- **Files modified:** 4 (2 created, 2 modified)

## Accomplishments
- `p.FilesResolver` — the exact field a `Parser` consumer already holds — grows after construction: a file `ParseFilesX`/`ParseOne` parses on demand becomes findable through that same field object, never a replacement (RSLV-01).
- Growth is incremental and provably non-quadratic: `FilesResolverRegistrationCount()` tracks cumulative distinct files exactly, `FilesResolverRegistrationErrorCount()` stays 0, and `GetFilesResolver()` returns the identical pointer across every subsequent call (RSLV-02).
- `ParseAll`'s whole-tree eager fallback registers into the same growable view through the very loop that already computes `lazyLoaded`'s before/after diff — no second insert point, no chance of `FileRegistry`/`filesResolver` divergence (D-02, closes 12-RESEARCH.md Pitfall 3).
- `ParseFilesX`'s only unlocked reader of a growable resolver is gone: its second-branch lookup now goes through the new `d.mu`-guarded `FindFileByPath` accessor (T-12-01 mitigation).
- Every eager consumer (`server`, `inserter`, `mutate`, `agent/filekv`) keeps byte-identical `GetFilesResolver()` behavior — verified by the D-03 blast-radius test run, including the pre-existing `TestDiscoveryScanDoesNotBackReflection`.

## Task Commits

Each task followed RED → GREEN TDD discipline:

1. **Task 1: End-to-end — growable FilesResolver (RSLV-01)**
   - `6eff342` test(12-01): add failing test for growable FilesResolver (RSLV-01)
   - `09b8c22` feat(12-01): grow FilesResolver incrementally under d.mu (RSLV-01, RSLV-02)
2. **Task 2: ParseAll registers through the same diff, with an error tally (D-02)**
   - `3e87e8e` test(12-01): add failing tests for ParseAll registration + error tally (D-02)
   - `26aef7e` feat(12-01): register ParseAll's eager-fallback files into the same view (D-02)
3. **Task 3: RSLV-02 gate — the counter and the resolver identity, both measured**
   - `7363a94` test(12-01): pin RSLV-02 as a number, not a code-reading claim
   - No REFACTOR/feat commit: this task adds only a pinning test over behavior Tasks 1-2 already implemented; the test passed on first run with no source change (not a TDD violation — the task's own `<files>` list names only the test file).

**Plan metadata:** (this commit)

## TDD Gate Compliance

| Task | RED | GREEN | REFACTOR | Status |
|------|-----|-------|----------|--------|
| Task 1 | `6eff342` (compile failure on `utils.ErrNoGrowableResolver`/`FindFileByPath`, confirmed via `go test` before any source change) | `09b8c22` | none needed | Pass |
| Task 2 | `3e87e8e` (compile failure on the two D-05 accessor methods, confirmed via `go test` before any source change) | `26aef7e` | none needed | Pass |
| Task 3 | n/a — pinning test over already-shipped behavior, passed on first run (expected; see task note above) | n/a | n/a | Pass |

## Files Created/Modified
- `utils/utils.go` — new `ErrNoGrowableResolver` sentinel; new `DescriptorRegistry` fields `filesResolver`, `registrationCount`, `registrationErrors`; `GetFileDescriptorSet` now wraps new unexported `fileDescriptorSetLocked`; `GetFilesResolver` caches and grows one instance for lazy registries, unchanged fresh-build behavior for eager ones; new `registerFileLocked` (the single `RegisterFile` call site), wired into `recordFileLocked` and `ParseAll`'s existing diff loop; new locked accessors `FindFileByPath`, `RangeFiles`, `FilesResolverRegistrationCount`, `FilesResolverRegistrationErrorCount`
- `compiler/lib/parser/parser.go` — `ParseFilesX`'s second-branch lookup now reads through `p.registry.FindFileByPath` instead of the unlocked `p.FilesResolver` field
- `compiler/lib/parser/growable_resolver_test.go` — new, `TestParserFilesResolverGrowsAfterConstruction` (RSLV-01 tracer, D-02, D-03, empty/nil edges)
- `utils/growable_resolver_test.go` — new, `TestParseAllRegistersIntoFilesResolver`, `TestFilesResolverRegistrationErrorsStayZero`, `TestRegistrationCountIsIncremental`

## Decisions Made
- Confirmed D-01's hybrid split on code evidence: only `Files` grows; `RegistryTypeResolver`/`LocalResolver` stay the Phase 11 miss-fallthrough wrapper, untouched.
- No second lock: `filesResolver`/`registrationCount`/`registrationErrors` hang off the existing `d.mu`, matching the plan's explicit prohibition against a resolver-only lock.
- Registration granularity is the whole transitive closure, riding `recordFileLocked`'s existing recursion — free, and keeps the two sets from diverging the moment a transitively-loaded dependency is later named directly.
- The registration-error tally (research Open Question 1) is asserted to be 0 in every test path, not just logged — a duplicate-registration error is exactly the Pitfall-3 divergence signature.

## Deviations from Plan

None - plan executed exactly as written. `registrationErrors` (Task 2's stated deliverable) was added in Task 1's edit to `utils.go` alongside `registrationCount`, since `registerFileLocked`'s error branch needed somewhere to tally into from the moment it was written — a natural consequence of the plan's own step ordering, not a scope change; the field, its wiring, and its accessors are exactly what Task 2 specified.

## Issues Encountered

- `go test -race ./...` (the plan-level, `.github/workflows/go.yml`-mirroring gate) fails on the `agent` package with a 10-minute test-binary timeout — reproduced both in the full run and in an isolated `go test -race ./agent/...` run. This is a **pre-existing, unrelated hang** already documented in Phase 11's SUMMARY (`.planning/phases/11-concurrency-safe-lazy-registry-core/11-05-SUMMARY.md:156`) and UAT (`11-UAT.md:85`) at `agent/command_test.go:118`, confirmed here to still reproduce identically and confirmed unrelated by inspection: no file in `agent/*_test.go` references `DescriptorRegistry`, `FilesResolver`, or the `parser` package this plan touches. `go test -race -count=1 $(go list ./... | grep -v '/agent$')` — the same exclusion Phase 11 used — passes clean, and the plan's own narrower verification commands (`./compiler/lib/parser/...`, `./compiler/... ./utils/...`, `./server/... ./inserter/... ./mutate/... ./agent/filekv/...`, `go vet`) all pass exactly as specified. Out of scope for this plan per the scope-boundary rule (pre-existing, unrelated to files this plan touches).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- `12-02` can now build on a growable `FilesResolver` and the locked `FindFileByPath`/`RangeFiles` accessors to fix `ParseFilesX`'s remaining non-canonical-pointer hazard (RSLV-03) on the hit branch, per the plan's explicit note that this plan intentionally left that rewrite untouched.
- The `agent` package's pre-existing `command_test.go:118` hang remains open across two phases now; still out of scope for a compiler-focused milestone, but worth a dedicated `/gsd-quick` or Phase 14+ cleanup item since it now blocks a clean unscoped `go test -race ./...` run twice in a row.

---
*Phase: 12-growable-resolver-views-race-safety*
*Completed: 2026-09-07*

## Self-Check: PASSED
- `compiler/lib/parser/growable_resolver_test.go` exists: FOUND
- `utils/growable_resolver_test.go` exists: FOUND
- `utils/utils.go` modified (contains `ErrNoGrowableResolver`, `registerFileLocked`, `FindFileByPath`, `FilesResolverRegistrationCount`): FOUND
- `compiler/lib/parser/parser.go` modified (`ParseFilesX` calls `p.registry.FindFileByPath`): FOUND
- Commits `6eff342`, `09b8c22`, `3e87e8e`, `26aef7e`, `7363a94` all present in `git log --oneline --all`: FOUND
- All plan-level `<verification>` commands re-run and green except the pre-existing, documented `agent` package hang (out of scope, see Issues Encountered)
