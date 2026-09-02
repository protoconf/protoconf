---
phase: quick-260902-ub9
plan: 01
subsystem: observability
tags: [opentelemetry, otlp, grpc, cli-flags, protobuf]

requires: []
provides:
  - "enable_otel bool field on AgentConfig (field 17) and ServerConfig (field 8)"
  - "observability.Init(ctx, serviceName, enabled) explicit opt-in gate"
  - "-enable-otel / PROTOCONF_AGENT_ENABLE_OTEL / PROTOCONF_SERVER_ENABLE_OTEL CLI flag+env pair"
  - "conditional otelgrpc stats handler on both the agent and mutation server gRPC servers"
affects: [agent, server, observability]

actuals:
  tokens: 9500
  tasks: 4
  commits: 5

tech-stack:
  added: []
  patterns:
    - "Proto3 zero-value-as-default: no defaulting code, an unset bool field IS the disabled state"
    - "Single gate inside the shared function (observability.Init) rather than duplicated in each caller"

key-files:
  created:
    - observability/observability_test.go
  modified:
    - agent/config/v1/agent_config.proto
    - agent/config/v1/agent_config.pb.go
    - server/config/v1/server_config.proto
    - server/config/v1/server_config.pb.go
    - observability/observability.go
    - agent/agent.go
    - agent/command.go
    - agent/command_test.go
    - server/server.go
    - server/server_test.go

key-decisions:
  - "D-06: otelkv.New tracing wrapper left ungated -- under noop providers it is a nearly-free no-op span, and threading a bool into the store wrapper buys nothing measurable"
  - "D-05: standard OpenTelemetry SDK env vars (OTEL_SDK_DISABLED etc.) intentionally not consulted anywhere -- the proto-derived flag/env pair is the single control; honoring the SDK env var, if ever wanted, is a separate future change with its own precedence rules"
  - "D-04: disabled path omits the grpc.ServerOption entirely rather than substituting a no-op stats handler -- removes the per-RPC allocation instead of shrinking it, and avoids porting upstream PR #496's noop_otel_handler.go"

requirements-completed: [QUICK-260902-ub9]

coverage:
  - id: D1
    description: "Agent and mutation server default to OTel disabled (no exporter, no OTLP dial, noop providers, no otelgrpc stats handler) unless -enable-otel/env is set"
    requirement: "QUICK-260902-ub9"
    verification:
      - kind: unit
        ref: "observability/observability_test.go#TestInit_disabled_installsNoopProviders"
        status: pass
      - kind: unit
        ref: "observability/observability_test.go#TestInit_enabled_installsSDKProviders"
        status: pass
      - kind: unit
        ref: "agent/command_test.go#Test_cliCommand_EnableOtelFlag"
        status: pass
      - kind: unit
        ref: "server/server_test.go#Test_cliCommand_EnableOtelFlag"
        status: pass
    human_judgment: false
  - id: D2
    description: "protoconf agent -h and protoconf serve -h document -enable-otel with off-by-default usage text"
    requirement: "QUICK-260902-ub9"
    verification:
      - kind: other
        ref: "built binary: `protoconf-verify agent -h` / `protoconf-verify serve -h`, grepped for 'enable-otel' and 'off by default'"
        status: pass
    human_judgment: false

duration: ~35min
completed: 2026-09-02
status: complete
---

# Quick Task 260902-ub9: Make OpenTelemetry opt-in Summary

**Added a single `enable_otel` proto field per config (agent, server), threaded through `observability.Init`'s new `enabled` parameter, so neither binary builds an OTLP exporter or installs the otelgrpc gRPC stats handler unless `-enable-otel` (or its `PROTOCONF_{AGENT,SERVER}_ENABLE_OTEL` env var) is set.**

## Performance

- **Duration:** ~35 min
- **Tasks:** 4 (Task 2 was TDD: RED + GREEN, no REFACTOR needed)
- **Files modified:** 10 (1 new test file, 4 hand-written source files, 2 proto files, 2 regenerated `.pb.go`, 1 more hand-written test file)

## Accomplishments
- `AgentConfig.enable_otel` (field 17) and `ServerConfig.enable_otel` (field 8) added and regenerated with `protoc-gen-go` v1.36.11, both proto3 bool zero-value-defaulted to `false` -- no hand-written flag registration or defaulting code needed, `libprotoconf` derives the CLI flag and env var straight from the field.
- `observability.Init` gained a third `enabled bool` parameter (D-01); the disabled branch runs before any `resource.New` detection work and installs noop trace/meter providers explicitly (D-03), returning a nil error and a working no-op shutdown.
- Both `RunAgent` and the mutation server's `cliCommand.run` now pass their config's `EnableOtel` through to `observability.Init`, and both `serverOpts` slices only append `grpc.StatsHandler(otelgrpc.NewServerHandler())` when that flag is true (D-04) -- confirmed by a whole-repo grep showing exactly two call sites of each, both gated.
- `protoconf agent -h` and `protoconf serve -h` both document `-enable-otel` with usage text stating it is off by default and that no collector is contacted when unset; the server's `Command()` factory gained its first `flag.VisitAll` usage-override block (it had none before).
- New tests: `observability/observability_test.go` (provider-type-identity assertions for both the disabled and enabled paths, restoring global OTel providers via `t.Cleanup`), `agent/command_test.go#Test_cliCommand_EnableOtelFlag`, `server/server_test.go#Test_cliCommand_EnableOtelFlag` (flag-plumbing default-false / bare-flag-true pairs).

## Task Commits

Each task was committed atomically:

1. **Task 1: Add the enable_otel field to both config protos and regenerate** - `d40fe55` (feat)
2. **Task 2: Gate observability.Init on an explicit enabled parameter** - `7a69371` (test, RED) + `42a4196` (feat, GREEN; no refactor commit needed)
3. **Task 3: Wire the agent** - `a5fe0b0` (feat)
4. **Task 4: Wire the mutation server** - `d40e9c7` (feat)

_Docs/state metadata commit made separately by the orchestrator, per constraints._

## Files Created/Modified
- `agent/config/v1/agent_config.proto` - added `bool enable_otel = 17`
- `agent/config/v1/agent_config.pb.go` - regenerated (protoc-gen-go v1.28.1 -> v1.36.11, large mechanical diff as expected)
- `server/config/v1/server_config.proto` - added `bool enable_otel = 8`
- `server/config/v1/server_config.pb.go` - regenerated (already on v1.36.11, small diff)
- `observability/observability.go` - `Init` signature gains `enabled bool`; disabled branch installs noop providers before any resource/exporter work
- `observability/observability_test.go` - new; provider-identity tests for both branches
- `agent/agent.go` - `observability.Init(ctx, "protoconf", config.EnableOtel)`; `otelgrpc.NewServerHandler()` moved out of the `serverOpts` literal into a conditional append
- `agent/command.go` - `enable-otel` case added to the existing `VisitAll` usage-override switch
- `agent/command_test.go` - new `Test_cliCommand_EnableOtelFlag`
- `server/server.go` - `observability.Init(ctx, "protoconf", c.config.EnableOtel)`; `serverOpts` now built empty with a conditional append; first `flag.VisitAll` block added to `Command()`
- `server/server_test.go` - new `Test_cliCommand_EnableOtelFlag`

## Decisions Made
- **D-06 (recorded per plan's `<output>` instruction):** `agent/agent.go`'s two `otelkv.New(ctx, store)` call sites (lines 119, 121) were left ungated. They wrap the KV store with tracing spans off the *global* tracer; once the global tracer is a noop provider (the disabled-by-default case), the span creation is a near-free no-op. Threading a bool into the store-wrapper constructor for that would add plumbing for no measurable gain. No code changed there.
- **D-05 (recorded per plan's `<output>` instruction):** No second control mechanism was added. Standard OpenTelemetry SDK environment variables (`OTEL_SDK_DISABLED`, etc.) are not consulted anywhere in this change or anywhere else in the repo -- confirmed by the whole-repo grep in the verification section below finding only the two proto-gated call sites. If operators later want the standard SDK env var honored, that is explicitly flagged as a **future, separate change** that must define precedence between it and `-enable-otel` deliberately; it was not built here.
- **D-07:** `slogotel.OtelHandler` in `agent/agent.go` also left untouched -- it reads span context from the global tracer and is a pass-through when there is no active span (noop-provider case).
- **D-08:** `grpc_prometheus` interceptors/handler untouched -- out of scope (Prometheus, not OpenTelemetry).

## Deviations from Plan

None affecting the shipped code. One process note:

**Task 3's literal `<verify>` block includes `go build ./...`, which cannot pass until Task 4 also updates `server/server.go`'s `observability.Init` call site** (Task 2's signature change breaks both callers simultaneously; Task 3 only fixes the agent's). This is consistent with the plan's own "Full sweep" note ("run once after Task 4 is committed"), so it was treated as expected sequencing rather than a defect: Task 3 was verified with the narrower checks that were actually in its own scope (`go build ./agent/...`, `go vet ./agent/...`, `go test ./agent/...`), and the full sweep (including both `-h` outputs) was run and confirmed passing after Task 4 landed.

## Pre-existing go vet findings (outside files_modified, per plan instruction: record and leave alone)

`go vet ./...` after Task 4 reported, all pre-existing and unrelated to this change:
- `compiler/lib/compiler.go:355:20` - literal copies lock value from `MessageRegistry` (contains `sync.RWMutex`)
- `agent/agent_test.go:27:11` - `context.WithTimeoutCause` cancel func discarded
- `agent/legacy.go:97:2` - unreachable code
- `test/e2e_test.go:327:8`, `test/e2e_test.go:411:11` - `context.WithTimeout` cancel func discarded

## Issues Encountered
None. Toolchain (`protoc`, `protoc-gen-go`, `protoc-gen-go-grpc`, `buf`) worked as documented; regeneration of the two named proto files only, with no drag-in of the four unrelated `.pb.go` files still on protoc-gen-go v1.28.1, was confirmed via `git status --short` after each regeneration step.

## Verification Sweep (run after Task 4, per plan's `<verification>` block)

```
go build ./...                                                          -> pass, no output
go vet ./...                                                             -> pre-existing findings only, listed above
go test ./observability/... ./agent/... ./server/... -count=1 -skip Test_cliCommand_Run  -> all ok
buf build -o /dev/null                                                   -> pass
buf breaking --against '.git#branch=main'                                -> pass
```

Manual sanity check (exactly two call sites of each, both gated):
```
$ grep -rn --include="*.go" -e 'observability\.Init' -e 'otelgrpc\.NewServerHandler' .
server/server.go:114:	shutdown, err := observability.Init(ctx, "protoconf", c.config.EnableOtel)
server/server.go:147:		serverOpts = append(serverOpts, grpc.StatsHandler(otelgrpc.NewServerHandler()))
agent/agent.go:59:	shutdown, otelErr := observability.Init(ctx, "protoconf", config.EnableOtel)
agent/agent.go:138:		serverOpts = append(serverOpts, grpc.StatsHandler(otelgrpc.NewServerHandler()))
```

Operator-experience sanity check -- built the binary and ran `-h` on both subcommands:
```
$ ./protoconf agent -h | grep -B1 -A2 enable-otel
  -enable-otel
    	Export OpenTelemetry traces and metrics to an OTLP/gRPC collector. Off by default -- no collector is contacted when unset
    	[env: PROTOCONF_AGENT_ENABLE_OTEL] (default false)

$ ./protoconf serve -h | grep -B1 -A2 enable-otel
  -enable-otel
    	Export OpenTelemetry traces and metrics to an OTLP/gRPC collector. Off by default -- no collector is contacted when unset
    	[env: PROTOCONF_SERVER_ENABLE_OTEL] (default false)
```

## User Setup Required
None - no external service configuration required. Operators who want telemetry restored pass `-enable-otel` or set `PROTOCONF_AGENT_ENABLE_OTEL=true` / `PROTOCONF_SERVER_ENABLE_OTEL=true`.

## Next Phase Readiness
- Both binaries are OTel-opt-in with no code path left that unconditionally builds an exporter.
- No blockers. If the standard `OTEL_SDK_DISABLED`/`OTEL_SDK_ENABLED` env vars are later wanted, that is a distinct future change that must define precedence against `-enable-otel` explicitly (D-05).

---
*Quick task: 260902-ub9*
*Completed: 2026-09-02*
