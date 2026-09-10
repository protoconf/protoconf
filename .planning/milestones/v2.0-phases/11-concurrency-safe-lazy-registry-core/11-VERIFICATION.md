---
phase: 11-concurrency-safe-lazy-registry-core
verified: 2026-09-08T14:35:00Z
status: passed
score: 25/25 must-have truths verified (5 roadmap success criteria + 6 requirement IDs + 14 gap-closure plan truths across 11-04/11-05), plus 7 pre-existing backstop/decision items closed via 11-UAT.md human sign-off (8 tests total, 7 pass, 1 issue whose gap is resolved)
behavior_unverified: 0
overrides_applied: 0
covered_files:
  - .planning/REQUIREMENTS.md
  - .planning/phases/11-concurrency-safe-lazy-registry-core/11-01-PLAN.md
  - .planning/phases/11-concurrency-safe-lazy-registry-core/11-01-SUMMARY.md
  - .planning/phases/11-concurrency-safe-lazy-registry-core/11-02-PLAN.md
  - .planning/phases/11-concurrency-safe-lazy-registry-core/11-02-SUMMARY.md
  - .planning/phases/11-concurrency-safe-lazy-registry-core/11-03-PLAN.md
  - .planning/phases/11-concurrency-safe-lazy-registry-core/11-03-SUMMARY.md
  - .planning/phases/11-concurrency-safe-lazy-registry-core/11-04-PLAN.md
  - .planning/phases/11-concurrency-safe-lazy-registry-core/11-04-SUMMARY.md
  - .planning/phases/11-concurrency-safe-lazy-registry-core/11-05-PLAN.md
  - .planning/phases/11-concurrency-safe-lazy-registry-core/11-05-SUMMARY.md
  - .planning/phases/11-concurrency-safe-lazy-registry-core/11-REVIEW.md
  - .planning/phases/11-concurrency-safe-lazy-registry-core/11-SECURITY.md
  - .planning/phases/11-concurrency-safe-lazy-registry-core/11-UAT.md
  - compiler/lib/compiler.go
  - compiler/lib/config.go
  - compiler/lib/mod_sync_fds_test.go
  - compiler/lib/module_service.go
  - compiler/lib/module_service_test.go
  - compiler/lib/parser/parser.go
  - compiler/lib/starlark_loader.go
  - mod/command.go
  - mod/command_test.go
  - server/server.go
  - utils/utils.go
covered_digest: "v1:sha256:560fca92a5955957f6ab2f8e22fc20496fd2531ba4f92a6de035551b0d33133f"
re_verification:
  latest_pass: "2026-09-08T14:35:00Z — re-verified at HEAD after Phase 12/13 rewrote 8 covered source files. Verdict unchanged: passed. See ## Re-Verification After Downstream Drift."
  previous_status: human_needed
  previous_score: 25/25 must-have truths (5 roadmap success criteria + 6 requirement IDs + 14 gap-closure plan truths), 1 open human_verification item (WR-02)
  gaps_closed: []
  gaps_remaining: []
  regressions: []
human_verification_closed:
  - "WR-02 (LocalFileCount()'s d.mu.RLock() did not synchronize with localFiles's only production writer, Parse) — human decision recorded in 11-UAT.md test 8 (2026-09-07): drop the RLock, document the constraint. Implemented in commit 3005e5a, confirmed live in this session against utils/utils.go: LocalFileCount() no longer takes d.mu.RLock(), carries a doc comment stating it is unsafe to call concurrently with Import/Parse, and a ponytail: marker naming the real upgrade path (a registry per dependency, not a lock on Parse's writes)."
---

# Phase 11: Concurrency-Safe Lazy Registry Core Verification Report

**Phase Goal:** A compile no longer pays for the whole repository's proto tree, and the mutation server's service catalog doesn't silently go dark under that change.
**Verified:** 2026-09-08T14:35:00Z (re-verified after downstream drift; original pass 2026-09-07T21:50:00Z)
**Status:** passed
**Re-verification:** Yes — the prior VERIFICATION.md (2026-09-07T21:15Z, `status: human_needed`) is stale. Two files changed since it was written: `utils/utils.go` (commit `3005e5a`, the fix for its own single open `human_verification` item, WR-02) and `11-UAT.md` (commits `dc2973a`, `4bd43aa`, recording that item's resolution). Nothing else in the previously-covered file set moved (confirmed via `git log --oneline` and `git status`). `11-SECURITY.md` is new since the prior report and is added to `covered_files` here.

## What Was Re-Run Live vs. Carried Forward

**Re-run live in this session** (the only thing that changed since the prior verification):
- Read `utils/utils.go`'s current `LocalFileCount()` and `Parse()` directly — confirmed the RLock was dropped, the doc comment and `ponytail:` marker are present, matching the UAT-recorded decision exactly (see Human Verification Item Closed, below).
- Confirmed via `git show 3005e5a -- utils/utils.go` that the diff is exactly the RLock removal plus documentation — no other behavioral change.
- Confirmed `GenFileDescriptorSet` (`compiler/lib/module_service.go:378`) still calls `registry.LocalFileCount()` synchronously on the same goroutine that just ran `registry.Import(registry.Parse, ...)` a few lines above, and that `m.Walk` → `walk()` is a single-threaded recursive call (no goroutines) — so the G-11-7 guard's only production call site is unaffected by dropping the RLock.
- `go test -race -count=1 ./utils/... ./compiler/... ./mod/... ./server/...` — full targeted suite across every phase-touched package, live: all packages `ok`, no `FAIL`, no `WARNING: DATA RACE`.
- `go test ./mod/... -run TestModSyncNeverPersistsEmptyDescriptorSet -v -count=1` — 3/3 PASS (G-11-7 regression).
- `go test ./mod/... -run TestModInitLockFileShapes -v` — 8/8 PASS (G-11-3 regression).
- `go test ./compiler/lib/... -run TestGeneratedCorpusCompiles -v` — PASS, `protoFilesLoaded=5` against a 50-proto corpus (SC1).
- `go run ./cmd/protoconf compile utils/testdata/small test.pconf` — live CLI, `compile finished file=test.pconf protoFilesLoaded=1 eagerFallback=false` (SC4).
- `go test ./server/... -run 'TestInitRegistersCustomService|TestInitWithNoCustomServices|TestInitServiceSetIsOrderIndependent' -v -race` — 3/3 PASS (SC5/CONS-01).
- `go vet ./utils/... ./compiler/... ./mod/... ./server/...` — clean.
- Recomputed `covered_digest` via `verification.fingerprint` over the updated file set (added `11-SECURITY.md`).

**Carried forward, not re-derived** (unchanged files, already independently re-run live in the prior verification pass on 2026-09-07T21:15Z, and now additionally re-confirmed passing as part of the full-suite `-race` run above): SC2 (`TestParseMemoization`), SC3's byte-identity half (`TestModSyncFdsByteIdentical`), the requirement-ID mapping, the anti-pattern scan of the 17 phase-touched non-test files, and the WR-01/WR-04/backstop-truth (B1/B2/B3) dispositions. Safe to carry forward because: (a) `compiler/lib/compiler.go`, `config.go`, `parser/parser.go`, `starlark_loader.go`, `server/server.go`, `mod/command.go` are byte-identical to the prior verification's digest (confirmed via `git log` showing zero commits touching them since), and (b) the full-suite `-race` re-run just executed exercises every test file in that set and came back green, which would have caught a regression even without re-deriving each claim from scratch.

## Goal Achievement

### Roadmap Success Criteria (the binding contract)

| # | Criterion | Status | Evidence |
|---|-----------|--------|----------|
| 1 | Compiling a config no longer incurs a delay proportional to total repo proto count | ✓ VERIFIED | Re-run live: `protoFilesLoaded=5` against a 50-proto-on-disk corpus |
| 2 | Requesting the same proto file twice costs one parse — the second is a map lookup | ✓ VERIFIED | Carried forward (unchanged file); re-confirmed green in this session's full `-race` run of `./compiler/lib/parser/...` |
| 3 | `protoconf mod sync` still writes a `.fds` cache file identical in content to before this change | ✓ VERIFIED | `TestModSyncFdsByteIdentical` carried forward (unchanged file, re-confirmed green in the full-suite run); `TestModSyncNeverPersistsEmptyDescriptorSet` re-run live: 3/3 PASS |
| 4 | An operator can see, from compiler output, how many proto files a compile loaded | ✓ VERIFIED | Live CLI re-run: `compile finished file=test.pconf protoFilesLoaded=1 eagerFallback=false` |
| 5 | A custom gRPC mutation service defined under `src/` is registered and reachable at server startup, before any config has been compiled | ✓ VERIFIED | Re-run live: `TestInitRegistersCustomService`/`TestInitWithNoCustomServices`/`TestInitServiceSetIsOrderIndependent`, all PASS under `-race` |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| LAZY-01 | 11-01 | `GetProtoRegistry()` stops bulk-parsing `src/` | ✓ SATISFIED | Roadmap SC1, above |
| LAZY-02 | 11-01, 11-03 | On-demand parse, memoised, concurrency-safe | ✓ SATISFIED | Roadmap SC2, above |
| LAZY-03 | 11-01 | On-demand parsing never mutates `localFiles` | ✓ SATISFIED | `TestLazyParseDoesNotMutateLocalFiles` present and green in the full-suite run |
| LAZY-04 | 11-03, 11-04, 11-05 | `mod sync` still writes an identical `.fds`, survives a concurrent lazy compile, never persists an empty descriptor set | ✓ SATISFIED | Roadmap SC3, above; `TestModInitLockFileShapes` (8/8) and `TestModSyncNeverPersistsEmptyDescriptorSet` (3/3) re-run live |
| LAZY-05 | 11-01 | Operator-visible loaded-proto count | ✓ SATISFIED | Roadmap SC4, above |
| CONS-01 | 11-02 | Custom gRPC service catalog survives the lazy switch | ✓ SATISFIED | Roadmap SC5, above |

`.planning/REQUIREMENTS.md`'s Traceability table lists exactly these 6 IDs against "Phase 11 / Complete" (lines 120-125), matching the union of `requirements:` fields declared across all five plans (re-confirmed by grep in this session). No orphaned requirements.

### Human Verification Item — Now Closed

The prior verification's sole `human_needed` blocker was WR-02 from `11-REVIEW.md`: `LocalFileCount()` (`utils/utils.go`, added by 11-05 to back the G-11-7 empty-descriptor-set guard) took `d.mu.RLock()`, but `Parse()` — the only production writer of `localFiles` — wrote it with no lock at all, so the RLock advertised a synchronization guarantee that did not exist.

**Decision recorded (`11-UAT.md` test 8, 2026-09-07):** Drop `LocalFileCount()`'s `RLock`/`RUnlock` and document the constraint, rather than extend `d.mu` to cover `Parse`'s writes.

**Verified live against the current code** (`utils/utils.go`):

```go
func (d *DescriptorRegistry) LocalFileCount() int {
	return len(d.localFiles)
}
```

with the doc comment immediately above stating plainly: *"NOT safe to call concurrently with Import/Parse on the same registry... Every caller today runs on the goroutine that just finished Import/Parse, which is what makes the read correct,"* followed by a `ponytail:` marker naming the real ceiling and upgrade path: *"safe only because Sync's walk is serial and Parse resets localFiles per dependency. Parallelising that walk needs a registry per dependency, not a lock here — a lock would silence the race detector while leaving the reset to clobber a sibling's entries, which would make the G-11-7 guard read 0 for a dependency that parsed fine."*

**This does not weaken the G-11-7 guard.** Confirmed live: `GenFileDescriptorSet` (`compiler/lib/module_service.go:378`) calls `registry.LocalFileCount()` synchronously, on the same goroutine, immediately after `registry.Import(registry.Parse, ...)` returns (lines 361-378) — and `ModuleService.Walk` → the package-level `walk()` function recurses with no goroutines, so `GenFileDescriptorSet` is never invoked concurrently with itself on the same registry in the production call graph today. The RLock being dropped removes a false safety claim, not real protection — there was never a second lock-holder to race against on this path.

**Sanity-check of the rejected alternative (lock `Parse`'s writes instead):** Agrees with the recorded reasoning. `Sync()` (`compiler/lib/module_service.go:490-497`) builds exactly ONE `*utils.DescriptorRegistry` outside its walk and passes the same pointer into every `GenFileDescriptorSet` call across all dependencies; `Parse()` unconditionally resets `d.localFiles = map[string]struct{}{}` on entry (line 211). If `Sync`'s walk were ever parallelized without also giving each dependency its own registry, wrapping `Parse`'s writes in `d.mu.Lock()` would make `-race` go quiet while a second dependency's `Parse` call reset and repopulated `localFiles` mid-flight of a sibling's `LocalFileCount()` read — silently making the G-11-7 guard observe 0 for a dependency that actually parsed fine, which is a correctness regression, not just a data race. A lock only serializes access to the *field*; it does nothing to protect the *per-dependency semantics* `LocalFileCount()` depends on, since every dependency shares one `localFiles` map that resets on every `Parse` call. The documented ceiling (registry-per-dependency as the real fix, not a lock) is the correct framing.

**Verdict:** WR-02 is resolved as described. No remaining human verification items.

### UAT Bookkeeping Note (non-blocking)

`11-UAT.md` test 7's `result:` field still reads `issue` even though the gap it produced (G-11-7) is separately marked `status: resolved`, `resolved_by: 11-05-PLAN.md` in the `## Gaps` section. This is an accurate historical record, not a stale/incorrect field: test 7, as actually run on 2026-09-07, did find prohibition 6 violated — the `result: issue` field documents what that specific test run observed, while the `## Gaps` section separately tracks the resulting gap's resolution lifecycle. The two fields serve different purposes (point-in-time test outcome vs. gap disposition) and are not in conflict. No correction needed.

### Out-of-Scope Defect Noted, Not Fixed (non-blocking)

`walk()` (`compiler/lib/module_service.go:541-554`) sorts `keys` (the map keys of `head.GetDeps()`) but then indexes the *unsorted* `deps` slice (built in the same map-iteration pass, before the sort) with the sorted index — so `sort.Strings(keys)` has no effect on which `RemoteRepo` is visited at each position; the walk order remains as nondeterministic as Go map iteration. Confirmed live by reading the code:

```go
keys := []string{}
deps := []*module.RemoteRepo{}
for k, dep := range head.GetDeps() {
	keys = append(keys, k)
	deps = append(deps, dep)
}
sort.Strings(keys)          // sorts keys...
for i := range keys {
	err = errors.Join(err, walk(deps[i], walkFn))  // ...but indexes deps, which was never reordered
}
```

**Does not affect any must-have truth.** T-11-23's error accumulation (`errors.Join`) is order-independent by construction — confirmed by reading the accumulation itself, which only joins errors regardless of sequence. It is the documented root cause of the "order-dependent red signal" noted in `11-05-SUMMARY.md`'s mutation-testing notes, and `11-SECURITY.md` records it as "Observation, out of scope, not a threat." Recorded here as an Info-level anti-pattern finding so it isn't lost; a real bug (the sort is dead code), but it fails no roadmap success criterion, no LAZY-01..05/CONS-01 requirement, and no plan-declared must-have — it affects visit *order* determinism, not correctness of any asserted outcome. Not a blocker.

### Code Review Findings Weighed (11-REVIEW.md: 0 critical, 2 warning, 3 info — advisory, does not gate)

- **WR-01** (`ParseAll` holds `d.mu` across its whole-tree parse): confirmed still present at `utils/utils.go:352-354`. Explicitly accepted as documented risk, decision recorded in `11-UAT.md` test 5 ("lock duration is a performance property, not a correctness one, and fails no must-have truth"). Resolved, not re-opened.
- **WR-02**: resolved — see above.
- **IN-01, IN-02, IN-03** (info, no fix required): `modTidyCommand.Run`'s redundant `LoadFromLockFile` call, a stale doc comment, and a duplicate `src/` parse on mutation-server startup defending against a scenario the current architecture cannot yet reach. None affect correctness or any must-have truth.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `utils/utils.go` | `ParseOne`/`ParseAll`/`LoadedFileCount`/`LocalFileCount`/`FellBackToEager` | ✓ VERIFIED | All present; `LocalFileCount` confirmed updated (RLock dropped, doc comment + `ponytail:` marker added, commit `3005e5a`) |
| `compiler/lib/module_service.go` | `NewLazyModuleService`, `GetProtoRegistry` double-checked lock, `GenFileDescriptorSet` empty-set guard, `walk()` accumulator, `LoadFromLockFile`/`Init`/`MergeLock` invariant fixes | ✓ VERIFIED | All confirmed present by direct reading; unchanged since prior verification |
| `mod/command.go`, `mod/command_test.go` | `modInitCommand`/`modTidyCommand`/`modSyncCommand`, `TestModInitLockFileShapes`, `TestModSyncNeverPersistsEmptyDescriptorSet` | ✓ VERIFIED | Confirmed present and green (re-run live) |
| `server/server.go` | `Init`'s throwaway discovery registry, unregistrable-service warning | ✓ VERIFIED | Confirmed at `server/server.go:325-360`; unchanged since prior verification |
| `compiler/lib/mod_sync_fds_test.go` | `TestModSyncFdsByteIdentical`, `TestModSyncFdsUnaffectedByConcurrentLazyCompile` | ✓ VERIFIED | Both present and green (confirmed in the full-suite `-race` re-run) |
| `.planning/phases/.../11-SECURITY.md` | Threat register for the phase | ✓ VERIFIED | Present, `threats_open: 0`, 28 threats all closed (16 mitigated, 12 accepted), sign-off complete |

### Behavioral Spot-Checks (live, re-run in this session)

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Compile loads only demanded protos | `go test ./compiler/lib/... -run TestGeneratedCorpusCompiles -v` | `protoFilesLoaded=5` (50-proto corpus) | ✓ PASS |
| Operator-visible loaded-count line | `go run ./cmd/protoconf compile utils/testdata/small test.pconf` | `compile finished file=test.pconf protoFilesLoaded=1 eagerFallback=false` | ✓ PASS |
| G-11-3 regression suite | `go test ./mod/... -run TestModInitLockFileShapes -v` | 8/8 PASS | ✓ PASS |
| G-11-7 regression suite | `go test ./mod/... -run TestModSyncNeverPersistsEmptyDescriptorSet -v -count=1` | 3/3 PASS | ✓ PASS |
| CONS-01 regression suite | `go test ./server/... -run 'TestInitRegistersCustomService\|TestInitWithNoCustomServices\|TestInitServiceSetIsOrderIndependent' -v -race` | 3/3 PASS | ✓ PASS |
| WR-02 fix live in code | Direct read of `utils/utils.go` | RLock dropped, doc comment + `ponytail:` marker present | ✓ PASS |
| `go vet` clean | `go vet ./utils/... ./compiler/... ./mod/... ./server/...` | No output | ✓ PASS |
| Full targeted suite under `-race` | `go test -race -count=1 ./utils/... ./compiler/... ./mod/... ./server/...` | All packages `ok`, no `FAIL`, no `WARNING: DATA RACE` | ✓ PASS |

### Anti-Patterns Found

None blocking. Scanned `utils/utils.go` and `.planning/phases/11-concurrency-safe-lazy-registry-core/11-UAT.md` (the two files changed since the prior verification) for `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` — zero matches. The one `ponytail:` marker present is a deliberate, explicitly-labeled simplification with a named upgrade path (per project convention), not a debt marker requiring a tracked issue. One Info-level finding carried into this report: the `walk()` sort/index mismatch (see "Out-of-Scope Defect Noted, Not Fixed," above) — real but non-blocking, fails no must-have truth.

### Documentation Ledger Note (non-blocking, carried forward)

`.planning/REQUIREMENTS.md`'s `BUG-03` entry ("`go vet` copylocks at `compiler/lib/compiler.go:355`") is still nested under `## Future Requirements` (line 82), even though the fix is real and confirmed here (`go vet` clean of copylocks repo-wide). Documentation-only inconsistency; does not affect this phase's code goal.

### Gaps Summary

None. All five roadmap success criteria, all six requirement IDs, and both UAT-discovered gaps (G-11-3, G-11-7) are verified live against the current working tree. The one item that kept the prior verification at `human_needed` — WR-02's lock-discipline inconsistency in `LocalFileCount()` — has been explicitly decided by the user, implemented (commit `3005e5a`), and independently confirmed in this session both by direct code reading and by a live `-race` run across every phase-touched package. No new blocking findings surfaced. Two non-blocking observations are carried in this report for completeness: the `walk()` sort/index mismatch (pre-existing, out of scope, fails no must-have) and the UAT test-7 `result: issue` field (accurate historical record, not a live gap).

---

_Verified: 2026-09-07T21:50:00Z_
_Verifier: Claude (gsd-verifier)_

---

## Post-Verification Digest Refresh (orchestrator, 2026-09-07)

`covered_digest` was recomputed via `gsd_run query verification.fingerprint` (never hand-written)
over the identical 25-file `covered_files` list, after this report's verdict was reached.

**What changed after the verifier ran, and why the verdict is unaffected:**

`git diff --name-only <verifier commit 330fdc0>..HEAD` returns exactly one path:
`.planning/phases/11-concurrency-safe-lazy-registry-core/11-UAT.md`. No source file, PLAN, SUMMARY,
REQUIREMENTS or REVIEW entry moved.

The single edit was UAT test 7's `result:` field, `issue` -> `pass`, on the user's explicit
decision (AskUserQuestion, "Re-test as passed"). This report's own body already assessed that
field and found the `issue` value defensible as historical record; the user chose to re-test it
instead, on the grounds that prohibition 6 -- the only one of the seven that failed -- has its
falsifying condition fixed by 11-05 and now covered at the CLI level by
`TestModSyncNeverPersistsEmptyDescriptorSet`, re-run live at 3/3 PASS including
`good_path_control`. The original finding is preserved verbatim in the test's `reported:` field
and in the full `G-11-7` gap entry; a `retested:` field records the basis for the change.

The refresh exists because `11-UAT.md` is itself a covered file, so any UAT write after
verification marks the report stale -- and UAT necessarily completes after verification in this
workflow. Recomputing the digest records that the verdict covers the current contents. It is NOT
a re-verification: no truth was re-derived here, and none needed to be, because no verified
artifact other than that one UAT result field changed.

---

## Re-Verification After Downstream Drift (2026-09-08)

`verification.status` reported `stale`. The cause is **not** a Phase 11 artifact change — it is
`covered_digest` drift from downstream phases. `git diff --stat 5800eaf..HEAD` over the 25-file
`covered_files` list shows 8 changed paths, every one of them rewritten by Phase 12 or Phase 13:
`utils/utils.go`, `compiler/lib/parser/parser.go`, `compiler/lib/compiler.go`,
`compiler/lib/starlark_loader.go`, `compiler/lib/config.go`, `compiler/lib/module_service.go`,
`server/server.go`, and `.planning/REQUIREMENTS.md`. No Phase 11 PLAN, SUMMARY, UAT, REVIEW or
SECURITY file moved.

The digest cannot distinguish "Phase 11's own work changed" from "a later phase edited a shared
file", so the drift is a signal to re-check, not a verdict. It was re-checked.

**Two Phase 11 test files were deliberately deleted by 13-03**, together with the code they
covered:

| Deleted | Covered | Replaced by |
|---------|---------|-------------|
| `utils/parse_all_deadlock_test.go` | WR-01 — `ParseAll` holds `d.mu` across the whole-tree parse without deadlocking | `utils/index_build_deadlock_test.go` (same deadlock property against the symbol-index build that replaced `ParseAll`) |
| `compiler/lib/eager_fallback_visible_test.go` | D-03's `eagerFallback` log attribute is always present | `compiler/lib/tier_observability_test.go` (pins the three tier counters that replaced the boolean) |

`ParseAll` and the `eagerFallback` latch no longer exist (`feat(13-03): delete the whole-tree
eager fallback, replace with a hard error`, `862410e`). UAT item 5's WR-01 lock-duration decision
and UAT item 7's prohibition 2 are therefore **moot at HEAD, not regressed** — the mechanism they
constrained was removed. `utils/lazy_parse_canonical_test.go` survived the same commit with its
seam re-pointed at a direct locked insert; the LAZY-02 pointer-identity truth it carries is intact.

**Re-run live at HEAD (2026-09-08):**

- `go test -race -count=1 ./utils/... ./compiler/lib/... ./server/... ./mod/...` — all packages
  `ok`, no `FAIL`, no `WARNING: DATA RACE`.
- Every named Phase 11 evidence test, individually under `-race`, all PASS:
  `TestParseOneReturnsCanonicalPointerWhenRacingConcurrentInsert` (LAZY-02 / WR-02),
  `TestGetProtoRegistryIsSingletonUnderConcurrency` (WR-04),
  `TestInitServiceSetIsOrderIndependent` (SC5 / CONS-01, backstop truth B2),
  `TestModSyncFdsByteIdentical` + `TestModSyncFdsUnaffectedByConcurrentLazyCompile` (SC3 / LAZY-04),
  `TestModSyncNeverPersistsEmptyDescriptorSet` 3/3 including `good_path_control` (G-11-7).
- SC4 live CLI: `go run ./cmd/protoconf compile utils/testdata/small test.pconf` →
  `compile finished file=test.pconf protoFilesLoaded=1 symbolIndexBuilds=0 symbolIndexCacheHits=0
  scanResolutions=0`. The operator-visible loaded-proto count survives; Phase 13 replaced the
  `eagerFallback` boolean beside it with three tier counters, which is SC4-compatible.
- `go vet ./utils/... ./compiler/... ./mod/... ./server/...` — clean.

**Verdict unchanged: `passed`.** No Phase 11 truth regressed under Phases 12–13; two were rendered
moot by an intentional, documented deletion. `covered_digest` recomputed via
`gsd_run query verification.fingerprint` over the same 25-file list — never hand-written.
