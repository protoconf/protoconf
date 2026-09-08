# Phase 13: Exact Symbol Index & Shared Type-URL Resolution - Context

**Gathered:** 2026-09-08
**Status:** Ready for planning

<domain>
## Phase Boundary

Type-URL resolution must answer "which file declares symbol X" for any message
symbol under `src/` — **including nested types**, which are 96.9% of the symbols
in the benchmark corpus — without linking the repository, and every nested
`google.protobuf.Any` in a config must resolve through one shared code path
rather than a per-consumer implementation.

Today `RegistryTypeResolver` (`compiler/lib/parser/parser.go:48-116`) resolves
top-level messages only, and on a second miss fires `ParseAll()` — the 4,639ms
whole-tree eager parse this milestone exists to delete.

**In scope:** TYPE-01..09 and CONS-05 — the symbol index (parse without
linking, persisted content-keyed under `.protoconf_cache`, built on first
need), the lexical pre-tier that fronts it, the single shared resolution path,
`parser.ReadConfig`'s nested-`@type` correctness, and the mutable-config load
path.

**Out of scope:** rewiring `inserter`, `server`, `mutate`, and `agent/filekv`
onto the shared path (CONS-02/03/04, Phase 14 — see D-03); the scaling-gate
flip and milestone numbers (Phase 15); replacing the index with a scan outright
(see Deferred Ideas — a roadmap amendment, not a phase decision).

</domain>

<decisions>
## Implementation Decisions

### Resolution Chain

- **D-01:** **Scan before index.** The resolution order is: construction-time
  `protoregistry.Types` snapshot → the registry's growable `MessageRegistry` →
  **scoped lexical scan** over `src/`, whose hits are confirmed by a no-link
  parse → **symbol index**, built and cached on demand → hard error. The scan
  sits *ahead of* index construction, not behind it.

  Rationale, measured this session on the 799-proto corpus at
  `../protoconf-terraform/example/src` (37.8 MB): a whole-corpus lexical scan
  costs **~11ms warm**, against **1,286ms** for `ParseFilesButDoNotLink` over
  the same tree (OPTIONS.md). OPTIONS.md's 623ms "naive lexical scan" figure
  measured a Go read+regexp-per-file implementation and undersells what the
  scan can be by ~50x. With the scan in front, most lookups resolve in
  milliseconds and the index build becomes genuinely rare rather than merely
  cached.

  Rejected: the scan as a tier *after* the index. If the index is exact and
  complete over `src/`, an index miss means the symbol is not in `src/` at all
  — and a scan of the same tree misses too. That placement can almost never
  fire usefully.

- **D-02:** **The index replaces `ParseAll` as the last tier.** A type URL that
  survives the whole chain is a **hard error**, not a reason to parse 799
  files. The error is engineered as the diagnostic: it names the unresolved
  type URL, the directories searched, and whether the index was cold or a cache
  hit. This folds PROJECT.md's active "loud, never silent fallback"
  requirement into this phase rather than deferring it.
  — **Reversibility:** one-way — this deletes the eager fallback that today
  makes *every* resolution failure recoverable-but-slow. Restoring it later
  means re-introducing `ParseAll`'s whole-tree parse and its `eagerFallback`
  latch (`utils/utils.go:433-460`), and any repo that compiled only because of
  that fallback will have been failing in the interim.
  **Accepted risk, stated explicitly:** an index or scan blind spot turns a
  slow compile into a failed one. This was chosen over the safer
  "index-then-`ParseAll`-with-a-warning" precisely because leaving the 4.6s
  path reachable is what PROJECT.md says this phase exists to prevent.

- **D-05:** **The scan pattern must be nesting-tolerant, and a lexical hit is
  never an answer.** Measured on the corpus: **4,054** message declarations are
  top-level, **126,530** are nested and indented. An `^message`-anchored
  pattern finds **3.1%** of symbols — structurally missing the nested-`Any`
  case TYPE-03 exists for. The scan therefore matches indented declarations and
  yields **candidate files**, and the symbol is confirmed by a no-link parse of
  those candidates, never by the lexical match itself (PITFALLS.md's
  non-negotiable). Candidate counts vary by three orders of magnitude on this
  corpus — `AwsS3Bucket` → 4 files, `Tags` → 15, `Timeouts` → **351** — so the
  type URL's own structure (package prefix → directory, first non-package
  segment → top-level message) is what narrows the set before verification.

### Blast Radius

- **D-03:** **Compiler only.** This phase builds the shared, index-backed path
  and wires the compiler onto it (TYPE-09, CONS-05).
  `inserter/inserter.go:369`, `server/server.go:607`, and `mutate/mutate.go:76`
  keep calling `LocalResolver` / `anyResolver` directly until Phase 14
  (CONS-02/03/04). This honors Phase 12's D-03 fence exactly.

  **Consequence to carry forward, stated rather than discovered later:**
  ROADMAP success criterion 1's "grepping for type-URL resolution logic finds
  that single implementation" does **not** fully pass when Phase 13 closes —
  three call sites remain. That criterion effectively completes in Phase 14.
  Verified this session so the two decisions are known not to conflict:
  `ParseAll` already short-circuits on `len(d.ImportPaths) == 0`
  (`utils/utils.go:436`), which is every eager registry, so "the index replaces
  `ParseAll`" (D-02) is *automatically* compiler-scoped. The four non-compiler
  consumers never reached that path; their `RegistryTypeResolver` fallthrough
  already returns `protoregistry.NotFound` after two misses and stays that way.

### Observables

- **D-04:** **Exported index-build counter** on `DescriptorRegistry`, mirroring
  Phase 12's D-05/D-06 precedent: an exported method alongside
  `LoadedFileCount()` (`utils/utils.go:512`) and the registration counter,
  **test-only** — no log line, no CLI surface. The criterion-3 test compiles a
  config whose only mutable value is a `google.protobuf.Value` and asserts the
  build count is **0**. Rejected: a timing assertion folded into
  `TestCompilerStartupScaling`, which Phase 12 already rejected for RSLV-02 as
  the flakiest gate shape.

### Claude's Discretion

The user explicitly deferred these. Decide them on evidence from the code and
from measurement, not by asking again.

- **Index symbol-kind coverage** — messages only (literal TYPE-01), messages +
  enums (mirroring what `GetTypesResolver` registers at `utils/utils.go:196-217`,
  so the index is a strict superset of the resolver it backs), or all named
  symbols including services and extensions. **Constraint:** decide on what the
  six consumers actually look up and on what a no-link parse yields for free;
  do not narrow a capability that resolves today.

- **Cache key and invalidation mechanism** (TYPE-05/06). Candidates:
  `dirhash.HashDir` over `src/` (already a direct dependency, already keys the
  module repo caches at `compiler/lib/module_service.go:273`, literally
  content-keyed); a `(path, size, mtime)` stat manifest as a warm fast path
  with a content hash on mismatch; or the `.fds` checksum idiom
  (`utils/utils.go:617-620`) re-keyed over source bytes. **Constraint:** record
  the measured hash cost over the 799-proto corpus alongside the choice, and
  any staleness check must be one-way — a mismatch may only ever cause *more*
  work, never a stale serve. Note that because the index is built on first need
  (TYPE-07), the validity check is *also* only paid on first need, so a
  `google.protobuf.Value`-only compile hashes nothing.

- **The scan → index escalation rule.** Escalate when the scan finds zero
  candidates and when candidates fail to verify; whether a *candidate-count
  threshold* also escalates is open. **Constraint:** derive the break-even by
  timing a single no-link parse against the index build, and record the number
  with the rule — parsing 351 candidates (the measured `Timeouts` worst case)
  one at a time may cost more than the 1,286ms index. If the rule never fires,
  TYPE-01/02/05/06 ship as dead code; that outcome must be visible, not silent.

- **On-disk index format and in-memory shape.** 130,584 message symbols on the
  benchmark corpus, ~138K including enums and services. **Constraint:** the
  format must be validated before it is trusted, in the spirit of
  `registry.Load`'s checksum gate.

- **Whether the index build is singleflighted** the way `ParseOne` is
  (`golang.org/x/sync/singleflight`, already a direct dependency).
  **Constraint:** two concurrent compiles must not both pay the build.

- **The scan's and the index build's concurrency contract.** **Constraint:**
  Phase 11's lock discipline holds — never hold `d.mu` while calling
  `parser.ParseFiles`; `utils/parse_all_deadlock_test.go` guards it. Both new
  tiers do filesystem work and then parse, so both are exposed to that
  inversion.

- **Whether `parser.ReadConfig` needs a pre-pass at all.** `protojson`
  consults its `Resolver` for every `Any` it meets, at any nesting depth, so an
  index-backed `TypeResolver` may make `ReadConfig` (`parser.go:187`) correct
  with **zero changes** — which would retire PITFALLS.md's proposed
  `proto_file` pre-pass entirely. **Constraint:** verify this against
  `protojson`'s actual behavior with a nested-`Any` fixture before relying on
  it; do not assume it.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

ROADMAP.md carries no `Canonical refs:` line for this phase; the list below was
accumulated from PROJECT.md, REQUIREMENTS.md, the committed milestone research
bundle, Phase 12's context, and this session's codebase scout.

### Milestone design record — note the supersession chain
- `.planning/PROJECT.md` §"Current Milestone" — **authoritative and most
  recent (2026-09-08).** Its "Key design decision" paragraph states that type
  URLs resolve through an exact symbol index built by parsing without linking,
  persisted under `.protoconf_cache`, and that **`proto_file` is superseded as
  a resolution mechanism**. This supersedes PITFALLS.md §2 below.
- `.planning/research/compiler-performance/PITFALLS.md` §2 — **partially
  superseded, read for its evidence, not its conclusion.** Its "no symbol index
  is needed, use `proto_file`" correction was itself overturned by PROJECT.md.
  Still authoritative for: the nested-`Any` analysis ("the case `proto_file`
  structurally cannot answer"), the package→directory measurement (13/13 exact
  on this corpus, files-per-package median 1 / max 256), and the **two
  non-negotiables** — verify the symbol after parsing rather than trusting a
  heuristic, and keep any path hint a hint rather than a source of truth.
- `.planning/research/compiler-performance/OPTIONS.md` §"Rejected: eager symbol
  index" — records that every *eager* index variant is rejected
  (`ParseFilesButDoNotLink` 1,286ms; naive lexical scan 623ms) and that an
  index, if required, **must be persisted and incrementally maintained**. That
  is the shape TYPE-05/06/07 encode. Its 623ms scan figure is superseded by
  this session's measurement (see D-01).
- `.planning/research/compiler-performance/BASELINE.md` — the measured
  breakdown; the 4,639ms `GetProtoRegistry()` line is the path D-02 deletes.
- `.planning/research/compiler-performance/TESTING.md` — scaling-gate and
  corpus harness conventions.
- `.planning/research/compiler-performance/SUMMARY.md` — milestone framing.

### Phase 12 (direct predecessor — its decisions are this phase's constraints)
- `.planning/phases/12-growable-resolver-views-race-safety/12-CONTEXT.md` —
  D-01 (hybrid growth: `FilesResolver` grows, `LocalResolver` stays a
  miss-fallthrough), D-02 (uniform insert point), D-03 (compiler-only gate
  derived from `ImportPaths`), D-05/D-06 (the counter-as-observable precedent
  D-04 above mirrors), and the canonical-pointer and lock-discipline patterns.
- `.planning/phases/12-growable-resolver-views-race-safety/12-VERIFICATION.md`
  and `12-REVIEW.md` — what Phase 12 proved, so this phase does not re-prove it.
- `.planning/phases/12-growable-resolver-views-race-safety/12-PATTERNS.md` —
  file-to-analog mapping, still valid for the packages this phase touches.
- `.planning/phases/11-concurrency-safe-lazy-registry-core/11-RESEARCH.md` —
  §"Scope boundary" and §"Anti-Patterns to Avoid": the six-consumer blast
  radius and the three things not to do.

### Requirements and roadmap
- `.planning/REQUIREMENTS.md` — TYPE-01..TYPE-09 (lines 50-58) and CONS-05
  (line 66). Line 45-46 carries the 2026-09-04 roadmap revision note that the
  index and the shared path consulting it ship together in one phase.
- `.planning/ROADMAP.md` §"Phase 13" — the goal and the four success criteria
  this phase is judged against. **Read criterion 1 alongside D-03 above:** its
  grep clause completes in Phase 14.

### Benchmark corpus
- `../protoconf-terraform/example/src/` — the 799-proto, 37.8 MB corpus every
  measurement in this document was taken on. Not a fixture; per PITFALLS.md §6
  it is a working tree and will drift.

### Codebase maps
- `.planning/codebase/ARCHITECTURE.md`, `.planning/codebase/CONCERNS.md`,
  `.planning/codebase/TESTING.md` — structure, known concerns, test conventions.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **`RegistryTypeResolver`** (`compiler/lib/parser/parser.go:48-116`) — the
  existing tiered miss-fallthrough, already shipped and tested. Its
  `FindMessageByURL` (`:66`) and `FindMessageByName` (`:87`) are where D-01's
  new tiers insert, and its `_ = r.registry.ParseAll()` calls (`:79`, `:98`)
  are exactly what D-02 replaces.
- **`registry.Load` / checksum gate** (`utils/utils.go:606-622`) — the existing
  `.protoconf_cache` idiom: read a serialized artifact, recompute its checksum,
  refuse it on mismatch. The index cache should follow this shape.
- **`dirhash.HashDir`** (`golang.org/x/mod/sumdb/dirhash`, used at
  `compiler/lib/module_service.go:273`) — content-hashing over a directory tree,
  already a direct dependency.
- **`golang.org/x/sync/singleflight`** — already a direct dependency, already
  used by `ParseOne` to collapse duplicate concurrent parses.
- **`LoadedFileCount()`** (`utils/utils.go:512`) — the shape D-04's build
  counter should mirror: `RLock`, return a number, documented as safe alongside
  `ParseOne`.
- **`testdata.GenerateCorpus(dir, n)`** — synthetic corpus generator behind the
  scaling gate and `TestConcurrentCompile`.

### Established Patterns
- **`DescriptorRegistry` is the single source of truth**, `d.mu` is its lock,
  `recordFileLocked` (`utils/utils.go:~400`) is the one writer. New structures
  hang off that, not off a parallel one.
- **Lock discipline (Phase 11):** never hold `d.mu` while calling
  `parser.ParseFiles` — `protoparse` calls `LookupImport` from inside, and that
  closure takes `RLock`. `utils/parse_all_deadlock_test.go` guards it.
- **Canonical-pointer contract:** one `*desc.FileDescriptor` per logical file;
  `ParseOne` returns the registry's entry, not the descriptor it just parsed.
- **Counters as test-only observables** (Phase 12 D-05/D-06): exported method,
  no log line, no CLI surface.
- CI already runs `go test -race ./...` (`.github/workflows/go.yml`).

### Integration Points
- **`compiler/lib/parser/parser.go:66-105`** — `RegistryTypeResolver`'s two
  lookup methods. Their doc comment already names the gap this phase closes:
  *"Only top-level messages resolve through the MessageRegistry branch […]
  nested-type resolution is TYPE-01, Phase 13."*
- **`utils/utils.go:196-217`** — `GetTypesResolver` iterates `fd.Messages()` and
  `fd.Enums()`, which are **top-level only**. This is the concrete source of the
  nested-type gap, and the reference point for the coverage question under
  Claude's Discretion.
- **`utils/utils.go:433-460`** — `ParseAll`, the 4,639ms fallback D-02 deletes.
  Verified short-circuiting on `len(d.ImportPaths) == 0` (`:436`), which is what
  makes D-02 automatically compiler-scoped.
- **`compiler/lib/module_service.go:449-455`** — on the lazy path
  `registry.ImportPaths = []string{srcPath}`, and **only** `src/`. Remote-module
  protos never travel through `ImportPaths`; they arrive at construction via
  `registry.Load(<label>.fds, sum)` → `MergeFileDescriptorSet`, so their symbols
  are already in `MessageRegistry`. **The index's `src/`-only scope is therefore
  derived from an existing invariant, not newly declared** — the same shape as
  Phase 12's D-03 gate.
- **`compiler/lib/parser/parser.go:187`** — `ReadConfig`, one
  `protojson.Unmarshal` with `Resolver: p.TypeResolver`. TYPE-09's whole
  surface. See the discretion item on whether it needs any change at all.
- **`compiler/lib/starlark_loader.go:167-190`** — `loadMutable`, the CONS-05
  path. It performs **three** lookups: `ReadConfig` (`:173`),
  `TypeResolver.FindMessageByURL` (`:177`), and
  `moduleService.GetProtoRegistry().MessageRegistry.FindMessageTypeByUrl`
  (`:187`). The third bypasses the shared path entirely and reaches into the
  registry directly — it is the in-scope divergence CONS-05 has to close.
- **`compiler/lib/config.go:66`** — `validate` resolves an `Any`'s type through
  `c.messageRegistry.FindMessageTypeByUrl` and **silently returns nil** on
  `ErrUnexpectedType` (`:68`). A fourth resolution site inside the compiler, and
  its silent-skip branch is in tension with D-02's loud-failure decision.
- **Phase 14's three call sites, listed so the boundary is explicit and not
  re-derived:** `inserter/inserter.go:369`, `server/server.go:607`,
  `mutate/mutate.go:76`. Untouched by this phase (D-03).

</code_context>

<specifics>
## Specific Ideas

- The lexical pre-tier originated as the user's own proposal during this
  session: *"`grep -e "^package packagename.v1" -e "^message MessageName"
  src/**/*.proto` should give back a pretty fast resolution path."* The speed
  instinct measured out correct (~11ms warm, whole corpus). Two corrections
  were folded in and are now D-01/D-05: the `^message` anchor finds only 3.1%
  of symbols on the benchmark corpus, and the tier belongs *before* index
  construction rather than after it.

- All measurements in this document were taken on
  `../protoconf-terraform/example/src` (799 protos, 37.8 MB) during the
  2026-09-08 discussion, with a warm page cache. The cold-cache number is
  I/O-bound on 37.8 MB and was **not** measured — a planner or executor
  relying on the scan's latency should measure it cold.

</specifics>

<deferred>
## Deferred Ideas

- **Replace the symbol index entirely with scan + verify-after-parse.** The
  ~11ms whole-corpus scan measurement makes this genuinely plausible: it would
  remove the cache format, the invalidation logic, and TYPE-05/06's machinery
  outright. It was **not** taken because it revises PROJECT.md's stated key
  design decision and guts TYPE-01/02/05/06/07 — a roadmap amendment via
  `/gsd-phase`, not something to decide inside a phase plan. If the escalation
  rule under Claude's Discretion turns out never to fire in practice, that is
  the evidence to revisit this with.

- **Rewiring `inserter`, `server`, `mutate`, and `agent/filekv` onto the shared
  resolution path** — CONS-02/03/04, already scoped to Phase 14 (D-03).

- **`compiler/lib/config.go:68`'s silent `ErrUnexpectedType` skip.** Noted as a
  fourth in-compiler resolution site whose silent-nil branch sits awkwardly
  beside D-02's loud-failure decision. Whether it is in scope for this phase or
  a follow-up is a planning call; it is recorded here so it is not lost.

</deferred>

---

*Phase: 13-exact-symbol-index-shared-type-url-resolution*
*Context gathered: 2026-09-08*
