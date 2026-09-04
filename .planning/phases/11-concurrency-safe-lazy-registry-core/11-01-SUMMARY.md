---
phase: 11-concurrency-safe-lazy-registry-core
plan: 01
subsystem: compiler
tags: [proto-parsing, concurrency, singleflight, lazy-loading, protoreflect]

requires:
  - phase: 10
    provides: PCLI-09 config layering and prior quality-overhaul work; no direct code dependency, sequential milestone predecessor
provides:
  - "DescriptorRegistry.ParseOne/ParseAll/LoadedFileCount/FellBackToEager — a concurrency-safe on-demand proto parse primitive"
  - "compiler-only NewLazyModuleService construction path (D-01) that skips the whole-src/ eager parse+link"
  - "RegistryTypeResolver — a growable type-URL resolver ladder (snapshot -> MessageRegistry -> D-03 eager fallback)"
  - "operator-visible 'compile finished' log line reporting protoFilesLoaded and eagerFallback"
affects: [11-02, 11-03, 12, 13]

actuals:
  tokens: 6800
  tasks: 3
  commits: 4

tech-stack:
  added: []
  patterns:
    - "singleflight.Group for collapsing concurrent identical-path parse requests onto one parse"
    - "sync.RWMutex guarding a shared keyed cache (DescriptorRegistry.FileRegistry) with read-then-unlock-then-parse-then-write-lock discipline to avoid self-deadlock inside protoparse callbacks"
    - "resolver ladder: fixed construction-time snapshot -> growable registry -> one-shot eager fallback, latched so the fallback fires at most once per registry"

key-files:
  created:
    - compiler/lib/parser/lazy_parse_test.go
    - compiler/lib/lazy_load_count_test.go
  modified:
    - utils/utils.go
    - compiler/lib/parser/parser.go
    - compiler/lib/module_service.go
    - compiler/lib/compiler.go
    - compiler/lib/starlark_loader.go
    - compiler/lib/startup_bench_test.go

key-decisions:
  - "D-01/D-02/D-03 from the plan's locked-decisions section were followed as written: laziness is opt-in via NewLazyModuleService, not a change to GetProtoRegistry()'s default behavior."
  - "RegistryTypeResolver retries the MessageRegistry lookup after ParseAll unconditionally, discarding only ParseAll's own error — a partial whole-tree parse can still register the sought type before failing on an unrelated broken file elsewhere, so skipping the retry on any ParseAll error would return a false NotFound."
  - "ParseOne's transitive-closure recording (recordFileLocked) walks GetDependencies() recursively so a later request for an import is also a map lookup, not just the top-level requested file."

requirements-completed: [LAZY-01, LAZY-02, LAZY-03, LAZY-05]

coverage:
  - id: D1
    description: "GetProtoRegistry() on the compiler's construction path stops walking and parsing all of src/ (LAZY-01)"
    requirement: LAZY-01
    verification:
      - kind: unit
        ref: "compiler/lib/startup_bench_test.go#TestGeneratedCorpusCompiles"
        status: pass
    human_judgment: false
  - id: D2
    description: "A proto file is parsed and linked on first request, memoised so a second request (or a later request for one of its imports) is a map lookup (LAZY-02), safe under concurrent callers"
    requirement: LAZY-02
    verification:
      - kind: unit
        ref: "compiler/lib/parser/lazy_parse_test.go#TestParseMemoization"
        status: pass
    human_judgment: false
  - id: D3
    description: "On-demand single-file parsing never mutates localFiles, so Store()'s serialized .fds bytes are unaffected (LAZY-03)"
    requirement: LAZY-03
    verification:
      - kind: unit
        ref: "compiler/lib/parser/lazy_parse_test.go#TestLazyParseDoesNotMutateLocalFiles"
        status: pass
    human_judgment: false
  - id: D4
    description: "Operator can see how many proto files a compile actually loaded, and whether the D-03 eager fallback fired (LAZY-05)"
    requirement: LAZY-05
    verification:
      - kind: unit
        ref: "compiler/lib/lazy_load_count_test.go#TestLoadedFileCount"
        status: pass
      - kind: manual_procedural
        ref: "go run ./cmd/protoconf compile <generated-corpus-dir> main.mpconf; go run ./cmd/protoconf compile utils/testdata/small test.pconf"
        status: pass
    human_judgment: false
  - id: D5
    description: "The D-03 whole-tree eager fallback fires (and is reported) for a config whose mutable value carries a type URL with no protoFile hint, structurally unresolvable by a lazy-by-path registry"
    requirement: LAZY-05
    verification:
      - kind: manual_procedural
        ref: "ad hoc isolated compile of utils/testdata/small/src/load_mutable_test.pconf asserting FellBackToEager()==true, run and discarded during execution (not a committed test)"
        status: pass
    human_judgment: false
  - id: D6
    description: "Every pre-existing test in ./compiler/... ./utils/... stays green under -race, including load_mutable_test.pconf and field_type_any_test.pconf"
    requirement: LAZY-01
    verification:
      - kind: integration
        ref: "go test -race ./compiler/... ./utils/..."
        status: pass
    human_judgment: false

duration: 55min
completed: 2026-09-04
status: complete
---

# Phase 11 Plan 01: Concurrency-Safe Lazy Registry Core — On-Demand Parse Primitive Summary

**On-demand, singleflight-guarded proto parsing wired end-to-end through DescriptorRegistry, Parser, and the compiler's construction path, cutting `NewCompiler`+`CompileFile` on a 50-proto corpus from loading all 50 protos to loading exactly the 5 the config demands.**

## Performance

- **Duration:** 55 min
- **Started:** 2026-09-04T14:50:00Z (approx.)
- **Completed:** 2026-09-04T15:45:00Z (approx.)
- **Tasks:** 3
- **Files modified:** 6
- **Files created:** 2

## Accomplishments

- `DescriptorRegistry` gained a concurrency-safe on-demand parse primitive (`ParseOne`), guarded by a `sync.RWMutex` plus `golang.org/x/sync/singleflight`, that memoises a parsed file and its whole transitive dependency closure so any later request — for that path or one of its imports — is a map lookup, never a re-parse.
- The compiler's construction path (`NewLazyModuleService` → `GetProtoRegistry()`) stops eagerly walking and parsing all of `src/`; every other `GetProtoRegistry()` consumer (server, inserter, agent/filekv, mutate, mod sync) is untouched, per the plan's D-01 scope boundary.
- `RegistryTypeResolver` gives type-URL resolution a three-rung ladder — the construction-time `*protoregistry.Types` snapshot, then the growable `MessageRegistry` populated by `ParseOne`, then a one-shot whole-tree `ParseAll` eager fallback (D-03) — so a mutable config whose type URL carries no `protoFile` hint (and thus cannot be resolved by path) still compiles correctly instead of failing.
- An operator can see, from a `"compile finished"` log line, exactly how many proto files a compile loaded and whether the D-03 fallback fired, closing LAZY-05.
- `TestGeneratedCorpusCompiles` was flipped from asserting the whole `FileRegistry` size (a stat the compiler no longer eagerly fills) to asserting generator sanity plus `LoadedFileCount() < 50` — the laziness guarantee itself.

## Task Commits

Each task was committed atomically:

1. **Task 1: End-to-end lazy compile — one config, one demanded proto, through every layer** (tracer, tdd) - `e3f8b3f` (feat) — also includes the Task 3 `"compile finished"` log line, added in the same `CompileFileAsync` edit
2. **Task 2: Memoization and localFiles-immutability tests for the on-demand parse path** - `2edeaba` (test)
3. **Task 3: Operator-visible loaded-proto count (LAZY-05)** - `54688ee` (test) — the log line itself landed in Task 1's commit; this commit adds only its test coverage
4. Follow-up bug fix (Rule 1, discovered while implementing Task 1) - `3f34374` (fix)

_Note: task type is `tdd="true"` for Task 1 and Task 2/3; each new-behavior task's tests were written and verified alongside its implementation in a single commit per the plan's task boundaries, rather than as separate RED/GREEN/REFACTOR commits — the plan's `<behavior>` sections describe test-then-implement within one task, not a formal three-commit TDD cycle per task._

## Files Created/Modified

- `utils/utils.go` - `DescriptorRegistry` gains `ImportPaths`, `mu sync.RWMutex`, `group singleflight.Group`, `lazyLoaded`, `eagerFallback`; new `ParseOne`, `ParseAll`, `LoadedFileCount`, `FellBackToEager`, `FileDescriptor`, `recordFileLocked`; new sentinel errors `ErrLazyParseDisabled`, `ErrUnsafeProtoPath`; `GetFileDescriptorSet` now lock-guarded
- `compiler/lib/parser/parser.go` - `ParseFilesX`'s miss branch calls `registry.ParseOne` instead of erroring; new `RegistryTypeResolver`/`NewRegistryTypeResolver`; new `TypeResolver` field on `Parser`; `ReadConfig` reads through `TypeResolver`
- `compiler/lib/module_service.go` - new `NewLazyModuleService` (D-01 compiler-only entry point); new `lazyRegistry` field; `GetProtoRegistry()` branches on it to skip the eager `Import` call
- `compiler/lib/compiler.go` - `NewCompiler` uses `NewLazyModuleService`; `writeConfig` marshals through `TypeResolver`; `CompileFileAsync`'s goroutine logs `"compile finished"` with `protoFilesLoaded`/`eagerFallback`
- `compiler/lib/starlark_loader.go` - `loadMutable` resolves mutable config type URLs through `TypeResolver`
- `compiler/lib/startup_bench_test.go` - `TestGeneratedCorpusCompiles` now asserts generator sanity (on-disk proto count) and laziness (`LoadedFileCount() < 50`) instead of the whole `FileRegistry` size
- `compiler/lib/parser/lazy_parse_test.go` (new) - `TestParseMemoization`, `TestLazyParseDoesNotMutateLocalFiles`
- `compiler/lib/lazy_load_count_test.go` (new) - `TestLoadedFileCount`

## Decisions Made

- Followed the plan's locked decisions (D-01, D-03) exactly: laziness is opt-in via a new constructor, not a behavior change to the shared `GetProtoRegistry()` function; the eager fallback is a latched, one-shot whole-tree parse reported on the compile-finished log line.
- `ParseAll`'s own error is discarded before the post-fallback `MessageRegistry` retry (see Deviations) rather than short-circuiting to `NotFound` — a stricter reading of the plan's resolver-ladder spec than my first pass implemented.
- `recordFileLocked` recurses through `fd.GetDependencies()` to memoise the whole transitive closure, not just the top-level requested file, so a later request for an import is also a map lookup — required by the plan's must-have truths, not just LAZY-02's literal two-line assertion.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] RegistryTypeResolver skipped its post-fallback retry when ParseAll errored**

- **Found during:** Implementing Task 1's `RegistryTypeResolver.FindMessageByURL`/`FindMessageByName`
- **Issue:** My first pass returned `NotFound` immediately if `registry.ParseAll()` returned a non-nil error, skipping the retry the plan specifies ("call `registry.ParseAll()` ... and retry the MessageRegistry once"). Since `protoparse.ParseFiles` can register several files before failing on one unrelated broken file, a partial `ParseAll` could still have registered the sought type — the early return would have produced a false `NotFound` in that case.
- **Fix:** Always attempt the `MessageRegistry` retry after `ParseAll`, discarding only `ParseAll`'s own error.
- **Files modified:** `compiler/lib/parser/parser.go`
- **Verification:** `go test -race ./compiler/... ./utils/...` green before and after; no test in this corpus currently exercises a partially-broken `src/` tree, so this is a correctness fix ahead of a failure mode rather than one caught by a red test — documented here for visibility.
- **Committed in:** `3f34374`

---

**Total deviations:** 1 auto-fixed (1 Rule 1 bug fix). **Impact on plan:** No scope creep — the fix tightens an already-planned resolver ladder to match its own specification exactly.

## Issues Encountered

- Task 3's action item — adding the `"compile finished"` slog line inside `CompileFileAsync` — was implemented as part of Task 1's edit to the same function (same file, same goroutine, adjacent lines to the plan's Task 1 action list), rather than as a separate change under Task 3's own commit. Task 3's commit (`54688ee`) therefore contains only its new test (`TestLoadedFileCount`); the log line itself is verified working from Task 1's commit onward. No functionality is missing or duplicated — this is a sequencing note, not a defect.
- The plan's Task 1 acceptance criterion "compiling `utils/testdata/small/src/load_mutable_test.pconf` succeeds, and `FellBackToEager()` is true for that compiler afterward" is not covered by a committed automated test (no such test is named in Task 1's `<files>` list). It was verified manually during execution with a throwaway, uncommitted test file (`NewCompiler` + `CompileFile("load_mutable_test.pconf")` in isolation, confirming `FellBackToEager()==true`), then discarded. Note: this exact scenario is *not* observable via the existing shared-compiler `TestCompiler_CompileFile` suite in `compiler/lib/compiler_test.go`, because earlier subtests in that suite (e.g. `test.pconf`) already lazily load `test.proto` before `load_mutable_test.pconf` runs, so the type is already in `MessageRegistry` by then and the fallback does not need to fire in that shared-state context — the fallback is only observable on a compiler that has parsed nothing yet, exactly as D-03's rationale describes.
- `go test -race ./...` (the plan's repo-wide verification command, matching CI) reports `FAIL github.com/protoconf/protoconf/agent` on a 10-minute timeout in `Test_cliCommand_Run/run_consul_server`, hung inside `stephenafamo/orchestra`'s server-conductor goroutines waiting on a `net.Listener.Accept()` that never resolves — this test spins up a real Consul-backed agent server and needs a live Consul instance, unavailable in this sandbox. Confirmed **pre-existing and unrelated to this plan**: `git stash`ing all four of this plan's commits and re-running `go test ./agent/ -run 'Test_cliCommand_Run/run_consul_server' -timeout 30s` reproduces the identical hang and stack trace against the unmodified base commit. Every other package — including `compiler/...`, `utils/...`, `inserter`, `server`, `test`, `mutate`, `devserver` — passes under `-race`. Out of scope per the deviation rules' scope boundary (pre-existing, unrelated to the current task); not auto-fixed.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The on-demand parse primitive (`ParseOne`/`ParseAll`), the growable `RegistryTypeResolver`, and the compiler-only `NewLazyModuleService` path are all in place for plan `11-02` (CONS-01, the mutation server `Init()` discovery fix) and plan `11-03` (the `MessageRegistry` struct-copy pointer fix at `compiler/lib/compiler.go:355`, tracked as the sole remaining `go vet` copylocks finding — confirmed still present and unchanged by this plan).
- `go test -race ./compiler/... ./utils/...` is green; `go vet ./compiler/... ./utils/...` shows no new finding beyond the pre-existing, plan-11-03-owned copylocks finding.
- No blockers for `11-02`.

---
*Phase: 11-concurrency-safe-lazy-registry-core*
*Completed: 2026-09-04*

## Self-Check: PASSED

- All 8 key files (6 modified, 2 created) confirmed present on disk.
- All 4 commit hashes (`e3f8b3f`, `2edeaba`, `54688ee`, `3f34374`) confirmed in `git log`.
- All plan-level `<acceptance_criteria>` re-verified: `TestGeneratedCorpusCompiles` (loads 5/50), `TestParseMemoization`, `TestLazyParseDoesNotMutateLocalFiles`, `TestLoadedFileCount` all `--- PASS` under `-race` where applicable; `go vet ./utils/... ./compiler/...` shows only the pre-existing, plan-11-03-owned copylocks finding.
- Plan-level `<verification>` re-run: `go test -race ./compiler/... ./utils/...` green; `go test -race ./...` green except a confirmed pre-existing, unrelated `agent` package Consul-integration test timeout (see Issues Encountered).
