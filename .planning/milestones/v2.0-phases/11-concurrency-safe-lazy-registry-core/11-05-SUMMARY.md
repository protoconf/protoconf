---
phase: 11-concurrency-safe-lazy-registry-core
plan: 05
subsystem: compiler (module/dependency management), mod CLI, testing
tags: [gap-closure, module-service, mod-cli, descriptor-registry, lock-file-integrity]

# Dependency graph
requires:
  - phase: 11-04 (concurrency-safe-lazy-registry-core)
    provides: LoadFromLockFile's restored non-nil Deps invariant, mod's first test file (mod/command_test.go) and its runModInit/readLockDeps harness shape this plan's runModSync/runModInitIn mirror
provides:
  - "GenFileDescriptorSet guard: registry.LocalFileCount() == 0 returns ErrorRemoteRepoNoProtoFiles before any Store call, so an empty parse can never become a persisted zero-byte .fds or a corrupted lock checksum"
  - "walk() accumulates recursive errors instead of replacing them, so a later-processed succeeding dependency can no longer erase an earlier failure's non-zero exit code"
  - "utils.DescriptorRegistry.LocalFileCount() -- read-only accessor exposing the exact set Store is about to serialize"
  - "TestModSyncNeverPersistsEmptyDescriptorSet -- 3-subtest mod-CLI-level regression suite pinning G-11-7 closed"
affects: [mod, compiler/lib, utils, requirements-ledger (LAZY-04, G-11-7)]

# Actuals (#2632) -- pairs with the plan's estimate to calibrate future estimates.
actuals:
  tokens: 3650    # chars/4 over the realized diff (14602 chars across all 4 changed files, both tasks) -- plan estimated 65000
  tasks: 2
  commits: 3      # 3f2ba32 (test, RED), 7ca3a16 (feat, GREEN), fc7bfe3 (test, Task 2 fencing)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Guard at the single point that converts an empty parse into persisted corruption (between Import and Store), rather than widening Import/find/Parse's tolerance of a missing path -- keeps the eager-consumer blast radius D-01 fenced off"
    - "Mutation-verification as phase standard, continued from 11-01..04: every edit proven by reverting it alone and observing the specific red signal, then restored"
    - "Fencing tests for a fix that already exists: Task 2 adds regression tests proving the guard can't be narrowed to a GetterUrl check and the walk() fix can't be reverted, without changing production code -- verified by mutating the implementation, not by writing tests before the code existed"

key-files:
  created: []
  modified:
    - utils/utils.go
    - compiler/lib/module_service.go
    - compiler/lib/module_service_test.go
    - mod/command_test.go

key-decisions:
  - "G-11-7 is PRE-EXISTING, not a Phase 11 regression: reproduces byte-identically on b69e3b2 (the commit before Phase 11 began, verified in a throwaway worktree during UAT). Closed under Phase 11 by explicit user decision, exactly as G-11-3 was in 11-04 -- LAZY-01..05 and CONS-01 are not credited with fixing it. This plan's requirements: [LAZY-04] reflects that mod sync's .fds output is LAZY-04's subject matter; G-11-7 itself is tracked via gap_ids."
  - "The guard is keyed on registry.LocalFileCount() == 0 (the exact set Store is about to serialize), never on r.GetterUrl == \"\". Measured during planning and re-confirmed here: a fully downloaded dependency with a correct getterUrl but a sourcePath absent from the extracted archive corrupts the lock identically to an unsynced dependency -- a GetterUrl check would leave that second shape broken."
  - "GetFileDescriptorSet() cannot serve as the emptiness signal because NewDescriptorRegistry seeds FileRegistry with ~65 well-known types before any parsing; LocalFileCount() reads localFiles instead, which Parse resets to empty at the top of every call and Store iterates directly."
  - "walk()'s one-word accumulator fix (errors.Join(walk(...)) -> errors.Join(err, walk(...))) is load-bearing independently of the guard: with the guard alone, a one-failing-one-succeeding sync could still exit 0 because the later-processed success erased the earlier failure."
  - "The keys/deps index mismatch in walk() (keys sorted, deps built in unsorted map-range order) is left alone, per the plan's explicit scope boundary. This plan's own Task 2 mutation-verification for the walk() fix (below) empirically confirmed the nondeterminism this causes: reverting the accumulator fix reproduced the pre-fix exit-0 bug in 4 of 5 runs, not all 5, because in the outlier run map iteration order happened to process the failing dependency last."
  - "TestModuleService_Sync's no integrity/good integrity/download deps fixtures gaining SourcePath: \"src\" is a strengthening, not a weakening: they previously omitted the field entirely (protoPaths skips empty path components), so they passed while parsing and storing nothing. The Starlark remote_repo builtin defaults SourcePath to \"src\", so no real lock file has the shape these fixtures used to have."

requirements-completed: [LAZY-04]

coverage:
  - id: D1
    description: "mod sync over a dependency whose resolved proto paths yield no files (unsynced, no getterUrl, OR downloaded with a bad sourcePath) fails the run with a message naming the dependency and the searched paths, writes no .fds, and leaves the lock's recorded fileDescriptorSetSum values byte-identical to their pre-sync values"
    requirement: "LAZY-04"
    verification:
      - kind: integration
        ref: "mod/command_test.go#TestModSyncNeverPersistsEmptyDescriptorSet (unsynced_dep_no_getter_url, downloaded_dep_bad_source_path subtests)"
        status: pass
      - kind: integration
        ref: "go test -race ./mod/... ./compiler/... ./utils/... -count=1"
        status: pass
      - kind: manual
        ref: "go run ./cmd/protoconf mod sync over a throwaway SmallTestDir copy with .fds deleted: exit 1, no .fds written, both lock sums intact at their committed values"
        status: pass
    human_judgment: false
  - id: D2
    description: "mod sync's good path is unchanged: mod init then mod sync over the fixture exits 0, writes both non-empty .fds files, and restores exactly the checksums the committed lock already records"
    requirement: "LAZY-04"
    verification:
      - kind: integration
        ref: "mod/command_test.go#TestModSyncNeverPersistsEmptyDescriptorSet/good_path_control"
        status: pass
      - kind: manual
        ref: "go run ./cmd/protoconf mod init then mod sync over a throwaway SmallTestDir copy: exit 0, 15M terraform_repo.fds + 2.7K vizceral_repo.fds, sums restored to 6556e5cf.../039f1e10..."
        status: pass
    human_judgment: false
  - id: D3
    description: "a mod sync in which one dependency fails and one succeeds exits non-zero -- the succeeding dependency does not erase the earlier failure's exit code"
    requirement: "LAZY-04"
    verification:
      - kind: integration
        ref: "mod/command_test.go#TestModSyncNeverPersistsEmptyDescriptorSet/downloaded_dep_bad_source_path (exit-code assertion)"
        status: pass
    human_judgment: true
    rationale: "The walk() accumulator fix is genuinely load-bearing (confirmed by mutation-verification below), but its own regression test's red signal is probabilistic: walk()'s pre-existing, explicitly out-of-scope keys/deps map-ordering nondeterminism means reverting the fix reproduces the bug in most but not all runs (4 of 5 observed here). A human should judge whether this residual flakiness in the mutation-proof (not in the fix itself, which is unconditionally correct) is acceptable, since fixing the ordering bug is out of this plan's scope by explicit instruction."

duration: ~50 min (single continuous session)
completed: 2026-09-07
status: complete
---

# Phase 11 Plan 05: Close G-11-7 (mod sync silently persists an empty descriptor set) Summary

**Added a single guard at the only point that converts an empty proto parse into persisted lock corruption, fixed `walk()`'s error-swallowing accumulator, and pinned both reproduced corruption triggers plus a good-path control at the `mod sync` CLI level.**

## Performance

- **Duration:** ~50 min (single continuous session)
- **Completed:** 2026-09-07
- **Tasks:** 2/2
- **Files modified:** 4 (`utils/utils.go`, `compiler/lib/module_service.go`, `compiler/lib/module_service_test.go`, `mod/command_test.go`)

## Accomplishments

- New `utils.DescriptorRegistry.LocalFileCount()` -- a read-only, `RLock`-guarded accessor reporting the size of `localFiles`, the exact set `Store` is about to serialize. Sibling of the existing `LoadedFileCount`; `GetFileDescriptorSet()` could not serve this purpose since it ranges `FileRegistry`, pre-seeded with ~65 well-known types and never zero.
- New `ErrorRemoteRepoNoProtoFiles` sentinel in `compiler/lib/module_service.go`. `GenFileDescriptorSet` now returns it (joined with the existing label-bearing error, plus the searched paths) when `registry.LocalFileCount() == 0`, placed between the `Import` error check and the `Store` call -- so no zero-byte `.fds` is ever written and no empty-md5 checksum is ever computed or persisted by the closing `Walk`. The guard is keyed on the resolved paths yielding nothing, not on `GetterUrl == ""`, so it catches both reproduced triggers: an unsynced dependency and a downloaded dependency with a bad `sourcePath`.
- `walk()`'s error accumulator changed from `err = errors.Join(walk(deps[i], walkFn))` to `err = errors.Join(err, walk(deps[i], walkFn))` -- one word, but load-bearing: without it, a later-processed succeeding dependency's `nil` erased an earlier dependency's failure and `mod sync` could still exit 0 after a real corruption-preventing failure.
- `TestModuleService_Sync`'s `no integrity`, `good integrity` and `download deps` fixtures gained `SourcePath: "src"` -- correcting three hand-built fixtures that previously resolved to an empty path list (and so passed while parsing and storing nothing) to match the Starlark `remote_repo` builtin's real default.
- New `TestModSyncNeverPersistsEmptyDescriptorSet` in `mod/command_test.go`, three subtests, all driving the real `modSyncCommand.Run`/`modInitCommand.Run` CLI entry points:
  - `unsynced_dep_no_getter_url` -- the exact `G-11-7` reproduction: both fixture deps have no `getterUrl`, cache emptied. Now exits non-zero, writes no `.fds`, leaves both recorded sums untouched.
  - `downloaded_dep_bad_source_path` -- proves the guard is keyed on resolved paths, not `GetterUrl`: a genuinely downloaded dependency with a bad `sourcePath` still corrupts the lock if the guard were narrowed.
  - `good_path_control` -- non-vacuity control: `mod init` + `mod sync` exits 0 and reproduces the committed fixture's exact checksums (`6556e5cfb73f535f9b91738dbd0df926`, `039f1e1023250b34054894cc58bc8b2b`).
- New `runModSync` and `runModInitIn` helpers, siblings of 11-04's `runModInit`, plus small `deleteFdsFiles`/`writeLockDeps` helpers.

## Task Commits

1. **Task 1: mod sync refuses to persist an empty descriptor set -- end-to-end**
   - `3f2ba32` (test, RED) -- `TestModSyncNeverPersistsEmptyDescriptorSet`'s `unsynced_dep_no_getter_url` subtest, written and run against unmodified code first; confirmed exit 0 (want non-zero), matching the exact UAT reproduction.
   - `7ca3a16` (feat, GREEN) -- `LocalFileCount()`, the `GenFileDescriptorSet` guard + `ErrorRemoteRepoNoProtoFiles` sentinel, the `walk()` accumulator fix, and the three fixture corrections.
2. **Task 2: fence the guard against being narrowed -- second trigger and good-path control**
   - `fc7bfe3` (test) -- `downloaded_dep_bad_source_path` and `good_path_control` subtests appended. No production code changed; both subtests passed immediately since Task 1's guard already covers both shapes.

**Plan metadata:** committed alongside STATE.md/ROADMAP.md updates below.

## Files Created/Modified

- `utils/utils.go` -- new `LocalFileCount()` method; `Import`, `find`, `Parse`, `Store` unchanged.
- `compiler/lib/module_service.go` -- new `ErrorRemoteRepoNoProtoFiles` sentinel; `GenFileDescriptorSet` gains the emptiness guard before `Store`; `walk()` accumulates errors.
- `compiler/lib/module_service_test.go` -- `no integrity`, `good integrity`, `download deps` fixtures gain `SourcePath: "src"`.
- `mod/command_test.go` -- new `TestModSyncNeverPersistsEmptyDescriptorSet` (3 subtests), `runModSync`, `runModInitIn`, `deleteFdsFiles`, `writeLockDeps`. `TestModInitLockFileShapes` and its helpers untouched.

## Decisions Made

See `key-decisions` in frontmatter. In brief: `G-11-7` is recorded as pre-existing (reproduces on `b69e3b2`), the guard is keyed on `LocalFileCount()` rather than `GetterUrl` so it catches both reproduced corruption triggers, the `walk()` one-word fix is independently load-bearing, the pre-existing `keys`/`deps` index-mismatch nondeterminism was deliberately left alone per the plan's scope boundary, and the three `SourcePath`-added fixtures are a strengthening rather than a weakening.

## Deviations from Plan

None -- plan executed exactly as written. Both tasks' `<action>` steps were followed verbatim, including the mandated mutation-verification for every edit.

## Mutation Verification

**Task 1** (action step 6):
1. Reverted the `GenFileDescriptorSet` guard alone (edit 3): `unsynced_dep_no_getter_url` went red -- exit 0, two zero-byte `.fds` written, both lock sums rewritten to `d41d8cd98f00b204e9800998ecf8427e`. Exactly the pre-fix bug. Restored -- green.
2. Reverted the `SourcePath: "src"` fixture correction alone (edit 5): `TestModuleService_Sync`'s `no_integrity` and `good_integrity` subtests went red on the new `ErrorRemoteRepoNoProtoFiles` sentinel (`wantErr <nil>`, got the sentinel-wrapped error). Confirms these fixtures previously exercised the empty-path bug silently. Restored -- both green, full `TestModuleService_Sync` (5/5 subtests, including network-dependent `download deps`) green.

**Task 2** (action step, three observations):
1. Narrowed the guard to `r.GetterUrl == "" && registry.LocalFileCount() == 0`: `downloaded_dep_bad_source_path` went red (exit 0 again, since the repository is downloaded and `GetterUrl` is set). `good_path_control` stayed green throughout. Restored -- green.
2. Reverted the `walk()` accumulator fix alone: `downloaded_dep_bad_source_path` went red on its exit-code assertion in 4 of 5 repeated runs (`go test -count=1` run five times). The 1-of-5 pass is explained by `walk()`'s pre-existing, plan-flagged `keys`/`deps` map-iteration-order nondeterminism -- in that run, Go's randomized map ranging happened to process the failing dependency last, so even the buggy assign-not-accumulate code retained its error. This is recorded as an honest, probabilistic finding (see coverage `D3`'s `human_judgment: true` rationale) rather than papered over; it confirms the `walk()` fix is genuinely load-bearing without overclaiming full determinism, and fixing the ordering bug itself is out of this plan's scope by explicit instruction. Restored -- `downloaded_dep_bad_source_path` green, `good_path_control` unaffected throughout.
3. `good_path_control` was confirmed to stay green through both mutations above -- a control that does not move with the mutation.

## Verification Run Log

- `go test ./mod/... -run TestModSyncNeverPersistsEmptyDescriptorSet -v -count=1` -- PASS, all 3 subtests.
- `go test ./compiler/lib/ -run TestModuleService_Sync -v -count=1` -- PASS, all 5 subtests including network-dependent `download_deps` (network available in this environment; confirmed via a live GitHub HTTP check before running).
- `go vet ./mod/... ./compiler/... ./utils/...` -- clean.
- `go test -race ./mod/... ./compiler/... ./utils/... -count=1` -- all green.
- `go test -race -count=1 $(go list ./... | grep -v '/agent$')` -- all 19 non-agent packages green (the `agent` package's pre-existing, unrelated hang at `agent/command_test.go:118` remains out of scope, per plan).
- Manual reproduction 1: `mod sync` over a throwaway `SmallTestDir()` copy with `.fds` deleted -- exit 1, `.protoconf_cache/` empty, both lock sums intact at `6556e5cfb73f535f9b91738dbd0df926` / `039f1e1023250b34054894cc58bc8b2b`. This is the exact `G-11-7` reproduction from the UAT report, now fixed.
- Manual reproduction 2: `mod init` then `mod sync` over the same copy -- exit 0, `terraform_repo.fds` 15M, `vizceral_repo.fds` 2.7K, sums restored to the committed values.

## Issues Encountered

None.

## User Setup Required

None -- no external service configuration required.

## Next Phase Readiness

- `G-11-7` is closed: no `mod sync` run can write a zero-byte `.fds`, and no `mod sync` run can replace a recorded `fileDescriptorSetSum` with the checksum of an empty descriptor set. `mod sync` now has exactly two outcomes per dependency -- generate a non-empty descriptor set and record its real checksum, or fail with a message naming the dependency and the searched directories, surviving the walk to become a non-zero exit code.
- UAT test 7's prohibition 6 is now tested at the `mod sync` CLI level, the level at which the registry-level `TestModSyncFdsByteIdentical` was structurally unable to see this gap.
- `Import`, `find`, `Parse` and `Store` in `utils/utils.go` are unchanged; D-01's fence around eager consumers (`server/server.go:347`, `utils/utils.go:365`'s D-03 fallback, `compiler/lib/module_service.go:434`'s eager build) holds -- confirmed by the full non-agent suite staying green under `-race`.
- Phase 11's `## Gaps` in `11-UAT.md` (both `G-11-3` and `G-11-7`) are now fully closed.
- No blockers.

## Self-Check: PASSED

All modified files confirmed present on disk (`utils/utils.go`, `compiler/lib/module_service.go`, `compiler/lib/module_service_test.go`, `mod/command_test.go`); all three commit hashes (`3f2ba32`, `7ca3a16`, `fc7bfe3`) confirmed present in `git log`.

---
*Phase: 11-concurrency-safe-lazy-registry-core*
*Completed: 2026-09-07*
