---
gsd_state_version: "1.0"
milestone: v2.0
milestone_name: Compiler Startup Performance (Planned)
current_phase: 15
current_phase_name: Verification, Decision & Gate Flip
status: executing
stopped_at: Completed 15-02-PLAN.md
last_updated: "2026-09-09T05:06:42.393Z"
last_activity: 2026-09-09
last_activity_desc: Phase 15 execution started
state_head: a8b7d80102f73c032d881ecba478ada0ace09039
progress:
  total_phases: 5
  completed_phases: 0
  total_plans: 25
  completed_plans: 24
  percent: 0
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-09-04)

**Core value:** Every component must be testable, consistent, and free of runtime surprises
**Current focus:** Phase 15 — Verification, Decision & Gate Flip

## Current Position

Phase: 15 (Verification, Decision & Gate Flip) — EXECUTING
Plan: 3 of 3
Status: Ready to execute
Last activity: 2026-09-09 — Phase 15 execution started

Progress: [████████████████░░░░] 4/5 phases ([░░░░░░░░░░] 0%)

## Performance Metrics

**Velocity:**

- Total plans completed: 28
- Average duration: -
- Total execution time: 0 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 08 | 6 | - | - |
| 11 | 5 | - | - |
| 12 | 4 | - | - |
| 13 | 4 | - | - |
| 14 | 9 | - | - |

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
| Phase 11 P01 | 55min | 3 tasks | 8 files |
| Phase 11-concurrency-safe-lazy-registry-core P02 | 8min | 2 tasks | 2 files |
| Phase 11-concurrency-safe-lazy-registry-core P03 | 42min | 2 tasks | 4 files |
| Phase 11 P04 | 45min | 2 tasks | 4 files |
| Phase 11 P05 | 50min | 2 tasks | 4 files |
| Phase 12 P01 | 35 min | 3 tasks | 4 files |
| Phase 12 P02 | 28min | 3 tasks | 4 files |
| Phase 12 P03 | 25min | 2 tasks | 2 files |
| Phase 12 P04 | 25min | 2 tasks | 2 files |
| Phase 13 P01 | 40min | 2 tasks | 5 files |
| Phase 13 P02 | 55min | 3 tasks | 8 files |
| Phase 13 P03 | 55min | 3 tasks | 12 files |
| Phase 13 P04 | 45min | 2 tasks | 4 files |
| Phase 14 P01 | 15min | 2 tasks | 2 files |
| Phase 14 P02 | 22min | 2 tasks | 2 files |
| Phase 14 P03 | 12min | 2 tasks | 2 files |
| Phase 14 P08 | 15min | 2 tasks | 1 files |
| Phase 14 P04 | 15min | 2 tasks | 4 files |
| Phase 14 P07 | 35min | 2 tasks | 2 files |
| Phase 14 P05 | 30min | 2 tasks | 4 files |
| Phase 14 P06 | 25min | 2 tasks | 1 files |
| Phase 14 P09 | 15min | 5 tasks | 6 files |
| Phase 15-verification-decision-gate-flip P01 | 29min | 3 tasks | 3 files |
| Phase 15 P02 | 12min | 2 tasks | 2 files |

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- v2.0 roadmap (2026-09-04, revision): Phase 13 (exact symbol index) and Phase 14 (shared type-URL resolution path) merged into a single Phase 13 — splitting them left the index phase's success criteria unobservable through real behavior (nothing consults the index until the wiring phase exists, so criteria could only assert the artifact exists). Roadmap is now 5 phases (11-15), not 6; the former Phase 15/16 renumbered to 14/15.
- v2.0 roadmap (2026-09-04, revision): the unconditional global-registry seed (`utils/utils.go:38-45`) already seeds well-known types (`google/...`, `buf/validate/...`, `validate/...`, `protoconf/v1/...`) before any lazy path runs; `google.protobuf.Value` resolves via that seed with no index and no `src/` parsing. Since production mutable configs overwhelmingly carry `google.protobuf.Value` (String/Int64/Float64), the merged Phase 13 gained a success criterion asserting that resolving a mutable `google.protobuf.Value` never triggers index construction — a regression guard against putting index-build cost on the compiler's hot path.
- v2.0 roadmap (2026-09-04): type-URL resolution mechanism is a decided exact symbol index (parse-without-link, persisted under `.protoconf_cache`, content-keyed) — NOT the heuristic/scan/verification menu research left open. `proto_file` is superseded as a resolution mechanism but the field stays populated for compatibility.
- v2.0 roadmap (2026-09-04): mutation server `Init()`'s service-registration fix (CONS-01) is bundled into Phase 11, the same phase that makes the registry lazy — it is a blocking co-requirement, not a follow-up, per PITFALLS.md pitfall 1.
- v2.0 roadmap (2026-09-04): only Phase 11 (lazy parse core) and Phase 12 (growable resolver) move the measured baseline numbers (4,639ms + 260ms of the 6.97s total). Phases 13-14 are correctness-only, protecting non-compiler-hot-path consumers from regressing under the change; Phase 13 also confirms the compiler's dominant mutable-config case (`google.protobuf.Value`) never builds the symbol index, since it resolves via the existing global-registry seed.
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
- [Phase 11]: [Phase 11-01] Laziness is opt-in via NewLazyModuleService; D-01/D-02/D-03 followed as written, GetProtoRegistry()'s default behavior unchanged for the other four consumers
- [Phase 11]: [Phase 11-01] RegistryTypeResolver retries the MessageRegistry lookup after ParseAll unconditionally, discarding only ParseAll's own error, so a partial whole-tree parse can't produce a false NotFound
- [Phase 11]: Phase 11-02: Init() discovers mutation services via own eager src/ scan (D-02: registration-only, reflection calls unchanged) — CONS-01 blocking co-requirement closed before registry laziness lands
- [Phase 11]: D-04 executed: BUG-03 (go vet copylocks at compiler/lib/compiler.go) fixed in Phase 11, overriding REQUIREMENTS.md's 'deferred beyond this milestone' framing — LAZY-02 removed the invariant that made the value-copy benign (AddFile now runs mid-compile, not just at construction), making the shared-pointer fix a correctness prerequisite
- [Phase 11]: TestModSyncFdsByteIdentical compares lazy vs eager FileRegistry counts directly instead of the plan's literal corpus-size threshold — NewDescriptorRegistry seeds ~65 well-known types before any src/ parsing, exceeding the 40-file corpus regardless of laziness, so the literal threshold fails unconditionally; comparing against eager's own count preserves the distinguishability guard
- [Phase 11]: G-11-3 recorded as pre-existing (reproduces on b69e3b2), not attributed to Phase 11's LAZY/CONS work
- [Phase 11]: LoadFromLockFile is the single chokepoint fix location for the nil-map invariant; no per-caller nil-checks added
- [Phase 11]: G-11-7 recorded as pre-existing (reproduces on b69e3b2), not attributed to Phase 11's own LAZY-04/CONS-01 work; closed under Phase 11 by explicit user decision, matching G-11-3's treatment in 11-04
- [Phase 11]: mod sync's empty-descriptor-set guard is keyed on utils.DescriptorRegistry.LocalFileCount() == 0, never on GetterUrl == "" -- a fully downloaded dependency with a bad sourcePath corrupts the lock identically to an unsynced one, so a GetterUrl check would leave that shape broken
- [Phase 12]: Growable FilesResolver hangs off the existing d.mu — no second lock; growth is the single registerFileLocked insert point called from both recordFileLocked and ParseAll's diff loop — Keeps FileRegistry and filesResolver from ever diverging and avoids a lock-ordering hazard in recordFileLocked
- [Phase 12]: ParseFilesX's resolver-hit branch now returns the registry's own canonical descriptor via FileDescriptor(resolved.Path()) before falling back to desc.WrapFile — Closes the RSLV-03 pointer-identity hazard growth made reachable, while preserving the wrap fallback the mutation server's hand-registered well-known files depend on
- [Phase 12]: Deleted the dead config.protoResolver field instead of rewiring it to the growable TypeResolver — A frozen construction-time snapshot with zero readers is a stale-resolution hazard; a future consumer should read c.parser.TypeResolver directly
- [Phase 12]: Proved SAFE-01 with two race tests: extended TestConcurrentCompile with a post-g.Wait() resolver-view assertion block, and added a dedicated utils.TestRegisterFileRacesRangeFiles with unpaced tight-loop readers forcing the RegisterFile-vs-RangeFiles interleaving continuously. — TestConcurrentCompile's goroutines spend most of their time inside protoparse.ParseFiles with no lock held, so the read/write window is rare there; only a dedicated test with continuously-running readers reliably forces it. Sanity-checked the dedicated test's failure-detection capability by temporarily removing the lock and confirming WARNING: DATA RACE before restoring.
- [Phase 12]: [Phase 12-04]: Fallback for eager-registry hand-registered files lives in ParseFilesX (errors.Is on ErrNoGrowableResolver), not in DescriptorRegistry.FindFileByPath, keeping FindFileByPath's sentinel contract intact for growable_resolver_test.go Test 3 and other callers.
- [Phase 13]: [Phase 13] protojson's unmarshalAny discards the wrapped resolver error's identity through its own internal/errors.New; errors.Is(readConfigErr, protoregistry.NotFound) can never hold on a ReadConfig-returned error — Verified against go.mod-pinned google.golang.org/protobuf v1.36.12 source; pinned the sentinel contract at the resolver boundary (direct FindMessageByURL call) instead
- [Phase 13]: [Phase 13] scanCandidateLimit = 32 needed no adjustment after edge-case testing — The 33-file candidate-limit stress test confirms the escalation boundary fires exactly at the constant's derivation
- [Phase 13]: 13-02: exact symbol index built via ParseFilesButDoNotLink, persisted content-keyed under .protoconf_cache (dirhash.HashDir), wired as Tier 3 behind the D-01 scan tier; measured 1.2s cold build / ~30ms dirhash / ~13.6ms warm cache read on the 799-proto corpus
- [Phase 13]: D-02 executed: deleted DescriptorRegistry.ParseAll, its eagerFallback latch, and FellBackToEager(). Developer confirmed option 1 at the Task 1 checkpoint. — Orchestrator's grep found exactly one non-test caller of each (parser.go's resolution chain and the compile-finished log line, both handled by this plan), so no carve-out for a surviving caller (option 3) was needed.
- [Phase 13]: Deleted (not re-pointed) utils/growable_resolver_test.go's TestParseAllRegistersIntoFilesResolver and TestFilesResolverRegistrationErrorsStayZero. — Their registration-diff behavior lived entirely inside ParseAll's own deleted diff loop; no production caller runs Import/Parse on a lazy registry, so re-pointing would only test reimplemented ParseAll glue with no real analog.
- [Phase 13]: 13-04: Closed CONS-05 -- loadMutable resolves the mutable value's type once via desc.WrapMessage(mt.Descriptor()) instead of a second direct MessageRegistry lookup, which could return (nil,nil) on a cold registry and panic downstream; also fixed a nil-panic on an absent mutable-config value via GetValue()/GetTypeUrl().
- [Phase 13]: 13-04: The literal plan fixture (top-level TestMessage value, nested inner Any) does not itself force loadMutable's bypass to fail, since parser.ReadConfig pre-warms the same MessageRegistry before either lookup runs; the RED test isolates the divergence with a cold moduleService instead.
- [Phase 14]: [Phase 14-01]: D-02 diagnostic surfaces from parser.ReadConfig's protojson unmarshal of the Any field, not XXXinsertVersion's later FindMessageByURL — both route through the same resolveTiers chain so the error text is identical either way — Verified with scratch tests before writing the fixture-based assertion
- [Phase 14]: [Phase 14-02] agent/filekv flipped to lib.NewLazyModuleService (D-01), a one-line diff -- Get already routes through parser.ReadConfig's tiered TypeResolver, no second edit needed — Proven, not assumed: TestGetResolvesTypeAbsentFromConstructionSnapshot shows the type is absent from the construction snapshot both before and after a successful Get
- [Phase 14]: [Phase 14-02] TDD task 2's four proof tests committed as a single test(14-02) commit with no feat/refactor -- Task 1 already shipped the only implementation change, matching 14-01's precedent — No new implementation exists for GREEN to make pass; the tests exist purely to prove Task 1's flip correct
- [Phase 14]: [Phase 14-03] Mutation server flipped to lib.NewLazyModuleService (D-01) and MutateConfig's marshal resolver swapped to s.parser.TypeResolver (D-03)
- [Phase 14]: [Phase 14-03] Init mirrors six well-known files onto the retained discovery-registry resolver and wires both reflection.ServerOptions.DescriptorResolver fields to it (D-06), proven by TestReflectionDescribesCustomAndBuiltinServices; no new struct field added, preserving D-04 narrowness
- [Phase 14]: SAFE-03 escalation guard proven on the serving *utils.DescriptorRegistry handle: a single index-tier-only resolution grows the loaded-file count by exactly one closure, never the full candidate/corpus count; eight sequential resolutions over one long-lived registry stay bounded and end far below corpus size.
- [Phase 14]: [Phase 14] [Phase 14-04] Fingerprinting/log-on-change lives inside collectExamples (not GenReflectionUI) since the mandated two-value return signature has no room for a third failures slice; collectExamples is production-only-called-from GenReflectionUI
- [Phase 14]: [Phase 14] [Phase 14-04] Updated TestProtoconfMutationServer_GenReflectionUI to expect a non-nil error naming bad_json/bad_proto_file -- D-05 correctly surfaces those pre-existing ReadConfig failures instead of silently swallowing them
- [Phase 14]: [Phase 14] [Phase 14-05] mutate CLI flipped to lib.NewLazyModuleService and parser.TypeResolver (D-01/D-03), completing all four consumer conversions; fixed a pre-existing server/legacy.go proto.Merge panic on cross-package conversion (Rule 1), exposed by the first real end-to-end mutate-CLI-to-server round trip
- [Phase 14]: [Phase 14] [Phase 14-05] Corrected the plan's literal end-state gate to exclude _test.go files -- 14-01/14-02's intentional snapshot-vs-tiered contrast test assertions are verification code, not a second production resolution source, and the gate's own worked example predated those tests
- [Phase 14]: Task 1's 16 clients wrap the same fixture type (test.v1.TestMessage), resolved once before the writer wave starts, rather than 16 distinct types -- Task 2 owns forcing independent first-time resolutions under concurrency.
- [Phase 14]: GenReflectionUI is invoked with a bare grpc.NewServer() per the plan's literal instruction (no reflection registered), so standalone.HandlerViaReflection fails with Unimplemented on every call -- harmless, since collectExamples() (the concurrency-relevant read of mutable_config/) runs and completes before that failure.
- [Phase 14]: Phase 14: SAFE-02 validated for the mutation server via 3/3 lock-removed runs of TestMutationServerResolverTightLoopIsRaceFree reporting WARNING: DATA RACE on recordFileLocked's map write, restored via git checkout -- utils/utils.go byte-identical.
- [Phase 14]: [Phase 14-09]: Symlink-inside-protoconfRoot escape accepted as risk (T-14-24) rather than fixed — filepath.EvalSymlinks would close it at a stat-syscall cost on the agent's hot read path; planting such a symlink already requires config-repo write access, a strictly larger compromise than the unauthenticated remote read this plan closes
- [Phase 14]: [Phase 14-09]: resolveKeyPath stays private to agent/filekv, not hoisted for MutateConfig's second call site — the two roots (protoconfRoot vs protoconfRoot/mutable_config) and failure shapes (store.KVPair errors vs logError-wrapped gRPC errors) differ; the plan scoped this as one local containment check at one call site
- [Phase 15]: TestCompilerStartupScaling stays in the existing -race Run coverage step rather than moving to a new non-race step — Its allocation ratio is race-insensitive (0.91x plain, 1.02x under -race); moving it saves under 0.5% of an ~11.5-12min job and adds bookkeeping that can silently rot into running twice or nowhere; it also keeps its Codecov contribution
- [Phase 15]: GATE-02 budget threshold calibrated to 160ms from a real ubuntu-latest CI observation (77.58ms, run 34311638861), per D-03's 2x rule — A threshold chosen from any number already written in a planning document (all measured on darwin/arm64) is the error D-03 exists to prevent
- [Phase 15]: D-04/D-05 executed: CHANGELOG.md and README.md document that protoconf compile no longer parses unreferenced protos, with buf named as remedy and an explicit buf-lint/buf.yaml caveat so no reader assumes protoconf already covers whole-tree validation — GATE-03 requires a written decision, not silent absorption of Phase 13/14's D-02 behavior change

### Pending Todos

None yet.

### Blockers/Concerns

- Phase 11's mutation-server `Init()` fix (CONS-01) is a hard blocking co-requirement, not deferrable — a lazy registry with unfixed `Init()` silently drops every custom mutation service registration for the process lifetime (PITFALLS.md pitfall 1, orchestrator-verified most severe finding).
- Phase 13's symbol index build shape (parse-without-link) and its shared type-URL resolution path are new design surface not present in prior milestones — no existing pattern in this codebase to copy; plan this phase with extra care. The two pieces are now a single merged phase (was split into 13/14 in the first draft), so both land together in one plan pass.
- `add_validator`'s last-write-wins clobbering (BUG-01) must not be "fixed" incidentally by recoupling validator discovery to proto registry order during Phase 11 — keep `loadValidators`' filesystem-walk order explicitly decoupled from load()-driven proto order.

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
| 260904-fwk | Synthetic proto corpus generator + scaling benchmark for compiler startup — alloc-ratio gate (7.24x today vs 2.0x target) is the lazy-loading milestone's definition of done | 2026-09-04 | c83f249 | [260904-fwk-add-a-synthetic-proto-corpus-generator-a](./quick/260904-fwk-add-a-synthetic-proto-corpus-generator-a/) |

## Session Continuity

Last session: 2026-09-09T05:06:42.367Z
Stopped at: Completed 15-02-PLAN.md
Resume file: None
