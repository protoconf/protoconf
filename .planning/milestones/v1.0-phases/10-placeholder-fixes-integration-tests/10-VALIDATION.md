---
phase: 10
slug: placeholder-fixes-integration-tests
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-31
---

# Phase 10 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — existing infrastructure |
| **Quick run command** | `go test ./server/... ./compiler/lib/parser/... ./inserter/... ./agent/... ./test/...` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./server/... ./compiler/lib/parser/... ./inserter/... ./agent/... ./test/...`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 10-01-01 | 01 | 1 | TEST-07 | unit | `go test ./server/... -run TestMutateConfig` | ✅ | ⬜ pending |
| 10-01-02 | 01 | 1 | TEST-08 | unit | `go test ./compiler/lib/parser/... -run TestParser` | ✅ | ⬜ pending |
| 10-01-03 | 01 | 1 | TEST-09 | unit | `go test ./inserter/... -run TestInserter` | ✅ | ⬜ pending |
| 10-01-04 | 01 | 1 | TEST-10 | unit | `go test ./agent/... -run TestRollout` | ✅ | ⬜ pending |
| 10-02-01 | 02 | 2 | TEST-11 | e2e | `go test ./test/... -run TestMutationE2E` | ❌ W0 | ⬜ pending |
| 10-02-02 | 02 | 2 | TEST-12 | e2e | `go test ./test/... -run TestTLS` | ❌ W0 | ⬜ pending |
| 10-02-03 | 02 | 2 | TEST-13 | e2e | `go test ./test/... -run TestAuth` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- Existing infrastructure covers placeholder fixes (test files already exist)
- E2E tests extend existing `test/e2e_test.go` — no new framework needed
- TLS e2e requires `generateSelfSignedCert` helper (exists in `utils/tls_test.go`, may need extraction to testutil)

*Existing infrastructure covers most phase requirements.*

---

## Manual-Only Verifications

*All phase behaviors have automated verification.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
