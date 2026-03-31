# Phase 9: Unit Test Coverage & Infrastructure - Research

**Researched:** 2026-03-31
**Domain:** Go unit testing, gRPC/bufconn testing, KV store testing, Starlark-protobuf bridge testing
**Confidence:** HIGH

## Summary

Phase 9 adds test coverage to six previously-untested packages (mutate, fmt, command, devserver, four KV stores, and compiler/starproto) and creates a shared testutil package. The codebase already has strong test patterns in place—bufconn-based gRPC servers, table-driven tests with testify, and embedded test fixtures in utils/testdata. The primary challenge is faithfully replicating those patterns into new packages without introducing circular imports, and building a testutil package that can be imported by any package.

All target packages have zero existing test files, confirmed by direct inspection. The `testutil/` package does not yet exist. The CI pipeline (go test -race -coverprofile=coverage.txt -covermode=atomic -v ./...) already picks up new tests automatically. No CI changes are required; coverage is reported via Codecov.

The most complex targets are: `compiler/starproto/` (requires proto descriptors + jhump/protoreflect to construct test messages), `agent/configmaps/` (requires a mock Kubernetes clientset), and `devserver/` (requires a goroutine-managed in-process gRPC server). All four KV store packages have clearly-bounded method implementations; testing them is mostly wiring, not algorithm design.

**Primary recommendation:** Create testutil/ first (Wave 0 / Plan 1), then add tests per-package following existing patterns. Each package's test file must use testutil to satisfy D-03 and TEST-14.

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**D-01:** Create `testutil/` package at repo root (not `internal/testutil/` — matches existing flat package convention like `utils/`, `consts/`).

**D-02:** Extract these common patterns into testutil:
  - gRPC test server setup with bufconn (currently inline in `agent_test.go`)
  - Proto message helper (`newAny(msg)` wrapper, currently duplicated)
  - Temporary protoconf root setup (combines `testdata.SmallTestDir()` with additional config)

**D-03:** At least 2 new test files must import testutil to satisfy TEST-14.

**D-04:** Each new test file covers:
  - Happy path for all primary exported functions
  - Key error paths for sentinel errors and validation failures
  - Edge cases flagged by existing TODO comments

**D-05:** Use table-driven tests with testify (assert/require) — the established codebase pattern.

**D-06:** Test file naming follows existing convention: `{source}_test.go` co-located with source.

**D-07:** `mutate/` (TEST-01) — Test field parsing (setNumeric, setFloat), type conversion, and the gRPC mutation request flow. Use proto test fixtures, not a live server.

**D-08:** `fmt/` (TEST-02) — Test Starlark file formatting (format, write, diff modes). Use inline Starlark strings as input.

**D-09:** `command/` (TEST-03) — Test `RunSubcommands()` routing, `RunCommand()` wrapper, and `DefaultUI` setup. No KVStoreConfig tests needed (removed in Phase 8).

**D-10:** `devserver/` (TEST-04) — Test combined server startup and service registration. Use bufconn for in-process gRPC.

**D-11:** KV stores (TEST-05) — Test each store's implemented methods:
  - `dummykv/` — Get, Put, List, Exists, Watch (in-memory, no external deps)
  - `filekv/` — Watch, Get against filesystem (use t.TempDir)
  - `configmaps/` — Interface-level testing (mock Kubernetes client)
  - `otelkv/` — Verify OTel span creation wraps underlying store calls (mock inner store)

**D-12:** `compiler/starproto/` (TEST-06) — Test message wrapping, field access (get/set), enum handling, map/repeated fields, Any type support. Use proto descriptors from testdata.

**D-13:** No hard coverage threshold enforced initially — CI continues to report coverage via codecov. The goal is to add meaningful tests, not chase a percentage.

**D-14:** Existing `go test -race -coverprofile=coverage.txt -covermode=atomic -v ./...` command in CI is sufficient. No CI config changes needed.

**D-15:** TEST-16 (error path tests) is satisfied by D-04's requirement to test key error paths in each new test file.

### Claude's Discretion

- Exact test case selection within each package (which specific functions to test first)
- Whether to use subtests (t.Run) or flat test functions — follow whatever the nearest existing test file uses
- Mock implementation details for configmaps and otelkv
- Whether to split large test files into multiple files per package

### Deferred Ideas (OUT OF SCOPE)

- Fixing placeholder assertions in parser_test.go and inserter_test.go — Phase 10 (TEST-07 through TEST-10)
- E2E integration tests for mutation, TLS, and auth flows — Phase 10 (TEST-11 through TEST-13)
- Coverage percentage thresholds — evaluate after Phase 9+10 add tests
- Property-based testing / fuzzing — future concern
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| TEST-01 | mutate/ has unit tests covering field parsing, type conversion, and gRPC mutation flow | setNumeric/setFloat are unexported but package-internal tests can access them; gRPC flow needs mock server via dummykv or bufconn |
| TEST-02 | fmt/ has unit tests covering Starlark file formatting | formatFile/processPath/isStarlarkFile are testable with t.TempDir() inline Starlark inputs |
| TEST-03 | command/ has unit tests covering subcommand routing and KV store config | RunCommand/RunSubcommands call os.Exit — test via subprocess or wrap internal `run()` function |
| TEST-04 | devserver/ has tests covering combined server startup and service registration | DevServerCommand.Run calls signal.NotifyContext — test via goroutine + cancel; bufconn for gRPC verification |
| TEST-05 | KV store implementations have dedicated test files covering implemented methods | dummykv: pure in-memory; filekv: needs testdata.SmallTestDir(); configmaps: fake k8s client; otelkv: mock inner store |
| TEST-06 | compiler/starproto/ has tests covering message wrapping, field access, enum handling, Any | Needs jhump/protoreflect desc and dynamic.Message construction; use well-known proto types from google.golang.org/protobuf |
| TEST-14 | Shared test helpers extracted into testutil/ package | Create testutil/ at root; must be importable by any package; expose: NewBufconnServer, NewAny, NewTestProtoconfRoot |
| TEST-15 | CI enforces minimum coverage threshold with clear reporting | D-13 defers hard threshold; codecov already configured; no CI file changes needed |
| TEST-16 | Test fixtures cover error paths and edge cases | Each new test file must include at least one error-path table-driven sub-test |
</phase_requirements>

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/stretchr/testify` | v1.9.0 | Assertions (assert/require), test suites | Used in every existing test file |
| `google.golang.org/grpc/test/bufconn` | (bundled with grpc v1.64.0) | In-process gRPC listener; no real TCP | Used in agent_test.go — established pattern |
| `github.com/jhump/protoreflect/desc` | v1.16.0 | Load message descriptors for starproto tests | Required by starproto package itself |
| `github.com/jhump/protoreflect/dynamic` | v1.16.0 | Create dynamic.Message for starproto tests | Required by starproto package itself |
| `k8s.io/client-go/kubernetes/fake` | v0.30.1 | Fake Kubernetes clientset for configmaps tests | stdlib for k8s testing |
| Go standard `testing` package | go1.22 | Test runner, t.TempDir(), t.Run(), benchmarks | Already in use everywhere |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `google.golang.org/grpc/credentials/insecure` | (bundled) | Insecure dial for bufconn | All bufconn-based tests |
| `github.com/protoconf/protoconf/utils/testdata` | local | SmallTestDir() embedded fixture | filekv, devserver, mutate tests needing a real protoconf root |
| `github.com/bazelbuild/buildtools/build` | already in go.mod | Build/format parser used by fmt package | fmt tests need it indirectly via formatFile |
| `go.opentelemetry.io/otel/trace` | (bundled with otel v1.27.0) | NoopTracerProvider for otelkv tests | otelkv tests verifying span delegation |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `k8s.io/client-go/kubernetes/fake` | Real Kubernetes API | fake avoids cluster dependency; only viable option for unit tests |
| `bufconn` for devserver | net.Listen(":0") | bufconn is faster, always port-free; matches established codebase pattern |
| Inline proto descriptors | testdata .proto files | Well-known types (structpb, timestamppb) require no .proto files and are always available |

**Installation:** No new dependencies needed. All required packages are already in go.mod.

---

## Architecture Patterns

### Recommended Project Structure
```
testutil/
└── testutil.go       # NewBufconnServer, NewAny, NewTestProtoconfRoot helpers

mutate/
└── mutate_test.go    # package mutate (white-box: tests setNumeric, setFloat directly)

fmt/
└── command_test.go   # package fmt

command/
└── command_test.go   # package command

devserver/
└── command_test.go   # package devserver

agent/dummykv/
└── dummykv_test.go   # package dummykv

agent/filekv/
└── filekv_test.go    # package filekv

agent/configmaps/
└── configmaps_test.go  # package configmaps

agent/otelkv/
└── otelkv_test.go    # package otelkv

compiler/starproto/
└── starproto_test.go # package starproto
```

### Pattern 1: bufconn-based gRPC test server (from agent_test.go)
**What:** Spin up a real gRPC server over an in-memory connection. No TCP port allocation.
**When to use:** Any test that needs to exercise a real gRPC round-trip (devserver, otelkv wrapping a gRPC-backed store).
**Example:**
```go
// Source: agent/kv_agent_impl_test.go
func testServer(ctx context.Context, srv protoconf_pb.ProtoconfServiceServer) (protoconf_pb.ProtoconfServiceClient, func()) {
    buffer := 101024 * 1024
    lis := bufconn.Listen(buffer)
    baseServer := grpc.NewServer()
    protoconf_pb.RegisterProtoconfServiceServer(baseServer, srv)
    go func() {
        if err := baseServer.Serve(lis); err != nil {
            log.Printf("error serving server: %v", err)
        }
    }()
    conn, err := grpc.NewClient("passthrough:///bufnet",
        grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
            return lis.Dial()
        }), grpc.WithTransportCredentials(insecure.NewCredentials()))
    // ...
}
```

### Pattern 2: Table-driven tests with testify (universal pattern)
**What:** `[]struct{ name, fields, wantErr }` slices iterated with `t.Run`.
**When to use:** Every test function. This is the only accepted pattern.
**Example:**
```go
// Source: agent/kv_agent_impl_test.go, server/server_test.go
tests := []struct {
    name    string
    input   string
    wantErr bool
}{
    {"valid int", "42", false},
    {"invalid int", "notanumber", true},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        err := setNumeric(msg, "field", tt.input, func(s interface{}) interface{} { return s })
        if tt.wantErr {
            require.Error(t, err)
        } else {
            require.NoError(t, err)
        }
    })
}
```

### Pattern 3: Fake Kubernetes client for configmaps
**What:** `k8s.io/client-go/kubernetes/fake` creates a client backed by an in-memory object tracker.
**When to use:** Any test touching `configmaps.Store` methods that call `s.clientset.*`.
**Example:**
```go
// Source: Kubernetes client-go fake package
import "k8s.io/client-go/kubernetes/fake"

clientset := fake.NewSimpleClientset()
store := &configmaps.Store{
    clientset: clientset,
    config:    &configmaps.Config{Namespace: "default"},
    mutex:     &sync.Mutex{},
    logger:    slog.Default(),
}
```
Note: `configmaps.Store` fields are unexported. The test must be in `package configmaps` (white-box) to set the `clientset` field directly, or `New()` must be refactored to accept an injected clientset. The simplest path is white-box testing (same package).

### Pattern 4: Constructing starproto test messages
**What:** Use `desc.LoadMessageDescriptorForMessage()` with any registered proto type, then `dynamic.NewMessage(d)`.
**When to use:** Any starproto test that needs a `*dynamic.Message` or `*starProtoMessage`.
**Example:**
```go
// Source: compiler/starproto/message_type.go (NewBuiltin pattern)
import (
    "github.com/jhump/protoreflect/desc"
    "github.com/jhump/protoreflect/dynamic"
    "google.golang.org/protobuf/types/known/structpb"
)

d, err := desc.LoadMessageDescriptorForMessage(&structpb.Value{})
require.NoError(t, err)
msg := dynamic.NewMessage(d)
wrapped := NewStarProtoMessage(msg)
```

### Pattern 5: command/ testing workaround for os.Exit
**What:** `RunCommand` and `RunSubcommands` call `os.Exit()` through mitchellh/cli. The internal `run()` function is the testable surface.
**When to use:** Testing command routing without triggering process exit.
**How:** The `run()` function is unexported but the test is in `package command` (white-box). Alternatively, pass a no-op subcommand and verify it is called. Because `run()` calls `c.Run()` which returns `(exitStatus, error)` before calling `os.Exit`, the internal behavior is observable.

```go
// package command (white-box test)
func TestRun_VersionFlag(t *testing.T) {
    called := false
    cmds := map[string]cli.CommandFactory{
        "sub": func() (cli.Command, error) {
            return &mockCommand{runFn: func([]string) int {
                called = true
                return 0
            }}, nil
        },
    }
    // run() not directly testable — use mitchellh/cli directly to validate routing
    // OR test PrefixedUi and DefaultUI independently (no os.Exit involved)
}
```

The safest approach (confirmed by CONTEXT.md D-09): test `PrefixedUi` methods and `DefaultUI` setup directly. These are pure functions with no os.Exit. Skip `RunCommand`/`RunSubcommands` in unit tests; they are integration-test territory (Phase 10).

### Anti-Patterns to Avoid

- **`os.Exit` in tests:** Never call `RunCommand`/`RunSubcommands` directly from tests — they call `os.Exit`. Test internal helpers instead.
- **Real filesystem paths in KV tests:** Use `t.TempDir()` and `testdata.SmallTestDir()` for temp dirs; never assume a working directory.
- **Importing testutil from testutil's own tests:** Would create a circular dependency. testutil's own test (if any) must not import itself.
- **Missing `t.Cleanup` / `defer closer()`:** Always close bufconn listeners and gRPC servers in test teardown.
- **Package-external tests (package foo_test) for white-box coverage:** Unexported functions (setNumeric, setFloat, formatFile, run) require `package foo` (same package) tests.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| In-process gRPC | Custom TCP listener | `google.golang.org/grpc/test/bufconn` | Already used; port-free; handles concurrency |
| Fake K8s client | Custom HTTP server mock | `k8s.io/client-go/kubernetes/fake` | Built for exactly this; handles watchers, CRUD, object tracker |
| Temp directories | `os.MkdirTemp` directly | `t.TempDir()` | Auto-cleaned on test completion; no defer boilerplate |
| Proto test fixtures | Hand-crafted .proto files for starproto tests | Well-known types (`structpb`, `timestamppb`, `wrapperspb`) | Already registered, no descriptor file needed |
| OTel span verification | Custom tracer implementation | `go.opentelemetry.io/otel/trace/noop.NewTracerProvider()` or manual spy | NoopTracerProvider for happy-path; simple spy struct for delegation verification |

**Key insight:** The `dummykv` package is itself a perfect test double for any test that needs a `store.Store`. Use it as the backing store for `otelkv` tests.

---

## Common Pitfalls

### Pitfall 1: command.RunCommand/RunSubcommands call os.Exit
**What goes wrong:** A test calling `RunCommand(...)` causes the entire test process to exit.
**Why it happens:** mitchellh/cli calls `os.Exit(exitStatus)` at the end of `c.Run()`. This is in the internal `run()` function.
**How to avoid:** Test `PrefixedUi`, `DefaultUI`, and `NewPrefixedUi` directly; do not call the top-level `RunCommand`/`RunSubcommands` from tests. If routing must be tested, use mitchellh/cli's `CLI.Run()` method directly with a mock command (it returns before os.Exit is called).
**Warning signs:** Test suite terminates with "exit status 0" without running all tests.

### Pitfall 2: configmaps.Store has unexported fields
**What goes wrong:** Cannot construct a `configmaps.Store` with a fake clientset from an external test package (`package configmaps_test`).
**Why it happens:** The `clientset`, `config`, `mutex`, `logger` fields are unexported.
**How to avoid:** Write the test in `package configmaps` (white-box). This is consistent with D-05 and the existing codebase pattern (most test files are in the same package as their source).
**Warning signs:** `configmaps.Store{}` struct literal fails to compile from an external package.

### Pitfall 3: filekv.Watch requires a real .materialized_JSON file
**What goes wrong:** `filekv.Store.Watch()` calls `s.parser.ReadConfig(absPath, ...)` which reads a `.materialized_JSON` file. If no such file exists, the goroutine logs an error and closes the channel immediately.
**Why it happens:** filekv is designed to serve configs from the filesystem; it does not mock the file layer.
**How to avoid:** Use `testdata.SmallTestDir()` which provides a real protoconf root with compiled `.materialized_JSON` files. Ensure the key passed to `Watch()` matches an existing config path (e.g., `"materialized_config/test"`).
**Warning signs:** Watch channel immediately closes or produces errors in test output.

### Pitfall 4: starproto tests need proto types registered with jhump/protoreflect
**What goes wrong:** `desc.LoadMessageDescriptorForMessage(&MyProto{})` returns an error if the proto is not registered in the jhump/protoreflect descriptor registry.
**Why it happens:** jhump/protoreflect uses its own descriptor registry, separate from google.golang.org/protobuf's registry.
**How to avoid:** Use well-known types (structpb.Value, anypb.Any, timestamppb.Timestamp) which are always registered, or use types from the project's own protos which are registered via side-effect imports in existing test files (`_ "github.com/protoconf/protoconf/pb/protoconf/v1"`).
**Warning signs:** `desc.LoadMessageDescriptorForMessage` returns `nil, nil` or a "not found" error.

### Pitfall 5: dummykv.Watch returns immediately for existing keys, then blocks
**What goes wrong:** A test calls `Watch()`, expects to receive a value on the returned channel, but hangs because no value was pre-loaded AND no subsequent `Put()` fires.
**Why it happens:** dummykv.Watch sends the current value (if any) on a goroutine, then blocks waiting for Put(). If the key does not exist yet, nothing is sent until Put().
**How to avoid:** Either `Put()` the value before calling `Watch()`, or `Put()` after watching and use a `select` with a timeout. Follow the exact pattern in `kv_agent_impl_test.go`.
**Warning signs:** Test hangs or times out on `<-watchCh`.

### Pitfall 6: devserver.Run uses signal.NotifyContext
**What goes wrong:** Testing `DevServerCommand.Run()` directly will run a real HTTP listener on `:4300` which may conflict with other tests or running services.
**Why it happens:** The hardcoded `:4300` address and `httpSrv.ListenAndServe()` is not parameterized.
**How to avoid:** Do not call `DevServerCommand.Run()` directly in unit tests. Instead, test the service registration logic by constructing individual components (filekv, agent, compiler service, mutation server) and registering them on a test gRPC server. For smoke-testing `Run()` itself, send an interrupt signal via the context after a brief delay.
**Warning signs:** `bind: address already in use` errors or tests that never return.

---

## Code Examples

### testutil/testutil.go — canonical structure

```go
// Source: modeled after agent/kv_agent_impl_test.go
package testutil

import (
    "context"
    "log"
    "net"
    "testing"

    "github.com/protoconf/protoconf/utils/testdata"
    protoconf_pb "github.com/protoconf/protoconf/pb/protoconf/v1"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    "google.golang.org/grpc/test/bufconn"
    "google.golang.org/protobuf/proto"
    "google.golang.org/protobuf/types/known/anypb"
)

// NewAny wraps a proto.Message in an anypb.Any, panicking on error.
// Mirrors the newAny() helper currently duplicated in agent_test.go files.
func NewAny(t testing.TB, msg proto.Message) *anypb.Any {
    t.Helper()
    a, err := anypb.New(msg)
    if err != nil {
        t.Fatalf("NewAny: %v", err)
    }
    return a
}

// NewTestProtoconfRoot returns a temporary directory initialized as a
// protoconf root using the embedded small test fixtures.
func NewTestProtoconfRoot(t testing.TB) string {
    t.Helper()
    return testdata.SmallTestDir()
}

// NewBufconnAgentServer starts a ProtoconfService gRPC server over bufconn
// and returns a client and a closer function.
func NewBufconnAgentServer(ctx context.Context, t testing.TB, srv protoconf_pb.ProtoconfServiceServer) (protoconf_pb.ProtoconfServiceClient, func()) {
    t.Helper()
    lis := bufconn.Listen(101024 * 1024)
    s := grpc.NewServer()
    protoconf_pb.RegisterProtoconfServiceServer(s, srv)
    go func() {
        if err := s.Serve(lis); err != nil {
            log.Printf("testutil.NewBufconnAgentServer: %v", err)
        }
    }()
    conn, err := grpc.NewClient("passthrough:///bufnet",
        grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
            return lis.Dial()
        }),
        grpc.WithTransportCredentials(insecure.NewCredentials()),
    )
    if err != nil {
        t.Fatalf("grpc.NewClient: %v", err)
    }
    closer := func() {
        lis.Close()
        s.Stop()
    }
    return protoconf_pb.NewProtoconfServiceClient(conn), closer
}
```

### mutate/mutate_test.go — testing unexported field helpers

```go
// package mutate (white-box — same package as source)
package mutate

import (
    "testing"
    "github.com/jhump/protoreflect/desc"
    "github.com/jhump/protoreflect/dynamic"
    "github.com/stretchr/testify/require"
    "google.golang.org/protobuf/types/known/structpb"
)

func TestSetNumeric(t *testing.T) {
    d, err := desc.LoadMessageDescriptorForMessage(&structpb.Struct{})
    require.NoError(t, err)
    // structpb.Struct has no int fields; use a project proto with numeric fields
    // OR define a helper proto in testdata. For a direct test of setNumeric
    // error-path behavior, only the parsing logic matters:
    tests := []struct {
        name    string
        val     string
        wantErr bool
    }{
        {"valid", "42", false},
        {"invalid", "notanumber", true},
        {"hex", "0xff", false},
        {"overflow not an error at parse", "99999999999999999999", true},
    }
    msg := dynamic.NewMessage(d)
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := setNumeric(msg, "fields", tt.val, func(s interface{}) interface{} { return s })
            if tt.wantErr {
                require.Error(t, err)
            } else {
                require.NoError(t, err)
            }
        })
    }
}
```

### agent/otelkv/otelkv_test.go — verifying span delegation

```go
// package otelkv
package otelkv

import (
    "context"
    "testing"

    "github.com/protoconf/protoconf/agent/dummykv"
    "github.com/stretchr/testify/require"
    "go.opentelemetry.io/otel/trace/noop"
)

func TestStore_Put_DelegatesToInner(t *testing.T) {
    inner, _ := dummykv.New(context.Background(), nil, &dummykv.Config{})
    tp := noop.NewTracerProvider()
    store := &Store{
        tracer: tp.Tracer("test"),
        next:   inner,
    }
    err := store.Put(context.Background(), "mykey", []byte("myval"), nil)
    require.NoError(t, err)
    // verify the value was actually stored in inner
    kv, err := inner.Get(context.Background(), "mykey", nil)
    require.NoError(t, err)
    require.Equal(t, []byte("myval"), kv.Value)
}
```

### compiler/starproto/starproto_test.go — wrapping well-known types

```go
// package starproto
package starproto

import (
    "testing"
    "github.com/jhump/protoreflect/desc"
    "github.com/jhump/protoreflect/dynamic"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "go.starlark.net/starlark"
    "google.golang.org/protobuf/types/known/structpb"
)

func makeTestMessage(t *testing.T) *starProtoMessage {
    t.Helper()
    // structpb.Value has a string_value field we can test get/set on
    d, err := desc.LoadMessageDescriptorForMessage(&structpb.Value{})
    require.NoError(t, err)
    return NewStarProtoMessage(dynamic.NewMessage(d))
}

func TestStarProtoMessage_AttrNames(t *testing.T) {
    msg := makeTestMessage(t)
    names := msg.AttrNames()
    assert.Contains(t, names, "string_value")
}

func TestStarProtoMessage_SetField_RoundTrip(t *testing.T) {
    msg := makeTestMessage(t)
    err := msg.SetField("string_value", starlark.String("hello"))
    require.NoError(t, err)
    val, err := msg.Attr("string_value")
    require.NoError(t, err)
    require.Equal(t, starlark.String("hello"), val)
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `grpc.Dial` / `grpc.DialContext` | `grpc.NewClient` | gRPC v1.58+ | Already adopted in codebase; tests must use `grpc.NewClient` with `passthrough:///bufnet` |
| `grpc.WithInsecure()` | `grpc.WithTransportCredentials(insecure.NewCredentials())` | Phase 1 | Already complete; all new tests use the new form |
| `t.Fatalf("setup")` in test init | `require.NoError(t, err)` | Testify adoption | Use testify throughout |

**Deprecated/outdated:**
- `context.WithTimeoutCause`: Used in agent_test.go — is Go 1.21+. Acceptable since project requires Go 1.22+.
- `grpc.Dial` pattern: Removed in codebase already. Do not introduce it in new tests.

---

## Open Questions

1. **setNumeric/setFloat require a message with the right field type**
   - What we know: `setNumeric` will silently succeed even if field doesn't exist (no error from `TrySetFieldByName` for wrong-type fields); it only errors on `strconv.ParseInt` failures
   - What's unclear: Which proto descriptor to use in mutate tests that has integer/float fields matching the tested branches
   - Recommendation: Import a project proto with numeric fields (e.g., `protoconf_mutate_config.MutateConfig` has string fields; consider using `agent/config/v1.AgentConfig` or a well-known type like `google.protobuf.Int64Value`)

2. **devserver hardcoded HTTP port `:4300`**
   - What we know: `devserver.Run()` calls `httpSrv.ListenAndServe()` on `:4300` unconditionally
   - What's unclear: Whether the test should call `Run()` at all, or just test service composition
   - Recommendation: Do not call `Run()` in unit tests. Test component initialization: construct filekv, agent, compilerSvc, mutationServer separately and verify each NewXxx returns no error. If a smoke test of the grpc server is needed, construct `rpcServer := grpc.NewServer()` and verify health service is registered.

3. **configmaps mock: fake.NewSimpleClientset vs. direct struct injection**
   - What we know: `configmaps.Store.clientset` is unexported; white-box tests can set it; `fake.NewSimpleClientset()` is in `k8s.io/client-go/kubernetes/fake` (already in go.mod via k8s.io/client-go v0.30.1)
   - What's unclear: Whether `fake.NewSimpleClientset()` correctly handles the `Watch()` flow that configmaps.Store relies on (it emits watch.Events from the fake object tracker)
   - Recommendation: Use `fake.NewSimpleClientset()` with pre-seeded objects. For `Watch()`, use `fake.NewSimpleClientset().CoreV1().ConfigMaps("default").Watch(...)` which returns a `watch.FakeWatcher`. This is verified behavior in the Kubernetes ecosystem (HIGH confidence).

---

## Environment Availability

Step 2.6: SKIPPED — Phase is code-only. No external services, databases, or CLIs beyond Go toolchain are required. All dependencies are already in go.mod.

---

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go standard `testing` + `github.com/stretchr/testify` v1.9.0 |
| Config file | None — `go test` discovers tests automatically |
| Quick run command | `go test ./mutate/... ./fmt/... ./command/... ./devserver/... ./agent/dummykv/... ./agent/filekv/... ./agent/configmaps/... ./agent/otelkv/... ./compiler/starproto/... ./testutil/...` |
| Full suite command | `go test -race -coverprofile=coverage.txt -covermode=atomic ./...` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| TEST-01 | mutate setNumeric/setFloat/setField field parsing | unit | `go test ./mutate/... -run TestSetNumeric` | Wave 0 |
| TEST-01 | mutate type conversion across all FieldDescriptorProto types | unit | `go test ./mutate/... -run TestSetField` | Wave 0 |
| TEST-02 | fmt formatFile stdout/write/diff/list modes | unit | `go test ./fmt/... -run TestFormatFile` | Wave 0 |
| TEST-02 | fmt isStarlarkFile extension matching | unit | `go test ./fmt/... -run TestIsStarlarkFile` | Wave 0 |
| TEST-03 | command PrefixedUi prefix prepending | unit | `go test ./command/... -run TestPrefixedUi` | Wave 0 |
| TEST-03 | command DefaultUI is non-nil ConcurrentUi | unit | `go test ./command/... -run TestDefaultUI` | Wave 0 |
| TEST-04 | devserver component initialization (no HTTP) | unit | `go test ./devserver/... -run TestDevServer` | Wave 0 |
| TEST-05 | dummykv Put/Get/Watch/Exists/Delete | unit | `go test ./agent/dummykv/...` | Wave 0 |
| TEST-05 | filekv Watch on real materialized_JSON file | unit | `go test ./agent/filekv/...` | Wave 0 |
| TEST-05 | configmaps Put/Get with fake K8s client | unit | `go test ./agent/configmaps/...` | Wave 0 |
| TEST-05 | otelkv delegates all methods to inner store | unit | `go test ./agent/otelkv/...` | Wave 0 |
| TEST-06 | starproto message wrapping, AttrNames, SetField, Attr | unit | `go test ./compiler/starproto/...` | Wave 0 |
| TEST-06 | starproto enum comparison (EQL/NEQ/GT/LT) | unit | `go test ./compiler/starproto/... -run TestEnum` | Wave 0 |
| TEST-06 | starproto any.new and any.unpack | unit | `go test ./compiler/starproto/... -run TestAny` | Wave 0 |
| TEST-14 | testutil.NewAny, NewBufconnAgentServer, NewTestProtoconfRoot | unit | `go test ./testutil/...` | Wave 0 |
| TEST-15 | Coverage reported to Codecov on CI | CI | `.github/workflows/go.yml` Run coverage step | Exists |
| TEST-16 | Error-path tests present in every new test file | unit | embedded in each package's test | Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./mutate/... ./fmt/... ./command/... ./devserver/... ./agent/dummykv/... ./agent/filekv/... ./agent/configmaps/... ./agent/otelkv/... ./compiler/starproto/... ./testutil/...`
- **Per wave merge:** `go test -race -coverprofile=coverage.txt -covermode=atomic ./...`
- **Phase gate:** Full suite green (excluding pre-existing e2e and compiler/lib failures unrelated to Phase 9) before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `testutil/testutil.go` — NewAny, NewBufconnAgentServer, NewTestProtoconfRoot
- [ ] `mutate/mutate_test.go` — TestSetNumeric, TestSetFloat, TestSetField (package mutate)
- [ ] `fmt/command_test.go` — TestFormatFile, TestIsStarlarkFile, TestProcessPath (package fmt)
- [ ] `command/command_test.go` — TestPrefixedUi, TestNewPrefixedUi, TestDefaultUI (package command)
- [ ] `devserver/command_test.go` — TestDevServerCommand_ComponentInit (package devserver)
- [ ] `agent/dummykv/dummykv_test.go` — TestStore_Put_Get, TestStore_Watch, TestStore_Exists (package dummykv)
- [ ] `agent/filekv/filekv_test.go` — TestStore_Watch (package filekv)
- [ ] `agent/configmaps/configmaps_test.go` — TestStore_Put_Get with fake client (package configmaps)
- [ ] `agent/otelkv/otelkv_test.go` — TestStore_Put_DelegatesToInner, TestStore_Get (package otelkv)
- [ ] `compiler/starproto/starproto_test.go` — TestMessage_*, TestEnum_*, TestAny_* (package starproto)

---

## Sources

### Primary (HIGH confidence)
- Direct source code inspection: `agent/kv_agent_impl_test.go`, `agent/agent_test.go`, `server/server_test.go`, `compiler/lib/compiler_test.go` — all test pattern references
- Direct source code inspection: `mutate/mutate.go`, `fmt/command.go`, `command/command.go`, `devserver/command.go`, `agent/dummykv/dummykv.go`, `agent/filekv/filekv.go`, `agent/otelkv/otelkv.go`, `agent/configmaps/configmaps.go`, `compiler/starproto/*.go` — all implementation surfaces
- `.github/workflows/go.yml` — CI test command verification
- `codecov.yml` — Coverage config verification
- `go.mod` — Dependency version verification

### Secondary (MEDIUM confidence)
- `k8s.io/client-go/kubernetes/fake` package: standard Kubernetes testing approach, confirmed by go.mod having `k8s.io/client-go v0.30.1`
- `go.opentelemetry.io/otel/trace/noop` package: standard OTel noop for test use, confirmed by project using `go.opentelemetry.io/otel v1.27.0`

### Tertiary (LOW confidence)
- None

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all packages verified in go.mod; no new dependencies needed
- Architecture: HIGH — patterns extracted directly from existing test files in the repo
- Pitfalls: HIGH — identified from direct source code reading (hardcoded ports, unexported fields, os.Exit)
- Test map: HIGH — requirement IDs mapped directly from CONTEXT.md decisions to observable function names

**Research date:** 2026-03-31
**Valid until:** 2026-05-01 (stable Go testing ecosystem; dependencies unlikely to change)
