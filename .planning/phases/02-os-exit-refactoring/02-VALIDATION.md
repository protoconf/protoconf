---
phase: 2
slug: os-exit-refactoring
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-24
---

# Phase 2 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — standard Go test tooling |
| **Quick run command** | `go test ./compiler/lib/... ./mutate/...` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./compiler/lib/... ./mutate/...`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 02-01-01 | 01 | 1 | REFC-01 | build+test | `go build ./compiler/lib/... && go test ./compiler/lib/...` | ✅ | ⬜ pending |
| 02-01-02 | 01 | 1 | REFC-02 | build+test | `go build ./compiler/lib/... && go test ./compiler/lib/...` | ✅ | ⬜ pending |
| 02-01-03 | 01 | 1 | REFC-03 | build+test | `go build ./mutate/... && go test ./mutate/...` | ✅ | ⬜ pending |
| 02-01-04 | 01 | 1 | REFC-04 | build+test | `go build ./... && go test ./...` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements. Go's built-in `go test` and `go build` are sufficient to verify os.Exit removal and error propagation.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| CLI exit behavior unchanged | REFC-04 | End-to-end CLI behavior | Run `go run ./cmd/protoconf compile --help` and verify exit code 0; run with bad args and verify non-zero exit |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
