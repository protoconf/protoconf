# Phase 2: os.Exit Refactoring - Research

**Researched:** 2026-03-24
**Domain:** Go error propagation patterns, library/CLI boundary design
**Confidence:** HIGH

## Summary

Phase 2 removes `os.Exit` calls from three library-layer files (`compiler/lib/module_service.go`, `compiler/lib/starlark_loader.go`, `mutate/mutate.go`) and propagates errors to CLI entry points where process termination is appropriate. This is a purely mechanical refactoring: each `os.Exit(1)` inside a function becomes a `return error`, callers receive the error and either propagate it further or, at the CLI layer, return a non-zero int which `command/command.go` converts to an `os.Exit`.

The three target files contain **14 total os.Exit calls** across three distinct refactoring shapes:
1. **module_service.go** — one exit in a helper that returns a plain string; signature must change
2. **starlark_loader.go** — three exits inside a closure passed to `RangeFiles`; the closure's return type must change
3. **mutate/mutate.go** — ten exits inside a single `Run(args []string) int` method; all exits convert directly to `return 1` since `Run` is already the CLI entry point

The mitchellh/cli `Command` interface requires `Run(args []string) int`, so `mutate/mutate.go` is already at the CLI boundary — its `os.Exit(1)` calls are wrong but the fix is `return 1`, not introducing a new error return.

**Primary recommendation:** Refactor in three independent tasks matching the three files. The `mutate/mutate.go` refactoring is strictly mechanical (os.Exit -> return 1). The `module_service.go` and `starlark_loader.go` refactorings require signature changes and must be done together since `starlark_loader.go` calls `module_service.getProtoconfPath()` indirectly via `getCacheDir()`.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| REFC-01 | compiler/lib/module_service.go os.Exit replaced with error return | `getProtoconfPath()` signature change from `string` to `(string, error)`; all callers updated |
| REFC-02 | compiler/lib/starlark_loader.go os.Exit calls (3 locations) replaced with error returns | Closure in `loadValidators()` passed to `RangeFiles` must signal errors; requires a local error accumulator or early termination via bool return |
| REFC-03 | mutate/mutate.go os.Exit calls (~10 locations) replaced with error returns | All exits are inside `Run(args []string) int` — replace with `return 1` (already CLI boundary) |
| REFC-04 | All refactored functions propagate errors to CLI entry points where os.Exit is appropriate | `mutate/mutate.go` already is CLI; `module_service` errors propagate through `Compiler`/`NewCompiler` to `compiler/command.go`; `starlark_loader` errors surface via `loadValidators` -> `loadConfig` -> `Compiler.load` -> `CompileFile` -> CLI |
</phase_requirements>

## Detailed os.Exit Inventory

### compiler/lib/module_service.go — 1 exit

| Line | Function | Context | Fix |
|------|----------|---------|-----|
| 67 | `getProtoconfPath()` | `filepath.Abs` fails (extremely unlikely with a relative path) | Change return to `(string, error)`; all callers propagate |

**Caller chain for `getProtoconfPath()`:**
- `getCacheDir()` — calls `getProtoconfPath()`, returns plain string
- `getLockFile()` — calls `getProtoconfPath()`, returns plain string
- `Init()`, `Add()`, `Lock()`, `Sync()`, `Download()`, etc. — all call `getCacheDir()` or `getLockFile()`

Because `getProtoconfPath` is called from many helpers, the cleanest fix is:
1. Change `getProtoconfPath()` to `(string, error)`
2. Change `getCacheDir()` to `(string, error)` and `getLockFile()` to `(string, error)`
3. Update all callers in the same file to propagate errors

Alternatively, resolve the path once at construction time in `NewModuleService` and store it (simpler and eliminates all the propagation noise). Since the path is set at creation and never changes, resolution at construction is idiomatic Go.

**Recommended approach:** Resolve `filepath.Abs(protoconfRoot)` in `NewModuleService`, store the resolved path back to `m.Config.ProtoconfPath`. Then `getProtoconfPath()` becomes a trivial `return m.Config.ProtoconfPath` with no error. `NewModuleService` itself may need to return `(*ModuleService, error)` or the caller (in `NewCompiler`) can handle the error there.

### compiler/lib/starlark_loader.go — 3 exits

All three exits are inside `loadValidators()` at lines 124, 127, 138, inside a closure passed to `l.parser.FilesResolver.RangeFiles(func(fd protoreflect.FileDescriptor) bool)`.

| Line | Condition | Fix |
|------|-----------|-----|
| 124 | `stat` returns error | Capture error in outer scope; return false to stop iteration |
| 127 | `stat` shows a directory where file expected | Same: capture error, return false |
| 138 | `l.Load` fails for validator file | Same: capture error, return false |

The `RangeFiles` callback returns `bool` — returning `false` stops iteration. The fix is a standard closure-captures-error pattern:

```go
func (l *starlarkLoader) loadValidators() (map[string]*starlark.Function, error) {
    validators := make(map[string]*starlark.Function)
    l.Modules["add_validator"] = starlark.NewBuiltin("add_validator", starAddValidator(&validators))
    var walkErr error
    l.parser.FilesResolver.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
        // ... check file existence ...
        if err != nil {
            walkErr = fmt.Errorf("error loading validator %s: %w", validatorAbsPath, err)
            return false // stop iteration
        }
        return true
    })
    if walkErr != nil {
        return nil, walkErr
    }
    return validators, nil
}
```

`loadValidators` already returns `(map[string]*starlark.Function, error)` — only the internal closure needs fixing. The error value is already wired through `loadConfig` -> `Compiler.load` -> `Compiler.CompileFile` -> CLI `Run`.

### mutate/mutate.go — 10 exits

All 10 exits are inside `func (c *cliCommand) Run(args []string) int`. This is the `mitchellh/cli.Command` interface's `Run` method — it is already the CLI entry point. The correct fix is simply replacing every `os.Exit(1)` with `return 1`.

Two helper functions (`setNumeric`, `setFloat`) also call `os.Exit(1)` after parsing failures. These should return errors instead:

| Helper | Current | Fix |
|--------|---------|-----|
| `setNumeric(msg, key, val, typer)` | calls `os.Exit(1)` on parse failure | return `error`; caller in `Run` does `return 1` |
| `setFloat(msg, key, val, typer)` | calls `os.Exit(1)` on parse failure | return `error`; caller in `Run` does `return 1` |
| `setField(msg, key, val, typer)` | no exit; logs on error | no change needed (error already swallowed by design) |

The global `var conn *grpc.ClientConn` (REFC-08, deferred to Phase 3) should remain untouched in this phase — Phase 2 is strictly os.Exit removal.

The `flag.ExitOnError` in `newFlagSet()` is NOT in scope for this phase — it is a separate concern and acceptable at the CLI layer.

## Architecture Patterns

### Pattern 1: Resolve-at-construction (for module_service)
**What:** Move `filepath.Abs` call from helper to constructor, store canonical path.
**When to use:** When a value is derived once from immutable input and reused many times.
**Example:**
```go
// In NewModuleService:
absPath, err := filepath.Abs(protoconfRoot)
if err != nil {
    return nil, fmt.Errorf("invalid protoconf root %q: %w", protoconfRoot, err)
}
config := &module.ModuleServiceConfig{
    ProtoconfPath: absPath,  // store resolved path
    ...
}
```
`NewModuleService` signature changes from `func NewModuleService(protoconfRoot string) *ModuleService` to `func NewModuleService(protoconfRoot string) (*ModuleService, error)`. The caller `NewCompiler` already checks errors; `NewModuleService` error can be logged and cause `NewCompiler` to return a partially-initialized compiler or panic at startup (acceptable in main-path initialization at program start).

**Note:** `NewCompiler` currently does not return an error. If `NewModuleService` returns an error, `NewCompiler` must handle it — options are: (a) log+panic in NewCompiler (caller is always CLI), (b) return `(*Compiler, error)` from `NewCompiler` and propagate. Option (b) is cleaner but requires updating `compiler/command.go` and `server/server.go` callers.

### Pattern 2: Closure error capture (for starlark_loader)
**What:** Declare `var walkErr error` before the closure, assign inside it, return `false` to stop, check after.
**When to use:** When a callback does not accept an error return but iteration must abort on failure.
**Example:**
```go
var walkErr error
l.parser.FilesResolver.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
    if err := someOp(); err != nil {
        walkErr = err
        return false
    }
    return true
})
if walkErr != nil {
    return nil, walkErr
}
```

### Pattern 3: Replace os.Exit with return (for mutate CLI)
**What:** Inside a `Run(args []string) int` method, replace `os.Exit(1)` with `return 1` and optionally log before returning.
**When to use:** When the function is already the CLI entry point (mitchellh/cli Command interface).
**Example:**
```go
// Before:
slog.Error("could not find typeUrl", "msg", config.protoMsg, "error", err)
os.Exit(1)

// After:
slog.Error("could not find typeUrl", "msg", config.protoMsg, "error", err)
return 1
```

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Stopping RangeFiles iteration on error | Custom iteration wrapper | `return false` from callback | RangeFiles bool return is designed for this |
| Error propagation in CLI | Custom exit manager | `return 1` from `Run` + `os.Exit` in `command/command.go` | mitchellh/cli already calls os.Exit with the returned int |

**Key insight:** `command/command.go:49` (`os.Exit(exitStatus)`) is the single correct process exit point for all CLI commands. All `Run()` methods should return int codes, never call `os.Exit` directly.

## Common Pitfalls

### Pitfall 1: Changing NewCompiler signature mid-phase
**What goes wrong:** `NewCompiler` is called from `compiler/command.go`, `server/server.go`, and `devserver/command.go`. Changing its return signature to `(*Compiler, error)` cascades changes across three files.
**Why it happens:** The os.Exit in `getProtoconfPath` is deep in the call chain; fixing it cleanly requires the constructor to propagate errors.
**How to avoid:** Either (a) resolve path in constructor and accept that `NewModuleService` returns `(*ModuleService, error)` — then update all three callers of `NewCompiler`, or (b) keep `getProtoconfPath` returning `(string, error)` and propagate through each helper. Option (a) is a single larger change but produces cleaner code.
**Warning signs:** If you see `NewCompiler` called without error check in 3+ places, option (b) may have less blast radius.

### Pitfall 2: Breaking the starlark_loader RangeFiles loop semantics
**What goes wrong:** The `RangeFiles` callback returns `bool` for continue/stop; not returning `false` on error means iteration continues silently past the error.
**Why it happens:** The pattern of using a bool return to signal "continue iterating" is easy to miss when converting from an os.Exit imperative style.
**How to avoid:** Always pair error capture with `return false` in RangeFiles callbacks. Verify no validator gets silently skipped after an error.

### Pitfall 3: Leaving flag.ExitOnError in scope
**What goes wrong:** `newFlagSet` in `mutate/mutate.go` uses `flag.ExitOnError` — this calls `os.Exit(2)` on unknown flags. This is acceptable (it's in the CLI layer) and should NOT be changed in this phase.
**Why it happens:** Conflating CLI-layer os.Exit (acceptable) with library-layer os.Exit (problematic).
**How to avoid:** Only remove os.Exit from the three target files listed in REFC-01/02/03. Do not refactor `flag.ExitOnError` or `compiler/command.go` pprof profile creation exits.

### Pitfall 4: Forgetting to remove the os import
**What goes wrong:** After removing all os.Exit calls, the `os` import may no longer be needed in a file. Leaving it causes a compile error.
**Why it happens:** os is used for both os.Exit and os.Stdin/os.Stderr etc; check whether remaining os usage exists.
**How to avoid:** After refactoring each file, check whether `os` package is still used for other calls (os.ReadFile, os.Stderr, etc.) before removing the import.

### Pitfall 5: setNumeric/setFloat — changing signatures breaks the dispatch switch
**What goes wrong:** The `setNumeric` and `setFloat` helpers in `mutate/mutate.go` are called inside a `switch` statement. If they return errors, all 14 call sites in the switch need to be updated to check the error and return 1.
**Why it happens:** Multiple homogeneous call sites easy to miss one.
**How to avoid:** Use the pattern `if err := setNumeric(...); err != nil { slog.Error(...); return 1 }` for each call site, or introduce a local error accumulator and check once after the switch.

## Code Examples

### module_service.go — getProtoconfPath fix (resolve at construction)
```go
// Source: project codebase + Go standard library
// In NewModuleService:
func NewModuleService(protoconfRoot string) (*ModuleService, error) {
    absPath, err := filepath.Abs(protoconfRoot)
    if err != nil {
        return nil, fmt.Errorf("invalid protoconf root %q: %w", protoconfRoot, err)
    }
    const lockFile = `protoconf.lock`
    const cacheDir = `.protoconf_cache`
    config := &module.ModuleServiceConfig{
        ProtoconfPath: absPath,
        CacheDir:      cacheDir,
        LockFile:      lockFile,
    }
    return &ModuleService{
        Config:      config,
        mutex:       sync.RWMutex{},
        downloadMux: &sync.Mutex{},
        head: &module.RemoteRepo{
            Url:  ".",
            Deps: map[string]*module.RemoteRepo{},
        },
    }, nil
}

// getProtoconfPath becomes trivial (no error possible):
func (m *ModuleService) getProtoconfPath() string {
    return m.Config.ProtoconfPath
}
```

### starlark_loader.go — closure error capture fix
```go
// Source: project codebase pattern
func (l *starlarkLoader) loadValidators() (map[string]*starlark.Function, error) {
    validators := make(map[string]*starlark.Function)
    l.Modules["add_validator"] = starlark.NewBuiltin("add_validator", starAddValidator(&validators))
    var walkErr error
    l.parser.FilesResolver.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
        protoFile := fd.Path()
        validatorFile := protoFile + consts.ValidatorExtensionSuffix
        validatorAbsPath := filepath.Join(l.srcDir, validatorFile)
        exists, isDir, err := stat(validatorAbsPath)
        if err != nil {
            walkErr = fmt.Errorf("error getting file stat for validator %s: %w", validatorAbsPath, err)
            return false
        }
        if isDir {
            walkErr = fmt.Errorf("expected validator file, got directory: %s", validatorAbsPath)
            return false
        }
        if !exists {
            return true
        }
        thread := &starlark.Thread{Print: starPrint, Load: l.Load}
        if _, err := l.Load(thread, filepath.ToSlash(validatorFile)); err != nil {
            walkErr = fmt.Errorf("error loading validator %s: %w", validatorFile, err)
            return false
        }
        return true
    })
    if walkErr != nil {
        return nil, walkErr
    }
    return validators, nil
}
```

### mutate/mutate.go — typical os.Exit -> return 1 conversion
```go
// Before:
root, err := filepath.Abs(config.protoconfRoot)
if err != nil {
    slog.Error("failed to get root path", "error", err)
    os.Exit(1)
}

// After:
root, err := filepath.Abs(config.protoconfRoot)
if err != nil {
    slog.Error("failed to get root path", "error", err)
    return 1
}
```

### mutate/mutate.go — setNumeric/setFloat returning errors
```go
// Before:
func setNumeric(msg *dynamic.Message, key, val string, typer typerFunc) {
    i, err := strconv.ParseInt(val, 0, 64)
    if err != nil {
        slog.Error("error parsing int", "error", err)
        os.Exit(1)
    }
    setField(msg, key, i, typer)
}

// After:
func setNumeric(msg *dynamic.Message, key, val string, typer typerFunc) error {
    i, err := strconv.ParseInt(val, 0, 64)
    if err != nil {
        return fmt.Errorf("error parsing int field %q value %q: %w", key, val, err)
    }
    setField(msg, key, i, typer)
    return nil
}
// Callers in Run switch:
if err := setNumeric(msg, ret[0], ret[1], ...); err != nil {
    slog.Error("error parsing field", "error", err)
    return 1
}
```

## Caller Impact Analysis

| File Changed | Callers Affected | Change Required |
|---|---|---|
| `compiler/lib/module_service.go` (if NewModuleService signature changes) | `compiler/lib/compiler.go:NewCompiler` | Handle `(*ModuleService, error)` return |
| `compiler/lib/compiler.go` (if NewCompiler signature changes) | `compiler/command.go`, `server/server.go`, `devserver/command.go` | Handle `(*Compiler, error)` return |
| `compiler/lib/starlark_loader.go` | `loadValidators` -> already returns error, no signature change | Internal refactor only |
| `mutate/mutate.go` | No external callers (self-contained CLI package) | Internal refactor only |

**Minimum blast radius option:** Keep `NewModuleService` and `NewCompiler` signatures unchanged. Instead, in `NewModuleService`, resolve the path and `panic` if it fails (extremely unlikely for valid input), or log+exit in `NewCompiler` (which is already CLI-adjacent). This minimizes changes but is not ideal. The recommended approach is clean propagation.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go standard `testing` package + `github.com/stretchr/testify v1.9.0` |
| Config file | none (standard `go test`) |
| Quick run command | `go test ./compiler/lib/... ./mutate/...` |
| Full suite command | `go test ./...` |

### Phase Requirements -> Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| REFC-01 | `module_service.getProtoconfPath` no longer calls os.Exit; invalid path returns error | unit | `go test ./compiler/lib/ -run TestModuleService` | Partial (module_service_test.go exists) |
| REFC-02 | `starlark_loader.loadValidators` returns error on stat failure or load failure; no os.Exit | unit | `go test ./compiler/lib/ -run TestCompile` | Partial (compiler_test.go likely covers) |
| REFC-03 | `mutate.Run` returns 1 on all error paths; no os.Exit | unit | `go test ./mutate/ -run TestRun` | No (no test files) |
| REFC-04 | Error propagates from library to CLI; CLI exits non-zero | integration | `go test ./...` (end-to-end compile test) | Partial |

### Sampling Rate
- **Per task commit:** `go test ./compiler/lib/... ./mutate/...`
- **Per wave merge:** `go test ./...`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `mutate/mutate_test.go` — covers REFC-03 (Run returns 1 on error paths)
- No new test infrastructure needed for REFC-01/02 — existing compiler/lib tests exercise the fixed code paths

## Environment Availability

Step 2.6: SKIPPED — this phase is purely code refactoring with no external tool or service dependencies beyond the Go toolchain already verified in Phase 1.

## Sources

### Primary (HIGH confidence)
- Direct codebase inspection of all three target files
- Go standard library `filepath.Abs` documentation — error returns `*PathError`
- `protoreflect.FileDescriptor.RangeFiles` signature from `google.golang.org/protobuf/reflect/protoreflect`
- `mitchellh/cli` Command interface: `Run(args []string) int`

### Secondary (MEDIUM confidence)
- Go community patterns for closure error capture with range/walk callbacks
- Project existing patterns in `compiler/lib/compiler.go` (errors.Join, sentinel errors)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new libraries needed
- Architecture: HIGH — patterns directly derived from existing code and standard Go idioms
- Pitfalls: HIGH — identified from direct code inspection of all 14 exit sites
- Caller impact: HIGH — all callers directly inspected

**Research date:** 2026-03-24
**Valid until:** Until any of the three target files are structurally changed
