# Roadmap: Protoconf

## Milestones

- ✅ **v1.0 Quality & Consistency Overhaul** — Phases 1-10 (shipped 2026-03-31)
- 📋 **v2.0 Compiler Startup Performance** — Phases 11-15 (planned)

## Phases

**Phase Numbering:**

- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)
- Phase numbering is continuous across milestones — v2.0 starts at Phase 11, not Phase 1

Decimal phases appear between their surrounding integers in numeric order.

<details>
<summary>✅ v1.0 Quality & Consistency Overhaul (Phases 1-10) — SHIPPED 2026-03-31</summary>

- [x] **Phase 1: Deprecated API Migrations** - Replace grpc.WithInsecure and v1alpha reflection with current stable APIs (completed 2026-03-23)
- [x] **Phase 2: os.Exit Refactoring** - Remove os.Exit from library code and propagate errors to CLI entry points
- [x] **Phase 3: Observability & Global State Cleanup** - Extract shared OTel bootstrap and remove global mutable state (completed 2026-03-27)
- [x] **Phase 4: Dead Code Removal** - Remove unnecessary init functions and dead error checks
- [x] **Phase 5: TLS Support** - Add TLS to gRPC servers and clients with insecure-mode warning (completed 2026-03-28)
- [x] **Phase 6: Token Auth & Script Security** - Add token-based mutation auth with credential forwarding and script validation
- [x] **Phase 7: Proto-Defined CLI Configs** - Define protobuf messages for all component configurations (completed 2026-03-28)
- [x] **Phase 8: CLI Flag Generation & Config Loading** - Generate CLI flags from protos and add env/file config loading (completed 2026-09-01)
- [x] **Phase 9: Unit Test Coverage & Infrastructure** - Add test files for untested packages and shared test helpers (completed 2026-03-31)
- [x] **Phase 10: Placeholder Fixes & Integration Tests** - Replace placeholder assertions and add e2e integration tests (completed 2026-03-31)

Full phase detail archived under `.planning/milestones/v1.0-phases/`.

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Deprecated API Migrations | 1/1 | Complete | 2026-03-23 |
| 2. os.Exit Refactoring | 2/2 | Complete | 2026-03-27 |
| 3. Observability & Global State Cleanup | 2/2 | Complete | 2026-03-27 |
| 4. Dead Code Removal | 1/1 | Complete | 2026-03-28 |
| 5. TLS Support | 2/2 | Complete | 2026-03-28 |
| 6. Token Auth & Script Security | 2/2 | Complete | 2026-03-31 |
| 7. Proto-Defined CLI Configs | 1/1 | Complete | 2026-03-28 |
| 8. CLI Flag Generation & Config Loading | 6/6 | Complete | 2026-09-01 |
| 9. Unit Test Coverage & Infrastructure | 4/4 | Complete | 2026-03-31 |
| 10. Placeholder Fixes & Integration Tests | 2/2 | Complete | 2026-03-31 |

</details>

### 📋 v2.0 Compiler Startup Performance (Planned)

**Milestone Goal:** `protoconf compile` completes in under 200ms on a repository of any size, by parsing and linking only the protos a config actually reaches — without silently breaking the consumers that currently assume the registry is complete.

**Already shipped toward this milestone (not phases of this roadmap):** the `loadValidators` filesystem-walk fix (quick 260904-f5j) and the synthetic corpus generator + `TestCompilerStartupScaling` scaling gate (quick 260904-fwk).

**Why 5 phases:** the strict dependency chain is on-demand parse primitive → growable resolver → index-backed shared resolution → non-compiler consumer correctness → verification — five natural links. Type-URL resolution's index build and its shared consulting path were originally drafted as two phases (index, then wiring), but that split leaves the index phase's success criteria unobservable through real behavior — nothing consults the index until the next phase exists, so its criteria could only assert that an artifact exists, not that anything works differently for a user or operator. Phase 13 below merges the two: the index is built AND consulted by the same phase, so every criterion is anchored in an observable round-trip (a config resolves correctly, a cache is reused, an unnecessary build is avoided) rather than an internal data structure. Consistent with this project's `fine` granularity setting, every phase below has 4-10 requirements and a coherent, independently-verifiable deliverable.

- [ ] **Phase 11: Concurrency-Safe Lazy Registry Core** - `GetProtoRegistry()` stops bulk-parsing `src/`; parse-on-demand is memoised and thread-safe; the mutation server's service catalog survives the switch
- [ ] **Phase 12: Growable Resolver Views & Race Safety** - Resolvers grow incrementally instead of snapshotting, and concurrent compiles are proven race-free
- [ ] **Phase 13: Exact Symbol Index & Shared Type-URL Resolution** - Every message symbol under `src/`, nested types included, resolves to its declaring file via a persisted, content-keyed, parse-without-link index, consulted through one shared resolution path that every nested `Any` lookup — including the compiler's own mutable-config load — resolves through
- [ ] **Phase 14: Non-Compiler Consumer Correctness** - Mutation server, inserter, agent, and reflection UI resolve types correctly and safely with no regression
- [ ] **Phase 15: Verification, Decision & Gate Flip** - The scaling gate flips green, the error-surface change is documented, and the milestone's numbers are recorded

## Phase Details

### Phase 11: Concurrency-Safe Lazy Registry Core

**Goal**: A compile no longer pays for the whole repository's proto tree, and the mutation server's service catalog doesn't silently go dark under that change.
**Depends on**: Nothing new this milestone (builds on the already-shipped `loadValidators` fix and scaling-gate harness)
**Requirements**: LAZY-01, LAZY-02, LAZY-03, LAZY-04, LAZY-05, CONS-01
**Success Criteria** (what must be TRUE):

  1. Compiling a config in a large repository no longer incurs a delay proportional to total repository proto count — construction cost is dominated by the config's own dependency graph, not `src/`'s size.
  2. Requesting the same proto file twice within one compile costs one parse — the second request is a map lookup.
  3. `protoconf mod sync` still writes a `.fds` cache file identical in content to before this change.
  4. An operator can see, from compiler output, how many proto files a given compile actually loaded.
  5. A custom gRPC mutation service defined anywhere under `src/` is registered and reachable at server startup, before any config has been compiled — proven by a fixture service and a test that fails without the fix.

**Measured impact**: Moves the numbers — this phase eliminates the 4,639ms eager parse+link, 94% of the 6.97s baseline.
**Plans**: TBD

Plans:
- [ ] 11-01: TBD

### Phase 12: Growable Resolver Views & Race Safety

**Goal**: The compiler's live resolvers grow safely and correctly as new protos are demanded mid-compile, without rebuilding a snapshot or racing under concurrent load.
**Depends on**: Phase 11
**Requirements**: RSLV-01, RSLV-02, RSLV-03, SAFE-01
**Success Criteria** (what must be TRUE):

  1. Compiling a config that loads proto A, then proto B, then re-references A resolves all three correctly in one pass.
  2. Demanding a newly-requested proto mid-compile does not trigger a rebuild of resolver state over every previously-loaded file.
  3. Two concurrent compiles against one shared compiler, each reaching a proto the other hasn't touched, complete without error and without a data race under `go test -race`.

**Measured impact**: Moves the numbers — this phase eliminates the 260ms resolver-snapshot rebuild. Together with Phase 11, this closes the full 6.97s → ~200ms compile-path gap; everything after this phase is correctness work protecting consumers that were never on this budget.
**Plans**: TBD

Plans:
- [ ] 12-01: TBD

### Phase 13: Exact Symbol Index & Shared Type-URL Resolution

**Goal**: Any message symbol declared anywhere under `src/`, including nested types, resolves to its declaring file without linking the whole repository, and every nested `google.protobuf.Any` in a config — including one reached only through a mutable config load — resolves through one shared code path backed by that index, not a per-consumer implementation.
**Depends on**: Phase 12
**Requirements**: TYPE-01, TYPE-02, TYPE-03, TYPE-04, TYPE-05, TYPE-06, TYPE-07, TYPE-08, TYPE-09, CONS-05
**Success Criteria** (what must be TRUE):

  1. A materialized config carrying a nested `Any` at any depth — including one reached only through a mutable config load — round-trips correctly: `parser.ReadConfig` resolves every nested `@type` through one shared resolution path, and grepping the codebase for type-URL resolution logic finds that single implementation, not one per consumer.
  2. A repeat compile against an unchanged `src/` tree reuses the persisted, content-keyed index under `.protoconf_cache` instead of rebuilding it; changing any `.proto` under `src/` invalidates it on the next run, so a stale index is never served.
  3. Compiling a config whose only mutable value is a `google.protobuf.Value` — String, Int64, or Float64, the common production shape — resolves correctly via the existing unconditional global-registry seed and never triggers index construction at all.
  4. When resolution does require the index — an unknown, custom-type symbol — looking it up returns the file that declares it, including for a nested type, and only the files actually referenced by resolved symbols get linked afterward; building the index itself parses every file under `src/` but links none of them.

**Measured impact**: Correctness/infrastructure only — not required to hit the 200ms compile-path target, which Phases 11-12 already deliver. This phase exists so type-URL resolution never falls back to the 4.6s eager path, and criterion 3 guards that the compiler's dominant production case (mutable `google.protobuf.Value`) stays on the near-zero-cost global-seed path rather than paying for an index it doesn't need.
**Plans**: TBD

Plans:
- [ ] 13-01: TBD

### Phase 14: Non-Compiler Consumer Correctness

**Goal**: The mutation server, inserter, agent, and reflection UI all resolve types correctly and safely now that the registry is no longer eager — no regression, no silent wrong answers.
**Depends on**: Phase 13
**Requirements**: CONS-02, CONS-03, CONS-04, SAFE-02, SAFE-03
**Success Criteria** (what must be TRUE):

  1. The inserter reads and inserts a materialized config whose type isn't already resolved, resolving it correctly.
  2. The agent's filekv store serves a subscribed client a config whose type is resolved on demand, correctly.
  3. `GenReflectionUI`'s periodic walk resolves every mutable config's type and reports — rather than silently skips — one it cannot resolve.
  4. Concurrent requests against a long-lived mutation server or agent process complete without error under `go test -race`.
  5. A long-running process handling many different configs over time keeps its loaded-file count proportional to what was actually demanded — it never jumps to the full repository count after one unusual request.

**Measured impact**: Correctness-only — the "no regression, no silent wrong answers" pass. None of these four consumers were ever on the 200ms compile budget.
**Plans**: TBD

Plans:
- [ ] 14-01: TBD

### Phase 15: Verification, Decision & Gate Flip

**Goal**: The milestone's definition of done is measured and asserted in CI, and the one deliberate behavior change is written down rather than silently absorbed.
**Depends on**: Phase 14
**Requirements**: GATE-01, GATE-02, GATE-03, GATE-04, GATE-05
**Success Criteria** (what must be TRUE):

  1. `TestCompilerStartupScaling`'s allocation ratio at n=50 vs n=400 is at or below 2.0x, asserted with `require.LessOrEqual` — the `t.Skipf` branch is gone.
  2. Compiling the in-repo 800-proto synthetic corpus completes end-to-end in under 200ms, asserted in CI.
  3. A written decision exists explaining whether `protoconf compile` still reports a broken proto that no config loads, and why.
  4. The full pre-existing test suite passes, including the four validator cases and `field_type_any_test.pconf`, with unchanged outcomes.
  5. The real protoconf-terraform corpus (799 protos, 6.97s at baseline) is compiled once and its end-to-end time is recorded as milestone-close evidence, not a CI gate.

**Measured impact**: This phase is the measurement itself — it confirms Phases 11-12 delivered the target. No new mechanism ships here.
**Plans**: TBD

Plans:
- [ ] 15-01: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → ... → 10 → 11 → 12 → 13 → 14 → 15

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|-----------------|--------|-----------|
| 1-10. (see v1.0 table above) | v1.0 | 23/23 | Complete | 2026-03-31 |
| 11. Concurrency-Safe Lazy Registry Core | v2.0 | 0/TBD | Not started | - |
| 12. Growable Resolver Views & Race Safety | v2.0 | 0/TBD | Not started | - |
| 13. Exact Symbol Index & Shared Type-URL Resolution | v2.0 | 0/TBD | Not started | - |
| 14. Non-Compiler Consumer Correctness | v2.0 | 0/TBD | Not started | - |
| 15. Verification, Decision & Gate Flip | v2.0 | 0/TBD | Not started | - |
