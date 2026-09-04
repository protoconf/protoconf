# Phase 9: Unit Test Coverage & Infrastructure - Context

**Gathered:** 2026-03-31
**Status:** Ready for planning

<domain>
## Phase Boundary

Add _test.go files for all previously-untested packages (mutate, fmt, command, devserver, KV stores, starproto). Create a shared test helpers package that at least two test files use. Ensure CI coverage reporting works. Placeholder test fixes (TEST-07 through TEST-13) and integration tests belong to Phase 10.

</domain>

<decisions>
## Implementation Decisions

### Shared Test Helpers — TEST-14
- **D-01:** Create `testutil/` package at repo root (not `internal/testutil/` — matches existing flat package convention like `utils/`, `consts/`).
- **D-02:** Extract these common patterns into testutil:
  - gRPC test server setup with bufconn (currently inline in `agent_test.go`)
  - Proto message helper (`newAny(msg)` wrapper, currently duplicated)
  - Temporary protoconf root setup (combines `testdata.SmallTestDir()` with additional config)
- **D-03:** At least 2 new test files must import testutil to satisfy TEST-14.

### Test Depth Per Package — TEST-01 through TEST-06
- **D-04:** Each new test file covers:
  - Happy path for all primary exported functions
  - Key error paths for sentinel errors and validation failures
  - Edge cases flagged by existing TODO comments
- **D-05:** Use table-driven tests with testify (assert/require) — the established codebase pattern.
- **D-06:** Test file naming follows existing convention: `{source}_test.go` co-located with source.

### Package-Specific Test Scope
- **D-07:** `mutate/` (TEST-01) — Test field parsing (setNumeric, setFloat), type conversion, and the gRPC mutation request flow. Use proto test fixtures, not a live server.
- **D-08:** `fmt/` (TEST-02) — Test Starlark file formatting (format, write, diff modes). Use inline Starlark strings as input.
- **D-09:** `command/` (TEST-03) — Test `RunSubcommands()` routing, `RunCommand()` wrapper, and `DefaultUI` setup. No KVStoreConfig tests needed (removed in Phase 8).
- **D-10:** `devserver/` (TEST-04) — Test combined server startup and service registration. Use bufconn for in-process gRPC.
- **D-11:** KV stores (TEST-05) — Test each store's implemented methods:
  - `dummykv/` — Get, Put, List, Exists, Watch (in-memory, no external deps)
  - `filekv/` — Watch, Get against filesystem (use t.TempDir)
  - `configmaps/` — Interface-level testing (mock Kubernetes client)
  - `otelkv/` — Verify OTel span creation wraps underlying store calls (mock inner store)
- **D-12:** `compiler/starproto/` (TEST-06) — Test message wrapping, field access (get/set), enum handling, map/repeated fields, Any type support. Use proto descriptors from testdata.

### Coverage & CI — TEST-15, TEST-16
- **D-13:** No hard coverage threshold enforced initially — CI continues to report coverage via codecov. The goal is to add meaningful tests, not chase a percentage.
- **D-14:** Existing `go test -race -coverprofile=coverage.txt -covermode=atomic -v ./...` command in CI is sufficient. No CI config changes needed.
- **D-15:** TEST-16 (error path tests) is satisfied by D-04's requirement to test key error paths in each new test file.

### Claude's Discretion
- Exact test case selection within each package (which specific functions to test first)
- Whether to use subtests (t.Run) or flat test functions — follow whatever the nearest existing test file uses
- Mock implementation details for configmaps and otelkv
- Whether to split large test files into multiple files per package

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Existing test patterns (reference implementations)
- `agent/agent_test.go` — gRPC bufconn server setup pattern, proto helpers, testify usage
- `agent/kv_agent_impl_test.go` — Table-driven tests with KV store mocking via dummykv
- `compiler/lib/compiler_test.go` — Compiler test setup with testdata directories
- `utils/tls_test.go` — TLS helper testing with self-signed cert generation
- `server/server_test.go` — Mutation server test with bufconn (has TODO placeholder — Phase 10)

### Test data infrastructure
- `utils/testdata/embed.go` — Embedded test fixtures with SmallTestDir/LargeTestDir helpers
- `utils/testdata/` — Pre-built test data directories (small/, large/, bad_proto/)

### Packages to test (source files)
- `mutate/mutate.go` — Mutation CLI with field parsing and gRPC client
- `fmt/command.go` — Starlark formatter command
- `command/command.go` — CLI routing and shared utilities (KVStoreConfig removed in Phase 8)
- `devserver/command.go` — Combined dev server
- `agent/dummykv/dummykv.go` — In-memory KV store
- `agent/filekv/filekv.go` — File-based KV store
- `agent/configmaps/configmaps.go` — Kubernetes ConfigMaps KV store
- `agent/otelkv/otelkv.go` — OTel-instrumented KV store wrapper
- `compiler/starproto/` — Starlark-protobuf bridge (message.go, field_accessors.go, etc.)

### CI & Coverage
- `.github/workflows/go.yml` — Test execution with race detection and coverage
- `codecov.yml` — Coverage config (ignores *.pb.go)

### Requirements
- `.planning/REQUIREMENTS.md` — TEST-01 through TEST-06, TEST-14, TEST-15, TEST-16

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `utils/testdata/embed.go`: SmallTestDir(), LargeTestDir() — embedded test fixtures with auto-initialized git repos
- `agent/dummykv`: Already an in-memory KV implementation usable as test double
- Existing bufconn pattern in agent_test.go: `testServer(ctx, srv)` function
- `newAny(msg proto.Message)` helper duplicated across test files — candidate for testutil

### Established Patterns
- Table-driven tests with `[]struct{ name, args, wantErr }` — used in 8+ existing test files
- testify assert/require for assertions (never raw `if err != nil { t.Fatal }`)
- `context.WithTimeout(context.Background(), 10*time.Second)` for test timeouts
- `t.TempDir()` for temporary directories (Go 1.15+ pattern)

### Integration Points
- New test files are co-located with source: `mutate/mutate_test.go`, `fmt/command_test.go`, etc.
- testutil/ package must be importable by any package — no circular dependencies
- CI picks up new tests automatically (`go test ./...` is recursive)

</code_context>

<specifics>
## Specific Ideas

No specific requirements — follow established test patterns, prioritize meaningful coverage over breadth.

</specifics>

<deferred>
## Deferred Ideas

- Fixing placeholder assertions in parser_test.go and inserter_test.go — Phase 10 (TEST-07 through TEST-10)
- E2E integration tests for mutation, TLS, and auth flows — Phase 10 (TEST-11 through TEST-13)
- Coverage percentage thresholds — evaluate after Phase 9+10 add tests
- Property-based testing / fuzzing — future concern

</deferred>

---

*Phase: 09-unit-test-coverage-infrastructure*
*Context gathered: 2026-03-31*
