---
phase: quick-260904-f5j
plan: 01
subsystem: compiler
tags: [validators, starlark-loader, filesystem-walk, correctness, lazy-loading-prep]

requires: []
provides:
  - "loadValidators discovers *.proto-validator files by walking srcDir, independent of the descriptor registry's contents"
affects: [future-lazy-proto-loading]

actuals:
  tokens: 1626
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "filepath.WalkDir over srcDir for filesystem-driven discovery, replacing registry-driven RangeFiles + per-descriptor stat"

key-files:
  created:
    - compiler/lib/starlark_loader_test.go
  modified:
    - compiler/lib/starlark_loader.go
    - compiler/lib/filesystem.go
    - compiler/lib/filesystem_js.go

key-decisions:
  - "Deleted the now-orphaned stat() helper from both filesystem.go and filesystem_js.go (deliberate scope widening beyond loadValidators itself), because its only caller was the RangeFiles block being replaced and CI's golangci-lint only-new-issues gate would flag a fresh unused (U1000) finding otherwise. mkdirAll/openFile/writeFile were left untouched in both files."
  - "Named the new orphan-validator test fixture zz_no_such_proto.proto-validator, not no_such_proto.proto-validator as the plan's prose suggested, so it sorts lexically after the pre-existing src/test.proto-validator. Discovered during TDD confirmation: add_validator (starlark_functions.go) keeps only the most-recently-registered function per message name in a plain map, so once two validator files both target ValidateMe, whichever the walk visits last wins. Walk order is now correctly deterministic (this task's whole point) but that determinism meant the plan's literal filename would have been silently overwritten by test.proto-validator's own validators and the new test would have passed vacuously. Renaming is a test-fixture-only change; add_validator's last-write-wins semantics are pre-existing, unrelated production behavior and were left untouched per scope boundaries."

patterns-established:
  - "Filesystem-driven discovery under srcDir is now the validator-loading strategy of record, ahead of the planned registry shrink (864 -> ~7 files) from lazy proto loading."

requirements-completed: [QUICK-260904-f5j]

coverage:
  - id: D1
    description: "loadValidators walks srcDir for *.proto-validator files instead of ranging the descriptor registry"
    requirement: "QUICK-260904-f5j"
    verification:
      - kind: unit
        ref: "go test ./compiler/lib/ -run TestCompiler_CompileFile -count=1 (all four pre-existing validator subtests keep their outcomes)"
        status: pass
  - id: D2
    description: "A .proto-validator whose companion .proto is absent from the registry is now discovered and enforced"
    requirement: "QUICK-260904-f5j"
    verification:
      - kind: unit
        ref: "go test ./compiler/lib/ -run TestLoadValidators -count=1 -v (TestLoadValidators_OrphanValidatorIsDiscovered, TestLoadValidators_DirectoryShapedLikeValidator)"
        status: pass
      - kind: regression-guard-confirmation
        ref: "Both new subtests were re-run against the pre-Task-1 starlark_loader.go/filesystem.go/filesystem_js.go (via git show 20d6521:... written to disk, tests run, then Task 1's committed files restored) -- both FAILED, confirming they exercise genuinely new behaviour rather than being tautologies"
        status: pass
        human_judgment: false

duration: ~20min (18cc828 to c8a6ff6, 5m09s wall between commits; total session including reading/verification longer)
completed: 2026-09-04
status: complete
---

# Quick Task 260904-f5j: Fix loadValidators to Walk the Filesystem Summary

**Inverted `loadValidators` from ranging the descriptor registry (with a stat-per-descriptor lookup) to walking `srcDir` directly for `*.proto-validator` files — closing a silent-skip correctness gap ahead of the planned lazy-proto-loading registry shrink, and making discovery cheaper and deterministic in the process.**

## Performance

- **Duration:** ~5 minutes between the two task commits (18cc828 → c8a6ff6); total wall time including reading, TDD confirmation, and verification was longer
- **Completed:** 2026-09-04
- **Tasks:** 2 (Task 1: filesystem walk + dead-code removal; Task 2: regression test)
- **Files modified:** 3 (starlark_loader.go, filesystem.go, filesystem_js.go); 1 created (starlark_loader_test.go)

## Accomplishments

- `loadValidators` now walks `l.srcDir` via `filepath.WalkDir`, matching files by `consts.ProtoExtension + consts.ValidatorExtensionSuffix` (composed, not hardcoded), instead of ranging `l.parser.FilesResolver.RangeFiles` and `stat`-ing a sibling path per descriptor.
- **Free improvement #1 — deterministic walk order:** `RangeFiles` iterated a map, whose order is unspecified in Go; `filepath.WalkDir` visits entries in lexical order, so validator loading order is now reproducible run-to-run.
- **Free improvement #2 — orphan validators enforced:** a `.proto-validator` file whose companion `.proto` is not reachable from the descriptor registry (e.g. under future lazy loading, or simply a typo'd sibling name) is now found and executed, rather than silently skipped. This was the correctness motivation for the task: a config system that silently approves what should fail validation is the worst failure mode.
- Preserved every existing behaviour: the not-exist-root case (`srcDir` absent → zero validators, no error), the directory-shaped-validator error (`expected validator file, got directory: %s`, same wording), and symlink-follows-through semantics (checking only `IsDir()`, not `d.Type()`).
- Deleted the now-dead `stat()` helper from both `compiler/lib/filesystem.go` and `compiler/lib/filesystem_js.go` — its only caller was the `RangeFiles` block being replaced, and CI's golangci-lint `only-new-issues: true` gate would flag a freshly-orphaned unexported function as a new `unused` finding. `mkdirAll`, `openFile`, `writeFile` were left untouched in both files, keeping the build-tagged pair symmetric.
- Removed the now-unused `google.golang.org/protobuf/reflect/protoreflect` import (its only use was the `RangeFiles` callback parameter) and added `io/fs`.

## Task Commits

1. **Task 1: Walk srcDir for validator files instead of ranging the registry** — `18cc828` (feat)
2. **Task 2: Regression test for filesystem-driven validator discovery** — `c8a6ff6` (test)

## Files Created/Modified

- `compiler/lib/starlark_loader.go` — `loadValidators` rewritten to `filepath.WalkDir`; import list updated
- `compiler/lib/filesystem.go` — `stat()` deleted
- `compiler/lib/filesystem_js.go` — `stat()` deleted (build-tagged js/wasm sibling)
- `compiler/lib/starlark_loader_test.go` — new file; two subtests

## Decisions Made

- **Delete `stat()` from both filesystem files (scope widening beyond `loadValidators`):** justified purely by the CI lint gate (`only-new-issues: true` against an 84-issue backlog would flag a new U1000 `unused` finding). Not a general refactor of the pair — `mkdirAll`/`openFile`/`writeFile` are untouched.
- **Test fixture renamed from the plan's literal `no_such_proto.proto-validator` to `zz_no_such_proto.proto-validator`:** discovered while running the mandated pre-Task-1 failure confirmation. `add_validator` (in `starlark_functions.go`, untouched, out of scope) stores validators in a plain `map[string]*starlark.Function` keyed by message name — the *last* `add_validator` call for a given message wins, with no accumulation. `src/test.proto-validator` already registers three validators for `ValidateMe`, ending on `validateme_map_validator`. Under alphabetical walk order, `no_such_proto.proto-validator` (starts with `n`) would be visited *before* `test.proto-validator` (starts with `t`), so the new orphan validator's registration would be immediately overwritten by the pre-existing fixture's own validators — and the test would pass "by accident" (returning `ErrInvalidConfig` due to the *existing* validators, not because the orphan was ever enforced), or in this specific case actually returns `nil` and fails, since `validator_passing_test.pconf` satisfies the pre-existing map-check validator. Prefixing with `zz_` guarantees the orphan file is visited last, so its always-failing rule is the one genuinely observed. This is a test-fixture-only change; no production code depends on the filename.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug in test fixture, discovered via TDD] Orphan-validator test fixture name would have collided with an existing fixture's validator registration**
- **Found during:** Task 2, while executing the plan's mandated pre-Task-1 failure confirmation
- **Issue:** The plan specified `src/no_such_proto.proto-validator` as the orphan fixture name. Because `add_validator` keeps only the last-registered function per message name, and `no_such_proto.proto-validator` sorts alphabetically before the existing `src/test.proto-validator` (both target `ValidateMe`), the orphan's always-failing validator would be overwritten by `test.proto-validator`'s own validators during the walk, making the test either vacuous or (as observed) failing even after Task 1's fix.
- **Fix:** Renamed the fixture to `zz_no_such_proto.proto-validator` so it is walked and registered after `test.proto-validator`, guaranteeing its rule is the one enforced. Documented the reasoning in a code comment on the fixture write.
- **Files modified:** `compiler/lib/starlark_loader_test.go`
- **Commit:** `c8a6ff6`

## TDD Confirmation (per orchestrator constraint)

Both `TestLoadValidators_*` subtests were run against the pre-Task-1 implementation to rule out tautologies:

1. Copied `compiler/lib/{starlark_loader,filesystem,filesystem_js}.go` from commit `20d6521` (the phase's expected base, one commit before Task 1) over the working tree, keeping the new test file.
2. Ran `go test ./compiler/lib/ -run TestLoadValidators -count=1 -v`. **Both subtests FAILED**: `TestLoadValidators_OrphanValidatorIsDiscovered` got `nil` instead of `ErrInvalidConfig` (the orphan was never reachable via the old registry-ranging code), and `TestLoadValidators_DirectoryShapedLikeValidator` also got `nil` (the old code only checked `stat()` on paths derived from registered descriptors' names, never visited a `*.proto-validator`-suffixed directory sitting elsewhere in `srcDir`).
3. Restored the three files to Task 1's committed state (byte-identical `git diff` afterward) and re-ran the full suite — both subtests pass.

This confirms both subtests are genuine regression guards, not tautologies.

## Verification Results

- `go build ./...` — clean.
- `go test ./compiler/lib/ -count=1` — fully green (including `TestModuleService_Sync`'s slower `download_deps` subtest and both new `TestLoadValidators_*` subtests). One pre-existing skip unrelated to this change: `load_remote_with_load_local.pconf` (stale remote module pin, documented in `compiler_test.go` with a `ponytail:` comment predating this task).
- `go vet ./compiler/...` — **NOT clean**, but the one reported issue is pre-existing and unrelated to this task: `compiler/lib/compiler.go:355:20: literal copies lock value from c.ModuleService.GetProtoRegistry().MessageRegistry: ... contains sync.RWMutex`. Confirmed present identically at the baseline commit `20d6521` (before any of this task's edits, via `git stash`/`go vet`/`git stash pop`). `compiler.go` is not in this plan's `files_modified` list and was not touched. Left untouched per the scope boundary ("Only auto-fix issues DIRECTLY caused by the current task's changes").
- `grep -rn "func stat(" compiler/lib/` — no matches.
- `grep -rn "RangeFiles" compiler/lib/starlark_loader.go` — no matches (other `RangeFiles` callers elsewhere in the tree are untouched).
- `gofmt -l` on all four changed/created files — no output (already formatted).

## Known Issues (pre-existing, out of scope)

- `go vet ./compiler/...` reports one pre-existing lock-copy finding in `compiler/lib/compiler.go:355`, present before this task and unrelated to `loadValidators`. Logged to `.planning/WINDOWS.md`.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- Validator discovery is now filesystem-driven and ready for the planned lazy proto loading work (registry shrink from 864 to ~7 files) without silently dropping cross-field validators for protos the registry no longer eagerly loads.
- The pre-existing `go vet` finding in `compiler.go:355` and the stale `load_remote_with_load_local.pconf` module pin remain open, unrelated cleanup candidates for future quick tasks.

---
*Quick task: 260904-f5j*
*Completed: 2026-09-04*

## Self-Check: PASSED

- FOUND: compiler/lib/starlark_loader_test.go
- FOUND: .planning/quick/260904-f5j-fix-loadvalidators-to-walk-the-filesyste/260904-f5j-SUMMARY.md
- FOUND: commit 18cc828 in git log
- FOUND: commit c8a6ff6 in git log
- CONFIRMED: `grep -n "func stat(" compiler/lib/*.go` returns nothing
