# Phase 6: Token Auth & Script Security - Context

**Gathered:** 2026-03-28
**Status:** Ready for planning

<domain>
## Phase Boundary

Add token-based authentication to the mutation server. When auth is configured, unauthenticated requests are rejected. Auth credentials are forwarded to pre/post mutation scripts as environment variables. Script paths are validated before execution. The agent (read-only) does NOT get auth — only the mutation server (write path).

</domain>

<decisions>
## Implementation Decisions

### Auth Token Format — SECR-04, SECR-06
- **D-01:** Static API key (bearer token) sent via gRPC metadata header `authorization: Bearer <token>`. No JWT, no OIDC — simple shared secret.
- **D-02:** Server validates by comparing against configured expected token. Constant-time comparison to prevent timing attacks.
- **D-03:** When auth is configured (`--auth-token` flag is set), requests without a valid token are rejected with `codes.Unauthenticated` gRPC status.
- **D-04:** When auth is NOT configured (no `--auth-token` flag), all requests are accepted — backward compatible. Log a warning at startup: "Mutation server running without authentication."

### Auth Configuration
- **D-05:** Add `--auth-token` flag to mutation server's `cliConfig` and `newFlagSet()` in `server/server.go`. This is the expected bearer token value.
- **D-06:** Implement auth check as a gRPC unary interceptor, not inline in `MutateConfig`. This keeps auth logic separate and reusable.
- **D-07:** The interceptor extracts the token from `metadata.FromIncomingContext(ctx)` using the `authorization` key.

### Credential Forwarding to Scripts — SECR-05
- **D-08:** Forward the raw auth token to pre/post scripts as `PROTOCONF_AUTH_TOKEN` environment variable.
- **D-09:** Forward the `script_metadata` field from `ConfigMutationRequest` as `PROTOCONF_SCRIPT_METADATA` environment variable. This field already exists in the proto but is unused.
- **D-10:** Fix the existing bug in `runScript`: `"PROTOCONF_COMPILER_ADDR"+s.config.grpcAddress` is missing `=` — should be `"PROTOCONF_COMPILER_ADDR="+s.config.grpcAddress`.

### Script Path Validation — SECR-07
- **D-11:** Validate script paths at server startup (in the `run()` method), not per-request. If `--pre` or `--post` script flags are set, verify the file exists and is executable. Fail startup with a clear error if validation fails.
- **D-12:** Reject paths containing `..` to prevent path traversal. Scripts must be absolute paths or relative to CWD.
- **D-13:** At runtime in `runScript`, re-check file existence before `exec.Command` as a defense-in-depth measure (file could be deleted after startup).

### Claude's Discretion
- Whether to use `crypto/subtle.ConstantTimeCompare` directly or wrap it in a helper
- Exact gRPC interceptor registration approach (unary only vs stream+unary — mutation service is unary-only)
- Whether to extract the auth interceptor into a separate file or keep it in `server/server.go`
- Whether the startup warning for no-auth should match the no-TLS warning format from Phase 5

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Mutation server
- `server/server.go` — Current mutation server implementation: `cliConfig` struct (line 62), `newFlagSet()` (line 71), `runScript()` (line 446), `MutateConfig()` (line 235)
- `server/api/proto/v1/protoconf_mutation.proto` — Mutation request/response proto with `script_metadata` field

### Auth requirements
- `.planning/REQUIREMENTS.md` — SECR-04 through SECR-07 define the auth and script security requirements

### TLS context (builds on Phase 5)
- `.planning/phases/05-tls-support/05-CONTEXT.md` — TLS implementation decisions and patterns used for server flag config
- `utils/tls.go` — Shared TLS helper created in Phase 5, pattern to follow for shared auth helpers

### Proto definitions
- `server/api/proto/v1/protoconf_mutation.proto` — `ConfigMutationRequest.script_metadata` field (line 12)
- `pb/protoconf/v1/protoconf.proto` — `ConfigMutationResponse` with script duration fields

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `server/server.go` `cliConfig` struct and `newFlagSet()` — established pattern for adding new flags (used in Phase 5 for TLS flags)
- `server/server.go` `runScript()` — existing script execution function that needs env var additions and bug fix
- `google.golang.org/grpc/metadata` — already imported in `server/server.go` for path extraction
- `ConfigMutationRequest.script_metadata` — existing proto field, currently unused, can be forwarded to scripts

### Established Patterns
- Mutation server uses manual `flag.FlagSet` with `cliConfig` struct (not proto-defined config like agent)
- gRPC metadata extraction already done in `MutateConfig` for the `path` field (line 240)
- Script env vars set via `cmd.Env = append(cmd.Env, ...)` in `runScript` (line 448)
- Server options built as `[]grpc.ServerOption` slice, interceptors can be appended

### Integration Points
- `server/server.go` `grpc.NewServer(serverOpts...)` — add unary interceptor for auth
- `server/server.go` `runScript()` — add auth token and script_metadata env vars
- `server/server.go` `run()` — add script path validation before server starts
- `server/server.go` `newFlagSet()` — add `--auth-token` flag

</code_context>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches for Go gRPC auth interceptors with bearer tokens.

</specifics>

<deferred>
## Deferred Ideas

- mTLS support (SECR-08) — deferred to v2 per REQUIREMENTS.md
- Role-based authorization on config paths (SECR-09) — deferred to v2
- Auth for the agent (read path) — not in scope, agent is read-only
- JWT/OIDC token validation — overkill for current needs, static token is sufficient

</deferred>

---

*Phase: 06-token-auth-script-security*
*Context gathered: 2026-03-28*
