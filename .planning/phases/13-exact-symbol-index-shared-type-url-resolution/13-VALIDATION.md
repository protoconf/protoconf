---
phase: "13"
slug: "exact-symbol-index-shared-type-url-resolution"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: "2026-09-08"
---

# Phase 13 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Derived from `13-RESEARCH.md` § Validation Architecture.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go stdlib `testing` + `github.com/stretchr/testify` (project-wide convention) |
| **Config file** | none — `go test ./...` |
| **Quick run command** | `go test -race ./compiler/... ./utils/...` |
| **Full suite command** | `go test -race -coverprofile=coverage.txt -covermode=atomic ./...` (matches CI) |
| **Estimated runtime** | ~60–120 seconds (quick), ~5 min (full, `-race`) |

---

## Sampling Rate

- **After every task commit:** Run `go test -race ./compiler/... ./utils/...`
- **After every plan wave:** Run `go test -race ./...`
- **Before `/gsd-verify-work`:** Full suite green under `-race` — this phase
  touches lock-guarded shared state in two new code paths, so skipping `-race`
  at the gate defeats the concurrency contract
- **Max feedback latency:** 120 seconds

---

## Per-Task Verification Map

> Task IDs are assigned by the planner; this table maps requirements to their
> intended verification shape. The planner fills Task ID / Plan / Wave.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| TBD | TBD | TBD | TYPE-01 | — | N/A | unit | `go test -race ./compiler/... -run TestSymbolIndexNestedType -v` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | TYPE-02 | — | N/A | unit/timing | `go test -race ./compiler/... -run TestSymbolIndexBuildDoesNotLink -v` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | TYPE-03 | — | N/A | integration | `go test -race ./compiler/lib/parser/... -run TestReadConfigNestedAny -v` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | TYPE-04 | — | N/A | unit | `go test -race ./compiler/... -run TestIndexBuildLinksZeroFiles -v` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | TYPE-05 | — | N/A | unit | `go test -race ./compiler/... -run TestIndexCacheReusedOnUnchangedTree -v` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | TYPE-06 | — | N/A | unit | `go test -race ./compiler/... -run TestIndexCacheInvalidatedOnProtoEdit -v` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | TYPE-07 | — | N/A | unit | `go test -race ./compiler/... -run TestValueOnlyMutableNeverBuildsIndex -v` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | TYPE-08 | — | N/A | structural | `grep -rn "FindMessageTypeByUrl\|FindMessageByURL" --include=*.go compiler/` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | TYPE-09 | — | N/A | integration | shares TYPE-03's command | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | CONS-05 | — | N/A | integration | `go test -race ./compiler/lib/... -run TestLoadMutableResolvesNestedAny -v` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*
*Test names above are the researcher's intent, not a contract — the planner may rename; the requirement→behavior mapping is what binds.*

---

## Wave 0 Requirements

- [ ] A doubly-nested-symbol fixture proto (`Outer.Middle.Inner`) for TYPE-01's index-correctness test — does not exist today
- [ ] A materialized-JSON-with-nested-`Any` fixture for TYPE-03 / TYPE-09 / CONS-05 — `field_type_any_test.pconf` exercises only the compile-time `load()` path, not `ReadConfig` / `loadMutable`
- [ ] A replacement (not deletion) for `eager_fallback_visible_test.go`'s operator-observability contract — the new tiers need an equivalent "can an operator see what happened" test
- [ ] A lock-discipline race test for the new scan/index tiers, mirroring `utils/parse_all_deadlock_test.go`'s shape but racing the index build against `ParseOne`

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| One type-URL resolution implementation, not one per consumer | TYPE-08 | Structural/grep assertion over the codebase, not a runtime behavior a unit test can observe | `grep -rn "FindMessageTypeByUrl\|FindMessageByURL" --include=*.go compiler/` and confirm every compiler-side caller routes through the shared `TypeResolver`; full repo-wide grep-clean state is Phase 14's scope |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 120s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
