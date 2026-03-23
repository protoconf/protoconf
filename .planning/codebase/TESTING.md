# Testing Patterns

**Analysis Date:** 2026-03-23

## Test Framework

**Runner:**
- Go standard `testing` package
- No third-party test runner

**Assertion Library:**
- `github.com/stretchr/testify/assert` -- non-fatal assertions
- `github.com/stretchr/testify/require` -- fatal assertions (test stops on failure)
- Mixed usage: some tests use raw `t.Errorf()` instead of testify (especially older/generated tests)

**Run Commands:**
```bash
go test ./...                                          # Run all tests
go test -race -coverprofile=coverage.txt ./...         # With race detection and coverage
go test -v ./compiler/lib/...                          # Run specific package tests
go test -short ./...                                   # Skip long-running e2e tests
go test -run TestCompiler_CompileFile ./compiler/lib/  # Run specific test
go test -bench=. ./agent/                              # Run benchmarks
```

## Test File Organization

**Location:**
- Co-located with source code (standard Go convention)
- Exception: `test/e2e_test.go` and `test/e2e.go` for end-to-end integration tests

**Naming:**
- `<source>_test.go` pattern: `compiler.go` -> `compiler_test.go`
- Test functions: `Test<FunctionName>` or `Test_<unexportedFunc>_<Method>`

**Structure:**
```
compiler/
  command.go
  command_test.go
compiler/lib/
  compiler.go
  compiler_test.go
  module_service.go
  module_service_test.go
compiler/lib/parser/
  parser.go (assumed)
  parser_test.go
agent/
  agent.go
  agent_test.go
  kv_agent_impl.go
  kv_agent_impl_test.go
  kv_agent_rollout_impl.go (assumed)
  kv_agent_rollout_impl_test.go
  command.go (assumed)
  command_test.go
server/
  server.go
  server_test.go
inserter/
  inserter.go (assumed)
  inserter_test.go
importers/
  importers_test.go
utils/
  utils_test.go
test/
  e2e.go           # Test helpers (exported TestServer function)
  e2e_test.go       # Full end-to-end integration test
```

## Test Structure

**Table-Driven Tests (primary pattern):**
```go
func Test_server_MutateConfig(t *testing.T) {
    type fields struct {
        config        *cliConfig
        protoconfRoot string
    }
    type args struct {
        ctx context.Context
        in  *protoconf_pb.ConfigMutationRequest
    }
    tests := []struct {
        name    string
        fields  fields
        args    args
        want    *protoconf_pb.ConfigMutationResponse
        wantErr error
    }{
        {
            name: "test no workspace",
            fields: fields{ ... },
            args: args{ ... },
            wantErr: nil,
        },
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test body using tt.fields, tt.args, tt.wantErr
        })
    }
}
```
Files: `server/server_test.go`, `inserter/inserter_test.go`, `compiler/lib/module_service_test.go`, `agent/command_test.go`

**Subtest with Helper Function Pattern:**
```go
func TestCompiler_CompileFile(t *testing.T) {
    // setup phase
    t.Run("Prepare", func(t *testing.T) { ... })
    t.Run("NewModuleService", func(t *testing.T) { ... })
    t.Run("Setup", func(t *testing.T) { ... })

    // test cases using helper
    t.Run("test.pconf", compilerTest(c, nil))
    t.Run("validator_test.pconf", compilerTest(c, ErrInvalidConfig))
}

func compilerTest(c *Compiler, wantErr error) func(*testing.T) {
    return func(t *testing.T) {
        err := c.CompileFile(filepath.Base(t.Name()))
        assert.Truef(t, errors.Is(err, wantErr), "err = %v, want = %v", err, wantErr)
    }
}
```
File: `compiler/lib/compiler_test.go`

## Test Data Management

**Embedded Test Data:**
- Test data lives in `utils/testdata/` with subdirectories: `small/`, `large/`, `bad_proto/`
- Embedded using Go's `embed.FS` in `utils/testdata/embed.go`:
  ```go
  //go:embed bad_proto large small
  var TestData embed.FS
  ```

**Test Directory Creation:**
- `testdata.SmallTestDir()` creates a temporary directory, copies embedded test data, and initializes a git repo
- Used across nearly all test files as the standard test fixture
- Each call creates a fresh copy -- tests are isolated
- Files: `utils/testdata/embed.go`

**Usage across tests:**
```go
protoconfRoot := testdata.SmallTestDir()
// ... use protoconfRoot as protoconf workspace
```
Packages using this: `compiler/lib`, `compiler`, `agent`, `server`, `inserter`, `utils`, `test`

## Mocking

**Framework:** No dedicated mocking framework (no gomock, mockgen, etc.)

**Patterns:**
- **In-memory KV store:** `agent/dummykv` package provides a fake key-value store for testing:
  ```go
  storeClient, _ := dummykv.New(ctx, []string{}, &dummykv.Config{})
  ```
  Used in: `agent/kv_agent_impl_test.go`, `agent/kv_agent_rollout_impl_test.go`, `inserter/inserter_test.go`, `test/e2e_test.go`

- **In-memory gRPC server with bufconn:** Tests create real gRPC servers using `google.golang.org/grpc/test/bufconn`:
  ```go
  buffer := 101024 * 1024
  lis := bufconn.Listen(buffer)
  baseServer := grpc.NewServer()
  protoconf_pb.RegisterProtoconfServiceServer(baseServer, srv)
  go func() { baseServer.Serve(lis) }()
  conn, _ := grpc.DialContext(ctx, "",
      grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
          return lis.Dial()
      }), grpc.WithTransportCredentials(insecure.NewCredentials()))
  ```
  Helper in `agent/kv_agent_impl_test.go` as `testServer()` and in `test/e2e.go` as exported `TestServer()`

- **File-based KV store:** `agent/filekv` provides a filesystem-backed store for dev/test mode

**What to Mock:**
- External KV stores (consul, etcd, zookeeper) are replaced with `dummykv`
- Network connections are replaced with `bufconn`

**What NOT to Mock:**
- The compiler, parser, and proto registry are used as real instances
- File I/O uses real temporary directories via `testdata.SmallTestDir()`

## Coverage

**Requirements:** No minimum threshold enforced

**CI Coverage:**
- Codecov integration via GitHub Actions (`.github/workflows/go.yml`)
- Coverage profile generated with: `go test -race -coverprofile=coverage.txt -covermode=atomic -v ./...`
- Uploaded to Codecov with token
- `codecov.yml` ignores generated proto files: `**/*.pb.go`

**View Coverage:**
```bash
go test -coverprofile=coverage.txt ./...
go tool cover -html=coverage.txt           # Open HTML report
go tool cover -func=coverage.txt           # Text summary
```

## Test Types

**Unit Tests:**
- Most test files contain unit tests for individual functions/methods
- Table-driven tests for CLI commands, parsers, inserters
- Compiler tests validate individual `.pconf` file compilation against expected error types
- Files: `compiler/lib/compiler_test.go`, `compiler/lib/module_service_test.go`, `compiler/lib/parser/parser_test.go`, `utils/utils_test.go`, `importers/importers_test.go`

**Integration Tests:**
- Agent tests spin up real gRPC servers with in-memory stores
- Server tests exercise full mutation flow including script execution
- Files: `agent/kv_agent_impl_test.go`, `agent/kv_agent_rollout_impl_test.go`, `server/server_test.go`

**End-to-End Tests:**
- `test/e2e_test.go` runs a full workflow: compile -> insert -> subscribe -> mutate -> recompile -> verify update
- Skipped in short mode: `if testing.Short() { t.Skip("skipping test in short mode.") }`
- Exercises: compiler, agent (dev + production), inserter, mutation server, gRPC streaming

**Benchmarks:**
- `BenchmarkProtoconfAgent` in `agent/kv_agent_impl_test.go` benchmarks subscription throughput

## Common Patterns

**Context-based test timeouts:**
```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
```

**Error checking with sentinel errors:**
```go
assert.Truef(t, errors.Is(err, wantErr), "err = %v, want = %v", err, wantErr)
```

**Proto equality checking:**
```go
if !proto.Equal(got, expected) {
    t.Errorf("expected \n%s, got \n%s", expected, got)
}
```

**Async testing with channels (rollout tests):**
```go
func recvCh(ctx context.Context, watcher protoconf_pb.ProtoconfService_SubscribeForConfigClient) chan *protoconf_pb.ConfigUpdate {
    ch := make(chan *protoconf_pb.ConfigUpdate)
    go func() {
        for {
            select {
            case <-ctx.Done(): return
            default:
                item, err := watcher.Recv()
                if err != nil { return }
                ch <- item
            }
        }
    }()
    return ch
}
```
File: `agent/kv_agent_rollout_impl_test.go`

## CI Test Configuration

**GitHub Actions workflow:** `.github/workflows/go.yml`
- Trigger: push to `main`, PRs to `main`
- Go version: 1.22
- Steps: build -> test with race detection and coverage -> report -> upload to Codecov
- Test output format: JSON piped to `gotestsum` for testdox format
- Test results archived as GitHub artifact

**Key CI command:**
```bash
go test -race -coverprofile=coverage.txt -covermode=atomic -v ./... -json > test_results.json
```

## Test Gaps

**Packages without test files:**
- `consts/` -- only constants, low risk
- `datatypes/` -- proto-generated code
- `devserver/` -- no tests detected
- `mod/` -- no tests detected
- `mutate/` -- no tests detected
- `fmt/` -- newly added `protoconf fmt` command, no tests
- `command/` -- CLI utilities, no tests
- `web_ide/` -- no tests detected

**Incomplete test patterns:**
- Multiple test files contain `// TODO: Add test cases.` comments: `server/server_test.go`, `compiler/lib/parser/parser_test.go`, `inserter/inserter_test.go`, `agent/kv_agent_rollout_impl_test.go`
- Several tests have placeholder assertions: `// Add assertions here to verify the expected behavior`
- `server/server_test.go` line 139: `// TODO(smintz): assert the response`

---

*Testing analysis: 2026-03-23*
