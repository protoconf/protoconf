# Phase 3: Observability & Global State Cleanup - Research

**Researched:** 2026-03-27
**Domain:** OpenTelemetry extraction, sync.Once patterns, Go global variable elimination
**Confidence:** HIGH

## Summary

Phase 3 has four distinct but related tasks: extract duplicated OTel bootstrap code into a shared `observability/` package, make OTel init failures non-fatal (warning + noop providers), move Starlark `resolve.*` globals to a `sync.Once`-guarded initializer, and localize the package-level `grpc.ClientConn` in `mutate/mutate.go`.

The OTel extraction is a pure deduplication: `server/server.go` lines 106–149 and `agent/agent.go` lines 60–103 are nearly identical. Both call `otlptracegrpc.New`, `otlpmetricgrpc.New`, `resource.New` with identical detectors, and set the same `"protoconf"` service name attribute. The shared package will expose `Init(ctx, serviceName) (shutdown func(context.Context) error, err error)` and fall back to noop providers on exporter failure. The noop fallback uses the already-available `go.opentelemetry.io/otel/trace/noop` and `go.opentelemetry.io/otel/metric/noop` sub-packages that ship with OTel v1.27.0.

The Starlark resolve globals require a subtle finding: as of the current `go.starlark.net` version in go.mod, `AllowNestedDef`, `AllowLambda`, and `AllowFloat` are **already no-ops** (they are marked as "obsolete flags" in the package docs — standard features now always enabled). Only `AllowSet`, `AllowGlobalReassign`, and `AllowRecursion` are still operative. A `sync.Once` guard in `compiler/lib` that sets only the operative three is the correct approach. `devserver/command.go` does NOT initialize OTel — it creates a plain `grpc.NewServer()` with no OTel handlers — so it does not need to call the new observability package.

**Primary recommendation:** Create `observability/observability.go` with an `Init` function, update callers in server and agent to use it, wrap the three operative Starlark resolve flags in a package-level `sync.Once`, and move `conn` from package scope to inside `Run()` in mutate.

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** Create a new top-level `observability/` package (consistent with existing top-level packages like `consts/`, `command/`)
- **D-02:** Package exposes a single `Init(ctx context.Context, serviceName string) (shutdown func(context.Context) error, err error)` function
- **D-03:** Init sets up both tracer and meter exporters (otlptracegrpc + otlpmetricgrpc), resource detection (FromEnv, Process, OS, Container, Host), and service name/version attributes
- **D-04:** Return a shutdown function — let callers choose cleanup strategy (context.AfterFunc in server, defer in agent). This preserves existing shutdown patterns while eliminating code duplication.
- **D-05:** When OTel exporter creation fails, log a warning via slog and continue with no-op tracer/meter providers — the process must not crash because a telemetry collector is unavailable
- **D-06:** Return error from Init but make it a "soft" error — callers should log it but not abort. Consider returning a no-op shutdown function alongside the error so callers don't need nil checks.
- **D-07:** Move `resolve.AllowNestedDef`, `resolve.AllowLambda`, `resolve.AllowFloat`, `resolve.AllowSet`, `resolve.AllowGlobalReassign`, `resolve.AllowRecursion` out of `NewCompiler()` constructor
- **D-08:** Use a `sync.Once` guard in compiler/lib to ensure resolve settings are configured exactly once, regardless of how many Compiler instances are created. This eliminates both redundant writes and potential data race on concurrent NewCompiler calls.
- **D-09:** Move `var conn *grpc.ClientConn` from package-level (mutate.go:30) to a local variable inside the `Run()` method where it's created and used
- **D-10:** Leave the global `ui` var as-is — it's a CLI presentation concern initialized at package load time, not mutable connection state

### Claude's Discretion

- Exact internal structure of the observability package (helper functions, option patterns)
- Whether to use functional options for Init or keep it simple with positional args
- otelkv.go tracer initialization — whether to also use the shared package or leave as-is (it's a different pattern: per-package tracer, not bootstrap)
- Whether to add any unit tests for the new observability package in this phase or defer to Phase 9

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| REFC-05 | Shared observability package extracts duplicate OTel tracer/meter setup from server/server.go and agent/agent.go | OTel v1.27.0 noop packages available; code duplication confirmed at lines 60–103 (agent) and 106–149 (server) |
| REFC-06 | OTel init failures log warnings and continue instead of panicking | `trace/noop.NewTracerProvider()` and `metric/noop.NewMeterProvider()` are available in existing OTel version; no new imports needed |
| REFC-07 | Starlark resolve.* global settings moved to program startup, not Compiler constructor | sync.Once is standard library; only 3 of 6 flags are operative (see Standard Stack section) |
| REFC-08 | mutate/mutate.go global grpc.ClientConn moved to local scope within Run method | conn is assigned and deferred-closed at mutate.go:212–217; trivially localizable |
</phase_requirements>

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `go.opentelemetry.io/otel` | v1.27.0 | OTel API — `otel.SetTracerProvider`, `otel.SetMeterProvider` | Already in go.mod |
| `go.opentelemetry.io/otel/sdk/trace` | v1.27.0 | `trace.NewTracerProvider`, `trace.WithBatcher`, `trace.WithResource` | Already in go.mod |
| `go.opentelemetry.io/otel/sdk/metric` | v1.27.0 | `metric.NewMeterProvider`, `metric.NewPeriodicReader` | Already in go.mod |
| `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc` | v1.27.0 | OTLP gRPC trace exporter | Already in go.mod |
| `go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc` | v1.27.0 | OTLP gRPC metric exporter | Already in go.mod |
| `go.opentelemetry.io/otel/sdk/resource` | v1.27.0 | Resource detection (env, process, OS, container, host) | Already in go.mod |
| `go.opentelemetry.io/otel/semconv/v1.4.0` | (bundled) | `ServiceNameKey`, `ServiceVersionKey` constants | Already imported in both server and agent |
| `go.opentelemetry.io/otel/trace/noop` | v1.27.0 | No-op TracerProvider for fallback | Sub-package of otel/trace v1.27.0; no new dep |
| `go.opentelemetry.io/otel/metric/noop` | v1.27.0 | No-op MeterProvider for fallback | Sub-package of otel/metric v1.27.0; no new dep |
| `sync` (stdlib) | — | `sync.Once` for resolve init guard | Standard library |
| `go.starlark.net/resolve` | v0.0.0-20240314022150 | Starlark global options | Already in go.mod |

### Notes on Starlark Resolve Flags (HIGH confidence)

From `go doc go.starlark.net/resolve`:

```
// obsolete flags for features that are now standard. No effect.
AllowNestedDef = true
AllowLambda    = true
AllowFloat     = true
AllowBitwise   = true
```

Only THREE flags are operative at the current version:
- `resolve.AllowSet = true`
- `resolve.AllowGlobalReassign = true`
- `resolve.AllowRecursion = true`

The other three in NewCompiler (`AllowNestedDef`, `AllowLambda`, `AllowFloat`) are now no-ops per the package documentation. The `sync.Once` init block should set all six for forward compatibility, but implementors should note the three are currently inert.

The package docs also recommend using `syntax.FileOptions` per-file arguments instead of globals for new code. Since we are not changing the compilation engine in this phase, the `sync.Once` + globals approach is the correct minimal change per D-08.

## Architecture Patterns

### Recommended Project Structure
```
observability/
└── observability.go    # Init function, shutdown, noop fallback
```

No sub-packages needed. Single file, single exported function matches the `consts/` and `command/` top-level utility package pattern.

### Pattern 1: OTel Init with Noop Fallback

**What:** Attempt OTLP exporter creation. On failure, log warning and substitute noop providers. Always return a valid (possibly noop) shutdown function so callers never need nil checks.

**When to use:** Any long-running process that should remain operational when a telemetry collector is unreachable.

**Example:**
```go
// observability/observability.go
package observability

import (
    "context"
    "fmt"
    "log/slog"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
    "go.opentelemetry.io/otel/metric/noop"
    sdkmetric "go.opentelemetry.io/otel/sdk/metric"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
    tracenoop "go.opentelemetry.io/otel/trace/noop"
    "github.com/protoconf/protoconf/consts"
)

// Init initializes OTel tracer and meter providers.
// On exporter failure it logs a warning and falls back to noop providers.
// Always returns a non-nil shutdown function.
func Init(ctx context.Context, serviceName string) (func(context.Context) error, error) {
    resources, _ := resource.New(ctx,
        resource.WithFromEnv(),
        resource.WithProcess(),
        resource.WithOS(),
        resource.WithContainer(),
        resource.WithHost(),
        resource.WithAttributes(
            semconv.ServiceNameKey.String(serviceName),
            semconv.ServiceVersionKey.String(consts.Version),
        ),
    )

    expTracer, err := otlptracegrpc.New(ctx)
    if err != nil {
        slog.Warn("OTel trace exporter unavailable, using noop", "error", err)
        otel.SetTracerProvider(tracenoop.NewTracerProvider())
        otel.SetMeterProvider(noop.NewMeterProvider())
        return func(context.Context) error { return nil }, fmt.Errorf("trace exporter: %w", err)
    }

    tracerProvider := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(expTracer),
        sdktrace.WithResource(resources),
    )
    otel.SetTracerProvider(tracerProvider)

    expMeter, err := otlpmetricgrpc.New(ctx)
    if err != nil {
        slog.Warn("OTel metric exporter unavailable, using noop", "error", err)
        otel.SetMeterProvider(noop.NewMeterProvider())
        shutdown := func(ctx context.Context) error {
            return tracerProvider.Shutdown(ctx)
        }
        return shutdown, fmt.Errorf("metric exporter: %w", err)
    }

    meterProvider := sdkmetric.NewMeterProvider(
        sdkmetric.WithReader(sdkmetric.NewPeriodicReader(expMeter)),
        sdkmetric.WithResource(resources),
    )
    otel.SetMeterProvider(meterProvider)

    shutdown := func(ctx context.Context) error {
        var errs []error
        if err := tracerProvider.Shutdown(ctx); err != nil {
            errs = append(errs, err)
        }
        if err := meterProvider.Shutdown(ctx); err != nil {
            errs = append(errs, err)
        }
        if len(errs) > 0 {
            return errors.Join(errs...)
        }
        return nil
    }
    return shutdown, nil
}
```

### Pattern 2: Caller Integration — context.AfterFunc (server)

```go
// server/server.go — replace lines 106-149 with:
shutdown, err := observability.Init(ctx, "protoconf")
if err != nil {
    slog.Warn("OTel init failed, continuing without telemetry", "error", err)
}
context.AfterFunc(ctx, func() {
    if err := shutdown(ctx); err != nil {
        slog.Default().Error("error shutting down OTel providers", "error", err)
    }
})
```

### Pattern 3: Caller Integration — defer (agent)

```go
// agent/agent.go — replace lines 60-103 with:
shutdown, err := observability.Init(ctx, "protoconf")
if err != nil {
    slog.Warn("OTel init failed, continuing without telemetry", "error", err)
}
defer func() {
    if err := shutdown(ctx); err != nil {
        slog.Default().Error("error shutting down OTel providers", "error", err)
    }
}()
```

### Pattern 4: sync.Once Starlark Resolve Guard

```go
// compiler/lib/compiler.go
var initResolveOnce sync.Once

func initResolveSettings() {
    initResolveOnce.Do(func() {
        // AllowNestedDef, AllowLambda, AllowFloat are already no-ops in current
        // go.starlark.net version but set for forward compatibility documentation.
        resolve.AllowNestedDef = true
        resolve.AllowLambda = true
        resolve.AllowFloat = true
        resolve.AllowSet = true
        resolve.AllowGlobalReassign = true
        resolve.AllowRecursion = true
    })
}

func NewCompiler(protoconfRoot string, verboseLogging bool) (*Compiler, error) {
    initResolveSettings()
    // ... rest of constructor unchanged
}
```

### Pattern 5: Mutate conn Localization

```go
// mutate/mutate.go — remove line 30: var conn *grpc.ClientConn
// In Run():
conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
// (was: conn, err = grpc.NewClient(...))
```

The assignment becomes `:=` (short var decl), not `=` (assignment to package var). The `defer conn.Close()` on the next line stays unchanged.

### Anti-Patterns to Avoid

- **Returning nil shutdown function:** Callers must not nil-check before calling shutdown. Always return a valid no-op func on error.
- **Panicking on exporter failure:** The current `panic(err)` in both server and agent is the exact behavior being replaced by D-05/D-06.
- **Calling observe.Init in devserver:** devserver/command.go does NOT set up OTel (no OTel imports, plain `grpc.NewServer()`). Do not add OTel init to devserver in this phase.
- **Setting all 6 resolve flags in separate statements after sync.Once:** All six must be set inside the `sync.Once.Do` callback, not outside it.
- **Using `=` instead of `:=` for conn:** After removing the package var, the Run method must use `:=` to declare conn as a local variable.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Noop trace provider | Custom stub TracerProvider | `go.opentelemetry.io/otel/trace/noop.NewTracerProvider()` | Already in OTel v1.27.0; implements full interface |
| Noop meter provider | Custom stub MeterProvider | `go.opentelemetry.io/otel/metric/noop.NewMeterProvider()` | Already in OTel v1.27.0; implements full interface |
| One-time init guard | `bool` flag + mutex | `sync.Once` | Standard library; race-free by design |
| Combined shutdown | Manual error tracking in Init callers | Return single combined `func(ctx) error` from Init | Encapsulates both tracerProvider.Shutdown and meterProvider.Shutdown |

## Common Pitfalls

### Pitfall 1: Partial Noop Fallback
**What goes wrong:** Trace exporter succeeds, metric exporter fails. Code installs a real tracerProvider but a noop meterProvider. If shutdown is only built for one of them, the tracerProvider leaks goroutines.
**Why it happens:** Two-step exporter creation with early returns.
**How to avoid:** When metric exporter fails, still build a partial shutdown function that calls `tracerProvider.Shutdown()`. See Pattern 1 above for the exact structure.
**Warning signs:** Test exits with goroutine leaks when metric exporter is deliberately broken.

### Pitfall 2: Resolve Globals Outside sync.Once
**What goes wrong:** The `sync.Once` is declared but the flag assignments happen before or after the `Do` call, not inside it.
**Why it happens:** Developer restructures the guard without reading it carefully.
**How to avoid:** All six `resolve.*` assignments must be inside the `Do(func() { ... })` callback, never outside.
**Warning signs:** Running `go test -race ./compiler/lib/...` with concurrent NewCompiler calls fails.

### Pitfall 3: conn `:=` vs `=` After Package Var Removal
**What goes wrong:** After deleting `var conn *grpc.ClientConn` at package scope, the `conn, err = grpc.NewClient(...)` line still uses `=` assignment, causing a compile error ("undefined: conn").
**Why it happens:** The line was originally assigning to the package-level var.
**How to avoid:** Change `conn, err =` to `conn, err :=` (or declare `var conn *grpc.ClientConn` locally before the assignment).
**Warning signs:** Compile error `undefined: conn` in mutate package.

### Pitfall 4: Import Cycles
**What goes wrong:** `observability/` imports `consts/` for `consts.Version`. Both `server/` and `agent/` import `observability/`. If `observability/` were to import `server/` or `agent/`, a cycle would form.
**Why it happens:** Accidental dependency direction.
**How to avoid:** `observability/` must only import `consts/` and OTel packages. Verified: `consts/consts.go` has no external imports, no cycle risk.
**Warning signs:** `go build ./...` reports "import cycle not allowed".

### Pitfall 5: devserver Incorrectly Receiving OTel Init
**What goes wrong:** Planner adds `observability.Init` to `devserver/command.go` by analogy with server and agent, but devserver has no OTel at all currently.
**Why it happens:** devserver "combines agent and server" so one might assume it needs both.
**How to avoid:** devserver/command.go has zero OTel imports and uses plain `grpc.NewServer()`. Adding OTel init is not in scope for this phase.
**Warning signs:** Unexpected import additions to devserver in a phase 3 plan.

## Code Examples

### Noop Providers (verified via go doc)

```go
// Source: go doc go.opentelemetry.io/otel/trace/noop
import tracenoop "go.opentelemetry.io/otel/trace/noop"
otel.SetTracerProvider(tracenoop.NewTracerProvider())

// Source: go doc go.opentelemetry.io/otel/metric/noop
import metricnoop "go.opentelemetry.io/otel/metric/noop"
otel.SetMeterProvider(metricnoop.NewMeterProvider())
```

### sync.Once Pattern (standard library)

```go
var resolveOnce sync.Once

func initResolveSettings() {
    resolveOnce.Do(func() {
        resolve.AllowNestedDef = true
        resolve.AllowLambda = true
        resolve.AllowFloat = true
        resolve.AllowSet = true
        resolve.AllowGlobalReassign = true
        resolve.AllowRecursion = true
    })
}
```

### Existing resource.New call (confirmed identical in both files)

```go
// Source: server/server.go:111-121 and agent/agent.go:65-75 (verified — identical)
resources, _ := resource.New(ctx,
    resource.WithFromEnv(),
    resource.WithProcess(),
    resource.WithOS(),
    resource.WithContainer(),
    resource.WithHost(),
    resource.WithAttributes(
        semconv.ServiceNameKey.String(serviceName),
        semconv.ServiceVersionKey.String(consts.Version),
    ),
)
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `panic(err)` on OTel exporter failure | Log warning + noop fallback | This phase | Process survives collector outage |
| Duplicate OTel bootstrap in each binary | Shared `observability.Init` | This phase | Single source of truth |
| `resolve.*` set on every NewCompiler call | Set once via `sync.Once` | This phase | No data race on concurrent construction |
| Package-level `var conn *grpc.ClientConn` | Local var in `Run()` | This phase | Eliminates accidental state sharing |

**Deprecated/outdated in go.starlark.net resolve:**
- `AllowNestedDef`, `AllowLambda`, `AllowFloat`: Now no-ops (features became standard). Still safe to set (ignored), but they do nothing.

## Open Questions

1. **serviceName parameter usage**
   - What we know: Both callers currently hardcode `"protoconf"` as the service name
   - What's unclear: Whether the Init signature should accept serviceName as a param (D-02) or use a constant
   - Recommendation: Honor D-02 (pass serviceName as param). Both callers pass `"protoconf"` explicitly, which is self-documenting.

2. **otelkv.go per-package tracer**
   - What we know: `agent/otelkv/otelkv.go` uses `otel.Tracer("valkeyrie")` — a per-package tracer acquired from the global provider, not a bootstrap init
   - What's unclear: Whether Claude's Discretion means "leave it alone" or "optionally refactor"
   - Recommendation: Leave as-is. It relies on whatever global provider was set by the bootstrap (which will now be set by observability.Init). No change needed.

## Environment Availability

Step 2.6: SKIPPED (no external dependencies — all changes are pure Go code refactoring within existing packages)

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | testify v1.9.0 + Go stdlib testing |
| Config file | none (no pytest.ini / jest.config equivalent for Go) |
| Quick run command | `go test ./observability/... ./compiler/lib/... ./mutate/...` |
| Full suite command | `go test ./... -timeout 120s` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| REFC-05 | `observability.Init` returns a shutdown func with no panic | unit | `go test ./observability/... -run TestInit` | Wave 0 — new file |
| REFC-06 | OTel init failure (unavailable collector) returns error + sets noop providers | unit | `go test ./observability/... -run TestInit_NoopFallback` | Wave 0 — new file |
| REFC-07 | Multiple concurrent `NewCompiler` calls do not race on resolve.* | unit+race | `go test -race ./compiler/lib/... -run TestNewCompiler` | ✅ compiler_test.go exists |
| REFC-08 | `mutate` package has no package-level `conn` var | compile | `go build ./mutate/...` | ✅ mutate.go exists |

### Sampling Rate
- **Per task commit:** `go test ./observability/... ./compiler/lib/... ./mutate/...`
- **Per wave merge:** `go test ./...`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `observability/observability.go` — new package, covers REFC-05 and REFC-06
- [ ] `observability/observability_test.go` — covers REFC-05, REFC-06 (if Claude's Discretion selects "add tests in this phase")

Note: REFC-07 and REFC-08 are verified by the existing compiler_test.go and a `go build` check respectively. No new test files needed for those two requirements.

## Sources

### Primary (HIGH confidence)
- `server/server.go` lines 106–149 — direct code inspection of OTel init block to extract
- `agent/agent.go` lines 60–103 — direct code inspection, confirmed identical pattern
- `compiler/lib/compiler.go` lines 35–41 — resolve flags confirmed in constructor
- `mutate/mutate.go` line 30 — package-level var confirmed
- `go doc go.opentelemetry.io/otel/trace/noop` — NewTracerProvider confirmed available in v1.27.0
- `go doc go.opentelemetry.io/otel/metric/noop` — NewMeterProvider confirmed available in v1.27.0
- `go doc go.starlark.net/resolve` — obsolete flag list confirmed for current version
- `go.mod` — all OTel version numbers and starlark version confirmed

### Secondary (MEDIUM confidence)
- `devserver/command.go` — confirmed no OTel imports, plain grpc.NewServer

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all packages already in go.mod, versions confirmed
- Architecture: HIGH — code directly inspected, patterns confirmed in codebase
- Pitfalls: HIGH — derived directly from current code structure and Go semantics

**Research date:** 2026-03-27
**Valid until:** 2026-06-27 (stable OTel API, stable Go stdlib sync.Once, stable starlark resolve)
