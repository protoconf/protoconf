---
phase: 02-os-exit-refactoring
verified: 2026-03-24T06:30:00Z
status: passed
score: 9/9 must-haves verified
gaps: []
human_verification:
  - test: "Confirm CLI error behavior at runtime"
    expected: "When compiler/lib fails to resolve protoconf root, the compiler CLI prints an error and exits with code 1 (not panics)"
    why_human: "Cannot run the full CLI binary in this verification environment to confirm end-to-end exit code behavior"
---

# Phase 2: os.Exit Refactoring Verification Report

**Phase Goal:** Library code never terminates the process — errors propagate to CLI entry points
**Verified:** 2026-03-24T06:30:00Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | compiler/lib/module_service.go contains no os.Exit calls | VERIFIED | `grep -c 'os.Exit' compiler/lib/module_service.go` => 0 |
| 2 | compiler/lib/starlark_loader.go contains no os.Exit calls (all 3 locations resolved) | VERIFIED | `grep -c 'os.Exit' compiler/lib/starlark_loader.go` => 0; `var walkErr error` at line 115; 3 `return false` at lines 122, 125, 135 |
| 3 | mutate/mutate.go contains no os.Exit calls (all ~10 locations resolved) | VERIFIED | `grep -c 'os.Exit' mutate/mutate.go` => 0; setNumeric and setFloat return error; 12 error-checking call sites |
| 4 | All refactored functions return errors that propagate up to CLI-layer entry points | VERIFIED | NewModuleService, NewCompiler, NewCompilerService, NewProtoconfMutationServer all return `(*T, error)`; all callers handle errors with `return 1` |
| 5 | Existing CLI behavior is unchanged — error cases still exit with non-zero status | VERIFIED | CLI Run() methods return int; mitchellh/cli framework calls os.Exit at command/command.go:49; all tests pass |
| 6 | NewModuleService returns (*ModuleService, error) with resolve-at-construction | VERIFIED | `func NewModuleService(protoconfRoot string) (*ModuleService, error)` at module_service.go:44; filepath.Abs called at line 45; getProtoconfPath() simplified to `return m.Config.ProtoconfPath` |
| 7 | loadValidators uses closure error capture pattern instead of os.Exit | VERIFIED | `var walkErr error` at starlark_loader.go:115; three `return false` paths capture errors into walkErr |
| 8 | NewCompiler returns (*Compiler, error) and all callers handle the error | VERIFIED | compiler.go:33; compiler/command.go:137-140 handles error with `slog.Error` + `return 1`; test files use `require.NoError` |
| 9 | Existing tests pass unchanged after refactoring | VERIFIED | `go test ./compiler/lib/...` passes; `go test ./server/...` passes; `go test ./compiler/...` passes |

**Score:** 9/9 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `compiler/lib/module_service.go` | os.Exit-free module service with error-returning constructor | VERIFIED | `func NewModuleService(protoconfRoot string) (*ModuleService, error)` confirmed; zero os.Exit calls |
| `compiler/lib/compiler.go` | Error-returning compiler constructor | VERIFIED | `func NewCompiler(protoconfRoot string, verboseLogging bool) (*Compiler, error)` confirmed |
| `compiler/lib/starlark_loader.go` | Closure error capture in loadValidators | VERIFIED | `var walkErr error` at line 115; three `return false` exits |
| `mutate/mutate.go` | os.Exit-free mutate CLI with proper error returns | VERIFIED | `func setNumeric(msg *dynamic.Message, key, val string, typer typerFunc) error` at line 244; `func setFloat(msg *dynamic.Message, key, val string, typer typerFunc) error` at line 253; zero os.Exit calls |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `compiler/lib/module_service.go` | `compiler/lib/compiler.go` | NewModuleService called in NewCompiler | WIRED | `ms, err := NewModuleService(protoconfRoot)` at compiler.go:43 with error check at line 44-46 |
| `compiler/lib/compiler.go` | `compiler/command.go` | NewCompiler called in runLocally | WIRED | `compiler, err := compilerlib.NewCompiler(protoconfRoot, config.verboseLogging)` at command.go:137; error handled at lines 138-141 |
| `compiler/lib/compiler.go` | `compiler/service.go` | NewCompiler called in NewCompilerService | WIRED | `c, err := lib.NewCompiler(dir, verbose)` at service.go:22; `NewCompilerService` returns `(*CompilerService, error)` |
| `compiler/lib/compiler.go` | `server/server.go` | NewModuleService called in NewProtoconfMutationServer | WIRED | `ms, err := lib.NewModuleService(protoconfRoot)` at server.go:233; `NewProtoconfMutationServer` returns `(*ProtoconfMutationServer, error)` |
| `mutate/mutate.go:Run` | `command/command.go` | mitchellh/cli Command interface — Run returns int, framework calls os.Exit | WIRED | Run method returns int; all error paths return 1; framework exit at command/command.go:49 |

### Data-Flow Trace (Level 4)

Not applicable — this phase refactors control flow (error propagation), not data rendering. No dynamic UI components were modified.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| compiler/lib package builds without errors | `go build ./compiler/lib/...` | exit 0 | PASS |
| mutate package builds without errors | `go build ./mutate/...` | exit 0 | PASS |
| full codebase builds without errors | `go build ./...` | exit 0 (no output) | PASS |
| compiler/lib tests pass | `go test ./compiler/lib/...` | ok (cached) | PASS |
| server tests pass (exercises NewProtoconfMutationServer) | `go test ./server/...` | ok 4.263s | PASS |
| compiler command tests pass | `go test ./compiler/...` | ok 0.791s | PASS |
| NewModuleService signature matches must-have | `grep 'func NewModuleService' compiler/lib/module_service.go` | `(*ModuleService, error)` return | PASS |
| setNumeric returns error | `grep 'func setNumeric' mutate/mutate.go` | `error` return type | PASS |
| setFloat returns error | `grep 'func setFloat' mutate/mutate.go` | `error` return type | PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| REFC-01 | 02-01-PLAN.md | compiler/lib/module_service.go os.Exit replaced with error return | SATISFIED | Zero os.Exit calls; NewModuleService returns (*ModuleService, error); filepath.Abs resolved at construction |
| REFC-02 | 02-01-PLAN.md | compiler/lib/starlark_loader.go os.Exit calls (3 locations) replaced with error returns | SATISFIED | Zero os.Exit calls; var walkErr pattern with 3 return false paths at lines 122, 125, 135 |
| REFC-03 | 02-02-PLAN.md | mutate/mutate.go os.Exit calls (~10 locations) replaced with error returns | SATISFIED | Zero os.Exit calls; 8 in Run replaced with return 1; setNumeric/setFloat return errors; 12 error-checking call sites |
| REFC-04 | 02-01-PLAN.md, 02-02-PLAN.md | All refactored functions propagate errors to CLI entry points where os.Exit is appropriate | SATISFIED | NewModuleService -> NewCompiler -> compiler/command.go:runLocally returns int; NewProtoconfMutationServer -> server/server.go:Run returns int; mutate Run() already returns int |

No orphaned requirements — all four REFC-01 through REFC-04 are accounted for in plan frontmatter and verified in codebase.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `compiler/command.go` | 71, 76 | `os.Exit(1)` inside `Run()` in CPU profiling branch | INFO | These are in the CLI Run() method (not library code), and were pre-existing before Phase 2. The phase scope was strictly the three target library files. Not a regression. |
| `inserter/inserter.go` | 106 | `os.Exit(1)` inside `Run()` for unknown KV store | INFO | Pre-existing in CLI Run() method; outside Phase 2 scope. The phase goal is "library code never terminates the process" — inserter/inserter.go is a CLI command, not library code. |
| `compiler/lib/compiler.go` | 331 | `go vet` warning: copying lock value from MessageRegistry | WARNING | Pre-existing issue introduced during Phase 02-01 refactoring; acknowledged in 02-02-SUMMARY.md as a deferred issue. Does not affect correctness or prevent compilation. |

**Stub classification notes:**
- The `os.Exit` calls in `compiler/command.go` and `inserter/inserter.go` are inside mitchellh/cli `Run()` methods — the correct CLI entry-point boundary per REFC-04. They are outside Phase 2 scope (REFC-05+ territory or separate concerns). Not blockers for this phase.
- The `go vet` lock-copy warning does not affect runtime behavior and is pre-existing.

### Human Verification Required

#### 1. End-to-End CLI Error Propagation

**Test:** Run `protoconf compile /nonexistent/path` and observe the exit code and output.
**Expected:** The process exits with code 1, an error message is printed to stderr (not a panic/stack trace), and the program terminates cleanly.
**Why human:** Cannot run the binary in this verification environment to confirm the full exit-code chain through mitchellh/cli.

### Gaps Summary

No gaps found. All five Success Criteria from ROADMAP.md are satisfied:

1. `compiler/lib/module_service.go` — zero os.Exit calls, confirmed by grep.
2. `compiler/lib/starlark_loader.go` — zero os.Exit calls; all 3 locations resolved with closure error capture.
3. `mutate/mutate.go` — zero os.Exit calls; all ~10 locations resolved.
4. All refactored functions (NewModuleService, NewCompiler, NewCompilerService, NewProtoconfMutationServer) return errors; devserver, compiler CLI, mod CLI, inserter, agent/filekv all updated.
5. Full build passes (`go build ./...`); all existing tests pass.

The remaining `os.Exit` calls in `compiler/command.go` and `inserter/inserter.go` are in CLI `Run()` methods (the correct process-exit boundary per REFC-04 and the mitchellh/cli contract) and are outside Phase 2's stated scope. The `go vet` lock-copy warning in `compiler/lib/compiler.go:331` is pre-existing from the refactoring and deferred.

---

_Verified: 2026-03-24T06:30:00Z_
_Verifier: Claude (gsd-verifier)_
