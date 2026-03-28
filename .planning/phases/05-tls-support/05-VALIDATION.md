---
phase: 5
slug: tls-support
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-28
---

# Phase 5 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — standard Go testing |
| **Quick run command** | `go build ./... && go test ./utils/... ./agent/... ./server/... ./mutate/...` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~45 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go build ./...` (compile check)
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 45 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | Status |
|---------|------|------|-------------|-----------|-------------------|--------|
| 05-01-01 | 01 | 1 | shared | build+test | `go build ./utils/... && go test ./utils/...` | pending |
| 05-02-01 | 02 | 2 | SECR-01 | build+test | `go build ./agent/... && go test ./agent/...` | pending |
| 05-02-02 | 02 | 2 | SECR-01,SECR-03 | build+test | `go build ./server/... && go test ./server/...` | pending |
| 05-03-01 | 03 | 2 | SECR-02 | build+test | `go build ./mutate/... && go test ./mutate/...` | pending |

*Status: pending / green / red / flaky*

---

## Wave 0 Requirements

TLS testing requires self-signed certificates for integration tests. The TLS helper should include a test that exercises cert loading. No external test infrastructure needed — Go's `crypto/x509` can generate test certs in-memory.

---

## Manual-Only Verifications

- Verify that connecting with `grpcurl -insecure` to a TLS-enabled server works
- Verify that connecting without TLS to a TLS-enabled server is rejected

These are integration-level checks that may need manual or e2e testing (Phase 10 scope).

---

## Validation Sign-Off

- [ ] All tasks have automated verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 45s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
