---
phase: 10
plan: 02
subsystem: test
tags: [e2e, integration-tests, tls, auth, mutation-scripts]
dependency_graph:
  requires: []
  provides: [TEST-11, TEST-12, TEST-13]
  affects: [test/e2e_test.go]
tech_stack:
  added: []
  patterns:
    - Self-signed ECDSA P-256 cert for in-test TLS setup
    - Real TCP listener for TLS e2e (bufconn cannot carry TLS)
    - makeTokenInterceptor mirrors production bearerTokenInterceptor behavior
    - mustNewAny/makeTempScript helpers local to test package
key_files:
  created: []
  modified:
    - test/e2e_test.go
decisions:
  - Real TCP listener (not bufconn) required for TLS tests — TLS requires proper hostname/IP verification
  - makeTokenInterceptor duplicates server.bearerTokenInterceptor logic — unexported, cannot reuse directly
  - subtle.ConstantTimeCompare used in makeTokenInterceptor to mirror production timing-safe comparison
metrics:
  duration_seconds: 123
  completed_date: "2026-03-31"
  tasks_completed: 2
  files_modified: 1
---

# Phase 10 Plan 02: E2E Integration Tests (Mutation Scripts, TLS, Auth) Summary

**One-liner:** Three e2e tests covering mutation with pre/post scripts, TLS gRPC connections, and token-based auth reject/accept flow.

## What Was Built

Added three new test functions to `test/e2e_test.go` along with supporting helpers:

### Helpers Added

- **`makeTempScript(t, body)`** — Creates a temp executable shell script; prevents dependency on unexported `server.makeTempScript`
- **`mustNewAny(msg)`** — Wraps a `proto.Message` in `anypb.Any` for concise test fixtures
- **`generateSelfSignedCert(t)`** — Generates ECDSA P-256 self-signed cert for 127.0.0.1 (duplicated from `utils/tls_test.go` since unexported)
- **`makeTokenInterceptor(token)`** — Creates a gRPC unary interceptor mirroring `bearerTokenInterceptor` in `server/server.go`

### Tests Added

**`TestMutationWithScripts` (TEST-11)**
- Creates a `ProtoconfMutationServer` with `PreMutationScript` and `PostMutationScript` set
- Calls `MutateConfig` over gRPC via `TestServer` (bufconn)
- Asserts `resp.Uuid` is non-empty and both `PreScriptDuration`/`PostScriptDuration` are non-nil

**`TestTLSMutation` (TEST-12)**
- Uses a real TCP listener on `127.0.0.1:0` (not bufconn — TLS requires real transport)
- Builds server TLS config via `utils.BuildTLSConfig` with PEM text inputs
- Client connects with `credentials.NewTLS` and `ServerName: "127.0.0.1"` to match the SAN
- Asserts `MutateConfig` succeeds and returns a non-empty UUID

**`TestAuthFlow` (TEST-13)**
- Creates a gRPC server with `makeTokenInterceptor` wrapping an expected secret token
- Three sub-tests:
  - `valid_token_accepted`: sends correct `Bearer` token, expects success
  - `invalid_token_rejected`: sends wrong token, expects `codes.Unauthenticated`
  - `missing_token_rejected`: sends no auth header, expects `codes.Unauthenticated`

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None.

## Self-Check: PASSED

- `test/e2e_test.go` exists and compiles: confirmed
- Commit `24a9e5e` (Task 1): confirmed
- Commit `3687498` (Task 2): confirmed
- All three tests pass: `TestMutationWithScripts`, `TestTLSMutation`, `TestAuthFlow`
