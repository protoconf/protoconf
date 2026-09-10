---
phase: "14"
slug: "non-compiler-consumer-correctness"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: "2026-09-08"
---

# Phase 14 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Derived from `14-RESEARCH.md` § Validation Architecture.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go standard `testing` + `github.com/stretchr/testify` (assert/require) |
| **Config file** | none — `go test` via module, no framework config file |
| **Quick run command** | `go test -race ./<package>/... -run <TestName> -v` |
| **Full suite command** | `go test -race -count=1 ./...` (matches CI, `.github/workflows/go.yml:37`) |
| **Estimated runtime** | ~10s targeted · ~180s full suite under `-race` |

---

## Sampling Rate

- **After every task commit:** Run `go test -race ./<package>/... -run <TestName> -v`
- **After every plan wave:** Run `go test -race -count=1 ./...`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 180 seconds

---

## Per-Task Verification Map

Seeded per requirement; task IDs bind when PLAN.md files are created.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| TBD | TBD | TBD | CONS-02 | — | Inserter resolves an on-demand type correctly; no silent wrong answer | unit | `go test -race ./inserter/... -run TestProtoconfInserter_InsertConfig_AnyResolution -v` | ✅ existing (`inserter/inserter_test.go:80`) | ⬜ pending |
| TBD | TBD | TBD | CONS-03 | — | filekv serves a subscribed client a type resolved on demand | unit | `go test -race ./agent/filekv/... -run <new> -v` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | CONS-04 | — | `GenReflectionUI` reports rather than silently skips an unresolvable config | unit | `go test -race ./server/... -run <new> -v` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | SAFE-02 | — | Concurrent requests against a long-lived process complete without a race | integration (bufconn e2e) | `go test -race ./server/... ./agent/... -run <e2e> -v` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | SAFE-02 | — | Dedicated unpaced tight loop forces the interleaving an e2e test misses | unit | `go test -race -count=2 ./server/... ./agent/... -run <tightloop> -v` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | SAFE-03 | — | Escalation guard (D-08a): one unusual request never jumps to full repo count | unit | `go test -race ./utils/... -run <escalation-guard> -v` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | SAFE-03 | — | Sequence guard (D-08b): N configs over one process stay bounded, below corpus size | unit | `go test -race ./utils/... -run <sequence-guard> -v` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `agent/filekv/filekv_test.go` — a test resolving a type only reachable via the scan or index tier (not the global well-known seed, not construction-time), covering CONS-03. Existing `TestGet_ValidKey` (`agent/filekv/filekv_test.go:100`) does not exercise that path.
- [ ] `server/server_test.go` or a new `server/gen_reflection_ui_test.go` — a two-config fixture (one resolvable, one not) proving CONS-04's aggregate-and-continue behavior; extends `TestProtoconfMutationServer_GenReflectionUI` (`server/server_test.go:144`).
- [ ] `server/mutate_config_race_test.go` (new) and/or `agent/*_race_test.go` (new) — SAFE-02's bufconn e2e and dedicated tight-loop pair, per consumer (D-07).
- [ ] `utils/symbol_scan_test.go` or a new `utils/loaded_file_count_test.go` — SAFE-03's escalation guard (D-08a) and sequence guard (D-08b). The escalation fixture extends the existing `TestSymbolScanRespectsCandidateLimit` precedent (`utils/symbol_scan_test.go:91-103`), which already builds against `scanCandidateLimit = 32` (`utils/symbol_scan.go:41`).
- Framework install: **none required** — testify and errgroup are already direct/transitive dependencies (`go.mod`).

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Lock-removal sanity check for the D-07 tight-loop test | SAFE-02 | Proving the test *can* fail requires temporarily deleting the lock, which cannot ship as a committed test | Temporarily remove the resolver's lock, run the tight-loop test under `-race`, confirm `WARNING: DATA RACE` appears, restore the lock, confirm green |

All other phase behaviors have automated verification.

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 180s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
