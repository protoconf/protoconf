---
phase: 8
slug: cli-flag-generation-config-loading
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-29
---

# Phase 8 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — standard Go test infrastructure |
| **Quick run command** | `go build ./...` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go build ./...`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 08-01-01 | 01 | 1 | PCLI-05 | build | `go build ./compiler/...` | TBD | ⬜ pending |
| 08-01-02 | 01 | 1 | PCLI-05 | build | `go build ./server/...` | TBD | ⬜ pending |
| 08-01-03 | 01 | 1 | PCLI-05 | build | `go build ./inserter/...` | TBD | ⬜ pending |
| 08-01-04 | 01 | 1 | PCLI-05 | build | `go build ./mutate/...` | TBD | ⬜ pending |
| 08-02-01 | 02 | 2 | PCLI-06 | integration | `go build ./cmd/protoconf/... && ./protoconf --help` | TBD | ⬜ pending |
| 08-02-02 | 02 | 2 | PCLI-07, PCLI-08, PCLI-09 | integration | `go test ./...` | TBD | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

*Existing infrastructure covers all phase requirements — Go build and test toolchain is standard.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| --help flag output matches current flags | PCLI-06 | Requires visual comparison | Run each component with --help, compare flag names with current output |
| Env var precedence over config file | PCLI-09 | Requires runtime environment setup | Set env var and config file with different values, verify env wins |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
