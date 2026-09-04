# Phase 05: TLS Support - Research

**Researched:** 2026-03-27
**Domain:** Go TLS / gRPC transport credentials
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**TLS Helper Package**
- D-01: Create a shared TLS helper (e.g., `utils/tls.go` or similar) that constructs `tls.Config` from cert/key/CA file paths. Both agent and server will use this to avoid duplicating the `tls.LoadX509KeyPair` / `x509.CertPool` logic.
- D-02: The helper accepts file paths (not inline text) for the common case. The agent's proto `TLSConfig` has both `key_file`/`key_text` oneofs — the helper should handle both forms since the agent proto already defines them.

**Agent (gRPC Server) — SECR-01**
- D-03: Wire the existing `AgentConfig.TlsConfig` proto fields to actual `grpc.Creds(credentials.NewTLS(...))` server option in `agent/agent.go` at `grpc.NewServer()`.
- D-04: Wire the existing `AgentConfig.Insecure` bool — when true (or when no TLS config provided), skip TLS. When false and TLS config is present, enforce TLS.
- D-05: The agent already has flags auto-generated from proto via `libprotoconf.PopulateFlagSet` — no new flag registration needed, just Go wiring.

**Mutation Server (gRPC Server) — SECR-01**
- D-06: Add `--tls-cert` and `--tls-key` flags (and optional `--tls-ca` for client cert verification) to `server/server.go`'s `cliConfig` and `newFlagSet()`.
- D-07: Wire these flags to `grpc.Creds(credentials.NewTLS(...))` server option at `grpc.NewServer()` in `server/server.go`.

**Mutate CLI (gRPC Client) — SECR-02**
- D-08: Add `--tls-cert`, `--tls-key`, `--tls-ca`, and `--insecure` flags to `mutate/mutate.go`'s CLI config.
- D-09: When TLS flags are provided, use `credentials.NewTLS(...)` instead of `insecure.NewCredentials()` for `grpc.NewClient()`.
- D-10: When `--insecure` (or no TLS flags), keep current `insecure.NewCredentials()` behavior.

**Insecure Warning — SECR-03**
- D-11: At server startup (both agent and mutation server), if no TLS config is provided, log `slog.Warn("gRPC server running without TLS — connections are not encrypted")`.
- D-12: The warning is informational — no behavioral change, no failure, no env var to suppress.

**Out of Scope**
- D-13: DevServer does NOT get TLS support.
- D-14: Example clients in `examples/` are NOT modified.
- D-15: Compiler service gRPC connection (`compiler/command.go`) does NOT get TLS.

### Claude's Discretion
- Exact file placement of the TLS helper (could be `utils/tls.go`, `internal/tls/`, or a new `tls/` package)
- Whether to use `AgentConfig_TLSConfig` type directly in the helper or abstract to a simpler struct
- Whether `--tls-ca` flag name should be `--tls-ca` or `--tls-ca-cert` — prefer `--tls-ca` for brevity

### Deferred Ideas (OUT OF SCOPE)
- mTLS support (SECR-08) — deferred to v2 per REQUIREMENTS.md
- TLS for KV store connections (agent's `store_tls` field) — related but not in SECR-01/02/03 scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SECR-01 | gRPC servers accept --tls-cert and --tls-key flags to enable TLS | `grpc.Creds(credentials.NewTLS(...))` is the standard server option; agent uses proto-generated flags, server uses manual FlagSet |
| SECR-02 | gRPC clients support TLS connections when server has TLS enabled | `credentials.NewTLS(...)` replaces `insecure.NewCredentials()` at `grpc.NewClient()` call site in `mutate/mutate.go:210` |
| SECR-03 | Insecure mode remains the default but logs a warning | `slog.Warn(...)` at server startup before `grpc.NewServer()` when no TLS config is detected |
</phase_requirements>

## Summary

Phase 05 wires Go's standard `crypto/tls` package to gRPC's `credentials.TransportCredentials` interface for two server components (agent and mutation server) and one client (mutate CLI). The TLS primitives are entirely standard library — no new dependencies are needed. The implementation is purely additive: new code paths activate when TLS flags are present; existing code paths (insecure) remain unchanged except for a startup warning log line.

The agent already has proto-defined TLS config (`AgentConfig.TlsConfig` with key/cert/CA oneofs) and auto-generated CLI flags via `libprotoconf.PopulateFlagSet`. For the agent, this phase is purely Go wiring — reading the proto fields and constructing a `tls.Config`. The mutation server and mutate CLI have no TLS config today; they need manual `flag.FlagSet` additions consistent with their existing patterns.

The primary technical decision left to Claude's discretion is TLS helper placement. Given that `utils/` already exists as a shared-utility package (it contains `DescriptorRegistry` and other helpers), placing `BuildTLSConfig()` in `utils/tls.go` requires no new package, is consistent with project conventions, and keeps the helper co-located with existing test infrastructure.

**Primary recommendation:** Place the TLS helper in `utils/tls.go`. Use a simple `TLSFiles` struct (cert, key, CA paths) as the input type rather than `AgentConfig_TLSConfig` directly — this keeps the helper package-agnostic and usable by both agent and server without creating an import dependency on `agent/config/v1` from `utils/`.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `crypto/tls` (stdlib) | Go 1.22 | Build `*tls.Config` from cert/key/CA | The only TLS implementation in Go stdlib; no alternative |
| `crypto/x509` (stdlib) | Go 1.22 | Load CA cert pool for server/client verification | Standard for building `CertPool` from PEM files |
| `google.golang.org/grpc/credentials` | v1.64.0 (already in go.mod) | `credentials.NewTLS(*tls.Config)` → `TransportCredentials` | Official gRPC-Go credentials package |
| `google.golang.org/grpc/credentials/insecure` | v1.64.0 (already in go.mod) | Insecure fallback (existing, unchanged) | Already used throughout codebase post-Phase-1 migration |

### No New Dependencies
All required packages are Go stdlib or already in `go.mod`. This phase adds zero new `require` directives.

## Architecture Patterns

### Recommended Project Structure
```
utils/
├── utils.go          # existing — DescriptorRegistry etc.
├── utils_test.go     # existing
├── tls.go            # NEW — BuildTLSConfig helper
├── tls_test.go       # NEW — unit tests for TLS helper
└── testdata/         # existing test fixtures
```

### Pattern 1: TLS Helper Function Signature

**What:** A standalone function that accepts cert/key/CA file paths (or inline PEM text) and returns `(*tls.Config, error)`.

**When to use:** Called by agent (from proto fields) and server/mutate (from flag values) at startup before gRPC server/client construction.

**Example:**
```go
// utils/tls.go
package utils

import (
    "crypto/tls"
    "crypto/x509"
    "fmt"
    "os"
)

// TLSFiles holds file paths for TLS configuration.
// Either CertFile+KeyFile or CertText+KeyText must be set for server TLS.
// CAFile or CAText is optional — used for client cert verification.
type TLSFiles struct {
    CertFile string
    CertText string // PEM text (agent proto key_text/cert_text oneofs)
    KeyFile  string
    KeyText  string
    CAFile   string
    CAText   string
}

// BuildTLSConfig constructs a *tls.Config from the provided file paths or
// inline PEM text. Returns nil, nil if no cert/key is configured (no TLS).
// Returns an error if cert/key are partially configured or files are unreadable.
func BuildTLSConfig(f TLSFiles) (*tls.Config, error) {
    hasCert := f.CertFile != "" || f.CertText != ""
    hasKey  := f.KeyFile != "" || f.KeyText != ""

    if !hasCert && !hasKey {
        return nil, nil // no TLS configured
    }
    if hasCert != hasKey {
        return nil, fmt.Errorf("tls: cert and key must both be set")
    }

    var certPEM, keyPEM []byte
    var err error

    if f.CertFile != "" {
        if certPEM, err = os.ReadFile(f.CertFile); err != nil {
            return nil, fmt.Errorf("tls: reading cert file: %w", err)
        }
    } else {
        certPEM = []byte(f.CertText)
    }

    if f.KeyFile != "" {
        if keyPEM, err = os.ReadFile(f.KeyFile); err != nil {
            return nil, fmt.Errorf("tls: reading key file: %w", err)
        }
    } else {
        keyPEM = []byte(f.KeyText)
    }

    cert, err := tls.X509KeyPair(certPEM, keyPEM)
    if err != nil {
        return nil, fmt.Errorf("tls: parsing cert/key pair: %w", err)
    }

    tlsCfg := &tls.Config{Certificates: []tls.Certificate{cert}}

    if f.CAFile != "" || f.CAText != "" {
        var caPEM []byte
        if f.CAFile != "" {
            if caPEM, err = os.ReadFile(f.CAFile); err != nil {
                return nil, fmt.Errorf("tls: reading ca file: %w", err)
            }
        } else {
            caPEM = []byte(f.CAText)
        }
        pool := x509.NewCertPool()
        if !pool.AppendCertsFromPEM(caPEM) {
            return nil, fmt.Errorf("tls: failed to parse CA certificate")
        }
        tlsCfg.ClientCAs = pool
        tlsCfg.ClientAuth = tls.RequireAndVerifyClientCert
    }

    return tlsCfg, nil
}
```

### Pattern 2: Agent TLS Wiring (proto fields → grpc.ServerOption)

**What:** At `grpc.NewServer()` in `agent/agent.go`, check `config.TlsConfig` and convert to `grpc.Creds(credentials.NewTLS(...))`. Log warning if no TLS.

**Example:**
```go
// agent/agent.go — inside RunAgent(), before grpc.NewServer()
import (
    "google.golang.org/grpc/credentials"
    "github.com/protoconf/protoconf/utils"
)

serverOpts := []grpc.ServerOption{
    grpc.StatsHandler(otelgrpc.NewServerHandler()),
    grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor),
    grpc.UnaryInterceptor(grpc_prometheus.UnaryServerInterceptor),
}

if config.TlsConfig != nil {
    tlsFiles := utils.TLSFiles{
        CertFile: config.TlsConfig.GetCertFile(),
        CertText: config.TlsConfig.GetCertText(),
        KeyFile:  config.TlsConfig.GetKeyFile(),
        KeyText:  config.TlsConfig.GetKeyText(),
        CAFile:   config.TlsConfig.GetCaFile(),
        CAText:   config.TlsConfig.GetCaText(),
    }
    tlsCfg, err := utils.BuildTLSConfig(tlsFiles)
    if err != nil {
        return fmt.Errorf("tls config error: %w", err)
    }
    if tlsCfg != nil {
        serverOpts = append(serverOpts, grpc.Creds(credentials.NewTLS(tlsCfg)))
    }
} else {
    slog.Warn("gRPC server running without TLS — connections are not encrypted")
}

rpcServer := grpc.NewServer(serverOpts...)
```

### Pattern 3: Mutation Server TLS Wiring (flags → grpc.ServerOption)

**What:** Add `tlsCert`, `tlsKey`, `tlsCA` fields to `cliConfig` in `server/server.go`, register in `newFlagSet()`, then apply at `grpc.NewServer()`.

**Example:**
```go
// server/server.go — cliConfig additions
type cliConfig struct {
    grpcAddress        string
    preMutationScript  string
    postMutationScript string
    tlsCert            string
    tlsKey             string
    tlsCA              string
}

// In newFlagSet():
flags.StringVar(&config.tlsCert, "tls-cert", "", "TLS certificate file path")
flags.StringVar(&config.tlsKey, "tls-key", "", "TLS key file path")
flags.StringVar(&config.tlsCA, "tls-ca", "", "TLS CA certificate file path (enables client cert verification)")

// In run(), before grpc.NewServer():
serverOpts := []grpc.ServerOption{grpc.StatsHandler(otelgrpc.NewServerHandler())}

tlsCfg, err := utils.BuildTLSConfig(utils.TLSFiles{
    CertFile: config.tlsCert,
    KeyFile:  config.tlsKey,
    CAFile:   config.tlsCA,
})
if err != nil {
    slog.Error("failed to build TLS config", "error", err)
    return 1
}
if tlsCfg != nil {
    serverOpts = append(serverOpts, grpc.Creds(credentials.NewTLS(tlsCfg)))
} else {
    slog.Warn("gRPC server running without TLS — connections are not encrypted")
}

rpcServer := grpc.NewServer(serverOpts...)
```

### Pattern 4: Mutate CLI TLS Wiring (flags → grpc.DialOption)

**What:** Add TLS flags to `mutate/mutate.go`'s `cliConfig`, choose `credentials.NewTLS(...)` vs `insecure.NewCredentials()` at `grpc.NewClient()`.

**Example:**
```go
// mutate/mutate.go — cliConfig additions
type cliConfig struct {
    // ... existing fields ...
    tlsCert  string
    tlsKey   string
    tlsCA    string
    insecure bool
}

// In newFlagSet():
flags.StringVar(&config.tlsCert, "tls-cert", "", "TLS client certificate file")
flags.StringVar(&config.tlsKey, "tls-key", "", "TLS client key file")
flags.StringVar(&config.tlsCA, "tls-ca", "", "TLS CA certificate file")
flags.BoolVar(&config.insecure, "insecure", true, "Use insecure connection (default true; set false to enforce TLS)")

// In Run(), replacing grpc.NewClient() call:
var dialCreds grpc.DialOption
if config.tlsCert != "" || config.tlsKey != "" || config.tlsCA != "" {
    tlsCfg, err := utils.BuildTLSConfig(utils.TLSFiles{
        CertFile: config.tlsCert,
        KeyFile:  config.tlsKey,
        CAFile:   config.tlsCA,
    })
    if err != nil {
        slog.Error("failed to build TLS config", "error", err)
        return 1
    }
    if tlsCfg == nil {
        // CA only — connect with system roots, no client cert
        tlsCfg = &tls.Config{}
    }
    dialCreds = grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg))
} else {
    dialCreds = grpc.WithTransportCredentials(insecure.NewCredentials())
}

conn, err := grpc.NewClient(address, dialCreds)
```

**Note on mutate client TLS:** The mutate client connecting to a TLS-enabled server typically only needs the CA (or system roots) to verify the server cert — it does not need its own cert/key unless the server requires client auth (mTLS, which is deferred to v2). The `--tls-cert` and `--tls-key` flags are added for completeness but `--tls-ca` alone (or an empty `tls.Config{}` using system roots) is the common path.

### Anti-Patterns to Avoid
- **Do not use `tls.LoadX509KeyPair(certFile, keyFile)` directly at call sites:** The agent proto has both file and text oneofs — consolidate this branching in the helper once, not at every call site.
- **Do not set `InsecureSkipVerify: true` in any `tls.Config`:** This defeats TLS entirely; never use in production paths. Testing with self-signed certs uses proper CA verification.
- **Do not add `credentials.NewTLS` to the grpcui bufconn connection in `server.go:GenReflectionUI`:** That internal loopback connection uses `insecure.NewCredentials()` intentionally — it is not a network-facing connection.
- **Do not modify `devserver/`:** Per D-13, DevServer is explicitly out of scope.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Certificate loading | Custom PEM parser | `tls.X509KeyPair(cert, key []byte)` | Handles all cert formats, well-tested |
| CA pool construction | Manual cert parsing | `x509.NewCertPool()` + `pool.AppendCertsFromPEM(pem)` | Standard library; handles chain certs |
| TLS over gRPC | Custom transport layer | `credentials.NewTLS(*tls.Config)` | Official gRPC-Go mechanism; handles TLS handshake, SNI, etc. |

**Key insight:** The entire TLS stack for this phase fits in ~50 lines of glue code. Everything is stdlib + already-imported gRPC packages.

## Common Pitfalls

### Pitfall 1: nil TLSConfig vs Empty TLSConfig
**What goes wrong:** Calling `credentials.NewTLS(nil)` panics. Calling `credentials.NewTLS(&tls.Config{})` with no certificates fails at handshake time, not construction time.
**Why it happens:** The server needs `Certificates` set; the client only needs `RootCAs` (or trusts system roots with empty config).
**How to avoid:** The helper returns `(nil, nil)` when no cert/key is provided — callers must nil-check before passing to `credentials.NewTLS`.
**Warning signs:** Nil pointer panic in `credentials` package at server start.

### Pitfall 2: Agent `Insecure` Bool Logic
**What goes wrong:** The proto has `insecure bool` (field 7) which defaults to `false`. If code treats `insecure == false` as "enforce TLS", then any agent started without TLS config would fail to accept connections.
**Why it happens:** The proto field name is misleading — it means "skip TLS when true", not "require TLS when false".
**How to avoid:** The rule is: if `TlsConfig != nil` → use TLS; if `TlsConfig == nil` → use insecure (log warning). The `Insecure` bool is a gate to skip TLS even when TlsConfig is set (D-04: when true OR when no TLS config, skip TLS).
**Warning signs:** Agent fails to start after upgrade when no `--tls-config` was ever passed.

### Pitfall 3: `x509.CertPool.AppendCertsFromPEM` Silent Failure
**What goes wrong:** `AppendCertsFromPEM` returns `false` if no valid certs are found in the PEM block, but callers often ignore this return value.
**Why it happens:** The function signature doesn't return an error — it returns bool.
**How to avoid:** Check the return value and return an error from `BuildTLSConfig` if it's false. See code example above.
**Warning signs:** TLS handshake fails with "certificate signed by unknown authority" even when a CA cert was provided.

### Pitfall 4: grpcui bufconn Connection Must Stay Insecure
**What goes wrong:** `server/server.go:GenReflectionUI` creates an in-process `bufconn` connection for the grpcui frontend. If the outer gRPC server is given TLS credentials, the bufconn connection still uses `insecure.NewCredentials()` — this is intentional.
**Why it happens:** The bufconn connection never goes over the network; TLS on it would require cert generation at runtime.
**How to avoid:** Keep `grpc.WithTransportCredentials(insecure.NewCredentials())` in `GenReflectionUI` unchanged. The outer `httpServer.ListenAndServe()` path is where TLS matters.
**Warning signs:** Unit test for `GenReflectionUI` fails after TLS changes.

### Pitfall 5: Test Cert Generation for Unit Tests
**What goes wrong:** Tests that exercise TLS code paths need valid cert/key pairs. Using hardcoded PEM strings is fragile; file paths that don't exist fail the helper.
**Why it happens:** Real cert files cannot be committed to version control.
**How to avoid:** Use `crypto/tls.X509KeyPair` with self-signed certs generated in-memory via `crypto/x509.CreateCertificate` inside a test helper function, OR use `net/http/httptest.NewTLSServer`-style patterns. For `BuildTLSConfig` unit tests, generate a self-signed cert in `TestMain` or as a `t.TempDir()` file.
**Warning signs:** Tests fail with "no such file or directory" for cert/key paths.

## Code Examples

### Generating Self-Signed Cert for Tests
```go
// Source: Go stdlib crypto/x509 + crypto/tls
import (
    "crypto/ecdsa"
    "crypto/elliptic"
    "crypto/rand"
    "crypto/tls"
    "crypto/x509"
    "crypto/x509/pkix"
    "encoding/pem"
    "math/big"
    "time"
)

func generateSelfSignedCert(t *testing.T) (certPEM, keyPEM []byte) {
    t.Helper()
    key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
    require.NoError(t, err)

    tmpl := &x509.Certificate{
        SerialNumber: big.NewInt(1),
        Subject:      pkix.Name{Organization: []string{"test"}},
        NotBefore:    time.Now().Add(-time.Hour),
        NotAfter:     time.Now().Add(time.Hour),
        KeyUsage:     x509.KeyUsageDigitalSignature,
        ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
        IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
    }
    certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
    require.NoError(t, err)

    certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
    keyBytes, err := x509.MarshalECPrivateKey(key)
    require.NoError(t, err)
    keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})
    return
}
```

### Full TLS Test Pattern (Agent)
```go
// agent/agent_test.go — TLS test case additions
{
    name: "run with TLS",
    args: args{
        ctx: timeoutCtx(5 * time.Second),
        config: func() *protoconf_agent_config.AgentConfig {
            certPEM, keyPEM := generateSelfSignedCert(t)
            // Write to temp files
            certFile := writeTempFile(t, certPEM)
            keyFile  := writeTempFile(t, keyPEM)
            return &protoconf_agent_config.AgentConfig{
                GrpcAddress: ":0",
                HttpAddress: ":0",
                DevRoot:     testdata.SmallTestDir(),
                TlsConfig: &protoconf_agent_config.AgentConfig_TLSConfig{
                    Cert: &protoconf_agent_config.AgentConfig_TLSConfig_CertFile{CertFile: certFile},
                    Key:  &protoconf_agent_config.AgentConfig_TLSConfig_KeyFile{KeyFile: keyFile},
                },
            }
        }(),
    },
},
{
    name: "run without TLS logs warning",
    // test that slog.Warn is called — use slog.SetDefault with a custom handler
    // that captures log records, verify a WARN record with "not encrypted"
},
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `grpc.WithInsecure()` | `grpc.WithTransportCredentials(insecure.NewCredentials())` | Phase 1 | Already done; all call sites migrated |
| Custom TLS loading | `crypto/tls` stdlib + `grpc/credentials` | N/A (always standard) | No change needed |

**No deprecated patterns to address in this phase.** Phase 1 already migrated all `grpc.WithInsecure()` calls.

## Environment Availability

Step 2.6: SKIPPED — this phase adds no external CLI tools, services, or runtimes. All required packages are Go stdlib or already in `go.mod`.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + testify v1.9.0 |
| Config file | none (standard `go test`) |
| Quick run command | `go test ./utils/... ./agent/... ./server/... ./mutate/...` |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SECR-01 | Agent accepts TLS via `--tls-config` proto flags | unit | `go test ./agent/... -run TestRunAgent` | ✅ `agent/agent_test.go` (needs TLS case) |
| SECR-01 | Mutation server accepts `--tls-cert`/`--tls-key` flags | unit | `go test ./server/... -run Test_cliCommand_Run` | ✅ `server/server_test.go` (needs TLS case) |
| SECR-01 | `BuildTLSConfig` constructs valid `*tls.Config` from files | unit | `go test ./utils/... -run TestBuildTLSConfig` | ❌ Wave 0 |
| SECR-01 | `BuildTLSConfig` returns nil,nil for empty input | unit | `go test ./utils/... -run TestBuildTLSConfig` | ❌ Wave 0 |
| SECR-01 | `BuildTLSConfig` errors on partial cert/key | unit | `go test ./utils/... -run TestBuildTLSConfig` | ❌ Wave 0 |
| SECR-02 | Mutate CLI uses TLS creds when `--tls-cert` set | unit | `go test ./mutate/... -run TestRun_TLS` | ❌ Wave 0 |
| SECR-03 | Agent logs WARN when no TLS config | unit | `go test ./agent/... -run TestRunAgent_InsecureWarning` | ❌ Wave 0 |
| SECR-03 | Server logs WARN when no TLS flags | unit | `go test ./server/... -run Test_cliCommand_Run_InsecureWarning` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./utils/... ./agent/... ./server/... ./mutate/...`
- **Per wave merge:** `go test ./...`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `utils/tls.go` — new TLS helper (covers SECR-01 helper)
- [ ] `utils/tls_test.go` — unit tests for `BuildTLSConfig` (covers SECR-01 unit)
- [ ] `mutate/mutate_test.go` — new test file (covers SECR-02; mutate has no tests today)
- [ ] Self-signed cert generator helper — inline in test files or extracted to `utils/testdata`

## Project Constraints (from CLAUDE.md)

- **CGO_ENABLED=0**: No cgo dependencies — `crypto/tls` is pure Go, no issue.
- **Error handling**: Use `fmt.Errorf("...: %w", err)` for context, sentinel errors as `var ErrX = errors.New(...)`, never silent failures.
- **Logging**: Use `slog.Warn(...)` with structured fields — `slog.Warn("gRPC server running without TLS — connections are not encrypted")`.
- **No panics in production code**: `BuildTLSConfig` must return error, not panic, on bad input.
- **File naming**: New files follow `snake_case.go` — `tls.go`, `tls_test.go`.
- **Package naming**: Fits in `utils` package (lowercase, single word).
- **Constructor pattern**: `BuildTLSConfig` is a function not a constructor, which is fine since it returns a stdlib type, not a project-specific struct.
- **Import organization**: `google.golang.org/grpc/credentials` import uses no alias (no collision with other packages).
- **No os.Exit in library code**: `BuildTLSConfig` must return error — Phase 2 already fixed os.Exit patterns and this helper must not introduce new ones.

## Sources

### Primary (HIGH confidence)
- Go stdlib `crypto/tls` — verified via `go doc crypto/tls` in working directory
- Go stdlib `crypto/x509` — standard library, no version concerns
- `google.golang.org/grpc/credentials` v1.64.0 — verified via `go doc` in working directory; `credentials.NewTLS(*tls.Config) TransportCredentials` confirmed
- `agent/config/v1/agent_config.pb.go` — read directly; confirmed `AgentConfig_TLSConfig` with key/cert/ca oneofs and accessor methods
- `agent/agent.go` — read directly; confirmed `grpc.NewServer()` call site at line 128
- `server/server.go` — read directly; confirmed `grpc.NewServer()` at line 120, `cliConfig` struct, `newFlagSet()` pattern
- `mutate/mutate.go` — read directly; confirmed `grpc.NewClient()` at line 210, `cliConfig` struct pattern
- `utils/utils.go` — read directly; confirmed `utils` package is the right home for the helper

### Secondary (MEDIUM confidence)
- Existing `agent/agent_test.go` and `server/server_test.go` patterns — direct code read; test structure established, TLS cases need to be added

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all stdlib + already-imported packages; verified via go doc
- Architecture: HIGH — insertion points read directly from source; pattern is standard gRPC-Go
- Pitfalls: HIGH — identified from direct code reading (bufconn pattern, nil TLSConfig, insecure bool semantics)

**Research date:** 2026-03-27
**Valid until:** 2026-06-27 (stable APIs — crypto/tls and gRPC credentials are extremely stable)
