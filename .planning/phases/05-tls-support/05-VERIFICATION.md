---
phase: 05-tls-support
verified: 2026-03-27T00:00:00Z
status: passed
score: 11/11 must-haves verified
re_verification: false
gaps: []
human_verification:
  - test: "Start the agent with TLS cert/key configured and connect a gRPC client using matching credentials"
    expected: "Encrypted connection established; connecting without TLS credentials should fail with a transport error"
    why_human: "Requires running the agent process and a live gRPC client — cannot verify encrypted transport programmatically without starting servers"
  - test: "Start the mutation server without --tls-cert/--tls-key and observe startup logs"
    expected: "slog.Warn message 'gRPC server running without TLS -- connections are not encrypted' appears on stderr"
    why_human: "Log output requires a running process"
---

# Phase 05: TLS Support Verification Report

**Phase Goal:** gRPC servers and clients support TLS connections; insecure mode warns operators
**Verified:** 2026-03-27
**Status:** PASSED
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | BuildTLSConfig returns a valid *tls.Config when given cert+key file paths | VERIFIED | `tls_test.go` subtest "CertFile+KeyFile returns valid tls.Config" passes |
| 2 | BuildTLSConfig returns a valid *tls.Config when given cert+key PEM text | VERIFIED | `tls_test.go` subtest "CertText+KeyText returns valid tls.Config" passes |
| 3 | BuildTLSConfig returns nil,nil when no cert or key is provided | VERIFIED | `tls_test.go` subtest "empty TLSFiles returns nil,nil" passes |
| 4 | BuildTLSConfig returns an error when only cert or only key is provided | VERIFIED | Two subtests for cert-only and key-only both pass with "cert and key must both be set" |
| 5 | BuildTLSConfig adds CA to ClientCAs pool when CA file is provided | VERIFIED | `tls_test.go` subtests "CertFile+KeyFile+CAFile" and "CertFile+KeyFile+CAText" pass; `tls.go:100` sets `ClientAuth = tls.RequireAndVerifyClientCert` |
| 6 | Agent gRPC server uses TLS credentials when TlsConfig proto fields are set | VERIFIED | `agent/agent.go:136–156` conditionally appends `grpc.Creds(credentials.NewTLS(tlsCfg))` when `config.TlsConfig != nil && !config.Insecure` |
| 7 | Mutation server gRPC server uses TLS credentials when --tls-cert and --tls-key flags are provided | VERIFIED | `server/server.go:128–145` builds dynamic serverOpts with `grpc.Creds(credentials.NewTLS(tlsCfg))` |
| 8 | Mutate CLI gRPC client uses TLS credentials when --tls-cert/--tls-key/--tls-ca flags are provided | VERIFIED | `mutate/mutate.go:221–239` conditionally sets `dialOpt = grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg))` |
| 9 | Agent logs slog.Warn when no TLS config is provided | VERIFIED | `agent/agent.go:153` `slog.Warn("gRPC server running without TLS -- connections are not encrypted")` in else branch |
| 10 | Mutation server logs slog.Warn when no TLS flags are provided | VERIFIED | `server/server.go:143` same warning string in else branch |
| 11 | Existing insecure-mode usage continues without flag changes | VERIFIED | mutate CLI defaults to `insecure.NewCredentials()` when no TLS flags set; GenReflectionUI bufconn retains `insecure.NewCredentials()` at `server.go:521` |

**Score:** 11/11 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `utils/tls.go` | TLS helper: TLSFiles struct + BuildTLSConfig function | VERIFIED | 104 lines; exports `TLSFiles` and `BuildTLSConfig`; contains `tls.X509KeyPair`, `x509.NewCertPool`, `tls.RequireAndVerifyClientCert`; no `InsecureSkipVerify`; no `os.Exit` |
| `utils/tls_test.go` | Unit tests for BuildTLSConfig | VERIFIED | 194 lines; `TestBuildTLSConfig` with 11 table-driven subtests; `generateSelfSignedCert` helper |
| `agent/agent.go` | TLS wiring for agent gRPC server + insecure warning | VERIFIED | Imports `utils` and `credentials`; contains `utils.BuildTLSConfig(utils.TLSFiles{...})` at line 137; `grpc.Creds(credentials.NewTLS(tlsCfg))` at line 149; `slog.Warn` at line 153 |
| `server/server.go` | TLS flags + wiring for mutation server + insecure warning | VERIFIED | `cliConfig` has `tlsCert`, `tlsKey`, `tlsCA`; three `flags.StringVar` registrations; `utils.BuildTLSConfig` at line 130; `grpc.Creds(credentials.NewTLS(...))` at line 140; `slog.Warn` at line 143 |
| `mutate/mutate.go` | TLS flags + wiring for mutate CLI gRPC client | VERIFIED | `cliConfig` has `tlsCert`, `tlsKey`, `tlsCA`, `insecureTLS`; four flag registrations; `utils.BuildTLSConfig` at line 223; `credentials.NewTLS(tlsCfg)` at line 236 |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `agent/agent.go` | `utils/tls.go` | `utils.BuildTLSConfig` call | WIRED | Line 137: `utils.BuildTLSConfig(utils.TLSFiles{...})` |
| `server/server.go` | `utils/tls.go` | `utils.BuildTLSConfig` call | WIRED | Line 130: `utils.BuildTLSConfig(utils.TLSFiles{...})` |
| `mutate/mutate.go` | `utils/tls.go` | `utils.BuildTLSConfig` call | WIRED | Line 223: `utils.BuildTLSConfig(utils.TLSFiles{...})` |
| `agent/agent.go` | `grpc.Creds` | `credentials.NewTLS(tlsCfg)` | WIRED | Line 149: `grpc.Creds(credentials.NewTLS(tlsCfg))` appended to `serverOpts` |
| `server/server.go` | `grpc.Creds` | `credentials.NewTLS(tlsCfg)` | WIRED | Line 140: `grpc.Creds(credentials.NewTLS(tlsCfg))` appended to `serverOpts` |
| `mutate/mutate.go` | `grpc.WithTransportCredentials` | `credentials.NewTLS(tlsCfg)` | WIRED | Line 236: `grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg))` |
| `server/server.go` GenReflectionUI | `insecure.NewCredentials()` | bufconn stays insecure | WIRED | Line 521: retained per design (in-process bufconn, not network-facing) |

### Data-Flow Trace (Level 4)

Not applicable — phase 05 produces no data-rendering components. All artifacts are infrastructure helpers (TLS config builders, gRPC option injectors), not UI components.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| All 11 TestBuildTLSConfig subtests pass | `go test ./utils/... -run TestBuildTLSConfig -v` | 11/11 PASS, exit code 0 | PASS |
| `go build ./agent/...` succeeds | `go build ./agent/...` | No output (success) | PASS |
| `go build ./server/...` succeeds | `go build ./server/...` | No output (success) | PASS |
| `go build ./mutate/...` succeeds | `go build ./mutate/...` | No output (success) | PASS |
| TLS helper exports correct symbols | `grep -c "BuildTLSConfig" utils/tls.go` | 2 occurrences | PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| SECR-01 | 05-01, 05-02 | gRPC servers accept --tls-cert and --tls-key flags to enable TLS | SATISFIED | Agent uses proto TlsConfig fields; mutation server adds three TLS flags; both wire `grpc.Creds(credentials.NewTLS(...))` |
| SECR-02 | 05-02 | gRPC clients support TLS connections when server has TLS enabled | SATISFIED | `mutate/mutate.go` wires `credentials.NewTLS(tlsCfg)` via `--tls-cert/--tls-key/--tls-ca` flags |
| SECR-03 | 05-02 | Insecure mode remains the default but logs a warning | SATISFIED | Both agent and mutation server call `slog.Warn("gRPC server running without TLS -- connections are not encrypted")` in the else/nil branch |

No orphaned requirements found — REQUIREMENTS.md maps only SECR-01, SECR-02, SECR-03 to Phase 5, and all three are covered by the two plans.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `agent/filekv/filekv.go` | 88–195 | `go vet`: mutex passed by value | Warning | Pre-existing before phase 05; out of scope |
| `agent/legacy.go` | 97 | `go vet`: unreachable code | Warning | Pre-existing before phase 05; out of scope |
| `agent/agent_test.go` | 27 | `go vet`: context leak (WithTimeoutCause cancel not called) | Warning | Pre-existing before phase 05; out of scope |

No anti-patterns introduced by phase 05. All vet warnings originate in files not touched by this phase. No TODO/FIXME, no `return null`/empty stubs, no `InsecureSkipVerify`, no `os.Exit` in any phase 05 file.

### Human Verification Required

#### 1. Agent TLS Round-Trip

**Test:** Generate a self-signed cert, start the agent with `--tls-config.cert-file=cert.pem --tls-config.key-file=key.pem`, connect a gRPC client configured with matching credentials.
**Expected:** Encrypted connection succeeds; a client using `insecure.NewCredentials()` should receive a transport error.
**Why human:** Requires running the agent process and a live gRPC client — cannot verify encrypted transport programmatically without starting servers.

#### 2. Insecure Startup Warning in Logs

**Test:** Start the mutation server without any `--tls-*` flags and observe stderr.
**Expected:** `WARN gRPC server running without TLS -- connections are not encrypted` appears at startup.
**Why human:** Log output requires a running process; programmatic check would only verify the `slog.Warn` call exists (which has been verified at code level).

### Gaps Summary

No gaps. All 11 must-have truths are verified, all 5 artifacts are substantive and wired, all 7 key links are confirmed, and all 3 requirements (SECR-01, SECR-02, SECR-03) are satisfied. The phase goal — "gRPC servers and clients support TLS connections; insecure mode warns operators" — is achieved.

---

_Verified: 2026-03-27_
_Verifier: Claude (gsd-verifier)_
