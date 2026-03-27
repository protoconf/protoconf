# Requirements: Protoconf Quality & Consistency Overhaul

**Defined:** 2026-03-23
**Core Value:** Every component must be testable, consistent, and free of runtime surprises

## v1 Requirements

Requirements for this milestone. Each maps to roadmap phases.

### Testing — Coverage

- [ ] **TEST-01**: mutate/ package has unit tests covering field parsing, type conversion, and gRPC mutation flow
- [ ] **TEST-02**: fmt/ package has unit tests covering Starlark file formatting
- [ ] **TEST-03**: command/ package has unit tests covering subcommand routing and KV store config
- [ ] **TEST-04**: devserver/ package has tests covering combined server startup and service registration
- [ ] **TEST-05**: KV store implementations (dummykv, filekv, configmaps, otelkv) have dedicated test files covering implemented methods
- [ ] **TEST-06**: compiler/starproto/ package has tests covering message wrapping, field access, enum handling, and Any type support

### Testing — Fix Placeholders

- [ ] **TEST-07**: server/server_test.go MutateConfig test asserts response content (resolves TODO(smintz))
- [ ] **TEST-08**: compiler/lib/parser/parser_test.go placeholder test cases are filled with real assertions
- [ ] **TEST-09**: inserter/inserter_test.go placeholder test cases are filled with real assertions
- [ ] **TEST-10**: agent/kv_agent_rollout_impl_test.go placeholder test cases are completed

### Testing — Integration

- [ ] **TEST-11**: e2e test suite covers the mutation flow with pre/post script execution
- [ ] **TEST-12**: e2e test suite covers TLS-enabled gRPC connections (once TLS is implemented)
- [ ] **TEST-13**: e2e test suite covers token-based auth flow (once auth is implemented)

### Testing — Infrastructure

- [ ] **TEST-14**: Shared test helpers extracted for common patterns (gRPC server setup, config compilation, KV store creation)
- [ ] **TEST-15**: CI enforces minimum coverage threshold with clear reporting
- [ ] **TEST-16**: Test fixtures cover error paths and edge cases, not just happy paths

### Refactoring — os.Exit

- [x] **REFC-01**: compiler/lib/module_service.go os.Exit replaced with error return
- [x] **REFC-02**: compiler/lib/starlark_loader.go os.Exit calls (3 locations) replaced with error returns
- [x] **REFC-03**: mutate/mutate.go os.Exit calls (~10 locations) replaced with error returns
- [x] **REFC-04**: All refactored functions propagate errors to CLI entry points where os.Exit is appropriate

### Refactoring — OTel

- [x] **REFC-05**: Shared observability package extracts duplicate OTel tracer/meter setup from server/server.go and agent/agent.go
- [x] **REFC-06**: OTel init failures log warnings and continue instead of panicking

### Refactoring — Global State

- [x] **REFC-07**: Starlark resolve.* global settings moved to program startup, not Compiler constructor
- [x] **REFC-08**: mutate/mutate.go global grpc.ClientConn moved to local scope within Run method

### Refactoring — Dead Code

- [x] **REFC-09**: inserter/inserter.go unnecessary runtime.GOMAXPROCS init() function removed
- [x] **REFC-10**: filekv.Watch dead error check at lines 143-145 cleaned up

### Security — TLS

- [ ] **SECR-01**: gRPC servers accept --tls-cert and --tls-key flags to enable TLS
- [ ] **SECR-02**: gRPC clients support TLS connections when server has TLS enabled
- [ ] **SECR-03**: Insecure mode remains the default but logs a warning

### Security — Auth

- [ ] **SECR-04**: Mutation server supports token-based authentication (JWT or API key) via gRPC metadata
- [ ] **SECR-05**: Auth credentials are forwarded to pre/post mutation scripts as environment variables
- [ ] **SECR-06**: Unauthenticated requests are rejected when auth is configured

### Security — Scripts

- [ ] **SECR-07**: Pre/post mutation script paths are validated (exist, executable) before execution

### Deprecated APIs

- [x] **DEPR-01**: All grpc.WithInsecure() calls migrated to grpc.WithTransportCredentials(insecure.NewCredentials())
- [x] **DEPR-02**: grpc_reflection_v1alpha migrated to grpc_reflection_v1

### Proto CLI — Config Definitions

- [ ] **PCLI-01**: Proto definitions exist for server configuration (address, TLS, auth, scripts)
- [ ] **PCLI-02**: Proto definitions exist for compiler configuration (proto paths, output settings)
- [ ] **PCLI-03**: Proto definitions exist for inserter configuration (KV store, prefix, rollout)
- [ ] **PCLI-04**: Proto definitions exist for mutate CLI configuration (target server, field path, value)

### Proto CLI — Generation

- [ ] **PCLI-05**: CLI flag parsing is generated from proto definitions for all components
- [ ] **PCLI-06**: Generated CLI matches current flag interface (backward compatible)

### Proto CLI — Config Loading

- [ ] **PCLI-07**: All components support config loading via environment variables (PROTOCONF_* prefix)
- [ ] **PCLI-08**: All components support config loading via config files (JSON/YAML/protobuf)
- [ ] **PCLI-09**: Config precedence follows: flags > env vars > config file > defaults

## v2 Requirements

Deferred to future milestone. Tracked but not in current roadmap.

### Deprecated APIs

- **DEPR-03**: Migrate from jhump/protoreflect/dynamic to official dynamicpb package

### Security

- **SECR-08**: mTLS support for mutual authentication
- **SECR-09**: Role-based authorization on config paths

### CLI

- **PCLI-10**: Migrate from mitchellh/cli to a modern CLI framework (cobra or urfave/cli)

## Out of Scope

| Feature | Reason |
|---------|--------|
| KV store unimplemented method implementations | Panics are intentional interface stubs signaling future needs |
| New feature development | This milestone is purely quality/consistency |
| jhump/protoreflect migration | Large scope, defer to v2 — would touch compiler/starproto extensively |
| PROTOCONF_COMPILER_ADDR env var bug fix | Not selected for this milestone |
| Mobile/web client SDKs | Backend focus only |
| Python SDK improvements | Out of scope for this milestone |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| DEPR-01 | Phase 1 | Complete |
| DEPR-02 | Phase 1 | Complete |
| REFC-01 | Phase 2 | Complete |
| REFC-02 | Phase 2 | Complete |
| REFC-03 | Phase 2 | Complete |
| REFC-04 | Phase 2 | Complete |
| REFC-05 | Phase 3 | Complete |
| REFC-06 | Phase 3 | Complete |
| REFC-07 | Phase 3 | Complete |
| REFC-08 | Phase 3 | Complete |
| REFC-09 | Phase 4 | Complete |
| REFC-10 | Phase 4 | Complete |
| SECR-01 | Phase 5 | Pending |
| SECR-02 | Phase 5 | Pending |
| SECR-03 | Phase 5 | Pending |
| SECR-04 | Phase 6 | Pending |
| SECR-05 | Phase 6 | Pending |
| SECR-06 | Phase 6 | Pending |
| SECR-07 | Phase 6 | Pending |
| PCLI-01 | Phase 7 | Pending |
| PCLI-02 | Phase 7 | Pending |
| PCLI-03 | Phase 7 | Pending |
| PCLI-04 | Phase 7 | Pending |
| PCLI-05 | Phase 8 | Pending |
| PCLI-06 | Phase 8 | Pending |
| PCLI-07 | Phase 8 | Pending |
| PCLI-08 | Phase 8 | Pending |
| PCLI-09 | Phase 8 | Pending |
| TEST-01 | Phase 9 | Pending |
| TEST-02 | Phase 9 | Pending |
| TEST-03 | Phase 9 | Pending |
| TEST-04 | Phase 9 | Pending |
| TEST-05 | Phase 9 | Pending |
| TEST-06 | Phase 9 | Pending |
| TEST-14 | Phase 9 | Pending |
| TEST-15 | Phase 9 | Pending |
| TEST-16 | Phase 9 | Pending |
| TEST-07 | Phase 10 | Pending |
| TEST-08 | Phase 10 | Pending |
| TEST-09 | Phase 10 | Pending |
| TEST-10 | Phase 10 | Pending |
| TEST-11 | Phase 10 | Pending |
| TEST-12 | Phase 10 | Pending |
| TEST-13 | Phase 10 | Pending |

**Coverage:**
- v1 requirements: 37 total
- Mapped to phases: 37
- Unmapped: 0

---
*Requirements defined: 2026-03-23*
*Last updated: 2026-03-23 after roadmap creation*
