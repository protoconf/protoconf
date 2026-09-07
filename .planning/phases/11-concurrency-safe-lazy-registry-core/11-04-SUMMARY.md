---
phase: 11-concurrency-safe-lazy-registry-core
plan: 04
subsystem: compiler (module/dependency management), testing
tags: [protojson, module-service, mod-cli, errgroup, race-detector, lazy-registry]

# Dependency graph
requires:
  - phase: 11-03 (concurrency-safe-lazy-registry-core)
    provides: NewLazyModuleService / GetProtoRegistry double-checked locking (WR-04 fix, commit 84efad0), the lazy Compiler construction path this plan's Task 2 exercises
provides:
  - "LoadFromLockFile restores the non-nil Deps invariant on every return path, including the unmarshal-error path"
  - "ModuleService.Init loads its own lock file and returns the parse error before any CONFIGSPACE file executes; MergeLock no longer discards Init's merge"
  - "TestModInitLockFileShapes: mod's first test file, 8 CLI-level regression cases for every protoconf.lock shape"
  - "TestModSyncFdsUnaffectedByConcurrentLazyCompile: executed proof that mod sync's eager .fds serialization is unmoved by a concurrent in-process lazy compile"
affects: [mod, compiler/lib, requirements-ledger (LAZY-04, G-11-3)]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
actuals:
  tokens: 3400    # chars/4 over the realized diff (13564 chars across all 4 changed files, both tasks) — plan estimated 70000
  tasks: 2
  commits: 3      # baf8308 (test, RED), 243ab6f (fix, GREEN), c4ee904 (test, Task 2)

# Tech tracking
tech-stack:
  added: []       # golang.org/x/sync/errgroup was already a direct dependency (used by compiler/service.go)
  patterns:
    - "Chokepoint invariant restoration: fix the single function every caller routes through (LoadFromLockFile) rather than scattering nil-checks across eight call sites"
    - "Mutation-verification as phase standard: every fix and every new concurrency test in this plan was proven by reverting it and observing the specific red signal, matching 11-01..03 and the UAT session"
    - "Stand-in test with an honest scope note when the real path (ModuleService.Sync) needs the network — assert the same instance-separation property against the reachable eager registry instead of skipping the truth entirely"

key-files:
  created:
    - mod/command_test.go
  modified:
    - compiler/lib/module_service.go
    - mod/command.go
    - compiler/lib/mod_sync_fds_test.go

key-decisions:
  - "G-11-3 is PRE-EXISTING, not a Phase 11 regression: it reproduces identically on b69e3b2 (the commit before Phase 11 began, verified in a throwaway worktree); the line number moved only because Phase 11 added lines above it. It is closed under Phase 11 by explicit user decision, so LAZY-01..05 and CONS-01 are not credited with fixing it — this plan's own requirements: [LAZY-04] covers Task 2 only, and G-11-3 is tracked separately via gap_ids."
  - "G-11-3 was three separate defects sharing one root cause (protojson.Unmarshal resetting m.head before populating it): (1) the nil-map panic on Init's write, (2) MergeLock silently discarding Init's own CONFIGSPACE merge by reloading before writing — the {\"url\":\".\",\"deps\":{}} row that never panicked but persisted nothing, and (3) modInitCommand.Run swallowing LoadFromLockFile's parse error, which — if only defects 1 and 2 were fixed — would have made mod init exit 0 after overwriting a corrupt lock file with a fresh one, strictly worse for the operator than the crash."
  - "The invariant is restored at LoadFromLockFile alone (the one chokepoint all eight production callers route through), not with nil-checks in each caller — matching the plan's own reasoning."
  - "Init now self-loads the lock file at its own start and returns that error immediately, making it self-sufficient; the now-redundant, error-swallowing LoadFromLockFile call in modInitCommand.Run was deleted rather than fixed in place."
  - "Task 2's concurrency test stands in for mod sync using the eager GetProtoRegistry() instance rather than ModuleService.Sync itself, because Sync's own utils.NewDescriptorRegistry() sits behind a Download step that needs the network. The instance-separation property is identical in both cases; only the stand-in is executed, which is why the corresponding must_haves entry stays tagged verification: backstop in the plan."

requirements-completed: [LAZY-04]

coverage:
  - id: D1
    description: "mod init / mod tidy survive every protoconf.lock shape (missing deps key, empty object, explicit empty deps, absent file, zero-byte, truncated, unknown-key) without a nil-map panic, and never regenerate a lock file that failed to parse"
    requirement: "LAZY-04"
    verification:
      - kind: integration
        ref: "mod/command_test.go#TestModInitLockFileShapes (8 subtests: no_deps_key, empty_object, explicit_empty_deps, absent, zero_byte, truncated_json, unknown_key, merges_existing_entry)"
        status: pass
      - kind: integration
        ref: "go test -race ./mod/... ./compiler/... -count=1"
        status: pass
    human_judgment: false
  - id: D2
    description: "mod sync's eager .fds serialization is byte-identical and checksum-identical to a pre-concurrency golden while 6 lazy CompileFile calls run concurrently over the same protoconf root, under -race, with the two DescriptorRegistry instances asserted distinct"
    requirement: "LAZY-04"
    verification:
      - kind: integration
        ref: "compiler/lib/mod_sync_fds_test.go#TestModSyncFdsUnaffectedByConcurrentLazyCompile"
        status: pass
    human_judgment: true
    rationale: "This test stands in for the real mod sync path (ModuleService.Sync constructs its own registry behind a network-dependent Download step) using the reachable eager GetProtoRegistry() instance instead. The instance-separation mechanism is identical, but the true Sync path itself was never executed, which is why the plan's own must_haves entry for this truth is tagged verification: backstop rather than a plain pass — a human should confirm the stand-in is an acceptable proxy for the real call path."

duration: 45min (Task 1 in a prior, separately-dispatched session; Task 2 executed in this resumed session after a harness-watchdog stall — see Issues Encountered)
completed: 2026-09-07
status: complete
---

# Phase 11 Plan 04: Close G-11-3 (protoconf.lock nil-map panic) and prove mod sync's .fds survives a concurrent lazy compile Summary

**Restored the non-nil `Deps` invariant at `LoadFromLockFile` (the one chokepoint all eight production callers route through), stopped `MergeLock` from discarding `Init`'s own `CONFIGSPACE` merge, and added a genuinely concurrent test proving `mod sync`'s eager `.fds` bytes are unmoved by an in-process lazy compile running at the same time.**

## Performance

- **Duration:** ~45 min total across two dispatches (Task 1's exact wall time is not tracked separately; this resumed session for Task 2 ran ~15–20 min of active tool use)
- **Completed:** 2026-09-07
- **Tasks:** 2/2
- **Files modified:** 4 (`compiler/lib/module_service.go`, `mod/command.go`, `mod/command_test.go` created, `compiler/lib/mod_sync_fds_test.go`)

## Accomplishments

- `LoadFromLockFile` now captures `protojson.Unmarshal`'s error into a local and unconditionally re-initializes `m.head.Deps` to `map[string]*module.RemoteRepo{}` when nil, before returning the captured error — so every one of the eight production callers, whether or not it checks the returned error, can never observe a nil map.
- `Init` now begins with its own `m.LoadFromLockFile()` call and returns that error immediately, making it self-sufficient and ensuring a lock file that cannot be parsed aborts before any write. `MergeLock`'s body reduced to `return m.Lock()` — the redundant reload that discarded `Init`'s own merge is gone.
- `modInitCommand.Run`'s now-dead, error-swallowing `LoadFromLockFile` call was deleted; `Init`'s own returned error now flows into the existing error-handling block.
- New `mod/command_test.go` (mod's first test file): `TestModInitLockFileShapes`, 8 table-driven CLI-level subtests covering every lock-file shape named in the plan, each driving the real `modInitCommand.Run` entry point against its own `t.TempDir()`.
- New `TestModSyncFdsUnaffectedByConcurrentLazyCompile` in `compiler/lib/mod_sync_fds_test.go`: a golden `.fds` is captured from an eager `ModuleService`'s registry before any concurrency starts, then an `errgroup` runs 6 lazy `Compiler.CompileFile` calls concurrently with 6 goroutines each calling `reg.Store()` 3 times into uniquely named paths — every stored file matches the golden bytes and checksum, `require.NotSame` pins the two `*utils.DescriptorRegistry` instances as distinct, and `LoadedFileCount()` bounds (`> 6` and `< 40`) rule out a vacuous pass against an idle or fully-loaded registry.

## Task Commits

1. **Task 1: mod init/tidy survive every protoconf.lock shape — end-to-end** - `baf8308` (test, RED) + `243ab6f` (fix, GREEN) — completed in a prior, separately-dispatched executor session.
2. **Task 2: mod sync's .fds bytes are unmoved by a concurrent in-process lazy compile** - `c4ee904` (test) — completed in this resumed session.

**Plan metadata:** committed alongside STATE.md/ROADMAP.md updates below.

## Files Created/Modified

- `mod/command_test.go` (new, Task 1) - `TestModInitLockFileShapes`, 8 subtests pinning every `protoconf.lock` shape's exit code and on-disk outcome.
- `compiler/lib/module_service.go` (Task 1) - `LoadFromLockFile` restores the `Deps` invariant on every path; `Init` self-loads the lock file first; `MergeLock` reduced to `m.Lock()`.
- `mod/command.go` (Task 1) - deleted `modInitCommand.Run`'s redundant, error-swallowing `LoadFromLockFile` call.
- `compiler/lib/mod_sync_fds_test.go` (Task 2) - appended `TestModSyncFdsUnaffectedByConcurrentLazyCompile`; `TestModSyncFdsByteIdentical` (the sequential half of the same truth) left unmodified.

## Decisions Made

See `key-decisions` in frontmatter. In brief: G-11-3 is recorded as pre-existing (reproduces on `b69e3b2`) so the requirements ledger does not credit Phase 11's own LAZY/CONS work with fixing it; the fix lives entirely at the `LoadFromLockFile` chokepoint rather than in per-caller nil-checks; and Task 2's concurrency test is an honest stand-in for `mod sync`'s real (network-gated) `Sync` path rather than an exercise of `Sync` itself.

## Deviations from Plan

None — plan executed exactly as written. Both tasks' `<action>` steps were followed verbatim, including the mandated mutation-verification for each edit.

## Mutation Verification (both tasks)

**Task 1** (performed by the prior session; recorded here per the plan's requirement that both tasks' observations live in this SUMMARY):
1. Reverting the `LoadFromLockFile` `Deps`-reinitialization edit alone: the `no_deps_key`, `empty_object`, `absent`, and `merges_existing_entry` subtests went red with the original nil-map panic. Restored — green.
2. Reverting the `MergeLock` reduction alone (restoring its reload-then-write body): the `explicit_empty_deps` subtest went red because the dependency was missing from the rewritten lock. Restored — green.
3. Reverting the `modInitCommand.Run` deletion alone (restoring the unchecked `LoadFromLockFile` call): the `zero_byte` and `truncated_json` subtests went red because the lock file was regenerated instead of left untouched. Restored — green.

**Task 2** (performed in this session): changed the concurrent `Store` target from the eager registry (`reg`) to the lazy compiler's own registry (`c.ModuleService.GetProtoRegistry()`), so both halves shared one instance. Observed result, reproduced across 3 consecutive `-race` runs:
```
Store checksum drifted: got d41d8cd98f00b204e9800998ecf8427e, want 67f038fed8b829d532337dd570d79b4d
```
i.e. the golden taken before the compiles started no longer matched — the bytes visibly drifted as the shared registry's lazy parse populated it mid-`Store`. The `-race` detector did not additionally report a `WARNING: DATA RACE` block in the runs observed (race detection is sampling-based and not guaranteed on every interleaving); the byte/checksum drift alone is a sufficient, reproducible red signal and is what is recorded here rather than an unobserved race report. Restored the target to `reg` — green again, confirmed with `go test -race -count=2` (2 full iterations, both tests, no race warning).

## Issues Encountered

**Prior executor stalled on a harness watchdog after completing Task 1.** Task 1 (commits `baf8308`, `243ab6f`) was fully committed and verified — `go build ./...` clean, `go vet ./compiler/... ./mod/...` clean, `TestModInitLockFileShapes` green with all 8 subtests passing, and the full suite green under `-race` (agent package excluded, pre-existing hang) — before the executor stalled and had to be resumed as a fresh dispatch for Task 2. Task 1's committed work was left untouched by this resumed session; only Task 2 (`compiler/lib/mod_sync_fds_test.go`) was added and committed here.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `G-11-3` is closed: no input shape of `protoconf.lock` can reach `m.head.Deps[name] = msg` with a nil map from any of the eight production `LoadFromLockFile` callers, and `mod init` / `mod tidy` now have exactly two outcomes — persist the `CONFIGSPACE` dependencies and exit 0, or report the lock file's parse error and exit 1 leaving the file untouched.
- UAT item 3's backstop truth is now executed under `-race` rather than merely reasoned about, closing the last open gap from `11-UAT.md`.
- Full suite (`go test -race -count=1 $(go list ./... | grep -v '/agent$')`) is green across all 19 non-agent packages, including `mod` (6.595s) and `compiler/lib` (127.588s).
- No blockers. Phase 11's `## Gaps` in `11-UAT.md` is now fully closed.

## Self-Check: PASSED

All created/modified files confirmed present on disk (`mod/command_test.go`, `compiler/lib/mod_sync_fds_test.go`, `compiler/lib/module_service.go`, `mod/command.go`); all three commit hashes (`baf8308`, `243ab6f`, `c4ee904`) confirmed present in `git log`.

---
*Phase: 11-concurrency-safe-lazy-registry-core*
*Completed: 2026-09-07*
