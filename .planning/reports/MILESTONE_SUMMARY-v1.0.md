# Milestone v1.0 — Project Summary

**Generated:** 2026-09-01
**Purpose:** Team onboarding and project review
**Milestone:** Protoconf — Quality & Consistency Overhaul
**Status:** All 10 phases complete, all verifications passed

---

## 1. Project Overview

Protoconf is a configuration management tool that uses **Protocol Buffers as schema** and **Starlark as the configuration language**. It compiles Starlark configs into materialized protobuf, distributes them via KV stores (Consul, etcd, ZooKeeper, Kubernetes ConfigMaps), and serves them to applications over gRPC streaming.

This milestone shipped **no new product features**. It was a quality and consistency overhaul of the existing codebase, driven by one principle:

> Every component must be testable, consistent, and free of runtime surprises — no panics in production code, no `os.Exit` in libraries, no deprecated APIs, and proper test coverage across all packages.

Four themes ran through the work:

1. **Modernization** — deprecated gRPC APIs migrated; global mutable state removed.
2. **Library hygiene** — `os.Exit` eliminated from library code and replaced with error propagation; OTel bootstrap deduplicated and made non-fatal.
3. **Security** — TLS for gRPC servers and clients; bearer-token auth on the mutation server; validated pre/post script paths.
4. **Configuration as protobuf** — all five CLI components now define their configuration as `.proto` messages, generate flags from those definitions, and load config from flags, env vars, and config files with a documented precedence order.

The last theme turned out to be the milestone's centre of gravity: Phases 7 and 8 account for 7 of the 23 plans and the milestone's only two verification blockers.

**Who it's for:** platform and infrastructure teams that want configuration reviewed, versioned, and type-checked like code, then distributed to running services without redeploys.

---

## 2. Architecture & Technical Decisions

The system's shape did not change this milestone. What changed is how components are configured, secured, and observed.

### The pipeline

```
src/*.pconf  ──compile──▶  materialized_config/*.materialized_JSON
   (Starlark)                        (protobuf, JSON-encoded)
                                              │
                                          insert
                                              ▼
                            KV store (Consul / etcd / ZooKeeper / ConfigMaps / file)
                                              │
                                           watch
                                              ▼
                             agent ──gRPC stream──▶ applications
```

The mutation server writes runtime changes back into `mutable_config/`, optionally recompiles, and runs pre/post scripts (typically git commit + push).

### Decisions that shaped this milestone

- **Proto-defined CLI configuration for every component**
  - **Why:** Consistency with protoconf's own philosophy — the agent already self-configured from `agent/config/v1/agent_config.proto`, so extending that pattern to the other four components removed the last hand-written `cliConfig` structs.
  - **How:** Four new proto packages (`server/config/v1`, `compiler/config/v1`, `inserter/config/v1`, `mutate/config/v1`), each field carrying a `json_name` that matches the existing CLI flag so the interface stayed backward compatible.
  - **Phase:** 7

- **`libprotoconf` for flag generation and env loading, not a CLI framework migration**
  - **Why:** The real problem was that configs were not proto-defined, not that `mitchellh/cli` is in maintenance mode. `configtool.NewConfig` + `SetEnvKeyPrefix` + `Environment` + `PopulateFlagSet` gave flag generation and `PROTOCONF_*` env support without rewriting the command router.
  - **Outcome:** `mitchellh/cli` survives as the subcommand router only. A full framework migration is deferred to v2 as PCLI-10.
  - **Phase:** 8

- **Track config provenance from the `flag.FlagSet`, not by comparing values against defaults**
  - **Why:** This is the milestone's most consequential decision, and it was reached the hard way. Phase 8's first design (`LayerConfigFile` + `matchesBase`) inferred "was this value explicitly set?" by diffing against an accumulating baseline. Verification found two blockers that no patch to the comparison could fix: an env var whose value coincidentally matched a config-file value was silently lost, and later-file-wins was inverted for message-typed fields. A single evolving baseline cannot distinguish "carried over from an earlier file" from "explicitly supplied."
  - **How:** `command.ConfigLayerer` records provenance from two independent sources — `flag.FlagSet.Visit` inside the `-config-file` callback (exact, ordering-based, because Go's flag package records `f.actual[name]` only *after* `Value.Set` returns) and a field-level diff against a per-call `lastResult` snapshot for env vars. Provenance entries are never removed.
  - **Phase:** 8, plans 05–06 (gap closure)

- **Shared TLS helper decoupled from proto types**
  - **Why:** `utils.TLSFiles` is a plain struct with six string fields rather than a proto type, so agent, mutation server, and the mutate CLI could all reuse it. `tls.X509KeyPair` on bytes (not `LoadX509KeyPair` on paths) supports both file and inline-PEM inputs, which the agent's proto already exposed as a oneof.
  - **Phase:** 5

- **Insecure remains the default, but warns**
  - **Why:** Backward compatibility. Every gRPC server entry point logs `slog.Warn` when starting without TLS; nothing fails. Same pattern applied to unconfigured auth.
  - **Phase:** 5, 6

- **Constant-time token comparison, pass-through when unconfigured**
  - **Why:** `crypto/subtle.ConstantTimeCompare` closes the timing-attack channel; an empty `--auth-token` passes all requests through so existing deployments keep working.
  - **Phase:** 6

- **Noop OTel providers on exporter failure**
  - **Why:** A telemetry collector being unavailable must never crash the process. `observability.Init` installs noop tracer/meter providers and returns a soft error plus an always-non-nil shutdown function, so callers never nil-check.
  - **Phase:** 3

- **`sync.Once` for Starlark resolver globals**
  - **Why:** `resolve.Allow*` are package-level vars in `go.starlark.net`. Concurrent `NewCompiler` calls raced on them. One guard, six assignments, exactly once.
  - **Phase:** 3

- **Dual reflection registration retained**
  - **Why:** `grpc_reflection_v1` is now the primary service, satisfying DEPR-02, but `grpc_reflection_v1alpha` stays registered as a compatibility shim for `grpcui@v1.4.1`, a locked transitive dependency. Documented as intentional, not a gap.
  - **Phase:** 1

---

## 3. Phases Delivered

| Phase | Name | Plans | Status | One-Liner |
|-------|------|-------|--------|-----------|
| 01 | Deprecated API Migrations | 1/1 | ✓ passed | Migrated `grpc.WithInsecure`/`grpc.Dial` to `grpc.NewClient` + `insecure.NewCredentials()` across 8 files, and reflection from v1alpha to v1 with a v1alpha shim for grpcui. |
| 02 | os.Exit Refactoring | 2/2 | ✓ passed | Removed every `os.Exit` from `compiler/lib` and `mutate`, converting constructors to `(T, error)` and propagating failures to CLI entry points that return exit codes. |
| 03 | Observability & Global State Cleanup | 2/2 | ✓ passed | Extracted the duplicated OTel bootstrap into `observability.Init` with noop fallback; guarded Starlark resolver globals with `sync.Once` and localized mutate's `grpc.ClientConn` to `Run()`. |
| 04 | Dead Code Removal | 1/1 | ✓ passed | Deleted the no-op `runtime.GOMAXPROCS` init in inserter and an unreachable error check in the filekv watch goroutine. |
| 05 | TLS Support | 2/2 | ✓ passed | Added `utils.BuildTLSConfig` + `TLSFiles` and wired TLS into the agent server, mutation server, and mutate client, with an insecure-mode warning. |
| 06 | Token Auth & Script Security | 2/2 | ✓ passed | Bearer-token gRPC interceptor with constant-time comparison, credential forwarding to pre/post scripts, and startup + TOCTOU validation of script paths. |
| 07 | Proto-Defined CLI Configs | 1/1 | ✓ passed | Four new proto config packages (server, compiler, inserter, mutate) with `json_name` fields matching existing CLI flags. |
| 08 | CLI Flag Generation & Config Loading | 6/6 | ✓ passed | Migrated all five components onto libprotoconf-generated flags, `PROTOCONF_*` env vars, and `-config-file`, then built `command.ConfigLayerer` to make precedence (flags > env > file > defaults) correct across repeated files and message-typed fields. |
| 09 | Unit Test Coverage & Infrastructure | 4/4 | ✓ passed | First test files for `command`, `fmt`, `mutate`, `devserver`, all four KV stores, and `compiler/starproto` (55 cases), plus a shared `testutil` package. |
| 10 | Placeholder Fixes & Integration Tests | 2/2 | ✓ passed | Replaced TODO placeholder assertions with real ones in four test files and added e2e tests for the mutation-script, TLS, and auth flows. |

**Phase 8 is the one to read first** if you want to understand how this codebase thinks. It took six plans, two rounds of gap closure, and produced the milestone's only reproduced blockers — and its `08-REVIEW.md` and `08-VERIFICATION.md` document the reasoning better than any other artifact here.

---

## 4. Requirements Coverage

**43 of 43 v1 requirements met.** No partial, no unmet.

| Group | IDs | Status |
|-------|-----|--------|
| Testing — Coverage | TEST-01…06 | ✅ 6/6 |
| Testing — Fix Placeholders | TEST-07…10 | ✅ 4/4 |
| Testing — Integration | TEST-11…13 | ✅ 3/3 |
| Testing — Infrastructure | TEST-14…16 | ✅ 3/3 |
| Refactoring — os.Exit | REFC-01…04 | ✅ 4/4 |
| Refactoring — OTel | REFC-05, 06 | ✅ 2/2 |
| Refactoring — Global State | REFC-07, 08 | ✅ 2/2 |
| Refactoring — Dead Code | REFC-09, 10 | ✅ 2/2 |
| Security — TLS | SECR-01…03 | ✅ 3/3 |
| Security — Auth | SECR-04…06 | ✅ 3/3 |
| Security — Scripts | SECR-07 | ✅ 1/1 |
| Deprecated APIs | DEPR-01, 02 | ✅ 2/2 |
| Proto CLI — Definitions | PCLI-01…04 | ✅ 4/4 |
| Proto CLI — Generation | PCLI-05, 06 | ✅ 2/2 |
| Proto CLI — Config Loading | PCLI-07…09 | ✅ 3/3 |

**One requirement with a caveat worth knowing:** TEST-15 ("CI enforces minimum coverage threshold"). `codecov.yml` has no numeric `coverage: status:` block. CI does run `go test -race -coverprofile` and upload to Codecov, which is what design decision D-13 scoped it to ("no hard coverage threshold enforced initially"). Coverage is *reported*, not *enforced*. If you expected a failing build below N%, it isn't there.

### Deferred to v2

| ID | Item |
|----|------|
| DEPR-03 | Migrate `jhump/protoreflect/dynamic` to official `dynamicpb` |
| SECR-08 | mTLS for mutual authentication |
| SECR-09 | Role-based authorization on config paths |
| PCLI-10 | Migrate `mitchellh/cli` to cobra or urfave/cli |

### Explicitly out of scope

- KV store unimplemented methods — the panics are intentional interface stubs that signal gaps if the methods ever get called.
- Mobile or web client SDKs.
- Any new feature development.

---

## 5. Key Decisions Log

Decisions are recorded per-phase in `{phase}-CONTEXT.md` under `## Implementation Decisions` and in each `{plan}-SUMMARY.md` frontmatter under `decisions:`. The ones with lasting consequences:

| ID / Source | Decision | Phase | Rationale |
|---|---|---|---|
| 01 D-03 | Migrate `grpc.Dial` → `grpc.NewClient` alongside `WithInsecure` | 1 | Avoids a second deprecation pass later |
| 01 D-02 | Keep v1alpha reflection registered next to v1 | 1 | grpcui@v1.4.1 is a locked transitive dep |
| 02 | Resolve-at-construction: `filepath.Abs` in `NewModuleService` | 2 | Eliminates an error path in every helper downstream |
| 03 D-04 | `Init` returns a shutdown func, always non-nil | 3 | Callers pick `defer` vs `context.AfterFunc`; no nil checks |
| 03 D-05/06 | OTel failures are soft — warn and install noop providers | 3 | Telemetry outage must not crash the process |
| 03 D-08 | `sync.Once` guard for the six `resolve.Allow*` globals | 3 | Concurrent `NewCompiler` calls raced |
| 05 D-01/02 | `TLSFiles` plain struct, `X509KeyPair` on bytes | 5 | Reusable across agent/server/mutate; supports file *and* inline PEM |
| 05 D-13/14/15 | DevServer, examples, and the compiler's in-process conn get no TLS | 5 | Local/illustrative code; friction without benefit |
| 06 | `crypto/subtle.ConstantTimeCompare`; pass-through on empty token | 6 | Timing-safe, backward compatible |
| 06 | `validateScriptPath` at startup **and** `os.Stat` in `runScript` | 6 | Defense in depth against TOCTOU |
| 07 D-07/10 | `store_address` is `repeated string`; each component owns its store enum | 7 | Future multi-address support; no premature shared proto package |
| 08 | Superseded `proto.Merge` direction-flip with `LayerConfigFile` | 8 | The one-line reversal was wrong two ways: defaults merged back over the file, and reassigning `c.config` orphaned post-`-config-file` flags |
| 08 | Replaced `matchesBase` value comparison with `ConfigLayerer` provenance | 8 | Value comparison cannot tell "explicitly set" from "unset"; see §2 |
| 08 | `markExplicitFlags` matches top-level `JSONName` only | 8 | `PopulateFlagSet` routes nested message flags onto a detached `dynamicpb` message with no top-level field number |
| 08 | `setFieldReplacing` deep-copies message fields via `proto.Clone` | 8 | `fileLayer` is a long-lived accumulator; aliasing a caller's submessage would corrupt it (08-REVIEW IN-01) |
| 09 D-13 | Coverage reporting, no enforced threshold | 9 | Evaluate a number after Phases 9+10 land |
| 10 | Real TCP listener, not bufconn, for TLS e2e | 10 | TLS needs real hostname/IP verification |

### Two accepted limitations, documented rather than solved

Both live in `command/configfile.go`'s `ConfigLayerer` doc comment, folded in from the deleted `matchesBase` before it was removed:

1. An env var whose value happens to equal the factory default is indistinguishable from unset.
2. Proto3 implicit presence makes a zero-valued scalar, `false`, or a zero-numbered enum invisible to `protoreflect.Message.Range`. Concretely: `PROTOCONF_INSERTER_STORE=consul` (where `consul` = 0) cannot override a config file's non-zero value. `-store consul` is the escape hatch, and it works because `flag.Parse` writes directly into the live message before layering runs.

---

## 6. Tech Debt & Deferred Items

### Known bugs, still open

- **`mutate/mutate.go` converts `TYPE_SINT32` with a `uint32` cast instead of `int32`** — found by the Phase 8 code review (`08-REVIEW.md` WR-02). Negative sint32 values will be wrong. Still open in PROJECT.md's Active requirements.
- **`dummykv.Exists` always returns true** — listed in PROJECT.md Active requirements as a known bug, never closed. It is a test-only store, so the blast radius is test fidelity, not production.
- **`08-REVIEW.md` WR-01: a later config file cannot clear an earlier file's repeated/map field to empty.** `setFieldReplacing` replaces a list with the new file's list, but an *absent* field in the later file leaves the earlier one standing.

### Structural debt

- **`08-REVIEW.md` WR-03: ConfigLayerer wiring is duplicated near-verbatim across all five CLI components.** Roughly 35 lines of `proto.Clone` / `SetEnvKeyPrefix` / `Environment` / `PopulateFlagSet` / `NewConfigLayerer` / `-config-file` callback repeated in `agent/command.go`, `server/server.go`, `compiler/command.go`, `inserter/inserter.go`, and `mutate/mutate.go`. Flagged during Phase 8 review and consciously not addressed.
- **`filekv.go` has 10 `go vet` warnings** — value receivers on a struct containing a mutex. Documented and deferred in `09-02-SUMMARY.md`. Pre-existing production code, not test code.
- **`08-REVIEW.md` IN-02: `agent/kv_agent_rollout_impl_test.go` spawns unsynchronized goroutines that call `t.Run`.**

### Repo weight and dead code

A repo-wide over-engineering audit run at milestone close (not part of any phase) surfaced roughly 2,600 removable lines and 6 removable dependencies. The largest items:

- **`utils/testdata/large/`** — 1,489 files, ~16.5 MB of AWS Terraform proto fixtures, embedded via `//go:embed` in a non-test package, reachable only through `LargeTestDir()`, which has **zero callers**. Every test uses `SmallTestDir()`.
- **`testutil/`** — the shared test-helper package built in Phase 9 (09-01). All three functions are unreachable; its only references are `var _ = testutil.NewAny` no-op assignments planted in three test files to make the import legal. TEST-14 is satisfied on paper by a package nothing calls.
- **`importers/` and `importers/wktbuilders/`** — 253 lines, entirely unreachable.
- **`web_ide/`** — 925 lines; its `build_wasm.sh` bazel-builds `//compiler/wasm`, which does not exist, and there is no root `WORKSPACE`. The `js`/`wasm` filesystem shim in `compiler/lib/filesystem_js.go` exists only to serve this dead target — and its build tag (`// +js -build`) is not even a valid constraint.
- **Unused direct dependencies** — `hashicorp/go-getter/v2` and `pelletier/go-toml/v2` are both in `go.mod` with zero imports.
- **`command.PrefixedUi`** — 45 lines plus ~100 lines of test re-implementing `cli.PrefixedUi`, which `mitchellh/cli` already ships.
- **`cmd/agent`, `cmd/compiler`, `cmd/inserter`, `cmd/server`** — four standalone binaries that `.goreleaser.yaml` does not build; it ships only `./cmd/protoconf`.

### Process observations

- **No RETROSPECTIVE.md exists** for this milestone.
- **No UAT was run.** All ten phases passed automated verification, but zero phases went through conversational user acceptance testing. Several verification reports note human verification was needed and not performed — Phase 2 ("cannot run the full CLI binary to confirm end-to-end exit code behavior"), Phase 5 ("requires running the agent process and a live gRPC client"), Phase 6 ("end-to-end test requires a running server; spot-check was unit-level only"). The TLS and auth paths are covered by e2e tests added in Phase 10, which substantially closes that gap, but no human has driven the binaries.
- **STATE.md drifted** — it reports `current_phase: 2` alongside `status: Milestone complete`. The phase directories on disk are authoritative.

### Deferred ideas parked in CONTEXT files

- Consolidating `KVStoreConfig` and `AgentConfig.StoreType` into a shared proto package (Phases 7, 8).
- Adding protovalidate rules to the config protos (Phases 7, 8).
- TLS for KV store connections — the agent's `store_tls` field exists but is out of SECR-01/02/03 scope (Phase 5).
- Auth for the agent read path — deliberately excluded; the agent is read-only (Phase 6).
- Property-based testing / fuzzing (Phase 9).
- `fmt` and `devserver` proto configs — judged too trivial or too composite to be worth proto-defining.

---

## 7. Getting Started

### Build and run

```bash
go build ./cmd/protoconf          # the single shipped binary
./protoconf --help
```

Subcommands: `agent`, `compile`, `devserver`, `fmt`, `insert`, `mutate`, `serve`, `mod init|sync|tidy`.

For local development, `protoconf devserver` runs the agent, compiler service, and mutation server together against a file-backed KV store.

### Tests

```bash
go test ./...
go test -race -coverprofile=coverage.txt -covermode=atomic ./...   # what CI runs
```

CI (`.github/workflows/go.yml`) builds, runs the race-enabled coverage suite on Go 1.22, and uploads to Codecov. `codecov.yml` ignores `**/*.pb.go`. There is no coverage threshold gate.

### Configuring any component

Every component now follows the same three-source model, in this precedence order:

```
flags  >  PROTOCONF_<COMPONENT>_* env vars  >  -config-file  >  proto defaults
```

`-config-file` accepts JSON, YAML, or binary protobuf and may be repeated; later files win over earlier ones. The env prefix per component is `PROTOCONF_AGENT_`, `PROTOCONF_SERVER_`, `PROTOCONF_COMPILER_`, `PROTOCONF_INSERTER_`, `PROTOCONF_MUTATE_`.

`--help` output is generated from the proto definitions, so it is always in sync with what the component actually accepts.

**Breaking change for operators:** the agent's config-vs-env ordering changed in this milestone. See `CHANGELOG.md` under Unreleased / BREAKING CHANGES.

### Where to look first

| To understand… | Read |
|---|---|
| The whole CLI surface | `cmd/protoconf/main.go` |
| Config precedence — the milestone's hardest problem | `command/configfile.go`, then `.planning/phases/08-*/08-VERIFICATION.md` |
| A component's config schema | `{component}/config/v1/*.proto` |
| Starlark → protobuf compilation | `compiler/lib/compiler.go`, `compiler/starproto/message.go` |
| Serving configs to apps | `agent/kv_agent_impl.go` (simple), `agent/kv_agent_rollout_impl.go` (rollout-aware) |
| Runtime mutations | `server/server.go` |
| TLS | `utils/tls.go` |
| Auth | `server/server.go` → `bearerTokenInterceptor` |
| Telemetry | `observability/observability.go` |
| End-to-end behavior | `test/e2e_test.go` |

### Key directories

```
agent/          gRPC agent + KV store backends (filekv, otelkv, configmaps, dummykv)
compiler/       Starlark compilation; starproto/ is the Starlark↔protobuf bridge
server/         mutation server
inserter/       materialized config → KV store
mutate/         mutation CLI client
command/        shared CLI plumbing, including ConfigLayerer
consts/         file extensions, paths, default addresses
observability/  OTel bootstrap
test/           e2e suite
.planning/      GSD planning artifacts — phases, roadmap, requirements
```

### Conventions

`CLAUDE.md` at the repo root carries the full convention set. The short version: `snake_case.go` filenames; `New<Type>` constructors; every CLI package exports `Command() (cli.Command, error)`; sentinel errors are package-level `var Err*` combined with `errors.Join`; logging is `log/slog` with structured fields.

---

## Stats

- **Timeline:** 2026-03-23 → 2026-09-01
- **Phases:** 10 / 10 complete, all verifications passed
- **Plans:** 23 / 23 executed
- **Requirements:** 43 / 43 met (4 deferred to v2)
- **Commits:** 162
- **Code changed:** 66 files, +7,167 / −642 (excluding `.planning/`)
- **All changes incl. planning artifacts:** 165 files, +26,005 / −642
- **Contributors:** Shahar Mintz (162 commits)
- **Verification debt:** 0 open items, 0 open windows
- **UAT sessions:** 0

---

*Generated by `/gsd-milestone-summary` from ROADMAP, REQUIREMENTS, PROJECT, and per-phase CONTEXT / SUMMARY / VERIFICATION / REVIEW artifacts.*
