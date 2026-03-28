---
phase: 05-tls-support
plan: "02"
subsystem: infra
tags: [tls, grpc, credentials, agent, server, mutate]

requires:
  - phase: 05-01
    provides: "utils.BuildTLSConfig helper and TLSFiles struct"

provides:
  - "Agent gRPC server wired to BuildTLSConfig via proto-generated TlsConfig fields"
  - "Mutation server grpc.NewServer wired to BuildTLSConfig via --tls-cert/--tls-key/--tls-ca flags"
  - "Mutate CLI grpc.NewClient wired to BuildTLSConfig via --tls-cert/--tls-key/--tls-ca flags"
  - "Insecure-mode slog.Warn in both agent and mutation server"

affects: [06-auth, 09-unit-tests, 10-integration-tests]

tech-stack:
  added: []
  patterns:
    - "Dynamic grpc.ServerOption slice built before grpc.NewServer so TLS credentials can be conditionally appended"
    - "TLS wiring guarded by nil-check on tlsCfg so BuildTLSConfig returning nil,nil means insecure path"

key-files:
  created: []
  modified:
    - agent/agent.go
    - server/server.go
    - mutate/mutate.go

key-decisions:
  - "GenReflectionUI bufconn stays insecure.NewCredentials() — it is an in-process connection, not a network connection"
  - "mutate CLI insecure flag named insecureTLS to avoid collision with existing field names"
  - "When --insecure not set and no TLS flags set, mutate CLI still uses insecure.NewCredentials() to preserve backward compatibility"

patterns-established:
  - "Collect grpc.ServerOption slice, conditionally append grpc.Creds, then pass slice to grpc.NewServer"
  - "Log slog.Warn when no TLS config present on any gRPC server entry point"

requirements-completed:
  - SECR-01
  - SECR-02
  - SECR-03

duration: 8min
completed: 2026-03-27
---

# Phase 05 Plan 02: TLS Wiring Summary

**gRPC TLS credentials wired into agent server (via proto config), mutation server, and mutate CLI (via new --tls-cert/--tls-key/--tls-ca flags), with slog.Warn on insecure startup**

## Performance

- **Duration:** ~8 min
- **Started:** 2026-03-27T00:00:00Z
- **Completed:** 2026-03-27T00:08:00Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments

- Agent reads TLS credentials from existing proto-generated TlsConfig fields and passes them to grpc.NewServer via credentials.NewTLS
- Mutation server accepts --tls-cert/--tls-key/--tls-ca CLI flags, builds TLS config, and applies it to grpc.NewServer
- Mutate CLI accepts --tls-cert/--tls-key/--tls-ca/--insecure flags, builds TLS config, and applies it to grpc.NewClient
- Both agent and mutation server log slog.Warn("gRPC server running without TLS -- connections are not encrypted") when TLS is not configured

## Task Commits

1. **Task 1: Wire TLS into agent gRPC server with insecure warning** - `8bec6cd` (feat)
2. **Task 2: Add TLS flags and wiring to mutation server and mutate CLI** - `afe8343` (feat)

## Files Created/Modified

- `agent/agent.go` - Added utils/credentials imports; dynamic serverOpts slice with conditional TLS; slog.Warn on insecure path
- `server/server.go` - Added utils/credentials imports; tlsCert/tlsKey/tlsCA to cliConfig; three new flags; dynamic serverOpts with BuildTLSConfig; slog.Warn on insecure path
- `mutate/mutate.go` - Added crypto/tls/utils/credentials imports; tlsCert/tlsKey/tlsCA/insecureTLS to cliConfig; four new flags; conditional dialOpt using BuildTLSConfig

## Decisions Made

- GenReflectionUI bufconn connection stays `insecure.NewCredentials()` — this is an in-process loopback connection (bufconn), not a network-facing socket, so TLS adds no security value there.
- `insecureTLS` field name in mutate's cliConfig avoids collision with any hypothetical `insecure` field.
- When no TLS flags are provided in mutate CLI, it falls back to `insecure.NewCredentials()` for backward compatibility with existing deployments.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

Pre-existing vet warnings in `agent/filekv/filekv.go` (mutex passed by value) and `agent/legacy.go` (unreachable code) are out of scope. These were present before this plan and are not caused by the current changes.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- SECR-01, SECR-02, SECR-03 complete — TLS is available on all three gRPC entry points.
- Phase 06 (auth/token forwarding) can now build on a TLS-secured transport layer.

---
*Phase: 05-tls-support*
*Completed: 2026-03-27*
