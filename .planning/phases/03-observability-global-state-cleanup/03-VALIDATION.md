---
phase: 3
slug: observability-global-state-cleanup
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-27
---

# Phase 3 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — standard Go testing |
| **Quick run command** | `go test ./...` |
| **Full suite command** | `go test -race -count=1 ./...` |
| **Estimated runtime** | ~60 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./...`
- **After every plan wave:** Run `go test -race -count=1 ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 60 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 03-01-01 | 01 | 1 | REFC-05 | build+test | `go build ./observability/... && go test ./server/... ./agent/...` | ❌ W0 | ⬜ pending |
| 03-01-02 | 01 | 1 | REFC-06 | build | `go build ./observability/...` | ❌ W0 | ⬜ pending |
| 03-02-01 | 02 | 1 | REFC-07 | build+test | `go test ./compiler/lib/...` | ✅ | ⬜ pending |
| 03-02-02 | 02 | 1 | REFC-08 | build+test | `go test ./mutate/...` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- Existing infrastructure covers all phase requirements. No new test framework or fixtures needed.
- Go test infrastructure is already in place across the project.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| OTel failure graceful degradation | REFC-06 | Requires unavailable OTel collector | Start server without OTel collector running, verify slog warning output and process continues |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
