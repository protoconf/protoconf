# Phase 4: Dead Code Removal - Research

**Researched:** 2026-03-27
**Domain:** Go dead code cleanup — unused init() functions and unreachable error checks
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Remove only the `init()` function in `inserter/inserter.go` (lines 47-53) that sets `runtime.GOMAXPROCS(runtime.NumCPU())` — this is a no-op since Go 1.5 already defaults to NumCPU
- **D-02:** Scope limited to inserter/inserter.go per REFC-09 — do not scan or modify init() functions in other packages
- **D-03:** Remove the unreachable `if err != nil { return }` block at `agent/filekv/filekv.go` lines 146-148 — this checks `err` from line 137 which was already handled at lines 138-141 and cannot be non-nil at that point
- **D-04:** Do not modify surrounding code, TODO comments, or other logic in filekv.go — only remove the dead check per REFC-10

### Claude's Discretion
- Whether to also remove the `runtime` import if it becomes unused after init() removal
- Exact formatting after dead code removal (gofmt will handle this)

### Deferred Ideas (OUT OF SCOPE)
- None — discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| REFC-09 | inserter/inserter.go unnecessary runtime.GOMAXPROCS init() function removed | Code read confirms init() at lines 47-53 sets runtime.GOMAXPROCS(runtime.NumCPU()). After removal, `runtime` package becomes unused — import must also be removed. `go vet` and compiler enforce no unused imports in Go. |
| REFC-10 | filekv.Watch dead error check at lines 143-145 cleaned up | Code read confirms `if err != nil { return }` at lines 146-148 checks `err` last assigned at line 137 (`b, err := proto.Marshal(...)`) which was fully handled at lines 138-141 (`if err != nil { ... return }`). The check at 146-148 is unreachable — `err` cannot be non-nil there. Safe to remove. |
</phase_requirements>

## Summary

Phase 4 is the smallest phase in the milestone: two surgical line-level removals in two files. No new code is introduced. No logic is changed. Only dead statements are removed.

**REFC-09 (inserter/inserter.go):** Lines 47-53 contain an `init()` function whose sole purpose is to call `runtime.GOMAXPROCS(runtime.NumCPU())`. Since Go 1.5 (released 2015), the Go runtime already defaults GOMAXPROCS to the number of available CPUs. This call is a no-op and the entire `init()` block can be deleted. After removal, `runtime` is no longer referenced anywhere in `inserter/inserter.go`, so the `"runtime"` import on line 14 must also be removed — Go's compiler will reject unused imports.

**REFC-10 (agent/filekv/filekv.go):** Lines 146-148 contain `if err != nil { return }`. The variable `err` was last assigned at line 137 (`b, err := proto.Marshal(protoconfValue)`). That assignment's error was already fully handled at lines 138-141 (`if err != nil { slog.Error(...); return }`). If that block did not return, `err` is guaranteed to be `nil` at line 146. The check is dead and can be removed without changing any observable behavior.

Both changes are in separate packages. Tests pass today (`go test ./inserter/... ok`) and filekv has no test files. The existing inserter tests cover `Run()` and `InsertConfigFile()` and will continue to pass after init() removal because the init() function's effect — setting GOMAXPROCS to NumCPU — is already the runtime default.

**Primary recommendation:** Remove the init() block and its `runtime` import from inserter/inserter.go; remove the three dead lines (146-148) from agent/filekv/filekv.go. Run `go test ./inserter/... ./agent/filekv/...` and `go build ./...` to verify.

## Standard Stack

No new libraries. This phase touches only the Go standard library and existing project code.

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go compiler | 1.22+ | Enforces no unused imports | Built-in; unused import = compile error |
| `gofmt` | 1.22+ | Re-format after line removal | Standard Go formatter |

### Supporting
None — pure removal.

### Alternatives Considered
None applicable — there is no alternative to removing dead code.

**Installation:**
No installation required.

## Architecture Patterns

### Exact Code to Remove

#### REFC-09: inserter/inserter.go lines 47-53

Current code (lines 47-53):
```go
func init() {
	// set the number of CPUs to use.
	// By default, the number of CPUs is the number of CPUs on the machine.
	// Just to show we can change the number of CPUs
	// I have added this below command.
	runtime.GOMAXPROCS(runtime.NumCPU())
}
```

Remove: The entire `init()` block above (7 lines).

Current import block (line 14): `"runtime"` — must also be removed since `runtime` is not referenced anywhere else in the file.

Resulting import block: remove the `"runtime"` line from the import group on lines 3-37. `gofmt` handles blank line consolidation automatically.

#### REFC-10: agent/filekv/filekv.go lines 146-148

Current code (lines 142-148):
```go
		watchCh <- &store.KVPair{
			Key:   key,
			Value: []byte(base64.StdEncoding.EncodeToString(b)),
		}
		if err != nil {
			return
		}
```

Remove: lines 146-148 only (`if err != nil { return }`). Lines 142-145 (the channel send) remain unchanged.

### Anti-Patterns to Avoid
- **Widening scope:** D-02 and D-04 explicitly lock scope. Do not touch other init() functions, other error checks, or surrounding logic.
- **Leaving unused imports:** After removing the init() body, `runtime` has zero references in inserter.go. The Go compiler will reject the build. Remove the import in the same edit.
- **Removing the init() in filekv.go:** filekv.go has its own `init()` at lines 28-30 that registers the store with Valkeyrie — this is a functional init() and must NOT be touched.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Import cleanup | Manual string search | `goimports` or compiler error | Go compiler enforces unused import removal; let the compiler catch it |
| Formatting after removal | Manual spacing | `gofmt` | Standard tool handles all whitespace/blank-line normalization |

## Runtime State Inventory

Step 2.5: SKIPPED — This is not a rename/refactor/migration phase. No stored data, live service config, OS-registered state, secrets, or build artifacts reference the removed code.

## Environment Availability

Step 2.6: SKIPPED — No external dependencies. Phase is code-only changes to two Go source files. Go toolchain is already confirmed available (CI uses Go 1.22; local `go test` ran successfully).

## Common Pitfalls

### Pitfall 1: Leaving the `runtime` import after init() removal
**What goes wrong:** Build fails with `"runtime" imported and not used`
**Why it happens:** Go compiler treats unused imports as errors, not warnings
**How to avoid:** Remove `"runtime"` from the import block in the same edit that removes the init() function
**Warning signs:** `go build ./inserter/...` fails immediately

### Pitfall 2: Touching filekv.go's functional init()
**What goes wrong:** filekv store stops being registered with Valkeyrie; any code that calls `valkeyrie.NewStore(ctx, filekv.StoreName, ...)` panics at runtime with "unregistered store"
**Why it happens:** filekv.go has two init-related things: the Valkeyrie registration init() at lines 28-30 (functional) and the dead error check at lines 146-148 (target). They are in different locations.
**How to avoid:** Only remove lines 146-148. The init() at lines 28-30 is untouched.
**Warning signs:** Any agent test using filekv store fails

### Pitfall 3: Removing the channel send alongside the dead check
**What goes wrong:** Watch() goroutine never sends updates to callers; effectively breaks the watch mechanism
**Why it happens:** Lines 142-145 (the `watchCh <- &store.KVPair{...}` send) immediately precede the dead check. Careless selection might grab lines 142-148 instead of 146-148.
**How to avoid:** Remove exactly lines 146-148. Lines 142-145 stay.
**Warning signs:** filekv Watch goroutine sends no values after the first read

### Pitfall 4: Assuming `go vet` is currently clean on filekv
**What goes wrong:** Developer runs `go vet ./agent/filekv/...` and sees 10 pre-existing errors about value-receiver methods on a struct containing sync.Mutex, then believes the phase introduced them
**Why it happens:** These are pre-existing issues unrelated to REFC-10 (they concern receiver types on methods like Put, Get, Delete, etc.)
**How to avoid:** Accept that `go vet ./agent/filekv/...` has pre-existing failures; verify only that no NEW errors appear after the edit. The pre-existing vet errors are out of scope per D-04.
**Warning signs:** `go vet` output matches exactly the 10 pre-existing "passes lock by value" lines

## Code Examples

### Verified: inserter.go import block after removal
```go
import (
    "bytes"
    "context"
    "encoding/base64"
    "errors"
    "flag"
    "fmt"
    "log/slog"
    "os"
    "os/signal"
    "path/filepath"
    "strings"
    "sync"
    "time"

    "github.com/go-git/go-git/v5"
    "github.com/kvtools/consul"
    "github.com/kvtools/etcdv3"
    "github.com/kvtools/valkeyrie"
    "github.com/kvtools/valkeyrie/store"
    "github.com/kvtools/zookeeper"
    "github.com/mitchellh/cli"
    "github.com/protoconf/protoconf/agent/configmaps"
    "github.com/protoconf/protoconf/command"
    "github.com/protoconf/protoconf/compiler/lib"
    "github.com/protoconf/protoconf/compiler/lib/parser"
    "github.com/protoconf/protoconf/consts"
    protoconf_pb "github.com/protoconf/protoconf/pb/protoconf/v1"
    "google.golang.org/protobuf/encoding/protojson"
    "google.golang.org/protobuf/proto"
    "google.golang.org/protobuf/types/dynamicpb"
    "google.golang.org/protobuf/types/known/durationpb"
    "google.golang.org/protobuf/types/known/timestamppb"
)
```
Source: direct read of inserter/inserter.go lines 3-37; `"runtime"` on line 14 is the only removal.

### Verified: filekv.go Watch goroutine after removal (lines 130-158)
```go
    for {
        protoconfValue := &protoconfvalue.ProtoconfValue{}
        err := s.parser.ReadConfig(absPath, protoconfValue)
        if err != nil {
            slog.Error("Error reading config", "Error", err)
            return
        }
        b, err := proto.Marshal(protoconfValue)
        if err != nil {
            slog.Error("Error Marshaling config", "Error", err)
            return
        }
        watchCh <- &store.KVPair{
            Key:   key,
            Value: []byte(base64.StdEncoding.EncodeToString(b)),
        }

        select {
        case _, ok := <-fsCh:
            if !ok {
                return
            }
        case <-ctx.Done():
            return
        }
    }
```
Source: direct read of filekv.go; lines 146-148 are the only removal.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Manual GOMAXPROCS in init() | Go runtime defaults GOMAXPROCS to NumCPU | Go 1.5 (2015) | Setting it manually is a no-op; init() is pure noise |

**Deprecated/outdated:**
- `runtime.GOMAXPROCS(runtime.NumCPU())` at program startup: Unnecessary since Go 1.5. Go's runtime sets this automatically. Official Go docs confirm the default is `runtime.NumCPU()`.

## Open Questions

None. Both removals are unambiguous based on direct code inspection.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go standard `testing` package + `github.com/stretchr/testify v1.9.0` |
| Config file | none — standard `go test` |
| Quick run command | `go test ./inserter/... ./agent/filekv/...` |
| Full suite command | `go test ./...` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| REFC-09 | inserter package builds without `runtime` import; init() is gone; existing tests still pass | unit + compile | `go build ./inserter/... && go test ./inserter/...` | Yes — inserter/inserter_test.go |
| REFC-10 | filekv.Watch goroutine compiles and logic is correct after dead check removal | compile + smoke | `go build ./agent/filekv/...` | No test file — compile-only verification |

### Sampling Rate
- **Per task commit:** `go test ./inserter/... && go build ./agent/filekv/...`
- **Per wave merge:** `go test ./...`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `agent/filekv/filekv_test.go` — no test file exists for filekv; compile verification only for REFC-10. Creating a test file is out of scope for this phase (TEST-05 is Phase 9). Compile check is sufficient since the removed code is demonstrably unreachable.

## Sources

### Primary (HIGH confidence)
- Direct read: `inserter/inserter.go` lines 1-53 — confirmed init() block, runtime import, and that no other code references `runtime` package
- Direct read: `agent/filekv/filekv.go` lines 110-161 — confirmed dead `if err != nil { return }` at lines 146-148
- Direct read: `inserter/inserter_test.go` — confirmed existing tests cover Run() and InsertConfig(); init() removal does not affect test behavior

### Secondary (MEDIUM confidence)
- Go 1.5 release notes (training data): GOMAXPROCS defaults to NumCPU since Go 1.5; init() in question is confirmed no-op
- `go test ./inserter/... ok` — tests pass today; baseline established

### Tertiary (LOW confidence)
- None.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new libraries, all changes are removals
- Architecture: HIGH — code read directly; exact lines identified
- Pitfalls: HIGH — derived from direct code inspection and Go language rules

**Research date:** 2026-03-27
**Valid until:** Stable — removals are non-controversial; Go import rules and GOMAXPROCS default do not change
