---
phase: 11-concurrency-safe-lazy-registry-core
verified: 2026-09-04T17:50:00Z
status: human_needed
score: 18/18 must-have truths verified (plus 5 roadmap success criteria, all independently reproduced)
behavior_unverified: 0
overrides_applied: 0
coincidental_reliance_items:
  - truth: "N concurrent CompileFile calls against one shared *lib.Compiler, each demanding a different proto plus one shared proto, all succeed and produce no data race under go test -race (LAZY-02 concurrency facet, 11-03 Task 1)"
    reason: fixture-only
    harden: "utils/testdata/corpus.go's generator never emits a google.protobuf.Any field, so config.validate's `case *anypb.Any:` branch — the only reader of config.messageRegistry — is never exercised by TestConcurrentCompile. The test's green -race result therefore does not discriminate between the pointer fix being present and the pre-fix value-copy bug: it never drives the read side of the race it claims to guard (independently confirmed: 0/2 counted races across a fresh -count=2 run here, matching the executor's own ~30-run disclosure). go vet's copylocks finding (confirmed clean, independently re-run) is the actual deterministic evidence the fix is real; TestConcurrentCompile only proves the AddFile write path and the fatal-crash path are race-free. Harden by adding an Any-typed field to the corpus generator (or a small standalone fixture) so the read path is genuinely exercised, per code review WR-05."
human_verification:
  - test: "NewCompiler / NewLazyModuleService against a protoconf root whose src/ contains zero .proto files, and against one containing exactly one .proto file (plan 11-01 backstop truth)"
    expected: "Construction succeeds in both cases rather than panicking or erroring on an empty/near-empty tree"
    why_human: "No committed test exercises either edge case — utils/testdata/corpus.go's GenerateCorpus refuses n<5 (`n must be >= 5`), and no other fixture with 0 or 1 proto file is used anywhere in the diffed test files. This is a structured `verification: backstop` truth in the PLAN frontmatter; per the honest-verifier contract it must abstain rather than be inferred from the rest of the lazy-parse code reading correctly for the general case."
  - test: "ProtoconfMutationServer.Init registers an identical set of services across repeated runs regardless of protoregistry.Files.RangeFiles' map iteration order (plan 11-02 backstop truth)"
    expected: "The registered service set is order-independent; only registration order varies between runs, never which services end up registered"
    why_human: "No test runs Init multiple times or otherwise forces distinct map iteration orders to confirm this. grpc.Server.GetServiceInfo() being backed by a Go map is suggestive but not itself the executed evidence the backstop tag requires — presence/plausibility does not qualify per the honest-verifier contract."
  - test: "protoconf mod sync's registry serializes from its own instance, so a concurrent lazy compile in the same process cannot alter the .fds bytes mod sync writes (plan 11-03 backstop truth)"
    expected: "Running mod sync concurrently with an in-process lazy CompileFile does not change mod sync's serialized output"
    why_human: "TestModSyncFdsByteIdentical exercises the eager mod-sync path in isolation, sequentially — no test in this phase runs a lazy compile and a mod-sync Store() concurrently in the same process to observe the claimed non-interference directly."
  - test: "WR-02 (code review): ParseOne can return a non-canonical *desc.FileDescriptor pointer when ParseAll (the D-03 fallback) parses and inserts the same path while a concurrent ParseOne call for that same path is mid-parse outside the lock"
    expected: "Every caller of ParseOne for a given path — regardless of interleaving with a concurrent ParseAll — observes the one canonical FileRegistry pointer, matching ParseOne's documented pointer-identity contract"
    why_human: "This is a genuine, code-review-identified race between ParseOne's singleflight closure (which parses outside d.mu, per its own documented lock discipline) and ParseAll (which holds d.mu across its entire whole-tree parse per WR-01). TestParseMemoization's concurrency subtest only races ParseOne against ParseOne, never against ParseAll, so it cannot catch this interleaving. Reproducing it live requires deliberately racing ParseOne and ParseAll on an overlapping path, which no test in this phase does. A human should decide whether to fix now (return the post-recordFileLocked canonical value, per the reviewer's suggested one-line fix) or accept as scoped-out follow-up."
  - test: "WR-01 (code review): ParseAll holds d.mu across its entire whole-tree parse, contradicting ParseOne's own documented 'never hold d.mu across parser.ParseFiles' rule, and serializing every other DescriptorRegistry caller for the fallback's full duration"
    expected: "A confirmation of whether this is acceptable given D-03's 'fires at most once per registry' bound, or whether it should be fixed to match ParseOne's discipline before the phase is considered fully closed"
    why_human: "Not deadlock-prone (confirmed: Import's LookupImport/Accessor closures never re-enter d.mu) but it is a real lock-hygiene inconsistency within the same file, with a concrete stall scenario (one goroutine's D-03 miss blocking every concurrently-compiling file on a shared *lib.Compiler for the whole-tree parse's duration). No must-have truth in this phase's plans asserts anything about ParseAll's lock duration, so this doesn't fail a stated truth, but it directly touches the phase's own concurrency-safety framing."
  - test: "WR-04 (code review): ModuleService.GetProtoRegistry()'s cachedRegistry field is read/written with no lock, safe today only because NewCompiler primes it synchronously before any concurrent caller exists"
    expected: "A decision on whether this unenforced construction-order invariant needs a lock now (module_service.go already has an unused m.mutex for this exact purpose) or is acceptable as documented risk for a future direct-construction caller"
    why_human: "Not a live bug today (confirmed: NewCompiler's single synchronous priming call happens-before every later concurrent reader), but it is exactly the class of implicit invariant this phase's own threat model (T-11-01) treats as high severity elsewhere. No must-have truth covers it."
  - test: "Prohibition (11-01, 11-02, 11-03 plan frontmatter, verification: manual): a proto that fails to parse on the lazy path must not be silently swallowed into a successful compile; the D-03 eager fallback must never fire invisibly; pre-existing fixtures/assertions must not be weakened; a service Init cannot register must not vanish without a trace; the discovery scan must not back gRPC reflection; mod sync must not write a truncated .fds; a data race must not be quieted by removing -race or coarsening a lock"
    expected: "Each prohibition holds under close reading, not just under the tests that happen to pass"
    why_human: "All seven are explicitly tagged `verification: manual` in the PLAN frontmatter — the plan authors themselves deferred these to human judgment rather than an automated check. Independent evidence gathered during this verification supports most of them (D-03 fallback is logged via the `eagerFallback` field on the `compile finished` line, confirmed live; reflection registrations still read `s.parser.FilesResolver`/`LocalResolver`, confirmed by grep; `TestModSyncFdsByteIdentical` directly targets the truncated-.fds prohibition; no new lock or `-race`-stripping was found in the diff) but none of these were exercised by a dedicated adversarial test (e.g., actually feeding a broken .proto through the lazy path and asserting the compile fails rather than silently materializing wrong output), so the plan's own manual-verification tag should get an explicit human sign-off rather than being closed by inference here."
---

# Phase 11: Concurrency-Safe Lazy Registry Core Verification Report

**Phase Goal:** A compile no longer pays for the whole repository's proto tree, and the mutation server's service catalog doesn't silently go dark under that change.
**Verified:** 2026-09-04T17:50:00Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Roadmap Success Criteria (the binding contract)

All five were independently reproduced against the running code in this session, not accepted from SUMMARY.md claims.

| # | Criterion | Status | Evidence |
|---|-----------|--------|----------|
| 1 | Compiling a config no longer incurs a delay proportional to total repo proto count | ✓ VERIFIED | `go test ./compiler/lib/... -run TestGeneratedCorpusCompiles -v` (re-run live): `compile loaded 5 proto files` out of a 50-on-disk-proto corpus; `c.ModuleService.GetProtoRegistry().LoadedFileCount()` is 5, asserted `< 50` |
| 2 | Requesting the same proto file twice costs one parse — the second is a map lookup | ✓ VERIFIED | `TestParseMemoization` (`compiler/lib/parser/lazy_parse_test.go`), re-run live under `-race`: `require.Same(t, fd1, fd2)` passes; concurrency subtest (8 goroutines, cold registry, `singleflight`) — all 8 pointers `require.Same` |
| 3 | `protoconf mod sync` still writes a `.fds` cache file identical in content to before this change | ✓ VERIFIED | `TestModSyncFdsByteIdentical` (`compiler/lib/mod_sync_fds_test.go`), re-run live: eager `ModuleService.GetProtoRegistry()` and an independently constructed eager `utils.DescriptorRegistry` over the same 40-proto corpus produce byte-identical `.fds` output (both checksum and raw bytes); a lazy `ModuleService`'s `FileRegistry` is confirmed strictly smaller, proving the guard is not vacuous |
| 4 | An operator can see, from compiler output, how many proto files a compile loaded | ✓ VERIFIED | Live CLI run: `go run ./cmd/protoconf compile utils/testdata/small test.pconf` → `INFO compile finished file=test.pconf protoFilesLoaded=1 eagerFallback=false` |
| 5 | A custom gRPC mutation service defined under `src/` is registered and reachable at server startup, before any config has been compiled | ✓ VERIFIED | `TestInitRegistersCustomService` (`server/server_test.go`), re-run live: swaps in a bare (near-empty) resolver snapshot, calls `Init` with no prior compile, asserts `test.v1.TestService` and both its methods are registered — `--- PASS` |

### Plan-Level Must-Have Truths (18 non-backstop truths across the 3 plans)

All 18 were independently re-run in this session (not taken from SUMMARY.md). All pass.

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | `NewCompiler` on a 50-proto corpus doesn't parse all 50; compile writes `out0..out4` | ✓ VERIFIED | `TestGeneratedCorpusCompiles` re-run live |
| 2 | `ParseOne` returns the identical pointer on a second request | ✓ VERIFIED | `TestParseMemoization` re-run live |
| 3 | `LoadedFileCount()` counts each file at most once | ✓ VERIFIED | `TestParseMemoization` (`countAfterFirst == countAfterSecond`) |
| 4 | Concurrent `ParseOne` calls for the same not-yet-parsed path collapse to one parse, same pointer | ✓ VERIFIED | `TestParseMemoization` concurrency subtest, re-run under `-race` |
| 5 | Calling `ParseOne` twice leaves `Store()` byte-identical; same pointer both times | ✓ VERIFIED | `TestLazyParseDoesNotMutateLocalFiles` re-run live |
| 6 | `LoadedFileCount()` is safe to call from one goroutine while another is inside `ParseOne` | ✓ VERIFIED | `TestConcurrentCompile` re-run under `-race -count=2`: 8 goroutines' `"compile finished"` log lines call `LoadedFileCount()` while siblings are still parsing; no race reported |
| 7 | Every pre-existing test in `./compiler/...` `./utils/...` `./server/...` stays green under `-race`, including the D-03 fallback path | ✓ VERIFIED | `go test -race ./compiler/... ./utils/... ./server/...` re-run live: all green. `load_mutable_test.pconf` fallback independently reproduced live: `go run ./cmd/protoconf compile utils/testdata/small load_mutable_test.pconf` → `eagerFallback=true` |
| 8 | `protoconf compile` emits a loaded-count line | ✓ VERIFIED | Live CLI run (see roadmap criterion 4) |
| 9 | `Init` registers `test.v1.TestService` with no prior `CompileFile` | ✓ VERIFIED | `TestInitRegistersCustomService` re-run live |
| 10 | `Init` discovers via its own disk scan, not the parser's construction-time snapshot | ✓ VERIFIED | Same test: the snapshot is deliberately emptied before `Init` runs, and the service is still found — proven structurally by the test design, and confirmed by reading `Init`'s discovery loop, which ranges `discoveryFiles` (a fresh throwaway registry), never `s.parser.FilesResolver`, for service registration |
| 11 | `Init` registers each service at most once (duplicate-registration guard) | ✓ VERIFIED | Same test: exactly one `test.v1.TestService` entry, `Init` does not panic |
| 12 | `Init` against an empty `src/` registers only built-ins, no error | ✓ VERIFIED | `TestInitWithNoCustomServices` re-run live |
| 13 | `TestProtoconfMutationServer_GenReflectionUI` and every other pre-existing `./server/...` test stays green | ✓ VERIFIED | `go test -race ./server/...` re-run live: green |
| 14 | N concurrent `CompileFile` calls, shared + per-goroutine proto, all succeed, no race | ✓ VERIFIED (coincidental-reliance) | `TestConcurrentCompile` re-run under `-race -count=2`: green. See `coincidental_reliance_items` — the test's green result does not exercise the specific read path (`config.validate`'s `Any` branch) the pointer fix protects; `go vet`'s copylocks result is the real evidence the fix is necessary and correct |
| 15 | `go vet ./compiler/...` reports no copylocks finding | ✓ VERIFIED | `go vet ./...` re-run live: zero copylocks/`passes lock by value` findings anywhere in the repository |
| 16 | `mod sync`'s `.fds` bytes are identical to an independently constructed eager registry | ✓ VERIFIED | `TestModSyncFdsByteIdentical` re-run live |
| 17 | A lazy `ModuleService`'s registry is strictly smaller than an eager one over the same root | ✓ VERIFIED | Same test: `require.Less(lazyCount, len(reg.FileRegistry))` — confirmed to genuinely distinguish the two construction paths, not compare a value to itself (the plan's literal "< 40" assertion was replaced because `NewDescriptorRegistry`'s ~65-entry well-known-type seed already exceeds 40; the substituted assertion still proves the two paths differ) |
| 18 | `LoadedFileCount()` read from the compile goroutine while others parse doesn't race | ✓ VERIFIED | Same as #6 |

### Backstop Truths (must abstain — routed to human verification, not scored)

Per the honest-verifier contract, a `verification: backstop` truth without direct executed evidence must abstain rather than be inferred. All three plans' backstop truths lack a dedicated test:

| # | Truth | Status | Reason |
|---|-------|--------|--------|
| B1 | `NewCompiler` succeeds against `src/` with zero or exactly one `.proto` file (11-01) | ⚠️ insufficient_spec | No test exercises either edge case; `GenerateCorpus` refuses `n < 5` |
| B2 | `Init`'s registered service set is order-independent across `RangeFiles` iteration orders (11-02) | ⚠️ insufficient_spec | No test forces or observes multiple iteration orders |
| B3 | `mod sync` serializes from its own instance, immune to a concurrent lazy compile in-process (11-03) | ⚠️ insufficient_spec | `TestModSyncFdsByteIdentical` runs the eager path in isolation, never concurrently with a lazy compile |

### Required Artifacts

All artifacts declared in the three plans' `must_haves.artifacts` exist, are substantive (no stubs), and are wired — confirmed by direct reading, not by grep alone.

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `utils/utils.go` | `ParseOne`/`ParseAll`/`LoadedFileCount`/`FellBackToEager`/`FileDescriptor`, `sync.RWMutex` + `singleflight.Group` | ✓ VERIFIED | All five methods present, exact signatures match plan; lock discipline matches the documented "unlock before parse" rule (confirmed by reading) |
| `compiler/lib/parser/parser.go` | `RegistryTypeResolver`, `TypeResolver` field, `ParseFilesX` dispatch to `ParseOne` | ✓ VERIFIED | Confirmed; resolver ladder (snapshot → `MessageRegistry` → `ParseAll`) present exactly as specified |
| `compiler/lib/module_service.go` | `NewLazyModuleService`, `lazyRegistry` branch in `GetProtoRegistry` | ✓ VERIFIED | Confirmed; `Sync`/`GenFileDescriptorSet` untouched (LAZY-04 boundary held) |
| `compiler/lib/compiler.go` | `NewCompiler` uses `NewLazyModuleService`; `"compile finished"` log line; `&...MessageRegistry` pointer | ✓ VERIFIED | Confirmed at lines 57, 221, 363 |
| `compiler/lib/starlark_loader.go` | `loadMutable` resolves via `TypeResolver` | ✓ VERIFIED | Confirmed at line 177 |
| `compiler/lib/config.go` | `messageRegistry *msgregistry.MessageRegistry` (pointer, not value) | ✓ VERIFIED | Confirmed |
| `server/server.go` | `Init`'s `discoveryRegistry` scan, reflection calls unchanged | ✓ VERIFIED | Confirmed; reflection registrations still read `s.parser.FilesResolver`/`LocalResolver` (D-02 boundary held) |
| Test files (`lazy_parse_test.go`, `lazy_load_count_test.go`, `concurrent_compile_test.go`, `mod_sync_fds_test.go`, `server_test.go` additions) | Named tests exist and pass | ✓ VERIFIED | All re-run live in this session, all `--- PASS` |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| `starlarkLoader.loadProto` | `Parser.ParseFilesX` → `DescriptorRegistry.ParseOne` | miss branch | ✓ WIRED | Confirmed by reading `ParseFilesX` |
| `Compiler.writeConfig` / `starlarkLoader.loadMutable` | `Parser.TypeResolver` → `MessageRegistry` → snapshot fallback | `Resolver:` field | ✓ WIRED | Confirmed at `compiler.go:297`, `starlark_loader.go:177` |
| `RegistryTypeResolver` double miss | `DescriptorRegistry.ParseAll` → retry | D-03 fallback | ✓ WIRED | Confirmed; live-reproduced (`eagerFallback=true` for `load_mutable_test.pconf` run in isolation) |
| `NewCompiler` | `NewLazyModuleService` → `GetProtoRegistry` lazy branch | D-01 | ✓ WIRED | Confirmed |
| `ProtoconfMutationServer.Init` | throwaway `utils.NewDescriptorRegistry()` + `Import` → `GetFilesResolver` → `RangeFiles` | discovery scan | ✓ WIRED | Confirmed; independent of `s.parser` |
| discovery loop's `protoregistry.GlobalFiles` skip | `rpcServer.RegisterService` | duplicate guard | ✓ WIRED | Confirmed, preserved verbatim |
| `config.messageRegistry` (pointer) | `DescriptorRegistry.MessageRegistry` via `AddFile` | shared mutex | ✓ WIRED | Confirmed: `&c.ModuleService.GetProtoRegistry().MessageRegistry` |
| `compiler/service.go` errgroup-per-file / `compiler/command.go` `runLocally` | one shared `*lib.Compiler` | production concurrency shape | ✓ WIRED | Confirmed unchanged; matches what `TestConcurrentCompile` mirrors |
| `ModuleService.Sync` / `GenFileDescriptorSet` | own `utils.NewDescriptorRegistry()` → `Import(Parse)` → `Store` | eager path (LAZY-04) | ✓ WIRED | Confirmed untouched by this phase |

### Behavioral Spot-Checks (live, run in this session)

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Compile loads only demanded protos | `go run ./cmd/protoconf compile <50-corpus> main.mpconf` (via `TestGeneratedCorpusCompiles`) | `protoFilesLoaded=5` | ✓ PASS |
| Operator-visible loaded-count line | `go run ./cmd/protoconf compile utils/testdata/small test.pconf` | `compile finished file=test.pconf protoFilesLoaded=1 eagerFallback=false` | ✓ PASS |
| D-03 fallback fires and is reported | `go run ./cmd/protoconf compile utils/testdata/small load_mutable_test.pconf` (isolated) | `eagerFallback=true` | ✓ PASS |
| No copylocks anywhere in the repo | `go vet ./...` | zero copylocks/`passes lock by value` findings | ✓ PASS |
| Full `-race` suite for this phase's packages | `go test -race ./compiler/... ./utils/... ./server/...` | all green | ✓ PASS |
| Adjacent packages unaffected | `go test -race -count=1 ./test/... ./inserter/... ./mutate/... ./devserver/...` | all green (`./test/...` 203s, genuine fresh run) | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| LAZY-01 | 11-01 | `GetProtoRegistry()` stops bulk-parsing `src/` on the compiler path | ✓ SATISFIED | `TestGeneratedCorpusCompiles`, live CLI |
| LAZY-02 | 11-01, 11-03 | On-demand parse, memoised, concurrency-safe | ✓ SATISFIED | `TestParseMemoization`, `TestConcurrentCompile` (see coincidental-reliance note) |
| LAZY-03 | 11-01 | On-demand parsing never mutates `localFiles` | ✓ SATISFIED | `TestLazyParseDoesNotMutateLocalFiles` |
| LAZY-04 | 11-03 | `mod sync` still writes an identical `.fds` | ✓ SATISFIED | `TestModSyncFdsByteIdentical` |
| LAZY-05 | 11-01 | Operator-visible loaded-proto count | ✓ SATISFIED | `TestLoadedFileCount`, live CLI |
| CONS-01 | 11-02 | Custom gRPC service catalog survives the lazy switch | ✓ SATISFIED | `TestInitRegistersCustomService`, `TestInitWithNoCustomServices` |

No orphaned requirements: `.planning/REQUIREMENTS.md`'s Traceability table lists exactly these 6 IDs against "Phase 11 / Complete", matching the union of `requirements:` fields declared across all three plans exactly.

### Anti-Patterns Found

None. Scanned all 13 files touched by this phase (`utils/utils.go`, `compiler/lib/{compiler,config,module_service,starlark_loader,startup_bench_test}.go`, `compiler/lib/parser/{parser,lazy_parse_test}.go`, `compiler/lib/{lazy_load_count_test,concurrent_compile_test,mod_sync_fds_test}.go`, `server/{server,server_test}.go`) for `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` and placeholder-language patterns — zero matches.

### Code Review Findings Weighed (11-REVIEW.md: 0 critical, 5 warning, 2 info)

Per instruction, these warnings were independently re-read against the current code (not re-derived) and are reflected in `human_verification` above where they bear on a must-have:

- **WR-01** (ParseAll holds `d.mu` across the whole fallback parse, contradicting ParseOne's own documented discipline) — real, not deadlock-prone, no must-have covers it. → human-verification item.
- **WR-02** (ParseOne can hand out a non-canonical pointer when racing ParseAll for the same path) — real, provable, breaks the pointer-identity contract only under a specific untested interleaving. → human-verification item.
- **WR-03** (RegistryTypeResolver swallows ParseAll's error with no log) — real but low-severity (operator sees `NotFound` rather than the root cause); not gating, not separately listed as a human-verification item since it affects diagnosability, not correctness.
- **WR-04** (`ModuleService.cachedRegistry` check-then-act is unsynchronized, safe today only by construction-order convention) — real, not a live bug today, no must-have covers it. → human-verification item.
- **WR-05** (`TestConcurrentCompile`'s doc comment overstates what it exercises) — confirmed by independent analysis and a fresh `-race -count=2` run. → recorded as `coincidental_reliance_items`, not a gap (the truth as literally worded still holds).
- **IN-01, IN-02** (informational, no fix required) — no action needed; not surfaced further.

### `./test/...` End-to-End Suite (initially misread as a regression — corrected)

A first pass at `go test -race ./test/...` in this session showed `Test/get_first_message_on_devClient` and two sibling subtests failing with a type-URL mismatch (`test.v1.TestMessage` vs. an expected `google.protobuf.Value`). This turned out to be **self-inflicted test contamination from this verification session, not a defect in the phase's code**: an earlier manual CLI spot-check in this same session (`go run ./cmd/protoconf compile utils/testdata/small load_mutable_test.pconf`, used to confirm the D-03 fallback fires) was run directly against the real, git-tracked `utils/testdata/small/` path rather than a copy, which overwrote the checked-in `materialized_config/load_mutable_test.materialized_JSON` fixture on disk. `utils/testdata/embed.go` embeds that same tree via `//go:embed`, so the corrupted file got baked into the next test binary build, producing the mismatch.

Corrective steps taken: `git checkout -- utils/testdata/small/materialized_config/load_mutable_test.materialized_JSON` to restore the pristine committed fixture (confirmed via `git diff` before reverting — the committed content is `protoFile: google/protobuf/wrappers.proto`, `@type: google.protobuf.Value`, matching the e2e test's expectation exactly), `go clean -testcache`, then a full clean re-run. Final, trustworthy result: `go test -race -count=1 ./test/... ./inserter/... ./mutate/... ./devserver/...` — **all green** (`ok github.com/protoconf/protoconf/test 203.138s`, a genuine fresh execution, not a cache hit). `git status` after cleanup showed no stray modifications from this session. This is recorded here in the interest of transparency about the verification process, not as a finding about the phase.

### REQUIREMENTS.md Ledger Consistency (checked per verification notes)

`BUG-03`'s entry (`go vet copylocks at compiler/lib/compiler.go:355`) no longer says "deferred beyond this milestone" in its own line, but it is still physically nested under the `## Future Requirements` heading, whose intro sentence reads "Acknowledged, deferred beyond this milestone." The code fix is real and independently confirmed (`go vet ./...` is clean of copylocks repo-wide). This is a real, still-unresolved ledger inconsistency exactly as flagged by the 11-03 executor's own summary ("REQUIREMENTS.md's Known Defects entry ... should be updated ... to resolved-by-Phase-11"). Non-blocking for this phase's code goal — it is a documentation correction, not a functional gap — but it should be fixed (move or annotate the BUG-03 line so it no longer reads as deferred) before the milestone closes, so a future reader doesn't reintroduce the value-copy under the belief it's still an accepted, unfixed defect.

### Human Verification Required

See the `human_verification` list in the frontmatter for the full, structured set (3 backstop truths, 3 code-review concurrency gaps, and the 7 plan-declared `verification: manual` prohibitions). Summarized:

1. **Zero-proto / one-proto `src/` construction (11-01 backstop)** — untested edge case.
2. **Service-registration order-independence (11-02 backstop)** — untested claim about map iteration.
3. **`mod sync` immunity to a concurrent in-process lazy compile (11-03 backstop)** — untested concurrent scenario.
4. **WR-02: `ParseOne`/`ParseAll` pointer-identity race** — a real, code-review-identified gap in the pointer-identity guarantee under a specific untested interleaving. Needs a decision: fix now (one-line, reviewer-supplied) or accept as scoped follow-up.
5. **WR-01: `ParseAll`'s lock-duration inconsistency** — needs a decision on whether the stall risk is acceptable given the one-shot latch, or should be fixed to match `ParseOne`'s own documented discipline.
6. **WR-04: `cachedRegistry` unsynchronized check-then-act** — needs a decision on whether to guard with the already-present `m.mutex` now or accept as documented risk.
7. **Seven plan-declared `verification: manual` prohibitions** (data integrity / fallback visibility / no weakened fixtures / no silently-dropped services / reflection scope / no truncated `.fds` / no quieted races) — all plan authors deferred these to explicit human sign-off; independent evidence gathered here supports most of them but none were exercised by a dedicated adversarial test.

### Gaps Summary

No must-have truth failed, no artifact is missing or a stub, and no key link is unwired — the phase's core deliverable (on-demand parsing, memoized and concurrency-safe, wired end-to-end through the compiler; the mutation server's service catalog independent of the lazy registry) is genuinely present and independently reproduced against the running code, not accepted from SUMMARY.md narrative.

What keeps this from a clean `passed`: three plan-declared `backstop` truths have no executed evidence (must abstain, not infer); the code review's own warnings (WR-01, WR-02, WR-04) identify real, unfixed concurrency-correctness edge cases outside what any must-have literally tests; and `TestConcurrentCompile`'s green result is coincidental with respect to the specific bug D-04 fixed (real evidence for that fix is `go vet`, not this test). None of these block the phase's stated goal from being true today, but per the honest-verifier contract none of them may be silently waved through either — they are surfaced for an explicit human decision rather than closed here.

---

_Verified: 2026-09-04T17:50:00Z_
_Verifier: Claude (gsd-verifier)_
