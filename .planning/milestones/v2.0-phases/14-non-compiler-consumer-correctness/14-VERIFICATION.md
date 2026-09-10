---
phase: 14-non-compiler-consumer-correctness
verified: 2026-09-08T23:30:00Z
status: passed
score: 6/6 truths verified
behavior_unverified: 0
overrides_applied: 0
covered_files: [".planning/REQUIREMENTS.md", ".planning/phases/14-non-compiler-consumer-correctness/14-01-PLAN.md", ".planning/phases/14-non-compiler-consumer-correctness/14-01-SUMMARY.md", ".planning/phases/14-non-compiler-consumer-correctness/14-02-PLAN.md", ".planning/phases/14-non-compiler-consumer-correctness/14-02-SUMMARY.md", ".planning/phases/14-non-compiler-consumer-correctness/14-03-PLAN.md", ".planning/phases/14-non-compiler-consumer-correctness/14-03-SUMMARY.md", ".planning/phases/14-non-compiler-consumer-correctness/14-04-PLAN.md", ".planning/phases/14-non-compiler-consumer-correctness/14-04-SUMMARY.md", ".planning/phases/14-non-compiler-consumer-correctness/14-05-PLAN.md", ".planning/phases/14-non-compiler-consumer-correctness/14-05-SUMMARY.md", ".planning/phases/14-non-compiler-consumer-correctness/14-06-PLAN.md", ".planning/phases/14-non-compiler-consumer-correctness/14-06-SUMMARY.md", ".planning/phases/14-non-compiler-consumer-correctness/14-07-PLAN.md", ".planning/phases/14-non-compiler-consumer-correctness/14-07-SUMMARY.md", ".planning/phases/14-non-compiler-consumer-correctness/14-08-PLAN.md", ".planning/phases/14-non-compiler-consumer-correctness/14-08-SUMMARY.md", ".planning/phases/14-non-compiler-consumer-correctness/14-09-PLAN.md", ".planning/phases/14-non-compiler-consumer-correctness/14-09-SUMMARY.md", ".planning/phases/14-non-compiler-consumer-correctness/14-REVIEW.md", ".planning/phases/14-non-compiler-consumer-correctness/deferred-items.md", "agent/filekv/filekv.go", "agent/filekv/filekv_race_test.go", "agent/filekv/filekv_test.go", "agent/kv_agent_race_test.go", "compiler/lib/module_service.go", "compiler/lib/parser/loaded_file_count_test.go", "devserver/command.go", "inserter/inserter.go", "inserter/lazy_resolution_test.go", "mutate/mutate.go", "mutate/mutate_test.go", "server/gen_reflection_ui_test.go", "server/legacy.go", "server/mutate_config_path_test.go", "server/mutate_config_race_test.go", "server/server.go", "server/server_test.go"]
covered_digest: "v1:sha256:033866c0aa9290daeb1d78b92fb0075557ab1613a5f2a51e0f2016719494996d"
re_verification:
  previous_status: gaps_found
  previous_score: 5/6
  gaps_closed:
    - "agent/filekv's path-traversal guard now rejects a leading '..' key segment in BOTH Get and Watch via one shared resolveKeyPath helper, proven with tests independently reproduced failing against the unfixed code before the fix landed, and passing after."
  gaps_remaining: []
  regressions: []
advisory:
  - finding: "Two independently-maintained lexical containment checks now exist (filekv.resolveKeyPath and server.MutateConfig's inline check) with different normalization strictness for the same input class, in two different packages, with no shared helper."
    category: architectural
    reason: "Code review WR-01. Both checks correctly reject every real escape traced by the reviewer (no working bypass found); the concern is future drift if a third call site is hand-copied rather than routed through a shared helper. 14-09's plan explicitly scoped filekv's helper as package-private and declined to hoist it, given the two checks have different bases and failure shapes. Not a phase-goal violation on its own."
    evidence_status: "reviewer traced both checks by hand against multiple escape shapes; no bypass found. Not independently re-derived by this verification beyond confirming both checks exist and pass their own tests."
  - finding: "server.MutateConfig's write-path containment check has no in-source note acknowledging the same symlink-inside-root caveat filekv's resolveKeyPath documents."
    category: security
    reason: "Code review WR-02. The check is lexical (filepath.Rel, no EvalSymlinks) on both sides; filekv explicitly documents and accepts this ceiling with a ponytail: comment, MutateConfig's does not. Planting the symlink already requires write access to the config repo (a strictly larger compromise than either read or write escape this plan closes), so the risk class is the same one T-14-24 already accepted for filekv — just undocumented on the write side."
    evidence_status: "reviewer-identified; not independently re-derived here."
  - finding: "server.MutateConfig accepts an empty in.Path and silently writes a file literally named '..materialized_JSON' under mutable_config, rather than rejecting the degenerate input."
    category: other
    reason: "Code review WR-03. Independently reproduced the underlying mechanism: filepath.Clean(\"\") returns \".\", and filename := filepath.Join(base, filepath.Clean(in.Path)+ext) produces base/..materialized_JSON for an empty in.Path -- a real, if oddly-named, file INSIDE mutableConfigBase, not a traversal escape. This behavior predates 14-09 (the plan's fix adds only a containment check and explicitly declined to add a normalization/empty-path check, matching filekv's pre-14-09 scope boundary). Not a security escape; a validation-UX gap outside this plan's declared scope."
    evidence_status: "independently reproduced the filepath.Clean/Join mechanics in a scratch program; did not call MutateConfig directly to observe the write, but the mechanism is deterministic stdlib behavior."
gaps: []
deferred: []
---

# Phase 14: Non-Compiler Consumer Correctness Verification Report

**Phase Goal:** The mutation server, inserter, agent, and reflection UI all resolve types
correctly and safely now that the registry is no longer eager — no regression, no silent
wrong answers.
**Verified:** 2026-09-08T23:30:00Z
**Status:** passed
**Re-verification:** Yes — after gap closure (plan 14-09)

## Goal Achievement

### Observable Truths

| # | Truth (ROADMAP Success Criterion) | Status | Evidence |
|---|---|---|---|
| 1 | The inserter reads and inserts a materialized config whose type isn't already resolved, resolving it correctly. | ✓ VERIFIED | Regression check only (passed in prior verification, no files in scope for this gap closure): `inserter/inserter.go` still constructs via `lib.NewLazyModuleService`, `inserter/lazy_resolution_test.go`'s 3 named tests unaffected by 14-09. Confirmed present via grep; no re-run needed since inserter files are untouched by 14-09. |
| 2 | The agent's filekv store serves a subscribed client a config whose type is resolved on demand, correctly. | ✓ VERIFIED | Re-ran `TestGetResolvesTypeAbsentFromConstructionSnapshot`, `TestGetIsIdempotentForSameKey`, `TestGetUnresolvableTypeReturnsDiagnostic` alongside the full `./agent/filekv/...` package under `-race` — all pass; on-demand resolution behavior is unchanged by 14-09's guard addition (the guard runs before `Get`'s existing `ReadConfig`/`ReadConfig` call, not inside it). |
| 3 | `GenReflectionUI`'s periodic walk resolves every mutable config's type and reports — rather than silently skips — one it cannot resolve. | ✓ VERIFIED | Regression check only (`server/server.go`'s `collectExamples`/`GenReflectionUI` region is untouched by 14-09; 14-09's edit is scoped to `MutateConfig`, a different function). `grep -n "collectExamples\|reflectionFailure" server/server.go` still shows the aggregate-and-continue shape from the prior verification. |
| 4 | Concurrent requests against a long-lived mutation server or agent process complete without error under `go test -race`. | ✓ VERIFIED | Independently ran `go test -race -count=1 ./agent/...`, `go test -race -count=1 ./server/...`, and the full bounded `go test -race -count=1 -timeout 300s -skip 'Test_cliCommand_Run' ./...` — all green, zero `WARNING: DATA RACE`, 20/20 packages `ok`. 14-09's `resolveKeyPath` reads only `s.protoconfRoot` (immutable after `New`), introducing no new lock or shared state, confirmed by source read. |
| 5 | A long-running process handling many different configs over time keeps its loaded-file count proportional to what was actually demanded — it never jumps to the full repository count after one unusual request. | ✓ VERIFIED | Regression check only (`compiler/lib/parser/loaded_file_count_test.go` untouched by 14-09); prior verification's evidence stands, package passes in the bounded full-suite re-run above. |
| 6 | (Goal-level) "...resolve types correctly and **safely**... no regression, **no silent wrong answers**" — a caller-supplied key cannot escape the intended config tree, on either the read or write side. | ✓ VERIFIED | **This is the previously-failed truth; now closed.** `agent/filekv/filekv.go`'s `Get` and `Watch` both route through one new `resolveKeyPath` helper (confirmed: exactly 1 `filepath.Join(s.protoconfRoot` in the file, `resolveKeyPath` called 1x in `Get` before any filesystem access and 1x in `Watch` before `addWatch`). Independently reproduced RED evidence by checking out commit `6d35971` (before the `211a643` fix) into a scratch worktree and running the new tests: both `TestGetRejectsTraversalKey` and `TestWatchRejectsTraversalKey` FAIL against the unguarded code exactly as the SUMMARY claims. Ran the same tests against HEAD: both PASS. Independently reproduced the scope-widened `MutateConfig` write-path fix the same way: checked out `d841035` (before `042d031`), ran `TestMutateConfigRejectsTraversalPath` — it FAILS and the unguarded code writes a real file (`.../escaped.materialized_JSON`) outside `protoconfRoot`, confirming the SUMMARY's RED claim is genuine, not asserted. Ran the same test against HEAD: PASSES, and `TestMutateConfigAllowsNestedPath` (legitimate nested path) passes both before and after. `server/server.go`'s containment check for `MutateConfig` sits textually before the `protojson` marshal (line 531), before `runScript` (line 538), before `os.MkdirAll` (line 547), and before `os.WriteFile` (line 551) — read in full, confirmed. Both `14-02-PLAN.md`'s `T-14-03` row and `14-07-PLAN.md`'s `T-14-21` row no longer claim the risk was already mitigated; both now state the guard was insufficient and cite `14-09` as the actual fix location (confirmed by direct read of both files). `deferred-items.md` is confirmed unmodified by any of 14-09's 5 commits. |

**Score:** 6/6 truths verified — all 5 numbered ROADMAP criteria plus the phase goal's own "safely"/"no silent wrong answers" clause now pass on independently-reproduced evidence, not SUMMARY claims.

### Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `agent/filekv/filekv.go` | One shared validated key-to-path helper used by both `Get` and `Watch` | ✓ VERIFIED | `resolveKeyPath` declared once (lines 122-135), called once in `Get` (line 139) before any filesystem access, called once in `Watch` (line 179) before `addWatch` (line 185). Exactly 1 `filepath.Join(s.protoconfRoot` occurrence in the whole file (confirmed via grep). `ErrInvalidKey` sentinel declared via `errors.New`. `ponytail:` comment present on the containment step naming the symlink ceiling and `filepath.EvalSymlinks` upgrade path. |
| `agent/filekv/filekv_test.go` | Falsifiable traversal proofs for both `Get` and `Watch` | ✓ VERIFIED | `newTraversalFixture` plants a real, schema-valid file at `<base>/secret/leak.materialized_JSON`, outside the store's `root`. `TestGetRejectsTraversalKey` rewritten to assert both an error AND (guarded) absence of the secret payload in any returned pair. `TestWatchRejectsTraversalKey` and `TestGetMissingKeyIsNotAnInvalidKey` are new. Independently reproduced both traversal tests FAILING against the pre-fix commit and PASSING against HEAD. |
| `server/server.go` | Containment check on `MutateConfig`'s caller-supplied path, before every side effect | ✓ VERIFIED | `mutableConfigBase` computed (line 517), `filename` joined (518), containment check via `filepath.Rel` + `strings.HasPrefix` (526-528) — textually before the marshal (531), `runScript` (538), `MkdirAll` (547), `WriteFile` (551). Exactly 3 `filepath.Join(s.protoconfRoot` occurrences in the file (`srcPath`, `MutateConfig`'s base, `collectExamples`' root) — confirmed via grep, matching the plan's declared end-state. |
| `server/mutate_config_path_test.go` | Write-escape proof + legitimate nested-path regression | ✓ VERIFIED | New file. `TestMutateConfigRejectsTraversalPath` asserts file ABSENCE at the escape target via `errors.Is(statErr, os.ErrNotExist)`, not merely an error return — the exact defect class the prior gap named. `TestMutateConfigAllowsNestedPath` pins that interior separators still work. Independently reproduced the traversal test FAILING against the pre-fix commit (with a live outside-root write observed: `Written to filename=.../escaped.materialized_JSON`) and PASSING against HEAD. |
| `14-02-PLAN.md`, `14-07-PLAN.md` | Corrected threat-model rows, no longer claiming an absent mitigation | ✓ VERIFIED | `14-02-PLAN.md`'s `T-14-03` row: disposition `transferred → 14-09`, mitigation cell states the guard did not reject a leading `..` and the risk was live until 14-09. `<prohibition_breadcrumbs>` appends a line naming the false claim and its correction. `14-07-PLAN.md`'s `T-14-21` row: same correction pattern, disposition `transferred → 14-09`. Both confirmed via direct read, not grep alone. |
| `deferred-items.md` | Unmodified — nothing from 14-09 is deferred | ✓ VERIFIED | Confirmed via `git show --stat` on all 5 of 14-09's commits (`6d35971`, `211a643`, `d841035`, `042d031`, `ee398bb`): none touches `deferred-items.md`. |

### Key Link Verification

| From | To | Via | Status | Details |
|---|---|---|---|---|
| `agent/filekv/filekv.go Get` | `agent/filekv/filekv.go resolveKeyPath` | direct call, before filesystem access | ✓ WIRED | Line 139; on error, `Get` returns immediately (line 140-142), no `os.Stat`/`ReadConfig` reached |
| `agent/filekv/filekv.go Watch` | `agent/filekv/filekv.go resolveKeyPath` | direct call, before `addWatch` | ✓ WIRED | Line 179; on error, `Watch` returns immediately (line 180-182), `addWatch` (line 185) never reached |
| `server/server.go MutateConfig` | its own containment check | inline, before marshal/script/write | ✓ WIRED | Confirmed by line-order read: check (526-528) precedes marshal (531), `runScript` (538), `MkdirAll` (547), `WriteFile` (551) |
| End-state gate: exactly 1 `filepath.Join(s.protoconfRoot` in `filekv.go`, exactly 3 in `server.go` | — | repo-wide grep | ✓ PASS | Independently re-ran: `filekv.go` → 1 match; `server.go` → 3 matches (`srcPath`, `MutateConfig`'s base, `collectExamples`' root) |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|---|---|---|---|---|
| CONS-02 | 14-01, 14-05 | Inserter resolves types correctly | ✓ SATISFIED | Unaffected by 14-09 (regression-only recheck); prior verification's independent test runs stand |
| CONS-03 | 14-02, 14-05, 14-07, 14-09 | Agent's filekv store resolves types correctly AND safely | ✓ SATISFIED | Functional correctness from prior verification; safety gap now independently confirmed closed by 14-09 |
| CONS-04 | 14-04, 14-05 | GenReflectionUI reports rather than skips | ✓ SATISFIED | Unaffected by 14-09 (different function in same file); prior verification's evidence stands |
| SAFE-02 | 14-03, 14-06, 14-07 | Concurrent requests race-free under `-race` | ✓ SATISFIED | Re-ran `-race` suites for `./agent/...` and `./server/...` post-14-09 — both clean; `resolveKeyPath` and `MutateConfig`'s containment check introduce no new shared mutable state |
| SAFE-03 | 14-08 | Loaded-file count stays proportional | ✓ SATISFIED | Unaffected by 14-09; prior verification's evidence stands, package passes in the bounded full-suite re-run |

All 5 requirement IDs (CONS-02, CONS-03, CONS-04, SAFE-02, SAFE-03) declared across all 9 plans (14-01 through 14-09) are present in `.planning/REQUIREMENTS.md`, each marked `[x]`/`Complete`. No orphaned requirements found.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|---|---|---|---|---|
| `agent/filekv/filekv.go` | multiple (167, 178, 220, 228, 234, 240, 247, 253) | `// TODO implement me` on unrelated stub methods (`Delete`, `Exists`, `WatchTree`, `NewLock`, `List`, `DeleteTree`, `AtomicPut`, `AtomicDelete`) | ℹ️ Info | Pre-existing template scaffolding, untouched by 14-09 (verified none of these lines fall in 14-09's diff). Not a phase-goal blocker. |
| `agent/filekv/filekv.go` | 306-322 (`readEvents`) | Send-on-closed-channel ceiling (WR-01 from prior review) | ⚠️ Warning (pre-existing, out of 14-09's scope) | Carried forward unchanged; marked with its own `ponytail:` comment already. |
| `server/server.go` | 517-528 vs `agent/filekv/filekv.go` 122-135 | Two independently-maintained containment checks, different normalization strictness (14-REVIEW.md WR-01) | 📋 Advisory | Reviewer traced no working bypass in either. Deliberate scope decision in 14-09's plan (helper stays private to `filekv`). See `advisory` frontmatter. |
| `server/server.go` | 517-528 | No in-source symlink-risk note mirroring `filekv`'s (14-REVIEW.md WR-02) | 📋 Advisory | Same accepted-risk class as `filekv`'s documented T-14-24, just undocumented on the write side. See `advisory` frontmatter. |
| `server/server.go` | 518, 526-528 | Empty `in.Path` silently writes `..materialized_JSON` inside `mutableConfigBase` (14-REVIEW.md WR-03) | 📋 Advisory | Independently reproduced the `filepath.Clean("")`/`Join` mechanics. Contained (not an escape); pre-existing behavior 14-09 deliberately did not touch (plan explicitly scoped out adding a normalization check to `MutateConfig`). See `advisory` frontmatter. |

No `TBD`/`FIXME`/`XXX` debt markers found in any file touched by 14-09.

### Behavioral Spot-Checks / Test Execution

Independently ran (not trusting SUMMARY claims):
- `go build ./...` — exit 0.
- `go test -race -count=1 ./agent/filekv/... -run 'TestGetRejectsTraversalKey|TestWatchRejectsTraversalKey|TestGetMissingKeyIsNotAnInvalidKey' -v` — all PASS at HEAD.
- `go test -race -count=1 ./server/... -run 'TestMutateConfigRejectsTraversalPath|TestMutateConfigAllowsNestedPath|TestAuthFlow' -v` (`TestAuthFlow` run separately via `./test/...`, per plan) — all PASS at HEAD.
- **RED-evidence independent reproduction:** used `git worktree add --detach` to check out commit `6d35971` (Task 1's RED commit, before Task 2's fix `211a643`) into a scratch worktree, and ran the same two filekv tests — both genuinely FAIL (`An error is expected but got nil`), matching the SUMMARY's quoted `--- FAIL:` lines. Repeated for `d841035` (Task 3's RED commit, before Task 4's fix `042d031`) against `TestMutateConfigRejectsTraversalPath` — genuinely FAILS, and the unguarded code is observed writing a real file (`Written to filename=.../escaped.materialized_JSON`) outside `protoconfRoot`. Worktrees removed after use.
- `go test -race -count=1 -timeout 300s -skip 'Test_cliCommand_Run' ./...` — run twice independently, both exit 0, 20/20 packages `ok`, zero `WARNING: DATA RACE`.
- `go vet ./agent/filekv/... ./server/...` — silent.
- `git diff --exit-code go.mod go.sum` — clean, no dependency added.
- `git diff <pre-14-05>..HEAD --numstat` for `14-02-PLAN.md`/`14-07-PLAN.md` — 3+1 and 1+1 lines respectively, within the plan's own ≤6-line budget.
- `git show --stat` on all 5 of 14-09's commits confirms none touches `deferred-items.md`.

### Human Verification Required

None. The previously-recorded gap was deterministic and reproducible without human judgment, and the closure evidence was independently re-derived (not merely re-read from the SUMMARY) via scratch git worktrees that reproduce the exact FAIL/PASS transition claimed.

### Gaps Summary

The single gap recorded in the prior verification — `agent/filekv`'s `Get`/`Watch` guard failing to reject a leading `..` key segment, plus two threat-model rows falsely claiming that risk was mitigated — is closed. Verified independently, not from SUMMARY claims: rebuilding the pre-fix state in a scratch worktree reproduces the exact FAIL both for the filekv read-path test and for the scope-widened `MutateConfig` write-path test; the post-fix state passes both; the shared-helper structural requirement holds (exactly 1 join in `filekv.go`, exactly 3 in `server.go`, containment check placed before every side effect in `MutateConfig`); both threat-model rows now state the truth and cite the real fix location; `deferred-items.md` was correctly left untouched.

Three non-blocking findings from the code review (cross-package containment-check duplication, an undocumented symlink caveat on the write path, and an unrejected empty `in.Path`) are recorded as advisory items — none is a working bypass of the fix this phase's goal required, and the empty-path behavior predates 14-09 and was explicitly out of its declared scope. These are candidates for a future hardening pass, not blockers to Phase 14's goal.

---

_Verified: 2026-09-08T23:30:00Z_
_Verifier: Claude (gsd-verifier)_
