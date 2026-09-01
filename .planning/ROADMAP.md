# Roadmap: Protoconf Quality & Consistency Overhaul

## Overview

This milestone transforms Protoconf from a functional but organically-grown codebase into a consistently structured, thoroughly tested, and security-hardened platform. Work proceeds from foundational hygiene (deprecated APIs, os.Exit cleanup) through security hardening (TLS, auth) to proto-defined CLI configuration and comprehensive test coverage — each phase unblocking the next.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: Deprecated API Migrations** - Replace grpc.WithInsecure and v1alpha reflection with current stable APIs (completed 2026-03-23)
- [ ] **Phase 2: os.Exit Refactoring** - Remove os.Exit from library code and propagate errors to CLI entry points
- [x] **Phase 3: Observability & Global State Cleanup** - Extract shared OTel bootstrap and remove global mutable state (completed 2026-03-27)
- [ ] **Phase 4: Dead Code Removal** - Remove unnecessary init functions and dead error checks
- [x] **Phase 5: TLS Support** - Add TLS to gRPC servers and clients with insecure-mode warning (completed 2026-03-28)
- [ ] **Phase 6: Token Auth & Script Security** - Add token-based mutation auth with credential forwarding and script validation
- [x] **Phase 7: Proto-Defined CLI Configs** - Define protobuf messages for all component configurations (completed 2026-03-28)
- [x] **Phase 8: CLI Flag Generation & Config Loading** - Generate CLI flags from protos and add env/file config loading (completed 2026-03-28)
- [x] **Phase 9: Unit Test Coverage & Infrastructure** - Add test files for untested packages and shared test helpers (completed 2026-03-31)
- [x] **Phase 10: Placeholder Fixes & Integration Tests** - Replace placeholder assertions and add e2e integration tests (completed 2026-03-31)

## Phase Details

### Phase 1: Deprecated API Migrations
**Goal**: All gRPC connections and reflection registrations use current stable APIs
**Depends on**: Nothing (first phase)
**Requirements**: DEPR-01, DEPR-02
**Success Criteria** (what must be TRUE):
  1. No call site in the codebase uses grpc.WithInsecure() — all use grpc.WithTransportCredentials(insecure.NewCredentials())
  2. All gRPC servers register reflection using grpc_reflection_v1, not grpc_reflection_v1alpha
  3. The binary compiles and existing tests pass after migration
**Plans**: 1 plan
Plans:
- [x] 01-01-PLAN.md — Migrate all deprecated gRPC APIs (WithInsecure, reflection v1alpha, Dial/DialContext)

### Phase 2: os.Exit Refactoring
**Goal**: Library code never terminates the process — errors propagate to CLI entry points
**Depends on**: Phase 1
**Requirements**: REFC-01, REFC-02, REFC-03, REFC-04
**Success Criteria** (what must be TRUE):
  1. compiler/lib/module_service.go contains no os.Exit calls
  2. compiler/lib/starlark_loader.go contains no os.Exit calls (all 3 locations resolved)
  3. mutate/mutate.go contains no os.Exit calls (all ~10 locations resolved)
  4. All refactored functions return errors that propagate up to CLI-layer entry points where os.Exit is appropriate
  5. Existing CLI behavior is unchanged — error cases still exit the process with non-zero status
**Plans**: 2 plans
Plans:
- [x] 02-01-PLAN.md — Remove os.Exit from compiler/lib (module_service + starlark_loader) and propagate errors through NewCompiler to all callers
- [x] 02-02-PLAN.md — Remove os.Exit from mutate/mutate.go (Run method + setNumeric/setFloat helpers)

### Phase 3: Observability & Global State Cleanup
**Goal**: OTel initialization is shared and non-fatal; global mutable state is eliminated from library packages
**Depends on**: Phase 2
**Requirements**: REFC-05, REFC-06, REFC-07, REFC-08
**Success Criteria** (what must be TRUE):
  1. A single shared observability package initializes OTel — server/server.go and agent/agent.go both import it instead of duplicating setup
  2. An OTel init failure logs a warning and continues rather than panicking
  3. Starlark resolve.* global settings are configured at program startup, not inside the Compiler constructor
  4. mutate/mutate.go holds its gRPC ClientConn as a local variable within Run, not as a package-level global
**Plans**: 2 plans
Plans:
- [x] 03-01-PLAN.md — Extract shared OTel bootstrap with noop fallback into observability package
- [x] 03-02-PLAN.md — Move Starlark resolve globals to sync.Once and localize mutate grpc.ClientConn

### Phase 4: Dead Code Removal
**Goal**: Codebase contains no unnecessary init functions or unreachable error handling
**Depends on**: Phase 3
**Requirements**: REFC-09, REFC-10
**Success Criteria** (what must be TRUE):
  1. inserter/inserter.go has no runtime.GOMAXPROCS init() function
  2. filekv.Watch lines 143-145 dead error check is removed and the surrounding logic is correct
  3. All tests still pass after removal
**Plans**: 1 plan
Plans:
- [x] 04-01-PLAN.md — Remove dead init() from inserter and dead error check from filekv

### Phase 5: TLS Support
**Goal**: gRPC servers and clients support TLS connections; insecure mode warns operators
**Depends on**: Phase 4
**Requirements**: SECR-01, SECR-02, SECR-03
**Success Criteria** (what must be TRUE):
  1. A gRPC server started with --tls-cert and --tls-key flags accepts only TLS connections
  2. A gRPC client can connect to a TLS-enabled server using a matching certificate
  3. A server started without TLS flags logs a visible warning that the connection is insecure
  4. Existing insecure-mode usage continues to work without any flag changes
**Plans**: 2 plans
Plans:
- [x] 05-01-PLAN.md — Create shared TLS helper (BuildTLSConfig) with unit tests
- [x] 05-02-PLAN.md — Wire TLS into agent, mutation server, and mutate CLI with insecure warnings

### Phase 6: Token Auth & Script Security
**Goal**: Mutation server enforces token-based auth; credentials reach pre/post scripts; script paths are validated
**Depends on**: Phase 5
**Requirements**: SECR-04, SECR-05, SECR-06, SECR-07
**Success Criteria** (what must be TRUE):
  1. A mutation request with a valid token in gRPC metadata succeeds
  2. A mutation request with no token (when auth is configured) is rejected with an Unauthenticated error
  3. Pre/post mutation scripts receive auth credentials as environment variables
  4. A mutation request referencing a non-existent or non-executable script path is rejected with a clear error before execution begins
**Plans**: 2 plans
Plans:
- [x] 06-01-PLAN.md — Add bearer token auth interceptor with --auth-token flag and unit tests
- [x] 06-02-PLAN.md — Add script path validation, credential forwarding to scripts, and COMPILER_ADDR bug fix

### Phase 7: Proto-Defined CLI Configs
**Goal**: Every component's configuration is expressed as a protobuf message
**Depends on**: Phase 6
**Requirements**: PCLI-01, PCLI-02, PCLI-03, PCLI-04
**Success Criteria** (what must be TRUE):
  1. A .proto file defines the server configuration message (address, TLS, auth, scripts)
  2. A .proto file defines the compiler configuration message (proto paths, output settings)
  3. A .proto file defines the inserter configuration message (KV store, prefix, rollout)
  4. A .proto file defines the mutate CLI configuration message (target server, field path, value)
  5. All proto definitions pass protoc compilation without errors
**Plans**: 1 plan
Plans:
- [x] 07-01-PLAN.md — Define proto config messages for server, compiler, inserter, and mutate CLI

### Phase 8: CLI Flag Generation & Config Loading
**Goal**: CLI flags are generated from proto definitions; all components accept env vars and config files
**Depends on**: Phase 7
**Requirements**: PCLI-05, PCLI-06, PCLI-07, PCLI-08, PCLI-09
**Success Criteria** (what must be TRUE):
  1. Running any component with --help shows flags generated from the proto definition, not hand-written ones
  2. All existing flags are present in the generated output (backward compatible)
  3. Setting a PROTOCONF_* environment variable configures the corresponding component option
  4. Passing a config file path loads configuration from JSON, YAML, or protobuf format
  5. Flag values override env vars, which override config file values, which override compiled defaults
**Plans**: 4 plans
Plans:
- [x] 08-01-PLAN.md — Migrate server and compiler to libprotoconf-generated flags with env var and config file support
- [x] 08-02-PLAN.md — Migrate inserter and mutate to libprotoconf-generated flags, remove KVStoreConfig dead code
- [ ] 08-03-PLAN.md — Gap closure (PCLI-09): shared config-file layering helper + server env-over-file precedence, proven end-to-end
- [ ] 08-04-PLAN.md — Gap closure (PCLI-09): apply precedence fix to compiler, inserter, mutate and agent; record the agent behavior change
**UI hint**: no

### Phase 9: Unit Test Coverage & Infrastructure
**Goal**: Every previously-untested package has test files; shared test helpers eliminate boilerplate
**Depends on**: Phase 2
**Requirements**: TEST-01, TEST-02, TEST-03, TEST-04, TEST-05, TEST-06, TEST-14, TEST-15, TEST-16
**Success Criteria** (what must be TRUE):
  1. mutate/, fmt/, command/, and devserver/ each have at least one _test.go file with passing tests
  2. All four KV store packages (dummykv, filekv, configmaps, otelkv) have dedicated test files covering their implemented methods
  3. compiler/starproto/ has tests covering message wrapping, field access, enum handling, and Any type support
  4. A shared test helpers package exists and is used by at least two test files (gRPC server setup, KV store creation, config compilation)
  5. CI reports coverage with a minimum threshold enforced; test failures on edge cases and error paths are present
**Plans**: 4 plans
Plans:
- [x] 09-01-PLAN.md — Create testutil/ package, add tests for command/ and fmt/
- [x] 09-02-PLAN.md — Add tests for all four KV store implementations (dummykv, filekv, otelkv, configmaps)
- [x] 09-03-PLAN.md — Add tests for mutate/ and devserver/ packages
- [x] 09-04-PLAN.md — Add tests for compiler/starproto/ (message wrapping, field access, Any type)

### Phase 10: Placeholder Fixes & Integration Tests
**Goal**: No test in the codebase has placeholder assertions; e2e tests cover mutation, TLS, and auth flows
**Depends on**: Phase 6, Phase 9
**Requirements**: TEST-07, TEST-08, TEST-09, TEST-10, TEST-11, TEST-12, TEST-13
**Success Criteria** (what must be TRUE):
  1. server/server_test.go MutateConfig test asserts actual response content, not a TODO stub
  2. compiler/lib/parser/parser_test.go and inserter/inserter_test.go contain real assertions against known inputs and outputs
  3. agent/kv_agent_rollout_impl_test.go placeholder cases are completed with meaningful assertions
  4. An e2e test exercises the full mutation flow including pre/post script execution and verifies the outcome
  5. E2e tests cover TLS-enabled gRPC connections and token-based auth rejection/acceptance
**Plans**: 2 plans
Plans:
- [x] 10-01-PLAN.md — Replace placeholder assertions in server, parser, inserter, and rollout test files
- [x] 10-02-PLAN.md — Add e2e integration tests for mutation+scripts, TLS, and token auth

## Progress

**Execution Order:**
Phases execute in numeric order: 1 -> 2 -> 3 -> 4 -> 5 -> 6 -> 7 -> 8 -> 9 -> 10

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Deprecated API Migrations | 1/1 | Complete   | 2026-03-23 |
| 2. os.Exit Refactoring | 1/2 | In Progress|  |
| 3. Observability & Global State Cleanup | 2/2 | Complete   | 2026-03-27 |
| 4. Dead Code Removal | 0/TBD | Not started | - |
| 5. TLS Support | 2/2 | Complete   | 2026-03-28 |
| 6. Token Auth & Script Security | 1/2 | In Progress|  |
| 7. Proto-Defined CLI Configs | 1/1 | Complete   | 2026-03-28 |
| 8. CLI Flag Generation & Config Loading | 2/2 | Complete   | 2026-03-28 |
| 9. Unit Test Coverage & Infrastructure | 4/4 | Complete   | 2026-03-31 |
| 10. Placeholder Fixes & Integration Tests | 2/2 | Complete    | 2026-03-31 |
