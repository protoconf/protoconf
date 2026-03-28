# Phase 6: Token Auth & Script Security - Research

**Researched:** 2026-03-28
**Domain:** gRPC unary interceptors, bearer token auth, script execution security, Go stdlib security primitives
**Confidence:** HIGH

## Summary

Phase 6 adds token-based authentication to the mutation server (write path only), forwards auth credentials to pre/post scripts, and validates script paths at startup. All four requirements (SECR-04, SECR-05, SECR-06, SECR-07) are well-supported by the existing codebase without new dependencies.

The key implementation artifacts are: (1) a gRPC unary interceptor using `grpc.ChainUnaryInterceptor` added to `serverOpts`, (2) `crypto/subtle.ConstantTimeCompare` for timing-safe token comparison, (3) env var additions in `runScript`, and (4) an `os.Stat` + mode-bit check in `run()`. All patterns follow directly from Phase 5's TLS work and the agent's existing `grpc.UnaryInterceptor` usage.

**Primary recommendation:** Implement auth as a single `grpc.UnaryServerInterceptor` function appended to `serverOpts` via `grpc.ChainUnaryInterceptor` (to compose with the existing `otelgrpc.NewServerHandler` stats handler). Add `--auth-token` flag to `cliConfig`/`newFlagSet`. Fix the `PROTOCONF_COMPILER_ADDR` env var bug while adding the new env vars. Add script path validation in `run()`.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** Static API key (bearer token) sent via gRPC metadata header `authorization: Bearer <token>`. No JWT, no OIDC.
- **D-02:** Server validates by comparing against configured expected token. Constant-time comparison to prevent timing attacks.
- **D-03:** When auth is configured (`--auth-token` flag is set), requests without a valid token are rejected with `codes.Unauthenticated` gRPC status.
- **D-04:** When auth is NOT configured (no `--auth-token` flag), all requests are accepted — backward compatible. Log a warning at startup: "Mutation server running without authentication."
- **D-05:** Add `--auth-token` flag to mutation server's `cliConfig` and `newFlagSet()` in `server/server.go`. This is the expected bearer token value.
- **D-06:** Implement auth check as a gRPC unary interceptor, not inline in `MutateConfig`. This keeps auth logic separate and reusable.
- **D-07:** The interceptor extracts the token from `metadata.FromIncomingContext(ctx)` using the `authorization` key.
- **D-08:** Forward the raw auth token to pre/post scripts as `PROTOCONF_AUTH_TOKEN` environment variable.
- **D-09:** Forward the `script_metadata` field from `ConfigMutationRequest` as `PROTOCONF_SCRIPT_METADATA` environment variable. This field already exists in the proto but is unused.
- **D-10:** Fix the existing bug in `runScript`: `"PROTOCONF_COMPILER_ADDR"+s.config.grpcAddress` is missing `=` — should be `"PROTOCONF_COMPILER_ADDR="+s.config.grpcAddress`.
- **D-11:** Validate script paths at server startup (in the `run()` method), not per-request. If `--pre` or `--post` script flags are set, verify the file exists and is executable. Fail startup with a clear error if validation fails.
- **D-12:** Reject paths containing `..` to prevent path traversal. Scripts must be absolute paths or relative to CWD.
- **D-13:** At runtime in `runScript`, re-check file existence before `exec.Command` as a defense-in-depth measure.

### Claude's Discretion

- Whether to use `crypto/subtle.ConstantTimeCompare` directly or wrap it in a helper
- Exact gRPC interceptor registration approach (unary only vs stream+unary — mutation service is unary-only)
- Whether to extract the auth interceptor into a separate file or keep it in `server/server.go`
- Whether the startup warning for no-auth should match the no-TLS warning format from Phase 5

### Deferred Ideas (OUT OF SCOPE)

- mTLS support (SECR-08)
- Role-based authorization on config paths (SECR-09)
- Auth for the agent (read path)
- JWT/OIDC token validation
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SECR-04 | Mutation server supports token-based authentication (JWT or API key) via gRPC metadata | gRPC metadata API + `grpc.ChainUnaryInterceptor` + `crypto/subtle` |
| SECR-05 | Auth credentials are forwarded to pre/post mutation scripts as environment variables | `runScript` env var pattern already in use; add `PROTOCONF_AUTH_TOKEN` and `PROTOCONF_SCRIPT_METADATA` |
| SECR-06 | Unauthenticated requests are rejected when auth is configured | Interceptor returns `status.Error(codes.Unauthenticated, ...)` — standard gRPC pattern |
| SECR-07 | Pre/post mutation script paths are validated (exist, executable) before execution | `os.Stat` + mode bit check in `run()` before server starts |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `google.golang.org/grpc` | v1.64.0 | `grpc.ChainUnaryInterceptor`, `grpc.UnaryServerInterceptor` | Already used in codebase; `ChainUnaryInterceptor` is the correct API for multiple interceptors |
| `google.golang.org/grpc/metadata` | (grpc v1.64.0) | Extract `authorization` header from incoming context | Already imported in `server/server.go` (line 44) |
| `google.golang.org/grpc/codes` | (grpc v1.64.0) | `codes.Unauthenticated` for rejected requests | Standard gRPC error code for auth failures |
| `google.golang.org/grpc/status` | (grpc v1.64.0) | `status.Error(codes.Unauthenticated, msg)` | Standard gRPC error creation |
| `crypto/subtle` | Go stdlib | `subtle.ConstantTimeCompare` for timing-safe token comparison | Prevents timing attacks on secret comparisons |
| `strings` | Go stdlib | `strings.TrimPrefix` for `Bearer ` prefix stripping | Already used in `server/server.go` |
| `os` | Go stdlib | `os.Stat` for script file existence check | Standard file system probe |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/stretchr/testify` | v1.9.0 | Test assertions | All new test cases |

**No new dependencies needed.** All required packages are already in the Go standard library or already imported in `server/server.go`.

**Version verification:** All packages verified against current `go.mod` — no version changes needed.

## Architecture Patterns

### Recommended Project Structure

No new files required. All changes land in:
```
server/
├── server.go          # cliConfig, newFlagSet, run(), runScript(), MutateConfig() changes
└── server_test.go     # New test cases for auth interceptor and script validation
```

Optionally extract to a separate file (at Claude's discretion):
```
server/
├── server.go          # unchanged interface
└── auth.go            # bearerTokenInterceptor() + validateScriptPath() helpers
```

### Pattern 1: gRPC Unary Auth Interceptor

**What:** A `grpc.UnaryServerInterceptor` that extracts the `authorization` header, validates it, and either calls `handler` or returns `codes.Unauthenticated`.

**When to use:** Always for mutation server when `--auth-token` is set. No-op (pass-through) when not configured.

**Critical detail:** The server already uses `grpc.StatsHandler(otelgrpc.NewServerHandler())` as a `ServerOption`. Since `grpc.NewServer` accepts at most one `UnaryInterceptor` option (additional calls override), use `grpc.ChainUnaryInterceptor` to compose with the existing prometheus interceptor pattern (see agent.go line 133). The mutation server currently has NO unary interceptor — but to be future-proof and composable, use `grpc.ChainUnaryInterceptor` rather than `grpc.UnaryInterceptor`.

```go
// Source: google.golang.org/grpc documentation + agent/agent.go:133 pattern
func bearerTokenInterceptor(expectedToken string) grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
        if expectedToken == "" {
            // Auth not configured — pass through (backward compatible)
            return handler(ctx, req)
        }
        md, ok := metadata.FromIncomingContext(ctx)
        if !ok {
            return nil, status.Error(codes.Unauthenticated, "missing metadata")
        }
        values := md.Get("authorization")
        if len(values) == 0 {
            return nil, status.Error(codes.Unauthenticated, "missing authorization header")
        }
        token := strings.TrimPrefix(values[0], "Bearer ")
        if subtle.ConstantTimeCompare([]byte(token), []byte(expectedToken)) != 1 {
            return nil, status.Error(codes.Unauthenticated, "invalid token")
        }
        return handler(ctx, req)
    }
}
```

**Registration in `run()`:**
```go
// Add after existing serverOpts setup
serverOpts = append(serverOpts, grpc.ChainUnaryInterceptor(bearerTokenInterceptor(config.authToken)))
```

### Pattern 2: Dynamic Service Handler — Interceptor Already Supported

The `Init()` method already handles the `interceptor` parameter in its dynamically-registered `grpc.MethodDesc.Handler` (lines 282-300). When `interceptor != nil`, it calls `interceptor(ctx, in, info, handler)`. This means the auth interceptor added to `grpc.NewServer(serverOpts...)` will automatically apply to dynamically-registered proto services.

The standard `MutateConfig` and `ReportProgress` handlers registered via `protoconfmutation.RegisterProtoconfMutationServiceServer` and `protoconf_pb.RegisterProtoconfMutationServiceServer` also go through the interceptor chain automatically — no changes needed to those registrations.

### Pattern 3: Token Forwarding to Scripts

**What:** Add new env vars to `cmd.Env` in `runScript`. The method needs the auth token and the `script_metadata` value — pass them as additional parameters or store in the server struct.

**Two options:**
1. Pass `authToken` and `scriptMetadata` as parameters to `runScript(filename, uuid, authToken, scriptMetadata string)`
2. Store `authToken` in `ProtoconfMutationServer` struct; pass `scriptMetadata` as parameter

**Recommended:** Option 1 (parameter approach) — avoids mutating the struct after construction. `MutateConfig` already has `in.ScriptMetadata` in scope; `config.authToken` is accessible via `s.config.authToken`. Pass both from the call sites in `MutateConfig`.

```go
func (s *ProtoconfMutationServer) runScript(filename string, uuid string, authToken string, scriptMetadata string) error {
    cmd := exec.Command(filename)
    cmd.Env = append(cmd.Env,
        "PROTOCONF_MUTATION_UUID="+uuid,
        "PROTOCONF_COMPILER_ADDR="+s.config.grpcAddress,  // bug fix: was missing "="
        "PROTOCONF_AUTH_TOKEN="+authToken,
        "PROTOCONF_SCRIPT_METADATA="+scriptMetadata,
    )
    // ... rest unchanged
}
```

### Pattern 4: Script Path Validation at Startup

**What:** In `run()`, after `protoconfServer.config = config`, validate any non-empty `--pre`/`--post` paths before starting the server.

```go
// Source: os.Stat documentation + fs.FileMode standard patterns
func validateScriptPath(path string) error {
    if strings.Contains(path, "..") {
        return fmt.Errorf("script path must not contain '..': %q", path)
    }
    info, err := os.Stat(path)
    if err != nil {
        return fmt.Errorf("script path does not exist: %w", err)
    }
    if info.IsDir() {
        return fmt.Errorf("script path is a directory, not a file: %q", path)
    }
    if info.Mode()&0111 == 0 {
        return fmt.Errorf("script path is not executable: %q", path)
    }
    return nil
}
```

**Defense-in-depth in `runScript`** (D-13): Add `os.Stat` check before `exec.Command` — return an error if the file no longer exists. This protects against TOCTOU but does not replace startup validation.

### Pattern 5: Startup Warning for No-Auth

Match the TLS warning format from Phase 5 (`server/server.go` line 143):
```go
// Phase 5 pattern (reference):
slog.Warn("gRPC server running without TLS -- connections are not encrypted")

// Phase 6 pattern (to match):
slog.Warn("mutation server running without authentication -- requests are not authenticated")
```

### Anti-Patterns to Avoid

- **`strings.Compare` for token comparison:** Not constant-time. Use `crypto/subtle.ConstantTimeCompare`.
- **`grpc.UnaryInterceptor` when already using ChainUnaryInterceptor or multiple interceptors:** gRPC panics if you call `grpc.UnaryInterceptor` more than once. Use `grpc.ChainUnaryInterceptor` exclusively.
- **Inline auth check in `MutateConfig`:** D-06 is locked — interceptor pattern is required.
- **Per-request script path validation:** D-11 is locked — validate at startup only, with defense-in-depth re-check at runtime.
- **Logging the token value:** Never log `authToken` at any log level — treat it as a secret.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Timing-safe comparison | `token == expectedToken` or `strings.EqualFold` | `crypto/subtle.ConstantTimeCompare` | Prevents timing side-channel attacks that leak token length via early-exit comparison |
| gRPC error for auth failure | Custom error type | `status.Error(codes.Unauthenticated, msg)` | gRPC clients receive correct HTTP/2 status; grpcui shows correct error |
| Interceptor composition | Manual wrapper chain | `grpc.ChainUnaryInterceptor` | gRPC's built-in chain handles context propagation and panic recovery |

**Key insight:** All required primitives exist in the standard library and already-imported gRPC packages. No new `go get` commands needed.

## Common Pitfalls

### Pitfall 1: ConstantTimeCompare Length Leak
**What goes wrong:** `subtle.ConstantTimeCompare` returns 0 immediately when lengths differ (by spec — this is correct behavior). An attacker can detect wrong-length tokens via timing. However, for a static bearer token comparison where the token value is fixed at server startup, this is an acceptable tradeoff — the token length is not itself secret after the server is running.
**Why it happens:** Spec behavior of `crypto/subtle`.
**How to avoid:** This is acceptable for static tokens. The important thing is that it prevents character-by-character early-exit timing attacks against tokens of the correct length.
**Warning signs:** Only a concern if attempting to prevent token-length inference — out of scope here (D-01 says simple shared secret).

### Pitfall 2: Bearer Prefix Not Stripped
**What goes wrong:** Client sends `"Bearer mytoken123"`. Server compares `"Bearer mytoken123"` against `"mytoken123"` — always fails.
**Why it happens:** `metadata.Get("authorization")` returns the raw header value including the `Bearer ` prefix.
**How to avoid:** Always call `strings.TrimPrefix(values[0], "Bearer ")` before comparison. Unit test this case explicitly.

### Pitfall 3: Empty Metadata Values Slice
**What goes wrong:** `md.Get("authorization")` returns `nil` or empty slice when header absent. Indexing `values[0]` panics.
**Why it happens:** gRPC metadata `Get` returns nil slice for absent keys.
**How to avoid:** Always check `len(values) == 0` before accessing `values[0]`.

### Pitfall 4: `grpc.UnaryInterceptor` Called Twice
**What goes wrong:** `grpc.NewServer` panics with "The unary server interceptor was already set and may not be reset" if `grpc.UnaryInterceptor` appears more than once in server options.
**Why it happens:** `grpc.UnaryInterceptor` is a singular option — not idempotent.
**How to avoid:** Use `grpc.ChainUnaryInterceptor(interceptorA, interceptorB)` for all unary interceptors. The mutation server currently has zero unary interceptors, so `grpc.ChainUnaryInterceptor(bearerTokenInterceptor(...))` is safe even with a single interceptor.

### Pitfall 5: Dynamic Service Handlers and Interceptors
**What goes wrong:** Developer assumes interceptor won't fire for dynamically-registered services (those added by `Init()` via `rpcServer.RegisterService`).
**Why it happens:** Confusion about gRPC interceptor scope.
**How to avoid:** gRPC interceptors registered via `grpc.NewServer(opts...)` apply to ALL services registered on that server, including dynamic ones. The `Init()` method explicitly supports this — lines 289-299 check `if interceptor == nil` and call the interceptor when it's not nil. The auth interceptor will fire for dynamically-registered services automatically.

### Pitfall 6: Script Validation on Windows (Not Applicable)
**What goes wrong:** File mode bit `0111` check for executable doesn't apply to Windows.
**Why it happens:** Windows doesn't use Unix permission bits.
**How to avoid:** Target platforms are `linux/amd64`, `darwin/amd64`, `darwin/arm64` per `.goreleaser.yaml`. Windows is not a target. The `0111` check is correct for all target platforms.

### Pitfall 7: Path Traversal With `..` Check
**What goes wrong:** A script path like `/usr/local/bin/../bin/malicious` passes the `..` check but resolves differently than expected. Alternatively, `./scripts/../../etc/passwd` contains `..`.
**Why it happens:** String-based `..` check is necessary but should be applied to the raw path, not after resolution.
**How to avoid:** Check raw input for `..` component (D-12). Consider `filepath.Clean` to normalize and then check the cleaned path against the raw path — if they differ, the original contained traversal. Simple string check `strings.Contains(path, "..")` catches all cases in practice.

## Code Examples

### Auth Interceptor Registration
```go
// Source: agent/agent.go line 133 pattern + grpc.ChainUnaryInterceptor docs
serverOpts := []grpc.ServerOption{grpc.StatsHandler(otelgrpc.NewServerHandler())}
// ... TLS setup ...
serverOpts = append(serverOpts, grpc.ChainUnaryInterceptor(bearerTokenInterceptor(config.authToken)))
rpcServer := grpc.NewServer(serverOpts...)
```

### Interceptor Function Signature
```go
// Source: google.golang.org/grpc UnaryServerInterceptor type
type UnaryServerInterceptor func(ctx context.Context, req interface{}, info *UnaryServerInfo, handler UnaryHandler) (interface{}, error)
```

### Extracting Bearer Token from gRPC Metadata
```go
// Source: google.golang.org/grpc/metadata package + server/server.go line 240-241 pattern
md, ok := metadata.FromIncomingContext(ctx)
// md.Get returns []string — lowercase key per gRPC metadata convention
values := md.Get("authorization")
if len(values) == 0 {
    return nil, status.Error(codes.Unauthenticated, "missing authorization header")
}
token := strings.TrimPrefix(values[0], "Bearer ")
```

### Constant-Time Comparison
```go
// Source: crypto/subtle package docs
import "crypto/subtle"
if subtle.ConstantTimeCompare([]byte(token), []byte(expectedToken)) != 1 {
    return nil, status.Error(codes.Unauthenticated, "invalid token")
}
```

### Script Path Validation
```go
// Source: os.Stat + fs.FileMode standard library
info, err := os.Stat(path)
if err != nil {
    return fmt.Errorf("script path does not exist or is inaccessible: %w", err)
}
if info.Mode()&0111 == 0 {
    return fmt.Errorf("script path is not executable (mode %s): %q", info.Mode(), path)
}
```

### New Env Var in runScript
```go
// Source: server/server.go line 448 pattern
cmd.Env = append(cmd.Env,
    "PROTOCONF_MUTATION_UUID="+uuid,
    "PROTOCONF_COMPILER_ADDR="+s.config.grpcAddress,  // bug fix: was "PROTOCONF_COMPILER_ADDR"+addr
    "PROTOCONF_AUTH_TOKEN="+authToken,
    "PROTOCONF_SCRIPT_METADATA="+scriptMetadata,
)
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `grpc.UnaryInterceptor` (singular) | `grpc.ChainUnaryInterceptor` (composable) | grpc v1.x stable API | Allows multiple interceptors without panic |
| `strings.Compare` for secrets | `crypto/subtle.ConstantTimeCompare` | Security best practice | Timing attack prevention |

**Deprecated/outdated:**
- `grpc.WithInsecure()`: Migrated to `insecure.NewCredentials()` in Phase 1 — already complete.

## Open Questions

1. **`runScript` signature change: parameters vs struct field for auth token**
   - What we know: `authToken` is in `s.config.authToken` (accessible on receiver). `scriptMetadata` is in `in.ScriptMetadata` (local to `MutateConfig` caller).
   - What's unclear: Whether to pass both as parameters to `runScript` or store `scriptMetadata` elsewhere.
   - Recommendation: Pass both as parameters — `runScript(filename, uuid, authToken, scriptMetadata string)`. This makes the function self-contained and testable without needing a fully constructed `cliConfig`.

2. **Extract auth interceptor to `server/auth.go`**
   - What we know: TLS was kept in `utils/tls.go` (separate file). Auth is server-specific.
   - What's unclear: Whether the interceptor warrants its own file given the small size (~20 lines).
   - Recommendation: Keep in `server/server.go` for the first pass. Extract only if the file grows unwieldy.

## Environment Availability

Step 2.6: SKIPPED — phase is code-only changes with no external dependencies beyond the existing Go toolchain.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | `testing` + `testify` v1.9.0 |
| Config file | none (standard `go test`) |
| Quick run command | `go test ./server/ -run TestAuth -v` |
| Full suite command | `go test ./server/ -v` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SECR-04 | Valid bearer token passes interceptor | unit | `go test ./server/ -run Test_bearerTokenInterceptor/valid_token -v` | ❌ Wave 0 |
| SECR-04 | Missing `--auth-token` flag allows all requests (backward compat) | unit | `go test ./server/ -run Test_bearerTokenInterceptor/no_auth_configured -v` | ❌ Wave 0 |
| SECR-06 | No token when auth configured returns Unauthenticated | unit | `go test ./server/ -run Test_bearerTokenInterceptor/missing_header -v` | ❌ Wave 0 |
| SECR-06 | Wrong token returns Unauthenticated | unit | `go test ./server/ -run Test_bearerTokenInterceptor/invalid_token -v` | ❌ Wave 0 |
| SECR-06 | Token without Bearer prefix returns Unauthenticated | unit | `go test ./server/ -run Test_bearerTokenInterceptor/no_bearer_prefix -v` | ❌ Wave 0 |
| SECR-05 | PROTOCONF_AUTH_TOKEN set in script env | unit | `go test ./server/ -run Test_runScript/auth_token_env -v` | ❌ Wave 0 |
| SECR-05 | PROTOCONF_SCRIPT_METADATA set in script env | unit | `go test ./server/ -run Test_runScript/script_metadata_env -v` | ❌ Wave 0 |
| SECR-05 | PROTOCONF_COMPILER_ADDR bug fix (= separator) | unit | `go test ./server/ -run Test_runScript/compiler_addr_env -v` | ❌ Wave 0 |
| SECR-07 | Non-existent script path fails startup | unit | `go test ./server/ -run Test_validateScriptPath/nonexistent -v` | ❌ Wave 0 |
| SECR-07 | Non-executable script path fails startup | unit | `go test ./server/ -run Test_validateScriptPath/not_executable -v` | ❌ Wave 0 |
| SECR-07 | Path containing `..` fails validation | unit | `go test ./server/ -run Test_validateScriptPath/path_traversal -v` | ❌ Wave 0 |
| SECR-07 | Valid executable script path passes startup | unit | `go test ./server/ -run Test_validateScriptPath/valid -v` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./server/ -v`
- **Per wave merge:** `go test ./server/ -v`
- **Phase gate:** `go test ./server/ -v` — full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `server/server_test.go` — extend with `Test_bearerTokenInterceptor`, `Test_validateScriptPath`, and updated `Test_server_MutateConfig` cases for auth (test cases for new behavior added alongside implementation)

*(The test file `server/server_test.go` already exists — new test functions are added to it, not a new file.)*

## Sources

### Primary (HIGH confidence)
- Go standard library `crypto/subtle` — `ConstantTimeCompare` function signature and behavior verified via `go doc`
- Go standard library `os` — `Stat`, `FileInfo.Mode()` verified via `go doc`
- `google.golang.org/grpc` v1.64.0 — `ChainUnaryInterceptor`, `UnaryServerInterceptor`, `metadata` package verified via `go doc`
- `google.golang.org/grpc/status` — `status.Error`, `codes.Unauthenticated` verified via `go doc`
- `server/server.go` — Full source read; existing patterns for metadata extraction (line 240-241), env vars (lines 448-451), `serverOpts` assembly (lines 128-145) confirmed
- `agent/agent.go` lines 130-134 — `grpc.UnaryInterceptor` pattern confirmed as reference

### Secondary (MEDIUM confidence)
- Phase 5 TLS implementation (`utils/tls.go`, `server/server.go`) — startup warning format (line 143) confirmed as the pattern to match

### Tertiary (LOW confidence)
- None — all findings verified against source code and stdlib docs

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependencies, all stdlib + already-imported packages
- Architecture: HIGH — implementation points identified precisely (specific file + line numbers confirmed by reading source)
- Pitfalls: HIGH — derived from direct code inspection of the specific interceptor path and existing tests
- Test patterns: HIGH — existing `server_test.go` reviewed; test infrastructure is in place

**Research date:** 2026-03-28
**Valid until:** 2026-04-27 (stable stdlib + grpc API)
