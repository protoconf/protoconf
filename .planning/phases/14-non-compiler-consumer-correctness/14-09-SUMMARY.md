---
phase: 14-non-compiler-consumer-correctness
plan: 09
subsystem: security
tags: [path-traversal, filekv, mutation-server, tdd, containment-check]

# Dependency graph
requires:
  - phase: 14-non-compiler-consumer-correctness
    provides: filekv Get/Watch lazy-registry construction (14-02, 14-07) and MutateConfig's lazy TypeResolver marshal (14-03) that this plan's guards sit in front of
provides:
  - "agent/filekv.resolveKeyPath: the single validated key-to-path helper both Get and Watch route through"
  - "server.MutateConfig containment check rejecting an in.Path that escapes protoconfRoot/mutable_config, placed before every side effect"
  - "Two corrected threat-model rows (14-02 T-14-03, 14-07 T-14-21) that no longer claim a mitigation that did not exist"
affects: [gsd-secure-phase, future-consumer-correctness-phases]

# Actuals (#2632)
actuals:
  tokens: 5226
  tasks: 5
  commits: 5

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Single validated key-to-path helper (resolveKeyPath) shared by every call site that joins a caller key onto a filesystem root, so two call sites cannot drift apart (WR-03)"
    - "Containment check via filepath.Rel + HasPrefix(\"..\"+Separator), independent of any normalization/cleaned-form check, placed before every side effect a caller-controlled path could trigger"
    - "RED evidence captured verbatim in commit messages and SUMMARY: a security-control test is only credible once it is shown failing against the unguarded code"

key-files:
  created:
    - server/mutate_config_path_test.go
  modified:
    - agent/filekv/filekv.go
    - agent/filekv/filekv_test.go
    - server/server.go
    - .planning/phases/14-non-compiler-consumer-correctness/14-02-PLAN.md
    - .planning/phases/14-non-compiler-consumer-correctness/14-07-PLAN.md

key-decisions:
  - "Symlink-inside-protoconfRoot escape is accepted risk (T-14-24), not fixed: filepath.EvalSymlinks would close it at a stat-syscall cost on the agent's hot read path, and planting a symlink already requires write access to the config repo -- a strictly larger compromise than the unauthenticated remote read this plan closes"
  - "MutateConfig's containment check adds no normalization check (no './x'/'a/../b'/empty-path rejection) -- only containment is enforced, matching the plan's explicit instruction not to newly reject paths that resolve inside the base today"
  - "filekv's resolveKeyPath stays private to the filekv package; not hoisted for MutateConfig's second call site, since the two roots (protoconfRoot vs protoconfRoot/mutable_config) and failure shapes differ"

patterns-established:
  - "A test cited as a security control must be demonstrated capable of failing against the unguarded code before the guard lands -- the RED phase's --- FAIL: lines are recorded verbatim in the commit message and SUMMARY as the falsifiability proof"

requirements-completed: [CONS-03]

coverage:
  - id: D1
    description: "agent/filekv.Get and .Watch reject a caller-supplied key whose joined path resolves outside protoconfRoot, before any filesystem access or fsnotify registration"
    requirement: "CONS-03"
    verification:
      - kind: unit
        ref: "agent/filekv/filekv_test.go#TestGetRejectsTraversalKey"
        status: pass
      - kind: unit
        ref: "agent/filekv/filekv_test.go#TestWatchRejectsTraversalKey"
        status: pass
      - kind: unit
        ref: "agent/filekv/filekv_test.go#TestGetMissingKeyIsNotAnInvalidKey"
        status: pass
    human_judgment: false
  - id: D2
    description: "server.MutateConfig rejects a caller-supplied in.Path that escapes protoconfRoot/mutable_config, before the marshal, the pre-mutation script, or any filesystem write; a legitimate nested path still works"
    verification:
      - kind: unit
        ref: "server/mutate_config_path_test.go#TestMutateConfigRejectsTraversalPath"
        status: pass
      - kind: unit
        ref: "server/mutate_config_path_test.go#TestMutateConfigAllowsNestedPath"
        status: pass
      - kind: integration
        ref: "go test -race -count=1 ./test/... -run TestAuthFlow"
        status: pass
    human_judgment: false
  - id: D3
    description: "Two threat-model rows (14-02 T-14-03, 14-07 T-14-21) that falsely claimed the traversal guard was already mitigated/pinned now say something true"
    verification:
      - kind: other
        ref: "grep -q 14-09 in both 14-02-PLAN.md and 14-07-PLAN.md; node gsd-tools.cjs query verify.plan-structure task_count unchanged; frontmatter.validate --schema plan valid on both; numstat budget <= 6 added/deleted per file"
        status: pass
    human_judgment: false

duration: 15min
completed: 2026-09-08
status: complete
---

# Phase 14 Plan 09: Filekv and MutateConfig path-traversal containment (gap closure) Summary

**One shared `resolveKeyPath` helper closes the live filekv `Get`/`Watch` traversal read, and a containment check closes the weaker `MutateConfig` write-path escape the sibling audit found -- both proven with tests demonstrated failing against the unguarded code first.**

## Performance

- **Duration:** 15 min
- **Started:** 2026-09-08T22:52:28+07:00 (first task commit)
- **Completed:** 2026-09-08T23:07:41+07:00 (last task commit)
- **Tasks:** 5
- **Files modified:** 6 (1 created, 5 modified)

## Accomplishments

- Closed the recorded gap: `agent/filekv/filekv.go`'s `Get` and `Watch` now both route through one validated `resolveKeyPath` helper that rejects a leading `..` key segment via `filepath.Rel` containment (the old guard only rejected keys differing from their own `filepath.Clean` form, which a leading `..` cannot trigger).
- Strengthened `TestGetRejectsTraversalKey` and added `TestWatchRejectsTraversalKey` with a fixture that plants a real, readable, schema-valid materialized config outside `protoconfRoot` -- both tests are now falsifiable (they FAIL against the unfixed guard) rather than passing on `os.Stat`'s `ErrNotExist`.
- Widened scope at the user's direction to fix a weaker, unguarded instance of the same defect class on `server/server.go`'s `MutateConfig` write path: a caller-supplied `in.Path` could escape `protoconfRoot/mutable_config` into an `os.WriteFile` target with attacker-controlled content. Fixed with a containment check placed before the marshal, the pre-mutation script, and any filesystem write.
- Corrected two threat-model rows (`14-02-PLAN.md` T-14-03, `14-07-PLAN.md` T-14-21) that had falsely claimed this exact risk was already mitigated/pinned.

## Task Commits

Each task was committed atomically:

1. **Task 1: Prove the traversal end-to-end (RED)** - `6d35971` (test)
2. **Task 2: One validated key-to-path helper (GREEN)** - `211a643` (feat)
3. **Task 3: Prove the MutateConfig write escapes (RED)** - `d841035` (test)
4. **Task 4: Containment check on MutateConfig (GREEN)** - `042d031` (feat)
5. **Task 5: Correct the two false mitigation claims** - `ee398bb` (docs)

_No REFACTOR commits -- both GREEN implementations (`resolveKeyPath`, `MutateConfig`'s containment check) were already minimal; nothing needed cleanup._

## TDD Gate Compliance

| Task | Feature | RED | GREEN | REFACTOR | Status |
|------|---------|-----|-------|----------|--------|
| 1-2 | filekv `resolveKeyPath` | `6d35971` | `211a643` | — (not needed) | Pass |
| 3-4 | `MutateConfig` containment | `d841035` | `042d031` | — (not needed) | Pass |

**RED evidence, quoted verbatim (Task 1, against unmodified `agent/filekv/filekv.go`):**
```
=== RUN   TestGetRejectsTraversalKey
--- FAIL: TestGetRejectsTraversalKey (0.01s)
=== RUN   TestWatchRejectsTraversalKey
--- FAIL: TestWatchRejectsTraversalKey (0.01s)
```

**GREEN evidence (Task 2, after `resolveKeyPath` landed):**
```
--- PASS: TestGetRejectsTraversalKey (0.05s)
--- PASS: TestWatchRejectsTraversalKey (0.04s)
--- PASS: TestGetMissingKeyIsNotAnInvalidKey (0.04s)
```

**RED evidence (Task 3, against unmodified `server/server.go`)** -- reproduced the live write escape: `Written to filename=<T>/escaped.materialized_JSON` with the caller-controlled secret payload, `err=<nil>`:
```
--- FAIL: TestMutateConfigRejectsTraversalPath (0.57s)
--- PASS: TestMutateConfigAllowsNestedPath (0.56s)
```

**GREEN evidence (Task 4, after the containment check landed):**
```
--- PASS: TestMutateConfigRejectsTraversalPath (0.55s)
--- PASS: TestMutateConfigAllowsNestedPath (0.58s)
```

The `gsd_run check tdd-red-evidence` verb was not invoked for these RED phases: its TAP parser (`# tests`/`# pass`/`# fail` directives, `not ok N - <name>` lines) targets `node --test` output and does not parse Go's `--- FAIL: TestName` / `--- PASS: TestName` format, so it would misclassify every one of these runs as `zero_tests_discovered` regardless of whether the target test actually failed. The plan's own literal `<verify>` blocks encode the equivalent gate directly (`grep -q -- "--- FAIL: TestName"` against the unfixed code, required before the GREEN task may begin) and were run and satisfied as specified above -- see Deviations.

## Files Created/Modified

- `agent/filekv/filekv.go` - Added `ErrInvalidKey` sentinel and `resolveKeyPath`, the single key-to-path helper `Get` and `Watch` both call before any filesystem access or fsnotify registration
- `agent/filekv/filekv_test.go` - Added `newTraversalFixture` (plants a real file outside a temp `protoconfRoot`), rewrote `TestGetRejectsTraversalKey`, added `TestWatchRejectsTraversalKey` and `TestGetMissingKeyIsNotAnInvalidKey`
- `server/server.go` - `MutateConfig` now computes a `mutableConfigBase` and rejects any `in.Path` whose joined filename resolves outside it, checked immediately after `filename` is computed and before every side effect
- `server/mutate_config_path_test.go` (new) - `TestMutateConfigRejectsTraversalPath` (asserts file absence at the escape target, not merely an error) and `TestMutateConfigAllowsNestedPath` (pins that separators are not rejected)
- `.planning/phases/14-non-compiler-consumer-correctness/14-02-PLAN.md` - `T-14-03` row corrected, disposition now `transferred → 14-09`; one line appended to `<prohibition_breadcrumbs>`
- `.planning/phases/14-non-compiler-consumer-correctness/14-07-PLAN.md` - `T-14-21` row corrected, disposition now `transferred → 14-09`

## Decisions Made

- **Symlink escape declined (T-14-24, accept):** `filepath.EvalSymlinks` would close a symlink-inside-`protoconfRoot` escape but costs a stat syscall on every `Get`, on the agent's hot read path. Declined because planting such a symlink already requires write access to the config repository -- a strictly larger compromise than the unauthenticated remote read this plan closes. Marked in-source with a `ponytail:` comment on `resolveKeyPath`'s containment step.
- **No normalization check added to `MutateConfig`:** unlike `filekv`, `server` never had a "key differs from its own cleaned form" check, and the plan explicitly instructed not to add one -- `./x`, `a/../b`, and the empty path must keep resolving inside the base exactly as before. Only containment is new.
- **`resolveKeyPath` stays private to `filekv`:** not hoisted into a shared package for `MutateConfig`'s second call site. The two roots (`protoconfRoot` vs `protoconfRoot/mutable_config`) and failure shapes (`store.KVPair` errors vs `logError`-wrapped gRPC errors) differ enough that a shared helper would need its own parameterization; the plan scoped this as one local check at one call site.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `gsd_run check tdd-red-evidence` cannot classify Go test output**
- **Found during:** Task 1 (RED phase for filekv traversal tests)
- **Issue:** The tool's classifier (`.claude/gsd-core/bin/lib/tdd-red-evidence.cjs`) parses TAP-style `node --test` output (`# tests N`, `# pass N`, `# fail N`, `not ok N - <name>`). Go's `go test -v` emits `--- FAIL: TestName` / `--- PASS: TestName` instead, which the parser's regexes do not match -- every run would classify as `zero_tests_discovered` (`INVALID_RED`) regardless of whether the named test actually failed for the right reason.
- **Fix:** Relied on the plan's own literal `<verify>` blocks instead, which encode the identical discipline directly for this Go project: `go test ... -v 2>&1 | tee <file>; grep -q -- "--- FAIL: <TestName>" <file>`, required to pass (i.e. the FAIL line present) before the corresponding GREEN task could begin. Ran exactly as specified for both Task 1 and Task 3; the verbatim `--- FAIL:` lines are quoted above and in the Task 1/Task 3 commit messages.
- **Files modified:** None (verification-tooling gap, not a code issue)
- **Verification:** RED evidence captured and GREEN transition confirmed for both features, per the plan's own acceptance criteria
- **Committed in:** Documented here; no separate commit (informational deviation)

---

**Total deviations:** 1 auto-fixed (1 blocking, tooling gap worked around via the plan's own equivalent gate)
**Impact on plan:** None on the security fix itself -- both traversal defects were proven falsifiable and then closed exactly per the plan's `<verify>` and `<acceptance_criteria>`. The gap is in a supporting Node-oriented verification tool, not in this Go project's test discipline.

## Issues Encountered

None beyond the tooling deviation above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Phase 14's goal-level safety clause now holds on both sides of the trust boundary this plan touched: a caller-supplied key cannot read/watch a file outside `protoconfRoot` via `filekv`, and a caller-supplied `in.Path` cannot write a file outside `protoconfRoot/mutable_config` via `MutateConfig`.
- All plan-level `<verification>` commands re-run clean: `go test -race -count=1 -skip 'Test_cliCommand_Run' ./agent/...` (0 races), `go test -race -timeout 300s -skip 'Test_cliCommand_Run' ./...` (0 races, run twice), `go test -race -count=1 ./server/...` + `TestAuthFlow` green, `go vet ./agent/filekv/... ./server/...` silent, `git diff --exit-code go.mod go.sum` clean.
- `agent/filekv/filekv.go` contains exactly one `filepath.Join(s.protoconfRoot` (inside `resolveKeyPath`); `server/server.go` contains exactly 3 (srcPath, `MutateConfig`'s base, `collectExamples`' root).
- No open items remain for this gap; `deferred-items.md` deliberately untouched since nothing from this plan is deferred.
- Ready for `/gsd-verify-work` re-verification of Phase 14, or for the milestone to proceed.

---
*Phase: 14-non-compiler-consumer-correctness*
*Completed: 2026-09-08*

## Self-Check: PASSED

- FOUND: agent/filekv/filekv.go
- FOUND: agent/filekv/filekv_test.go
- FOUND: server/server.go
- FOUND: server/mutate_config_path_test.go
- FOUND commit: 6d35971 (test)
- FOUND commit: 211a643 (feat)
- FOUND commit: d841035 (test)
- FOUND commit: 042d031 (feat)
- FOUND commit: ee398bb (docs)
