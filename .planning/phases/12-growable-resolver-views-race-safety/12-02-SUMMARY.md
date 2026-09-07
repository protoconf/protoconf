---
phase: 12-growable-resolver-views-race-safety
plan: 02
subsystem: compiler
tags: [protoregistry, descriptor-registry, canonical-pointer, tdd]

requires:
  - phase: 12-growable-resolver-views-race-safety
    provides: "Plan 12-01's growable filesResolver and locked FindFileByPath/RangeFiles accessors — this plan closes the pointer-identity hole growth makes reachable on ParseFilesX's resolver-hit branch"
provides:
  - "ParseFilesX returns the registry's canonical *desc.FileDescriptor for every file the registry already holds, on every lookup branch, while retaining the desc.WrapFile/desc.CreateFileDescriptor fallback for files genuinely absent from FileRegistry"
  - "A pointer-identity regression test (TestReReferencedProtoKeepsPointerIdentity) covering the A->B->A sequence, repeated-argument ordering, empty input, and unknown-path error"
  - "compiler/lib/config.go's config struct no longer carries the dead construction-time protoResolver snapshot field"
affects: [phase-13-symbol-index]

actuals:
  tokens: 1729
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "ParseFilesX's resolver-hit branch re-derives the canonical pointer via p.registry.FileDescriptor(resolved.Path()) before falling back to desc.WrapFile — the same canonical accessor ParseOne's own miss-branch already trusts, reused rather than duplicated"
    - "Dead resolver reference deleted outright (not rewired) when the underlying mechanism it referenced changes shape underneath it — a frozen snapshot with zero readers is a stale-resolution hazard waiting for a future phase to 'finish' it"

key-files:
  created:
    - compiler/lib/parser/canonical_identity_test.go
  modified:
    - compiler/lib/parser/parser.go
    - compiler/lib/config.go
    - compiler/lib/compiler.go

key-decisions:
  - "Canonical-first, wrap-as-fallback confirmed as correct per locked decision: server/server.go hand-registers six well-known files into the resolver that are deliberately absent from FileRegistry, so a canonical-only rewrite would turn a working mutation-server lookup into a hard error"
  - "The dead config.protoResolver field is deleted, not rewired to the growable TypeResolver — matches the plan's explicit prohibition; a future phase needing resolution inside config takes it from c.parser.TypeResolver directly"
  - "RegistryTypeResolver left byte-identical (D-01): this plan touches file resolution only, confirmed by diffing parser.go against the pre-task-1 commit"

requirements-completed: [RSLV-03]

coverage:
  - id: D1
    description: "Compiling a path that loads proto A, then proto B, then re-references A resolves all three correctly in one pass, and A's descriptor is the identical Go pointer both times it is returned (RSLV-03, ROADMAP criterion 1)"
    requirement: "RSLV-03"
    verification:
      - kind: unit
        ref: "compiler/lib/parser/canonical_identity_test.go#TestReReferencedProtoKeepsPointerIdentity"
        status: pass
    human_judgment: false
  - id: D2
    description: "Every descriptor ParseFilesX returns for a file the registry already holds is the registry's own canonical entry, regardless of which lookup branch served it"
    requirement: "RSLV-03"
    verification:
      - kind: unit
        ref: "compiler/lib/parser/canonical_identity_test.go#TestReReferencedProtoKeepsPointerIdentity"
        status: pass
    human_judgment: false
  - id: D3
    description: "A file genuinely external to the registry (mutation server's six hand-registered well-known files) still resolves through the pre-existing desc.WrapFile fallback"
    verification:
      - kind: unit
        ref: "go test -race ./server/... (TestDiscoveryScanDoesNotBackReflection)"
        status: pass
    human_judgment: false
  - id: D4
    description: "compiler/lib/config.go's config struct no longer carries the dead construction-time protoResolver snapshot field, and the compiler package builds clean with no unused import"
    verification:
      - kind: unit
        ref: "go build ./... && go vet ./compiler/... ./utils/..."
        status: pass
      - kind: unit
        ref: "grep -rln protoResolver --include=\"*.go\" . (no matches)"
        status: pass
    human_judgment: false
  - id: D5
    description: "Whole-module go test -race ./... stays green with the change in place (mirrors .github/workflows/go.yml, using the Phase-11-proven agent-package exclusion for the pre-existing, unrelated hang)"
    verification:
      - kind: unit
        ref: "go test -race -count=1 $(go list ./... | grep -v '/agent$')"
        status: pass
    human_judgment: false

duration: 28min
completed: 2026-09-08
status: complete
---

# Phase 12 Plan 02: ParseFilesX Canonical-Pointer Fix Summary

**`ParseFilesX`'s resolver-hit branch now returns the registry's own canonical `*desc.FileDescriptor` — the same pointer `dr.FileDescriptor(path)` returns — instead of minting a second descriptor via `desc.WrapFile`, closing the RSLV-03 hazard plan 12-01's growable resolver made reachable, and the dead `config.protoResolver` snapshot field is gone.**

## Performance

- **Duration:** 28 min
- **Started:** 2026-09-08T00:00:00Z (approx, sequential executor, no worktree)
- **Completed:** 2026-09-08T00:28:00Z (approx)
- **Tasks:** 3
- **Files modified:** 4 (1 created, 3 modified)

## Accomplishments
- `ParseFilesX`'s second lookup branch, on a `FindFileByPath` hit, now checks `p.registry.FileDescriptor(resolved.Path())` before falling back to `desc.WrapFile`/`desc.CreateFileDescriptor` — on a hit it returns the registry's own canonical pointer, closing the RSLV-03 non-canonical-pointer hazard.
- The `desc.WrapFile` fallback is preserved unchanged as the fourth/last-resort step, so the mutation server's six hand-registered well-known files (`server/server.go:292-297`, absent from `FileRegistry` by design) still resolve — verified by `go test -race ./server/...` including `TestDiscoveryScanDoesNotBackReflection`.
- New `TestReReferencedProtoKeepsPointerIdentity` pins the A->B->A sequence (ROADMAP criterion 1), asserts every returned descriptor is `require.Same` as the registry's own `FileDescriptor(path)` entry (the assertion that actually guards the contract, since branch 2 itself can't be forced from a black-box test), covers repeated-argument ordering, empty input, and an unknown-path error.
- `compiler/lib/config.go`'s dead `protoResolver protoregistry.MessageTypeResolver` field and its `c.parser.LocalResolver` assignment in `compiler.go`'s `load()` are deleted, along with the now-orphaned `protoregistry` import — a zero-reader field frozen at construction time, sitting in a struct next to a resolver mechanism this phase just made grow.
- `RegistryTypeResolver` is untouched (D-01 boundary held) — confirmed by diffing the final `parser.go` against the pre-Task-1 commit, which shows only the new canonical-lookup insertion and its comments.

## Task Commits

1. **Task 1: ParseFilesX returns the registry's canonical descriptor on a resolver hit** - `088f6fd` (feat)
2. **Task 2: RSLV-03 gate — load A, load B, re-reference A, one pointer for A** - `10f00a1` (test)
3. **Task 3: Delete the dead config.protoResolver field and its assignment** - `c2b855c` (refactor)

**Plan metadata:** (this commit)

## TDD Gate Compliance

| Task | RED | GREEN | REFACTOR | Status |
|------|-----|-------|----------|--------|
| Task 1 | n/a — see note below | `088f6fd` | none needed | Pass (with documented ordering deviation) |
| Task 2 | `10f00a1` (see note) | n/a — test-only task | n/a | Pass (with documented ordering deviation) |
| Task 3 | n/a — plain `type="auto"`, no `tdd="true"` | n/a | `c2b855c` (deletion, classified refactor) | Pass |

**Ordering deviation, explicitly authored into the plan (not an executor violation):** the plan sequences Task 1 (implementation) before Task 2 (the pointer-identity test), inverting the usual RED-before-GREEN commit order. The plan's own Task 2 `<action>` explicitly directs: "Write the test before verifying Task 1's change is complete, and record in the plan summary whether it was red against the pre-Task-1 dispatch. If it passes against the old code too, say so plainly ... rather than claiming a red-to-green transition."

Following that instruction, after writing `canonical_identity_test.go` I temporarily reverted `compiler/lib/parser/parser.go` to its pre-Task-1 content (the commit immediately before `088f6fd`) and re-ran `TestReReferencedProtoKeepsPointerIdentity`: **it passed against the old code too.** This confirms the plan's own prediction — `ParseFilesX`'s branch-2 hazard is near-unreachable from a black-box test once `FileRegistry` and the growable resolver are written at the same locked insert point (a resolver hit essentially always coincides with a `FileRegistry` hit on the compiler's lazy path), so the pre-existing eager-path test corpus never happens to exercise the wrap-vs-canonical distinction either way. The file was restored to the post-Task-1 state (verified via `git diff --stat` showing zero diff against `HEAD`) before Task 2 was committed. This test locks in the canonical-pointer contract prospectively — it is a regression guard, not a proof that Task 1 fixed a bug this specific test could previously observe failing.

## Files Created/Modified
- `compiler/lib/parser/parser.go` — `ParseFilesX`'s resolver-hit branch gains a `p.registry.FileDescriptor(resolvedFd.Path())` lookup with an early `continue`, inserted ahead of the `desc.WrapFile` fallback; no signature change, no import change
- `compiler/lib/parser/canonical_identity_test.go` — new, `TestReReferencedProtoKeepsPointerIdentity` (RSLV-03, ROADMAP criterion 1)
- `compiler/lib/config.go` — deleted `protoResolver protoregistry.MessageTypeResolver` field and the orphaned `protoregistry` import
- `compiler/lib/compiler.go` — deleted the `protoResolver: c.parser.LocalResolver` assignment in `load()`

## Decisions Made
- Canonical-first-then-wrap-fallback confirmed correct on evidence (`server/server.go:292-297`'s six hand-registered files, absent from `FileRegistry`): a canonical-only rewrite would have broken the mutation server's reflection/grpcui path.
- Dead `config.protoResolver` field deleted outright rather than rewired to the growable `TypeResolver` — per the plan's explicit prohibition; re-verified zero readers via `grep -rn '\.protoResolver\b'` immediately before deleting, since this plan's own Task 1/2 edits could in principle have introduced a new reader (they did not).
- `require.Same` used for every identity assertion in the new test (never `GetName()` alone, `proto.Equal`, or materialized-JSON comparison) since `desc.WrapFile`'s output is semantically equivalent to the canonical descriptor and would pass an output-only assertion while the contract stays broken.

## Deviations from Plan

### Auto-fixed Issues

None — no Rule 1-3 auto-fixes were needed; the plan's action steps mapped directly onto the code with no blocking issues, missing functionality, or bugs discovered outside the plan's own scope.

---

**Total deviations:** 0 auto-fixed. One plan-authored TDD ordering deviation, documented above under "TDD Gate Compliance" (not an executor-introduced deviation — the plan itself specifies this task order and explicitly anticipates the honest-reporting requirement).
**Impact on plan:** None. All three tasks executed exactly as specified.

## Issues Encountered

- The known pre-existing, unrelated `agent/command_test.go:118` hang (documented in Phase 11's SUMMARY/UAT and reconfirmed in 12-01's SUMMARY) still reproduces on an unscoped `go test -race ./...`. Used the Phase-11/12-01-proven workaround: `go test -race -count=1 $(go list ./... | grep -v '/agent$')`, which passed clean across all 25 testable packages including `agent/filekv`, `server`, `compiler/lib/parser`, and `utils`. Out of scope for this plan (no file this plan touches overlaps `agent/*_test.go`).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- RSLV-01, RSLV-02, RSLV-03, and SAFE-01 are now all closed for Phase 12: the growable `FilesResolver` (12-01) plus this plan's canonical-pointer fix and dead-field cleanup (12-02) satisfy every phase success criterion.
- Phase 13 (exact symbol index, nested-type/TYPE-01 resolution) can build on a `ParseFilesX` that never hands back a second Go pointer for a file the registry already holds — a precondition for any downstream map keyed by descriptor identity.
- The `agent` package's pre-existing `command_test.go:118` hang remains open across three phases now (11, 12-01, 12-02); still out of scope for a compiler-focused milestone, but the cost of re-confirming it each phase is starting to add up — worth a dedicated `/gsd-quick` or Phase 14+ cleanup item.

---
*Phase: 12-growable-resolver-views-race-safety*
*Completed: 2026-09-08*

## Self-Check: PASSED
- `compiler/lib/parser/canonical_identity_test.go` exists: FOUND
- `compiler/lib/parser/parser.go` modified (contains `p.registry.FileDescriptor(resolvedFd.Path())`): FOUND
- `compiler/lib/config.go` no longer contains `protoResolver`: CONFIRMED (`grep -rln protoResolver --include="*.go" .` returns no matches)
- `compiler/lib/config.go`'s `config` struct still declares `messageRegistry *msgregistry.MessageRegistry`: FOUND
- `compiler/lib/compiler.go`'s `load()` still assigns `messageRegistry: &c.ModuleService.GetProtoRegistry().MessageRegistry`: FOUND
- Commits `088f6fd`, `10f00a1`, `c2b855c` all present in `git log --oneline --all`: FOUND
- All plan-level `<verification>` commands re-run and green:
  - `go test -race ./compiler/lib/parser/... -run TestReReferencedProtoKeepsPointerIdentity -v`: PASS
  - `go test -race ./compiler/lib/parser/... -run TestParser_ParseFilesX -v`: PASS
  - `go test -race ./server/...` (incl. `TestDiscoveryScanDoesNotBackReflection`): PASS
  - `grep -rln protoResolver --include="*.go" .`: no matches
  - `go build ./...` and `go vet ./compiler/... ./utils/...`: exit 0
  - `go test -race -count=1 $(go list ./... | grep -v '/agent$')`: PASS (agent-package hang pre-existing, documented)
