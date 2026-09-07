---
status: testing
phase: 11-concurrency-safe-lazy-registry-core
source: [11-VERIFICATION.md]
started: 2026-09-04T17:50:00Z
updated: 2026-09-07T21:20:00Z
---

## Current Test

number: 8
name: WR-02 (new) — LocalFileCount()'s RLock does not synchronize with localFiles's only writer
expected: |
  A decision on whether to extend mu's documented scope to genuinely cover localFiles
  (lock Parse's writes too), or drop LocalFileCount's RLock/RUnlock and document plainly
  that it is unsafe to call concurrently with Import/Parse on the same registry — so the
  code's safety claim matches what callers can actually rely on.
awaiting: user response

## Tests

### 1. Construction against a src/ tree with zero .proto files, and with exactly one

expected: NewCompiler / NewLazyModuleService succeed in both cases rather than panicking or erroring on an empty/near-empty tree.
why_human: No committed test exercises either edge case — `utils/testdata/corpus.go`'s `GenerateCorpus` refuses `n<5`, and no other fixture with 0 or 1 proto file appears anywhere in the diffed test files. This is a structured `verification: backstop` truth in 11-01's frontmatter; the honest-verifier contract requires it to abstain rather than be inferred from the general case reading correctly.
result: pass

### 2. Init registers an order-independent service set across repeated runs

expected: The registered service set does not vary with `protoregistry.Files.RangeFiles`' map iteration order; only registration *order* differs between runs, never *which* services end up registered.
why_human: No test runs `Init` multiple times or forces distinct map iteration orders. `grpc.Server.GetServiceInfo()` being map-backed is suggestive but is not executed evidence, which is what the `backstop` tag requires.
result: pass
evidence: server/init_order_test.go - TestInitServiceSetIsOrderIndependent. 20 repeated Init runs asserting registered-service SET equality while iteration order varies freely (both orderings observed in the run log). Two-service temp fixture, so the set is not trivially size-1 (the WR-05 trap). A require.Subset guard keeps the assertion from going vacuous. Mutation-verified: an early stop in the RangeFiles callback turns it red. Commit 75d4da6.

### 3. mod sync's .fds is immune to a concurrent in-process lazy compile

expected: Running `mod sync` concurrently with an in-process lazy `CompileFile` does not change the `.fds` bytes `mod sync` writes.
why_human: `TestModSyncFdsByteIdentical` exercises the eager mod-sync path in isolation, sequentially. No test runs a lazy compile and a mod-sync `Store()` concurrently in the same process to observe the claimed non-interference directly.
result: pass
prior_result: issue
prior_reported: "go run ./cmd/protoconf mod tidy -protoconfPath /tmp/uat-11/ -> panic: assignment to entry in nil map at compiler/lib/module_service.go:146 (ModuleService.Init), via mod/command.go:153 (modTidyCommand.Run)."
note: The reported panic was a DIFFERENT defect from this test's stated truth - it was hit while building the fixture, not while observing mod sync under a concurrent compile. That panic is now closed as gap G-11-3 by 11-04 (commit 243ab6f), which unblocked the fixture and let this truth be observed for the first time.
evidence: CLI-LEVEL validation on the real utils/testdata/small fixture (two local .tgz deps, fully offline), at user request - the in-process test alone was not accepted as sufficient.
  Fixture: `mod init` to populate getterUrl, then `mod sync` -> terraform_repo.fds 15,942,747B, vizceral_repo.fds 2,736B. Non-vacuous: vizceral's sum 039f1e1023250b34054894cc58bc8b2b matches the committed protoconf.lock's recorded fileDescriptorSetSum exactly, independently corroborating the fixture.
  Trials: 12 total (8 + 4 after a lock restore). Each trial deletes both .fds, then launches 12 lazy `protoconf compile` processes (6 before, 6 after) concurrently with `mod sync` against the SAME protoconf root, so both contend on the shared .protoconf_cache/. All 12 trials: both .fds byte-identical to golden (md5 + size). Compiles independently confirmed exit 0 and logging "module service loaded" (the lazy NewCompiler path).
  Negative control: stripping getterUrl makes sync write 0-byte .fds and the harness reports MISMATCH on both files - so a MATCH is a real signal, not an artifact of comparing nothing.
  Scope note: the CLI runs mod sync and compile as separate PROCESSES sharing .protoconf_cache/. The same-process half of the truth remains covered by TestModSyncFdsUnaffectedByConcurrentLazyCompile (compiler/lib/mod_sync_fds_test.go, PASS twice under -race -count=2). Together these cover both interference surfaces; neither alone does.

### 4. WR-02 — ParseOne can return a non-canonical descriptor pointer when racing ParseAll

expected: Every caller of `ParseOne` for a given path — regardless of interleaving with a concurrent `ParseAll` — observes the one canonical `FileRegistry` pointer, matching `ParseOne`'s documented pointer-identity contract.
why_human: A genuine code-review-identified race between `ParseOne`'s singleflight closure (which parses outside `d.mu`, per its own documented lock discipline) and `ParseAll` (which holds `d.mu` across its whole-tree parse, per WR-01). `TestParseMemoization`'s concurrency subtest only races `ParseOne` against `ParseOne`, never against `ParseAll`, so it cannot catch this interleaving. **Decision needed:** fix now (return the post-`recordFileLocked` canonical value — the reviewer's suggested one-line fix) or accept as scoped-out follow-up.
result: pass
evidence: FIXED in commit 0713f86. ParseOne now returns the canonical FileRegistry entry instead of its own freshly-parsed fds[0], keyed by GetName() with an !ok fallback so a key divergence cannot turn the fix into a nil return. New unexported afterParseHook seam forces the ParseAll interleaving deterministically; utils/lazy_parse_canonical_test.go - TestParseOneReturnsCanonicalPointerWhenRacingParseAll asserts the hook actually ran (no vacuous cache-fast-path pass) and that ParseOne, FileRegistry, and a subsequent ParseOne all agree on one pointer. Mutation-verified: restoring `return fds[0]` turns it red. go vet clean; utils + compiler green under -race.

### 5. WR-01 — ParseAll holds d.mu across its entire whole-tree parse

expected: A decision on whether this is acceptable given D-03's "fires at most once per registry" bound, or whether it should be fixed to match `ParseOne`'s discipline before the phase is considered closed.
why_human: Not deadlock-prone (confirmed: `Import`'s `LookupImport`/`Accessor` closures never re-enter `d.mu`), but it is a real lock-hygiene inconsistency *within the same file*, with a concrete stall scenario — one goroutine's D-03 miss blocking every concurrently-compiling file on a shared `*lib.Compiler` for the whole-tree parse's duration. No must-have truth asserts anything about `ParseAll`'s lock duration, so this fails no stated truth, but it goes directly to the phase's own concurrency-safety framing.
result: pass
decision: DEFERRED as accepted, documented risk - lock duration is a performance property, not a correctness one, and fails no must-have truth. ParseAll is the latched D-03 fallback guarded by d.eagerFallback, so the expensive path runs at most once per registry.
sync_map_considered: REJECTED. sync.Map gives per-key atomicity, but recordFileLocked updates FileRegistry + lazyLoaded + MessageRegistry as a unit then recurses over dependencies - per-key safety leaves that SET non-atomic (a reader could see FileRegistry populated while MessageRegistry does not know the type yet, a nastier cousin of WR-02). MessageRegistry is also a third-party type with its own RWMutex, eagerFallback is a once-guard not a map, and Import/Parse write FileRegistry directly on the shared eager path so changing its type reaches every eager consumer - the blast radius D-01 fenced off.
future_shape: if the critical section is ever shortened, mirror ParseOne - parse into a scratch registry with no lock held, then take the write lock only to merge. Cost is one extra registry allocation plus a benign double-fire window (singleflight is already imported in utils.go).
evidence: testable half covered by commit e8939cf - utils/parse_all_deadlock_test.go, TestParseAllConcurrentWithParseOneDoesNotDeadlock. ParseAll runs concurrently with 6 ParseOne calls over a corpus whose files import each other (so LookupImport is genuinely driven); all must finish inside 30s. Guards against a vacuous pass by asserting eagerFallback fired and LoadedFileCount >= corpus size, and re-checks pointer identity under contention. Mutation-verified: adding d.mu.RLock() to Import LookupImport deadlocks the run and the test fails on its timeout.

### 6. WR-04 — ModuleService.cachedRegistry is read/written without a lock

expected: A decision on whether this unenforced construction-order invariant needs a lock now (`module_service.go` already carries an unused `m.mutex` for exactly this purpose) or is acceptable as documented risk for a future direct-construction caller.
why_human: Not a live bug today (confirmed: `NewCompiler`'s single synchronous priming call happens-before every later concurrent reader), but it is precisely the class of implicit invariant this phase's own threat model (T-11-01) treats as high severity elsewhere. No must-have truth covers it.
result: pass
evidence: FIXED in commit 84efad0. GetProtoRegistry now uses double-checked locking on the m.mutex already present on the struct (RLock fast path, then Lock + re-check before building). Holding the write lock across the build is safe - Walk is walk(m.head, walkFn) and takes no lock, so no re-entrancy of the WR-01 kind. New test compiler/lib/registry_cache_test.go - TestGetProtoRegistryIsSingletonUnderConcurrency constructs the service directly rather than via NewCompiler (whose synchronous priming is exactly what hid the bug) and releases 16 goroutines together. Two independent detectors: require.Same is deterministic without -race, and -race flags the write/write on the field. Mutation-verified: restoring the bare check-then-act yields two distinct registry pointers and the test fails. Impact was not merely a race warning - the losing caller received an ORPHAN registry with its own FileRegistry, silently breaking LAZY-02 pointer identity across callers (same class as WR-02).

### 7. Seven plan-declared prohibitions tagged `verification: manual`

expected: Each holds under close reading, not merely under the tests that happen to pass — a proto that fails to parse on the lazy path is not silently swallowed into a successful compile; the D-03 eager fallback never fires invisibly; pre-existing fixtures/assertions were not weakened; a service `Init` cannot register does not vanish without a trace; the discovery scan does not back gRPC reflection; `mod sync` does not write a truncated `.fds`; a data race was not quieted by removing `-race` or coarsening a lock.
why_human: All seven are tagged `verification: manual` in the PLAN frontmatter — the plan authors themselves deferred these to human judgment. Independent evidence gathered during verification supports most (the D-03 fallback is logged via the `eagerFallback` field on the `compile finished` line, confirmed live; reflection registrations still read `s.parser.FilesResolver`/`LocalResolver`, confirmed by grep; `TestModSyncFdsByteIdentical` targets the truncated-`.fds` prohibition directly; no new lock or `-race`-stripping appears in the diff) — but none were exercised by a dedicated adversarial test, e.g. actually feeding a broken `.proto` through the lazy path and asserting the compile fails rather than silently materializing wrong output.
result: issue
reported: "CLI re-test of item 3 falsified prohibition 6's coverage claim: `protoconf mod sync` on a protoconf.lock with no getterUrl exits 0, writes zero-byte .fds files for every dep, and overwrites the lock's recorded fileDescriptorSetSum with the md5 of empty. Silent cache + lock corruption behind a success exit code."
severity: major
issue_scope: "Prohibition 6 only. Prohibitions 1, 2, 4, 5, 7 remain verified as recorded below; prohibition 4's own violation was already found and fixed in commit 60459de. Tracked as gap G-11-7 (pre-existing, reproduces on b69e3b2)."
finding: prohibition 4 did NOT hold. server.go logged registration but not non-registration - a service under src/ whose rpcs do not return protoconf.v1.ConfigMutationResponse was dropped by a bare `continue`, leaving the operator with no service and no explanation. CONS-01 stopped the catalog going dark at DISCOVERY; this was the same failure at ELIGIBILITY. FIXED in commit 60459de: a Warn naming the service, its file, and the required output type.
per-prohibition disposition:
  1 broken proto not silently swallowed -> TESTED (utils/lazy_parse_error_test.go): malformed proto must error, must not return a descriptor, and must not be memoised as loaded (a poisoned entry would make retry succeed).
  2 D-03 fallback never fires invisibly -> TESTED (compiler/lib/eager_fallback_visible_test.go): asserts the structured eagerFallback attribute on the "compile finished" record and that it is ALWAYS present, so absent and false stay distinguishable. Attribute, not formatted text.
  3 pre-existing fixtures/assertions not weakened -> NOT a unit test; diff-review property, covered by the code-review gate (11-REVIEW.md).
  4 a service Init cannot register does not vanish -> VIOLATED, FIXED, TESTED (server/init_prohibitions_test.go).
  5 discovery scan does not back gRPC reflection (D-02) -> TESTED (server/init_prohibitions_test.go): behavioural, not a grep. With a bare parser, registration must see the custom service while reflection must not know its file. Widening reflection turns it red on purpose.
  6 mod sync must not write a truncated .fds -> COVERAGE CLAIM FALSIFIED at CLI level during the item-3 re-test (2026-09-07). TestModSyncFdsByteIdentical builds its registry via an explicit Import over a corpus that EXISTS, so it never reaches the missing-path branch. Live CLI: on a protoconf.lock with no getterUrl, `mod sync` exits 0, prints "Parsing protos."/"Storing in cache.", writes a ZERO-BYTE .fds for every dep, and overwrites the lock's real fileDescriptorSetSum (6556e5cf.../039f1e10...) with d41d8cd98f00b204e9800998ecf8427e - the md5 of empty. Root cause: Download() returns nil early when GetterUrl == "" (module_service.go:312-314), nothing is extracted, protoPaths() then points Import at a non-existent dir, Import returns no error, Store writes an empty FileDescriptorSet. PRE-EXISTING, not a Phase 11 regression: reproduces byte-identically on b69e3b2 (pre-phase baseline, verified in a throwaway worktree). See gap G-11-7..
  7 race not quieted by stripping -race or coarsening a lock -> the -race half is enforced by .github/workflows/go.yml:37; the "coarsening" half is a diff-review property.
mutation-verified: restoring the silent continue fails the prohibition-4 test; assigning the discovery resolver to s.parser.FilesResolver fails the D-02 test.
regression: full suite green under -race (agent package excluded - pre-existing unrelated hang at agent/command_test.go:118).

### 8. WR-02 — LocalFileCount()'s d.mu.RLock() does not synchronize with localFiles's only production writer

expected: A decision on whether to extend mu's documented scope to genuinely cover localFiles (lock Parse's writes too), or drop LocalFileCount's RLock/RUnlock and document plainly that it is unsafe to call concurrently with Import/Parse on the same registry - so the code's safety claim matches what callers can actually rely on.
why_human: LocalFileCount() (utils/utils.go:390-394, added by 11-05 to back the G-11-7 guard) takes d.mu.RLock(), but Parse() (utils/utils.go:210-222), the sole production writer of localFiles, writes it with zero locking. Not a live bug today - GenFileDescriptorSet calls Import/Parse to completion before calling LocalFileCount() synchronously on the same goroutine, and Sync()'s dependency walk is single-threaded - so -race cannot catch it until a future change (e.g. parallelizing Sync()'s walk) actually drives the interleaving. Same "latent, not yet live" class as WR-04 before it became a real bug and was fixed in this phase (84efad0). No must-have truth in any of the 5 plans covers LocalFileCount's lock discipline, and this finding postdates this file's own 2026-09-07T19:40Z sign-off (11-REVIEW.md, 2026-09-07T21:10Z), so it has never had a human verification pass.
source: 11-REVIEW.md WR-02 (new), carried into 11-VERIFICATION.md human_verification
result: [pending]

## Summary

total: 8
passed: 6
issues: 1
pending: 1
skipped: 0
blocked: 0

## Gaps

- gap_id: G-11-3
  truth: "protoconf mod tidy completes without panicking when protoconf.lock omits the deps key"
  status: resolved
  resolved_by: 11-04-PLAN.md
  resolved_at: 2026-09-07
  reason: "User reported: go run ./cmd/protoconf mod tidy -protoconfPath /tmp/uat-11/ -> panic: assignment to entry in nil map at compiler/lib/module_service.go:146 (ModuleService.Init), via mod/command.go:153 (modTidyCommand.Run)."
  severity: blocker
  test: 3
  pre_existing: true
  pre_existing_evidence: "Reproduces identically on b69e3b2, the commit before Phase 11 began (verified in a throwaway worktree). Line number differs only because Phase 11 added lines above it. Recorded under Phase 11 by explicit user decision, not because Phase 11 caused it."
  root_cause: "LoadFromLockFile (compiler/lib/module_service.go:228-234) is the only site that unmarshals into m.head. protojson.Unmarshal resets the destination message before populating it, so a lock file with no deps key (e.g. {\"url\": \".\"}) leaves m.head.Deps nil - destroying the non-nil-map invariant NewModuleService established at line 79-82. ModuleService.Init then executes m.head.Deps[name] = msg at line 146 and panics. Violates CLAUDE.md's 'no panics in production code'."
  artifacts:
    - path: "compiler/lib/module_service.go"
      issue: "LoadFromLockFile does not restore the Deps non-nil invariant after protojson.Unmarshal resets m.head (line 233)"
    - path: "compiler/lib/module_service.go"
      issue: "Init assumes m.head.Deps is non-nil at line 146 (and reads it at line 133)"
  missing:
    - "Re-initialize m.head.Deps to an empty map in LoadFromLockFile when protojson.Unmarshal leaves it nil - the single chokepoint all 8 production callers route through (inserter/inserter.go:197, server/server.go:290, agent/filekv/filekv.go:74, mutate/mutate.go:72, compiler/lib/compiler.go:61, mod/command.go:68/112/148)"
    - "Regression test: a protoconf root whose protoconf.lock omits deps, asserting mod tidy / ModuleService.Init completes without panic"
    - "Re-test UAT item 3's actual backstop truth (mod sync .fds byte-identity under a concurrent in-process lazy compile) once the panic no longer blocks fixture setup"
  debug_session: ""

- gap_id: G-11-7
  truth: "protoconf mod sync must not write a truncated (zero-byte) .fds, nor overwrite protoconf.lock's fileDescriptorSetSum with the checksum of one, while exiting 0"
  status: failed
  reason: "Found during the item-3 CLI re-test (2026-09-07): on a protoconf.lock with no getterUrl (the shape utils/testdata/small ships), `protoconf mod sync` prints 'Parsing protos.' and 'Storing in cache.', exits 0, writes a ZERO-BYTE .fds for every dep, and rewrites the lock's real fileDescriptorSetSum values to d41d8cd98f00b204e9800998ecf8427e (md5 of empty). Silent cache and lock-file corruption with a success exit code."
  severity: major
  test: 7
  prohibition: 6
  pre_existing: true
  pre_existing_evidence: "Reproduces byte-identically on b69e3b2, the commit before Phase 11 began (verified in a throwaway git worktree with a separately built binary). Not caused by Phase 11; surfaced by Phase 11's UAT."
  root_cause: "Download (compiler/lib/module_service.go:312-314) returns nil early when r.GetterUrl == \"\", so no tarball is extracted. GenFileDescriptorSet then calls protoPaths(), which points at .protoconf_cache/<label>/<sourcePath> - a directory that does not exist. registry.Import over a non-existent path returns NO error, so the 'failed to parse' guard never fires, and registry.Store serializes an empty FileDescriptorSet (0 bytes) and returns the empty md5, which is then persisted into the lock by the Walk/Lock call at the end of GenFileDescriptorSet."
  why_test_missed_it: "TestModSyncFdsByteIdentical constructs its registry via an explicit Import over testdata.GenerateCorpus output, which always exists, so the missing-path branch is unreachable from that test. UAT item 7 recorded prohibition 6 as 'ALREADY COVERED' on the strength of that test; the coverage claim does not hold at the CLI level."
  artifacts:
    - path: "compiler/lib/module_service.go"
      issue: "Download returns nil when GetterUrl is empty, silently skipping extraction rather than reporting an unsynced dep"
    - path: "compiler/lib/module_service.go"
      issue: "GenFileDescriptorSet stores and persists an empty FileDescriptorSet without asserting the parsed set is non-empty or that protoPaths exist"
    - path: "utils/utils.go"
      issue: "DescriptorRegistry.Import over a non-existent path returns nil rather than an error, so the caller's parse-failure guard cannot fire"
  missing:
    - "Fail (or at minimum warn loudly and skip the lock rewrite) when a dep's resolved protoPaths do not exist on disk, instead of storing an empty set"
    - "Never overwrite a non-empty fileDescriptorSetSum in protoconf.lock with the checksum of a zero-byte descriptor set"
    - "Regression test at the mod-sync level, not the registry level: a lock file with no getterUrl must not yield a 0-byte .fds and must not clobber recorded sums"
  debug_session: ""
