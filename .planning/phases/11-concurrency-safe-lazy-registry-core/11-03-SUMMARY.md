---
phase: 11-concurrency-safe-lazy-registry-core
plan: 03
subsystem: compiler
tags: [concurrency, data-race, msgregistry, protoreflect, mod-sync]

requires:
  - phase: 11
    provides: "plan 11-01's on-demand DescriptorRegistry.ParseOne/singleflight parse primitive, which made config.messageRegistry's pre-existing value-copy a live data race by running AddFile mid-compile instead of only at construction"
provides:
  - "config.messageRegistry shared by pointer (*msgregistry.MessageRegistry), closing the copylocks finding and the underlying concurrent-map-access race it flagged"
  - "TestConcurrentCompile — a regression guard mirroring compiler/service.go's CompileFiles and compiler/command.go's runLocally: N goroutines calling CompileFile concurrently against one shared *lib.Compiler"
  - "TestModSyncFdsByteIdentical — a byte-identity regression guard pinning protoconf mod sync's eager registry construction path against an independently constructed eager registry"
affects: [12, 13]

actuals:
  tokens: 1900
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "share a lock-carrying struct by pointer, not by value, across long-lived goroutine-visible state (config.messageRegistry) once any writer can run after construction"
    - "byte-identity + raw-bytes comparison (not just a checksum) as a serialization regression guard, so a silently truncated-but-self-consistent cache file is caught"

key-files:
  created:
    - compiler/lib/concurrent_compile_test.go
    - compiler/lib/mod_sync_fds_test.go
  modified:
    - compiler/lib/config.go
    - compiler/lib/compiler.go

key-decisions:
  - "D-04 executed as locked: BUG-03's go vet copylocks finding at compiler/lib/compiler.go was fixed in this phase, overriding REQUIREMENTS.md's 'deferred beyond this milestone' framing. LAZY-02 (plan 11-01) removed the invariant that made the value-copy benign — AddFile no longer runs only during construction, it now runs mid-compile on every on-demand parse — so the copy's independent zero-value mutex over shared maps became a genuine correctness prerequisite, not an opportunistic cleanup. REQUIREMENTS.md's Known Defects / traceability table should be updated to reflect BUG-03 as resolved by Phase 11, not deferred."
  - "TestModSyncFdsByteIdentical's distinguishability assertion compares the lazy ModuleService's FileRegistry count against the eager registry's own count, not against the literal corpus size (40) the plan specified. NewDescriptorRegistry seeds ~65 well-known types (google/, buf/validate, protoconf/v1) from the global proto registry before any src/ parsing happens, so a 40-file corpus is smaller than the baseline seed regardless of laziness — the literal '< 40' assertion fails unconditionally in this environment, unrelated to any regression. Comparing lazy against eager's own count preserves the intended guard (a lazy ModuleService must not have walked src/) without depending on the seed's exact size."
  - "The concurrent-compile race did not reproduce live under go test -race across ~30 runs (2, 5, 20-iteration batches) either before or after the pointer fix — see Deviations for the full analysis of why the exact race window (a bare struct copy of msgregistry.MessageRegistry racing against another goroutine's AddFile call) is narrow enough that it did not manifest in this 14-core sandbox, even though the corpus generates real concurrent ParseOne/AddFile traffic. go vet's copylocks finding is the deterministic, environment-independent evidence for the underlying bug this task fixes; TestConcurrentCompile stays in the suite as a permanent regression guard exercising the real production fan-out shape."

requirements-completed: [LAZY-02, LAZY-04]

coverage:
  - id: D1
    description: "N concurrent CompileFile calls against one shared *lib.Compiler, each demanding a different proto plus one shared proto, all succeed and produce no data race under go test -race (LAZY-02's concurrency facet)"
    requirement: LAZY-02
    verification:
      - kind: unit
        ref: "compiler/lib/concurrent_compile_test.go#TestConcurrentCompile"
        status: pass
    human_judgment: false
  - id: D2
    description: "go vet ./compiler/... reports no copylocks finding at compiler/lib/compiler.go — the MessageRegistry is shared by pointer, so every reader and writer holds the same mutex instance (D-04 / BUG-03)"
    requirement: LAZY-02
    verification:
      - kind: unit
        ref: "go vet ./compiler/... ./utils/..."
        status: pass
    human_judgment: false
  - id: D3
    description: "protoconf mod sync's registry path still parses and links the whole module tree: the .fds bytes an eager ModuleService serializes over a fixture corpus are byte-identical to those of an independently constructed eager utils.DescriptorRegistry over the same corpus (LAZY-04)"
    requirement: LAZY-04
    verification:
      - kind: unit
        ref: "compiler/lib/mod_sync_fds_test.go#TestModSyncFdsByteIdentical"
        status: pass
    human_judgment: false
  - id: D4
    description: "A lazy ModuleService's registry does not contain the whole corpus, while an eager one over the same root does — the two construction paths are distinguishable"
    requirement: LAZY-04
    verification:
      - kind: unit
        ref: "compiler/lib/mod_sync_fds_test.go#TestModSyncFdsByteIdentical (lazyCount < len(reg.FileRegistry) assertion)"
        status: pass
    human_judgment: false
  - id: D5
    description: "Every pre-existing test in the repository (excluding the pre-existing, unrelated agent Consul-integration hang) stays green under -race after both changes land"
    requirement: LAZY-02
    verification:
      - kind: integration
        ref: "go test -race $(go list ./... | grep -v '/agent$')"
        status: pass
    human_judgment: false
  - id: D6
    description: "The actual runtime race the copy caused before the fix is a genuine, narrow-timing-window race rather than an artifact of test construction — the live evidence for this is go vet's deterministic copylocks finding at compiler/lib/compiler.go:357, not a live -race repro, which did not manifest across ~30 attempted runs"
    requirement: LAZY-02
    verification: []
    human_judgment: true
    rationale: "Coverage requires judging whether an observed absence of a live race repro across many runs is acceptable evidence that the fix is still necessary and correct, given the deterministic go vet signal and the code-level analysis in this SUMMARY's Deviations section. A human should confirm this reasoning is sound rather than treating the missing red-state repro as silently swept under the rug."

duration: 42min
completed: 2026-09-04
status: complete
---

# Phase 11 Plan 03: Concurrency-Safe Lazy Registry Core — Shared-Pointer MessageRegistry Fix and Mod-Sync Regression Guard Summary

**`config.messageRegistry` is now shared by pointer instead of copied by value, closing the last `go vet` copylocks finding in `compiler/lib`, backed by a concurrent-compile test mirroring the production fan-out shape and a byte-identity regression guard for `protoconf mod sync`'s `.fds` output.**

## Performance

- **Duration:** 42 min
- **Started:** ~2026-09-04T09:17:00Z (approx, following 11-02's completion)
- **Completed:** 2026-09-04T09:59:24Z
- **Tasks:** 2
- **Files modified:** 2
- **Files created:** 2

## Accomplishments

- `config.messageRegistry` changed from `msgregistry.MessageRegistry` (a value copy carrying its own zero-value `sync.RWMutex` over maps shared with the original) to `*msgregistry.MessageRegistry`, so every reader (`validate`'s `FindMessageTypeByUrl`) and the `AddFile` writer (now running mid-compile on every on-demand parse, per plan 11-01's LAZY-02) hold the exact same mutex instance.
- `go vet ./compiler/...` no longer reports a copylocks finding anywhere — confirmed the sole pre-existing finding at `compiler/lib/compiler.go:357` is gone, and `go vet ./...` reports zero copylocks/`passes lock by value` findings repository-wide.
- `TestConcurrentCompile` exercises the exact production concurrency shape (`compiler/service.go`'s `CompileFiles` handler and `compiler/command.go`'s `runLocally` both fan out `errgroup.Go` per file against one shared `*lib.Compiler`): 8 goroutines each compile a `.pconf` demanding one shared proto plus one proto unique to that goroutine, all succeed, and `LoadedFileCount()` confirms the compile stayed lazy (>8, <30 for a 30-file corpus).
- `TestModSyncFdsByteIdentical` pins `protoconf mod sync`'s registry construction path (`NewModuleService` → `GetProtoRegistry` → `Store`) against an independently constructed eager `utils.DescriptorRegistry` over the same corpus, asserting both checksum and raw-byte equality, plus a distinguishability check that a lazy `ModuleService`'s registry holds strictly fewer files than the eager one — so the guard cannot pass vacuously if `GetProtoRegistry` ever became lazy for every consumer.
- D-04 executed: BUG-03's `go vet` copylocks finding is fixed in this phase, overriding `REQUIREMENTS.md`'s "deferred beyond this milestone" framing for the stated reason (see Decisions Made).

## Task Commits

Each task was committed atomically:

1. **Task 1: Concurrent-compile race test and the shared-MessageRegistry pointer fix** - `c8b154b` (fix) — test written first and run against the pre-fix code (see Deviations for the observed non-red result), then the pointer fix applied to `compiler/lib/config.go` and `compiler/lib/compiler.go`, both landing in one commit since the plan's task boundary covers red-then-green within a single task.
2. **Task 2: mod sync .fds byte-identity regression guard (LAZY-04)** - `42e28f8` (test)

## Files Created/Modified

- `compiler/lib/concurrent_compile_test.go` (new) - `TestConcurrentCompile`: 8 concurrent `CompileFile` calls against one shared `*lib.Compiler`, asserting no data race, all outputs materialized, and `LoadedFileCount()` bounds proving laziness held.
- `compiler/lib/mod_sync_fds_test.go` (new) - `TestModSyncFdsByteIdentical`: golden eager `utils.DescriptorRegistry` vs. subject eager `ModuleService.GetProtoRegistry()`, byte-identity + checksum equality, lazy-vs-eager distinguishability, and idempotent re-`Store` equality.
- `compiler/lib/config.go` - `config.messageRegistry` field type changed from `msgregistry.MessageRegistry` to `*msgregistry.MessageRegistry`.
- `compiler/lib/compiler.go` - `load`'s `config` construction now assigns `&c.ModuleService.GetProtoRegistry().MessageRegistry`, with a comment explaining why the pointer is required.

## Decisions Made

- **D-04 (locked, executed as specified):** BUG-03 is fixed in this phase, not deferred. LAZY-02 removed the "nothing mutates the registry after construction" invariant that made the value-copy benign; on-demand parsing now runs `AddFile` mid-compile, making the copy's independent mutex a live correctness issue rather than a cosmetic `go vet` nag. `REQUIREMENTS.md`'s Known Defects entry for BUG-03 and its traceability table row should be updated from "deferred beyond this milestone" to resolved-by-Phase-11 — flagging this explicitly per the plan's locked decision so the ledger does not silently diverge.
- **TestModSyncFdsByteIdentical's distinguishability assertion** compares the lazy `ModuleService`'s `FileRegistry` count against the eager registry's own count rather than the plan's literal "< 40" (corpus size) comparison — see Deviations for why the literal form fails unconditionally in this environment and why this substitution preserves the guard's intent.
- Kept `TestConcurrentCompile` in the suite as a permanent regression guard even though it did not reproduce a live race locally (see Deviations) — it exercises the real production concurrency shape and stays a meaningful smoke test: any future regression here should not need a hand-tuned timing window to be caught by the primary, deterministic signal (`go vet`'s copylocks check), but the runtime test still guards against a different class of bug (a fatal `concurrent map read and map write` crash, or a future change that reintroduces the value copy in a form `go vet` cannot see, e.g. through an interface boundary).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `TestModSyncFdsByteIdentical`'s literal "< 40" distinguishability assertion fails unconditionally, unrelated to any regression**

- **Found during:** Task 2, first run of the test as written per the plan's `<behavior>` (`require.Less(t, len(lazyMs.GetProtoRegistry().FileRegistry), 40, ...)`)
- **Issue:** `utils.NewDescriptorRegistry()`'s constructor seeds `FileRegistry` with ~65 well-known types (`google/`, `buf/validate`, `protoconf/v1`) scanned from the global proto registry, before any `src/` parsing happens. A 40-file generated corpus is therefore smaller than this baseline seed regardless of laziness, so the plan's literal assertion (`lazyCount < 40`) fails on every run — it measures the seed's size, not whether `src/` was walked.
- **Fix:** Compare the lazy `ModuleService`'s `FileRegistry` count against the eager registry's own `FileRegistry` count (`require.Less(t, lazyCount, len(reg.FileRegistry), ...)`) instead of the hardcoded corpus size. This preserves the acceptance criterion's actual intent — "the two construction paths are distinguishable, which is what makes the mod sync regression guard meaningful" — without depending on the seed's exact size, which is a `jhump/protoreflect`/global-registry implementation detail, not something this test should assert about.
- **Files modified:** `compiler/lib/mod_sync_fds_test.go`
- **Verification:** `go test ./compiler/lib/... -run TestModSyncFdsByteIdentical -v` → `--- PASS`.
- **Committed in:** `42e28f8` (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 Rule 1 bug fix in a test assertion). **Impact on plan:** No scope creep — the fix corrects a magic-number assertion that could never have passed in this environment, while preserving the guard's stated purpose exactly.

## Issues Encountered

- **`TestConcurrentCompile` did not reproduce a live data race, before or after the pointer fix, across extensive repeated runs** (`-race -count=2` twice, `-race -count=1` five times, `-race -count=20` once — roughly 30 total executions of the pre-fix code). Root-cause analysis, performed by reading `config.go`'s `validate` method and `github.com/jhump/protoreflect/dynamic/msgregistry`'s source (`message_registry.go`): the only *read* of `config.messageRegistry` is `validate`'s `case *anypb.Any:` branch calling `FindMessageTypeByUrl`. None of this test's fixture messages carry an `Any`-typed field (the plan's `<behavior>` specifies plain `Msg{k+10}(name="c{k}")` returns, with no `Any` wrapping), so that read path is never exercised, and the only unsynchronized operation actually exercised is the bare struct-copy assignment at `compiler.go`'s old `messageRegistry: c.ModuleService.GetProtoRegistry().MessageRegistry` line, racing against another goroutine's `AddFile` call (itself guarded by the *original* registry's mutex, but not synchronized with the unguarded copy). This is a real but very narrow timing window — a single struct copy of a few dozen bytes racing against microsecond-scale critical sections inside `AddFile` — and Go's race detector, being a dynamic/interleaving-dependent tool, did not observe an actual conflicting pair of accesses in this 14-core sandbox across the runs attempted. The plan's own phase notes anticipated this possibility ("confirm it fails ... *if you can arrange this cheaply*"), so this is disclosed rather than treated as a test failure: `go vet`'s copylocks finding is the deterministic, environment-independent evidence for the bug this task fixes (confirmed present before the fix, confirmed absent after), and `TestConcurrentCompile` remains a valid regression guard against the production fan-out shape (a fatal `concurrent map` crash or a reintroduced value copy) even though it did not happen to catch this specific narrow race live. Not treated as a Rule 4 architectural question — no test design or corpus-shape choice available within the plan's exact prescribed shape (8 goroutines, corpus of 30, no `Any` fields) would deterministically force this race without either widening the corpus/goroutine count beyond the plan's literal acceptance criteria or introducing artificial synchronization delays into production code, neither of which the plan calls for.
- `go test -race ./...` (the plan's repo-wide verification command, matching CI) was run via `go test -race $(go list ./... | grep -v '/agent$')` per the phase notes' documented pre-existing, unrelated `agent` package Consul-integration hang (confirmed pre-existing by both 11-01 and 11-02's SUMMARYs, out of scope for this plan's files). Every other package passed cleanly under `-race`, including `compiler/lib` (both new tests), `server`, `test`, `devserver`, `utils`.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- LAZY-02 and LAZY-04 are both closed. BUG-03 (`go vet` copylocks) is resolved — `REQUIREMENTS.md`'s Known Defects section and traceability table should be updated to reflect this override of its prior "deferred beyond this milestone" status (D-04).
- `go test -race -count=2 ./compiler/lib/... -run TestConcurrentCompile` is green; `go test ./compiler/lib/... -run TestModSyncFdsByteIdentical` is green; `go vet ./...` reports zero copylocks findings; `go test -race` on every package except the pre-existing, unrelated `agent` hang is green.
- Phase 11 (Concurrency-Safe Lazy Registry Core) is now complete: all three plans (11-01 on-demand parse primitive, 11-02 mutation server Init discovery fix, 11-03 this plan) have landed. No blockers for Phase 12.

---
*Phase: 11-concurrency-safe-lazy-registry-core*
*Completed: 2026-09-04*

## Self-Check: PASSED

- All 4 key files confirmed present on disk: `compiler/lib/concurrent_compile_test.go`, `compiler/lib/mod_sync_fds_test.go` (created), `compiler/lib/config.go`, `compiler/lib/compiler.go` (modified).
- Both commit hashes (`c8b154b`, `42e28f8`) confirmed in `git log --oneline -5`.
- All plan-level `<acceptance_criteria>` re-verified: `go test -race -count=2 ./compiler/lib/... -run TestConcurrentCompile -v` → `--- PASS: TestConcurrentCompile` (x2), no `WARNING: DATA RACE`; `go vet ./compiler/... ./utils/...` → exit 0, no copylocks/`passes lock by value` output; `go test ./compiler/lib/... -run TestModSyncFdsByteIdentical -v` → `--- PASS: TestModSyncFdsByteIdentical`.
- Plan-level `<verification>` re-run: `go vet ./...` reports no copylocks findings anywhere; `go test -race $(go list ./... | grep -v '/agent$')` green across every package including `compiler/lib`, `utils`, `server`, `test`, `devserver`.
