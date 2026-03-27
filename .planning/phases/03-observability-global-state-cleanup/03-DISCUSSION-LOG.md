# Phase 3: Observability & Global State Cleanup - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-03-27
**Phase:** 03-observability-global-state-cleanup
**Areas discussed:** Shared OTel package design, OTel failure handling, Starlark resolve initialization, Mutate global state cleanup
**Mode:** --auto (all decisions auto-selected with recommended defaults)

---

## Shared OTel Package Design

### Package Location

| Option | Description | Selected |
|--------|-------------|----------|
| New `observability/` top-level package | Consistent with consts/, command/ pattern | ✓ |
| New `pkg/otel/` nested package | Common in larger Go projects | |
| Inline in `command/` package | Reuse existing shared package | |

**User's choice:** [auto] New `observability/` top-level package (recommended default)
**Notes:** Top-level packages are the established pattern in this codebase

### Package API

| Option | Description | Selected |
|--------|-------------|----------|
| Single `Init(ctx, serviceName) (shutdown, error)` | Minimal API, callers handle shutdown | ✓ |
| Struct-based `Provider` with methods | More configurable, heavier | |
| Functional options pattern | Flexible but more complex | |

**User's choice:** [auto] Single Init function (recommended default)
**Notes:** Minimal API preferred for a utility that's called once per process

### Shutdown Pattern

| Option | Description | Selected |
|--------|-------------|----------|
| Return shutdown function, caller chooses | Preserves existing patterns | ✓ |
| Standardize on context.AfterFunc | Forces server pattern on agent | |
| Standardize on defer | Forces agent pattern on server | |

**User's choice:** [auto] Return shutdown function (recommended default)
**Notes:** Both patterns are valid; no need to force consistency

---

## OTel Failure Handling

| Option | Description | Selected |
|--------|-------------|----------|
| Log warning, continue with no-op providers | Graceful degradation | ✓ |
| Return error, let caller decide | More control but inconsistent handling risk | |
| Panic (current behavior) | Ensures telemetry or nothing | |

**User's choice:** [auto] Log warning, continue with no-op (recommended default)
**Notes:** Matches REFC-06 requirement. Process should never crash because OTel collector is down.

---

## Starlark Resolve Initialization

| Option | Description | Selected |
|--------|-------------|----------|
| sync.Once guard in compiler/lib | Idempotent, race-safe | ✓ |
| Package-level init() function | Simpler but always runs even if compiler unused | |
| Move to cmd/protoconf/main.go | Centralizes but spreads compiler knowledge | |

**User's choice:** [auto] sync.Once guard (recommended default)
**Notes:** Eliminates redundant writes and data race potential on concurrent NewCompiler calls

---

## Mutate Global State Cleanup

### grpc.ClientConn

| Option | Description | Selected |
|--------|-------------|----------|
| Move to local var in Run() | Direct fix, matches REFC-08 | ✓ |
| Pass as parameter to Run() | More testable but changes signature | |

**User's choice:** [auto] Move to local var in Run() (recommended default)
**Notes:** Minimal change, directly addresses the requirement

### Global ui var

| Option | Description | Selected |
|--------|-------------|----------|
| Leave as-is | CLI presentation, not connection state | ✓ |
| Move to Run() | Consistent with conn cleanup | |

**User's choice:** [auto] Leave as-is (recommended default)
**Notes:** Not mutable connection state, minimal change principle

---

## Claude's Discretion

- Exact internal structure of observability package
- Whether to use functional options for Init
- otelkv.go tracer handling
- Unit tests for new package (this phase vs Phase 9)

## Deferred Ideas

None — discussion stayed within phase scope
