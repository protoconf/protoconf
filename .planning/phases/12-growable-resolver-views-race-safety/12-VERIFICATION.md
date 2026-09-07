---
phase: 12-growable-resolver-views-race-safety
verified: 2026-09-08T02:00:00Z
status: gaps_found
score: 9/10 must-haves verified
covered_files:
  - ".planning/REQUIREMENTS.md"
  - ".planning/phases/12-growable-resolver-views-race-safety/12-01-PLAN.md"
  - ".planning/phases/12-growable-resolver-views-race-safety/12-01-SUMMARY.md"
  - ".planning/phases/12-growable-resolver-views-race-safety/12-02-PLAN.md"
  - ".planning/phases/12-growable-resolver-views-race-safety/12-02-SUMMARY.md"
  - ".planning/phases/12-growable-resolver-views-race-safety/12-03-PLAN.md"
  - ".planning/phases/12-growable-resolver-views-race-safety/12-03-SUMMARY.md"
  - "compiler/lib/compiler.go"
  - "compiler/lib/concurrent_compile_test.go"
  - "compiler/lib/config.go"
  - "compiler/lib/parser/canonical_identity_test.go"
  - "compiler/lib/parser/growable_resolver_test.go"
  - "compiler/lib/parser/parser.go"
  - "utils/growable_resolver_race_test.go"
  - "utils/growable_resolver_test.go"
  - "utils/utils.go"
covered_digest: "v1:sha256:d57f4fc2af8954d5cd14f7b50412c63a7004d34244ed2f8013617f38906da6de"
behavior_unverified: 0
overrides_applied: 0
gaps:
  - truth: "A file that is genuinely external to this registry — present in the resolver because a consumer hand-registered it, absent from FileRegistry — still resolves through the pre-existing wrap path, so the mutation server's six hand-registered well-known files keep working. (12-02-PLAN.md must_haves.truths)"
    status: failed
    reason: >
      Falsified with reproducible evidence. Plan 12-01 rerouted ParseFilesX's resolver read from
      the raw p.FilesResolver field to p.registry.FindFileByPath (compiler/lib/parser/parser.go:125).
      For an eager registry (len(d.ImportPaths) == 0 — the mutation server's construction shape,
      D-03), GetFilesResolver (utils/utils.go:171-194) always returns a freshly built
      *protoregistry.Files and NEVER assigns d.filesResolver, which stays permanently nil.
      FindFileByPath (utils/utils.go:468-475) returns ErrNoGrowableResolver whenever
      d.filesResolver == nil. So for any Parser built over an eager registry, ParseFilesX's
      resolver lookup always errors, falls into the ParseOne branch, and ParseOne itself always
      fails with ErrLazyParseDisabled for an eager registry (ImportPaths empty) — the request fails
      outright instead of reaching the canonical-lookup/desc.WrapFile fallback the plan says
      protects this exact case.

      Independently reproduced (not just code-read): a throwaway test built an eager
      DescriptorRegistry, registered a file directly onto Parser.FilesResolver the same way
      server.go:292-297 registers its six well-known files, confirmed the raw field lookup still
      succeeds (pre-existing behavior, unbroken), then called ParseFilesX on the same path and
      confirmed it returns a non-nil error ("registry has no growable files resolver \n on-demand
      parsing requires import paths"). The test was deleted after confirming; it was not part of
      the phase deliverable.

      The claimed test coverage for this exact truth does not exist: 12-02-SUMMARY.md and
      12-02-PLAN.md's task acceptance criteria cite `go test -race ./server/...
      (TestDiscoveryScanDoesNotBackReflection)` as verification that "the mutation server's six
      hand-registered well-known files... still resolve." That test
      (server/init_prohibitions_test.go:123-144) calls `s.parser.FilesResolver.FindFileByPath`
      directly — the raw field, bypassing ParseFilesX entirely — so it cannot and does not observe
      this regression.

      Mitigating fact, also independently confirmed: ParseFilesX's only production caller
      (compiler/lib/starlark_loader.go:211) always runs against the compiler's lazy registry
      (NewLazyModuleService sets ImportPaths, module_service.go:452-454), and the mutation
      server's own parser (server/server.go:291) never calls ParseFilesX today. So this is a latent
      defect in the general Parser.ParseFilesX contract, not a currently-triggered production
      failure — but it is a real, reproducible regression with zero test coverage, misrepresented
      in the phase's own docs as covered.
    artifacts:
      - path: "compiler/lib/parser/parser.go"
        issue: "ParseFilesX's resolver-hit dispatch (lines 125-171) is unreachable for eager registries — FindFileByPath always errors, so ParseOne (which always fails with ErrLazyParseDisabled for eager registries) is the only path ever taken. The canonical-lookup (line 142) and desc.WrapFile fallback (line 151) are both dead code for the eager-registry shape this bug targets."
    missing:
      - "Fall back to the raw p.FilesResolver field when p.registry.FindFileByPath reports ErrNoGrowableResolver (eager registry), instead of treating that error as a hard not-found. Code review 12-REVIEW.md CR-01 proposes the exact fix."
      - "A test that builds an eager registry, registers a file directly onto p.FilesResolver (mirroring server.go's pattern) without adding it to FileRegistry, and asserts ParseFilesX still resolves it via the desc.WrapFile fallback — the state this whole fallback branch exists for is currently untested in both directions (before this phase, the raw field read covered it implicitly; after this phase, nothing covers it)."
advisory:
  - finding: "Parser.FilesResolver remains a public field holding the live, unsynchronized growable *protoregistry.Files. server/server.go:292-297,426,433 and this phase's own compiler/lib/parser/growable_resolver_test.go:31,41 read/write it directly and unlocked. Safe today only because every concrete caller happens to sit on an eager (non-growing) registry — not enforced by the type system. (12-REVIEW.md WR-01)"
    category: architectural
    reason: "Not a must-have for this phase (RSLV-01's stated truth is specifically 'findable through the parser's own FilesResolver field', which the test correctly exercises) and not currently triggered by a concurrent caller. Flagged so a future phase does not treat the public field as safe to read/write from a second goroutine against a lazy-registry-backed Parser."
    evidence_status: "confirmed via grep, not exercised concurrently"
  - finding: "FileRegistry mutation methods (Merge, Import, Parse, MergeFileDescriptorSet in utils/utils.go) still take no lock of their own; only ParseAll's caller-side d.mu.Lock() makes them safe today. This is an existing invariant enforced by caller discipline, not by the methods themselves. (12-REVIEW.md WR-02)"
    category: architectural
    reason: "Pre-existing pattern, not introduced or worsened by this phase's must-haves; the phase's own concurrency claims (SAFE-01) are about ParseOne/registerFileLocked, which are correctly locked. Flagged for awareness, not blocking."
    evidence_status: "confirmed via code read, not exercised"
gaps_summary: >
  Three of Phase 12's four requirements (RSLV-01, RSLV-02, SAFE-01) are cleanly and verifiably
  achieved, each with independently-reproduced test evidence (not just SUMMARY claims). RSLV-03's
  core deliverable — the A-then-B-then-re-reference-A pointer-identity contract — is also verified.
  However, plan 12-02's own stated scope for RSLV-03 included a second, narrower guarantee: that
  fixing the canonical-pointer hazard would not regress the mutation server's hand-registered
  external-file case. That guarantee is false as shipped, for eager registries generally (of which
  the mutation server is the only production instance), and the specific test the plan's own docs
  point to as proof does not exercise the changed code path at all. The regression is currently
  latent (no live caller triggers it), but it is a reproducible defect in the general contract of
  a function (ParseFilesX) this phase directly modified, and it was reported as tested when it was
  not. That combination — a false must-have claim plus miscited test coverage — is a phase-goal
  gap, not just a code-quality nit, even though it does not currently break anything running in
  production.
---

# Phase 12: Growable Resolver Views & Race Safety Verification Report

**Phase Goal:** The compiler's live resolvers grow safely and correctly as new protos are demanded mid-compile, without rebuilding a snapshot or racing under concurrent load.
**Verified:** 2026-09-08T02:00:00Z
**Status:** gaps_found
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | A file parsed on demand is findable through the parser's own `FilesResolver` field, not a construction-time snapshot (RSLV-01) | ✓ VERIFIED | `compiler/lib/parser/growable_resolver_test.go#TestParserFilesResolverGrowsAfterConstruction`; re-ran `go test -race ./compiler/lib/parser/...` — pass |
| 2 | Growth is incremental, never a rebuild: registration count tracks distinct files exactly AND the resolver pointer never changes (RSLV-02, ROADMAP criterion 2) | ✓ VERIFIED | `utils/growable_resolver_test.go#TestRegistrationCountIsIncremental` asserts both the count-equality AND `require.Same(t, resolver0, dr.GetFilesResolver())` pointer check in the same test — the two-assertion shape the emphasis flagged as fakeable is genuinely present. Re-ran: pass |
| 3 | Registration-error count stays 0 across every growth path | ✓ VERIFIED | Same test file, 3 tests assert `FilesResolverRegistrationErrorCount() == 0` under both orderings. Re-ran: pass |
| 4 | `ParseAll`'s eager fallback registers through the same before/after diff as `lazyLoaded` — one insert point | ✓ VERIFIED | `utils/growable_resolver_test.go#TestParseAllRegistersIntoFilesResolver`; code inspection of `registerFileLocked` call sites in `recordFileLocked` and `ParseAll`'s diff loop confirms single insert point |
| 5 | Every eager consumer (server, inserter, mutate, agent/filekv, mod sync) keeps byte-identical, non-cached `GetFilesResolver()` behavior (D-03) | ✓ VERIFIED | `GetFilesResolver` (utils/utils.go:171-194): `len(d.ImportPaths)==0` branch always builds fresh, never touches `d.filesResolver`. `TestParserFilesResolverGrowsAfterConstruction` Test 3 asserts `require.NotSame(r1, r2)` for an eager registry. `go test -race ./server/... ./inserter/... ./mutate/... ./agent/filekv/...` pass (per 12-01-SUMMARY, not independently re-run but low-risk/well-isolated) |
| 6 | Two concurrent `ParseOne` calls for distinct paths both complete, both files findable, no registration lost (SAFE-01) | ✓ VERIFIED | `utils/growable_resolver_race_test.go#TestRegisterFileRacesRangeFiles`; re-ran `go test -race -count=1` — pass. **Teeth independently confirmed**: temporarily stripped `d.mu.RLock()`/`RUnlock()` from `FindFileByPath` and `RangeFiles`, re-ran the test, observed two genuine `WARNING: DATA RACE` reports pointing exactly at `registerFileLocked`'s write racing `RangeFiles`/`FindFileByPath` reads, then restored the file (`git checkout -- utils/utils.go`, confirmed empty diff) |
| 7 | Cumulative registration count under concurrency equals distinct files, never N× (SAFE-01, ROADMAP criterion 3) | ✓ VERIFIED | Same race test asserts `FilesResolverRegistrationErrorCount()==0` and `FilesResolverRegistrationCount() > 8`; `TestConcurrentCompile`'s extended assertions (D-02 under concurrency) re-ran with `-race -count=2` — pass |
| 8 | Compiling A, then B, then re-referencing A resolves all three correctly in one pass, with A's descriptor the identical Go pointer both times (RSLV-03, ROADMAP criterion 1) | ✓ VERIFIED | `compiler/lib/parser/canonical_identity_test.go#TestReReferencedProtoKeepsPointerIdentity`; re-ran `go test -race` — pass. 12-02-SUMMARY honestly discloses this test passes against both pre- and post-fix code (a prospective contract-lock, not a red→green transition) — judged as acceptable per plan's own explicit instruction to report this honestly rather than overclaim, and the pointer-identity property does genuinely hold for the lazy-registry path this test exercises |
| 9 | `compiler/lib/config.go`'s config struct no longer carries the dead `protoResolver` snapshot field; compiler builds with no unused import | ✓ VERIFIED | `grep -rln protoResolver --include="*.go" .` returns no matches; `go build ./...` exits 0 |
| 10 | A file present in the resolver but absent from `FileRegistry` (the mutation server's six hand-registered well-known files) still resolves through `ParseFilesX`'s pre-existing wrap path | ✗ FAILED | See Gaps below — falsified with a reproducible probe test, independent of the code-review finding |

**Score:** 9/10 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `utils/utils.go` | Growable, locked `filesResolver` + accessors | ✓ VERIFIED | `filesResolver`, `registrationCount`, `registrationErrors` fields; `GetFilesResolver`, `registerFileLocked`, `FindFileByPath`, `RangeFiles`, `FilesResolverRegistrationCount`, `FilesResolverRegistrationErrorCount` all present, all locked via `d.mu` |
| `compiler/lib/parser/parser.go` | `ParseFilesX` rerouted through locked accessor + canonical-pointer fix | ⚠️ PARTIAL | Locked accessor correctly wired for branch 1/lazy path; canonical-pointer fix correct for lazy registries; **eager-registry dispatch is broken** (see gap) |
| `compiler/lib/config.go` | Dead `protoResolver` field removed | ✓ VERIFIED | Field and orphaned import both gone, confirmed by grep and successful build |
| `compiler/lib/parser/canonical_identity_test.go` | RSLV-03 pointer-identity test | ✓ VERIFIED | Exists, `TestReReferencedProtoKeepsPointerIdentity`, 6 `require.Same` assertions, passes under `-race` |
| `utils/growable_resolver_race_test.go` | Dedicated RegisterFile-vs-RangeFiles race test | ✓ VERIFIED | Exists, forces continuous interleaving via unpaced tight-loop readers, confirmed capable of catching a real regression (lock-removal sanity check reproduced independently) |
| `compiler/lib/concurrent_compile_test.go` | Production-shaped concurrency test extended with resolver assertions | ✓ VERIFIED | `TestConcurrentCompile` retains original `LoadedFileCount` bounds, gains D-02 non-divergence block, passes under `-race -count=2` |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `starlark_loader.go:211` `loadProto` | `ParseFilesX` | direct call | ✓ WIRED | Confirmed only production caller, always against lazy registry — this is why the eager-path gap is latent rather than live |
| `server/server.go:291-297` mutation server | `Parser.FilesResolver` (raw field) | `RegisterFile` × 6 | ✓ WIRED (unchanged) | These registrations still work via the raw field; the break is specifically in `ParseFilesX`'s ability to see them, not in the registrations themselves |
| `recordFileLocked`/`ParseAll` | `registerFileLocked` → `filesResolver.RegisterFile` | single locked insert point | ✓ WIRED | Confirmed by code read; both callers converge on one function |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| RSLV-01/02 growth + pointer-identity | `go test -race ./compiler/lib/parser/... -run 'TestReReferencedProtoKeepsPointerIdentity'` | PASS | ✓ PASS |
| RSLV-02 counter + `require.Same` pointer gate | `go test -race ./utils/... -run 'TestRegistrationCountIsIncremental'` | PASS | ✓ PASS |
| SAFE-01 dedicated race test | `go test -race -count=1 ./utils/... -run 'TestRegisterFileRacesRangeFiles'` | PASS | ✓ PASS |
| SAFE-01 race-test teeth (independently reproduced) | Stripped `d.mu` from `FindFileByPath`/`RangeFiles`, re-ran test | 2× `WARNING: DATA RACE` fired against `registerFileLocked` write vs. reads | ✓ PASS (confirms detector capability) |
| CR-01 eager hand-registered file via `ParseFilesX` | Throwaway probe: eager registry + `p.FilesResolver.RegisterFile` (mirroring server.go) + `p.ParseFilesX(path)` | Non-nil error: `registry has no growable files resolver / on-demand parsing requires import paths` | ✗ FAIL (confirms the gap; probe deleted after use, not part of deliverable) |
| Whole-module regression | `go build ./...`; `go test -race -count=1 $(go list ./... \| grep -v '/agent$')` | exit 0, no FAIL, no race lines | ✓ PASS |
| Debt markers | `grep -n -E "TBD\|FIXME\|XXX\|TODO\|HACK\|PLACEHOLDER"` across all 9 files this phase touched | no matches | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|--------------|--------|----------|
| RSLV-01 | 12-01 | `FilesResolver`/`LocalResolver` reflect files parsed after construction | ✓ SATISFIED | Truths 1, 5 |
| RSLV-02 | 12-01 | Incremental registration, no `FileDescriptorSet` rebuild | ✓ SATISFIED | Truths 2, 3, 4 |
| RSLV-03 | 12-02 | load A, load B, re-read A resolves all three correctly | ⚠️ SATISFIED WITH FLAGGED REGRESSION | Truth 8 satisfied; truth 10 (a truth the same plan's own frontmatter also scoped to RSLV-03's fix) is FAILED — see gap |
| SAFE-01 | 12-01, 12-03 | Concurrent compiles race-free under `-race` | ✓ SATISFIED | Truths 6, 7, and independently-reproduced race-detector teeth |

No orphaned requirements — all four IDs mapped to this phase in REQUIREMENTS.md appear in a plan's `requirements:` frontmatter, and all four appear in the phase's ROADMAP entry.

### Anti-Patterns Found

None of TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER in any of the 9 files this phase created or modified. No stub returns, no hardcoded empty data flowing to output.

Two pre-existing-pattern findings from `12-REVIEW.md` (WR-01, WR-02) are carried into `advisory:` above — neither is a must-have regression introduced beyond what's already flagged as CR-01, and neither is exercised by a concurrent caller today.

### Human Verification Required

None. All must-haves in this phase are resolvable by code reading, targeted test execution, or direct reproduction — no visual, real-time, or external-service behavior is in scope.

### Gaps Summary

See `gaps_summary` in frontmatter. In short: RSLV-01, RSLV-02, and SAFE-01 are cleanly verified with independently-reproduced evidence, including confirming the race test's teeth by deliberately breaking the lock and watching the detector fire. RSLV-03's headline pointer-identity contract (ROADMAP success criterion 1) is also genuinely verified. The one gap is plan 12-02's own secondary must-have — that the canonical-pointer fix would not regress the mutation server's external-file resolution path — which is false as shipped for any `Parser` built over an eager registry, is currently latent only because no production caller reaches it that way today, and was reported as verified by a test that does not exercise the changed code path. `12-REVIEW.md` (CR-01) already identified this from static analysis; this verification independently reproduced the failure behaviorally with a standalone probe test and confirmed the miscited test coverage claim directly.

**This looks like an incomplete fix, not an intentional deviation** — no override is suggested. The developer should decide whether to accept this as a documented, latent gap (given zero current production impact) or route it to a closure plan before the phase is considered done. Given the phase explicitly created this hazard and explicitly claimed (in its own plan text and threat model T-12-06) that it was mitigated and tested, the recommended path is a small closure plan implementing `12-REVIEW.md`'s suggested fix (fall back to the raw field on `ErrNoGrowableResolver`) plus the missing eager-registry test, rather than an override.

Also carried forward for developer attention (not a gap, not blocking): plan 12-03's `<flagged_planner_assumptions>` notes that SAFE-01's spec-less `unclassified` edge-coverage probe row was left unresolved rather than backstopped, and no executor identified a specific uncovered SAFE-01 edge (e.g., concurrent compiles against different registries in one process, or a compile racing `mod sync`'s own eager registry). Surfacing per the plan's own instruction; does not block this verification.

---

_Verified: 2026-09-08T02:00:00Z_
_Verifier: Claude (gsd-verifier)_
