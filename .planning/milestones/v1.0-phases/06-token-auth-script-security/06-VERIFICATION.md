---
phase: 06-token-auth-script-security
verified: 2026-03-28T00:00:00Z
status: passed
score: 12/12 must-haves verified
re_verification: false
human_verification:
  - test: "Bearer token rejection end-to-end via live gRPC client"
    expected: "A mutation client without Authorization header receives gRPC status Unauthenticated when --auth-token is set"
    why_human: "End-to-end test requires a running server; behavioral spot-check was done at unit test level only"
---

# Phase 6: Token Auth & Script Security Verification Report

**Phase Goal:** Mutation server enforces token-based auth; credentials reach pre/post scripts; script paths are validated
**Verified:** 2026-03-28
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | A mutation request with a valid Bearer token succeeds when auth is configured | VERIFIED | `Test_bearerTokenInterceptor/valid_token` passes; interceptor returns handler result for matching token |
| 2 | A mutation request with no token is rejected with codes.Unauthenticated when auth is configured | VERIFIED | `Test_bearerTokenInterceptor/missing_metadata` and `missing_authorization_header` pass with codes.Unauthenticated |
| 3 | A mutation request with a wrong token is rejected with codes.Unauthenticated | VERIFIED | `Test_bearerTokenInterceptor/invalid_token` passes with codes.Unauthenticated |
| 4 | All requests pass through when --auth-token is not set (backward compatible) | VERIFIED | `Test_bearerTokenInterceptor/no_auth_configured` passes; interceptor returns handler(ctx, req) when expectedToken=="" |
| 5 | Server logs a warning at startup when auth is not configured | VERIFIED | `server/server.go` line 185: `slog.Warn("mutation server running without authentication -- requests are not authenticated")` |
| 6 | Pre/post scripts receive PROTOCONF_AUTH_TOKEN env var with the bearer token value | VERIFIED | `server/server.go` line 520: `"PROTOCONF_AUTH_TOKEN="+authToken`; `Test_server_MutateConfig/scripts_receive_auth_credentials` passes |
| 7 | Pre/post scripts receive PROTOCONF_SCRIPT_METADATA env var from the mutation request | VERIFIED | `server/server.go` line 521: `"PROTOCONF_SCRIPT_METADATA="+scriptMetadata`; call sites pass `in.GetScriptMetadata()` |
| 8 | PROTOCONF_COMPILER_ADDR env var has correct = separator (bug fix) | VERIFIED | `server/server.go` line 519: `"PROTOCONF_COMPILER_ADDR="+s.config.grpcAddress`; old form `"PROTOCONF_COMPILER_ADDR"` not present |
| 9 | Server fails to start if --pre or --post script path does not exist | VERIFIED | `validateScriptPath` returns error "does not exist"; wired in `run()` lines 154-161 with `return 1` |
| 10 | Server fails to start if --pre or --post script path is not executable | VERIFIED | `validateScriptPath` checks `info.Mode()&0111 == 0`; `Test_validateScriptPath/non-executable_file_returns_error_containing_not_executable` passes |
| 11 | Server fails to start if script path contains .. | VERIFIED | `validateScriptPath` checks `strings.Contains(path, "..")`; two test cases pass |
| 12 | runScript re-checks file existence before exec.Command (defense-in-depth) | VERIFIED | `server/server.go` lines 513-515: `os.Stat(filename)` before `exec.Command(filename)` |

**Score:** 12/12 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `server/server.go` | bearerTokenInterceptor function and --auth-token flag | VERIFIED | Function at lines 96-115; flag at line 89; wired via ChainUnaryInterceptor at line 183 |
| `server/server.go` | validateScriptPath function and updated runScript with credential forwarding | VERIFIED | validateScriptPath at lines 491-509; runScript signature at line 511; both env vars present |
| `server/server_test.go` | Auth interceptor unit tests | VERIFIED | Test_bearerTokenInterceptor at line 217 with 6 table-driven subtests; all pass |
| `server/server_test.go` | Tests for validateScriptPath and runScript env vars | VERIFIED | Test_validateScriptPath at line 275 with 7 subtests; scripts_receive_auth_credentials subtest at line 112 |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| server/server.go bearerTokenInterceptor | grpc.ChainUnaryInterceptor | serverOpts append in run() | WIRED | Line 183: `grpc.ChainUnaryInterceptor(bearerTokenInterceptor(config.authToken))` |
| server/server.go bearerTokenInterceptor | metadata.FromIncomingContext | authorization header extraction | WIRED | Lines 101-105: `md, ok := metadata.FromIncomingContext(ctx)` then `md.Get("authorization")` |
| server/server.go run() | validateScriptPath | startup validation before grpc.NewServer | WIRED | Lines 154-161: both pre and post validated; `grpc.NewServer` at line 188 is after validation |
| server/server.go runScript | PROTOCONF_AUTH_TOKEN | cmd.Env append | WIRED | Line 520: `"PROTOCONF_AUTH_TOKEN="+authToken` in cmd.Env |
| server/server.go runScript | PROTOCONF_COMPILER_ADDR= | bug fix: missing = separator | WIRED | Line 519: `"PROTOCONF_COMPILER_ADDR="+s.config.grpcAddress` |

### Data-Flow Trace (Level 4)

Not applicable — phase produces security middleware and validation functions, not data-rendering components.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Test_bearerTokenInterceptor (all 6 subtests) | `go test ./server/ -run Test_bearerTokenInterceptor -v` | All 6 subtests PASS | PASS |
| Test_validateScriptPath (all 7 subtests) | `go test ./server/ -run Test_validateScriptPath -v` | All 7 subtests PASS | PASS |
| Full server test suite | `go test ./server/ -v` | All 10 top-level test functions PASS (10.4s) | PASS |
| Full project build | `go build ./...` | Zero errors | PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| SECR-04 | 06-01-PLAN.md | Mutation server supports token-based authentication via gRPC metadata | SATISFIED | bearerTokenInterceptor wired into grpc.NewServer via ChainUnaryInterceptor; --auth-token flag registered |
| SECR-05 | 06-02-PLAN.md | Auth credentials are forwarded to pre/post mutation scripts as environment variables | SATISFIED | runScript appends PROTOCONF_AUTH_TOKEN and PROTOCONF_SCRIPT_METADATA to cmd.Env |
| SECR-06 | 06-01-PLAN.md | Unauthenticated requests are rejected when auth is configured | SATISFIED | interceptor returns codes.Unauthenticated for missing/invalid tokens when expectedToken != "" |
| SECR-07 | 06-02-PLAN.md | Pre/post mutation script paths are validated (exist, executable) before execution | SATISFIED | validateScriptPath checks existence, executability, and path traversal; called in run() at startup |

All four requirements from REQUIREMENTS.md for Phase 6 are accounted for across the two plans. No orphaned requirements.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| server/server_test.go | 174-215 | Test_cliCommand_Run creates non-executable temp files for -pre/-post but passes them to a separate FlagSet and then calls command.run with only protoconfRoot (flags.Args() excludes parsed flags) — the scripts are never validated | Info | Test does not cover script validation at the CLI level; this is a pre-existing test design issue, not introduced by this phase |

No blockers. The anti-pattern noted is a pre-existing cosmetic test issue that does not affect the phase goal. The temp files in that test are never actually passed to validateScriptPath.

### Human Verification Required

#### 1. Bearer token rejection via live gRPC client

**Test:** Start the mutation server with `--auth-token=secret`. Send a MutateConfig RPC without an Authorization header using grpcurl or a Go client. Then send one with `Authorization: Bearer secret`.
**Expected:** First request returns gRPC status Unauthenticated. Second request succeeds.
**Why human:** Requires a running server process and a gRPC client; cannot test without starting the HTTP/gRPC server.

### Gaps Summary

No gaps. All 12 must-have truths are verified, all 4 artifacts pass all levels, all 5 key links are wired, all 4 requirements are satisfied, and the full test suite passes with a clean build.

The one noted observation (Test_cliCommand_Run's non-executable temp file setup) is cosmetic and pre-existing — it does not affect the correctness of any phase deliverable.

---

_Verified: 2026-03-28_
_Verifier: Claude (gsd-verifier)_
