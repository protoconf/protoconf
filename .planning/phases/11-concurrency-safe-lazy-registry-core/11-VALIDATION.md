---
phase: "11"
slug: "concurrency-safe-lazy-registry-core"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: "2026-09-04"
---

# Phase 11 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go stdlib `testing` + `github.com/stretchr/testify` (already used project-wide) |
| **Config file** | none — `go test ./...` |
| **Quick run command** | `go test -race ./compiler/... ./server/... ./utils/...` |
| **Full suite command** | `go test -race -coverprofile=coverage.txt -covermode=atomic ./...` |
| **Estimated runtime** | ~90 seconds (quick ~25s) |

---

## Sampling Rate

- **After every task commit:** Run `go test -race ./compiler/... ./server/... ./utils/...`
- **After every plan wave:** Run `go test -race ./...`
- **Before `/gsd-verify-work`:** Full suite must be green **under `-race`** — this phase's whole
  point is concurrency correctness, so skipping `-race` at the gate defeats the phase
- **Max feedback latency:** 90 seconds

---

## Per-Task Verification Map

> Task IDs are filled in by the planner. This table records the requirement → test-type →
> command mapping the planner must honour; `❌ W0` marks tests that do not exist yet and must
> be created in Wave 0.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| TBD | TBD | TBD | LAZY-01 | — | N/A | unit/scaling | `go test ./compiler/lib/... -run TestCompilerStartupScaling` | ✅ (`compiler/lib/startup_bench_test.go:108`) | ⬜ pending |
| TBD | TBD | TBD | LAZY-02 | — | N/A | unit | `go test -race ./compiler/lib/parser/... -run TestParseMemoization` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | LAZY-02 (concurrency) | T-11-01 | Concurrent compiles never observe a torn/partially-written descriptor cache | race/integration | `go test -race ./compiler/lib/... -run TestConcurrentCompile` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | LAZY-03 | — | N/A | unit | `go test -race ./compiler/lib/parser/... -run TestLazyParseDoesNotMutateLocalFiles` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | LAZY-04 | — | N/A | regression | `go test ./compiler/lib/... -run TestModSyncFdsByteIdentical` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | LAZY-05 | — | N/A | unit | `go test ./compiler/lib/... -run TestLoadedFileCount` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | CONS-01 | — | Service catalog cannot silently empty; a missing service fails loudly at startup | integration | `go test ./server/... -run TestInitRegistersCustomService` | ❌ W0 (fixture `TestService` already exists) | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] A concurrent-compile race test — does not exist today; the highest-value new test in the
  set, since it is the only one that actually exercises the concurrency claim in the phase name.
  One shared `*lib.Compiler`, `errgroup` over N goroutines each calling `CompileFile` on a
  distinct generated config, run under `-race`.
- [ ] A `ParseFilesX`/`ParseOne` pointer-identity memoization test (LAZY-02) — `require.Same`
  on two requests for the same path proves no re-parse.
- [ ] A CONS-01 service-registration assertion on the existing `TestService` fixture
  (`utils/testdata/small/src/test.proto`) — no new `.proto` needed; assert
  `rpcServer.GetServiceInfo()` contains the service **without** calling `CompileFile` first.
- [ ] A `.fds` byte-identity regression check for `mod sync` (LAZY-04) — **verify no existing
  test already asserts on `.fds` file content before adding one**; existing `mod sync` CLI
  tests appear to assert only on file existence (unverified — check first).
- [ ] A lazy-parse non-mutation test for `localFiles` (LAZY-03).

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Operator-visible loaded-file count in real compiler output | LAZY-05 | The automated test asserts the counter/accessor value; whether the emitted log line is legible to a human operator is a judgement call | Run `protoconf compile <config>` against `utils/testdata/small` and confirm the output states how many proto files were loaded, and that the number is the config's transitive dep count, not `src/`'s total |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 90s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
