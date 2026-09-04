---
gsd_state_version: 1.0
milestone: v1.0
current_phase: 2
current_phase_name: os.Exit Refactoring
status: Milestone complete
stopped_at: Quick task 260904-f5j complete — loadValidators now walks srcDir for validator files instead of ranging the descriptor registry (unblocks lazy proto loading)
last_updated: "2026-09-04T04:15:00.000Z"
state_head: 14e5f14
progress:
  total_phases: 10
  completed_phases: 10
  total_plans: 23
  completed_plans: 23
milestone_name: milestone
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-09-01)

**Core value:** Every component must be testable, consistent, and free of runtime surprises
**Current focus:** Phase 2 — os.Exit Refactoring

## Current Position

Phase: 2 — os.Exit Refactoring
Plan: Not started

## Performance Metrics

**Velocity:**

- Total plans completed: 6
- Average duration: -
- Total execution time: 0 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 08 | 6 | - | - |

**Recent Trend:**

- Last 5 plans: -
- Trend: -

*Updated after each plan completion*
| Phase 01 P01 | 1363 | 2 tasks | 8 files |
| Phase 02 P02 | 588 | 1 tasks | 14 files |
| Phase 02 P01 | 15 | 2 tasks | 11 files |
| Phase 03 P01 | 420 | 2 tasks | 3 files |
| Phase 03 P02 | 5 | 2 tasks | 2 files |
| Phase 04 P01 | 3 | 2 tasks | 2 files |
| Phase 05 P01 | 77 | 1 tasks | 2 files |
| Phase 05 P02 | 8 | 2 tasks | 3 files |
| Phase 06 P01 | 8 | 1 tasks | 2 files |
| Phase 06 P02 | 183 | 2 tasks | 2 files |
| Phase 07 P01 | 164 | 2 tasks | 8 files |
| Phase 08 P01 | 900 | 2 tasks | 4 files |
| Phase 08 P02 | 343 | 2 tasks | 4 files |
| Phase 09 P01 | 180 | 2 tasks | 3 files |
| Phase 09 P04 | 335 | 2 tasks | 3 files |
| Phase 09 P02 | 900 | 2 tasks | 7 files |
| Phase 09 P03 | 600 | 2 tasks | 2 files |
| Phase 10 P02 | 123 | 2 tasks | 1 files |
| Phase 10 P01 | 6 | 2 tasks | 4 files |
**Per-Plan Metrics:**

| Plan | Duration | Tasks | Files |
|------|----------|-------|-------|
| Phase 08 P03 | 20min | 2 tasks | 4 files |
| Phase 08 P04 | 25min | 3 tasks | 9 files |
| Phase 08 P05 | 30min | 2 tasks | 4 files |
| Phase 08-cli-flag-generation-config-loading P06 | 45min | 2 tasks | 8 files |

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- Keep KV store panic stubs: Intentional interface satisfaction; panics signal future needs
- Token-based auth over mTLS: Simpler to implement and forward to scripts as env vars
- Proto-defined CLI configs: Consistency with protoconf's own philosophy; agent already does this
- Migrate jhump/protoreflect to dynamicpb: Deferred to v2 — large scope, touches compiler/starproto extensively
- [Phase 01]: Register both grpc_reflection_v1 and grpc_reflection_v1alpha in server.go: v1 is primary, v1alpha kept for grpcui@v1.4.1 backward compatibility
- [Phase 01]: Use passthrough:///bufnet as grpc.NewClient target for in-process bufconn: grpc.NewClient requires non-empty DNS-resolvable target
- [Phase 02]: Fix all NewModuleService/NewCompiler caller sites as Rule 3 deviation to unblock the full project build
- [Phase 02]: Resolve filepath.Abs at construction time in NewModuleService - eliminates error propagation through all string-returning helpers
- [Phase 02]: NewCompiler/NewCompilerService/NewProtoconfMutationServer all return errors - library code must propagate to CLI entry points, never silently fail
- [Phase 03]: observability.Init returns (shutdown, error) to let callers choose shutdown strategy
- [Phase 03]: noop providers installed on exporter failure so OTel instrumentation downstream never panics
- [Phase 03]: Init always returns non-nil shutdown function for safe deferred calls
- [Phase 03]: sync.Once guards all six resolve.Allow* assignments so concurrent NewCompiler calls are race-free
- [Phase 03]: grpc.ClientConn localized to Run() — no package-level mutable connection state needed
- [Phase 05]: TLSFiles is a plain struct (not tied to proto types) for reuse across agent, server, and mutate CLI
- [Phase 05]: Use tls.X509KeyPair with bytes (not tls.LoadX509KeyPair) to support both file and text PEM inputs
- [Phase 05]: CAFile/CAText sets both ClientCAs pool and RequireAndVerifyClientCert for mutual TLS
- [Phase 05]: GenReflectionUI bufconn stays insecure.NewCredentials() — in-process loopback, TLS adds no value
- [Phase 05]: mutate CLI insecureTLS field name avoids collision; no TLS flags defaults to insecure.NewCredentials() for backward compat
- [Phase 06]: Use crypto/subtle.ConstantTimeCompare to prevent timing attacks on token comparison
- [Phase 06]: bearerTokenInterceptor pass-through when authToken empty for backward compatibility
- [Phase 06]: validateScriptPath rejects bare command names implicitly via existence check, enforcing absolute path convention
- [Phase 06]: Defense-in-depth os.Stat in runScript handles TOCTOU between startup validation and script execution
- [Phase 07]: Flat TLS strings in ServerConfig to match existing server CLI flags for Phase 8 compatibility
- [Phase 07]: InserterConfig defines own StoreType enum (no file, adds configmaps) per D-10 no cross-component imports
- [Phase 07]: InserterConfig.store_address is repeated string for future multi-address support per D-07
- [Phase 07]: compiler_address added to CompilerConfig despite omission in D-06 spec — real flag exists in compiler/command.go
- [Phase 08]: PROTOCONF_COMPILER_ADDR legacy env var preserved in runScript for backward compat with existing mutation scripts
- [Phase 08]: ~~proto.Merge direction for config-file loading: file overrides env vars~~ — SUPERSEDED in 08-03/08-05/08-06. PCLI-09 requires flags > env vars > config file > proto defaults, so env vars now win over config files
- [Phase 08]: Provenance is recorded from the flag.FlagSet, never inferred by comparing a value to the compiled-in default — value comparison cannot tell "explicitly set to the default" from "unset", and loses an env var to a config file that coincidentally matches it
- [Phase 08]: command.ConfigLayerer is the single layering entry point for all five components; the free LayerConfigFile function and matchesBase helper were deleted in 08-06 so no component can be wired to the defective path
- [Phase 08]: setFieldReplacing deep-copies message-typed values — the layerer's accumulated file layer outlives a single call and must not alias a caller's message
- [Phase 08]: Use default consul address 127.0.0.1:8500 when StoreAddress empty in inserter (parity with etcd/zookeeper defaults)
- [Phase 09]: NewTestProtoconfRoot delegates to testdata.SmallTestDir() which already provides isolated temp dir per call
- [Phase 09]: testutil imports no protoconf service protos to prevent circular dependency risk across all packages
- [Phase 09]: Use well-known proto types (Duration, Struct, DescriptorProto) as test fixtures in starproto tests — no .proto files needed
- [Phase 09]: Use kubernetes.Interface instead of *kubernetes.Clientset in configmaps.Store to enable fake client injection in tests
- [Phase 09]: Use google.protobuf.Duration as dynamic message fixture in mutate tests — no .proto files needed, has both int64 and int32 fields
- [Phase 09]: devserver Run tests use goroutine + time.After since Run blocks on signal.NotifyContext with no external context injection
- [Phase 10]: Real TCP listener required for TLS e2e tests — TLS requires proper hostname/IP verification, bufconn cannot carry TLS
- [Phase 10]: makeTokenInterceptor duplicates unexported server.bearerTokenInterceptor — subtle.ConstantTimeCompare used to match production timing-safe behavior
- [Phase 10]: with_config_rollout fixture has no proto_file field so wantProtoFile guarded by empty check to avoid false failures
- [Phase 10]: no_rollout test case uses full 40-char commit hash to satisfy inserter[0:8] slice requirement
- [Phase 08]: Superseded D-03's proto.Merge(orig, config) file-loading mechanism with command.LayerConfigFile, since the one-line merge-direction reversal was wrong in two independent ways — Reversing proto.Merge(orig, c.config) alone would make factory defaults in orig beat the file, and reassigning c.config orphans flags parsed after -config-file
- [Phase 08]: command.LayerConfigFile base accumulates only the config-file layer across multiple -config-file flags, never env/flag values — Lets a second file be told apart from the first without also needing to exclude env-supplied fields from that comparison
- [Phase 08]: Replicated 08-03's command.LayerConfigFile rewiring across compiler, inserter, mutate and agent, closing PCLI-09 project-wide
- [Phase 08]: agent/command.go's precedence comment is newly added (not replaced) and explicitly flags the config-vs-env behavior change for operators
- [Phase 08]: Replaced command.LayerConfigFile's value-comparison provenance with command.ConfigLayerer, a field-number provenance set recorded from flag.FlagSet.Visit and env-difference-against-lastResult — Closes VERIFICATION.md gaps #7 (env value coinciding with an earlier file's value was silently lost) and #8 (later-file-wins was inverted for message-typed tls_config/store_tls fields) at the root: 08-REVIEW.md CR-01 shows value equality against one accumulating baseline cannot distinguish explicitly-supplied from carried-over.
- [Phase 08]: Retained LayerConfigFile and matchesBase as superseded-but-compiling free functions in command/configfile.go rather than deleting them in 08-05 — compiler, server, inserter, and mutate still call the free function; 08-06 owns migrating them onto ConfigLayerer and removing the superseded code, so 08-05 must keep the package compiling for those four components.
- [Phase 08]: Generalized command.ConfigLayerer from the agent (08-05) to serve/compile/insert/mutate and removed the superseded LayerConfigFile/matchesBase pair — Closes PCLI-09 for all five CLI components, not only the agent; leaving two layering entry points would let a future component be wired to the defective one
- [Phase 2]: [260901-wom]: golangci-lint removed entirely from trunk.yaml rather than re-pinned — trunk CLI install blocked by sudo, no v1-line pin can typecheck go1.25.8; go build/vet/test already cover Go correctness
- [Phase 2]: [260901-wom]: buf-lint moved to trunk lint.disabled (not deleted) — 73 findings require enum/package renames that break wire compatibility, a hard CLAUDE.md constraint
- [Phase 2]: [quick-260903-c93]: Upgraded protovalidate-go v0.6.2 -> v0.8.0 (option-b); rejected v1.4.0 (module rename + Go 1.26 floor + legacy/ PGV removal, which breaks CLAUDE.md backward-compat constraint)

### Pending Todos

None yet.

### Blockers/Concerns

- Phase 9 (Unit Test Coverage) depends on Phase 2 (os.Exit Refactoring) so tests can test real error paths — plan Phase 9 only after Phase 2 is complete
- Phase 10 (Integration Tests) depends on both Phase 6 (Auth) and Phase 9 (Unit Tests) — schedule last

### Quick Tasks Completed

| # | Description | Date | Commit | Directory |
|---|-------------|------|--------|-----------|
| 260901-vaj | Merge 5 stale security dependency bumps (grpc, x/net, go-git, go-getter, otel) + raise Go floor to 1.25.8 | 2026-09-01 | 1817300 | [260901-vaj-merge-5-stale-security-dependency-bumps-](./quick/260901-vaj-merge-5-stale-security-dependency-bumps-/) |
| 260901-wom | CI hardening: Lint workflow (golangci-lint + buf breaking + actionlint), buf.yaml excludes, hardened go.yml, consolidated renovate.json. Trunk job dropped — its pinned tools had rotted upstream | 2026-09-01 | 74a193b | [260901-wom-ci-hardening-enforce-lint-via-trunk-gate](./quick/260901-wom-ci-hardening-enforce-lint-via-trunk-gate/) |
| 260902-14f | Fix dummykv pubSub races — lost watcher registrations and duplicate delivery; TestProtoconfKVAgentRollout_SubscribeForConfig now 20/20 | 2026-09-02 | 807bf7b | [260902-14f-fix-dummykv-pubsub-registration-race-roo](./quick/260902-14f-fix-dummykv-pubsub-registration-race-roo/) |
| 260902-cov | Give codecov a 1% project threshold and mark patch informational, so codecov/project stops failing on rounding noise | 2026-09-02 | 3238abe | — |
| 3 | Give codecov a 1% threshold so codecov/project stops failing on rounding noise | 2026-09-01 | 3238abe | — |
| 260902-ub9 | Make OpenTelemetry opt-in: off by default for both agent and mutation server, enabled via -enable-otel flag / PROTOCONF_{AGENT,SERVER}_ENABLE_OTEL | 2026-09-02 | (see branch) | [260902-ub9-make-otel-opt-in-off-by-default-enable-o](./quick/260902-ub9-make-otel-opt-in-off-by-default-enable-o/) |
| 260902-eie | Resolve google.protobuf.Any types when the inserter marshals config.json (backport of upstream PR #496) | 2026-09-02 | 841ca1b | [260902-eie-fix-any-resolution-in-inserter-json-mars](./quick/260902-eie-fix-any-resolution-in-inserter-json-mars/) |
| 260902-erj | Fix agent startup against etcd — store health probe used "/", which etcd normalizes to an empty key and rejects (backport of upstream PR #496) | 2026-09-02 | db33bbd | [260902-erj-fix-etcd-agent-startup-invalid-store-hea](./quick/260902-erj-fix-etcd-agent-startup-invalid-store-hea/) |
| 260902-hp5 | Fix filekv data race and double-close between Close() and readEvents(); make closeWatchers locked and idempotent; sound pointer receivers (10 copylocks) | 2026-09-02 | ffda1bb | [260902-hp5-fix-filekv-data-race-and-double-close-be](./quick/260902-hp5-fix-filekv-data-race-and-double-close-be/) |
| 260902-ggd | Serve GetConfig to non-gRPC clients over plain HTTP via connectrpc vanguard-go, using google.api.HttpBody for verbatim JSON passthrough | 2026-09-02 | a412e19 | [260902-ggd-serve-getconfig-over-plain-http-via-conn](./quick/260902-ggd-serve-getconfig-over-plain-http-via-conn/) |
| 260902-f8i | Add GetConfig one-shot RPC to ProtoconfService — both agent impls, filekv.Get, legacy passthrough (backport of upstream PR #496) | 2026-09-02 | d871d10 | [260902-f8i-add-getconfig-one-shot-rpc-to-protoconfs](./quick/260902-f8i-add-getconfig-one-shot-rpc-to-protoconfs/) |
| 260903-c93 | Upgrade protovalidate-go v0.6.2 -> v0.8.0 (option-b); rejected v1.4.0 (module rename, Go 1.26 floor, legacy PGV package removal breaking CLAUDE.md backward-compat constraint) | 2026-09-03 | 24aab2b | [260903-c93-upgrade-protovalidate-go-to-v1-4-0](./quick/260903-c93-upgrade-protovalidate-go-to-v1-4-0/) |
| 260904-f5j | Fix loadValidators to walk srcDir for *.proto-validator files instead of ranging the descriptor registry — prerequisite for lazy proto loading (validators on unreached protos would be silently skipped); CompileFile 197ms -> 184ms | 2026-09-04 | c8a6ff6 | [260904-f5j-fix-loadvalidators-to-walk-the-filesyste](./quick/260904-f5j-fix-loadvalidators-to-walk-the-filesyste/) |

## Session Continuity

Last session: 2026-09-04T04:15:00.000Z
Stopped at: Quick task 260904-f5j complete — loadValidators now walks srcDir for validator files instead of ranging the descriptor registry (unblocks lazy proto loading)
Resume file: None
