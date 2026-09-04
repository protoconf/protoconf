# Phase 10: Placeholder Fixes & Integration Tests - Research

**Researched:** 2026-03-31
**Domain:** Go testing — placeholder assertion completion, gRPC e2e tests with TLS and auth
**Confidence:** HIGH

## Summary

Phase 10 closes all outstanding placeholder assertions in four test files and extends the existing e2e test suite with three missing scenarios: full mutation flow with scripts, TLS-enabled gRPC, and token-based auth rejection/acceptance.

The placeholder work is purely mechanical: the code under test already works correctly (verified by running the existing tests), but the test functions lack assertions on the returned values. The four files are `server/server_test.go`, `compiler/lib/parser/parser_test.go`, `inserter/inserter_test.go`, and `agent/kv_agent_rollout_impl_test.go`.

The integration work extends `test/e2e_test.go`. The existing e2e test (`Test`) already demonstrates the full pattern: `testdata.SmallTestDir()` provides an isolated temp workspace, `TestServer` in `test/e2e.go` wraps `bufconn` for in-process gRPC, and the mutation server's `MutateConfig` is exercised end-to-end. The three new scenarios require adding TLS-aware server wiring (using `utils.BuildTLSConfig` + `credentials.NewTLS`) and a bearer-token auth interceptor test using `metadata.NewOutgoingContext`.

**Primary recommendation:** Add concrete assertions to the four placeholder test files and add three focused e2e sub-tests in `test/e2e_test.go` using the in-process `bufconn` pattern already established.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| TEST-07 | server/server_test.go MutateConfig test asserts response content (resolves TODO(smintz)) | `MutateConfig` returns `*protoconf_pb.ConfigMutationResponse` with `Uuid` field and optional `PreScriptDuration`/`PostScriptDuration` populated by `StoreReport`; verified by running tests |
| TEST-08 | compiler/lib/parser/parser_test.go placeholder test cases filled with real assertions | `ParseFilesX` returns `[]*desc.FileDescriptor` with `GetFullyQualifiedName()` verifiable; `ReadConfig` populates `proto.Message` with known JSON fixtures in testdata |
| TEST-09 | inserter/inserter_test.go placeholder test cases filled with real assertions | `InsertConfig` writes three KV paths (`config.data`, `config.json`, `metadata.json`) to dummykv; Help() already tested, TODO just needs more input variations |
| TEST-10 | agent/kv_agent_rollout_impl_test.go placeholder cases completed | Rollout test already has one realistic test case (`test1`) with goroutine-based assertions; TODO just needs additional edge case(s) |
| TEST-11 | e2e test covers mutation flow with pre/post script execution | `server.ProtoconfMutationServer` supports `PreMutationScript`/`PostMutationScript`; test infrastructure in `test/e2e.go` ready |
| TEST-12 | e2e test covers TLS-enabled gRPC connections | `utils.BuildTLSConfig` + `credentials.NewTLS` already exist; `utils/tls_test.go` shows cert generation helper `generateSelfSignedCert` |
| TEST-13 | e2e test covers token-based auth flow | `bearerTokenInterceptor` already in `server/server.go`; test in `server/server_test.go` shows the metadata patterns |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/stretchr/testify` | v1.9.0 | `assert.*` / `require.*` assertions | Project standard throughout all existing tests |
| `google.golang.org/grpc/test/bufconn` | (grpc v1.64.0) | In-process gRPC transport | Already used in `agent/kv_agent_impl_test.go` and `test/e2e.go` |
| `google.golang.org/grpc/metadata` | (grpc v1.64.0) | Inject auth tokens as gRPC metadata | Used in `server/server_test.go` for bearer token tests |
| `google.golang.org/grpc/credentials` | (grpc v1.64.0) | TLS credentials for gRPC | `credentials.NewTLS` used in `server/server.go` |
| `github.com/protoconf/protoconf/utils` | local | `BuildTLSConfig`, `TLSFiles` | Already wraps cert/key parsing; `generateSelfSignedCert` pattern in `utils/tls_test.go` |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `crypto/ecdsa`, `crypto/x509`, `encoding/pem` | stdlib | Self-signed cert generation | TLS e2e test; pattern already in `utils/tls_test.go` |
| `google.golang.org/grpc/codes`, `google.golang.org/grpc/status` | (grpc v1.64.0) | Assert gRPC error codes | Auth rejection test |
| `github.com/protoconf/protoconf/utils/testdata` | local | Isolated temp workspace | All tests needing a protoconf root |

## Architecture Patterns

### Recommended Project Structure

No new packages needed. Work is confined to:
```
agent/kv_agent_rollout_impl_test.go    # TEST-10: add edge case sub-test
compiler/lib/parser/parser_test.go    # TEST-08: add assertions + extra cases
inserter/inserter_test.go              # TEST-09: add assertions + extra Help cases
server/server_test.go                  # TEST-07: add assertions to existing stubs
test/e2e_test.go                       # TEST-11/12/13: new sub-tests
```

### Pattern 1: Asserting MutateConfig Response (TEST-07)

`MutateConfig` returns a `*protoconf_pb.ConfigMutationResponse`. When pre/post scripts run, `StoreReport` populates `PreScriptDuration` / `PostScriptDuration` (both `*durationpb.Duration`). The `Uuid` field is always set.

```go
// Source: server/server.go lines 398-477
resp, err := s.MutateConfig(ctx, req)
require.NoError(t, err)
require.NotNil(t, resp)
// When scripts ran, duration fields are populated
assert.NotNil(t, resp.PreScriptDuration)
assert.Greater(t, resp.PreScriptDuration.AsDuration(), time.Duration(0))
```

The existing `"run scripts"` test calls `MutateConfig` and discards the response with `_`. Change `_` to `resp` and add the assertions above.

Four server test functions currently have placeholder comment `// Add assertions here`:
- `Test_server_MutateConfig/run_scripts` — assert `resp.PreScriptDuration != nil && resp.PostScriptDuration != nil`
- `TestProtoconfMutationServer_GenReflectionUI` — assert `httpServer.Handler != nil` (set by `GenReflectionUI`)
- `TestProtoconfMutationServer_ReportProgress` — assert `got.Uuid == "test-uuid"`
- `TestProtoconfMutationServer_Put` — already asserts `errors.Is(err, ErrInternalCompilerError)`; add `assert.Nil(t, result)` for the returned `proto.Message`
- `Test_cliCommand_Run` — assert `exitCode == 0` (already done); verify run does not panic by adding `assert.NotNil(t, cmd)`

### Pattern 2: Parser Test Assertions (TEST-08)

`ParseFilesX` returns `[]*desc.FileDescriptor`. Each has `GetFullyQualifiedName()` (returns `"test.proto"` for the test fixture) and `GetMessageTypes()` which lists message descriptors.

`ReadConfig` populates a `proto.Message`. For `test.materialized_JSON` + `&protoconf_pb.ProtoconfValue{}`, the result has `proto_file == "test.proto"` and a non-nil `value`.

```go
// Source: compiler/lib/parser/parser_test.go current structure + parser.go
fds, err := p.ParseFilesX("test.proto")
require.NoError(t, err)
require.Len(t, fds, 1)
assert.Equal(t, "test.proto", fds[0].GetName())

// For ReadConfig:
msg := &protoconf_pb.ProtoconfValue{}
err = p.ReadConfig(filepath.Join(protoconfRoot, "materialized_config/test.materialized_JSON"), msg)
require.NoError(t, err)
assert.Equal(t, "test.proto", msg.ProtoFile)
assert.NotNil(t, msg.Value)
```

Add additional test cases for `ParseFilesX`: parse multiple files; parse a proto that has message types and verify at least one message type. Add additional test cases for `ReadConfig`: load `with_config_rollout.materialized_JSON` and assert `msg.RolloutConfig != nil`.

### Pattern 3: Inserter Test Assertions (TEST-09)

The `Test_cliCommand_Help` has a `// TODO: Add test cases.` inside the test table. Add a second table entry that verifies additional flags or the synopsis line. The test framework already loops the table so it is purely additive.

```go
// Additional Help test case
{
    name: "config-file flag",
    want: []string{"-config-file"},
},
```

The `TestProtoconfInserter_InsertConfig` test has an entry for `"test"` that already verifies the `config.data` key prefix. Strengthen by also asserting `config.json` and `metadata.json` exist in the dummykv store.

```go
// After InsertConfigFile, check all three KV paths
for _, suffix := range []string{"config.data", "config.json", "metadata.json"} {
    key := "test/" + suffix
    v, err := kvStore.Get(context.Background(), key, &store.ReadOptions{})
    assert.NoError(t, err, "key %s", key)
    assert.NotEmpty(t, v.Value, "key %s", key)
}
```

### Pattern 4: Rollout Test Additional Cases (TEST-10)

The existing `test1` case exercises a normal two-stage rollout. The `// TODO: Add test cases.` comment suggests adding a test case for a config update with no rollout stages (plain stable update). A minimal additional case:

```go
{
    name: "no_rollout_stages",
    args: args{
        updates: []*update{
            {
                configName:     "simple_key",
                protoconfValue: &protoconf_pb.ProtoconfValue{Value: newAny(structpb.NewStringValue("plain value"))},
                metadata:       &protoconf_pb.Metadata{Commit: "abcdef1234567890", CommittedAt: timestamppb.Now()},
            },
        },
        want: []*want{
            {
                agentChannel: "alpha",
                agentClient:  alphaClient,
                request:      &protoconf_pb.ConfigSubscriptionRequest{Path: "simple_key"},
                expects: []*result{
                    {update: &protoconf_pb.ConfigUpdate{Value: newAny(structpb.NewStringValue("plain value"))}, within: time.Second * 2},
                },
            },
        },
    },
},
```

Note: the rollout test currently takes ~16 seconds because `test1` has 5-second cooldown windows. The new case for no-rollout should be fast (under 2 seconds).

### Pattern 5: E2e Mutation Flow with Scripts (TEST-11)

The `test/e2e_test.go` `Test` function already tests mutation without scripts. Add a new sub-test that:
1. Creates pre/post scripts using `makeTempScript`-style helper (copy `server.makeTempScript` pattern or add a `test/` local helper)
2. Creates `server.NewProtoconfMutationServer` with those scripts set
3. Calls `MutateConfig` via the gRPC client
4. Asserts the response `PreScriptDuration` and `PostScriptDuration` are non-nil and > 0

The `makeTempScript` helper is defined in `server/server_test.go` (package-private). Either duplicate it in the `test` package or add a test helper in `test/helpers_test.go`.

```go
// test/e2e_test.go — new sub-test inside Test()
t.Run("mutation_with_scripts", func(t *testing.T) {
    preScript := makeTempScript(t, "exit 0")
    postScript := makeTempScript(t, "exit 0")
    srv, err := server.NewProtoconfMutationServer(protoconfRoot)
    require.NoError(t, err)
    srv.PreMutationScript = preScript
    srv.PostMutationScript = postScript

    var mutClient protoconf_pb.ProtoconfMutationServiceClient
    closer := TestServer(ctx, func(s *grpc.Server) {
        protoconf_pb.RegisterProtoconfMutationServiceServer(s, srv)
    }, func(conn *grpc.ClientConn) {
        mutClient = protoconf_pb.NewProtoconfMutationServiceClient(conn)
    })
    defer closer()

    resp, err := mutClient.MutateConfig(ctx, &protoconf_pb.ConfigMutationRequest{
        Path:  "mutation_test",
        Value: &protoconf_pb.ProtoconfValue{ProtoFile: "test.proto"},
    })
    require.NoError(t, err)
    assert.NotNil(t, resp.PreScriptDuration)
    assert.NotNil(t, resp.PostScriptDuration)
})
```

### Pattern 6: TLS-Enabled gRPC E2e (TEST-12)

The `test/e2e.go` `TestServer` uses `bufconn` and `insecure.NewCredentials()`. For TLS e2e, wire a real TCP listener (or use bufconn with TLS — bufconn supports overlaying TLS on the in-memory conn).

The approach used in `utils/tls_test.go` generates a self-signed cert with `ecdsa.GenerateKey` and registers `127.0.0.1` as a SAN. The gRPC server uses `grpc.Creds(credentials.NewTLS(serverTLS))` and the client uses `grpc.WithTransportCredentials(credentials.NewTLS(clientTLS))` where `clientTLS.RootCAs` is the server cert pool.

**Key point:** bufconn does not do hostname verification by default. For a real TLS handshake that verifies SANs, bind to `localhost:0` (net.Listen) and let the OS assign a port.

```go
func TestTLSMutationServer(t *testing.T) {
    certPEM, keyPEM := generateSelfSignedCert(t)  // from utils/tls_test.go pattern
    // server side
    serverTLS, err := utils.BuildTLSConfig(utils.TLSFiles{CertText: string(certPEM), KeyText: string(keyPEM)})
    require.NoError(t, err)
    srv, err := server.NewProtoconfMutationServer(protoconfRoot)
    require.NoError(t, err)
    lis, err := net.Listen("tcp", "127.0.0.1:0")
    require.NoError(t, err)
    rpcServer := grpc.NewServer(grpc.Creds(credentials.NewTLS(serverTLS)))
    protoconf_pb.RegisterProtoconfMutationServiceServer(rpcServer, srv)
    go rpcServer.Serve(lis)
    defer rpcServer.Stop()

    // client side
    pool := x509.NewCertPool()
    pool.AppendCertsFromPEM(certPEM)
    clientTLS := &tls.Config{RootCAs: pool, ServerName: "127.0.0.1"}
    conn, err := grpc.NewClient(lis.Addr().String(),
        grpc.WithTransportCredentials(credentials.NewTLS(clientTLS)))
    require.NoError(t, err)
    client := protoconf_pb.NewProtoconfMutationServiceClient(conn)

    _, err = client.MutateConfig(context.Background(), &protoconf_pb.ConfigMutationRequest{
        Path:  "mutation_test",
        Value: &protoconf_pb.ProtoconfValue{ProtoFile: "test.proto"},
    })
    require.NoError(t, err)
}
```

The `generateSelfSignedCert` helper can be shared via a file in the `test` package or the `utils` package. Since `utils/tls_test.go` defines it as a package-private test helper, duplicate it in `test/tls_helpers_test.go`.

### Pattern 7: Token-Based Auth E2e (TEST-13)

`bearerTokenInterceptor` is package-private in `server`. The `cliCommand.run()` wires it via `grpc.ChainUnaryInterceptor`. For e2e testing without relying on internals, create the gRPC server manually with the interceptor.

To access `bearerTokenInterceptor` from the `test` package, the interceptor must be exported or the test must be in `package server_test`. The simplest approach is to write the e2e auth test directly in `server/server_test.go` (which is already in `package server`) using an in-process gRPC server.

Alternatively, `ProtoconfMutationServer.config.AuthToken` is accessible (exported field via `config` struct). The server already applies the interceptor when run via `cliCommand.run()`. For an isolated e2e test:

```go
// In test/e2e_test.go (or server/server_test.go)
t.Run("auth_rejection", func(t *testing.T) {
    srv, err := server.NewProtoconfMutationServer(protoconfRoot)
    require.NoError(t, err)
    rpcServer := grpc.NewServer(
        grpc.ChainUnaryInterceptor(server.BearerTokenInterceptor("secret-token")),
    )
    // ... register and call with wrong token
    // assert codes.Unauthenticated
})
```

The problem: `bearerTokenInterceptor` is unexported. **Resolution options:**
1. Export it as `BearerTokenInterceptor` — small surface addition, allows the test package to use it directly.
2. Keep it unexported and test via `cliCommand.run()` or the full server startup — but that requires a live TCP port and signal handling.
3. Write the e2e auth test in `package server` (same file as `server_test.go`) — already done for unit tests; just add a new test function.

**Recommended:** write the auth e2e test in `test/e2e_test.go` by building the gRPC server with an explicit `grpc.ChainUnaryInterceptor` that mirrors what `cliCommand.run()` does, but using a locally-defined token-check interceptor in the test file. This avoids exporting internal server symbols.

```go
// test/e2e_test.go
func makeTokenInterceptor(token string) grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, h grpc.UnaryHandler) (interface{}, error) {
        if token == "" { return h(ctx, req) }
        md, ok := metadata.FromIncomingContext(ctx)
        if !ok { return nil, status.Error(codes.Unauthenticated, "no metadata") }
        vals := md.Get("authorization")
        if len(vals) == 0 { return nil, status.Error(codes.Unauthenticated, "no auth header") }
        tok := strings.TrimPrefix(vals[0], "Bearer ")
        if subtle.ConstantTimeCompare([]byte(tok), []byte(token)) != 1 {
            return nil, status.Error(codes.Unauthenticated, "bad token")
        }
        return h(ctx, req)
    }
}
```

Then in the test:
```go
t.Run("auth_acceptance", func(t *testing.T) { /* send correct token, expect success */ })
t.Run("auth_rejection", func(t *testing.T) { /* send wrong/missing token, expect Unauthenticated */ })
```

### Anti-Patterns to Avoid

- **Duplicating `bearerTokenInterceptor` logic** in the test without `subtle.ConstantTimeCompare`: use the same constant-time comparison.
- **Using `testing.Short()` skip for TLS/auth tests**: these tests should run in normal CI (not just `-short`). The main e2e `Test` function already skips with `-short`; the new sub-tests can be independent functions that always run.
- **Writing to `/tmp` directly** for script files: use `t.TempDir()` so cleanup is automatic.
- **Calling `srv.config.AuthToken = ...` directly**: the `config` field is unexported in `ProtoconfMutationServer`; set it via `s.config = &ServerConfig{AuthToken: "..."}` which is done in existing tests because the test is in `package server`.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Self-signed cert generation | Custom ASN.1 encoding | Pattern from `utils/tls_test.go` `generateSelfSignedCert` | Already handles ECDSA P-256, SAN, valid duration |
| In-process gRPC server | Custom TCP mock | `bufconn.Listen` + `TestServer` from `test/e2e.go` | Already handles lifecycle, cleanup, passthrough dialer |
| TLS on real TCP | Custom TLS handshake | `net.Listen("tcp","127.0.0.1:0")` + `grpc.Creds(credentials.NewTLS(...))` | Standard gRPC TLS pattern; port 0 avoids conflicts |
| Proto message fixtures | Custom serialization | `testdata.SmallTestDir()` + existing `.materialized_JSON` files | Provides complete isolated workspace with git history |

**Key insight:** All infrastructure for TLS testing and auth testing already exists in the codebase. Phase 10 is about using it, not building it.

## Common Pitfalls

### Pitfall 1: Rollout Test Race Conditions
**What goes wrong:** `TestProtoconfKVAgentRollout_SubscribeForConfig` runs goroutines concurrently and uses `t.Run` inside goroutines. If the outer test function returns before goroutines finish, `t.Errorf` panics with "called after test finished".
**Why it happens:** The test iterates `args.updates`, calling `inserter.InsertConfig` with `time.Sleep(2s)` between steps. The goroutines for each `want` channel run concurrently. The main test loop ends before goroutines do.
**How to avoid:** Add a `sync.WaitGroup` to the outer test, increment for each goroutine, `wg.Done()` at the end of each goroutine body, and `wg.Wait()` after the update loop. The existing test already works (all 3 channels pass), but new test cases must follow the same goroutine pattern.
**Warning signs:** `panic: testing: t.Run called after test finished` in test output.

### Pitfall 2: TLS ServerName Mismatch
**What goes wrong:** TLS dial fails with `x509: certificate is valid for 127.0.0.1, not localhost`.
**Why it happens:** `generateSelfSignedCert` registers `127.0.0.1` as an IP SAN. Connecting to `"localhost:PORT"` uses hostname `localhost`, not the IP.
**How to avoid:** Use `lis.Addr().String()` which returns `"127.0.0.1:PORT"`. Set `clientTLS.ServerName = "127.0.0.1"` explicitly or ensure the dial address is `127.0.0.1`.
**Warning signs:** `x509: certificate is not valid for any names` or `certificate signed by unknown authority`.

### Pitfall 3: Parser Test FileDescriptor Assertions
**What goes wrong:** `fds[0].GetName()` returns the full proto path (e.g., `"test.proto"`) but `GetFullyQualifiedName()` is not a method on `*desc.FileDescriptor` — use `GetName()` instead.
**Why it happens:** `jhump/protoreflect/desc.FileDescriptor` has `GetName()` (base path), `GetPackage()` (package name), `GetMessageTypes()` (messages). Not `GetFullyQualifiedName()` which is on `desc.Descriptor` (message/field/etc.), not file descriptors.
**How to avoid:** Use `assert.Equal(t, "test.proto", fds[0].GetName())` and `assert.Equal(t, "test.v1", fds[0].GetPackage())`.

### Pitfall 4: Script Duration Zero on Fast Tests
**What goes wrong:** `resp.PreScriptDuration.AsDuration() > 0` assertion fails when the script runs so fast the duration rounds to zero.
**Why it happens:** `durationpb` stores nanosecond precision, but fast scripts may return sub-nanosecond durations in test environments.
**How to avoid:** Assert `resp.PreScriptDuration != nil` (non-nil is the real assertion — it proves the script ran and `StoreReport` was called). Duration > 0 is fragile; skip it.

### Pitfall 5: `load_remote_with_load_local.pconf` Fails in E2e Test
**What goes wrong:** The existing e2e `Test` function calls `c.CompileFile("load_remote_with_load_local.pconf")` and it fails with `open .../vizceral_repo/src/services/frontend.pinc: no such file or directory`.
**Why it happens:** The `mod tidy` step in the test calls `ms.Init` and `ms.Sync` but the vizceral cache may not be fully populated on the first run. This is a pre-existing flaky failure unrelated to Phase 10.
**How to avoid:** New e2e sub-tests (TEST-11/12/13) should be independent functions not depending on `load_remote_with_load_local`. Use `mutation_test` or `test` paths only. Do not call `c.CompileFile("load_remote_with_load_local.pconf")` in the new tests.

### Pitfall 6: Inserter `metadata.Commit` Slice Panic
**What goes wrong:** `inserter.go` line 261 calls `metadata.Commit[0:8]` directly. If metadata has a short commit hash, this panics.
**Why it happens:** `GatherMetadata` returns a real git commit hash (40 chars) for git repos. For non-git, it returns `"not_a_git_repo"` (14 chars, safe for `[0:8]`). New test cases must use `testdata.SmallTestDir()` which initializes a git repo via `go-git`.
**How to avoid:** Always use `testdata.SmallTestDir()` as the `protoconfRoot` for inserter tests; this guarantees a git repo with real commit hashes.

## Code Examples

### Verified: MutateConfig Response Fields
```go
// Source: server/server.go lines 398-477, StoreReport at lines 386-390
// PreScriptDuration is set only when PreMutationScript != "" and succeeds
// PostScriptDuration is set only when PostMutationScript != "" and succeeds
// Uuid is always populated as uuid.NewString()
resp, err := s.MutateConfig(ctx, req)
require.NoError(t, err)
require.NotNil(t, resp)
// Script duration assertions:
assert.NotNil(t, resp.PreScriptDuration)
assert.NotNil(t, resp.PostScriptDuration)
```

### Verified: ReportProgress Returns Merged Response
```go
// Source: server/server.go lines 479-488
// ReportProgress merges the incoming response into the stored one
// For test: store a report, call ReportProgress, assert Uuid matches
s.reports.Store("test-uuid", &protoconf_pb.ConfigMutationResponse{Uuid: "test-uuid"})
got, err := s.ReportProgress(ctx, &protoconf_pb.ConfigMutationResponse{Uuid: "test-uuid"})
require.NoError(t, err)
assert.Equal(t, "test-uuid", got.Uuid)
```

### Verified: GenReflectionUI Sets httpServer.Handler
```go
// Source: server/server.go line 616
// GenReflectionUI sets httpServer.Handler to an h2c mux
httpServer := &http.Server{}
err := server.GenReflectionUI(ctx, rpcServer, httpServer)
require.NoError(t, err)
assert.NotNil(t, httpServer.Handler)
```

### Verified: Parser FileDescriptor Name
```go
// Source: compiler/lib/parser/parser.go + jhump/protoreflect/desc
fds, err := p.ParseFilesX("test.proto")
require.NoError(t, err)
require.Len(t, fds, 1)
assert.Equal(t, "test.proto", fds[0].GetName())
assert.Equal(t, "test.v1", fds[0].GetPackage())
```

### Verified: ReadConfig Populates ProtoconfValue
```go
// Source: utils/testdata/small/materialized_config/test.materialized_JSON
// {"protoFile":"test.proto","value":{"@type":"type.googleapis.com/test.v1.TestMessage","stringValue":"Im here"}}
msg := &protoconf_pb.ProtoconfValue{}
err = p.ReadConfig(filepath.Join(protoconfRoot, "materialized_config/test.materialized_JSON"), msg)
require.NoError(t, err)
assert.Equal(t, "test.proto", msg.ProtoFile)
assert.NotNil(t, msg.Value)
assert.Contains(t, msg.Value.TypeUrl, "test.v1.TestMessage")
```

### Verified: Inserter Writes Three KV Keys
```go
// Source: inserter/inserter.go lines 334-385
// XXXinsertVersion writes: {prefix}/{configName}/{version}/config.data
//                           {prefix}/{configName}/{version}/config.json
//                           {prefix}/{configName}/{version}/metadata.json
// For Stable: {prefix}/{configName}/config.data etc.
for _, suffix := range []string{"config.data", "config.json", "metadata.json"} {
    v, err := kvStore.Get(ctx, "test/"+suffix, &store.ReadOptions{})
    assert.NoError(t, err)
    assert.NotEmpty(t, v.Value)
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `grpc.WithInsecure()` | `grpc.WithTransportCredentials(insecure.NewCredentials())` | Phase 1 | All test code must use the new form |
| `grpc.Dial` | `grpc.NewClient` | Phase 1 | `grpc.NewClient` requires non-empty DNS-resolvable target; use `passthrough:///bufnet` for bufconn |
| Direct `os.Exit` in libs | error returns propagated to CLI | Phase 2 | Test can now test error paths without subprocess |

## Open Questions

1. **Should the TLS e2e test use bufconn or a real TCP listener?**
   - What we know: bufconn supports overlaying TLS, but hostname verification is tricky; real TCP with `localhost:0` is simpler and more representative.
   - What's unclear: whether CI has constraints on binding to random ports.
   - Recommendation: Use `net.Listen("tcp", "127.0.0.1:0")` (port 0 = OS-assigned port); this works in all CI environments.

2. **Should `bearerTokenInterceptor` be exported to enable e2e auth test from `test` package?**
   - What we know: Currently unexported. The `Test_bearerTokenInterceptor` unit test in `server/server_test.go` (package `server`) tests it thoroughly.
   - What's unclear: whether a full e2e auth test in `test/` is worth exporting the function vs. just testing from within `package server`.
   - Recommendation: Write the e2e auth test within `package server` to avoid exporting internals. Add `TestE2E_Auth` to `server/server_test.go` using a real gRPC server with the interceptor applied, verifying both acceptance and rejection over the wire.

## Environment Availability

Step 2.6: SKIPPED — this phase adds test code only; no new external tools, services, or CLIs are required. All dependencies (`grpc`, `testify`, `bufconn`) are already in `go.mod`.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go standard `testing` + `github.com/stretchr/testify` v1.9.0 |
| Config file | none (Go test runner) |
| Quick run command | `go test ./server/... ./compiler/lib/parser/... ./inserter/... ./agent/... -run "Test_server_MutateConfig|TestParser|TestProtoconfInserter|TestProtoconfKVAgentRollout" -timeout 60s` |
| Full suite command | `go test ./... -timeout 300s` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| TEST-07 | MutateConfig response has Uuid + script durations when scripts run | unit | `go test ./server/... -run Test_server_MutateConfig -v` | Yes (assertions missing) |
| TEST-08 | ParseFilesX returns descriptor with correct name/package; ReadConfig populates fields | unit | `go test ./compiler/lib/parser/... -run TestParser -v` | Yes (assertions missing) |
| TEST-09 | InsertConfig writes config.data, config.json, metadata.json to KV store | unit | `go test ./inserter/... -run TestProtoconfInserter -v` | Yes (assertions missing) |
| TEST-10 | Rollout subscription delivers correct values per channel including no-rollout case | unit | `go test ./agent/... -run TestProtoconfKVAgentRollout -v -timeout 60s` | Yes (new case needed) |
| TEST-11 | Full mutation flow with pre/post scripts via gRPC | integration | `go test ./test/... -run TestMutationWithScripts -v` | No — Wave 0 |
| TEST-12 | TLS-enabled gRPC connection completes mutation successfully | integration | `go test ./test/... -run TestTLSMutation -v` (or `./server/... -run TestTLSE2E`) | No — Wave 0 |
| TEST-13 | Token auth: valid token accepted, invalid/missing rejected with Unauthenticated | integration | `go test ./test/... -run TestAuthFlow -v` (or `./server/... -run TestE2EAuth`) | No — Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./server/... ./compiler/lib/parser/... ./inserter/... ./agent/... -timeout 60s`
- **Per wave merge:** `go test ./... -timeout 300s`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `test/TestMutationWithScripts` or equivalent — covers TEST-11 — file `test/e2e_mutation_scripts_test.go` (new)
- [ ] `test/TestTLSMutation` or `server/TestTLSE2E` — covers TEST-12 — file to be decided
- [ ] `test/TestAuthFlow` or `server/TestE2EAuth` — covers TEST-13 — file to be decided
- [ ] `test/tls_helpers_test.go` — shared `generateSelfSignedCert` for TLS tests (or import from utils)

*(TEST-07 through TEST-10 use existing test files — no new files needed, only assertion additions)*

## Sources

### Primary (HIGH confidence)
- Direct code reading of `server/server.go`, `server/server_test.go` — `MutateConfig` return type, `StoreReport` behavior, `bearerTokenInterceptor` signature
- Direct code reading of `compiler/lib/parser/parser.go`, `compiler/lib/parser/parser_test.go` — `ParseFilesX` return type, `ReadConfig` behavior
- Direct code reading of `inserter/inserter.go`, `inserter/inserter_test.go` — KV key paths written by `XXXinsertVersion`
- Direct code reading of `agent/kv_agent_rollout_impl_test.go` — goroutine pattern and existing test structure
- Direct code reading of `test/e2e_test.go`, `test/e2e.go` — `TestServer` helper, existing e2e structure
- Direct code reading of `utils/tls_test.go` — `generateSelfSignedCert` helper pattern
- Live test runs confirming current pass/fail state of all affected tests

### Secondary (MEDIUM confidence)
- `utils/testdata/small/materialized_config/test.materialized_JSON` — known fixture values used for assertion targets
- `utils/testdata/small/src/test.proto` — proto package/message names for parser assertions

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all libraries already in `go.mod` and used in adjacent tests
- Architecture: HIGH — all patterns taken directly from existing test code
- Pitfalls: HIGH — identified by running tests and reading code; TLS SAN pitfall from `generateSelfSignedCert` implementation
- Placeholder fix targets: HIGH — verified by grep and code reading that TODOs exist exactly where noted

**Research date:** 2026-03-31
**Valid until:** 2026-04-30 (stable dependencies; no fast-moving ecosystem concerns)
