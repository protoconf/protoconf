# Phase 1: Deprecated API Migrations - Context

**Gathered:** 2026-03-23
**Status:** Ready for planning

<domain>
## Phase Boundary

Replace all deprecated gRPC API calls with their current stable equivalents. This covers two specific deprecations: `grpc.WithInsecure()` and `grpc_reflection_v1alpha`.

</domain>

<decisions>
## Implementation Decisions

### grpc.WithInsecure() Migration
- **D-01:** Replace `grpc.WithInsecure()` with `grpc.WithTransportCredentials(insecure.NewCredentials())` — only 1 location needs fixing: `compiler/command.go:101`. All other files already use the new API.

### grpc_reflection_v1alpha Migration
- **D-02:** Migrate all 3 usages in `server/server.go` (import at line 50, file registration at line 231, server registration at line 337) from `grpc_reflection_v1alpha` to `grpc_reflection_v1`.

### grpc.Dial Deprecation
- **D-03:** Also migrate `grpc.Dial` to `grpc.NewClient` across the codebase. `grpc.Dial` is deprecated in favor of `grpc.NewClient` which has different default behavior (no automatic connection). This avoids a second deprecation pass later. Files using `grpc.Dial`: `compiler/command.go`, `mutate/mutate.go`, example files.

### Example Code
- **D-04:** Update example code in `examples/` directory alongside production code to keep examples consistent as reference material.

### Claude's Discretion
- Handle any import path changes or minor API adjustments needed for the `grpc.NewClient` migration (e.g., different connection behavior, option changes)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Source Files to Modify
- `compiler/command.go` — Uses deprecated `grpc.WithInsecure()` at line 101 and `grpc.Dial`
- `server/server.go` — Uses `grpc_reflection_v1alpha` at lines 50, 231, 337
- `mutate/mutate.go` — Uses `grpc.Dial` at line 172
- `agent/legacy.go` — Uses `grpc.Dial` at line 42
- `test/e2e.go` — Uses `grpc.Dial` at line 31
- `agent/kv_agent_impl_test.go` — Uses `grpc.Dial` at line 149
- `examples/mutation/go_client/main.go` — Uses `grpc.Dial` at line 65
- `examples/grpc_clients/go_client/main.go` — Uses `grpc.Dial` at line 35

### Codebase Analysis
- `.planning/codebase/CONCERNS.md` — Documents deprecated API usage and migration paths

No external specs — requirements fully captured in decisions above.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- Most files already use `grpc.WithTransportCredentials(insecure.NewCredentials())` — only `compiler/command.go` uses the old `grpc.WithInsecure()`
- The `insecure` package is already imported in files that need it

### Established Patterns
- gRPC connections follow a consistent pattern: `grpc.Dial(address, opts...)` with transport credentials
- Server reflection is registered via a single call in `server/server.go`

### Integration Points
- `server/server.go` is the only file using reflection APIs — self-contained change
- `compiler/command.go` line 101 is the only deprecated `WithInsecure` call

</code_context>

<specifics>
## Specific Ideas

No specific requirements — straightforward API migration following gRPC-Go deprecation guidance.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 01-deprecated-api-migrations*
*Context gathered: 2026-03-23*
