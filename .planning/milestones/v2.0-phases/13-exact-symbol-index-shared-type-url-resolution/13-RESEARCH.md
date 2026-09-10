# Phase 13: Exact Symbol Index & Shared Type-URL Resolution - Research

**Researched:** 2026-09-08
**Domain:** Proto descriptor symbol indexing (parse-without-link), content-keyed disk cache, shared `protoregistry`-shaped type-URL resolution
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01: Scan before index.** Resolution order: construction-time
  `protoregistry.Types` snapshot → the registry's growable `MessageRegistry` →
  **scoped lexical scan** over `src/`, whose hits are confirmed by a no-link
  parse → **symbol index**, built and cached on demand → hard error. The scan
  sits *ahead of* index construction, not behind it. Measured this session on
  the 799-proto corpus at `../protoconf-terraform/example/src` (37.8 MB): a
  whole-corpus lexical scan costs ~11ms warm, against 1,286ms for
  `ParseFilesButDoNotLink` over the same tree. Rejected: the scan as a tier
  *after* the index (an index miss means the symbol truly is not in `src/`,
  and a same-tree scan misses too).

- **D-02: The index replaces `ParseAll` as the last tier.** A type URL that
  survives the whole chain is a **hard error**, not a reason to parse 799
  files. The error names the unresolved type URL, the directories searched,
  and whether the index was cold or a cache hit. One-way: this deletes the
  eager fallback that today makes every resolution failure recoverable-but-
  slow. **Accepted risk:** an index or scan blind spot turns a slow compile
  into a failed one — chosen over "index-then-`ParseAll`-with-a-warning"
  because leaving the 4.6s path reachable is what this milestone exists to
  prevent.

- **D-05: The scan pattern must be nesting-tolerant, and a lexical hit is
  never an answer.** Measured on the corpus: 4,054 message declarations are
  top-level, 126,530 are nested and indented; an `^message`-anchored pattern
  finds only 3.1% of symbols. The scan matches indented declarations and
  yields candidate files; the symbol is confirmed by a no-link parse of those
  candidates, never by the lexical match itself. Candidate counts vary by
  three orders of magnitude (`AwsS3Bucket` → 4 files, `Tags` → 15,
  `Timeouts` → 351), so the type URL's own structure (package prefix →
  directory, first non-package segment → top-level message) narrows the set
  before verification.

- **D-03: Compiler only.** This phase builds the shared, index-backed path
  and wires the compiler onto it (TYPE-09, CONS-05). `inserter/inserter.go:369`,
  `server/server.go:607`, and `mutate/mutate.go:76` keep calling
  `LocalResolver` / `anyResolver` directly until Phase 14 (CONS-02/03/04).
  **Consequence:** ROADMAP success criterion 1's "grepping for type-URL
  resolution logic finds that single implementation" does **not** fully pass
  when Phase 13 closes — three call sites remain; that criterion completes in
  Phase 14. `ParseAll` already short-circuits on `len(d.ImportPaths) == 0`
  (`utils/utils.go:436`), which is every eager registry, so D-02 is
  automatically compiler-scoped.

- **D-04: Exported index-build counter** on `DescriptorRegistry`, mirroring
  Phase 12's D-05/D-06 precedent: an exported method alongside
  `LoadedFileCount()` (`utils/utils.go:512`), test-only — no log line, no CLI
  surface. The criterion-3 test compiles a config whose only mutable value is
  a `google.protobuf.Value` and asserts the build count is 0. Rejected: a
  timing assertion folded into `TestCompilerStartupScaling`.

### Claude's Discretion

- **Index symbol-kind coverage** — messages only (literal TYPE-01), messages
  + enums (mirroring `GetTypesResolver`, `utils/utils.go:196-217`), or all
  named symbols including services and extensions. Constraint: decide on what
  the six consumers actually look up and on what a no-link parse yields for
  free; do not narrow a capability that resolves today.
- **Cache key and invalidation mechanism** (TYPE-05/06). Candidates:
  `dirhash.HashDir` over `src/` (already a direct dependency, already keys the
  module repo caches at `compiler/lib/module_service.go:273`); a
  `(path, size, mtime)` stat manifest as a warm fast path with a content hash
  on mismatch; or the `.fds` checksum idiom (`utils/utils.go:617-620`)
  re-keyed over source bytes. Constraint: record the measured hash cost over
  the 799-proto corpus alongside the choice; any staleness check must be
  one-way (a mismatch may only cause *more* work, never a stale serve). Since
  the index is built on first need (TYPE-07), the validity check is also only
  paid on first need.
- **The scan → index escalation rule.** Escalate when the scan finds zero
  candidates and when candidates fail to verify; whether a candidate-count
  threshold also escalates is open. Constraint: derive the break-even by
  timing a single no-link parse against the index build, and record the
  number with the rule. If the rule never fires, TYPE-01/02/05/06 ship as
  dead code; that outcome must be visible, not silent.
- **On-disk index format and in-memory shape.** 130,584 message symbols on
  the benchmark corpus, ~138K including enums and services. Constraint: the
  format must be validated before it is trusted, in the spirit of
  `registry.Load`'s checksum gate.
- **Whether the index build is singleflighted** the way `ParseOne` is
  (`golang.org/x/sync/singleflight`, already a direct dependency). Constraint:
  two concurrent compiles must not both pay the build.
- **The scan's and the index build's concurrency contract.** Constraint:
  Phase 11's lock discipline holds — never hold `d.mu` while calling
  `parser.ParseFiles`; `utils/parse_all_deadlock_test.go` guards it. Both new
  tiers do filesystem work and then parse, so both are exposed to that
  inversion.
- **Whether `parser.ReadConfig` needs a pre-pass at all.** `protojson`
  consults its `Resolver` for every `Any` it meets, at any nesting depth, so
  an index-backed `TypeResolver` may make `ReadConfig` (`parser.go:187`)
  correct with zero changes. Constraint: verify this against `protojson`'s
  actual behavior with a nested-`Any` fixture before relying on it. **This
  session's finding: verified true — see "Code Examples" and Pitfall 5
  below. No `ReadConfig` pre-pass is required.**

### Deferred Ideas (OUT OF SCOPE)

- **Replace the symbol index entirely with scan + verify-after-parse.** Not
  taken because it revises PROJECT.md's stated key design decision and guts
  TYPE-01/02/05/06/07 — a roadmap amendment via `/gsd-phase`, not a phase-plan
  decision. Revisit if the escalation rule never fires in practice.
- **Rewiring `inserter`, `server`, `mutate`, and `agent/filekv` onto the
  shared resolution path** — CONS-02/03/04, scoped to Phase 14 (D-03).
- **`compiler/lib/config.go:68`'s silent `ErrUnexpectedType` skip.** A
  fourth in-compiler resolution site whose silent-nil branch sits awkwardly
  beside D-02's loud-failure decision. Recorded here so it is not lost;
  whether it is in scope for this phase or a follow-up is a planning call.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| TYPE-01 | Symbol index maps every message symbol under `src/`, nested types included, to its declaring file | "Code Examples" §1 (`ParseFilesButDoNotLink` + recursive `DescriptorProto.NestedType` walk), "Architecture Patterns" Pattern 1 |
| TYPE-02 | Index built by parsing without linking | `protoparse.Parser.ParseFilesButDoNotLink` verified at `$GOMODCACHE/github.com/jhump/protoreflect@v1.16.0/desc/protoparse/parser.go:247` |
| TYPE-03 | A type URL nested inside an `Any`, at any depth, resolves through the index | Pitfall 5 + "Code Examples" §3 — `protojson.decoder.unmarshalAny` recurses through `d.unmarshalMessage`, re-consulting `d.opts.Resolver` at every depth, verified in source this session |
| TYPE-04 | Only files actually referenced by resolved symbols get linked; building the index links nothing | `ParseFilesButDoNotLink`'s doc comment (verified, quoted in Code Examples) — no linking happens; "Architecture Patterns" Pattern 1 keeps the index's symbol→path map separate from `FileRegistry`, which is populated only by the existing `ParseOne` (real parse+link) once a path is known |
| TYPE-05 | Index persisted under `.protoconf_cache`, content-keyed | "Architecture Patterns" Pattern 2 (cache shape modeled on `registry.Load`/`registry.Store`, `utils/utils.go:561-623`, verified) |
| TYPE-06 | A change to any `.proto` under `src/` invalidates the cached index | Same as TYPE-05 — `dirhash.HashDir` already a direct dependency (`compiler/lib/module_service.go:30,273`, verified) |
| TYPE-07 | Index built on first need only | "Architecture Patterns" Pattern 1 — the scan tier (D-01) sits ahead of index construction, so most lookups never reach it; D-04's counter is the observable |
| TYPE-08 | Type-URL resolution is one shared code path, not duplicated per call site | `RegistryTypeResolver` (`compiler/lib/parser/parser.go:57-115`, verified) already IS that single path for the compiler (D-03 scope); the index/scan tiers slot into its two lookup methods |
| TYPE-09 | `parser.ReadConfig` resolves every nested `@type`, at any depth, through that path | Pitfall 5 — verified via `protojson` source read this session; likely zero code change to `ReadConfig` itself |
| CONS-05 | Compiling a config that loads a mutable config resolves the mutable value's type and every nested `Any` within it correctly | Pitfall 4 — `starlark_loader.go:187` bypasses the shared path via `moduleService.GetProtoRegistry().MessageRegistry.FindMessageTypeByUrl` directly; this line is the CONS-05 divergence to close |

</phase_requirements>

## Summary

This phase has no new library to learn — every mechanism it needs already
exists in this codebase or its already-vendored dependencies. The real work is
composition: `protoparse.Parser.ParseFilesButDoNotLink` (verified present at
`jhump/protoreflect@v1.16.0`) becomes the index-build primitive, its
`[]*descriptorpb.FileDescriptorProto` output already carries every nested
message via `DescriptorProto.NestedType` (verified field names in
`descriptorpb`), `dirhash.HashDir` and the `registry.Load`/`Store` checksum
idiom (already used for `.fds` module caches) become the persistence shape for
TYPE-05/06, and `RegistryTypeResolver` (`compiler/lib/parser/parser.go:57-115`)
is already the single chokepoint (TYPE-08) — its two lookup methods
(`FindMessageByURL`, `FindMessageByName`) are exactly where D-01's scan and
index tiers insert, replacing the two `_ = r.registry.ParseAll()` calls D-02
deletes.

One significant, verified-this-session finding materially simplifies the
plan: `msgregistry.MessageRegistry.AddFile` (called from `recordFileLocked`,
`utils/utils.go:419`) already recurses into nested message types via
`addMessageTypesLocked`/`GetNestedMessageTypes()` (verified at
`jhump/protoreflect@v1.16.0/dynamic/msgregistry/message_registry.go:149-178`).
So **once a file's path is known and `ParseOne` has loaded it, nested-type
resolution through `MessageRegistry.FindMessageTypeByUrl` already works today**
— the actual gap TYPE-01 exists to close is narrower than "index every nested
symbol's full type information": the index only has to answer **"which file
declares symbol X"** for a symbol whose file is not yet known, so `ParseOne`
can load that one file and let the existing (already nested-aware)
`MessageRegistry.AddFile` machinery take over. This reframes the index as a
**path-lookup structure**, not a type-registration structure — smaller to
build, smaller to persist, and it explains why `GetTypesResolver`'s top-level-
only limitation (`utils/utils.go:196-217`, feeding the *construction-time
snapshot* `LocalResolver`) is a real but separate, already-known and
already-fenced gap (the snapshot tier), not evidence the whole resolution
chain is nested-blind.

A second verified finding closes an open discretion item outright:
`protojson`'s `unmarshalAny` (`google.golang.org/protobuf@v1.36.12/encoding/
protojson/well_known_types.go:169-220`, read this session) resolves the
top-level `@type`, then recurses via `d.unmarshalMessage(em, true)` using the
**same decoder**, so any `Any` nested inside that embedded message re-invokes
`d.opts.Resolver.FindMessageByURL` at whatever depth it appears. `parser.
ReadConfig` (`parser.go:187`) already passes `p.TypeResolver` as that
`Resolver`. Once `TypeResolver` is backed by the scan+index chain (D-01),
`ReadConfig` needs **zero code changes** to satisfy TYPE-09/TYPE-03 — the
"proto_file pre-pass" PITFALLS.md once proposed is confirmed unnecessary, as
the discretion item already suspected.

The remaining hazard is not designing the index — it's the **blast radius of
D-02's deletion of `ParseAll`**. Three existing, passing tests
(`compiler/lib/eager_fallback_visible_test.go`,
`utils/parse_all_deadlock_test.go`, `utils/growable_resolver_test.go`) assert
directly on `ParseAll`, `eagerFallback`, and `FellBackToEager()` — including a
structured-logging contract (`"compile finished"` with an `eagerFallback`
attribute) that an operator can currently query. D-02 is explicit that this is
a one-way deletion; the plan must replace these tests' *intent* (an operator
must still be able to see, after the fact, whether the slow tier fired) with
an index-build/scan-tier equivalent, not just delete them.

**Primary recommendation:** Extend `RegistryTypeResolver`'s two lookup methods
in place: insert the scan tier and the index tier between the existing
`MessageRegistry` branch and where `ParseAll` used to sit, replace the
`ParseAll` calls with a hard `fmt.Errorf` naming the URL/name, directories
searched, and index cache state, and give `DescriptorRegistry` the index's
build/cache logic as new methods alongside `ParseOne` — same file, same lock
(`d.mu`), same `singleflight.Group` pattern, following the "single source of
truth" precedent both prior phases established.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Symbol → declaring-file index (build, persist, load) | Backend / Library (`utils.DescriptorRegistry`) | Filesystem / Storage (`.protoconf_cache/`) | Pure in-process Go; the persisted artifact lives in storage, the build/query logic in the library, matching the `.fds` module-cache precedent |
| Scoped lexical scan tier | Backend / Library (`utils.DescriptorRegistry` or a sibling in `compiler/lib/parser`) | — | Filesystem read + regex, no network, no client tier |
| Shared type-URL resolution chokepoint | Backend / Library (`compiler/lib/parser.RegistryTypeResolver`) | — | Already the single implementation for the compiler (D-03); this phase extends its tiers, does not relocate it |
| `parser.ReadConfig` / nested `@type` resolution | Backend / Library (`compiler/lib/parser.Parser`) | — | One `protojson.Unmarshal` call whose `Resolver` this phase's chain backs; no new tier of its own |
| Mutable-config load (`loadMutable`, CONS-05) | Backend / Library (`compiler/lib/starlark_loader.go`) | — | In-process Starlark loader; the divergence to close (`:187`) is a same-tier bypass, not a cross-tier concern |
| Index-build observable counter (D-04) | Backend / Library (test-only exported method) | — | Mirrors `LoadedFileCount()` — no CLI/log surface |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/jhump/protoreflect` | v1.16.0 `[VERIFIED: go.mod:19]` | `protoparse.Parser.ParseFilesButDoNotLink` — the index-build primitive, confirmed present `[VERIFIED: $GOMODCACHE/github.com/jhump/protoreflect@v1.16.0/desc/protoparse/parser.go:247]` | Already the project's only proto-parsing dependency; this method is purpose-built for exactly TYPE-02's "parse without linking" requirement |
| `google.golang.org/protobuf` | v1.36.12 `[VERIFIED: go.mod:48 — note: newer than the v1.34.1 recorded in Phase 11's research, re-verify no breaking `protojson`/`protoregistry` API changes if the plan pins a different version]` | `descriptorpb.FileDescriptorProto`/`DescriptorProto` (index-build input shape), `protojson` (`ReadConfig`'s `Resolver` consumer) | Already in use everywhere; field names (`MessageType`, `NestedType`, `EnumType`, `Package`) verified this session in `$GOMODCACHE/google.golang.org/protobuf@v1.36.12/types/descriptorpb/descriptor.pb.go` |
| `golang.org/x/mod/sumdb/dirhash` | v0.38.0 `[VERIFIED: go.mod:43]` | `dirhash.HashDir` — content-keying the index cache (TYPE-05/06) | Already a direct dependency, already used identically for module-repo cache keys (`compiler/lib/module_service.go:30,273`) |
| `golang.org/x/sync/singleflight` | v0.22.0 `[VERIFIED: go.mod:45; already imported in utils/utils.go:25]` | Collapsing concurrent index builds into one (the discretion item) | Already the project's collapsing-duplicate-work primitive, used identically for `ParseOne` |

**No new external package is required for this phase.**

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/jhump/protoreflect/dynamic/msgregistry` | v1.16.0 | `MessageRegistry.AddFile`/`FindMessageTypeByUrl` — unchanged, but its already-nested-aware behavior (verified this session) is load-bearing for why the index only needs to answer path lookups | Reuse as-is; do not re-implement nested-type registration |
| `google.golang.org/protobuf/reflect/protoregistry` | v1.36.12 | `protoregistry.NotFound` sentinel — the hard-error shape D-02 wants (`fmt.Errorf("%w: %s", protoregistry.NotFound, url)` is the existing pattern at `parser.go:84`) | Reuse the existing error-wrapping convention for the new hard-error path |
| `log/slog` | stdlib | Structured logging for whatever replaces `eagerFallback`'s "compile finished" observable (see Pitfall 6) | Already the project's logging convention (`compiler/lib/eager_fallback_visible_test.go` pins the exact shape to preserve/replace) |
| `github.com/stretchr/testify` | v1.9.0 | New tests for TYPE-01..09, CONS-05 | Already the project's assertion library everywhere |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `ParseFilesButDoNotLink` for the index build | `protoparse.Parser.ParseFiles` (full link) then discard linked info | Rejected by the milestone's own measurement: 1,286ms unlinked vs 4,639ms linked over the 799-proto corpus (`.planning/research/compiler-performance/BASELINE.md`, `OPTIONS.md`) — linking is exactly the discarded-99%-of-work OPTIONS.md already rejected once for the eager-parse case |
| `dirhash.HashDir` for the cache key | A `(path, size, mtime)` stat manifest | Stat manifest is cheaper on a warm run (no content read) but requires a full content-hash fallback on any mismatch to stay correct against a `touch`-without-edit false-negative; `dirhash.HashDir` is simpler (one mechanism, not two) and its cost on this corpus is unmeasured — record the number before choosing (open per CONTEXT.md discretion) |
| A persisted, singleflighted index build | Rebuild the index every cold compile | Rejected — TYPE-05/06 explicitly require persistence; an unpersisted index defeats "warm run does not rebuild it" |

**Installation:** none — no new packages.

**Version verification:** `github.com/jhump/protoreflect@v1.16.0` confirmed present in `go.mod:19` and `$GOMODCACHE`, with `ParseFilesButDoNotLink` confirmed present at that exact version this session. `google.golang.org/protobuf` is now `v1.36.12` in `go.mod:48` — **this has moved since Phase 11's research recorded v1.34.1**; the `protojson`/`descriptorpb` behavior this document relies on was verified directly against the `v1.36.12` source tree in `$GOMODCACHE`, not assumed from the older research doc.

## Package Legitimacy Audit

Not applicable — this phase introduces zero new external packages. Every
primitive used (`ParseFilesButDoNotLink`, `dirhash.HashDir`,
`singleflight.Group`, `descriptorpb` field access, `protojson.Resolver`) is
part of an already-vendored, already-imported dependency, each confirmed
present in `$GOMODCACHE` and/or `go.mod` this session.

## Architecture Patterns

### System Architecture Diagram

```
RegistryTypeResolver.FindMessageByURL(url) / FindMessageByName(name)
   (compiler/lib/parser/parser.go:66,87 — the ONE chokepoint, D-03 scope)
        │
        ▼
  ┌─────────────────────────────────────────────────────────────┐
  │ Tier 0: construction-time snapshot (*protoregistry.Types)     │  <- unchanged
  │         r.snapshot.FindMessageByURL/Name                       │
  └─────────────────────────────────────────────────────────────┘
        │ miss (protoregistry.NotFound)
        ▼
  ┌─────────────────────────────────────────────────────────────┐
  │ Tier 1: growable MessageRegistry (already nested-aware,        │  <- unchanged
  │         verified: AddFile -> addMessageTypesLocked recurses    │
  │         GetNestedMessageTypes())                               │
  │         r.registry.MessageRegistry.FindMessageTypeByUrl        │
  └─────────────────────────────────────────────────────────────┘
        │ miss
        ▼
  ┌─────────────────────────────────────────────────────────────┐
  │ Tier 2: scoped lexical scan (D-01, D-05 — NEW)                 │
  │  1. Derive package prefix -> directory, top-level symbol       │
  │     from the URL/name.                                         │
  │  2. Scan candidate .proto files under src/ for a nesting-       │
  │     tolerant indented-declaration match (not ^message).        │
  │  3. For each candidate, ParseOne(candidatePath) (real parse+    │
  │     link, existing method) — the scan NEVER answers on its own. │
  │  4. Re-check Tier 1 after each ParseOne (recordFileLocked       │
  │     already ran MessageRegistry.AddFile with nested recursion). │
  └─────────────────────────────────────────────────────────────┘
        │ zero candidates, or all candidates fail to verify
        ▼
  ┌─────────────────────────────────────────────────────────────┐
  │ Tier 3: symbol index, built+cached on demand (TYPE-01/02/05/   │
  │         06/07 — NEW)                                           │
  │  1. Check .protoconf_cache/<index-file>, validate content key   │
  │     (dirhash.HashDir(srcPath) or equivalent, TBD per discretion)│
  │  2. On miss/stale: ParseFilesButDoNotLink(all .proto under      │
  │     src/) -> walk FileDescriptorProto.MessageType (+ recursive  │
  │     NestedType) -> build symbol->path map -> persist.           │
  │  3. Look up the URL/name's path in the map.                     │
  │  4. ParseOne(thatPath) (existing method — links ONLY that file  │
  │     and its transitive deps, TYPE-04).                          │
  │  5. Re-check Tier 1.                                            │
  └─────────────────────────────────────────────────────────────┘
        │ still miss
        ▼
  ┌─────────────────────────────────────────────────────────────┐
  │ Hard error (D-02 — replaces ParseAll entirely)                 │
  │  fmt.Errorf("%w: %s (searched: %v; index: %s)",                │
  │    protoregistry.NotFound, url, dirsSearched, indexState)       │
  └─────────────────────────────────────────────────────────────┘
```

### Recommended Project Structure

No new packages — extend existing files:

```
utils/utils.go                  # DescriptorRegistry: symbol index build/cache
                                 #   methods, alongside ParseOne/ParseAll (delete
                                 #   ParseAll's callers, D-02); scan-tier helper
compiler/lib/parser/parser.go   # RegistryTypeResolver.FindMessageByURL/Name:
                                 #   insert scan + index tiers, replace ParseAll
                                 #   calls with the hard error
compiler/lib/starlark_loader.go # loadMutable (CONS-05): route the third lookup
                                 #   (:187, moduleService.GetProtoRegistry().
                                 #   MessageRegistry.FindMessageTypeByUrl)
                                 #   through l.parser.TypeResolver instead
```

### Pattern 1: Index build — parse without linking, recurse nested types

**What:** Walk every `.proto` under `src/` with `ParseFilesButDoNotLink`
(returns raw, unlinked `*descriptorpb.FileDescriptorProto`), then recurse each
file's `MessageType` (top-level) and each message's `NestedType` (recursive)
to build a `map[fullSymbolName]declaringFilePath`.

**When to use:** Only on Tier 3 (index cold or stale), which Tier 2's scan is
designed to make rare.

**Example — field names and API verified this session, shape only:**

```go
// Source: verified this session against
// $GOMODCACHE/github.com/jhump/protoreflect@v1.16.0/desc/protoparse/parser.go:247
// and $GOMODCACHE/google.golang.org/protobuf@v1.36.12/types/descriptorpb/descriptor.pb.go:1437-1612
p := &protoparse.Parser{
    ImportPaths: []string{srcPath}, // note: doc comment at parser.go:247 says
                                     // "p.ImportPaths = nil // not used for this
                                     // 'do not link' operation" -- verify at plan
                                     // time whether filenames must be passed
                                     // relative to srcPath or as absolute paths,
                                     // and whether Accessor still needs os.Open
                                     // wiring identical to Import()/Parse().
    Accessor: func(f string) (io.ReadCloser, error) { return os.Open(f) },
}
protos, err := p.ParseFilesButDoNotLink(allProtoPaths...) // []*descriptorpb.FileDescriptorProto

index := map[string]string{} // fully-qualified symbol -> declaring file path

var walkMessages func(pkg string, path string, msgs []*descriptorpb.DescriptorProto)
walkMessages = func(pkg, path string, msgs []*descriptorpb.DescriptorProto) {
    for _, m := range msgs {
        full := pkg + "." + m.GetName() // adjust for nested scoping: a Nested
                                         // type's full name is Outer.Inner, not
                                         // pkg.Inner -- accumulate the enclosing
                                         // message name(s) into the prefix, not
                                         // just the package.
        index[full] = path
        walkMessages(full, path, m.GetNestedType()) // DescriptorProto.NestedType,
                                                      // verified field name
    }
}
for _, fd := range protos {
    walkMessages(fd.GetPackage(), fd.GetName(), fd.GetMessageType())
}
```

### Pattern 2: Content-keyed persisted cache — mirrors `registry.Load`/`Store`

**What:** The exact checksum-gate shape `registry.Load`/`Store` already use
for `.fds` module caches (`utils/utils.go:561-623`, read this session),
re-applied to the symbol index artifact.

**Example — the existing pattern to copy, verified this session:**

```go
// Source: utils/utils.go:606-623, read this session — the shape TYPE-05/06's
// index cache should follow (a different key derivation, same validate-before-
// trust structure):
func (d *DescriptorRegistry) Load(path, checksum string) error {
    b, err := os.ReadFile(path)
    if err != nil {
        return err
    }
    fds := &descriptorpb.FileDescriptorSet{}
    err = proto.Unmarshal(b, fds)
    if err != nil {
        return err
    }
    md5sum := fileDescriptorSetSum(fds)
    if checksum != md5sum {
        return fmt.Errorf("failed to validate file content: %s (expected: %s, got: %s)", path, checksum, md5sum)
    }
    d.MergeFileDescriptorSet(fds)
    return nil
}
```

The index's equivalent: read the persisted `symbol -> path` map, recompute
`dirhash.HashDir(srcPath, "", <hashFn>)` (verified signature used identically
at `compiler/lib/module_service.go:273`), compare against a stored key, and
refuse (rebuild) on mismatch — never serve stale (D-01's constraint).

### Pattern 3: `RegistryTypeResolver`'s existing miss-fallthrough — where tiers insert

**What:** The two lookup methods already have the exact shape D-01 describes;
this phase adds tiers between the existing `MessageRegistry` check and the
`ParseAll` call, which D-02 deletes.

**Example — current code, read this session, showing exactly where to cut:**

```go
// Source: compiler/lib/parser/parser.go:66-85, read this session
func (r *RegistryTypeResolver) FindMessageByURL(url string) (protoreflect.MessageType, error) {
    mt, err := r.snapshot.FindMessageByURL(url)          // Tier 0 — unchanged
    if err == nil {
        return mt, nil
    }
    if !errors.Is(err, protoregistry.NotFound) {
        return nil, err
    }
    if md, mErr := r.registry.MessageRegistry.FindMessageTypeByUrl(url); mErr == nil && md != nil {
        return dynamicpb.NewMessageType(md.UnwrapMessage()), nil // Tier 1 — unchanged
    }
    // <-- D-01's scan tier (Tier 2) and index tier (Tier 3) insert HERE,
    //     each followed by a re-check of the MessageRegistry branch above
    //     (ParseOne's recordFileLocked already re-populates it with nested
    //     types, verified — no new registration code needed).
    _ = r.registry.ParseAll()  // <-- D-02 DELETES this call and its FindMessageByName twin
    if md, mErr := r.registry.MessageRegistry.FindMessageTypeByUrl(url); mErr == nil && md != nil {
        return dynamicpb.NewMessageType(md.UnwrapMessage()), nil
    }
    return nil, fmt.Errorf("%w: %s", protoregistry.NotFound, url)  // <-- D-02's hard
                                                                     //     error needs
                                                                     //     more context
                                                                     //     (dirs searched,
                                                                     //     index state)
}
```

### Anti-Patterns to Avoid

- **Re-implementing nested-type registration.** `MessageRegistry.AddFile`
  already recurses `GetNestedMessageTypes()` (verified this session). Do not
  write a second nested-symbol registration path inside the index/scan tiers
  — they only need to answer "which file", then hand off to the existing
  `ParseOne` → `recordFileLocked` → `MessageRegistry.AddFile` chain.
- **Trusting a scan or index hit without verification.** D-05 and D-01 are
  explicit and non-negotiable: a lexical match or an index entry is a
  *candidate*, confirmed only by `ParseOne` actually loading the file and the
  `MessageRegistry` re-check succeeding. Protobuf permits any package/path
  relationship; a heuristic that skips this step will resolve to the wrong
  type on a hand-written repo even if it happens to work on the benchmark
  corpus.
- **Building the index or running the scan while holding `d.mu`.** Both do
  filesystem work followed by a parse; Phase 11's lock discipline
  (`utils/parse_all_deadlock_test.go`) exists specifically to catch this
  class of self-deadlock. Follow `ParseOne`'s existing pattern: read what's
  needed under `RLock`, release, do the I/O/parse, then take `Lock` only for
  the insert.
- **Letting the index answer a lookup without a subsequent real parse.** The
  index maps symbol → path; it must never itself return a `protoreflect.
  MessageType` or a `*desc.MessageDescriptor` — that would bypass `ParseOne`'s
  transitive-import handling and canonical-pointer contract (RSLV-03, Phase
  12), silently reintroducing a second source of truth for descriptors.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Parsing every proto without linking | A custom lightweight `.proto` tokenizer | `protoparse.Parser.ParseFilesButDoNotLink` | Purpose-built, already a dependency, already measured (1,286ms/799 files) |
| Content-keying a directory tree | A hand-rolled recursive-hash walker | `golang.org/x/mod/sumdb/dirhash.HashDir` | Already a direct dependency, already used identically for the module-repo cache (`module_service.go:273`) |
| Collapsing concurrent identical index builds | A custom mutex-per-key map | `golang.org/x/sync/singleflight.Group` | Already the project's collapsing-duplicate-work primitive (`ParseOne` uses it identically) |
| Registering nested message types from a parsed file | A second symbol-registration pass alongside the index | `msgregistry.MessageRegistry.AddFile` (already recurses nested types, verified) | Duplicating this is redundant work and a second source of truth for "is this symbol known" |
| Validating a cache artifact before trusting it | Ad-hoc file-exists checks | The `registry.Load`/`Store` checksum-gate pattern (`utils/utils.go:561-623`) | Already the project's cache-trust idiom; re-deriving it for the index cache invites a different (weaker) invalidation bug |

**Key insight:** every primitive TYPE-01..09/CONS-05 needs — unlinked parse,
content hashing, duplicate-work collapsing, nested-symbol registration,
cache-trust validation — already has a working, tested precedent in this
exact codebase. The phase composes them into two new tiers of one existing
chokepoint; it introduces no new library and, once the index/scan tiers are
built, arguably *less* new surface than Phase 11 or 12 needed.

## Common Pitfalls

### Pitfall 1: D-02 deletes tests, not just code — three existing tests assert directly on `ParseAll`/`eagerFallback`

**What goes wrong:** `compiler/lib/eager_fallback_visible_test.go`
(`TestCompileFinishedReportsEagerFallback`), `utils/parse_all_deadlock_test.go`
(`TestParseAllConcurrentWithParseOneDoesNotDeadlock`), and
`utils/growable_resolver_test.go` (`TestParseAllRegistersIntoFilesResolver`,
`TestFilesResolverRegistrationErrorsStayZero`) all call `ParseAll()`,
`FellBackToEager()`, or assert on the `"compile finished"` log line's
`eagerFallback` attribute directly — all verified present and passing this
session. D-02 says the index replaces `ParseAll` as the last tier; a plan
that deletes `ParseAll` without touching these tests leaves the package not
compiling, not just red.

**Why it happens:** These tests were written in Phase 11/12 specifically to
pin the *previous* fallback's observability contract (operators must be able
to see the slow path fired) — a good contract, whose *mechanism* this phase
is explicitly tasked with replacing.

**How to avoid:** Treat this as in-scope rework, not incidental breakage.
Preserve the *intent* (operator-visible "did the slow tier fire, and how
slow") under the new tier names — e.g. rename/replace the `eagerFallback`
boolean with something that distinguishes "resolved via scan", "resolved via
index (cold build)", "resolved via index (cache hit)", "hard error", matching
D-04's counter precedent. `TestParseAllConcurrentWithParseOneDoesNotDeadlock`
specifically should become (or be joined by) an equivalent test that races the
*new* index-build tier against `ParseOne` — the lock-discipline hazard it
guards against is identical for the new code path (Claude's Discretion,
concurrency contract).

### Pitfall 2: `ParseFilesButDoNotLink`'s `ImportPaths = nil` line needs verification against how `find()`/`Import()` currently pass filenames

**What goes wrong:** The doc comment at
`$GOMODCACHE/github.com/jhump/protoreflect@v1.16.0/desc/protoparse/parser.go:247`
reads `p.ImportPaths = nil // not used for this "do not link" operation`,
verified this session. `utils.DescriptorRegistry.Import` (`utils/utils.go:219-
269`) currently passes `paths...` as `ImportPaths` for the parser and separately
trims the path prefix off each filename before calling `parse(parser, files)`.
It is not yet confirmed this session whether `ParseFilesButDoNotLink` expects
filenames relative to CWD, relative to some other root, or resolved via its
internal `getResolver(filenames)` fallback when `Accessor` is unset.

**Why it happens:** `ParseFilesButDoNotLink` is a different entry point than
`ParseFiles` (used everywhere else in this codebase); its filename-resolution
contract has not been exercised in this repo before.

**How to avoid:** Before writing the index-build code, run a small isolated
call (e.g. against `utils/testdata/small/src/`) and confirm the returned
`FileDescriptorProto.GetName()` values match what the rest of the registry
uses as keys (`FileRegistry`'s map key convention) — this is a five-minute
spike the plan should schedule as its first task, not an assumption to carry
into the main implementation.

### Pitfall 3: Full symbol name construction for nested types must accumulate enclosing message names, not just the package

**What goes wrong:** A naive implementation might build the index key as
`pkg.Name` for every `DescriptorProto` encountered during the recursive walk,
which is correct for top-level messages but wrong for nested ones — a nested
`Timeouts` inside `AwsS3Bucket` in package `terraform.aws.resources.v6` has
full name `terraform.aws.resources.v6.AwsS3Bucket.Timeouts`, not
`terraform.aws.resources.v6.Timeouts`.

**Why it happens:** Easy to conflate "the package this file declares" with
"the full scope prefix for this specific message," especially since
`ParseFilesButDoNotLink`'s unlinked output does not pre-compute full names the
way a linked `desc.MessageDescriptor.GetFullyQualifiedName()` would.

**How to avoid:** Thread the accumulating prefix through the recursive walk
(Pattern 1's example does this: `full := pkg + "." + m.GetName()`, then
recurse with `full` as the new prefix for `m.GetNestedType()`). Add a unit
test asserting the index correctly maps a doubly-nested symbol using a small
fixture proto with an `Outer.Middle.Inner` shape.

### Pitfall 4: `loadMutable`'s third lookup bypasses the shared path entirely — this is CONS-05's actual failure mode

**What goes wrong:** `compiler/lib/starlark_loader.go:187`
(`l.moduleService.GetProtoRegistry().MessageRegistry.FindMessageTypeByUrl(protoconfValue.Value.TypeUrl)`,
read this session) calls `MessageRegistry.FindMessageTypeByUrl` **directly**,
not through `l.parser.TypeResolver`. This means that even after D-01's scan
and index tiers are wired into `RegistryTypeResolver`, this specific call site
never reaches them — it only ever sees Tier 1 (the growable `MessageRegistry`
as it currently stands), so a mutable config carrying a type whose file has
never been loaded by any other means will still fail here, silently
regressing CONS-05.

**Why it happens:** This line predates the lazy/growable-resolver work; it
was written when `GetProtoRegistry()` was always eagerly populated, so
`MessageRegistry` already contained everything.

**How to avoid:** Route this third lookup through `l.parser.TypeResolver.
FindMessageByURL(...)` instead — the same resolver the line immediately
above (`:177`) already uses. Note lines `:177` and `:187` currently perform
what looks like a redundant double-resolution of the same `TypeUrl` (once via
`TypeResolver`, once via raw `MessageRegistry`); the plan should verify
whether `:187`'s separate lookup is even semantically necessary (it appears
to exist only to obtain a `*desc.MessageDescriptor` for `dynamic.NewMessage`,
where `:177`'s result is a `protoreflect.MessageType`) before deciding whether
to collapse it into one call or keep two calls both routed through
`TypeResolver`.

### Pitfall 5: `protojson`'s `Any` recursion is real, but only as deep as the `Resolver` it was given at the top — confirms TYPE-09/TYPE-03 need no `ReadConfig` change, given the chain resolves correctly

**What goes wrong (if unverified):** A plan might over-engineer a
`ReadConfig` pre-pass (as PITFALLS.md originally proposed and later retracted)
out of caution, when the fix is entirely in what `TypeResolver` can answer.

**Verified this session:**
`$GOMODCACHE/google.golang.org/protobuf@v1.36.12/encoding/protojson/well_known_types.go:169-220`
— `decoder.unmarshalAny` resolves `typeURL` via `d.opts.Resolver.
FindMessageByURL(typeURL)`, then for a non-well-known embedded type calls
`d.unmarshalMessage(em, true)` **using the same decoder `d`** (not a fresh
one), so any `Any` field inside `em`, at any nesting depth, re-invokes
`d.opts.Resolver.FindMessageByURL` the same way. `parser.ReadConfig`
(`parser.go:187-193`) already does
`protojson.UnmarshalOptions{Resolver: p.TypeResolver}.Unmarshal(...)`.

**How to avoid the false lead:** Do not add a pre-pass. The entire TYPE-09/
TYPE-03 surface is: make `p.TypeResolver.FindMessageByURL` (i.e.
`RegistryTypeResolver.FindMessageByURL`) correctly answer *any* symbol,
including nested ones, via the D-01 chain — `ReadConfig` itself needs no
edits. Add a nested-`Any`-inside-materialized-JSON test (not just the
existing `field_type_any_test.pconf`, which exercises the *compile-time*
`load()` path, not the `ReadConfig`/`loadMutable` path) as the criterion-1
regression guard.

### Pitfall 6: The `google.protobuf.Value` never-builds-the-index guarantee (criterion 3, D-04) depends on Tier 0/1 actually catching it before Tier 2/3 run

**What goes wrong:** If the scan tier (Tier 2) is implemented as "always run
on any `MessageRegistry` miss" without confirming `google.protobuf.Value`
resolves at Tier 0 (the unconditional global-registry seed,
`utils/utils.go:38-45`/`132`, already verified in STATE.md's 2026-09-04
decision note), the scan could fire on every mutable-`Value` compile — and
if the scan escalates to the index on any miss, criterion 3 (index-build
count must be 0) fails even though the *symbol* resolves fine.

**How to avoid:** Confirm — with a test, not by inspection — that
`google.protobuf.Value`/`Int64Value`/`Float64Value`/`StringValue` resolve at
Tier 0 (`r.snapshot.FindMessageByURL`) and never reach the scan/index tiers
at all. `NewDescriptorRegistry`'s `globalRegexMatcher` (`utils/utils.go:132`,
verified this session: `` `(google|google/rpc|google/type|buf/validate|
validate|protoconf/v1)/(.*)\.proto` ``) seeds well-known types including
`google/protobuf/*` at construction — this is the existing mechanism the
criterion-3 test should assert continues to short-circuit before D-04's
counter is ever incremented.

## Code Examples

### 1. `ParseFilesButDoNotLink` — the index-build primitive (verified API)

```go
// Source: $GOMODCACHE/github.com/jhump/protoreflect@v1.16.0/desc/protoparse/parser.go:215-247
// Doc comment, read this session (quoted in full for accuracy):
// "ParseFilesButDoNotLink parses the named files into descriptor protos. The
//  results are just protos, not fully-linked descriptors. ... This method
//  will still validate the syntax of parsed files. If the parser's
//  ValidateUnlinkedFiles field is true, additional checks, beyond syntax will
//  also be performed."
func (p Parser) ParseFilesButDoNotLink(filenames ...string) ([]*descriptorpb.FileDescriptorProto, error)
```

### 2. Nested-type field shape (verified `descriptorpb` field names)

```go
// Source: $GOMODCACHE/google.golang.org/protobuf@v1.36.12/types/descriptorpb/descriptor.pb.go
// FileDescriptorProto (lines 1437-1451), quoted field tags:
//   Name        *string                 // protobuf tag 1
//   Package     *string                 // protobuf tag 2
//   MessageType []*DescriptorProto      // protobuf tag 4, "message_type"
//   EnumType    []*EnumDescriptorProto  // protobuf tag 5, "enum_type"
//
// DescriptorProto (lines 1608-1619), quoted field tags:
//   Name       *string             // protobuf tag 1
//   Field      []*FieldDescriptorProto  // protobuf tag 2
//   NestedType []*DescriptorProto  // protobuf tag 3, "nested_type"
//   EnumType   []*EnumDescriptorProto   // protobuf tag 4, "enum_type"
```

### 3. `protojson`'s recursive `Any` resolution (verified — the TYPE-03/TYPE-09 mechanism)

```go
// Source: $GOMODCACHE/google.golang.org/protobuf@v1.36.12/encoding/protojson/well_known_types.go:169-220
// (excerpted, read in full this session)
func (d decoder) unmarshalAny(m protoreflect.Message) error {
    // ... peeks and reads the "@type" field ...
    typeURL := tok.ParsedString()
    emt, err := d.opts.Resolver.FindMessageByURL(typeURL)   // <-- the ONE call
                                                              //     TYPE-01..09
                                                              //     must make
                                                              //     correct
    // ...
    em := emt.New()
    if unmarshal := wellKnownTypeUnmarshaler(emt.Descriptor().FullName()); unmarshal != nil {
        // scalar wrapper types (Value, Int64Value, ...) — never reaches here
        // for the common production case per Pitfall 6
    } else {
        // Custom message type: recurse using the SAME decoder `d`, so a
        // nested Any inside `em` calls d.opts.Resolver.FindMessageByURL again.
        if err := d.unmarshalMessage(em, true); err != nil {
            return err
        }
    }
    // ...
}
```

### 4. `MessageRegistry`'s existing nested-type recursion (verified — why the index only needs path lookups)

```go
// Source: $GOMODCACHE/github.com/jhump/protoreflect@v1.16.0/dynamic/msgregistry/message_registry.go:149-178
// (function signatures confirmed this session)
func (r *MessageRegistry) AddFile(baseUrl string, fd *desc.FileDescriptor)
func (r *MessageRegistry) addMessageTypesLocked(baseUrl string, msgs []*desc.MessageDescriptor) {
    // ... registers each msg ...
    // r.addMessageTypesLocked(baseUrl, md.GetNestedMessageTypes())  <-- recurses
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| `RegistryTypeResolver` misses fall through to `ParseAll()`, a whole-tree eager parse+link | Misses fall through to a scoped lexical scan, then a persisted no-link symbol index, then a hard error | This phase (D-01/D-02) | The 4.6s-shaped path becomes structurally unreachable from type-URL resolution — a resolution failure becomes a fast, diagnostic error instead of a slow success |
| Nested `Any` types resolved via `proto_file` heuristic + eager fallback (PITFALLS.md's superseded proposal) | Resolved via the exact symbol index, `proto_file` stays populated for compatibility only | 2026-09-04 roadmap revision, this phase implements it | Removes the package→directory heuristic and lexical-scan-as-answer risk PITFALLS.md flagged; `proto_file` is no longer load-bearing for correctness |

**Deprecated/outdated:** `ParseAll` (`utils/utils.go:433-460`) and its
`eagerFallback`/`FellBackToEager()` observable are deleted by this phase
(D-02) — not merely deprioritized. Any future code or test still calling
`ParseAll` after this phase lands is a regression to the deleted fallback,
not a legitimate use.

## Assumptions Log

> Note: the measured numbers in `## User Constraints` (D-01/D-05's ~11ms scan,
> 1,286ms `ParseFilesButDoNotLink`, 4,054/126,530 top-level/nested symbol
> counts, candidate-count examples) are **locked decisions carried verbatim
> from `13-CONTEXT.md`**, measured during this milestone's `/gsd-discuss-phase`
> session against `../protoconf-terraform/example/src`. This research session
> independently confirmed the corpus exists and its file count matches
> (`find ../protoconf-terraform/example/src -name '*.proto' | wc -l` → 799,
> verified this session) but did not independently re-run the timing
> measurements — they are treated as authoritative per the user-constraints
> contract, not re-verified from scratch.

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `ParseFilesButDoNotLink`'s filename-resolution contract (relative-to-CWD vs relative-to-`ImportPaths`) works identically enough to `Import`/`Parse`'s existing filename handling that the index build can reuse `find()`'s output paths with no adjustment | Pitfall 2, Code Example 1 | If the contract differs, the index build either fails outright (loud, cheap to catch in Wave 0) or silently indexes symbols under the wrong path strings, which would make every subsequent `ParseOne(indexedPath)` a miss — caught by the "verify after lookup" non-negotiable (D-05), so wrong-but-caught, not silently wrong |
| A2 | The `google.protobuf.protobuf@v1.36.12` version now in `go.mod` (vs. the `v1.34.1` Phase 11's research recorded) has not changed `protojson`'s `unmarshalAny` recursion behavior between those versions in a way that matters here | Standard Stack, Pitfall 5 | Low — this session verified directly against the `v1.36.12` source tree actually in `$GOMODCACHE`, not against the older recorded version, so this assumption is about historical continuity only, not about what ships |
| A3 | The index's symbol-kind coverage should be messages + enums (mirroring `GetTypesResolver`'s existing registration surface), not messages-only or all-symbol-kinds-including-services/extensions, since no consumer identified in the six-consumer blast radius (Phase 11 research) resolves a type URL for a service or extension | Standard Stack, Don't Hand-Roll | If a consumer does resolve extensions (none confirmed this session — `RegistryTypeResolver.FindExtensionByName`/`ByNumber` delegate to the snapshot only, unchanged by this phase per `parser.go:106-115`, verified), scoping the index to messages+enums would leave that path un-upgraded; low risk since extensions are explicitly out of the growable chain already |

## Open Questions

1. **Does `ParseFilesButDoNotLink` need `ValidateUnlinkedFiles: true` set to
   catch syntax errors in files the index build touches but no config ever
   loads, mirroring `Import`'s current `ValidateUnlinkedFiles: true` setting
   (`utils/utils.go:245`)?**
   - What we know: `Import`'s parser sets this today; `ParseFilesButDoNotLink`'s
     own doc comment says validation beyond syntax only runs "If the parser's
     ValidateUnlinkedFiles field is true."
   - What's unclear: whether turning this on for the index build reintroduces
     GATE-03's already-acknowledged behavior change (a broken proto that no
     config loads is no longer reported) in a different shape — the index
     build now touches every file, so a validation error there could
     re-surface broken-tree errors the milestone deliberately chose to make
     silent for the compile path.
   - Recommendation: leave `ValidateUnlinkedFiles` false/unset for the index
     build specifically, and treat a per-file parse error during index
     construction as "this file is unusable, skip it, do not fail the whole
     index build" — matching D-02's spirit that a slow/degraded path should
     never become a hard failure for files nobody asked about. Confirm this
     against GATE-03 during planning discussion, not assumed here.

2. **Exact wire format for the persisted index** — a serialized
   `map[string]string` (trivial, `encoding/gob` or JSON) vs. reusing
   `descriptorpb.FileDescriptorSet`-shaped storage (heavier, but reuses the
   existing `.fds` tooling).
   - What we know: the index only needs to answer "path for symbol X" (per
     this research's central finding) — it does not need to store descriptor
     content, since `ParseOne` re-derives that from the path.
   - What's unclear: whether the "must be validated before trusted" (Claude's
     Discretion, on-disk format) constraint is more naturally satisfied by a
     content-addressed simple map+checksum, or by wrapping it in a proto
     message (versioned, extensible, consistent with the rest of this
     codebase's serialization conventions).
   - Recommendation: a small dedicated proto message (`SymbolIndex{ map<string,
     string> symbol_to_file = 1; string src_hash = 2; }`) under
     `compiler/module/v1/` or similar, marshaled with `proto.Marshal` — matches
     the project's "proto-defined" convention used everywhere else (CLAUDE.md's
     documented preference) rather than introducing a bespoke `gob`/JSON format.
     A planning-time decision, not settled here.

## Environment Availability

Skipped — this phase is pure in-repo Go code (existing dependencies only,
verified in Standard Stack) with zero new environment/tool/service
dependencies. `go build`/`go test -race` are already required and already
used in CI (`.github/workflows/go.yml`).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` + `github.com/stretchr/testify` (project-wide convention) |
| Config file | none — `go test ./...` |
| Quick run command | `go test ./compiler/... ./utils/... -race -run "TestSymbolIndex\|TestScan\|TestTypeResolver\|TestCONS05"` (naming TBD at plan time) |
| Full suite command | `go test -race -coverprofile=coverage.txt -covermode=atomic ./...` (matches CI's existing step) |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| TYPE-01 | Index maps a nested-type symbol (2+ levels deep) to its declaring file | unit | New test with a fixture proto carrying `Outer.Middle.Inner` | ❌ Wave 0 |
| TYPE-02 | Index build cost is proportional to parse-without-link, not parse+link | benchmark/timing | Extend `startup_bench_test.go`-style measurement, or a direct `time.Since` assertion against a generated corpus | ❌ Wave 0 |
| TYPE-03 | A nested `Any` inside a materialized JSON config (not just a compile-time `load()`) resolves via `ReadConfig` | integration | New fixture: hand-write or generate a `.materialized_JSON` with a doubly-nested `Any`, call `parser.ReadConfig`, assert correct unmarshal | ❌ Wave 0 — existing `field_type_any_test.pconf` covers the compile-time path only, not `ReadConfig` |
| TYPE-04 | Building the index links zero files (only resolved-symbol files get linked) | unit | Assert `FileRegistry`'s size (or a new "files linked by index resolution" counter) stays 0/small immediately after an index build that finds no matching lookup | ❌ Wave 0 |
| TYPE-05 | A second compile against unchanged `src/` reuses the cached index (no rebuild) | unit | D-04's exported build counter: compile twice, assert counter is 1 not 2 | ❌ Wave 0 |
| TYPE-06 | Editing any `.proto` under `src/` invalidates the cache | unit | Build index, touch/edit a file, rebuild, assert the content key changed and a fresh build occurred | ❌ Wave 0 |
| TYPE-07 | A `google.protobuf.Value`-only mutable compile never triggers index construction | unit (criterion 3) | D-04's counter asserted == 0 after such a compile | ❌ Wave 0 — the exact criterion-3 test named in D-04 |
| TYPE-08 | grep for type-URL resolution logic finds one implementation (compiler-scoped; full grep-clean state is Phase 14) | manual/structural | `grep -rn "FindMessageTypeByUrl\|FindMessageByURL" --include=*.go` and confirm the compiler's callers all route through `RegistryTypeResolver`/`TypeResolver` | ❌ Wave 0 — a structural check, not a unit test; document as a plan-check item |
| TYPE-09 | `ReadConfig` resolves nested `@type` at any depth | integration | Same fixture as TYPE-03 | ❌ Wave 0 (shared with TYPE-03) |
| CONS-05 | Loading a mutable config resolves its type and every nested `Any` correctly | integration | New test exercising `loadMutable` end-to-end with a nested-`Any`-carrying mutable value whose declaring file is NOT already loaded via any other `load()` in the same compile | ❌ Wave 0 — this is the test that would have caught the `:187` direct-`MessageRegistry` bypass (Pitfall 4) |

### Sampling Rate
- **Per task commit:** `go test ./compiler/... ./utils/... -race`
- **Per wave merge:** `go test -race ./...` (matches CI)
- **Phase gate:** Full suite green under `-race` before `/gsd-verify-work` — this
  phase touches lock-guarded shared state (`d.mu`) in two new code paths, so
  skipping `-race` at the gate defeats its own concurrency-contract discretion
  item

### Wave 0 Gaps
- [ ] A doubly-nested-symbol fixture proto (`Outer.Middle.Inner`) for TYPE-01's
  index-correctness test — does not exist today
- [ ] A materialized-JSON-with-nested-`Any` fixture for TYPE-03/TYPE-09/CONS-05
  — `field_type_any_test.pconf` exists but only exercises the compile-time
  `load()` path, not `ReadConfig`/`loadMutable`
- [ ] A replacement (not deletion) for `eager_fallback_visible_test.go`'s
  operator-observability contract (Pitfall 1) — the new tiers need an
  equivalent "can an operator see what happened" test
- [ ] A lock-discipline race test for the new scan/index tiers, mirroring
  `utils/parse_all_deadlock_test.go`'s shape but racing the new index build
  against `ParseOne` instead of `ParseAll` against `ParseOne`

## Security Domain

`security_enforcement` is not set to `false` in `.planning/config.json`
`[VERIFIED: .planning/config.json — no security_enforcement key present, read
this session]`, so this section is included per protocol. As with Phase 11,
minimal new surface: no new network input, no new auth surface. The one
addition relative to Phase 11's assessment is a **new on-disk cache artifact**
(TYPE-05) under `.protoconf_cache`, parallel to the existing `.fds` files.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | No auth surface touched |
| V3 Session Management | No | N/A |
| V4 Access Control | No | N/A |
| V5 Input Validation | Marginal | The index cache is read from local disk (`.protoconf_cache`) and must be validated before use — mirror `registry.Load`'s existing checksum-gate pattern (refuse and rebuild on mismatch, never trust an unchecked deserialize), same control class as the existing `.fds` cache |
| V6 Cryptography | No | The existing `.fds` checksum uses MD5 as a content-identity key, not a security boundary (pre-existing, unchanged); the new index cache should follow the same non-cryptographic content-identity convention — `dirhash.HashDir`'s default hash (SHA-256-based `h1:` format, per its use in `module_service.go`) is already stronger than the MD5 idiom and is fine either way, since neither is defending against a malicious cache file, only against a stale one |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| A crafted/corrupted `.protoconf_cache` index file causing a panic or silently wrong resolution | Tampering / Denial of Service | Validate-before-trust (checksum gate, Pattern 2) — an invalid or mismatched cache must trigger a rebuild, never a partial/unsafe load, matching `registry.Load`'s existing refuse-on-mismatch behavior |
| Unbounded index growth from an attacker-controlled `src/` tree | Denial of Service | Out of scope, same reasoning as Phase 11's assessment — `src/` is project-controlled, not user/network input, in every current deployment shape |

## Sources

### Primary (HIGH confidence — read this session)
- `compiler/lib/parser/parser.go` (full file) — `RegistryTypeResolver`'s two lookup methods, the exact D-02 replacement targets, `ReadConfig`
- `utils/utils.go` (full file) — `DescriptorRegistry`, `ParseOne`, `ParseAll`, `recordFileLocked`, `Load`/`Store`/checksum idiom, `globalRegexMatcher`
- `compiler/lib/module_service.go` (full file) — `GetProtoRegistry()`'s lazy/eager branch, `dirhash.HashDir` usage at `:273`, `.fds` cache shape
- `compiler/lib/starlark_loader.go` (full file) — `loadMutable`'s three lookups, the `:187` CONS-05 divergence
- `compiler/lib/config.go` (full file) — `validate`'s silent `ErrUnexpectedType` skip (deferred item, confirmed still present)
- `pb/protoconf/v1/protoconf.proto` (full file) — `ProtoconfValue.proto_file` field, quoted verbatim (lines 11-13)
- `compiler/lib/eager_fallback_visible_test.go`, `utils/parse_all_deadlock_test.go`, `utils/growable_resolver_test.go` — the three tests D-02 breaks (Pitfall 1)
- `utils/testdata/corpus.go` (grep + partial read) — `GenerateCorpus`'s nested-message shape, `Nested%d`
- `utils/testdata/small/src/field_type_any_test.pconf` — existing nested-`Any` fixture, confirmed compile-time-only
- `.planning/PROJECT.md`, `.planning/REQUIREMENTS.md`, `.planning/STATE.md`, `.planning/config.json` — project decision records
- `.planning/phases/13-.../13-CONTEXT.md` — this phase's locked decisions (D-01..D-05, discretion items, deferred ideas)
- `.planning/phases/11-.../11-RESEARCH.md` — prior-phase grounding, six-consumer blast radius, lock-discipline precedent
- `.planning/research/compiler-performance/PITFALLS.md`, `OPTIONS.md` — superseded-but-evidence-bearing prior analysis, read for the nested-`Any` measurement and rejected-eager-index cost table
- `$GOMODCACHE/github.com/jhump/protoreflect@v1.16.0/desc/protoparse/parser.go` (lines 215-260) — `ParseFilesButDoNotLink` signature and doc comment, quoted
- `$GOMODCACHE/github.com/jhump/protoreflect@v1.16.0/dynamic/msgregistry/message_registry.go` (lines 149-178) — `AddFile`/`addMessageTypesLocked` nested-type recursion, confirmed by function-signature grep
- `$GOMODCACHE/google.golang.org/protobuf@v1.36.12/types/descriptorpb/descriptor.pb.go` (lines 1437-1619) — `FileDescriptorProto`/`DescriptorProto` field names, quoted
- `$GOMODCACHE/google.golang.org/protobuf@v1.36.12/encoding/protojson/well_known_types.go` (lines 169-220) — `unmarshalAny`'s recursive `Resolver` consultation, quoted and read in full
- `$GOMODCACHE/google.golang.org/protobuf@v1.36.12/encoding/protojson/decode.go` (grep) — confirmed `Resolver` interface shape and default (`protoregistry.GlobalTypes`)
- `go.mod` — all dependency versions cited above, verified directly (noting `google.golang.org/protobuf` is now v1.36.12, not the v1.34.1 recorded in Phase 11's research)
- `../protoconf-terraform/example/src/` — confirmed present, 799 `.proto` files (matches CONTEXT.md's corpus size claim), not independently re-timed this session

### Secondary (MEDIUM confidence)
- `13-CONTEXT.md`'s measured timing/count figures (~11ms scan, 1,286ms unlinked parse, 4,054/126,530 symbol counts, candidate-count examples) — locked decisions from this milestone's discuss-phase session, corpus existence confirmed but timings not independently re-run this session

### Tertiary (LOW confidence)
- None — every claim traces to a file read this session, the locked `13-CONTEXT.md`, or the already-committed `.planning/research/compiler-performance/` bundle (itself HIGH-confidence per Phase 11's research).

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependencies; every primitive (`ParseFilesButDoNotLink`, `dirhash.HashDir`, `singleflight`, `msgregistry.AddFile`'s nested recursion, `protojson`'s recursive `Any` resolution) verified directly against source in `$GOMODCACHE` this session
- Architecture: HIGH — the insertion points (`RegistryTypeResolver`'s two methods, `loadMutable`'s three lookups) were read this session with line numbers; the two open questions (index wire format, `ValidateUnlinkedFiles` setting) are genuinely judgment calls, flagged explicitly rather than asserted as settled
- Pitfalls: HIGH — all six trace to source reads this session, including two (the three tests D-02 breaks, and the `loadMutable:187` CONS-05 bypass) that are concrete, line-numbered findings not present in `13-CONTEXT.md` or the prior milestone research bundle

**Research date:** 2026-09-08
**Valid until:** Stable until the plan actually spikes `ParseFilesButDoNotLink`'s
filename-resolution contract (Pitfall 2/Open Question 1) — recommend treating
that spike as the plan's first task rather than a research gap, since it is a
five-minute check against a real API, not a design question.
