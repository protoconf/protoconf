# Phase 3: Observability & Global State Cleanup - Context

**Gathered:** 2026-03-27
**Status:** Ready for planning

<domain>
## Phase Boundary

Extract shared OTel bootstrap from duplicated code in server/server.go and agent/agent.go into a single shared package. Make OTel initialization non-fatal (log and continue). Move Starlark resolve.* global settings to program startup. Remove package-level global grpc.ClientConn from mutate/mutate.go.

</domain>

<decisions>
## Implementation Decisions

### Shared OTel Package Design
- **D-01:** Create a new top-level `observability/` package (consistent with existing top-level packages like `consts/`, `command/`)
- **D-02:** Package exposes a single `Init(ctx context.Context, serviceName string) (shutdown func(context.Context) error, err error)` function
- **D-03:** Init sets up both tracer and meter exporters (otlptracegrpc + otlpmetricgrpc), resource detection (FromEnv, Process, OS, Container, Host), and service name/version attributes
- **D-04:** Return a shutdown function — let callers choose cleanup strategy (context.AfterFunc in server, defer in agent). This preserves existing shutdown patterns while eliminating code duplication.

### OTel Failure Handling
- **D-05:** When OTel exporter creation fails, log a warning via slog and continue with no-op tracer/meter providers — the process must not crash because a telemetry collector is unavailable
- **D-06:** Return error from Init but make it a "soft" error — callers should log it but not abort. Consider returning a no-op shutdown function alongside the error so callers don't need nil checks.

### Starlark Resolve Initialization
- **D-07:** Move `resolve.AllowNestedDef`, `resolve.AllowLambda`, `resolve.AllowFloat`, `resolve.AllowSet`, `resolve.AllowGlobalReassign`, `resolve.AllowRecursion` out of `NewCompiler()` constructor
- **D-08:** Use a `sync.Once` guard in compiler/lib to ensure resolve settings are configured exactly once, regardless of how many Compiler instances are created. This eliminates both redundant writes and potential data race on concurrent NewCompiler calls.

### Mutate Global State
- **D-09:** Move `var conn *grpc.ClientConn` from package-level (mutate.go:30) to a local variable inside the `Run()` method where it's created and used
- **D-10:** Leave the global `ui` var as-is — it's a CLI presentation concern initialized at package load time, not mutable connection state

### Claude's Discretion
- Exact internal structure of the observability package (helper functions, option patterns)
- Whether to use functional options for Init or keep it simple with positional args
- otelkv.go tracer initialization — whether to also use the shared package or leave as-is (it's a different pattern: per-package tracer, not bootstrap)
- Whether to add any unit tests for the new observability package in this phase or defer to Phase 9

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### OTel Bootstrap (duplicated code to extract)
- `server/server.go` lines 106-149 — Server-side OTel initialization with context.AfterFunc shutdown
- `agent/agent.go` lines 60-103 — Agent-side OTel initialization with defer shutdown

### Global State to Fix
- `compiler/lib/compiler.go` lines 35-41 — Starlark resolve.* settings in NewCompiler constructor
- `mutate/mutate.go` line 30 — Package-level `var conn *grpc.ClientConn`

### Related Observability Code
- `agent/otelkv/otelkv.go` — OTel-instrumented KV store wrapper (131 lines, uses per-package tracer)
- `agent/agent.go` lines 171-172 — grpc_prometheus interceptor registration
- `agent/agent.go` line 117 — slog-otel bridge setup

### Requirements
- `.planning/REQUIREMENTS.md` — REFC-05, REFC-06, REFC-07, REFC-08

No external specs — requirements fully captured in decisions above and REQUIREMENTS.md.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `consts.Version` — already used by both server and agent for OTel service version attribute
- `otel.Tracer()` / `otel.Meter()` — standard OTel global provider pattern already in use
- `otlptracegrpc` and `otlpmetricgrpc` — exporter packages already imported in both files
- `autodetect.NewResourceDetection()` — resource detection utility already used identically in both

### Established Patterns
- Top-level utility packages: `consts/`, `command/` — new `observability/` follows this pattern
- Error returns from constructors (Phase 2 pattern): `NewCompiler() (*Compiler, error)` — observability Init should follow same signature style
- `sync.Once` is used elsewhere in Go stdlib patterns — natural fit for resolve settings

### Integration Points
- `server/server.go` RunServer function — replace OTel init block with `observability.Init()` call
- `agent/agent.go` RunAgent function — replace OTel init block with `observability.Init()` call
- `compiler/lib/compiler.go` NewCompiler — remove resolve.* lines, add sync.Once initializer
- `mutate/mutate.go` Run method — move `conn` from package to local scope
- `devserver/command.go` — may also need OTel init if it combines agent+server (verify during planning)

</code_context>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches. The OTel extraction is a straightforward deduplication with graceful error handling. The global state fixes are mechanical refactoring.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 03-observability-global-state-cleanup*
*Context gathered: 2026-03-27*
