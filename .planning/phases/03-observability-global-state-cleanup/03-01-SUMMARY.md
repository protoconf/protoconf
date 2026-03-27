---
phase: 03-observability-global-state-cleanup
plan: 01
subsystem: observability
tags: [otel, refactoring, shared-package, no-panic]
dependency_graph:
  requires: []
  provides: [observability/observability.go]
  affects: [server/server.go, agent/agent.go]
tech_stack:
  added:
    - observability package (new internal package)
  patterns:
    - Noop provider fallback pattern for OTel initialization
    - Combined shutdown via errors.Join
key_files:
  created:
    - observability/observability.go
  modified:
    - server/server.go
    - agent/agent.go
decisions:
  - "observability.Init returns (shutdown, error) to let callers choose shutdown strategy (AfterFunc vs defer)"
  - "noop providers installed on exporter failure so OTel instrumentation downstream never panics"
  - "Init always returns non-nil shutdown function for safe deferred calls"
metrics:
  duration_seconds: 420
  completed_date: "2026-03-27"
  tasks_completed: 2
  files_modified: 3
---

# Phase 03 Plan 01: Shared OTel Bootstrap Package Summary

**One-liner:** Extracted duplicated OpenTelemetry bootstrap into `observability.Init` with noop provider fallback, eliminating panics on exporter unavailability.

## What Was Built

Created `observability/observability.go` — a new internal package exporting a single `Init(ctx, serviceName)` function that:

1. Builds a `resource.Resource` with standard detectors (env, process, OS, container, host) and service name/version attributes
2. Initializes OTLP gRPC trace exporter; on failure installs noop trace+metric providers and returns early with error
3. Initializes OTLP gRPC metric exporter; on failure installs noop meter provider and returns partial shutdown
4. Returns a combined shutdown function using `errors.Join` for both providers

Updated `server/server.go` and `agent/agent.go` to call `observability.Init` instead of the 44-line duplicated OTel setup block. Both removed 6 OTel SDK imports no longer needed.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Create observability package with Init function and noop fallback | 3b1e9ca | observability/observability.go |
| 2 | Update server/server.go and agent/agent.go to use observability.Init | 0e59feb | server/server.go, agent/agent.go |

## Verification Results

- `go build ./observability/... ./server/... ./agent/...` — all pass
- `go test ./server/... ./agent/... -timeout 60s` — all pass
- `grep -r "panic(err)" server/server.go agent/agent.go` — no matches (panics eliminated)
- `grep -r "observability.Init" server/server.go agent/agent.go` — 2 matches
- `grep -r "otlptracegrpc.New" server/server.go agent/agent.go` — no matches (fully extracted)

## Deviations from Plan

None - plan executed exactly as written.

## Known Stubs

None.

## Self-Check: PASSED

- observability/observability.go: FOUND
- server/server.go updated: FOUND (`observability.Init` call present)
- agent/agent.go updated: FOUND (`observability.Init` call present)
- Commits 3b1e9ca and 0e59feb: FOUND in git log
