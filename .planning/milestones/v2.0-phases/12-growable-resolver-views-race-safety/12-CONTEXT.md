# Phase 12: Growable Resolver Views & Race Safety - Context

**Gathered:** 2026-09-07
**Status:** Ready for planning

<domain>
## Phase Boundary

The compiler's `FilesResolver` (`*protoregistry.Files`) and `LocalResolver`
(`*protoregistry.Types`) are built once, at `NewParserWithDescriptorRegistry`
time, from whatever `DescriptorRegistry.FileRegistry` held at that instant.
After Phase 11 that is only the ~65 well-known types seeded by
`NewDescriptorRegistry`, so every proto that `ParseOne` loads on demand
mid-compile is invisible to those views.

This phase makes the compiler's file-resolution view grow as protos are
demanded — incrementally, without rebuilding over previously-loaded files, and
race-free under `go test -race`.

**In scope:** RSLV-01, RSLV-02, RSLV-03, SAFE-01 — growable `FilesResolver`,
incremental registration, the load-A/load-B/re-reference-A correctness case,
and the concurrency proof.

**Out of scope:** nested-type and symbol-index resolution (TYPE-01, Phase 13);
changing behavior of the mutation server, inserter, agent, or reflection UI
(Phase 14); the scaling-gate flip and milestone numbers (Phase 15).

</domain>

<decisions>
## Implementation Decisions

### Growth Mechanism

- **D-01:** **Hybrid.** `FilesResolver` grows by real incremental registration
  into the underlying `*protoregistry.Files`. `LocalResolver` does **not** —
  it stays the Phase 11 miss-fallthrough wrapper (`RegistryTypeResolver`,
  `compiler/lib/parser/parser.go:57`), which already reads the immutable
  construction-time snapshot, then the growable `MessageRegistry`, then fires
  the D-03 `ParseAll` fallback. Rationale: enumeration (`RangeFiles`) is what
  file resolution and downstream reflection/grpcui actually need, and only
  `Files` provides it; types are already served correctly by a mechanism that
  mutates nothing. Two mechanisms, each the cheapest one that serves its
  consumer. — **Reversibility:** costly — collapsing to one mechanism later
  means re-touching every `Parser` field consumer in `server/`, `inserter/`,
  `mutate/`, and `agent/filekv/`.

- **D-02:** **Uniform insert point.** Files reaching the registry via the D-03
  `ParseAll` eager fallback (`utils/utils.go:352`) register into
  `FilesResolver` through the same path as `ParseOne`'s files — no special
  case. `FileRegistry` and `FilesResolver` must never hold divergent sets.
  Accepted cost: when the fallback fires it now also pays a whole-tree
  registration. That cliff is already known, is what Phase 13's symbol index
  exists to remove, and a set divergence would be a worse invariant to carry
  forward than a slow path that is already slow.

### Blast Radius

- **D-03:** **Compiler only.** Growth is gated on `registry.ImportPaths` being
  non-empty — the *existing* D-01 boundary from Phase 11, not a new one. This
  was verified this session: `ImportPaths` is written in exactly one place
  (`compiler/lib/module_service.go:454`), under `m.lazyRegistry`, which only
  `NewLazyModuleService` sets, which only `NewCompiler` calls. The other five
  consumers (`server/server.go:291`, `inserter/inserter.go:220`,
  `mutate/mutate.go:73`, `agent/filekv/filekv.go:80`, and `mod sync`'s own
  eager registry) therefore keep byte-identical behavior, and Phase 14
  *verifies* rather than *undoes*.

- **D-04:** No new constructor and no new flag for the opt-in. Deriving it
  from `ImportPaths` means "compiler only" falls out of an invariant that
  already exists and is already tested, rather than a second one to keep in
  sync. Rejected: `NewGrowableParserWithDescriptorRegistry` / a `With...`
  option — explicit at the call site, but a duplicate encoding of the same
  condition.

### RSLV-02 Observable

- **D-05:** **Registration counter on `DescriptorRegistry`**, exported as a
  method alongside the existing `LoadedFileCount()` (`utils/utils.go:378`). A
  test loads N protos one at a time and asserts total registrations grow by
  each file's own closure, not by N×(files-so-far). This measures "no rebuild"
  as a number rather than inferring it. Rejected: resolver-object-identity
  assertion (proves no rebuild, says nothing about per-registration work) and
  a timing assertion folded into `TestCompilerStartupScaling` (flakiest gate
  shape; cannot distinguish a resolver rebuild from any other regression).

- **D-06:** The counter is **test-only** — an exported method, no log line and
  no CLI surface. Operators already get the loaded-file count from LAZY-05; a
  second number in normal compile output is noise for anyone not debugging
  resolver churn.

### Claude's Discretion

The user explicitly deferred these. Decide them on evidence from the code, not
by asking again.

- **Locking / API shape for the growable `FilesResolver`.** `protoregistry.Files`
  is documented as unsafe for concurrent mutation, so growth means readers need
  synchronising too. Candidates: a parser-owned type holding the `*protoregistry.Files`
  plus an `RWMutex` and exposing `FindFileByPath`/`RangeFiles`/`RegisterFile`;
  or a lock held inside `DescriptorRegistry` under the `d.mu` it already owns.
  **Constraint:** whatever is chosen must be provable under `-race` *without*
  dragging the non-compiler consumers into this phase (D-03).
- **Whether `Parser.FilesResolver`'s exported field type changes.** Keeping it
  as `*protoregistry.Files` means `server/server.go:292-297` (six hand-registered
  well-known files) and `:426`/`:433` (grpcui `DescriptorResolver` wiring) compile
  untouched, but readers holding the field directly are outside any lock — so
  that choice must come with evidence that no such reader exists on a lazy
  registry. Making it a small interface forces readers through the lock but
  ripples a type change through four packages.
- **Registration granularity** — the requested file only, or its transitive
  closure. `recordFileLocked` (`utils/utils.go:333`) already walks
  `fd.GetDependencies()` recursively and skips names already present, so the
  closure is available for free at the same insert point. Constraint: whichever
  is chosen must keep `protodesc.NewFile`'s dependency requirement satisfiable
  without introducing a second resolution path.
- **`ParseFilesX` shape** (`compiler/lib/parser/parser.go:118-155`). Trace what
  the `FilesResolver.FindFileByPath` branch actually resolves that the registry
  does not, then cut or keep on that evidence. See Integration Points below for
  why the branch is a live hazard either way.
- **RSLV-03 test assertions.** Constraint: the test must fail loudly if the
  canonical-pointer contract regresses — a re-parse that mints a second
  descriptor for A can produce correct materialized JSON, so output-only
  assertions do not guard the criterion.
- **SAFE-01 test placement** — extend `TestConcurrentCompile`
  (`compiler/lib/concurrent_compile_test.go:25`) with resolver-view assertions,
  or add a dedicated test racing `RegisterFile` against `FindFileByPath`.
  Constraint: a resolver-growth race must fail loudly under `go test -race`.
- **The dead `config.protoResolver` field** (see Integration Points) — delete
  it, or point it at the growable view. Constraint: do not leave it holding a
  frozen snapshot in the struct this phase is making grow.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

ROADMAP.md carries no `Canonical refs:` line for this phase; the list below was
accumulated from REQUIREMENTS.md, the committed milestone research bundle, and
the codebase scout.

### Milestone design record (authoritative, committed 2026-09-04)
- `.planning/research/compiler-performance/OPTIONS.md` — records the chosen
  shape (Option A, lazy by path) and, at lines 38-44, the specific statement
  that the 260ms `NewParserWithDescriptorRegistry` cost is derived from
  `FileRegistry` and **"cannot stay eager: a lazy registry with an eager
  resolver snapshot would be a stale-cache bug the first time a new file is
  parsed"** — this phase's raison d'être. Also records what was rejected and
  must not be re-proposed (linker-lookup optimisation, parallelised eager
  parse, eager symbol index).
- `.planning/research/compiler-performance/BASELINE.md` — the measured
  breakdown; line 24 is the 260ms resolver-build line item this phase deletes.
- `.planning/research/compiler-performance/PITFALLS.md` — the not-by-path
  lookup problem, which is the boundary between this phase and Phase 13.
- `.planning/research/compiler-performance/TESTING.md` — the existing
  scaling-gate and corpus harness conventions.
- `.planning/research/compiler-performance/SUMMARY.md` — milestone framing.

### Phase 11 (direct predecessor — its invariants are this phase's constraints)
- `.planning/phases/11-concurrency-safe-lazy-registry-core/11-RESEARCH.md` —
  §"Scope boundary" and §"Anti-Patterns to Avoid" define the six-consumer blast
  radius and the three things not to do. Its Metadata block explicitly says it
  should be re-checked after Phase 12 because **"Phase 12's growable resolvers
  will change some of the exact call shapes described here (particularly
  `NewParserWithDescriptorRegistry`'s snapshot behavior)"** — that is this phase.
- `.planning/phases/11-concurrency-safe-lazy-registry-core/11-VERIFICATION.md`
  and `11-UAT.md` — what Phase 11 proved, so this phase does not re-prove it.
- `.planning/phases/11-concurrency-safe-lazy-registry-core/11-PATTERNS.md` —
  file-to-analog mapping still valid for this phase's touched packages.

### Requirements and roadmap
- `.planning/REQUIREMENTS.md` — RSLV-01 (line 21), RSLV-02 (line 22),
  RSLV-03 (line 23), SAFE-01 (line 70).
- `.planning/ROADMAP.md` §"Phase 12" — goal, dependencies, and the three
  success criteria this phase is judged against.

### Codebase maps
- `.planning/codebase/ARCHITECTURE.md`, `.planning/codebase/CONCERNS.md`,
  `.planning/codebase/TESTING.md` — current structure, known concerns, and test
  conventions.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **`RegistryTypeResolver`** (`compiler/lib/parser/parser.go:48-116`) — the
  miss-fallthrough pattern, already shipped and tested. D-01 keeps it as-is for
  types; it is also the reference shape if any part of file resolution ends up
  needing a fallthrough rather than growth.
- **`recordFileLocked`** (`utils/utils.go:333-343`) — the single insert point
  for descriptors, called under `d.mu` write lock, already recursive over
  `fd.GetDependencies()` and already idempotent on already-present names. This
  is where growth hooks in, and its already-present skip is most of what
  RSLV-02 needs.
- **`LoadedFileCount()`** (`utils/utils.go:378`) — the shape D-05's registration
  counter should mirror: `RLock`, return a map length, documented as safe
  alongside `ParseOne`.
- **`testdata.GenerateCorpus(dir, n)`** — synthetic corpus generator, already
  used by `TestConcurrentCompile` and the scaling gate.
- **`TestConcurrentCompile`** (`compiler/lib/concurrent_compile_test.go:25`) —
  already exercises 8 goroutines against one shared `*lib.Compiler`, each
  demanding one private proto plus one shared proto, under `-race`. SAFE-01's
  shape exists; this phase's job is to make it also cover resolver growth.
- **`golang.org/x/sync/singleflight`** — already a direct dependency, already
  used by `ParseOne` for duplicate suppression.

### Established Patterns
- **`DescriptorRegistry` is the single source of truth**, `d.mu` is its lock,
  and `recordFileLocked` is the one writer. Anything growable should hang off
  that, not off a parallel structure.
- **Lock discipline from Phase 11:** never hold `d.mu` while calling
  `parser.ParseFiles` — `protoparse` calls `LookupImport` from inside, and that
  closure takes `RLock` (`utils/utils.go:265-269`). Any new lock must not
  reintroduce that inversion; `utils/parse_all_deadlock_test.go` guards it.
- **Canonical-pointer contract:** `ParseOne` deliberately returns the registry's
  entry, not the descriptor it just parsed (`utils/utils.go:302-320`), so every
  caller and every map lookup see the same pointer. Growth must not break this.
- **Copying `msgregistry.MessageRegistry` by value is forbidden** post-Phase-11
  — `compiler/lib/compiler.go:356` passes it by pointer for exactly this reason.
- CI already runs `go test -race ./...` (`.github/workflows/go.yml`), so
  SAFE-01's gate is enforcement, not aspiration.

### Integration Points
- **`compiler/lib/parser/parser.go:36-46`** — `NewParserWithDescriptorRegistry`,
  where `GetFilesResolver()` builds the snapshot. This is the construction site
  the growth gate (D-03) attaches to.
- **`utils/utils.go:126-133`** — `GetFilesResolver()` builds a whole
  `FileDescriptorSet` and runs `protodesc.FileOptions{AllowUnresolvable: true}.NewFiles`
  over it. This is the 260ms rebuild in the eager world; the growable path must
  not call it per demanded file.
- **`compiler/lib/parser/parser.go:118-155`** — `ParseFilesX`, the function the
  Starlark loader reaches through `loadProto` (`compiler/lib/starlark_loader.go:206`).
  **Live hazard found this session:** on a `FilesResolver.FindFileByPath` hit it
  calls `desc.WrapFile` (falling back to `desc.CreateFileDescriptor`), minting a
  `*desc.FileDescriptor` unrelated to the registry's canonical entry. That is
  precisely the RSLV-03 "re-reference A" path, and it directly contradicts the
  canonical-pointer contract Phase 11 established in `ParseOne`. Once
  `FilesResolver` grows, this branch starts *hitting* where today it usually
  misses — so growth makes the hazard reachable rather than latent.
- **`compiler/lib/config.go:27` + `compiler/lib/compiler.go:356`** —
  **dead field found this session:** `config.protoResolver` is assigned
  `c.parser.LocalResolver` and never read anywhere in the package. It is a
  frozen-snapshot reference sitting in the struct this phase is making grow;
  wiring it up later without noticing would silently reintroduce stale
  resolution.
- **`server/server.go:292-297`** — six `parser.FilesResolver.RegisterFile(...)`
  calls directly on the exported field, plus `:426`/`:433` passing it as grpcui's
  `DescriptorResolver` and `LocalResolver` as `ExtensionResolver`. This is the
  call site that decides whether the field type can change (Claude's Discretion).
- **`compiler/lib/module_service.go:403-405`** — `GetProtoFilesRegistry()` has
  **zero callers in the tree** (verified this session). Likely deletable; at
  minimum it must not become a second, non-growing way to obtain a resolver.

</code_context>

<specifics>
## Specific Ideas

- The user consistently deferred *how* (locking shape, granularity, test
  placement) and locked *what* (hybrid growth, `ImportPaths` gate, uniform
  insert point, counter-based observable). Downstream agents should decide the
  deferred items from code evidence and record the reasoning, not re-ask.
- Two findings surfaced during the scout that the user chose to leave to
  Claude's judgment rather than rule on: the `ParseFilesX` non-canonical-pointer
  branch and the dead `config.protoResolver` field. Both are in the code this
  phase touches; neither should be silently left as-is.

</specifics>

<deferred>
## Deferred Ideas

- **Operator-visible resolver-churn metric** — D-06 keeps the registration
  counter test-only. If Phase 15's numbers turn out to want it in compile
  output, that is a one-line addition then, with real motivation behind it.
- **Removing the D-03 `ParseAll` whole-tree fallback cliff** — accepted as a
  known slow path here (D-02). Phase 13's exact symbol index is what removes it.
- **`ModuleService.GetProtoFilesRegistry()` deletion** — zero callers today. If
  it turns out not to be in this phase's path, it is a trivial cleanup for
  Phase 14 or a `/gsd-quick`.
- **Collapsing the two resolver mechanisms into one** — D-01's hybrid is
  deliberate for this phase. If Phase 13's index makes the `Types`
  miss-fallthrough redundant, unify then.

</deferred>

---

*Phase: 12-Growable Resolver Views & Race Safety*
*Context gathered: 2026-09-07*
