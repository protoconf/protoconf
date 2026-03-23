# Codebase Concerns

**Analysis Date:** 2026-03-23

## Tech Debt

**Unimplemented KV Store Methods (panic on call):**
- Issue: Multiple KV store implementations have methods that `panic("implement me")` at runtime instead of returning errors. Any caller invoking these methods will crash the process.
- Files:
  - `agent/dummykv/dummykv.go` lines 140, 148, 154, 167, 173, 179 (WatchTree, NewLock, List, AtomicPut, AtomicDelete, Close)
  - `agent/filekv/filekv.go` lines 93, 99, 164, 172, 178, 184, 191, 197 (Get, Delete, WatchTree, NewLock, List, DeleteTree, AtomicPut, AtomicDelete)
  - `agent/configmaps/configmaps.go` lines 291, 299, 305, 311, 318, 324, 330 (WatchTree, NewLock, List, DeleteTree, AtomicPut, AtomicDelete, Close)
- Impact: **Critical** - Runtime panics crash the entire process. These satisfy the `store.Store` interface but are landmines for any consumer that calls unimplemented methods.
- Fix approach: Return `fmt.Errorf("not implemented")` errors instead of panicking. Or implement the missing methods.

**Massive TODO Backlog in Tests:**
- Issue: Multiple test files contain stub test tables with `// TODO: Add test cases.` placeholders, providing false confidence in test coverage.
- Files:
  - `server/server_test.go` line 131 - MutateConfig test has minimal cases
  - `server/server_test.go` line 139 - `TODO(smintz): assert the response` - test runs but does not validate output
  - `inserter/inserter_test.go` line 133 - inserter help test has placeholder
  - `compiler/lib/parser/parser_test.go` lines 42, 90 - parser tests incomplete
  - `agent/kv_agent_rollout_impl_test.go` line 116 - rollout agent test incomplete
- Impact: **High** - Reduced confidence in correctness; bugs may go undetected.
- Fix approach: Write actual test cases covering error paths and edge cases.

**FIXME Comments in Starlark Proto Integration:**
- Issue: Two FIXME comments indicate incomplete functionality in the starproto message type system.
- Files:
  - `compiler/starproto/message_type.go` line 89: `// FIXME: iterate nested extensions as well?`
  - `compiler/starproto/message_type.go` line 94: `// FIXME: fields, nested enum/message types, nested extensions?` - `AttrNames()` returns `nil`, meaning tab-completion and introspection of message types is broken in Starlark.
- Impact: **Medium** - Users cannot enumerate available fields on proto message types in Starlark configs.
- Fix approach: Implement `AttrNames()` to return field names, nested types, and extensions.

**Global Mutable State in `mutate` Package:**
- Issue: `mutate/mutate.go` line 30 declares `var conn *grpc.ClientConn` as a package-level global. This is not goroutine-safe and prevents concurrent use of the mutate command.
- Files: `mutate/mutate.go` line 30
- Impact: **Low** - CLI tool is typically single-use, but this prevents reuse as a library.
- Fix approach: Move connection to local scope within the `Run` method (it is already assigned there at line 172).

**Duplicate OpenTelemetry Bootstrap Code:**
- Issue: Nearly identical OTel tracer/meter provider setup code is duplicated between the server and agent entry points (~40 lines each).
- Files:
  - `server/server.go` lines 105-148
  - `agent/agent.go` lines 60-99
- Impact: **Medium** - Any change to observability setup must be made in two places. Divergence risk.
- Fix approach: Extract a shared `otel.Setup(ctx)` helper into a common package (e.g., `utils/` or a new `observability/` package).

**Unnecessary `runtime.GOMAXPROCS` Call:**
- Issue: `inserter/inserter.go` lines 48-53 set `GOMAXPROCS` to `NumCPU()` in an `init()` function with a comment acknowledging this is redundant ("Just to show we can change the number of CPUs"). This is the Go runtime default since Go 1.5.
- Files: `inserter/inserter.go` lines 47-53
- Impact: **Low** - Dead code that adds confusion.
- Fix approach: Remove the `init()` function entirely.

## Known Bugs

**Missing `=` in Environment Variable Assignment:**
- Symptoms: The `PROTOCONF_COMPILER_ADDR` environment variable is never properly set when running pre/post mutation scripts. The string concatenation is missing an `=` sign.
- Files: `server/server.go` line 453
- Code: `"PROTOCONF_COMPILER_ADDR"+s.config.grpcAddress` should be `"PROTOCONF_COMPILER_ADDR="+s.config.grpcAddress`
- Trigger: Any mutation that uses pre/post mutation scripts.
- Workaround: None. Scripts relying on `PROTOCONF_COMPILER_ADDR` will see it undefined.

**`filekv.Watch` Dead Code Check:**
- Symptoms: In `agent/filekv/filekv.go` lines 143-145, there is an `if err != nil { return }` check after `proto.Marshal` - but `err` was already checked at line 135-137 and the variable is reassigned at line 134. The check at line 143 is checking the same `err` that was already handled, making it dead code.
- Files: `agent/filekv/filekv.go` lines 143-145
- Impact: **Low** - No functional bug, but misleading code.

**`dummykv.Exists` Always Returns True:**
- Symptoms: `agent/dummykv/dummykv.go` line 121 - `Exists()` always returns `(true, nil)` regardless of whether the key exists. This can mask missing keys.
- Files: `agent/dummykv/dummykv.go` lines 119-122
- Impact: **Medium** - Any code path relying on `Exists()` to check key presence will get false positives.

## Security Considerations

**All gRPC Connections Use Insecure Credentials:**
- Risk: Every gRPC client connection in the codebase uses `insecure.NewCredentials()` or the deprecated `grpc.WithInsecure()`. There is no TLS support, meaning all traffic (including config mutations) is unencrypted.
- Files:
  - `server/server.go` line 524
  - `mutate/mutate.go` line 172
  - `compiler/command.go` line 101 (also uses deprecated `grpc.WithInsecure()`)
  - `agent/legacy.go` line 42
  - `test/e2e.go` line 31
- Current mitigation: None.
- Recommendations: Add TLS configuration options. At minimum, support `--tls-cert` and `--tls-key` flags. The `compiler/command.go` usage also needs to migrate from the deprecated `grpc.WithInsecure()` to `grpc.WithTransportCredentials(insecure.NewCredentials())`.

**No Authentication or Authorization on Mutation Server:**
- Risk: The gRPC mutation server accepts unauthenticated requests. Anyone who can reach the server can mutate any configuration.
- Files: `server/server.go` - `MutateConfig` method (line 352) has no auth checks.
- Current mitigation: Assumed network-level isolation.
- Recommendations: Add gRPC interceptors for authentication (mTLS, JWT, or API key). Add authorization checks on config paths.

**Mutation Scripts Execute with No Sandboxing:**
- Risk: Pre/post mutation scripts (`server/server.go` line 450) are executed via `exec.Command(filename)` with a stripped environment. However, the script path comes from CLI flags with no validation - an attacker who can set flags could execute arbitrary binaries.
- Files: `server/server.go` lines 449-463
- Current mitigation: Script paths are set via CLI flags (not user input from gRPC requests).
- Recommendations: Validate script paths exist and are executable. Consider restricting to a specific directory.

**Panics in Production Code:**
- Risk: Multiple `panic()` calls in non-test code will crash the entire server process rather than returning errors gracefully.
- Files:
  - `server/server.go` lines 107, 136 - panics on OTel initialization failure
  - `agent/agent.go` lines 62, 91 - panics on OTel initialization failure
  - `compiler/starproto/field.go` lines 31, 99, 103, 110 - panics on type conversion errors
- Current mitigation: None.
- Recommendations: Replace panics with error returns. For OTel failures, log a warning and continue without telemetry. For starproto, propagate errors up the call stack.

## Performance Bottlenecks

**Full Recompilation on Mutation:**
- Problem: When `MutateConfig` is called with a compiler attached, it recompiles ALL config files, not just the affected one.
- Files: `server/server.go` lines 386-412 - calls `compiler.GetAllConfigs()` then compiles each file.
- Cause: No dependency graph to determine which configs are affected by a mutable config change.
- Improvement path: Build a dependency graph during initial compilation; on mutation, only recompile configs that depend on the changed mutable config.

## Fragile Areas

**Starlark Global Resolver Settings:**
- Files: `compiler/lib/compiler.go` lines 34-39
- Why fragile: The compiler constructor modifies global `resolve.*` variables (`AllowNestedDef`, `AllowLambda`, etc.). These are package-level globals in the `go.starlark.net/resolve` package. If multiple compilers are created or if tests run in parallel, these globals create race conditions.
- Safe modification: These settings should be set once at program startup, not in the constructor.
- Test coverage: Tests create compilers sequentially, masking the race.

**`os.Exit(1)` Calls Deep in Library Code:**
- Files:
  - `compiler/lib/module_service.go` line 67
  - `compiler/lib/starlark_loader.go` lines 124, 127, 138
  - `mutate/mutate.go` lines 103, 114, 119, 129, 150, 175, 181, 196, 208, 217
- Why fragile: Library code should never call `os.Exit()`. This makes the code untestable (tests cannot catch exits) and prevents graceful shutdown or error recovery.
- Safe modification: Replace all `os.Exit(1)` calls with error returns. The CLI entry point should be the only place that calls `os.Exit()`.
- Test coverage: Not tested because `os.Exit` terminates the test process.

**Inserter Git Root Discovery Loop:**
- Files: `inserter/inserter.go` lines 184-193
- Why fragile: The loop walks up the directory tree to find a git root, but silently ignores `filepath.Rel` errors (line 191 `rel, _ = filepath.Rel(...)`). If the working directory is outside the git root, behavior is undefined.
- Safe modification: Check the error from `filepath.Rel` and handle it.

## Dependencies at Risk

**Deprecated gRPC API Usage:**
- Risk: `compiler/command.go` line 101 uses `grpc.WithInsecure()` which has been deprecated since gRPC-Go v1.40.0.
- Impact: Will eventually be removed in a future gRPC-Go release.
- Migration plan: Replace with `grpc.WithTransportCredentials(insecure.NewCredentials())`.

**Deprecated gRPC Reflection Package:**
- Risk: `server/server.go` imports and uses `grpc_reflection_v1alpha` (line 50) which is the legacy reflection API.
- Impact: The `v1alpha` reflection API is deprecated in favor of `grpc_reflection_v1`.
- Migration plan: Switch to `google.golang.org/grpc/reflection/grpc_reflection_v1`.

**`github.com/jhump/protoreflect` and `dynamic` Package:**
- Risk: The codebase relies heavily on `jhump/protoreflect/dynamic` for dynamic protobuf message handling. The official `google.golang.org/protobuf/types/dynamicpb` package is the recommended replacement.
- Files: Used in `compiler/starproto/`, `mutate/mutate.go`, `compiler/lib/compiler.go`
- Impact: **Medium** - Maintaining two dynamic message systems (`dynamic.Message` and `dynamicpb.Message`) increases complexity and conversion overhead.
- Migration plan: Gradually migrate from `jhump/protoreflect/dynamic` to `dynamicpb` throughout the codebase.

**`github.com/mitchellh/cli` CLI Framework:**
- Risk: This library is in maintenance mode. HashiCorp has archived or reduced investment in many OSS projects.
- Files: Used in `server/server.go`, `inserter/inserter.go`, `mutate/mutate.go`, `compiler/command.go`, `agent/`, `command/command.go`
- Impact: **Low** - Still functional but unlikely to receive new features.
- Migration plan: Consider migrating to `cobra` or `urfave/cli` for long-term support.

## Test Coverage Gaps

**No Tests for KV Store Implementations:**
- What's not tested: `agent/dummykv/`, `agent/filekv/`, `agent/otelkv/`, `agent/configmaps/` have zero dedicated test files.
- Files: All files in those directories.
- Risk: Unimplemented panic methods, the `Exists` always-true bug, and file watcher logic are all untested.
- Priority: **High**

**Mutation Server Response Not Asserted:**
- What's not tested: `server/server_test.go` line 139 explicitly notes `// TODO(smintz): assert the response` - the MutateConfig test runs the mutation but does not verify the response content.
- Files: `server/server_test.go` lines 133-146
- Risk: Mutations could return incorrect data without detection.
- Priority: **High**

**No Tests for `mutate` CLI Package:**
- What's not tested: The entire `mutate/` package has no test file. Field parsing, type conversion, and gRPC mutation flow are untested.
- Files: `mutate/mutate.go`
- Risk: The complex type-switch field setter logic (lines 131-167) could silently mishandle field types.
- Priority: **Medium**

**No Tests for `devserver` Package:**
- What's not tested: `devserver/command.go` has no test file.
- Files: `devserver/command.go`
- Risk: **Low** - Development-only tool.
- Priority: **Low**

---

*Concerns audit: 2026-03-23*
