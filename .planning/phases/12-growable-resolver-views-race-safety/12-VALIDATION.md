---
phase: "12"
slug: "growable-resolver-views-race-safety"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: "2026-09-07"
---

# Phase 12 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Derived from `12-RESEARCH.md` § Validation Architecture.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go stdlib `testing` + `github.com/stretchr/testify` (already used project-wide) |
| **Config file** | none — `go test ./...` |
| **Quick run command** | `go test -race ./compiler/... ./utils/...` |
| **Full suite command** | `go test -race -coverprofile=coverage.txt -covermode=atomic ./...` (matches `.github/workflows/go.yml`) |
| **Estimated runtime** | ~90 seconds (quick ~25s) |

---

## Sampling Rate

- **After every task commit:** Run `go test -race ./compiler/... ./utils/...`
- **After every plan wave:** Run `go test -race ./...`
- **Before `/gsd-verify-work`:** Full suite must be green **under `-race`** — this phase's whole
  point is concurrency correctness over a growable structure, so skipping `-race` at the gate
  defeats the phase
- **Max feedback latency:** 90 seconds

---

## Per-Task Verification Map

> Task IDs are filled in by the planner. This table records the requirement → test-type →
> command mapping the planner must honour; `❌ W0` marks tests that do not exist yet and must
> be created in Wave 0.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| TBD | TBD | TBD | RSLV-01 | — | N/A | unit | `go test -race ./utils/... ./compiler/lib/parser/... -run 'TestFilesResolverGrows' -v` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | RSLV-02 | — | N/A | unit | `go test -race ./utils/... -run 'TestRegistrationCountIsIncremental' -v` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | RSLV-03 | — | N/A | unit | `go test -race ./compiler/lib/... -run 'TestReReferencedProtoKeepsPointerIdentity' -v` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | SAFE-01 | — | N/A | race | `go test -race -count=2 ./compiler/lib/... -run TestConcurrentCompile -v` | ⚠️ extend existing (`compiler/lib/concurrent_compile_test.go:25`) | ⬜ pending |
| TBD | TBD | TBD | SAFE-01 | — | N/A | race | `go test -race -count=2 ./utils/... -run 'TestRegisterFileRacesRangeFiles' -v` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*
*Test names above are the planner's starting proposal, not a contract — the planner may rename
them, but every requirement row must keep a runnable `<automated>` command.*

---

## Wave 0 Requirements

- [ ] **RSLV-01** — a "resolver grows after construction" assertion. Does not exist today: Phase 11's
      tests assert on `FileRegistry` / `LoadedFileCount`, never on `FilesResolver` / `RangeFiles`.
- [ ] **RSLV-02** — registration-counter test (CONTEXT D-05). New accessor method and new test; both
      absent today.
- [ ] **RSLV-03** — pointer-identity re-reference test. `TestParser_ParseFilesX`
      (`compiler/lib/parser/parser_test.go:20`) exists but only exercises the eager
      (`ImportPaths`-unset) path with a single filename per case — never two `load()`s of the same
      file across a lazy registry.
- [ ] **SAFE-01 (a)** — resolver-view assertion inside `TestConcurrentCompile`
      (`compiler/lib/concurrent_compile_test.go:25`). The test exists and already races 8 goroutines
      against one shared `*lib.Compiler`, but nothing today touches `FilesResolver` / `RangeFiles`.
- [ ] **SAFE-01 (b)** — dedicated low-level `RegisterFile`-vs-`RangeFiles` race test in `utils`. Does
      not exist; needed to force the adversarial interleaving `-race` requires (research Open
      Question 2).

---

## Manual-Only Verifications

All phase behaviors have automated verification. This is an internal descriptor-registry change
with no new network input, no new auth surface, and no user-facing UI.

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 90s
- [ ] Every `-race` command in this phase is actually run with `-race` (not silently dropped)
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
