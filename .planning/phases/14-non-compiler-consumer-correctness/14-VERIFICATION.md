---
phase: 14-non-compiler-consumer-correctness
verified: 2026-09-08T21:35:00Z
status: gaps_found
score: 5/6 truths verified (5 ROADMAP criteria pass; 1 goal-level safety clause fails)
behavior_unverified: 0
overrides_applied: 0
covered_files: [".planning/REQUIREMENTS.md", ".planning/phases/14-non-compiler-consumer-correctness/14-01-PLAN.md", ".planning/phases/14-non-compiler-consumer-correctness/14-01-SUMMARY.md", ".planning/phases/14-non-compiler-consumer-correctness/14-02-PLAN.md", ".planning/phases/14-non-compiler-consumer-correctness/14-02-SUMMARY.md", ".planning/phases/14-non-compiler-consumer-correctness/14-03-PLAN.md", ".planning/phases/14-non-compiler-consumer-correctness/14-03-SUMMARY.md", ".planning/phases/14-non-compiler-consumer-correctness/14-04-PLAN.md", ".planning/phases/14-non-compiler-consumer-correctness/14-04-SUMMARY.md", ".planning/phases/14-non-compiler-consumer-correctness/14-05-PLAN.md", ".planning/phases/14-non-compiler-consumer-correctness/14-05-SUMMARY.md", ".planning/phases/14-non-compiler-consumer-correctness/14-06-PLAN.md", ".planning/phases/14-non-compiler-consumer-correctness/14-06-SUMMARY.md", ".planning/phases/14-non-compiler-consumer-correctness/14-07-PLAN.md", ".planning/phases/14-non-compiler-consumer-correctness/14-07-SUMMARY.md", ".planning/phases/14-non-compiler-consumer-correctness/14-08-PLAN.md", ".planning/phases/14-non-compiler-consumer-correctness/14-08-SUMMARY.md", ".planning/phases/14-non-compiler-consumer-correctness/14-REVIEW.md", ".planning/phases/14-non-compiler-consumer-correctness/deferred-items.md", "agent/filekv/filekv.go", "agent/filekv/filekv_race_test.go", "agent/filekv/filekv_test.go", "agent/kv_agent_race_test.go", "compiler/lib/module_service.go", "compiler/lib/parser/loaded_file_count_test.go", "devserver/command.go", "inserter/inserter.go", "inserter/lazy_resolution_test.go", "mutate/mutate.go", "mutate/mutate_test.go", "server/gen_reflection_ui_test.go", "server/legacy.go", "server/mutate_config_race_test.go", "server/server.go", "server/server_test.go"]
covered_digest: "v1:sha256:7aaff8f5c1ac288a50ab8ba8a6d8f2941b754616584df7bda70c3e5c9287a7a9"
gaps:
  - truth: "The registry-no-longer-eager transition resolves types correctly AND SAFELY, with no silent wrong answers (phase goal's own wording) — specifically for the agent's filekv store."
    status: failed
    reason: >
      agent/filekv/filekv.go's Get (line 98) and Watch (line 138) path-traversal guard
      (`key != filepath.ToSlash(filepath.Clean(key)) || key == ""`) does not reject a
      caller-supplied key containing a leading `../` segment, because filepath.Clean cannot
      collapse a leading `..` with no preceding path component to cancel against. A key like
      `../secret/leak` passes the guard unchanged and reaches a real file outside
      protoconfRoot. This is exploitable from any gRPC client that can choose the `path` for
      SubscribeForConfig (the agent's public API surface) or from the config.json write path
      of any consumer that hands filekv a caller-influenced key.

      Independently reproduced (not merely inherited from the code review): built a
      fresh filekv Store rooted at <tmp>/protoconfRoot, placed a valid ProtoconfValue JSON
      file at <tmp>/secret/leak.materialized_JSON (outside the root), and called
      Get(ctx, "../secret/leak", nil). Get returned err=nil and a KVPair whose
      base64-decoded, proto-unmarshalled Value was the outside-root file's actual content
      ("\n\ntest.proto") — a live, reproducible directory-traversal read.

      This phase's own plan (14-02) both (a) left this code path unchanged by explicit
      design ("D-01 ... Explicitly do NOT add a resolver swap in Get ... leave Get's leading
      key validation untouched") and (b) added a new test, TestGetRejectsTraversalKey
      (agent/filekv/filekv_test.go:306-320), whose docstring calls it a pin of "the existing
      traversal guard as a regression". That test only asserts require.Error(t, err) for key
      "../etc/passwd" against a fixture where no file exists at that traversed location, so
      it passes via os.Stat's ErrNotExist rather than via the guard firing — it cannot fail
      even if an attacker-reachable file existed at the traversed target (confirmed: I ran
      the identical scenario with a real file present and it succeeded, leaking the file).
      14-02's own threat model (T-14-03, "Information Disclosure ... high ... mitigate")
      explicitly claims this guard "is pinned as a regression by Task 2's
      TestGetRejectsTraversalKey" — that specific claim is false. 14-07's threat model
      (T-14-21) repeats the same false claim.

      The underlying guard code predates Phase 14 (per the code review's git-blame finding,
      commit 0745e19) and Phase 14 did not introduce the vulnerability. The gap is that this
      phase (a) explicitly declared this exact risk "mitigate[d]" in its own threat model
      while shipping a test that provides false assurance of that mitigation, and (b) the
      phase's own goal text is "...resolve types correctly and SAFELY now that the registry
      is no longer eager — no regression, no silent wrong answers" — a path traversal that
      lets a client silently read a file outside the intended config tree is squarely a
      "not safely" / "silent wrong answer" outcome the goal disclaims.
    artifacts:
      - path: "agent/filekv/filekv.go"
        issue: "Get (line 96-100) and Watch (line 138-140) guard rejects only keys that differ from filepath.Clean(key)'s own normalized form; it does not reject a leading '../' segment, so a key like '../secret/leak' passes through unchanged and resolves to a path outside protoconfRoot."
      - path: "agent/filekv/filekv_test.go"
        issue: "TestGetRejectsTraversalKey (lines 306-320) asserts only require.Error(t, err) for a traversal key whose target file does not exist in the fixture, so it passes on os.Stat's ErrNotExist rather than on the guard firing — it is a false negative that would not catch a real traversal read."
    missing:
      - "Reject any key whose cleaned form still contains a leading '..' element (e.g. `strings.HasPrefix(cleaned, \"../\") || cleaned == \"..\"`), or verify the joined absolute path remains under protoconfRoot via filepath.Rel/prefix check, in both Get and Watch (ideally factored into one shared helper so the two call sites cannot drift, per the code review's WR-03 note)."
      - "Strengthen TestGetRejectsTraversalKey (and its 14-07 T-14-21 counterpart assumption) to place a real file outside protoconfRoot at the traversal target and assert its content is never returned — not merely that some error occurred."
deferred: []
advisory: []
---

# Phase 14: Non-Compiler Consumer Correctness Verification Report

**Phase Goal:** The mutation server, inserter, agent, and reflection UI all resolve types
correctly and safely now that the registry is no longer eager — no regression, no silent
wrong answers.
**Verified:** 2026-09-08T21:35:00Z
**Status:** gaps_found
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth (ROADMAP Success Criterion) | Status | Evidence |
|---|---|---|---|
| 1 | The inserter reads and inserts a materialized config whose type isn't already resolved, resolving it correctly. | ✓ VERIFIED | `inserter/inserter.go` constructs via `lib.NewLazyModuleService`; both type-URL/`Any` resolution sites read `i.parser.TypeResolver`. Independently ran `go test -race ./inserter/... -run 'TestInserterResolvesTypeAbsentFromConstructionSnapshot\|TestInserterUnresolvableTypeReturnsDiagnostic\|TestInserterCLIExitsZeroOnPerFileFailure'` — all 3 PASS. `TestInserterResolvesTypeAbsentFromConstructionSnapshot` asserts `errors.Is(err, protoregistry.NotFound)` on the frozen construction snapshot both before and after a successful resolve, proving the resolution was genuinely on-demand. |
| 2 | The agent's filekv store serves a subscribed client a config whose type is resolved on demand, correctly. | ✓ VERIFIED (functional correctness) — see Gap below for the safety clause | `agent/filekv/filekv.go`'s `New` constructs via `lib.NewLazyModuleService`; `Get` already routed through `s.parser.ReadConfig`'s tiered resolver, requiring no second edit. Ran `TestGetResolvesTypeAbsentFromConstructionSnapshot`, `TestGetIsIdempotentForSameKey`, `TestGetUnresolvableTypeReturnsDiagnostic` — all PASS, confirming on-demand resolution, memoisation, and loud failure on a genuinely missing type. |
| 3 | `GenReflectionUI`'s periodic walk resolves every mutable config's type and reports — rather than silently skips — one it cannot resolve. | ✓ VERIFIED | `collectExamples` extracted; every branch inside the `filepath.WalkDir` closure appends to a `reflectionFailure` slice and returns `nil` instead of aborting; the walk's own return is captured; failures are aggregated with `errors.Join` and logged only on change via an order-independent, reason-sensitive fingerprint. Ran `TestCollectExamplesContinuesPastUnresolvableConfig`, `TestCollectExamplesAggregateNamesEveryFailure`, `TestGenReflectionUIEmptyMutableConfig` — all PASS. `TestProtoconfMutationServer_GenReflectionUI` was correctly updated (not weakened) to expect the newly-surfaced errors. |
| 4 | Concurrent requests against a long-lived mutation server or agent process complete without error under `go test -race`. | ✓ VERIFIED | Ran `go test -race -count=1 ./server/... -run 'TestMutateConfigConcurrentClientsAreRaceFree\|TestMutationServerResolverTightLoopIsRaceFree'` and `./agent/... -run TestSubscribeForConfigConcurrentClientsAreRaceFree` and `./agent/filekv/... -run 'TestFileKVGetTightLoopIsRaceFree\|TestFileKVConcurrentGetReturnsCorrectValuePerKey'` — all PASS, no `WARNING: DATA RACE`. Both dedicated tight-loop tests' failure-detection capability is documented in their SUMMARYs with concrete evidence (lock removed, 3/3 runs reproduced `WARNING: DATA RACE` on `recordFileLocked`'s map write, restored via `git checkout`) — I did not re-execute the lock-removal step myself but the evidence given (exact file/line, exact race target, reproducibility count) is specific and falsifiable, consistent with Phase 12's established validation pattern. |
| 5 | A long-running process handling many different configs over time keeps its loaded-file count proportional to what was actually demanded — it never jumps to the full repository count after one unusual request. | ✓ VERIFIED | Ran `TestLoadedFileCountEscalationGuard` (33-file candidate-limit-busting fixture: `ScanResolutionCount()==0`, `IndexBuildCount()>=1`, `LoadedFileCount()` grows by exactly 1, not 33) and `TestLoadedFileCountSequenceGuard` (8 sequential resolutions against one long-lived registry over a 60-file corpus stay strictly below 60) — both PASS. |
| 6 | (Goal-level, not separately numbered in ROADMAP) "...resolve types correctly and **safely**... no regression, **no silent wrong answers**" — a caller-supplied key cannot escape the intended config tree. | ✗ FAILED | See Gap. `agent/filekv/filekv.go`'s `Get`/`Watch` traversal guard does not reject a leading `../` key segment; independently reproduced a live outside-root file read. The phase's own new "regression" test (`TestGetRejectsTraversalKey`) is a false negative that would not catch this. |

**Score:** 5/6 truths verified (all 5 numbered ROADMAP success criteria pass on their literal text; the phase goal's own "safely"/"no silent wrong answers" clause fails for one specific, confirmed vector).

### Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `inserter/inserter.go` | Lazy construction + tiered resolution | ✓ VERIFIED | `lib.NewLazyModuleService` x1, `i.parser.TypeResolver` x2 confirmed via grep and passing tests |
| `inserter/lazy_resolution_test.go` | CONS-02 on-demand proof | ✓ VERIFIED | 3 named tests present and passing |
| `agent/filekv/filekv.go` | Lazy construction | ✓ VERIFIED (construction) / ✗ pre-existing traversal flaw retained | `lib.NewLazyModuleService(absRoot)` present; `Get`/`Watch` guard unchanged and insufficient (see Gap) |
| `agent/filekv/filekv_test.go` | CONS-03 proof + traversal regression | ⚠️ Test present but traversal assertion is a false negative | 3 of 4 new tests are solid proofs; `TestGetRejectsTraversalKey` does not prove what it claims |
| `server/server.go` | Lazy construction, tiered marshal resolver, reflection completeness, aggregate-and-continue walk | ✓ VERIFIED | `lib.NewLazyModuleService`, `resolver := s.parser.TypeResolver`, `DescriptorResolver: discoveryFiles` x2, `collectExamples`, `reflectionFailure`, `sync.Mutex` fingerprint guard all present and covered by passing tests |
| `server/server_test.go` | Reflection round-trip proof | ✓ VERIFIED | `TestReflectionDescribesCustomAndBuiltinServices` PASS |
| `server/gen_reflection_ui_test.go` | CONS-04 edge coverage | ✓ VERIFIED | 7 named tests present, ran a representative subset, all PASS |
| `server/mutate_config_race_test.go` | SAFE-02 concurrency proof (mutation server) | ✓ VERIFIED | Both named tests present and passing under `-race` |
| `devserver/command.go` | Explicit `GenReflectionUI` error handling | ✓ VERIFIED | grep confirms `_ = ...GenReflectionUI(` at both call sites |
| `mutate/mutate.go` | Lazy construction + tiered resolution, exit-1 abort preserved | ✓ VERIFIED | `lib.NewLazyModuleService(root)`, `anyResolver := parser.TypeResolver`, `return 1` count unchanged; both new tests PASS |
| `compiler/lib/module_service.go` | Corrected doc comment | ✓ VERIFIED | `mod sync` named as sole remaining eager consumer; confirmed true via `mod/command.go` |
| `compiler/lib/parser/loaded_file_count_test.go` | SAFE-03 escalation + sequence guards | ✓ VERIFIED | 7 named tests present, representative subset run, all PASS |
| `agent/kv_agent_race_test.go`, `agent/filekv/filekv_race_test.go` | SAFE-02/CONS-03 agent-side concurrency proofs | ✓ VERIFIED | Named tests present and passing |
| `server/legacy.go` | (Discovered bug fix, not a plan artifact) | ✓ VERIFIED | `proto.Merge` → `proto.Marshal`/`proto.Unmarshal` fix confirmed in source; `TestAuthFlow` and `TestMutateResolvesMessageAbsentFromConstructionSnapshot` (which exercises this exact path) both PASS |

### Key Link Verification

| From | To | Via | Status | Details |
|---|---|---|---|---|
| `inserter/inserter.go` | `compiler/lib/parser/parser.go` | `i.parser.TypeResolver` | ✓ WIRED | grep confirms 2 occurrences; behavior confirmed by passing tests |
| `agent/filekv/filekv.go` `Get` | `compiler/lib/parser/parser.go` | `s.parser.ReadConfig` (unchanged, already tiered) | ✓ WIRED | Confirmed by passing on-demand resolution tests |
| `server/server.go` `MutateConfig` | `compiler/lib/parser/parser.go` | `s.parser.TypeResolver` | ✓ WIRED | grep + `TestAuthFlow` pass |
| `server/server.go` `Init` reflection | `discoveryFiles` (retained discovery registry) | `DescriptorResolver: discoveryFiles` | ✓ WIRED | grep confirms 2 occurrences; `TestReflectionDescribesCustomAndBuiltinServices` passes for both custom and hand-registered services |
| `server/server.go GenReflectionUI` | `server/server.go collectExamples` | delegation | ✓ WIRED | Confirmed by source and passing tests |
| End-state gate: no surviving construction-time-snapshot resolution site outside `compiler/lib/parser/` and the 2 allowed `ExtensionResolver` fields | — | repo-wide grep | ✓ PASS | Independently re-ran the corrected gate (`grep -rn 'LocalResolver' inserter/ mutate/ agent/ server/` filtered for comments, the 2 `ExtensionResolver` fields, and `_test.go` files) — zero offenders in production code |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|---|---|---|---|---|
| CONS-02 | 14-01, 14-05 | Inserter resolves types correctly | ✓ SATISFIED | Tests pass; on-demand claim independently verified |
| CONS-03 | 14-02, 14-05, 14-07 | Agent's filekv store resolves types correctly | ✓ SATISFIED (functional) / see gap for safety | Tests pass for correctness; traversal guard gap is separate from resolution correctness |
| CONS-04 | 14-04, 14-05 | GenReflectionUI reports rather than skips | ✓ SATISFIED | Tests pass; walk-completion and aggregation confirmed |
| SAFE-02 | 14-03, 14-06, 14-07 | Concurrent requests race-free under `-race` | ✓ SATISFIED | Full bounded suite green with 0 `WARNING: DATA RACE`; dedicated tight-loop tests pass with documented failure-detection validation |
| SAFE-03 | 14-08 | Loaded-file count stays proportional | ✓ SATISFIED | Escalation and sequence guards both pass |

All 5 requirement IDs from the phase's plans (CONS-02, CONS-03, CONS-04, SAFE-02, SAFE-03) are present in REQUIREMENTS.md and marked `[x]`/`Complete`. No orphaned requirements found: REQUIREMENTS.md's Phase 14 mapping matches exactly the 5 IDs claimed across the 8 plans.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|---|---|---|---|---|
| `agent/filekv/filekv.go` | 98, 138 | Path-traversal guard does not reject a leading `../` key segment | 🛑 Blocker | Confirmed live: a caller-supplied key can read a file outside `protoconfRoot`. Directly touches this phase's stated goal ("...safely... no silent wrong answers"). See Gap. |
| `agent/filekv/filekv_test.go` | 306-320 | `TestGetRejectsTraversalKey` asserts only `require.Error`, which passes via `os.Stat`'s `ErrNotExist` rather than the guard firing | 🛑 Blocker (false assurance) | A test this phase added claims to "pin" a security regression but cannot detect the regression it names. Confirmed via live reproduction with a real file present at the traversal target. |
| `agent/filekv/filekv.go` | 252-281 | `readEvents` sends on a per-watch channel outside the lock that also closes it — potential "send on closed channel" panic under concurrent `Close()` | ⚠️ Warning (pre-existing, code-review WR-01, not newly introduced) | Not independently re-verified with a repro; carried from `14-REVIEW.md`. Not blocking this phase's goal since it is a pre-existing, explicitly `ponytail:`-flagged ceiling. |
| `server/server.go` | 751-794 | `GenReflectionUI` allocates a new bufconn listener + goroutine per call, retained until process shutdown | ⚠️ Warning (pre-existing, code-review WR-02, exercised harder by this phase's 5s-ticker tests) | Not blocking; flagged by code review as a resource-leak concern growing in severity due to this phase's new call sites, but out of this phase's stated scope (D-01/D-03/D-04/D-05/D-06/D-07/D-08). |

No `TBD`/`FIXME`/`XXX` debt markers found in any phase-touched file (the `XXX` hits found are identifier substrings — `XXXinsertVersion`, `XXX_MessageName` — not debt-marker comments).

### Behavioral Spot-Checks / Test Execution

Ran the full bounded repo test suite once: `go test -race -timeout 300s -skip 'Test_cliCommand_Run' ./...` — exit 0, all 20 tested packages `ok`, zero `WARNING: DATA RACE` occurrences. `go build ./...` is silent. The repo-wide end-state gate (Task 2 of 14-05, corrected for test-file exclusions) independently re-run: zero offenders.

Individually re-ran every named test cited in each plan's `<verify>` block across all 8 plans (inserter, agent/filekv, server, mutate, compiler/lib/parser) rather than trusting SUMMARY claims — all passed. Independently wrote and ran two scratch reproduction tests (not committed) confirming the CR-01 path-traversal vulnerability is live: the first showed `Get` bypassing the guard and reaching a file outside `protoconfRoot`; the second, using a schema-valid fixture, showed `Get` returning `err=nil` and the outside-root file's actual decoded content.

### Human Verification Required

None required beyond the recorded gap — the traversal issue is deterministically reproducible and does not need human judgment to confirm; it needs a decision on remediation scope (fix now in this phase vs. route to `/gsd-secure-phase`, as 14-02's own threat model explicitly deferred formal STRIDE prohibition ownership there while still asserting — incorrectly — that the risk was mitigated).

### Gaps Summary

Five of the ROADMAP's five explicitly numbered success criteria are fully verified against
real, independently-executed tests and source inspection — not SUMMARY.md claims. All 8
plans' named tests pass; the repo-wide `-race` suite is clean; the end-state single-resolver
gate holds; the `server/legacy.go` bug the phase discovered and fixed is confirmed correct
and load-bearing (it is the exact path every real `protoconf mutate` invocation uses).

One gap remains, tied to the phase goal's own wording rather than to a numbered criterion:
`agent/filekv/filekv.go`'s `Get`/`Watch` path-traversal guard does not reject a leading
`../` key segment, and this phase shipped a "regression" test
(`TestGetRejectsTraversalKey`) and two threat-model entries (14-02's T-14-03, 14-07's
T-14-21) that assert this exact risk is "pinned"/"mitigate[d]" — a claim I independently
disproved with a live reproduction. The vulnerability itself predates Phase 14 (code
review traces it to commit `0745e19`), but the false assurance is new, and the phase goal
explicitly promises "safely... no silent wrong answers." This is judged a phase-level gap,
not a pre-existing-and-out-of-scope item, because the phase's own artifacts (test +
threat model) make a specific, false correctness claim about code this phase's plan
directly discusses and is titled around.

**This looks like a scoping judgment call, not a coding oversight** — the plan's own
`<prohibition_breadcrumbs>` in 14-02 explicitly says "Path traversal on Store.Get's key is
canon security — covered by /gsd-secure-phase... not minted as a bespoke prohibition,"
suggesting the executor deliberately treated deep security-hardening as another workflow's
job. If the project's intent is that `/gsd-secure-phase` is the correct venue for the actual
fix, the fix could be deferred there — but the false "mitigate"/"pinned as a regression"
claims in this phase's own threat model and test docstring should not stand uncorrected,
since a future reader (or an automated gate) could reasonably treat them as proof this is
already handled.

**To accept this as out-of-scope for Phase 14** (deferring the fix to `/gsd-secure-phase`
while requiring only the false-claim correction here), add to VERIFICATION.md frontmatter:

```yaml
overrides:
  - must_have: "The agent's filekv store serves configs safely with no silent wrong answers"
    reason: "Path-traversal guard predates Phase 14 (commit 0745e19); canon security fix is scoped to /gsd-secure-phase per 14-02's own prohibition_breadcrumbs. Accepting with the requirement that 14-02/14-07's threat-model 'mitigate' claims and TestGetRejectsTraversalKey's docstring be corrected to not overclaim."
    accepted_by: "{name}"
    accepted_at: "{ISO timestamp}"
```

---

_Verified: 2026-09-08T21:35:00Z_
_Verifier: Claude (gsd-verifier)_
