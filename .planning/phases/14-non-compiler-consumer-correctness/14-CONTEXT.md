# Phase 14: Non-Compiler Consumer Correctness - Context

**Gathered:** 2026-09-08
**Status:** Ready for planning

<domain>
## Phase Boundary

The four consumers Phase 13's D-03 explicitly fenced off — `inserter`, the
mutation `server`, `agent/filekv`, and the `mutate` CLI — must resolve types
correctly and safely once their registry is no longer eager. Correctness-only:
**no regression, no silent wrong answers.** None of these four was ever on the
compiler's 200ms budget.

**In scope:** CONS-02, CONS-03, CONS-04, SAFE-02, SAFE-03 — flipping the four
consumers to the lazy construction path, rewiring every remaining type-URL and
nested-`Any` resolution onto `parser.TypeResolver`, keeping gRPC reflection
complete under laziness, making `GenReflectionUI` report rather than silently
skip, and proving concurrency-safety and non-accumulation.

**Structural fact that drives the phase, verified this session:** ROADMAP
criteria 1, 2 and 5 are *unobservable* while these consumers stay eager — an
eager registry has resolved everything by construction and its loaded-file
count is the full repository at startup. This phase therefore necessarily
flips consumers to lazy; that is not an optional optimization here, it is the
precondition for the criteria being assertable at all.

**Out of scope:** the scaling-gate flip and milestone numbers (Phase 15);
`mod sync`, which needs the whole tree by definition and stays eager;
`compiler/lib/config.go:68`'s silent-skip branch (see Deferred Ideas).

</domain>

<decisions>
## Implementation Decisions

### Blast Radius

- **D-01:** **All four consumers construct via `NewLazyModuleService`.**
  `inserter.go:192`, `server.go:295`, `filekv.go:70`, and `mutate.go:67` all
  flip. `mod/command.go:81` (`mod sync`) stays eager. Uniform — one story, no
  per-consumer carve-out for a future reader to learn, and every consumer stops
  paying the 4,639ms whole-tree parse at construction.

  The flip itself is a one-call change per site: `GetProtoRegistry`'s lazy
  branch (`compiler/lib/module_service.go:449-455`) already sets
  `registry.ImportPaths = []string{srcPath}` and `registry.CacheDir`, which is
  exactly what arms the Phase 13 scan and symbol-index tiers. The correctness
  work is in the resolver call sites (D-03), not in the construction change.

  **`NewLazyModuleService`'s doc comment (`module_service.go:48-53`) is now
  wrong and must be rewritten by this phase** — it currently states that
  "every other consumer (server, inserter, agent/filekv, mutate, mod sync)
  must keep using `NewModuleService`." After this phase only `mod sync` does.

  — **Reversibility:** costly — undo means reverting four construction sites
  plus every resolver rewiring in D-03 that only exists because of laziness.
  No published contract breaks, so it is not one-way.

- **D-02:** **Failure surfaces at first request, loudly — no startup
  validation pass.** Under eager construction, a config whose type is missing
  from `src/` failed at startup; under lazy it fails when someone asks for it.
  That shift is accepted deliberately, not discovered later. The Phase 13 D-02
  hard error already names the unresolved type URL, the import roots searched,
  and the symbol index's build/cache state — it is returned to the caller and
  logged as-is.

  Rejected: a startup smoke-check resolving every config the process is already
  committed to serving, and a full pre-resolve of the repository. The first
  adds a pass no criterion requires; the second contradicts criterion 5
  outright by loading the whole repository at boot.

  **This decision does not soften CONS-04.** `GenReflectionUI` reports its
  failures (D-05) because it is already walking `mutable_config/` on a ticker,
  not because a validation pass was added. The walk is bounded by
  `mutable_config/`, never by repository size, so criterion 5 survives it.

### One Resolution Path

- **D-03:** **`parser.TypeResolver` resolves every type URL and every nested
  `Any`, in all four consumers.** Concretely:
  - `inserter/inserter.go:369` — `LocalResolver.FindMessageByURL` →
    `TypeResolver.FindMessageByURL`.
  - `inserter/inserter.go:379` — the `protojson.MarshalOptions{Resolver: ...}`
    that renders `config.json` swaps too. This one is load-bearing beyond the
    inserter: the agent serves that exact JSON verbatim to REST clients
    (`agent/kv_agent_rollout_impl.go:getRawConfigJSON`), so a construction-time
    snapshot here would silently drop nested `Any` content that the 260902-eie
    backport exists to render.
  - `server/server.go:466` — `MutateConfig`'s marshal resolver. Under lazy the
    construction snapshot is near-empty, so leaving this on `LocalResolver`
    would hard-fail **every** custom-type mutation.
  - `mutate/mutate.go:74-76` — `anyResolver := parser.LocalResolver` →
    `TypeResolver`, for `FindMessageByName`.
  - `server/server.go:607-635` — already on `TypeResolver` from Phase 13; no
    change.

  This closes the clause Phase 13's D-03 explicitly left open: ROADMAP
  criterion 1's *"grepping for type-URL resolution logic finds that single
  implementation"* did not fully pass when Phase 13 shipped, because these
  three call sites remained. **It completes here.** A planner should treat
  "no `LocalResolver` remains as a type-URL or `Any` resolution source" as a
  checkable end-state, distinct from the `ExtensionResolver` uses below.

  Rejected: giving the mutation server the eager discovery registry for both
  reflection and `MutateConfig`. It preserves today's server behavior with no
  on-demand path in the request handler, but leaves two resolution
  implementations alive and permanently blocks criterion 1's grep clause.

- **D-04:** **The kept-alive discovery registry (D-06) is reflection's
  completeness backstop and nothing else.** It must not become a second
  general-purpose resolver. Deliberately narrow so D-03's single-path property
  is a real invariant rather than a convention.

### Observables

- **D-05:** **`GenReflectionUI` completes the whole walk, aggregates failures,
  and logs on change.** Three parts, all required:

  1. **Stop aborting the walk.** Today a `FindMessageByURL` or `UnmarshalTo`
     failure does `return err` (`server.go:618-628`), which terminates
     `filepath.WalkDir` — and the walk's return value is **discarded at all
     four call sites** (`server.go:196`, `server.go:200`,
     `devserver/command.go:87`, `devserver/command.go:96`). One unresolvable
     config therefore silently loses every remaining example. This is precisely
     what CONS-04 forbids. The two failure paths must also stop diverging: a
     `ReadConfig` error currently logs-and-skips while a resolver error aborts.
  2. **Aggregate.** Collect every unresolvable path with its type URL and the
     Phase 13 D-02 diagnostic into one error, returned from `GenReflectionUI`
     so callers can act on it.
  3. **Log on change, not per pass.** `GenReflectionUI` runs on a **5-second
     ticker** (`server.go:190-200`, `devserver/command.go:89-99`). Re-log only
     when the failure set changes: a persistently-broken config logs once, a
     newly-broken one logs immediately. Requires a small piece of per-server
     state holding the last failure set.

  Rejected: logging the aggregate every pass — ~720 lines/hour for one broken
  config, which trains operators to filter it out, i.e. loud enough to be
  ignored. Also rejected (as scope this phase does not need): rendering broken
  configs as visible entries in the grpcui example list — it touches the
  `standalone.Example` rendering path, which no criterion requires and no test
  covers. Recorded in Deferred Ideas.

- **D-06:** **gRPC reflection sees everything: keep `Init()`'s discovery
  registry alive.** `Init()` already performs a full eager `src/` parse into a
  throwaway `discoveryRegistry` (`server.go:351-360`, the Phase 11 CONS-01
  fix) and discards it. Reflection's `DescriptorResolver` is currently
  `s.parser.FilesResolver` — which Phase 12 made *growable*, so under lazy it
  holds only what has been demanded. A client could then list a custom service
  by name via reflection and fail to describe it: a user-visible grpcui
  regression against today's behavior, which the phase goal forbids.

  Fix: retain the discovery registry past `Init()` and hand its files resolver
  to `reflection.NewServerV1` and `reflection.NewServer`
  (`server.go:433-444`). Reflection stays byte-identical to today, and it
  costs **no extra parse** — that walk is already paid. This is the explicit
  resolution of the question Phase 11's D-02 deferred: *"widening reflection
  completeness under a lazy registry is a later phase's decision"*
  (`server.go:351-360`).

  **Consequence, stated so criterion 5 is not accidentally failed:** the
  mutation server now holds a full parse for its process lifetime. **Criterion
  5's counter is measured on the *serving* registry (`s.parser`'s), never on
  the discovery registry** — the same scoping Phase 11 D-02 already applied to
  discovery. A planner must not write a criterion-5 assertion that reads a
  process-wide total.

  Rejected: leaving reflection on the growable resolver (accepts the grpcui
  regression). Rejected: an on-demand `DescriptorResolver` wrapper that
  `ParseOne`s on a reflection miss — it satisfies both criteria most honestly,
  but is new machinery on a path with no test coverage today and puts
  filesystem work inside reflection requests, exposed to Phase 11's lock
  discipline. Recorded in Deferred Ideas as the principled upgrade.

### How the Criteria Get Proven

- **D-07:** **SAFE-02 gets both an end-to-end test and a dedicated tight-loop
  test, per consumer.**
  - End-to-end over `bufconn` with N concurrent clients driving `MutateConfig`
    and `SubscribeForConfig` under `-race` — real interceptors, real codec,
    matching the criterion's wording.
  - A dedicated **unpaced tight-loop** test hammering the resolver
    concurrently, **sanity-checked by temporarily removing the lock and
    confirming `WARNING: DATA RACE` before restoring it.**

  The second is not redundant. Phase 12 established this the hard way for
  SAFE-01: `TestConcurrentCompile`'s goroutines spend most of their time inside
  `protoparse.ParseFiles` with no lock held, so the read/write window is rare
  there, and only a dedicated test with continuously-running readers reliably
  forces the interleaving. An e2e-only proof can pass green with the race still
  present.

- **D-08:** **SAFE-03 gets both of criterion 5's clauses as separate
  assertions**, using the existing exported-counter observable
  (`LoadedFileCount()`, `utils/utils.go:512`) — the Phase 12 D-05/D-06 and
  Phase 13 D-04 precedent. No new observable, no log line, no CLI surface.
  - **(a) Escalation guard** — "never jumps to the full repository count after
    one unusual request." Serve one config whose type only the symbol-index
    tier can resolve; assert `LoadedFileCount()` grows by that config's
    transitive closure alone. The scan tier parses candidates one at a time and
    the index build parses *without linking*, so neither should register the
    tree — this test is the regression guard proving that stays true.
  - **(b) Sequence guard** — "over time keeps its loaded-file count
    proportional." Serve N distinct configs against one long-lived process;
    assert the count grows by a bounded delta per config and ends strictly
    below corpus size.

  Rejected: either clause alone. (a) alone misses slow accumulation across many
  requests; (b) alone catches a single all-loading request only by luck.

### Claude's Discretion

Decide these from the code and from measurement. Do not ask again.

- **How the discovery registry is retained** (D-06) — a field on
  `ProtoconfMutationServer`, a second `*parser.Parser`, or handing only its
  `*protoregistry.Files` forward. **Constraint:** whatever is retained must not
  become reachable as a general type-URL resolver (D-04), and it must not be
  read by any criterion-5 assertion (D-06).

- **The shape of the aggregated CONS-04 error and its change-detection**
  (D-05) — `errors.Join` over per-file errors vs. a typed aggregate carrying
  paths; failure-set comparison by sorted path list vs. a hash. **Constraint:**
  a config that changes from one failure reason to a *different* failure reason
  must re-log; comparing paths alone would suppress that.

- **Whether the four `GenReflectionUI` call sites start checking the returned
  error** (`server.go:196/200`, `devserver/command.go:87/96`). **Constraint:**
  the ticker must keep running regardless — a failed pass may never stop the
  server or the loop.

- **Whether `agent/filekv` needs any code change at all beyond D-01's
  construction flip.** `filekv.go:111` already calls `s.parser.ReadConfig`,
  which Phase 13 wired to `p.TypeResolver` (TYPE-09). It may be correct with
  zero edits. **Constraint:** verify with a test that exercises a type not
  resolved at construction (criterion 2), rather than assuming it.

- **Test placement and fixtures** for D-07/D-08 — which package, whether the
  synthetic corpus (`testdata.GenerateCorpus`) or a hand-built fixture backs
  the sequence guard. **Constraint:** the escalation guard needs a type the
  scan tier cannot resolve but the index tier can, so the fixture must be
  built against the Phase 13 tier boundary, not guessed.

- **Whether `mutate`'s and `inserter`'s per-config failure aborts the run or
  skips the item.** Current behavior is abort; D-02 makes the error loud
  either way. **Constraint:** do not silently change a CLI's exit-code
  behavior as a side effect of the resolver rewiring.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

ROADMAP.md carries no `Canonical refs:` line for this phase. The list below was
accumulated from PROJECT.md, REQUIREMENTS.md, Phase 13's context, and this
session's direct scout of the four consumers' call sites.

### Direct predecessor — its decisions are this phase's constraints
- `.planning/phases/13-exact-symbol-index-shared-type-url-resolution/13-CONTEXT.md`
  — **read first.** D-01 (scan-before-index tier order), D-02 (hard error, no
  eager fallback, and the diagnostic's required content), D-03 (the fence this
  phase removes, and its stated consequence that criterion 1's grep clause
  completes here), D-04 (counter-as-observable precedent), and the Deferred
  Ideas entry that scoped CONS-02/03/04 to this phase.
- `.planning/phases/13-exact-symbol-index-shared-type-url-resolution/13-VERIFICATION.md`
  and `13-REVIEW.md` — what Phase 13 proved, so this phase does not re-prove it.
- `.planning/phases/12-growable-resolver-views-race-safety/12-CONTEXT.md` —
  D-01 (hybrid growth: `FilesResolver` grows, `LocalResolver` stays a
  miss-fallthrough — this is *why* D-06 exists), D-03 (the `ImportPaths`-derived
  compiler-only gate), D-05/D-06 (counters as test-only observables).
- `.planning/phases/12-growable-resolver-views-race-safety/12-VERIFICATION.md`
  §SAFE-01 — the measured finding that an e2e concurrency test does not reliably
  force the race window. **This is the evidence behind D-07; do not weaken D-07
  without reading it.**
- `.planning/phases/11-concurrency-safe-lazy-registry-core/11-RESEARCH.md`
  §"Scope boundary" and §"Anti-Patterns to Avoid" — the six-consumer blast
  radius this phase finally completes.

### Requirements and roadmap
- `.planning/REQUIREMENTS.md` — CONS-02 (line 63), CONS-03 (64), CONS-04 (65),
  SAFE-02 (71), SAFE-03 (72). CONS-01 (62) and CONS-05 (66) are already closed;
  read them for what has *already* been decided about these consumers.
- `.planning/ROADMAP.md` §"Phase 14" — the goal and five success criteria this
  phase is judged against.
- `.planning/PROJECT.md` §"Current Milestone" — the authoritative milestone
  design record, including the "loud, never silent fallback" requirement that
  D-02 and D-05 both serve.

### Milestone research (read for evidence, note the supersession chain)
- `.planning/research/compiler-performance/PITFALLS.md` §1 — the mutation-server
  `Init()` analysis (CONS-01) whose fix D-06 now reuses; §2 for the nested-`Any`
  analysis. Its "use `proto_file`" conclusion is superseded by PROJECT.md.
- `.planning/research/compiler-performance/BASELINE.md` — the 4,639ms
  `GetProtoRegistry()` line each of the four consumers stops paying under D-01.
- `.planning/research/compiler-performance/TESTING.md` — corpus harness and
  test conventions for D-07/D-08.

### Prior related work
- `.planning/quick/260902-eie-fix-any-resolution-in-inserter-json-mars/` — the
  backport that made nested `Any` render in `config.json`. D-03's
  `inserter.go:379` swap exists to keep that working under laziness.
- `.planning/quick/260902-14f-fix-dummykv-pubsub-registration-race-roo/` —
  `dummykv` pubSub races already fixed; relevant to D-07's agent-side e2e test.
- `.planning/quick/260902-hp5-fix-filekv-data-race-and-double-close-be/` —
  `filekv` data race and `Close()`/`readEvents()` double-close already fixed;
  D-07's filekv concurrency test builds on this, it does not re-litigate it.
- `.planning/quick/260902-ggd-serve-getconfig-over-plain-http-via-conn/` — the
  vanguard REST path that consumes the inserter's `config.json` verbatim,
  which is why D-03 covers `inserter.go:379`.

### Codebase maps
- `.planning/codebase/ARCHITECTURE.md`, `.planning/codebase/CONCERNS.md`,
  `.planning/codebase/TESTING.md`.

### Benchmark corpus
- `../protoconf-terraform/example/src/` — 799 protos, 37.8 MB. A sibling
  working tree, not a checked-in fixture; it drifts.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **`parser.TypeResolver` / `RegistryTypeResolver`**
  (`compiler/lib/parser/parser.go:48-116`) — the Phase 13 shared chain: Tier 0
  construction snapshot → Tier 1 growable `MessageRegistry` → Tier 2 scoped
  lexical scan → Tier 3 symbol index → hard `NotFound`-wrapping error naming
  the symbol, import roots, and `IndexState()`. D-03 points all four consumers
  at this. Nothing new is built.
- **`Init()`'s `discoveryRegistry`** (`server.go:351-360`) — a full eager
  `src/` parse, already performed and currently discarded. D-06 retains it.
- **`LoadedFileCount()`** (`utils/utils.go:512`) — `RLock`, return a number,
  documented safe alongside `ParseOne`. The observable D-08 asserts on.
- **`testdata.GenerateCorpus(dir, n)`** — synthetic corpus generator behind the
  scaling gate and `TestConcurrentCompile`.
- **`bufconn` + `passthrough:///bufnet`** — the in-process gRPC test pattern
  established in Phase 01 and reused since; the base for D-07's e2e half.
- **`dummykv`** (`agent/dummykv/`) — test KV store, races already fixed
  (260902-14f); the agent-side e2e substrate.

### Established Patterns
- **Lock discipline (Phase 11):** never hold `d.mu` while calling
  `parser.ParseFiles` — `protoparse` calls `LookupImport` from inside, and that
  closure takes `RLock`. `utils/parse_all_deadlock_test.go` guards it. Every
  on-demand resolution this phase makes reachable from a request handler is
  exposed to this.
- **Counters as test-only observables** (Phase 12 D-05/D-06, Phase 13 D-04):
  exported method, no log line, no CLI surface.
- **Loud, never silent** (Phase 13 D-02, PROJECT.md): an exhausted resolution
  is a diagnostic hard error, not a skip.
- CI already runs `go test -race ./...` (`.github/workflows/go.yml`).

### Integration Points
- **Construction sites flipped by D-01:** `inserter/inserter.go:192`,
  `server/server.go:295`, `agent/filekv/filekv.go:70`, `mutate/mutate.go:67`.
  Left eager: `mod/command.go:81`.
- **`compiler/lib/module_service.go:48-53`** — `NewLazyModuleService`'s doc
  comment, factually wrong after D-01, must be rewritten.
- **`compiler/lib/module_service.go:449-455`** — the lazy branch of
  `GetProtoRegistry` that sets `ImportPaths` and `CacheDir`. Note the eager
  branch sets neither, which is what makes `LoadSymbolByScan`
  (`utils/symbol_scan.go:153-159`) and the index tier
  (`utils/symbol_index.go:315`) no-op on an eager registry — the flip is what
  turns those tiers on for these consumers.
- **Resolver call sites rewired by D-03:** `inserter/inserter.go:369` and
  `:379`, `server/server.go:466`, `mutate/mutate.go:74-76`.
- **`server/server.go:433-444`** — the two `reflection.NewServer*` calls whose
  `DescriptorResolver` D-06 repoints. Their `ExtensionResolver:
  s.parser.LocalResolver` is a *separate* question and is intentionally
  untouched: `RegistryTypeResolver.FindExtensionByName` delegates straight to
  the same snapshot (`parser.go:118-121`), so swapping it would change nothing.
  Do not "complete" D-03 by touching these.
- **`server/server.go:601-660`** — `GenReflectionUI`, all of D-05's surface.
  Its four call sites discard the returned error: `server.go:196`, `:200`,
  `devserver/command.go:87`, `:96`.
- **`agent/filekv/filekv.go:111`** — `s.parser.ReadConfig`, already on the
  shared path via TYPE-09. Likely CONS-03's whole surface.
- **`agent/kv_agent_rollout_impl.go:344-372`** — `getRawConfigJSON`, which
  serves the inserter's `config.json` verbatim over REST. Downstream consumer
  of `inserter.go:379`'s marshal resolver.
- **`devserver/command.go:82-99`** — constructs a lazy compiler *and* a lazy
  mutation server in one process, so after D-01 that process carries two
  independent lazy registries (two scans, two index builds, two caches) plus
  D-06's retained discovery parse. Nothing breaks; noted so it is not
  discovered as a surprise. See Deferred Ideas.

</code_context>

<specifics>
## Specific Ideas

- The user's framing throughout was scope-and-observable first: *which*
  consumers flip, *when* a failure surfaces, *what* a broken config looks like
  to an operator, and *how* a criterion is proven. Mechanism was deliberately
  left to discretion with measurement constraints attached.

- On D-06 the user chose the option that costs nothing extra precisely because
  `Init()` already pays that parse — reuse over new machinery, with the
  criterion-5 scoping written down rather than left implicit.

- On D-05 the 5-second ticker was the deciding fact: the user rejected
  per-pass logging on the grounds that ~720 lines/hour is *loud enough to be
  ignored*, which is a silent failure wearing a log line's clothes.

</specifics>

<deferred>
## Deferred Ideas

- **On-demand `DescriptorResolver` for reflection.** Wrapping reflection's
  resolver so a miss triggers `ParseOne` and retries would satisfy criterion 5
  and reflection completeness simultaneously and honestly, without retaining
  the discovery parse. Deferred in favor of D-06 because it is new machinery on
  an untested path that puts filesystem work inside reflection requests under
  Phase 11's lock discipline. This is the principled upgrade if the retained
  discovery registry's memory footprint ever matters.

- **Rendering unresolvable configs as visible broken entries in the grpcui
  example list** (D-05's rejected third option). The strongest read of "reports
  rather than silently skips" — reported where the operator is actually
  looking. Deferred: touches the `standalone.Example` rendering path, no
  criterion requires it, no test covers it.

- **`compiler/lib/config.go:68`'s silent `ErrUnexpectedType` skip.** Carried
  forward unchanged from Phase 13's Deferred Ideas: a fourth in-compiler
  resolution site whose silent-nil branch sits awkwardly beside D-02's
  loud-failure decision. It is in the *compiler*, not in this phase's four
  consumers, so it stays deferred here too — but it is now the last silent-skip
  branch left standing in the milestone. A planner may fold it in if it is
  cheap; it is not a criterion.

- **`devserver`'s three registries.** After D-01, `devserver` runs a lazy
  compiler registry, a lazy mutation-server registry, and D-06's retained
  discovery parse in one process — "proportional to demand" paid three times
  over. Reviewed and accepted this session: `devserver` is a development
  convenience, no criterion covers it, and sharing one registry across the
  compiler and the mutation server is a real design change, not a cleanup.
  Recorded so it is not discovered as a surprise.

</deferred>

---

*Phase: 14-non-compiler-consumer-correctness*
*Context gathered: 2026-09-08*
