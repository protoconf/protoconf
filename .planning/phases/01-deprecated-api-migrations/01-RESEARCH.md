# Phase 1: Deprecated API Migrations - Research

**Researched:** 2026-03-23
**Domain:** gRPC-Go deprecated API migration (grpc.WithInsecure, grpc.Dial/DialContext, grpc_reflection_v1alpha)
**Confidence:** HIGH

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** Replace `grpc.WithInsecure()` with `grpc.WithTransportCredentials(insecure.NewCredentials())` — only 1 location: `compiler/command.go:101`. All other files already use the new API.
- **D-02:** Migrate all 3 usages in `server/server.go` (import at line 50, file registration at line 231, server registration at line 337) from `grpc_reflection_v1alpha` to `grpc_reflection_v1`.
- **D-03:** Also migrate `grpc.Dial` to `grpc.NewClient` across the codebase. Files: `compiler/command.go`, `mutate/mutate.go`, example files, and `grpc.DialContext` in `agent/legacy.go`, `test/e2e.go`, `agent/kv_agent_impl_test.go`, `server/server.go`.
- **D-04:** Update example code in `examples/` directory alongside production code to keep examples consistent as reference material.

### Claude's Discretion
- Handle any import path changes or minor API adjustments needed for the `grpc.NewClient` migration (e.g., different connection behavior, option changes)

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| DEPR-01 | All grpc.WithInsecure() calls migrated to grpc.WithTransportCredentials(insecure.NewCredentials()) | Confirmed: 1 call site in compiler/command.go:101. All other files already use the new API. Change is mechanical. |
| DEPR-02 | grpc_reflection_v1alpha migrated to grpc_reflection_v1 | Confirmed: 3 sites in server/server.go (import, RegisterFile, RegisterServer). v1 package available in grpc@v1.64.0, RegisterServerReflectionServer exists, file descriptor name changes from `File_grpc_reflection_v1alpha_reflection_proto` to `File_grpc_reflection_v1_reflection_proto`. |
</phase_requirements>

---

## Summary

This phase replaces three categories of deprecated gRPC-Go APIs with their current stable equivalents. The changes are purely mechanical: no logic changes, no behavior changes in production paths, and no dependency version bumps are required. The current module already uses `google.golang.org/grpc v1.64.0` which provides all the replacement APIs.

The most impactful change is `grpc.Dial`/`grpc.DialContext` to `grpc.NewClient`. There is one behavioral difference: `grpc.Dial` and `grpc.DialContext` use `"passthrough"` as the default name resolver scheme, while `grpc.NewClient` uses `"dns"`. For every call site in this codebase, the target is either an explicit host:port address (where both schemes work identically) or the empty string `""` used with `grpc.WithContextDialer` (in-process bufconn connections, where the scheme is irrelevant). No compatibility issues.

**Primary recommendation:** Three independent, self-contained changes. Execute as three separate commits: (1) grpc.WithInsecure migration, (2) grpc_reflection_v1alpha migration, (3) grpc.Dial/DialContext migration. Compile and test after each commit.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| google.golang.org/grpc | v1.64.0 (already in go.mod) | gRPC client/server | The only gRPC library used |
| google.golang.org/grpc/credentials/insecure | (part of grpc module) | Insecure transport credentials | The non-deprecated way to connect without TLS |
| google.golang.org/grpc/reflection/grpc_reflection_v1 | (part of grpc module) | gRPC server reflection v1 | Replaces deprecated v1alpha |

No new dependencies. No go.mod changes needed.

### Deprecated → Replacement Map

| Deprecated | Replacement | Package Change? |
|------------|-------------|----------------|
| `grpc.WithInsecure()` | `grpc.WithTransportCredentials(insecure.NewCredentials())` | No (insecure already imported at these sites) |
| `grpc.Dial(target, opts...)` | `grpc.NewClient(target, opts...)` | No (same package) |
| `grpc.DialContext(ctx, target, opts...)` | `grpc.NewClient(target, opts...)` | No — drop the ctx parameter |
| `grpc_reflection_v1alpha.RegisterServerReflectionServer(...)` | `grpc_reflection_v1.RegisterServerReflectionServer(...)` | Yes — new import alias |
| `grpc_reflection_v1alpha.File_grpc_reflection_v1alpha_reflection_proto` | `grpc_reflection_v1.File_grpc_reflection_v1_reflection_proto` | Yes — file descriptor constant name changes |

## Architecture Patterns

### Call Site Inventory (verified by direct source inspection)

**DEPR-01: grpc.WithInsecure (1 site)**

| File | Line | Current Code | Replacement |
|------|------|-------------|-------------|
| `compiler/command.go` | 101 | `grpc.Dial(config.compilerAddress, grpc.WithInsecure())` | `grpc.NewClient(config.compilerAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))` |

Note: `compiler/command.go` already imports `"google.golang.org/grpc"` but does NOT import `"google.golang.org/grpc/credentials/insecure"`. That import must be added.

**DEPR-02: grpc_reflection_v1alpha (3 sites, all in server/server.go)**

| Line | Current Code | Replacement |
|------|-------------|-------------|
| 50 (import) | `"google.golang.org/grpc/reflection/grpc_reflection_v1alpha"` | `"google.golang.org/grpc/reflection/grpc_reflection_v1"` |
| 231 | `parser.FilesResolver.RegisterFile(grpc_reflection_v1alpha.File_grpc_reflection_v1alpha_reflection_proto)` | `parser.FilesResolver.RegisterFile(grpc_reflection_v1.File_grpc_reflection_v1_reflection_proto)` |
| 337 | `grpc_reflection_v1alpha.RegisterServerReflectionServer(rpcServer, reflectionServer)` | `grpc_reflection_v1.RegisterServerReflectionServer(rpcServer, reflectionServer)` |

**D-03: grpc.Dial / grpc.DialContext (6 sites in production/test code + 2 in examples)**

| File | Line | Current | Replacement | Notes |
|------|------|---------|-------------|-------|
| `compiler/command.go` | 101 | `grpc.Dial(addr, grpc.WithInsecure())` | `grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))` | Combined with DEPR-01 fix |
| `mutate/mutate.go` | 172 | `grpc.Dial(address, grpc.WithTransportCredentials(...))` | `grpc.NewClient(address, grpc.WithTransportCredentials(...))` | Drop global `var conn` (separate concern, Phase 3 REFC-08) |
| `agent/legacy.go` | 39 | `grpc.DialContext(ctx, "", grpc.WithContextDialer(...), grpc.WithTransportCredentials(...))` | `grpc.NewClient("", grpc.WithContextDialer(...), grpc.WithTransportCredentials(...))` | Drop ctx param |
| `test/e2e.go` | 28 | `grpc.DialContext(ctx, "", grpc.WithContextDialer(...), grpc.WithTransportCredentials(...))` | `grpc.NewClient("", grpc.WithContextDialer(...), grpc.WithTransportCredentials(...))` | Drop ctx param |
| `agent/kv_agent_impl_test.go` | 146 | `grpc.DialContext(ctx, "", grpc.WithContextDialer(...), grpc.WithTransportCredentials(...))` | `grpc.NewClient("", grpc.WithContextDialer(...), grpc.WithTransportCredentials(...))` | Drop ctx param |
| `server/server.go` | 521 | `grpc.DialContext(ctx, "", grpc.WithContextDialer(...), grpc.WithTransportCredentials(...))` | `grpc.NewClient("", grpc.WithContextDialer(...), grpc.WithTransportCredentials(...))` | Drop ctx param |
| `examples/mutation/go_client/main.go` | 65 | `grpc.Dial(address, grpc.WithTransportCredentials(...))` | `grpc.NewClient(address, grpc.WithTransportCredentials(...))` | Example code |
| `examples/grpc_clients/go_client/main.go` | 35 | `grpc.Dial(address, grpc.WithTransportCredentials(...))` | `grpc.NewClient(address, grpc.WithTransportCredentials(...))` | Example code |

### Key API Difference: grpc.NewClient vs grpc.Dial/DialContext

Verified from grpc@v1.64.0 source (`clientconn.go`):

1. **No ctx parameter:** `grpc.NewClient(target string, opts ...DialOption)` does not accept a context. For `DialContext` call sites, the ctx parameter is dropped entirely.
2. **Name resolver default:** `grpc.NewClient` uses `"dns"` as default; `grpc.Dial`/`DialContext` use `"passthrough"`. For all call sites in this codebase, this distinction is irrelevant (explicit host:port addresses or empty-string bufconn targets).
3. **No automatic connection:** `grpc.NewClient` does not exit idle mode; `grpc.Dial` calls `ExitIdleMode()`. For this codebase, the first RPC call will trigger the connection — no functional difference for the use patterns here.
4. **Same return type:** Both return `(*grpc.ClientConn, error)`.

### Pattern: grpc.DialContext with ctx → grpc.NewClient without ctx

```go
// BEFORE (deprecated)
conn, err := grpc.DialContext(ctx, "",
    grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
        return lis.Dial()
    }), grpc.WithTransportCredentials(insecure.NewCredentials()))

// AFTER (current)
conn, err := grpc.NewClient("",
    grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
        return lis.Dial()
    }), grpc.WithTransportCredentials(insecure.NewCredentials()))
```

### Pattern: grpc.Dial with address → grpc.NewClient

```go
// BEFORE (deprecated)
conn, err := grpc.Dial(config.compilerAddress, grpc.WithInsecure())

// AFTER (current, also fixes WithInsecure)
conn, err := grpc.NewClient(config.compilerAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
```

### Pattern: grpc_reflection_v1alpha → grpc_reflection_v1

```go
// BEFORE (import)
"google.golang.org/grpc/reflection/grpc_reflection_v1alpha"

// AFTER (import)
grpc_reflection_v1 "google.golang.org/grpc/reflection/grpc_reflection_v1"

// BEFORE (RegisterFile)
parser.FilesResolver.RegisterFile(grpc_reflection_v1alpha.File_grpc_reflection_v1alpha_reflection_proto)

// AFTER (RegisterFile)
parser.FilesResolver.RegisterFile(grpc_reflection_v1.File_grpc_reflection_v1_reflection_proto)

// BEFORE (RegisterServer)
grpc_reflection_v1alpha.RegisterServerReflectionServer(rpcServer, reflectionServer)

// AFTER (RegisterServer)
grpc_reflection_v1.RegisterServerReflectionServer(rpcServer, reflectionServer)
```

Note: `reflection.NewServer(...)` in `server/server.go` line 331-335 returns a `v1alphareflectiongrpc.ServerReflectionServer`. In grpc@v1.64.0, `reflection.NewServer()` returns an adapter that satisfies the v1alpha interface. `reflection.NewServerV1()` returns the v1 interface. Since we are now calling `grpc_reflection_v1.RegisterServerReflectionServer`, we must use `reflection.NewServerV1(opts)` instead of `reflection.NewServer(opts)`.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Insecure transport credentials | Custom DialOption | `insecure.NewCredentials()` from `google.golang.org/grpc/credentials/insecure` | Official API, already in module |
| gRPC connection without TLS | Custom transport | `grpc.WithTransportCredentials(insecure.NewCredentials())` | Single-expression replacement |
| v1alpha reflection server | Custom compatibility shim | `reflection.NewServerV1(opts)` + `grpc_reflection_v1.RegisterServerReflectionServer` | Both already in module |

## Common Pitfalls

### Pitfall 1: Forgetting reflection.NewServer → reflection.NewServerV1
**What goes wrong:** `reflection.NewServer(opts)` returns a `v1alpha.ServerReflectionServer` type. If you only change the import and registration call but keep `reflection.NewServer`, you will get a compile error because `grpc_reflection_v1.RegisterServerReflectionServer` expects a `v1.ServerReflectionServer`.
**Why it happens:** The `reflection` package has two constructors: `NewServer` (returns v1alpha adapter) and `NewServerV1` (returns v1).
**How to avoid:** When migrating line 337, also change line 331 from `reflection.NewServer(opts)` to `reflection.NewServerV1(opts)`.
**Warning signs:** Compile error: "cannot use reflectionServer (type v1alpha.ServerReflectionServer) as type v1.ServerReflectionServer".

### Pitfall 2: Missing insecure import in compiler/command.go
**What goes wrong:** `compiler/command.go` does not currently import `"google.golang.org/grpc/credentials/insecure"`. Adding `insecure.NewCredentials()` without adding the import causes a compile error.
**How to avoid:** Add `"google.golang.org/grpc/credentials/insecure"` to the import block.

### Pitfall 3: Passing ctx to grpc.NewClient
**What goes wrong:** `grpc.NewClient` does not accept a context as first argument. Passing ctx causes a compile error.
**Why it happens:** `DialContext` had ctx as first parameter for cancellation of the dial attempt. `NewClient` is always non-blocking.
**How to avoid:** Remove the ctx argument entirely when converting `DialContext` calls. The `ctx` variable is still available for downstream use.

### Pitfall 4: Confusing the file descriptor constant names
**What goes wrong:** The file descriptor constant in v1alpha is `File_grpc_reflection_v1alpha_reflection_proto`; in v1 it is `File_grpc_reflection_v1_reflection_proto`. Using the wrong name compiles only if the old package is still imported.
**How to avoid:** After removing the v1alpha import, the compiler will immediately flag any remaining v1alpha symbol references.

### Pitfall 5: Treating grpc.Dial and grpc.NewClient as fully identical
**What goes wrong:** In rare cases (custom dialers, unusual target formats), the default name resolver change (passthrough → dns) can alter behavior.
**Why it matters here:** All bufconn targets use `""` as target with `grpc.WithContextDialer`. All real-address targets use explicit `host:port`. Neither is affected by the resolver change.
**How to avoid:** Verify each call site's target format before conversion — all sites in this codebase are safe.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `grpc.WithInsecure()` | `grpc.WithTransportCredentials(insecure.NewCredentials())` | gRPC-Go v1.40.0 (2021) | Deprecated; still functional in v1.64.0 |
| `grpc.Dial` / `grpc.DialContext` | `grpc.NewClient` | gRPC-Go v1.58+ | Deprecated; still functional in v1.64.0 |
| `grpc_reflection_v1alpha` | `grpc_reflection_v1` | gRPC-Go ~v1.45+ | v1alpha deprecated in favor of stable v1 |

## Open Questions

None — all call sites verified by direct source inspection. All replacement APIs confirmed present in grpc@v1.64.0.

## Environment Availability

Step 2.6: SKIPPED (no external dependencies — this is a pure code change within the existing Go module, no new tools or services required).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go standard `testing` package + testify v1.9.0 |
| Config file | none (standard go test) |
| Quick run command | `go test ./agent/... ./compiler/... ./server/... ./inserter/...` |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| DEPR-01 | grpc.WithInsecure removed, binary compiles | build | `go build ./...` | N/A (build check) |
| DEPR-02 | grpc_reflection_v1alpha removed, server package compiles and tests pass | unit | `go test ./server/...` | Yes (`server/server_test.go`) |
| D-03 | grpc.Dial/DialContext removed, all packages compile | build + unit | `go build ./... && go test ./...` | Yes (existing tests) |

### Sampling Rate
- **Per task commit:** `go build ./...` (fast build verification)
- **Per wave merge:** `go test ./...`
- **Phase gate:** `go test ./...` full suite green before `/gsd:verify-work`

### Wave 0 Gaps
None — existing test infrastructure covers all phase requirements. This phase does not require new tests; the changes are mechanical API substitutions that are validated by the compiler and existing tests.

## Sources

### Primary (HIGH confidence)
- Direct source inspection of `grpc@v1.64.0` in Go module cache (`/home/smintz/go/pkg/mod/google.golang.org/grpc@v1.64.0/`) — confirmed: `grpc.NewClient`, `reflection.NewServerV1`, `grpc_reflection_v1.RegisterServerReflectionServer`, `File_grpc_reflection_v1_reflection_proto` all exist
- Direct source inspection of all 8 target files in the repository — confirmed exact line numbers and current code
- `go test ./...` baseline run — all tests pass (`ok agent`, `ok compiler`, `ok server`, `ok inserter`)

### Secondary (MEDIUM confidence)
None needed — all claims verified from primary sources.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — verified from module cache and go.mod
- Architecture: HIGH — verified by direct file inspection of all call sites
- Pitfalls: HIGH — derived from API signatures in module cache and compile-time type system

**Research date:** 2026-03-23
**Valid until:** 2026-06-23 (stable APIs, not fast-moving)

## Project Constraints (from CLAUDE.md)

| Directive | Applies to This Phase |
|-----------|----------------------|
| Go 1.22+, CGO_ENABLED=0, static binaries | No new deps; no impact |
| No broken protobuf wire formats or gRPC service definitions | These changes are call-site only; no proto changes |
| `gofmt` enforced via Trunk | Run `gofmt` after each file edit |
| Error handling: use `errors.Join`, `fmt.Errorf`, sentinel vars | No new error handling needed here |
| Logging: use `slog.Default()` | No new logging |
| No `os.Exit` in libraries (Phase 2 concern) | Out of scope for this phase |
| Must not break existing CI (GitHub Actions with Codecov) | `go build ./...` + `go test ./...` must pass |
| `snake_case.go` filenames, `PascalCase` exports | No new files |
| GSD workflow enforcement | Working within GSD workflow |
