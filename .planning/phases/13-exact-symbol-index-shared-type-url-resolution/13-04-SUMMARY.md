---
phase: 13-exact-symbol-index-shared-type-url-resolution
plan: 04
subsystem: compiler
tags: [protobuf, protoreflect, type-url-resolution, mutable-config, jhump]

requires:
  - phase: 13-exact-symbol-index-shared-type-url-resolution
    plan: 02
    provides: "The exact symbol index (Tier 3) and the scan tier (Tier 2) this plan's fixture resolves test.v1.MessageWithSubMessage and its nested Any (test.v1.MessageWithSubMessage.SubMessage) through"
provides:
  - "loadMutable resolves the mutable value's type exactly once, through l.parser.TypeResolver.FindMessageByURL, deriving the *desc.MessageDescriptor dynamic.NewMessage needs via desc.WrapMessage(mt.Descriptor()) instead of a second, direct MessageRegistry lookup"
  - "A mutable config whose value field is absent fails loudly (error naming the file) instead of panicking on a nil *anypb.Any"
  - "compiler/lib/load_mutable_nested_any_test.go: the CONS-05 regression, isolating the bypass via a cold moduleService, plus a full-pipeline nested-Any round-trip check"
affects: [14-consumer-migration-and-repo-wide-grep-clean]

actuals:
  tokens: 2184
  tasks: 2
  commits: 5
  plan_head_before: 8c9f00e9ee6e7afd5adce7c35c7eec8e6c7afc08

tech-stack:
  added: []
  patterns:
    - "When a value is already resolved through the shared TypeResolver chain, derive any jhump-native descriptor the caller needs from that SAME result (desc.WrapMessage(mt.Descriptor())) rather than re-resolving the identical type URL through a different registry accessor"
    - "Use GetValue()/GetTypeUrl() (nil-safe generated getters) instead of direct field access when a proto field may legitimately be absent from materialized input"

key-files:
  created:
    - compiler/lib/load_mutable_nested_any_test.go
    - utils/testdata/small/src/load_mutable_nested_any_test.pconf
    - utils/testdata/small/mutable_config/nested_any_mutation.materialized_JSON
  modified:
    - compiler/lib/starlark_loader.go
    - utils/testdata/small/src/test.proto

key-decisions:
  - "desc.WrapMessage(mt.Descriptor()) worked exactly as the planner's flagged assumption hoped: no two-call fallback was needed. Verified by temporarily reverting starlark_loader.go (git stash) and confirming the pre-fix failure reproduces, then restoring and confirming both regression tests plus the full plan-scoped and whole-repo race suites are green."
  - "The literal plan fixture (a top-level TestMessage value carrying a nested Any) does NOT by itself force loadMutable's bypassed lookup to fail: parser.ReadConfig (line 172, called before either lookup) already resolves the SAME type URL through the identical shared MessageRegistry as a side effect of decoding the materialized JSON's own \"@type\" field, and jhump's ParseOne links the WHOLE file in one shot, so both the outer TestMessage and its nested SubMessage are already registered in MessageRegistry by the time loadMutable's own lookups run -- regardless of whether the second lookup goes through the shared chain or bypasses it. TestLoadMutableResolvesNestedAny therefore forces the divergence deterministically instead: it swaps loadMutable's moduleService for a cold, empty one (pointed at an empty temp directory that will never parse anything) while keeping its parser bound to the compiler's own correctly-shared TypeResolver. This is a genuine, non-vacuous RED (see below) and directly proves the must_haves.truths property (\"no consumer may resolve a type URL by reaching into a registry directly\") independent of any incidental pre-warming a specific fixture might produce. A second assertion in the same test still exercises the literal fixture end to end through the compiler's own consistent parser+moduleService, proving the nested Any round-trips correctly (the must_haves.truths property this plan also required)."
  - "The RED failure this fixture actually produces is a nil-pointer PANIC, not a clean protoregistry.NotFound error: MessageRegistry.FindMessageTypeByUrl on the cold registry returns (nil, nil) -- a miss represented as a nil descriptor with no error -- and the bypassed call's error check (`if err != nil`) does not also check `md != nil` the way resolveTiers' own re-check does at every tier. The nil *desc.MessageDescriptor then flows into dynamic.NewMessage(d).Unmarshal(b), which panics inside jhump/protoreflect's GetMessageOptions. This is arguably a more serious failure mode than a clean error return, and it is exactly the shape a shared chokepoint's own careful nil-checking (visible in resolveTiers' `mErr == nil && md != nil` checks) is designed to prevent by construction once bypasses are eliminated."

patterns-established:
  - "compiler/lib/load_mutable_nested_any_test.go's cold-moduleService construction (a *starlarkLoader built with a real, shared parser but an isolated, never-parsed moduleService) as the template for testing any future CONS-0x-style divergence between a consumer's own registry access and the shared TypeResolver chain."

requirements-completed: [CONS-05]

coverage:
  - id: D1
    description: "loadMutable resolves the mutable value's type -- and the depth-2 nested Any inside it -- entirely through l.parser.TypeResolver, proven by forcing the bypassed lookup's target registry to be cold/empty and confirming resolution still succeeds"
    requirement: "CONS-05"
    verification:
      - kind: unit
        ref: "compiler/lib/load_mutable_nested_any_test.go#TestLoadMutableResolvesNestedAny"
        status: pass
    human_judgment: false
  - id: D2
    description: "A mutable config whose value field is absent fails loudly with an error naming the file, never a panic"
    requirement: "CONS-05"
    verification:
      - kind: unit
        ref: "compiler/lib/load_mutable_nested_any_test.go#TestLoadMutableEmptyValueFailsLoudly"
        status: pass
    human_judgment: false
  - id: D3
    description: "No direct MessageRegistry/registry resolution survives in compiler/lib/starlark_loader.go -- structurally confirmed by grep, and existing mutable-config error-path tests (load_mutable_test.pconf, mutable_bad_json_err.pconf, mutable_bad_proto_filename_err.pconf, mutable_not_exists_err.pconf) keep passing with unchanged error shapes"
    requirement: "CONS-05"
    verification:
      - kind: unit
        ref: "compiler/lib/compiler_test.go#TestCompiler_CompileFile"
        status: pass
      - kind: other
        ref: "grep -n 'FindMessageTypeByUrl' compiler/lib/starlark_loader.go | grep -v '//' (no output)"
        status: pass
    human_judgment: false

duration: 45min
completed: 2026-09-08
status: complete
---

# Phase 13 Plan 4: Close CONS-05 -- Mutable-Config Load Path Now Routes Through the Shared TypeResolver Summary

**`loadMutable` resolves the mutable value's type exactly once, through `l.parser.TypeResolver`, deriving the jhump descriptor it needs via `desc.WrapMessage(mt.Descriptor())` instead of a second, direct `MessageRegistry` lookup that could return a nil descriptor with no error and panic -- and a mutable config with no `value` now fails loudly instead of crashing on a nil `*anypb.Any`.**

## Performance

- **Duration:** ~45 min
- **Started:** 2026-09-08 (continuing directly from 13-03)
- **Completed:** 2026-09-08
- **Tasks:** 2
- **Files modified:** 4 (1 modified, 3 created)

## Accomplishments
- Closed CONS-05: `compiler/lib/starlark_loader.go`'s `loadMutable` no longer reaches into `l.moduleService.GetProtoRegistry().MessageRegistry.FindMessageTypeByUrl` directly. It derives the `*desc.MessageDescriptor` `dynamic.NewMessage` needs from `mt` -- the `protoreflect.MessageType` already resolved through `l.parser.TypeResolver.FindMessageByURL`, the one shared chokepoint -- via `desc.WrapMessage(mt.Descriptor())`. Deleted the now-unused direct lookup and the dead, already-checked `if err != nil` immediately below it.
- Fixed a genuine crash the bypassed call's missing nil-check made possible: `MessageRegistry.FindMessageTypeByUrl` can return `(nil, nil)` on a miss (confirmed by reading `resolveTiers`' own `mErr == nil && md != nil` re-checks), and `loadMutable`'s old code checked only `err != nil`. A nil descriptor flowing into `dynamic.NewMessage(d).Unmarshal(b)` panics deep inside jhump/protoreflect. `desc.WrapMessage(mt.Descriptor())` removes the possibility entirely, since `mt` is only ever a genuinely-resolved, non-nil result.
- Closed T-13-17: a mutable config whose `value` field is absent (a `*anypb.Any` left nil after `protojson.Unmarshal`) previously panicked on `protoconfValue.Value.TypeUrl`'s direct field access. Switched to the nil-safe `protoconfValue.GetValue().GetTypeUrl()`, so an absent value now flows into `l.parser.TypeResolver.FindMessageByURL("")`, hits 13-03's own empty-type-URL guard, and returns a clean, file-naming error via `ErrReadMutable`.
- `compiler/lib/load_mutable_nested_any_test.go`'s `TestLoadMutableResolvesNestedAny` forces the CONS-05 divergence deterministically (see Decisions Made for why the literal fixture-as-specified does not, by itself, produce a non-vacuous failure) via a cold, never-parsed `moduleService`, then separately proves the fixture compiles correctly end to end through the compiler's own consistent parser+moduleService -- both the outer `test.v1.MessageWithSubMessage` value and its depth-2 nested `Any` (`test.v1.MessageWithSubMessage.SubMessage`, a NESTED symbol exercising the scan tier rather than a top-level-only lookup) resolve and round-trip correctly. (The outer type is `MessageWithSubMessage`, not the plan's originally-specified `TestMessage` -- see deviation 3.)
- Confirmed RED empirically, not just by construction: reverted `starlark_loader.go` via `git stash`, re-ran `TestLoadMutableResolvesNestedAny`, and observed the exact nil-descriptor panic described above (`--- FAIL: TestLoadMutableResolvesNestedAny` followed by a `dynamic.Message.Unmarshal` -> `GetMessageOptions` nil-pointer crash). Restored the fix and re-verified GREEN.

## Task Commits

Each task was committed atomically, following the RED-GREEN cycle for its `tdd="true"` attribute (no REFACTOR commit -- the GREEN implementation is already the smaller, final shape the plan called for):

1. **Task 1: Write the failing CONS-05 regression** -- `a6015f6` (test)
2. **Task 2: Route loadMutable's second resolution through the shared TypeResolver** -- `a0b2ec2` (feat)
3. **Post-hoc fix (deviation 3 below): retarget the fixture away from an RPC-input type** -- `698956a` (fix)

**Plan metadata:** `35bf4dd` (docs: complete plan, written before deviation 3 was discovered; this SUMMARY was updated in place afterward -- no second metadata commit was made solely to bump a word count)

**Commit-count note:** `git rev-list --count 8c9f00e..HEAD` measures **5**, not 4 -- `d63ce39` (`docs(13): add code review report`) landed on this branch between `35bf4dd` and `698956a` from a concurrent, phase-level review process outside this plan's own task sequence (its content is a `13-REVIEW.md` covering all of Phase 13, not just this plan). It is not this plan's work and is reported here rather than silently absorbed into the count, per the "measured, never narrated" contract -- the honest reading is 4 commits belonging to this plan's own tasks, plus 1 unrelated interloper the ledger's window happens to include.

## Files Created/Modified
- `compiler/lib/load_mutable_nested_any_test.go` - `TestLoadMutableResolvesNestedAny` (cold-moduleService isolation + full-pipeline round-trip check), `TestLoadMutableEmptyValueFailsLoudly` (absent-value, no-panic guard)
- `utils/testdata/small/src/load_mutable_nested_any_test.pconf` - two-line fixture, `load("mutable:nested_any_mutation", "value")` with no `load("//test.proto"`
- `utils/testdata/small/mutable_config/nested_any_mutation.materialized_JSON` - `test.v1.MessageWithSubMessage` value (a top-level, non-RPC-input message; see deviation 3) whose `sub.value` and `anyField` (a nested `test.v1.MessageWithSubMessage.SubMessage`) both resolve, no `protoFile` field
- `utils/testdata/small/src/test.proto` - added `google.protobuf.Any any_field = 2;` to `MessageWithSubMessage` (deviation 3)
- `compiler/lib/starlark_loader.go` - `loadMutable`: single resolution via `l.parser.TypeResolver.FindMessageByURL`, `desc.WrapMessage(mt.Descriptor())` replacing the direct `MessageRegistry` lookup, nil-safe `GetValue()/GetTypeUrl()` guard against an absent value

## Decisions Made
See `key-decisions` in frontmatter. In prose:
- **`desc.WrapMessage(mt.Descriptor())` worked as assumed** -- the planner's flagged uncertainty about this round trip through jhump/protoreflect did not materialize; no two-call fallback was needed.
- **The literal fixture (top-level value, nested inner Any) does not by itself force a RED failure.** `parser.ReadConfig` resolves the exact same outer type URL through the identical shared `MessageRegistry` before either of `loadMutable`'s own lookups run, and `ParseOne` links the whole `test.proto` file in one shot -- so the bypassed lookup would have trivially succeeded too, using the exact same already-populated registry. `TestLoadMutableResolvesNestedAny` isolates the true divergence with a cold `moduleService`, which is what a future accidental reintroduction of a similar bypass would need to be tested against.
- **The real bug is a missing nil-check, not just "wrong registry."** `MessageRegistry.FindMessageTypeByUrl`'s `(nil, nil)`-on-miss contract (already visible in `resolveTiers`' own defensive `mErr == nil && md != nil` checks) means any direct caller that checks only `err != nil` is one cold-registry lookup away from a nil-pointer panic. Routing through `l.parser.TypeResolver` removes this bypass surface entirely rather than patching the missing check in two places.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug/plan-assumption gap] Redesigned Task 1's RED test to force a genuine (non-vacuous) failure**
- **Found during:** Task 1
- **Issue:** The plan's literal `<action>` text specified constructing the fixture with a top-level outer value type (`test.v1.TestMessage`) carrying a nested-symbol inner `Any`, and asserting the compile fails with a `protoregistry.NotFound`-wrapped error before the fix. Running this exact test against the pre-fix code showed it PASSES: `parser.ReadConfig` (called at the top of `loadMutable`, before either of its own lookups) already resolves `test.v1.TestMessage` through the identical shared `MessageRegistry` as a side effect of decoding the materialized JSON's `"@type"` field, and `ParseOne` links the whole `test.proto` file in one pass -- so by the time `loadMutable`'s bypassed lookup runs, the registry it reads from is already populated via the exact route the shared chain would have used. The literal fixture is therefore a vacuous pass for THIS specific bug, exactly the failure mode the plan's own `<action>` text warned about ("A test that passes here means either the fixture accidentally loads test.proto by another route or the assertion is not reaching the bypassed call; fix the fixture, not the assertion").
- **Fix:** Kept the exact same fixture files (the `.pconf` and `.materialized_JSON`, satisfying every grep-based acceptance criterion in Task 1 verbatim), but changed `TestLoadMutableResolvesNestedAny`'s mechanism: it now constructs a `*starlarkLoader` directly, binding its `parser` to the compiler's own correctly-shared `TypeResolver` but its `moduleService` to a cold, empty `ModuleService` (pointed at an empty temp directory that will never parse anything). This deterministically reproduces the divergence CONS-05 exists to close -- and it uncovered a more serious defect than a clean "not found": a nil-pointer panic (see Accomplishments). The same test then separately compiles the fixture through the compiler's own consistent parser+moduleService (the literal, full-pipeline scenario the plan described) to prove the nested Any round-trips correctly end to end -- this part passes both before and after the fix, since it is a valid regression test even though it cannot itself distinguish the CONS-05 bug.
- **Files modified:** `compiler/lib/load_mutable_nested_any_test.go`
- **Verification:** Reverted `starlark_loader.go` via `git stash`, ran `go test ./compiler/lib/ -run TestLoadMutableResolvesNestedAny -v`, confirmed `--- FAIL: TestLoadMutableResolvesNestedAny` followed by the nil-descriptor panic inside `dynamic.Message.Unmarshal`. Restored the fix (`git stash pop`) and confirmed `go test -race ./compiler/lib/ -run 'TestLoadMutableResolvesNestedAny|TestLoadMutableEmptyValueFailsLoudly' -v` passes both tests.
- **Committed in:** `a6015f6` (Task 1 RED commit)

**2. [Rule 1 - Bug] Added a nil-safe guard against a mutable config's absent "value" field**
- **Found during:** Task 1 (writing `TestLoadMutableEmptyValueFailsLoudly`)
- **Issue:** `loadMutable`'s first lookup used `protoconfValue.Value.TypeUrl` -- direct field access on `protoconfValue.Value`, a `*anypb.Any` that stays `nil` when the materialized JSON has no `"value"` field. This panics with a nil-pointer dereference, violating must_haves.truths' explicit requirement that an absent value "produces an error naming the file, not a panic."
- **Fix:** Switched to `protoconfValue.GetValue().GetTypeUrl()` (nil-safe generated getters), and wrapped the resulting error with `ErrReadMutable` and the file name, matching the surrounding error-wrapping convention already used at every other `loadMutable` return path.
- **Files modified:** `compiler/lib/starlark_loader.go`
- **Verification:** `go test -race ./compiler/lib/ -run TestLoadMutableEmptyValueFailsLoudly -v` -- PASS, no panic, error message contains the file name.
- **Committed in:** `a0b2ec2` (Task 2 GREEN commit)

---

**3. [Rule 1 - Bug caused by this plan's own fixture] Retargeted the fixture's outer type to avoid crashing an unrelated, already-passing test**
- **Found during:** post-Task-2 full verification (`go test -race ./...`)
- **Issue:** With the fixture as originally specified (outer type `test.v1.TestMessage`, an RPC input for `test.v1.TestService.PutTestMessage`), `TestProtoconfMutationServer_GenReflectionUI` in `server/server_test.go` -- a pre-existing, unrelated, previously-passing test -- started failing: `server.GenReflectionUI` walks every file under the shared `testdata.SmallTestDir()`'s `mutable_config/`, and for any value type with a registered `s.exampleMaker` entry (i.e. any RPC input type), it builds a `standalone.Example` and eventually calls `standalone.ExampleRequest.MarshalJSON()` (vendored, `github.com/fullstorydev/grpcui@v1.4.1`). That method hardcodes a resolver-less `protojson.Marshal(data)` call with no injection point for a custom `Resolver` -- so it can never resolve a locally-parsed, non-globally-registered symbol like `test.v1.MessageWithSubMessage.SubMessage` sitting inside a populated `google.protobuf.Any` field, and the whole `GenReflectionUI` call fails. This is a genuine regression this plan's own fixture caused (the shared `utils/testdata/small` tree is `//go:embed`-shared across every package's tests, including `server`'s), not a pre-existing/unrelated finding -- squarely in scope per the deviation rules' scope boundary ("directly caused by the current task's changes").
- **Fix:** D-03 explicitly keeps `server/server.go`'s own resolution call sites (`LocalResolver`, CONS-04) out of Phase 13's scope, and the vendored library offers no fix surface regardless. Instead, retargeted the fixture: added a `google.protobuf.Any any_field = 2;` field to `test.v1.MessageWithSubMessage` (a top-level message that is NOT an RPC input -- `test.proto`'s service only declares `PutTestMessage`/`PutValidateMe` -- so `GenReflectionUI`'s `exampleMaker` lookup misses it entirely and the vendored marshal call is never reached), and changed `nested_any_mutation.materialized_JSON`'s outer type from `test.v1.TestMessage` to `test.v1.MessageWithSubMessage`. The nested Any this plan's `must_haves` require (`test.v1.MessageWithSubMessage.SubMessage`) is unchanged, so every Task 1 acceptance criterion (fixture existence, the nested-symbol grep, the no-`load("//test.proto"` grep) still passes verbatim.
- **Files modified:** `utils/testdata/small/src/test.proto`, `utils/testdata/small/mutable_config/nested_any_mutation.materialized_JSON`, `compiler/lib/load_mutable_nested_any_test.go` (JSON assertion shape updated from `stringValue` to `sub.value` to match the new outer type's fields)
- **Verification:** `go test -race ./server/... ./compiler/... ./utils/... ./inserter/... ./mutate/...` -- all green, including `TestProtoconfMutationServer_GenReflectionUI` passing again. `go test -race ./compiler/lib/ -run 'TestLoadMutableResolvesNestedAny|TestLoadMutableEmptyValueFailsLoudly' -v` still both PASS.
- **Committed in:** `698956a`

---

**Total deviations:** 3 auto-fixed (1 plan-assumption gap / test-mechanism redesign, 1 genuine pre-existing bug fix, 1 cross-package regression this plan's own fixture caused), all Rule 1 class
**Impact on plan:** No scope change and no weakening of the plan's requirements -- all three deviations make the plan's own `must_haves.truths` and `<verification>` MORE rigorously proven and MORE correct than the literal action text would have, not less. The committed fixture files exist at the exact paths the plan specified and every one of Task 1's grep-based acceptance criteria (fixture contents, nested-symbol presence, absence of `load("//test.proto"`) passes verbatim -- only the fixture's chosen outer *type* changed, from one that happened to collide with an unrelated third-party library limitation to one that doesn't.

## Issues Encountered
- `go test -race ./compiler/... ./utils/...` (this plan's own scoped verification) is fully green: `compiler` 19.2s, `compiler/lib` 150.8s, `compiler/lib/parser` 27.1s, `utils` 8.1s, all `ok`. `go build ./...` and `go vet ./compiler/...` are clean.
- A first full `go test -race ./...` run surfaced a real, this-plan-caused regression in `github.com/protoconf/protoconf/server` (`TestProtoconfMutationServer_GenReflectionUI`) -- see deviation 3. After the fix, `go test -race ./server/... ./compiler/... ./utils/... ./inserter/... ./mutate/...` is fully green, including that test passing again.
- A second full `go test -race ./...` run (after deviation 3's fix) was still running in the background at the time this SUMMARY was finalized; 13-01/13-02/13-03 all independently recorded the same pre-existing, unrelated `github.com/protoconf/protoconf/agent` `Conductor.playWithLogger` goroutine hang (~9-10 minutes, sometimes hitting the 10-minute test timeout) under `-race` for the whole-repo run, logged to `.planning/phases/13-exact-symbol-index-shared-type-url-resolution/deferred-items.md`. The FIRST full run (pre-deviation-3-fix) reached this same `agent` package and then continued past it, confirming the hang is not universal/deterministic every run but is a known, pre-existing, unrelated flake -- nothing in this plan's files (`compiler/lib/starlark_loader.go`, its own test file, and the testdata/`test.proto` changes) is reachable from `agent`'s orchestra-based process-lifecycle tests. This plan's own scoped race run (`./compiler/... ./utils/...`) plus the broader `./server/... ./inserter/... ./mutate/...` run above are the authoritative signal per that established precedent; both are green.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- CONS-05 is closed: `loadMutable` is the last in-compiler consumer this milestone's shared-path work targeted (D-03), and it now routes through `l.parser.TypeResolver` exclusively.
- Phase 14's known scope is unchanged and re-confirmed: `inserter/inserter.go:369`, `server/server.go:607`, `mutate/mutate.go:76` (CONS-02/03/04) and `compiler/lib/config.go:66/68`'s silent-skip validation call site all still bypass the shared `RegistryTypeResolver.resolveTiers` chain -- none were touched by this plan (D-03 compiler-only fence).
- ROADMAP.md success criterion 1's grep clause still completes in Phase 14, not this phase, exactly as 13-03 already recorded.
- Phase 13 is now complete: all four plans (scan tier, symbol index + Tier 3 wiring, delete-eager-fallback, mutable-config shared path) are summarized.

## Self-Check: PASSED

- All 5 key files found on disk (`compiler/lib/load_mutable_nested_any_test.go`, `utils/testdata/small/src/load_mutable_nested_any_test.pconf`, `utils/testdata/small/mutable_config/nested_any_mutation.materialized_JSON`, `utils/testdata/small/src/test.proto`, `compiler/lib/starlark_loader.go`)
- All 4 of this plan's own commits found in git log (`a6015f6`, `a0b2ec2`, `35bf4dd`, `698956a`)
- `commits: 5` matches `git rev-list --count 8c9f00e..HEAD` measured directly (no narrated count) -- includes 1 unrelated interloper commit (`d63ce39`, a concurrent phase-level code-review report) this plan did not make; see the Task Commits note.
- Plan-level `<verification>` commands re-run: `go build ./...` clean, `go vet ./compiler/...` clean (whole-repo `go vet ./...` shows only the same 3 pre-existing findings 13-01/02/03 already logged), `go test -race ./compiler/... ./utils/...` green, `go test -race ./server/... ./compiler/... ./utils/... ./inserter/... ./mutate/...` green (including the deviation-3 regression test passing again); a second full `go test -race ./...` was still completing at finalization time (see Issues Encountered) -- the known, pre-existing `agent`-package timeout is not attributable to this plan
- Every task's `<acceptance_criteria>` re-verified passing, including all grep-based structural checks (`FindMessageTypeByUrl` absent, `desc.WrapMessage(mt.Descriptor())` count 1, `l.parser.TypeResolver.FindMessageByURL` count 1, `load("//test.proto"` count 0 in the fixture, `test.v1.MessageWithSubMessage.SubMessage` present in the materialized_JSON fixture)

---
*Phase: 13-exact-symbol-index-shared-type-url-resolution*
*Completed: 2026-09-08*
