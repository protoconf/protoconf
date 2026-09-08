---
phase: 12-growable-resolver-views-race-safety
verified: 2026-09-08T14:55:00Z
status: passed
score: 10/10 must-haves verified
covered_files:
  - ".planning/REQUIREMENTS.md"
  - ".planning/phases/12-growable-resolver-views-race-safety/12-01-PLAN.md"
  - ".planning/phases/12-growable-resolver-views-race-safety/12-01-SUMMARY.md"
  - ".planning/phases/12-growable-resolver-views-race-safety/12-02-PLAN.md"
  - ".planning/phases/12-growable-resolver-views-race-safety/12-02-SUMMARY.md"
  - ".planning/phases/12-growable-resolver-views-race-safety/12-03-PLAN.md"
  - ".planning/phases/12-growable-resolver-views-race-safety/12-03-SUMMARY.md"
  - ".planning/phases/12-growable-resolver-views-race-safety/12-04-PLAN.md"
  - ".planning/phases/12-growable-resolver-views-race-safety/12-04-SUMMARY.md"
  - ".planning/phases/12-growable-resolver-views-race-safety/12-REVIEW.md"
  - ".planning/phases/12-growable-resolver-views-race-safety/12-UAT.md"
  - "compiler/lib/compiler.go"
  - "compiler/lib/concurrent_compile_test.go"
  - "compiler/lib/config.go"
  - "compiler/lib/parser/canonical_identity_test.go"
  - "compiler/lib/parser/eager_resolver_fallback_test.go"
  - "compiler/lib/parser/growable_resolver_test.go"
  - "compiler/lib/parser/parser.go"
  - "utils/growable_resolver_race_test.go"
  - "utils/growable_resolver_test.go"
  - "utils/utils.go"
covered_digest: "v1:sha256:699c8f0f527177a1832713b8520bf7bfe5679a73933abb6e0ac2f9c9400c2b99"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  latest_pass: "2026-09-08T14:55:00Z — re-verified at HEAD after 13-03 (862410e) rewrote 4 covered source files. Truth 4 moot (ParseAll deleted), truth 3 consolidated into TestRegistrationCountIsIncremental, truths 5 and 10 intact. Verdict unchanged: passed. See ## Re-Verification After Downstream Drift."
  previous_status: gaps_found
  previous_score: 9/10
  gaps_closed:
    - "A file present in the resolver but absent from FileRegistry (the mutation server's six hand-registered well-known files) now resolves through ParseFilesX's pre-existing wrap path, even on an eager registry (D-03)."
  gaps_remaining: []
  regressions: []
---

# Phase 12: Growable Resolver Views & Race Safety Verification Report

**Phase Goal:** The compiler's live resolvers grow safely and correctly as new protos are demanded mid-compile, without rebuilding a snapshot or racing under concurrent load.
**Verified:** 2026-09-08T14:55:00Z (re-verified after downstream drift; gap-closure pass 2026-09-08T09:00:00Z)
**Status:** passed
**Re-verification:** Yes — after gap closure (plan 12-04)

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | A file parsed on demand is findable through the parser's own `FilesResolver` field, not a construction-time snapshot (RSLV-01) | ✓ VERIFIED | Regression check: re-ran `go test -race ./compiler/lib/parser/...` — pass (unchanged since prior verification) |
| 2 | Growth is incremental, never a rebuild: registration count tracks distinct files exactly AND the resolver pointer never changes (RSLV-02) | ✓ VERIFIED | Regression check: `utils/growable_resolver_test.go#TestRegistrationCountIsIncremental` — pass |
| 3 | Registration-error count stays 0 across every growth path | ✓ VERIFIED | Same test file — pass |
| 4 | `ParseAll`'s eager fallback registers through the same before/after diff as `lazyLoaded` — one insert point | ✓ VERIFIED | `utils/growable_resolver_test.go#TestParseAllRegistersIntoFilesResolver` — pass (unchanged) |
| 5 | Every eager consumer keeps byte-identical, non-cached `GetFilesResolver()` behavior (D-03) | ✓ VERIFIED | `grep -n "len(d.ImportPaths)" utils/utils.go` confirms the gate is intact and unmodified by 12-04's diff (`git diff a2ebd45..HEAD -- utils/utils.go` is empty) |
| 6 | Two concurrent `ParseOne` calls for distinct paths both complete, both files findable, no registration lost (SAFE-01) | ✓ VERIFIED | `utils/growable_resolver_race_test.go#TestRegisterFileRacesRangeFiles` — pass under full-module `-race` run |
| 7 | Cumulative registration count under concurrency equals distinct files, never N× (SAFE-01) | ✓ VERIFIED | Full-module `go test -race -count=1 $(go list ./... | grep -v '/agent$')` — pass, no data races reported |
| 8 | Compiling A, then B, then re-referencing A resolves all three correctly in one pass, with A's descriptor the identical Go pointer both times (RSLV-03) | ✓ VERIFIED | `compiler/lib/parser/canonical_identity_test.go#TestReReferencedProtoKeepsPointerIdentity` — independently re-ran, pass; canonical-lookup branch (`p.registry.FileDescriptor(resolvedFd.Path())`, parser.go:153) confirmed unchanged and still positioned ahead of the wrap tail |
| 9 | `compiler/lib/config.go`'s config struct no longer carries the dead `protoResolver` snapshot field; compiler builds with no unused import | ✓ VERIFIED | `grep -rln protoResolver --include="*.go" .` returns no matches; `go build ./...` (via module test run) exits 0 |
| 10 | A file present in the resolver but absent from `FileRegistry` (the mutation server's six hand-registered well-known files) resolves through `ParseFilesX`'s pre-existing wrap path, even on an eager registry (D-03) | ✓ VERIFIED (gap closed) | See "Gap Closure Evidence" below — independently reproduced red-then-green, not accepted on SUMMARY claim alone |

**Score:** 10/10 truths verified

### Gap Closure Evidence (Truth 10 — previously FAILED)

The prior verification (2026-09-08T02:00:00Z) falsified this truth with a reproducible probe: `ParseFilesX`'s resolver read (`p.registry.FindFileByPath`) always returns `ErrNoGrowableResolver` on an eager registry, which was treated as hard not-found, making the canonical-lookup/`desc.WrapFile` tail unreachable for hand-registered external files. Plan 12-04 closed this. I independently re-verified the closure rather than trusting 12-04-SUMMARY.md's claims:

1. **Code diff read directly.** `compiler/lib/parser/parser.go:134-136` now contains:
   ```go
   if errors.Is(resolverErr, utils.ErrNoGrowableResolver) {
       resolvedFd, resolverErr = p.FilesResolver.FindFileByPath(filename)
   }
   ```
   inserted immediately after the `p.registry.FindFileByPath(filename)` call and before the `if resolverErr != nil` branch, exactly as the plan specified. `git diff a2ebd45..HEAD -- compiler/lib/parser/parser.go` shows an 11-line, purely additive change; no other line in the function moved.

2. **`utils/utils.go` untouched.** `git diff a2ebd45..HEAD -- utils/utils.go` is empty — `FindFileByPath`'s `ErrNoGrowableResolver` contract, `GetFilesResolver`'s `len(d.ImportPaths) == 0` gate, and D-03 are all unmodified, satisfying the plan's prohibitions.

3. **Canonical-lookup branch unmoved.** `p.registry.FileDescriptor(resolvedFd.Path())` (parser.go:153) is still present, still runs before the `desc.WrapFile` tail, and is unconditional on registry shape — 12-02's RSLV-03 pointer-identity fix is intact (confirmed by `TestReReferencedProtoKeepsPointerIdentity` passing).

4. **Regression test read and independently re-executed.** `compiler/lib/parser/eager_resolver_fallback_test.go` builds an eager `DescriptorRegistry`, hand-registers `grpc_health_v1.File_grpc_health_v1_health_proto` directly onto `p.FilesResolver` (mirroring `server/server.go:291-297`), and asserts resolution **exclusively through `p.ParseFilesX(...)`** — never through `p.FilesResolver.FindFileByPath` as the load-bearing assertion (the raw field is read only once, as a negative setup precondition on line 42). This satisfies the plan's explicit prohibition against inheriting the `TestDiscoveryScanDoesNotBackReflection` miscitation.

5. **Red-to-green transition independently reproduced**, not taken on the SUMMARY's word: checked out commit `3401e6c` (the RED commit — confirmed via `git show --name-only --format= 3401e6c` that it touches only the new test file, not `parser.go`) into a scratch worktree and ran the test directly:
   ```
   --- FAIL: TestParseFilesXResolvesEagerHandRegisteredFile
       --- FAIL: .../the_failed_truth:_eager_hand-registered_file_resolves_through_ParseFilesX
           Error: registry has no growable files resolver
                  on-demand parsing requires import paths
   ```
   Then ran the same test at `HEAD` (commit `f57d3fd` and after):
   ```
   --- PASS: TestParseFilesXResolvesEagerHandRegisteredFile (all 4 subtests)
   ```
   Both the failure text and the transition match the plan's documented claim exactly.

6. **No regression in already-verified truths.** `go test -race ./compiler/lib/parser/... -run 'TestParseFilesXResolvesEagerHandRegisteredFile|TestReReferencedProtoKeepsPointerIdentity|TestParserFilesResolverGrowsAfterConstruction|TestParser_ParseFilesX' -v` — all pass, no data races.

7. **Full-module race gate re-run** (the same exclusion phase 12's own verification used, `agent` package hangs on a real Consul dependency, documented in prior artifacts): `go test -race -count=1 $(go list ./... | grep -v '/agent$')` — all packages `ok`, zero `FAIL`, zero `WARNING: DATA RACE`.

8. **No debt markers introduced.** `grep -n -E "TBD|FIXME|XXX|TODO|HACK|PLACEHOLDER" compiler/lib/parser/parser.go compiler/lib/parser/eager_resolver_fallback_test.go` — no matches. `go vet ./compiler/... ./utils/... ./server/...` exits 0.

The gap is closed with independently-reproduced evidence, not accepted on SUMMARY.md's or the code review's claims alone.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `compiler/lib/parser/parser.go` | `ParseFilesX` rerouted through locked accessor + canonical-pointer fix + eager fallback | ✓ VERIFIED | All three properties confirmed present, wired, and correctly ordered |
| `compiler/lib/parser/eager_resolver_fallback_test.go` | Regression test proving the fallback via `ParseFilesX` | ✓ VERIFIED | Exists, `package parser`, single test function with 4 subtests, all invoke `ParseFilesX`, red-to-green independently reproduced |
| `utils/utils.go` | Growable, locked `filesResolver` + accessors (unchanged by this closure) | ✓ VERIFIED | Byte-identical since prior verification (`git diff` empty) |
| `compiler/lib/config.go` | Dead `protoResolver` field removed | ✓ VERIFIED | Unchanged since prior verification |
| `compiler/lib/parser/canonical_identity_test.go` | RSLV-03 pointer-identity test | ✓ VERIFIED | Still passes; branch it exercises unmoved |
| `utils/growable_resolver_race_test.go` | Dedicated RegisterFile-vs-RangeFiles race test | ✓ VERIFIED | Unchanged, still passes under `-race` |
| `compiler/lib/concurrent_compile_test.go` | Production-shaped concurrency test | ✓ VERIFIED | Unchanged, still passes under `-race` |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `starlark_loader.go:211` `loadProto` | `ParseFilesX` | direct call | ✓ WIRED | Unaffected — this call path always runs against the lazy registry, where the new branch is a no-op (confirmed by 12-REVIEW.md's trace and by `TestParser_ParseFilesX` passing unchanged) |
| `server/server.go:291-297` mutation server | `ParseFilesX`'s new fallback branch | `errors.Is(..., ErrNoGrowableResolver)` → `p.FilesResolver.FindFileByPath` | ✓ WIRED (newly restored) | Reproduced synthetically by `eager_resolver_fallback_test.go` using the exact hand-registration pattern `server.go` uses; the live `server` package itself never calls `ParseFilesX` today, so this remains latent-but-correct rather than exercised in production, same as before this phase |
| `recordFileLocked`/`ParseAll` | `registerFileLocked` → `filesResolver.RegisterFile` | single locked insert point | ✓ WIRED | Unchanged |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Gap-closure regression, RED (independently reproduced at commit 3401e6c) | `go test ./compiler/lib/parser/ -run TestParseFilesXResolvesEagerHandRegisteredFile -v` (scratch worktree at 3401e6c) | FAIL, sentinel text matches plan's documented claim exactly | ✓ PASS (confirms genuine RED) |
| Gap-closure regression, GREEN (at HEAD) | `go test -race ./compiler/lib/parser/... -run TestParseFilesXResolvesEagerHandRegisteredFile -v` | 4/4 subtests PASS | ✓ PASS |
| No regression in 12-01/12-02/12-03 gates | `go test -race ./compiler/lib/parser/... -run 'TestReReferencedProtoKeepsPointerIdentity|TestParserFilesResolverGrowsAfterConstruction|TestParser_ParseFilesX' -v` | all PASS | ✓ PASS |
| Whole-module regression (excluding known-hanging `agent` package) | `go test -race -count=1 $(go list ./... \| grep -v '/agent$')` | exit 0, all `ok`, no FAIL, no race lines | ✓ PASS |
| `go vet` clean | `go vet ./compiler/... ./utils/... ./server/...` | exit 0 | ✓ PASS |
| Debt markers | `grep -n -E "TBD\|FIXME\|XXX\|TODO\|HACK\|PLACEHOLDER"` across the 2 files 12-04 touched | no matches | ✓ PASS |
| Scope containment | `git diff a2ebd45..HEAD --stat` | Only `parser.go` (+11) and new test file (+84) changed in code; `utils/utils.go` untouched | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|--------------|--------|----------|
| RSLV-01 | 12-01 | `FilesResolver`/`LocalResolver` reflect files parsed after construction | ✓ SATISFIED | Truths 1, 5 |
| RSLV-02 | 12-01 | Incremental registration, no `FileDescriptorSet` rebuild | ✓ SATISFIED | Truths 2, 3, 4 |
| RSLV-03 | 12-02, 12-04 | load A, load B, re-read A resolves all three correctly; hand-registered external files still resolve | ✓ SATISFIED | Truth 8 (pointer identity) and truth 10 (eager-registry fallback, gap now closed) |
| SAFE-01 | 12-01, 12-03 | Concurrent compiles race-free under `-race` | ✓ SATISFIED | Truths 6, 7; full-module race gate green |

No orphaned requirements — all four IDs mapped to Phase 12 in REQUIREMENTS.md (lines 21-23, 70) appear in a plan's `requirements:` frontmatter (12-01, 12-02, 12-03, 12-04) and REQUIREMENTS.md marks all four `[x]` Complete.

### Anti-Patterns Found

None. `grep -n -E "TBD|FIXME|XXX|TODO|HACK|PLACEHOLDER"` across all files this phase (including 12-04's closure) created or modified returns no matches. No stub returns, no hardcoded empty data, no scope creep beyond the documented `parser.go` + new test file diff.

### Human Verification Required

None. All must-haves in this phase, including the gap-closure plan's, are resolvable by code reading, targeted test execution, and direct reproduction (red-then-green transition independently confirmed in a scratch worktree) — no visual, real-time, or external-service behavior is in scope.

### Gaps Summary

No gaps remain. The one gap from the prior verification — `ParseFilesX` failing to resolve hand-registered external files (the mutation server's six well-known protos) on an eager registry — is closed by plan 12-04's four-line `errors.Is(resolverErr, utils.ErrNoGrowableResolver)` fallback branch. This verification did not accept 12-04-SUMMARY.md's claims at face value: it independently re-read the diff, confirmed `utils/utils.go` was untouched (satisfying every "do not touch X" prohibition in 12-04-PLAN.md's frontmatter), reproduced the documented RED state in a scratch worktree at the pre-fix commit, confirmed the GREEN state at HEAD, and re-ran the full-module `-race` gate. The prior verification's specific concern — that the cited evidence (`TestDiscoveryScanDoesNotBackReflection`) bypassed `ParseFilesX` entirely — does not recur here: the new test's only load-bearing assertions invoke `p.ParseFilesX(...)` directly, and the raw-field read appears exactly once, as a negative setup precondition.

All four Phase 12 requirements (RSLV-01, RSLV-02, RSLV-03, SAFE-01) and all three ROADMAP success criteria are verified with independently-reproduced evidence:
1. A-then-B-then-re-reference-A resolves all three with pointer identity — verified (truth 8).
2. On-demand growth is incremental, no full rebuild — verified (truths 2-4).
3. Concurrent compiles race-free under `-race` — verified (truths 6-7, full-module race gate).

The advisory items carried forward from the initial verification (WR-01: public unsynchronized `Parser.FilesResolver` field; WR-02: unlocked `FileRegistry` mutators outside `ParseOne`) remain out of scope per the phase's own explicit classification and are unaffected by 12-04's closure — neither was must-have for this phase, and 12-04 introduced no new instance of either pattern (the new fallback read is race-free by construction, per D-03: nothing can grow an object an eager registry never arms).

---

_Verified: 2026-09-08T09:00:00Z_
_Verifier: Claude (gsd-verifier)_

---

## Re-Verification After Downstream Drift (2026-09-08)

`verification.status` reported `stale`. The cause is `covered_digest` drift from Phase 13, not a
Phase 12 artifact change. Of the 20 `covered_files`, five moved since the 09:00Z sign-off — four in
one commit, `862410e` (`feat(13-03): delete the whole-tree eager fallback, replace with a hard
error`): `utils/utils.go`, `compiler/lib/parser/parser.go`, `compiler/lib/compiler.go`,
`utils/growable_resolver_test.go`; plus `.planning/REQUIREMENTS.md` (`35bf4dd`, 13-04). No Phase 12
PLAN, SUMMARY or REVIEW file moved.

Because 13-03 deleted the very mechanism two of this phase's truths constrain, the drift was
re-checked truth by truth rather than re-stamped.

### Truth-by-truth disposition at HEAD

| # | Truth | Disposition |
|---|-------|-------------|
| 4 | `ParseAll`'s eager fallback registers through the same before/after diff as `lazyLoaded` — one insert point | **MOOT, not regressed.** `ParseAll` and its diff loop no longer exist. `TestParseAllRegistersIntoFilesResolver` was deleted by 13-03, which left a 19-line comment at `utils/growable_resolver_test.go:11` recording why it was deleted rather than re-pointed: the combination it exercised (ImportPaths set AND `Import`/`Parse` called directly) has no production caller outside `ParseAll` itself, so re-pointing it would assert only test-only glue. |
| 3 | Registration-error count stays 0 across every growth path | **STILL COVERED, test consolidated.** `TestFilesResolverRegistrationErrorsStayZero` was deleted in the same commit; its zero-error assertion lives on inside `TestRegistrationCountIsIncremental`, which 13-03's comment names as the sole remaining pin on the registration-diff invariant via `ParseOne`. Re-run green under `-race`. |
| 5 | Every eager consumer keeps byte-identical, non-cached `GetFilesResolver()` behavior (D-03) | **INTACT.** The `len(d.ImportPaths) == 0` gate survives 13-03's rewrite at `utils/utils.go:206`. |
| 10 | An eager hand-registered file resolves through `ParseFilesX`'s wrap path (D-03) | **INTACT.** The `errors.Is(resolverErr, utils.ErrNoGrowableResolver)` fallback survives at `compiler/lib/parser/parser.go:148`, and `ErrNoGrowableResolver` is still returned by the accessor at `utils/utils.go:468`. |
| 1, 2, 6, 7, 8, 9 | — | Unaffected; every covering test still exists and passes (below). |

### Re-run live at HEAD (2026-09-08)

Every surviving coverage ref, individually under `-race`, all PASS:
`TestConcurrentCompile`, `TestReReferencedProtoKeepsPointerIdentity`,
`TestParseFilesXResolvesEagerHandRegisteredFile` (4/4 subtests),
`TestParserFilesResolverGrowsAfterConstruction`, `TestRegisterFileRacesRangeFiles`,
`TestRegistrationCountIsIncremental`.

### 12-03 D3 re-executed, not accepted on claim

`12-03-SUMMARY.md` tagged its D3 deliverable `human_judgment: true` because the mutation
demonstration behind it was a one-time manual act with no re-runnable artifact — the SUMMARY's
claim was the only evidence. That claim was re-executed in this session rather than trusted:
stripping `d.mu.RLock()/RUnlock()` from `FindFileByPath` (`utils/utils.go:465`) and `RangeFiles`
(`:477`) produced `WARNING: DATA RACE` — write at `protoregistry.(*Files).RegisterFile` via
`registerFileLocked`, read at `RangeFiles` via `utils.go:478` — and `git checkout -- utils/utils.go`
restored it byte-identical (`git diff --stat` empty) and green. Recorded in `12-UAT.md` test 17.

**Verdict unchanged: `passed`.** No Phase 12 truth regressed under Phase 13; one (truth 4) was
rendered moot by an intentional, documented deletion, and one test's assertion was consolidated
into a sibling. `12-UAT.md` is added to `covered_files` here (17/17 pass, 0 issues).
`covered_digest` recomputed over the updated list via `computeCoveredDigest` — never hand-written.
