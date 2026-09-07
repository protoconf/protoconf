---
phase: 11-concurrency-safe-lazy-registry-core
verified: 2026-09-07T21:15:00Z
status: human_needed
score: 25/25 must-have truths verified (5 roadmap success criteria + 6 requirement IDs + 14 gap-closure plan truths across 11-04/11-05), plus 6 pre-existing backstop/decision items closed via 11-UAT.md human sign-off
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
covered_digest: "v1:sha256:1a1921fd5db575fa17e8ef665e83a67d85bc350b90422806d6418166e7d7900d"
re_verification:
  previous_status: human_needed
  previous_score: 18/18 must-have truths (plus 5 roadmap success criteria)
  gaps_closed:
    - "G-11-3 (protoconf.lock nil-map panic on mod init / mod tidy) — closed by 11-04, re-run live: TestModInitLockFileShapes 8/8 subtests PASS"
    - "G-11-7 (mod sync silently persists a zero-byte .fds / empty checksum over protoconf.lock) — closed by 11-05, re-run live: TestModSyncNeverPersistsEmptyDescriptorSet 3/3 subtests PASS"
    - "WR-02 (2026-09-04 code review, ParseOne non-canonical pointer race with ParseAll) — FIXED commit 0713f86, confirmed present in code (afterParseHook seam, canonical FileRegistry lookup)"
    - "WR-04 (2026-09-04 code review, ModuleService.cachedRegistry unsynchronized check-then-act) — FIXED commit 84efad0, confirmed present in code (double-checked locking under m.mutex)"
    - "Backstop truth B2 (11-02, Init's service set order-independence) — now has executed evidence: server/init_order_test.go TestInitServiceSetIsOrderIndependent, re-run live, PASS"
    - "Backstop truth B3 (11-03, mod sync immune to concurrent in-process lazy compile) — now has executed evidence: compiler/lib/mod_sync_fds_test.go TestModSyncFdsUnaffectedByConcurrentLazyCompile, re-run live under -race, PASS"
  gaps_remaining: []
  regressions: []
human_verification:
  - test: "WR-02 (11-REVIEW.md, new finding, 2026-09-07T21:10): LocalFileCount()'s d.mu.RLock() does not synchronize with localFiles's only production writer (Parse, which writes with no lock at all)"
    expected: "A decision on whether to extend mu's documented scope to genuinely cover localFiles (lock Parse's writes too), or drop LocalFileCount's RLock/RUnlock and document plainly that it is unsafe to call concurrently with Import/Parse on the same registry — so the code's safety claim matches what callers can actually rely on"
    why_human: "Confirmed by reading utils/utils.go: LocalFileCount() (utils/utils.go:390-394, added by 11-05 to back the G-11-7 guard) takes d.mu.RLock(), but Parse() (utils/utils.go:210-222), the sole production writer of localFiles, writes it with zero locking. Not a live bug today — GenFileDescriptorSet calls Import/Parse to completion before calling LocalFileCount() synchronously on the same goroutine, and Sync()'s dependency walk is single-threaded — so -race cannot catch it until a future change (e.g. parallelizing Sync()'s walk, a plausible follow-up given this phase's own concurrency-hardening work) actually drives the interleaving. This is the same 'latent, not yet live' class of issue WR-04 was before it became a real bug and was fixed in this same phase (84efad0). No must-have truth in any of the 5 plans covers LocalFileCount's lock discipline, and this finding postdates 11-UAT.md (UAT completed 2026-09-07T19:40Z; this review finding is timestamped 2026-09-07T21:10Z), so it has not yet had a human verification pass."
---

# Phase 11: Concurrency-Safe Lazy Registry Core Verification Report

**Phase Goal:** A compile no longer pays for the whole repository's proto tree, and the mutation server's service catalog doesn't silently go dark under that change.
**Verified:** 2026-09-07T21:15:00Z
**Status:** human_needed
**Re-verification:** Yes — after gap closure (11-04 closed G-11-3, 11-05 closed G-11-7; both independently re-run live in this session, not accepted from SUMMARY.md claims)

## Goal Achievement

### Roadmap Success Criteria (the binding contract)

All five re-run live in this session against the current working tree (not accepted from any prior VERIFICATION.md or SUMMARY.md claim).

| # | Criterion | Status | Evidence |
|---|-----------|--------|----------|
| 1 | Compiling a config no longer incurs a delay proportional to total repo proto count | ✓ VERIFIED | `go test ./compiler/lib/... -run TestGeneratedCorpusCompiles -v` re-run live: `compile finished file=main.mpconf protoFilesLoaded=5` against a 50-proto-on-disk corpus |
| 2 | Requesting the same proto file twice costs one parse — the second is a map lookup | ✓ VERIFIED | `go test ./compiler/lib/parser/... -run TestParseMemoization -v -race` re-run live: PASS |
| 3 | `protoconf mod sync` still writes a `.fds` cache file identical in content to before this change | ✓ VERIFIED | `go test ./compiler/lib/... -run 'TestModSyncFdsByteIdentical\|TestModSyncFdsUnaffectedByConcurrentLazyCompile' -v -race` re-run live: both PASS. Second test (11-04's gap closure) additionally proves byte-identity holds under a concurrent in-process lazy compile |
| 4 | An operator can see, from compiler output, how many proto files a compile loaded | ✓ VERIFIED | Live CLI: `go run ./cmd/protoconf compile utils/testdata/small test.pconf` → `INFO compile finished file=test.pconf protoFilesLoaded=1 eagerFallback=false` |
| 5 | A custom gRPC mutation service defined under `src/` is registered and reachable at server startup, before any config has been compiled | ✓ VERIFIED | `go test ./server/... -run 'TestInitRegistersCustomService\|TestInitWithNoCustomServices\|TestInitServiceSetIsOrderIndependent' -v -race` re-run live: all PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| LAZY-01 | 11-01 | `GetProtoRegistry()` stops bulk-parsing `src/` | ✓ SATISFIED | Roadmap SC1, above |
| LAZY-02 | 11-01, 11-03 | On-demand parse, memoised, concurrency-safe | ✓ SATISFIED | Roadmap SC2, above; `TestConcurrentCompile` (11-03) confirmed present and green under `-race` in the full-suite run below |
| LAZY-03 | 11-01 | On-demand parsing never mutates `localFiles` | ✓ SATISFIED | `TestLazyParseDoesNotMutateLocalFiles` confirmed present in `compiler/lib`; green in the full-suite run below |
| LAZY-04 | 11-03, 11-04, 11-05 | `mod sync` still writes an identical `.fds`, survives a concurrent lazy compile, and never persists an empty descriptor set | ✓ SATISFIED | Roadmap SC3, above; `TestModInitLockFileShapes` (8/8) and `TestModSyncNeverPersistsEmptyDescriptorSet` (3/3) re-run live, all PASS |
| LAZY-05 | 11-01 | Operator-visible loaded-proto count | ✓ SATISFIED | Roadmap SC4, above |
| CONS-01 | 11-02 | Custom gRPC service catalog survives the lazy switch | ✓ SATISFIED | Roadmap SC5, above |

`.planning/REQUIREMENTS.md`'s Traceability table lists exactly these 6 IDs against "Phase 11 / Complete" (lines 120-125), matching the union of `requirements:` fields declared across all five plans. No orphaned requirements.

### Gap Closure Verification (G-11-3, G-11-7 — re-run live, not accepted from SUMMARY.md)

Both gaps were found by `11-UAT.md` (human-in-the-loop UAT, 2026-09-04–2026-09-07), explicitly recorded as pre-existing defects on `b69e3b2` (before Phase 11 began) that the user directed be closed under this phase.

| Gap | Plan | Truth | Status | Evidence |
|-----|------|-------|--------|----------|
| G-11-3 | 11-04 | `mod init`/`mod tidy` survive every `protoconf.lock` shape (missing `deps` key, empty object, absent file, corrupt/truncated/unknown-key file) without a nil-map panic, and never regenerate a lock file that failed to parse | ✓ VERIFIED | `go test ./mod/... -run TestModInitLockFileShapes -v` re-run live: 8/8 subtests PASS (`no_deps_key`, `empty_object`, `explicit_empty_deps`, `absent`, `zero_byte`, `truncated_json`, `unknown_key`, `merges_existing_entry`) |
| G-11-3 (backstop) | 11-04 | `mod sync`'s serialized `.fds` bytes are unmoved by a concurrent in-process lazy compile | ✓ VERIFIED | `go test ./compiler/lib/... -run TestModSyncFdsUnaffectedByConcurrentLazyCompile -v -race` re-run live: PASS |
| G-11-7 | 11-05 | `mod sync` over an unsynced dependency (no `getterUrl`) fails non-zero, writes no `.fds`, leaves the recorded `fileDescriptorSetSum` intact | ✓ VERIFIED | `go test ./mod/... -run TestModSyncNeverPersistsEmptyDescriptorSet -v -count=1` re-run live: `unsynced_dep_no_getter_url` PASS |
| G-11-7 | 11-05 | The same guard catches a downloaded dependency with a bad `sourcePath` (not keyed on `getterUrl`), and a failed dependency's error survives `walk()`'s accumulation | ✓ VERIFIED | Same run: `downloaded_dep_bad_source_path` PASS; live-read `walk()` in `compiler/lib/module_service.go` confirms `err = errors.Join(err, walk(deps[i], walkFn))` (accumulates, does not replace) |
| G-11-7 | 11-05 | Good path unchanged — `mod init` + `mod sync` over the fixture restores the committed checksums | ✓ VERIFIED | Same run: `good_path_control` PASS |

Both gaps' root causes were independently confirmed in the current code, not merely by test presence: `LoadFromLockFile` (`compiler/lib/module_service.go`) re-initializes `m.head.Deps` on every return path; `Init` self-loads the lock file first; `GenFileDescriptorSet` guards on `registry.LocalFileCount() == 0` before any `Store` call (`compiler/lib/module_service.go`); `utils.DescriptorRegistry.LocalFileCount()` exists in `utils/utils.go` as documented.

### Code Review Findings Weighed (11-REVIEW.md: 0 critical, 2 warning, 3 info — advisory, does not gate)

Independently re-read against the current code in this session, per instruction, not taken on faith:

- **WR-01** (carried forward from the original 2026-09-04 review, still open): `ParseAll` holds `d.mu` across its entire whole-tree parse — confirmed still present at `utils/utils.go:352-373` (`d.mu.Lock(); defer d.mu.Unlock()` wraps the full `d.Import` call). Not deadlock-prone (backstopped by `utils/parse_all_deadlock_test.go`, re-run live: PASS). This was already explicitly decided by the human in `11-UAT.md` test 5 ("decision: DEFERRED as accepted, documented risk — lock duration is a performance property, not a correctness one, and fails no must-have truth"). **Treated as resolved**, not re-opened here.
- **WR-02** (new, 2026-09-07T21:10, postdates `11-UAT.md`'s 2026-09-07T19:40 completion): `LocalFileCount()`'s `d.mu.RLock()` does not synchronize with `localFiles`'s only production writer (`Parse`, unlocked). Confirmed present by direct reading of `utils/utils.go:390-394` and `utils/utils.go:210-222`. Not a live bug today (single-goroutine call graph), but genuinely unresolved and never seen by a human verification pass — this is the one item routed to `human_verification` below, which is why overall status is `human_needed` rather than `passed`.
- **IN-01, IN-02, IN-03** (info, no fix required): `modTidyCommand.Run`'s redundant `LoadFromLockFile` call, a stale doc comment on the lazy-registry branch, and a duplicate `src/` parse on mutation-server startup that defends against a scenario the current architecture cannot yet reach. None affect correctness or any must-have truth; recorded for completeness, not gating.

### Backstop Truths — Now Closed (was `human_needed` in the previous verification)

| # | Truth | Prior status | Current status | Evidence |
|---|-------|--------------|-----------------|----------|
| B1 | `NewCompiler`/`NewLazyModuleService` succeed against `src/` with zero or one `.proto` file (11-01) | ⚠️ insufficient_spec (human_needed) | Resolved via `11-UAT.md` test 1 human sign-off (`result: pass`) | Still no dedicated automated fixture (`utils/testdata/corpus.go`'s `GenerateCorpus` still refuses `n<5`, confirmed unchanged); a human explicitly reviewed and passed this in UAT. Not re-opened. |
| B2 | `Init`'s registered service set is order-independent across `RangeFiles` iteration orders (11-02) | ⚠️ insufficient_spec (human_needed) | ✓ VERIFIED (upgraded to executed evidence) | `server/init_order_test.go` `TestInitServiceSetIsOrderIndependent`, re-run live: 20 repeated `Init` runs, both orderings observed, service SET equality asserted — PASS |
| B3 | `mod sync` immune to a concurrent in-process lazy compile (11-03) | ⚠️ insufficient_spec (human_needed) | ✓ VERIFIED (upgraded to executed evidence) | `TestModSyncFdsUnaffectedByConcurrentLazyCompile` (11-04), re-run live under `-race`: PASS |
| WR-01 (2026-09-04) | `ParseAll` lock-duration inconsistency — decision needed | human_needed | Resolved — human decision recorded in `11-UAT.md` test 5 (accept as documented risk) | See above |
| WR-02 (2026-09-04, ParseOne/ParseAll pointer race) | Decision needed | human_needed | ✓ VERIFIED (FIXED) | Commit 0713f86, confirmed in code: `afterParseHook` seam, canonical `FileRegistry` lookup with `!ok` fallback |
| WR-04 (2026-09-04, `cachedRegistry` unsynchronized) | Decision needed | human_needed | ✓ VERIFIED (FIXED) | Commit 84efad0, confirmed in code: double-checked locking under `m.mutex` in `GetProtoRegistry` |
| 7 plan-declared `verification: manual` prohibitions (11-01/02/03) | Deferred to human | human_needed | Resolved via `11-UAT.md` test 7 (6/7 held; prohibition 6 violated, tracked as G-11-7, now closed above) | Per-prohibition disposition recorded in `11-UAT.md` |

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `utils/utils.go` | `ParseOne`/`ParseAll`/`LoadedFileCount`/`LocalFileCount`/`FellBackToEager` | ✓ VERIFIED | All present; `LocalFileCount` (new in 11-05) confirmed at lines 390-394 |
| `compiler/lib/module_service.go` | `NewLazyModuleService`, `GetProtoRegistry` double-checked lock, `GenFileDescriptorSet` empty-set guard, `walk()` accumulator, `LoadFromLockFile`/`Init`/`MergeLock` invariant fixes | ✓ VERIFIED | All confirmed present by direct reading |
| `mod/command.go`, `mod/command_test.go` | `modInitCommand`/`modTidyCommand`/`modSyncCommand`, `TestModInitLockFileShapes`, `TestModSyncNeverPersistsEmptyDescriptorSet` | ✓ VERIFIED | Confirmed present and green (re-run live) |
| `server/server.go` | `Init`'s throwaway discovery registry, unregistrable-service warning | ✓ VERIFIED | Confirmed at `server/server.go:325-360`; reflection registrations still read `s.parser.FilesResolver`/`LocalResolver` (D-02 boundary held) |
| `compiler/lib/mod_sync_fds_test.go` | `TestModSyncFdsByteIdentical`, `TestModSyncFdsUnaffectedByConcurrentLazyCompile` | ✓ VERIFIED | Both present and green (re-run live under `-race`) |

### Behavioral Spot-Checks (live, run in this session)

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Compile loads only demanded protos | `go test ./compiler/lib/... -run TestGeneratedCorpusCompiles -v` | `protoFilesLoaded=5` (50-proto corpus) | ✓ PASS |
| Operator-visible loaded-count line | `go run ./cmd/protoconf compile utils/testdata/small test.pconf` | `compile finished file=test.pconf protoFilesLoaded=1 eagerFallback=false` | ✓ PASS |
| G-11-3 regression suite | `go test ./mod/... -run TestModInitLockFileShapes -v` | 8/8 PASS | ✓ PASS |
| G-11-7 regression suite | `go test ./mod/... -run TestModSyncNeverPersistsEmptyDescriptorSet -v -count=1` | 3/3 PASS | ✓ PASS |
| CONS-01 regression suite | `go test ./server/... -run 'TestInitRegistersCustomService\|TestInitWithNoCustomServices\|TestInitServiceSetIsOrderIndependent' -v -race` | 3/3 PASS | ✓ PASS |
| `go vet ./...` clean of copylocks | `go vet ./...` | Only pre-existing, unrelated findings in `test/e2e_test.go`, `agent/agent_test.go`, `agent/legacy.go` (context-leak / unreachable-code lints, none in phase-touched files) | ✓ PASS |
| Full non-agent suite under `-race` | `go test -race -count=1 $(go list ./... \| grep -v '/agent$')` | All 24 packages `ok`, no `FAIL`, no `WARNING: DATA RACE` | ✓ PASS |

### Anti-Patterns Found

None. Scanned all 17 phase-touched files (`utils/utils.go`, `compiler/lib/{module_service,module_service_test,compiler,config,starlark_loader,mod_sync_fds_test}.go`, `compiler/lib/parser/parser.go`, `mod/{command,command_test}.go`, `server/{server,init_order_test,init_prohibitions_test}.go`, `utils/{lazy_parse_canonical_test,lazy_parse_error_test,parse_all_deadlock_test}.go`, `compiler/lib/{eager_fallback_visible_test,registry_cache_test}.go`) for `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` — zero matches.

### Documentation Ledger Note (non-blocking, carried forward from the prior verification)

`.planning/REQUIREMENTS.md`'s `BUG-03` entry ("`go vet` copylocks at `compiler/lib/compiler.go:355`") is still nested under the `## Future Requirements` heading (line 82), whose intro reads "Acknowledged, deferred beyond this milestone" (line 84), even though the fix is real and independently confirmed here (`go vet ./...` is clean of copylocks repo-wide). This is a documentation-only inconsistency, not a functional gap — it does not affect this phase's code goal — but should be corrected before the milestone closes so a future reader doesn't reintroduce the value-copy believing it unfixed.

### Human Verification Required

See the `human_verification` list in the frontmatter. One item:

1. **WR-02 (new, 2026-09-07T21:10 code review): `LocalFileCount()`'s lock discipline does not match `localFiles`'s actual write-side locking (none).** Not a live bug today — the current call graph is synchronous — but it postdates `11-UAT.md`'s human sign-off and has not itself been reviewed by a human. Needs a decision: extend `d.mu`'s documented scope to cover `Parse`'s writes to `localFiles`, or explicitly document `LocalFileCount()` as unsafe to call concurrently with `Import`/`Parse`.

### Gaps Summary

No must-have truth failed, no artifact is missing or a stub, no key link is unwired, and no requirement is unsatisfied. Every roadmap success criterion and every LAZY-01..05/CONS-01 requirement was independently re-run live in this session against the current working tree — not accepted from any SUMMARY.md or prior VERIFICATION.md claim. Both gaps found by `11-UAT.md` (G-11-3, G-11-7) are closed, with their regression suites re-run live and passing. Three of the four backstop/decision items left `human_needed` by the previous verification are now resolved (two upgraded to executed test evidence, two fixed in code, one explicitly accepted as documented risk by human decision in UAT).

What keeps this from a clean `passed`: the code review completed immediately before this verification (2026-09-07T21:10, after UAT's 2026-09-07T19:40 completion) surfaced one new, genuine, unfixed lock-discipline finding (WR-02) that has never had a human verification pass. It is not a live bug and fails no stated must-have truth, but per the honest-verifier contract it is surfaced for an explicit human decision rather than silently absorbed into a passing verdict.

---

_Verified: 2026-09-07T21:15:00Z_
_Verifier: Claude (gsd-verifier)_
