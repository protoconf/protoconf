---
phase: 7
slug: proto-defined-cli-configs
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-28
---

# Phase 7 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go generate + protoc + buf lint |
| **Config file** | buf.yaml, generate.go |
| **Quick run command** | `buf lint && go generate ./... && go build ./...` |
| **Full suite command** | `buf lint && go generate ./... && go build ./... && go test ./...` |
| **Estimated runtime** | ~45 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go build ./...`
- **After every plan wave:** Run `buf lint && go generate ./... && go build ./... && go test ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 45 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 07-01-01 | 01 | 1 | PCLI-01 | build | `protoc --go_out=. server/config/v1/server_config.proto && go build ./server/...` | ❌ W0 | ⬜ pending |
| 07-01-02 | 01 | 1 | PCLI-02 | build | `protoc --go_out=. compiler/config/v1/compiler_config.proto && go build ./compiler/...` | ❌ W0 | ⬜ pending |
| 07-01-03 | 01 | 1 | PCLI-03 | build | `protoc --go_out=. inserter/config/v1/inserter_config.proto && go build ./inserter/...` | ❌ W0 | ⬜ pending |
| 07-01-04 | 01 | 1 | PCLI-04 | build | `protoc --go_out=. mutate/config/v1/mutate_config.proto && go build ./mutate/...` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

*Existing infrastructure covers all phase requirements — protoc, buf, and go generate are already configured.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Proto field json_name matches current CLI flag | PCLI-01-04 | Semantic check | Compare each json_name against newFlagSet() flag names |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 45s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
