---
phase: 9
slug: unit-test-coverage-infrastructure
status: draft
nyquist_compliant: true
wave_0_complete: false
created: 2026-03-31
---

# Phase 9 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test with testify |
| **Config file** | none — standard Go test infrastructure |
| **Quick run command** | `go test ./testutil/... ./mutate/... ./fmt/... ./command/... ./devserver/... ./agent/dummykv/... ./agent/filekv/... ./agent/configmaps/... ./agent/otelkv/... ./compiler/starproto/...` |
| **Full suite command** | `go test -race ./...` |
| **Estimated runtime** | ~45 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test` on the specific package being tested
- **After every plan wave:** Run `go test -race ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 45 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 09-01-T1 | 01 | 1 | TEST-14 | unit | `go test ./testutil/...` | TBD | pending |
| 09-02-T1 | 02 | 2 | TEST-01 | unit | `go test ./mutate/...` | TBD | pending |
| 09-02-T2 | 02 | 2 | TEST-02 | unit | `go test ./fmt/...` | TBD | pending |
| 09-02-T3 | 02 | 2 | TEST-03 | unit | `go test ./command/...` | TBD | pending |
| 09-02-T4 | 02 | 2 | TEST-04 | unit | `go test ./devserver/...` | TBD | pending |
| 09-03-T1 | 03 | 2 | TEST-05 | unit | `go test ./agent/dummykv/... ./agent/filekv/... ./agent/configmaps/... ./agent/otelkv/...` | TBD | pending |
| 09-04-T1 | 04 | 2 | TEST-06 | unit | `go test ./compiler/starproto/...` | TBD | pending |

*Status: pending / green / red / flaky*

---

## Wave 0 Requirements

*Existing infrastructure covers all phase requirements — Go test toolchain and testify are standard.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| testutil used by 2+ test files | TEST-14 | Requires grep across imports | `grep -rl "testutil" *_test.go` returns 2+ files |
| Coverage reporting in CI | TEST-15 | Requires CI run inspection | Push branch, verify codecov report appears |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 45s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved
