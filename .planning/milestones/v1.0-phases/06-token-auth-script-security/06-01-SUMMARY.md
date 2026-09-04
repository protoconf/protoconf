---
phase: "06"
plan: "01"
subsystem: server
tags: [auth, security, grpc, interceptor]
dependency_graph:
  requires: []
  provides: [bearerTokenInterceptor, auth-token-flag]
  affects: [server/server.go, server/server_test.go]
tech_stack:
  added: [crypto/subtle, google.golang.org/grpc/codes, google.golang.org/grpc/status]
  patterns: [grpc-unary-interceptor, constant-time-compare, table-driven-tests]
key_files:
  created: []
  modified:
    - server/server.go
    - server/server_test.go
decisions:
  - "Use crypto/subtle.ConstantTimeCompare to prevent timing attacks on token comparison"
  - "Pass-through when authToken is empty for backward compatibility"
  - "Log startup warning when auth is not configured, matching TLS warning format"
metrics:
  duration_minutes: 8
  completed_date: "2026-03-28T15:57:49Z"
  tasks_completed: 1
  files_modified: 2
---

# Phase 6 Plan 1: Token-Based Auth Interceptor Summary

Bearer token authentication added to the mutation server via a gRPC unary interceptor using crypto/subtle for timing-safe comparison and pass-through when unconfigured.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Add auth interceptor and --auth-token flag | a0e6606 | server/server.go, server/server_test.go |

## What Was Built

### server/server.go

- Added `authToken string` field to `cliConfig` struct
- Registered `--auth-token` flag in `newFlagSet()` with empty default (auth disabled)
- Implemented `bearerTokenInterceptor(expectedToken string) grpc.UnaryServerInterceptor`:
  - Pass-through when `expectedToken` is empty (backward compatible)
  - Returns `codes.Unauthenticated "missing metadata"` when context has no gRPC metadata
  - Returns `codes.Unauthenticated "missing authorization header"` when authorization header absent
  - Strips `"Bearer "` prefix via `strings.TrimPrefix` then compares with `crypto/subtle.ConstantTimeCompare`
  - Returns `codes.Unauthenticated "invalid token"` on mismatch
- Wired interceptor via `grpc.ChainUnaryInterceptor(bearerTokenInterceptor(config.authToken))` in `run()`
- Logs startup warning when auth is not configured: `"mutation server running without authentication -- requests are not authenticated"`

### server/server_test.go

- Added imports: `google.golang.org/grpc/codes`, `google.golang.org/grpc/status`
- Added `Test_bearerTokenInterceptor` with 6 table-driven subtests:
  - `no_auth_configured`: empty expectedToken, passes through, codes.OK
  - `valid_token`: "Bearer secret123" in authorization header, codes.OK
  - `missing_metadata`: context.Background() with no metadata, codes.Unauthenticated
  - `missing_authorization_header`: metadata present but no authorization key, codes.Unauthenticated
  - `invalid_token`: "Bearer wrongtoken" in header, codes.Unauthenticated
  - `raw_token_without_bearer_prefix`: "secret123" (no "Bearer " prefix), codes.OK (TrimPrefix leaves it unchanged, matches expectedToken)

## Verification Results

- `go test ./server/ -run Test_bearerTokenInterceptor -v` — all 6 subtests PASS
- `go test ./server/ -v` — all existing tests continue to PASS
- `go build ./...` — full project builds with 0 errors
- `grep -c "bearerTokenInterceptor" server/server.go` — returns 3 (definition + 2 usages: wiring + warn check)

## Deviations from Plan

None - plan executed exactly as written.

## Known Stubs

None - all data flows are real.

## Self-Check: PASSED

- server/server.go exists with bearerTokenInterceptor: FOUND
- server/server_test.go exists with Test_bearerTokenInterceptor: FOUND
- Commit a0e6606 exists: FOUND
