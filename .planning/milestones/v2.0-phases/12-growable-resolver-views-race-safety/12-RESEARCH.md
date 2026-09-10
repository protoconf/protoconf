# Phase 12: Growable Resolver Views & Race Safety - Research

**Researched:** 2026-09-07
**Domain:** Go protobuf descriptor-registry growth (`google.golang.org/protobuf/reflect/protoregistry`), concurrency-safe incremental registration, `jhump/protoreflect` descriptor bridging
**Confidence:** HIGH

## Summary

This phase closes the gap Phase 11 deliberately left open. Phase 11 made `DescriptorRegistry.FileRegistry`
grow on demand (`ParseOne`, singleflight-guarded, `utils/utils.go:236-326`), but `Parser.FilesResolver`
(`*protoregistry.Files`) and the seed half of `Parser.TypeResolver` are still built **once**, at
`NewParserWithDescriptorRegistry` construction time (`compiler/lib/parser/parser.go:36-46`), from
whatever `FileRegistry` held at that instant — for a lazy (`ImportPaths`-set) registry, that's only the
~65 well-known types `NewDescriptorRegistry()` seeds (`utils/utils.go:73-95`), never anything `ParseOne`
loads afterward. `PITFALLS.md` (compiler-performance research, item 3) named this exactly: "A snapshot
plus a lazy source is a stale-cache bug waiting to happen." `[VERIFIED: .planning/research/compiler-performance/PITFALLS.md:160-174, read this session]`

**What "the 260ms rebuild" actually is, verified against current code:** `BASELINE.md` measured 260ms
for `parser.NewParserWithDescriptorRegistry` building resolvers over 864 *already eagerly loaded* files
(`BASELINE.md:24`). Phase 11 already collapsed that specific number, because the lazy registry is
near-empty at construction — `GetFilesResolver()`'s one-shot `protodesc.NewFiles` call today runs over
~65 entries, not 864. **The 260ms-shaped cost this phase actually guards against is a regression, not a
live measurement**: the naive way to fix `RSLV-01`'s staleness is "call `GetFilesResolver()` again
whenever the resolver might be stale" — and `GetFilesResolver()` rebuilds a `FileDescriptorSet` from
**every entry currently in `FileRegistry`** (`utils/utils.go:116-133`) every time it runs. Doing that
once per demanded proto reintroduces an O(total-loaded-so-far) cost on every miss — the same shape
`OPTIONS.md` explicitly rejected ("a lazy registry with an eager resolver snapshot would be a
stale-cache bug the first time a new file is parsed," `OPTIONS.md:41-43`). The fix is **true incremental
registration** (`(*protoregistry.Files).RegisterFile`, one call per newly-parsed file, O(that file's own
top-level symbols) — never a rebuild of the whole set) `[VERIFIED: $GOMODCACHE/google.golang.org/protobuf@v1.36.12/reflect/protoregistry/registry.go:114-176, read this session — RegisterFile appends to filesByPath/descsByName without touching prior entries]`.

**The mechanism has a natural, already-shipped insertion point.** `recordFileLocked`
(`utils/utils.go:333-343`) is the single writer for `FileRegistry`/`lazyLoaded`, called under `d.mu`
write lock, already idempotent (skips if `fd.GetName()` is present) and already recursive over
`fd.GetDependencies()`. Every `*desc.FileDescriptor` reaching this function carries a fully-linked
underlying `protoreflect.FileDescriptor`, retrievable via `fd.UnwrapFile()`
`[VERIFIED: $GOMODCACHE/github.com/jhump/protoreflect@v1.16.0/desc/descriptor.go:74-77, quoted: "// UnwrapFile returns the underlying protoreflect.FileDescriptor.\nfunc (fd *FileDescriptor) UnwrapFile() protoreflect.FileDescriptor {\n\treturn fd.wrapped\n}"]`.
Growing `FilesResolver` is therefore one extra line inside `recordFileLocked`'s "not yet present" branch:
`filesResolver.RegisterFile(fd.UnwrapFile())` — reusing the exact skip-if-present guard that already
prevents `FileRegistry`/`FilesResolver` from ever diverging (D-02's explicit requirement). The D-03
`ParseAll` eager-fallback path (`utils/utils.go:352-373`) already computes a `before`/`after` diff
against `FileRegistry` to drive `lazyLoaded` — the identical diff, keyed identically, is what makes its
files register through "the same path as `ParseOne`'s files" (D-02) without a second insertion point.

**The one real design decision this research resolves with evidence, not guesswork, is locking.**
`protoregistry.Files`'s doc comment says "The Find and Range methods are safe for concurrent use"
`[VERIFIED: $GOMODCACHE/google.golang.org/protobuf@v1.36.12/reflect/protoregistry/registry.go:82-84]` —
but reading `RegisterFile`'s body shows every lock (`globalMutex`) is gated on `r == GlobalFiles`; a
locally-constructed `*protoregistry.Files` (exactly what `GetFilesResolver()` builds) has **zero**
internal synchronization for any method, `RegisterFile` included
`[VERIFIED: $GOMODCACHE/google.golang.org/protobuf@v1.36.12/reflect/protoregistry/registry.go:114-118, 315-323, 350-357 — the `if r == GlobalFiles { ... }` guard is the only lock in each method]`.
The doc's "safe for concurrent use" claim is true only among concurrent *reads* of an object nobody is
mutating, or on `GlobalFiles` itself — for a growable local instance under concurrent compiles, every
`RegisterFile`/`FindFileByPath`/`RangeFiles` call must go through the same external lock. `DescriptorRegistry.mu`
already exists and already serializes every other mutation to this registry (`FileRegistry`, `lazyLoaded`,
`eagerFallback`) — the growable `FilesResolver` should live under that same lock, not a second one.

Grepping every direct reader of `Parser.FilesResolver` in the tree turns up exactly two call sites
outside `utils`/`parser` construction code: `server/server.go:292-297,426,433` (six `RegisterFile` calls
plus two grpcui `DescriptorResolver` assignments) and `compiler/lib/parser/parser.go:123` (`ParseFilesX`'s
second lookup branch) `[VERIFIED: grep -rn "FilesResolver" --include="*.go", read this session]`.
`server.go`'s parser is always built from `ms.GetProtoRegistry()` — the non-lazy path, `ImportPaths`
never set (D-03's gate, `module_service.go:450-461`) — so its `FilesResolver` never grows and stays safe
to read unlocked, exactly as it does today; `server/init_prohibitions_test.go:123-144`
(`TestDiscoveryScanDoesNotBackReflection`) already encodes this as a regression test and constructs an
eager, `ImportPaths`-empty registry to prove it. **The only unlocked reader of a *growable* `FilesResolver`
anywhere in the tree is `ParseFilesX` itself — a call site fully inside this phase's control.** That is
the evidence the "keep the field type" discretion item asks for: the field can stay `*protoregistry.Files`
unchanged (server.go and the grpcui wiring compile untouched) as long as `ParseFilesX`'s own line 123 is
rewritten to go through a new locked accessor on `DescriptorRegistry` (mirroring the existing
`FileDescriptor(name)` method, `utils/utils.go:106-114`) instead of touching `p.FilesResolver` directly.

**Primary recommendation:** Add a `filesResolver *protoregistry.Files` field to `DescriptorRegistry`
(built once via the existing `GetFilesResolver()` machinery, cached on the registry rather than on
`Parser`), grow it inside `recordFileLocked`'s already-idempotent, already-`d.mu`-guarded insert point
and inside `ParseAll`'s existing before/after diff, and add a `FindFileByPath`/`RangeFiles`-shaped
locked accessor for `ParseFilesX` to call instead of reading `Parser.FilesResolver` directly. Gate all of
it on `len(d.ImportPaths) > 0` (D-03's existing boundary — free, since `ParseOne` and `ParseAll` already
require it). Leave `LocalResolver`/`RegistryTypeResolver` untouched (D-01). Fix the `ParseFilesX`
non-canonical-pointer branch (`desc.WrapFile`/`desc.CreateFileDescriptor` on a `FilesResolver` hit) by
routing it through the registry's canonical `FileRegistry` lookup instead of minting a second descriptor —
this branch is dormant today only because `FilesResolver` is small and frozen; growing it makes the
branch fire for real. Delete `config.protoResolver` (confirmed zero readers) rather than wire it to a
growing view.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Growth Mechanism**

- **D-01: Hybrid.** `FilesResolver` grows by real incremental registration into the underlying
  `*protoregistry.Files`. `LocalResolver` does **not** — it stays the Phase 11 miss-fallthrough wrapper
  (`RegistryTypeResolver`, `compiler/lib/parser/parser.go:57`), which already reads the immutable
  construction-time snapshot, then the growable `MessageRegistry`, then fires the D-03 `ParseAll`
  fallback. Rationale: enumeration (`RangeFiles`) is what file resolution and downstream
  reflection/grpcui actually need, and only `Files` provides it; types are already served correctly by a
  mechanism that mutates nothing. Two mechanisms, each the cheapest one that serves its consumer. —
  **Reversibility:** costly — collapsing to one mechanism later means re-touching every `Parser` field
  consumer in `server/`, `inserter/`, `mutate/`, and `agent/filekv/`.

- **D-02: Uniform insert point.** Files reaching the registry via the D-03 `ParseAll` eager fallback
  (`utils/utils.go:352`) register into `FilesResolver` through the same path as `ParseOne`'s files — no
  special case. `FileRegistry` and `FilesResolver` must never hold divergent sets. Accepted cost: when the
  fallback fires it now also pays a whole-tree registration. That cliff is already known, is what Phase
  13's symbol index exists to remove, and a set divergence would be a worse invariant to carry forward
  than a slow path that is already slow.

**Blast Radius**

- **D-03: Compiler only.** Growth is gated on `registry.ImportPaths` being non-empty — the *existing*
  D-01 boundary from Phase 11, not a new one. This was verified this session: `ImportPaths` is written in
  exactly one place (`compiler/lib/module_service.go:454`), under `m.lazyRegistry`, which only
  `NewLazyModuleService` sets, which only `NewCompiler` calls. The other five consumers
  (`server/server.go:291`, `inserter/inserter.go:220`, `mutate/mutate.go:73`,
  `agent/filekv/filekv.go:80`, and `mod sync`'s own eager registry) therefore keep byte-identical
  behavior, and Phase 14 *verifies* rather than *undoes*.

- **D-04:** No new constructor and no new flag for the opt-in. Deriving it from `ImportPaths` means
  "compiler only" falls out of an invariant that already exists and is already tested, rather than a
  second one to keep in sync. Rejected: `NewGrowableParserWithDescriptorRegistry` / a `With...` option —
  explicit at the call site, but a duplicate encoding of the same condition.

**RSLV-02 Observable**

- **D-05: Registration counter on `DescriptorRegistry`**, exported as a method alongside the existing
  `LoadedFileCount()` (`utils/utils.go:378`). A test loads N protos one at a time and asserts total
  registrations grow by each file's own closure, not by N×(files-so-far). This measures "no rebuild" as
  a number rather than inferring it. Rejected: resolver-object-identity assertion (proves no rebuild,
  says nothing about per-registration work) and a timing assertion folded into
  `TestCompilerStartupScaling` (flakiest gate shape; cannot distinguish a resolver rebuild from any other
  regression).

- **D-06:** The counter is **test-only** — an exported method, no log line and no CLI surface. Operators
  already get the loaded-file count from LAZY-05; a second number in normal compile output is noise for
  anyone not debugging resolver churn.

### Claude's Discretion

The user explicitly deferred these. Decide them on evidence from the code, not by asking again.

- **Locking / API shape for the growable `FilesResolver`.** `protoregistry.Files` is documented as
  unsafe for concurrent mutation, so growth means readers need synchronising too. Candidates: a
  parser-owned type holding the `*protoregistry.Files` plus an `RWMutex` and exposing
  `FindFileByPath`/`RangeFiles`/`RegisterFile`; or a lock held inside `DescriptorRegistry` under the
  `d.mu` it already owns. **Constraint:** whatever is chosen must be provable under `-race` *without*
  dragging the non-compiler consumers into this phase (D-03).
- **Whether `Parser.FilesResolver`'s exported field type changes.** Keeping it as `*protoregistry.Files`
  means `server/server.go:292-297` (six hand-registered well-known files) and `:426`/`:433` (grpcui
  `DescriptorResolver` wiring) compile untouched, but readers holding the field directly are outside any
  lock — so that choice must come with evidence that no such reader exists on a lazy registry. Making it
  a small interface forces readers through the lock but ripples a type change through four packages.
- **Registration granularity** — the requested file only, or its transitive closure. `recordFileLocked`
  (`utils/utils.go:333`) already walks `fd.GetDependencies()` recursively and skips names already
  present, so the closure is available for free at the same insert point. Constraint: whichever is
  chosen must keep `protodesc.NewFile`'s dependency requirement satisfiable without introducing a second
  resolution path.
- **`ParseFilesX` shape** (`compiler/lib/parser/parser.go:118-155`). Trace what the
  `FilesResolver.FindFileByPath` branch actually resolves that the registry does not, then cut or keep
  on that evidence. See Integration Points below for why the branch is a live hazard either way.
- **RSLV-03 test assertions.** Constraint: the test must fail loudly if the canonical-pointer contract
  regresses — a re-parse that mints a second descriptor for A can produce correct materialized JSON, so
  output-only assertions do not guard the criterion.
- **SAFE-01 test placement** — extend `TestConcurrentCompile`
  (`compiler/lib/concurrent_compile_test.go:25`) with resolver-view assertions, or add a dedicated test
  racing `RegisterFile` against `FindFileByPath`. Constraint: a resolver-growth race must fail loudly
  under `go test -race`.
- **The dead `config.protoResolver` field** (see Integration Points) — delete it, or point it at the
  growable view. Constraint: do not leave it holding a frozen snapshot in the struct this phase is
  making grow.

### Deferred Ideas (OUT OF SCOPE)

- **Operator-visible resolver-churn metric** — D-06 keeps the registration counter test-only. If Phase
  15's numbers turn out to want it in compile output, that is a one-line addition then, with real
  motivation behind it.
- **Removing the D-03 `ParseAll` whole-tree fallback cliff** — accepted as a known slow path here (D-02).
  Phase 13's exact symbol index is what removes it.
- **`ModuleService.GetProtoFilesRegistry()` deletion** — zero callers today. If it turns out not to be
  in this phase's path, it is a trivial cleanup for Phase 14 or a `/gsd-quick`.
- **Collapsing the two resolver mechanisms into one** — D-01's hybrid is deliberate for this phase. If
  Phase 13's index makes the `Types` miss-fallthrough redundant, unify then.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| RSLV-01 | `FilesResolver`/`LocalResolver` reflect files parsed after construction, not a snapshot | Summary; Architecture Patterns Pattern 1; Common Pitfalls #1 — exact construction site (`parser.go:36-46`) and the doc-comment/locking gap in `protoregistry.Files` both located and verified |
| RSLV-02 | A newly parsed file is registered incrementally, without rebuilding the full `FileDescriptorSet` | Summary ("what the 260ms rebuild actually is"); Architecture Patterns Pattern 1; Validation Architecture RSLV-02 row — `RegisterFile`'s O(one file) cost verified against upstream source, contrasted with `GetFilesResolver()`'s O(whole-registry) rebuild cost |
| RSLV-03 | `load()` A, then B, then re-reference A resolves all three correctly | Common Pitfalls #2 (the `ParseFilesX` non-canonical-pointer branch); Architecture Patterns Pattern 2; Validation Architecture RSLV-03 row |
| SAFE-01 | Concurrent compiles against one shared compiler are race-free under `go test -race` | Common Pitfalls #1 (locking gap) and #3; Architecture Patterns Pattern 1; Validation Architecture SAFE-01 row — extends the already-shipped `TestConcurrentCompile` (`concurrent_compile_test.go:25`) shape, which already races N goroutines against one shared `*lib.Compiler` |

</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Growable file-descriptor resolver (`FilesResolver`) | Backend / Library (`utils.DescriptorRegistry`) | — | Pure in-process registry state; no network, no client tier |
| Lock discipline over the growable resolver | Backend / Library (`utils.DescriptorRegistry`, `d.mu`) | — | Same lock that already guards `FileRegistry`/`lazyLoaded` — one owner, not a second lock object |
| `ParseFilesX`'s file-lookup dispatch | Backend / Library (`compiler/lib/parser`) | — | In-process dispatch between "already resolved," "in the growable set," and "not yet parsed" |
| gRPC reflection / grpcui descriptor browsing | Backend / API (`server/server.go` `Init()`) | — | Reads `s.parser.FilesResolver` directly; stays on the always-eager, never-growing path per D-03, unaffected by this phase |
| Concurrent-compile safety | Backend / Library | — | All concurrency in scope is in-process goroutines sharing one `*lib.Compiler`/`*utils.DescriptorRegistry`, no cross-process concern |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `google.golang.org/protobuf` | v1.36.12 (pinned in `go.mod:48`) `[VERIFIED: go.mod:48, and confirmed present at $GOMODCACHE/google.golang.org/protobuf@v1.36.12]` | `protoregistry.Files.RegisterFile`/`FindFileByPath`/`RangeFiles` — the incremental-registration API this phase is built on | Already the project's resolver-layer dependency (`utils/utils.go:126-133`). **Correction to Phase 11's own RESEARCH.md**, which cited `v1.34.1` — that was stale even at the time Phase 11 landed; `go.mod` pins `v1.36.12` today. `RegisterFile`'s semantics are unchanged between the two versions (confirmed by diffing both installed copies' `registry.go`) |
| `github.com/jhump/protoreflect` | v1.16.0 (`go.mod:19`) `[VERIFIED: go.mod:19]` | `desc.FileDescriptor.UnwrapFile()` — the bridge from the parse engine's descriptor type to the `protoreflect.FileDescriptor` that `RegisterFile` requires | Already the project's only proto-parsing dependency; unchanged since Phase 11 |
| `golang.org/x/sync` | v0.22.0 (`go.mod:45`) | No new use in this phase — `singleflight.Group` (already used by `ParseOne`) needs no change; growth hangs off `recordFileLocked`, which already runs inside `ParseOne`'s critical section | Confirms no new concurrency primitive is needed for this phase; the existing `d.mu`/`singleflight` combination already serializes every path that needs to write |

**No new external package is required for this phase.** The only new surface is a field and a handful
of methods on the existing `DescriptorRegistry`/`Parser` types, plus one stdlib call
(`(*protoregistry.Files).RegisterFile`) already imported transitively.

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Hanging the growable `*protoregistry.Files` off `DescriptorRegistry`, guarded by the existing `d.mu` | A parser-owned type wrapping `*protoregistry.Files` + its own `RWMutex` | The parser-owned wrapper is what the CONTEXT's discretion section lists as a candidate, and it is viable — but `DescriptorRegistry` is already this codebase's established single source of truth (11-PATTERNS.md: "`DescriptorRegistry` is the single source of truth, `d.mu` is its lock"), and `recordFileLocked` already runs under `d.mu` on every write path (`ParseOne`, `ParseAll`). Reusing that lock means zero new lock-ordering surface to reason about; a second lock on `Parser` would need its own discipline for "never hold the Parser lock while calling into the registry, which holds its own lock" — a self-inflicted lock-ordering risk this phase does not need to take on |
| Rewriting `ParseFilesX`'s direct `p.FilesResolver.FindFileByPath` read to go through a new locked accessor | Changing `Parser.FilesResolver`'s exported type to an interface that forces every reader through a lock | The interface change ripples through `server/server.go` (6 call sites) and grpcui's `reflection.ServerOptions.DescriptorResolver`/`ExtensionResolver` fields, which expect the concrete `protoregistry.MessageResolver`-shaped types the stdlib package defines — verified this session (grep found exactly 2 non-construction readers of `Parser.FilesResolver` in the whole tree: `server.go` and `parser.go:123`). Since `server.go`'s registry never grows (D-03), only one call site actually needs the lock; forcing all readers through an interface fixes zero additional races at 4x the blast radius |
| Registering only the requested file into the growable resolver | Registering the requested file's whole transitive closure (mirrors `recordFileLocked`'s existing `FileRegistry`/`lazyLoaded` recursion) | Registering only the top-level file would let `FileRegistry` and `FilesResolver` diverge the moment a transitively-loaded dependency is later referenced independently (e.g., B imports A, then a later, unrelated `load()` names A directly) — exactly the divergence D-02 forbids. The closure is already computed for free by the existing recursion; skipping it to register less would need a *second* traversal to avoid registering it twice, which is more code for a worse invariant |

**Installation:** none — no new packages.

**Version verification:** `google.golang.org/protobuf v1.36.12` and `github.com/jhump/protoreflect v1.16.0`
confirmed present in `go.mod` and `$GOMODCACHE` this session. No registry lookup needed since nothing new
is installed.

## Package Legitimacy Audit

Not applicable — this phase introduces zero new external packages. All work is additive methods/fields
on `DescriptorRegistry`/`Parser` plus calls into `google.golang.org/protobuf/reflect/protoregistry`,
already a direct dependency.

## Architecture Patterns

### System Architecture Diagram

```
  NewCompiler() ──► ms.GetProtoRegistry() ──► *utils.DescriptorRegistry
        │             (lazy: ImportPaths=[srcDir], D-03 gate ON)
        │
        ▼
  parser.NewParserWithDescriptorRegistry(registry)
        │
        │   registry.GetFilesResolver() ── builds ONCE (cheap: near-empty
        │        seed, ~65 well-known types) ──► registry caches the
        │        resulting *protoregistry.Files as its own field
        │        (filesResolver), NOT rebuilt again — RSLV-02's target
        ▼
  Compiler{parser: *parser.Parser}  (SHARED across concurrent CompileFile calls)
        │
        ▼
  CompileFile(f) ──► loader.loadProto("A") ──► parser.ParseFilesX("A")
        │                                          │
        │                    ┌─────────────────────┴─────────────────────┐
        │                    │ 1. registry.FileDescriptor("A") (d.mu.RLock)│
        │                    │    HIT  → canonical *desc.FileDescriptor   │
        │                    │    MISS ↓                                  │
        │                    │ 2. registry.FindFileByPath("A") (locked,   │◄── NEW: locked accessor,
        │                    │    NEW method, reads the SAME growable      │    replaces today's direct
        │                    │    filesResolver under d.mu.RLock)          │    p.FilesResolver read
        │                    │    HIT  → route back through FileRegistry   │    (RSLV-03 fix — never
        │                    │           lookup, never re-wrap (RSLV-03)   │    mint a second descriptor)
        │                    │    MISS ↓                                  │
        │                    │ 3. registry.ParseOne("A") (singleflight)   │
        │                    │      parses A, recordFileLocked(A) under    │
        │                    │      d.mu.Lock():                           │
        │                    │        FileRegistry["A"] = A                │
        │                    │        lazyLoaded["A"] = struct{}{}         │
        │                    │        filesResolver.RegisterFile(          │◄── NEW: incremental growth,
        │                    │          A.UnwrapFile())                    │    O(A's own symbols), not
        │                    │        MessageRegistry.AddFile(A)           │    O(everything loaded so far)
        │                    │        recurse into A.GetDependencies()     │
        │                    └─────────────────────────────────────────────┘
        ▼
  loader.loadProto("B") ──► same path, B's own deps (possibly including A,
        │                    already in FileRegistry — recordFileLocked's
        │                    existing "already present" skip fires, so A is
        │                    NOT registered a second time)
        ▼
  loader.loadProto("A") again (RSLV-03: "re-reference A")
        │                    → branch 1 HITS (registry.FileDescriptor("A")),
        │                      returns the SAME canonical pointer as before —
        │                      branch 2's WrapFile/CreateFileDescriptor never
        │                      runs for this path, so no second descriptor
        │                      for A is ever minted
        ▼
  writeConfig()/c.parser.TypeResolver.FindMessageByURL(...)
        (RegistryTypeResolver, unchanged from Phase 11 — D-01)


  ─── SAFE-01: two concurrent CompileFile calls on the same *lib.Compiler ───

  goroutine 1: loadProto("pkgK/msgK.proto")  ─┐
                                                ├─► both funnel through the SAME
  goroutine 2: loadProto("pkgJ/msgJ.proto")  ─┘   registry.mu / filesResolver;
                                                    RegisterFile calls for K and J
                                                    interleave under d.mu.Lock(),
                                                    never concurrently with each
                                                    other or with a FindFileByPath
                                                    read — this is exactly what
                                                    `go test -race` must exercise
```

### Recommended Project Structure

No new packages or directories — this phase edits existing files in place:

```
utils/utils.go                     # DescriptorRegistry: + filesResolver field, + FindFileByPath/
                                    #   RangeFiles locked accessors, growth call in recordFileLocked
                                    #   and in ParseAll's before/after diff, + registration counter (D-05)
compiler/lib/parser/parser.go      # ParseFilesX: branch 2 goes through registry's locked accessor,
                                    #   not p.FilesResolver directly; fix the non-canonical-pointer hazard
compiler/lib/config.go             # delete the dead protoResolver field (see Integration Points)
compiler/lib/compiler.go           # drop the now-dead assignment to config.protoResolver
compiler/lib/concurrent_compile_test.go  # extend or add a dedicated resolver-growth race test (SAFE-01)
```

### Pattern 1: Incremental registration at the existing single-writer insert point

**What:** Grow the same `*protoregistry.Files` object the registry already builds once, by calling
`RegisterFile` inside the two places that already mutate `FileRegistry` under `d.mu` — never by
rebuilding from `FileRegistry`.

**When to use:** Any time a `*protoregistry.Files` needs to reflect state that grows after its own
construction, without paying for previously-registered entries again.

**Example (shape only — verify exact field/method names against the struct read this session,
`utils/utils.go:47-71`):**

```go
// utils/utils.go
type DescriptorRegistry struct {
    MessageRegistry msgregistry.MessageRegistry
    FileRegistry    map[string]*desc.FileDescriptor
    localFiles      map[string]struct{}
    ImportPaths     []string

    mu            sync.RWMutex
    group         singleflight.Group
    lazyLoaded    map[string]struct{}
    eagerFallback bool

    // filesResolver is the growable view RSLV-01 requires. Built once, lazily,
    // on first access via GetFilesResolver's existing protodesc.NewFiles call
    // (cheap: the seed is near-empty for a lazy registry) — then grown
    // in-place by recordFileLocked and ParseAll's diff, never rebuilt.
    // nil for every eager consumer (ImportPaths unset, D-03) — those keep
    // today's GetFilesResolver behavior byte-for-byte.
    filesResolver     *protoregistry.Files
    registrationCount int // D-05: incremented once per successful RegisterFile call
}

// FindFileByPath is the locked accessor ParseFilesX must use instead of
// reading Parser.FilesResolver directly — the one call site in the tree that
// can observe a growing resolver (server.go's is always eager, D-03).
func (d *DescriptorRegistry) FindFileByPath(path string) (protoreflect.FileDescriptor, error) {
    d.mu.RLock()
    defer d.mu.RUnlock()
    if d.filesResolver == nil {
        return nil, protoregistry.NotFound
    }
    return d.filesResolver.FindFileByPath(path)
}

// recordFileLocked (existing, extended) — callers must hold d.mu for writing.
func (d *DescriptorRegistry) recordFileLocked(fd *desc.FileDescriptor) {
    if _, ok := d.FileRegistry[fd.GetName()]; ok {
        return // already present — the exact guard that keeps FileRegistry
                // and filesResolver from ever diverging (D-02)
    }
    d.FileRegistry[fd.GetName()] = fd
    d.lazyLoaded[fd.GetName()] = struct{}{}
    d.MessageRegistry.AddFile("type.googleapis.com", fd)
    if d.filesResolver != nil { // nil for every eager (non-ImportPaths) registry
        if err := d.filesResolver.RegisterFile(fd.UnwrapFile()); err != nil {
            slog.Error("failed to register file in growable resolver", "file", fd.GetName(), "error", err)
        }
        d.registrationCount++
    }
    for _, dep := range fd.GetDependencies() {
        d.recordFileLocked(dep) // dependencies already present are skipped by
                                 // the guard above — no double RegisterFile
    }
}
```

### Pattern 2: Fixing the `ParseFilesX` non-canonical-pointer hazard (RSLV-03)

**What:** Today's second lookup branch in `ParseFilesX` (`compiler/lib/parser/parser.go:123-132`) calls
`p.FilesResolver.FindFileByPath(filename)` and, on a hit, mints a **new** `*desc.FileDescriptor` via
`desc.WrapFile`/`desc.CreateFileDescriptor` — a different Go pointer than whatever `*desc.FileDescriptor`
is already sitting in `registry.FileRegistry` under the same name. Today this branch is nearly
unreachable in practice: `FilesResolver` is frozen at construction and small, so branch 1
(`registry.FileDescriptor(filename)`) already satisfies every request for anything `ParseOne` has
loaded. **Once `FilesResolver` grows to include everything `ParseOne` has loaded, this branch starts
hitting on cases where branch 1 happened to miss** (e.g. a filename-string variant that doesn't exactly
match `fd.GetName()`'s map key) — and it would then hand the caller a second, non-canonical descriptor
for a file the registry already has a canonical entry for.

**How to avoid:** On a hit in the new locked `FindFileByPath` accessor, do not wrap/create a second
descriptor — instead look the file up by its canonical name (`fd.Path()`, the same key
`recordFileLocked` stores under) in `registry.FileRegistry` via the existing `FileDescriptor(name)`
method (`utils/utils.go:106-114`), which is the accessor `ParseOne` itself already trusts for
pointer-identity. If that lookup also misses (a state that should not occur once `FileRegistry` and
`filesResolver` are updated in the same locked call, per D-02), fall back to `ParseOne` rather than
minting a `desc.WrapFile` copy.

**Example:**

```go
// compiler/lib/parser/parser.go — ParseFilesX, branch 2 rewritten
func (p *Parser) ParseFilesX(filenames ...string) (results []*desc.FileDescriptor, err error) {
    for _, filename := range filenames {
        if fd, ok := p.registry.FileDescriptor(filename); ok {
            results = append(results, fd)
            continue
        }
        if resolved, resolverErr := p.registry.FindFileByPath(filename); resolverErr == nil {
            // A hit here means SOMETHING registered this path into the
            // growable resolver — which only ever happens at the same
            // locked insert point that also writes FileRegistry (D-02).
            // Route back through the canonical map instead of wrapping a
            // second descriptor (RSLV-03's canonical-pointer contract).
            if canonical, ok := p.registry.FileDescriptor(resolved.Path()); ok {
                results = append(results, canonical)
                continue
            }
        }
        parsed, parseErr := p.registry.ParseOne(filename)
        if parseErr != nil {
            return nil, parseErr
        }
        results = append(results, parsed)
    }
    return results, nil
}
```

### Anti-Patterns to Avoid

- **Calling `GetFilesResolver()` again anywhere on the compiler's hot path to "refresh" the snapshot.**
  This is the literal 260ms-shaped regression this phase exists to prevent — it rebuilds a
  `FileDescriptorSet` from every entry currently in `FileRegistry` (`utils/utils.go:116-124`) and reruns
  `protodesc.NewFiles` over it, an O(everything-loaded-so-far) cost repeated on every miss. Growth must
  be `RegisterFile`, one call per new file, never a rebuild.
- **A second, independent lock for the growable `FilesResolver`.** `d.mu` already serializes every write
  to this registry's other growable state (`FileRegistry`, `lazyLoaded`, `eagerFallback`); a second lock
  scoped to just the resolver creates a lock-ordering hazard (which one is acquired first when both are
  touched in the same call, as `recordFileLocked` must) for zero benefit — there is no scenario where the
  resolver needs to be written independently of `FileRegistry`.
- **Trusting `protoregistry.Files`'s "Find and Range are safe for concurrent use" doc comment as license
  to skip external locking.** Verified this session: that guarantee is real only for `GlobalFiles`
  (backed by the package-level `globalMutex`) or for a local instance nobody is mutating concurrently. A
  locally-constructed, growing `*protoregistry.Files` has zero internal synchronization in `RegisterFile`,
  `FindFileByPath`, or `RangeFiles` — external locking (here, `d.mu`) is mandatory for all three once
  growth starts.
- **Registering a `ParseAll`-fallback file a second time because its own before/after diff and
  `recordFileLocked`'s presence guard are computed against different maps.** Both must key off
  `FileRegistry` — reusing the exact same diff loop `ParseAll` already runs for `lazyLoaded`
  (`utils/utils.go:358-370`) for `filesResolver` registration too keeps the two invariants (D-05's
  counter and D-02's "never diverge") backed by the same evidence, not two parallel ones that could drift.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Incremental descriptor registration into a resolver readers can `RangeFiles`/`FindFileByPath` over | A custom `map[string]protoreflect.FileDescriptor` plus hand-rolled `Range`/`Find` methods | `(*protoregistry.Files).RegisterFile`/`FindFileByPath`/`RangeFiles` — already what `Parser.FilesResolver` is typed as, already what `server.go` and grpcui's `reflection.ServerOptions` expect | It is already the exact type every consumer expects; a hand-rolled map-based resolver would need its own adapter to satisfy `reflection.ServerOptions.DescriptorResolver`'s interface, duplicating what `protoregistry.Files` already does correctly |
| Detecting "did resolver growth just redo work for files already registered" | A parallel bookkeeping structure separate from `FileRegistry` | The existing before/after diff pattern `ParseAll` already uses for `lazyLoaded` (`utils/utils.go:358-370`), reused for the D-05 registration counter | One diff, one loop, backs two invariants (no double-count, no divergence) instead of two separate mechanisms that could disagree |
| Serializing concurrent `ParseOne`/`RegisterFile` calls for different paths | A new lock type, or `sync.Map` for the resolver | `DescriptorRegistry.mu` (already exists, already serializes every other write) | `recordFileLocked` already runs under `d.mu.Lock()` for every write path; adding the resolver write to the same critical section costs nothing new and introduces no new lock-ordering surface |

**Key insight:** Every mechanism this phase needs — incremental registration, growth-diff bookkeeping,
and single-writer locking — is either a stdlib/dependency API already imported (`protoregistry.Files`)
or an existing Phase-11 pattern (`recordFileLocked`'s idempotent insert, `ParseAll`'s before/after diff,
`d.mu`'s write discipline) being reused at one more call site each. Nothing here is new machinery.

## Common Pitfalls

### Pitfall 1: Trusting the doc comment instead of reading `RegisterFile`'s body

**What goes wrong:** `protoregistry.Files`'s doc comment reads "The Find and Range methods are safe for
concurrent use" `[VERIFIED: $GOMODCACHE/google.golang.org/protobuf@v1.36.12/reflect/protoregistry/registry.go:82-84]`.
A plan that stops at this sentence could conclude no external lock is needed around reads, or — worse —
that `RegisterFile` is therefore also implicitly safe.

**Why it happens:** The doc comment is accurate but incomplete for this use case: it is true for
`GlobalFiles` (every method gated by the package-level `globalMutex`) and true for any `*Files` instance
that nothing is concurrently *writing* to. It says nothing about `RegisterFile`'s own safety, and reading
the function body shows `RegisterFile`'s only lock is the same `r == GlobalFiles` gate
`[VERIFIED: $GOMODCACHE/google.golang.org/protobuf@v1.36.12/reflect/protoregistry/registry.go:114-118]`
— a locally-constructed instance (exactly what `GetFilesResolver()` returns) gets none of it.

**How to avoid:** Every `RegisterFile`/`FindFileByPath`/`RangeFiles` call against the growable local
`*protoregistry.Files` must go through `d.mu` (write lock for `RegisterFile`, read lock for the other
two) — no exceptions, regardless of what the doc comment seems to promise.

**Warning signs:** A plan or review that cites "the docs say Find/Range are concurrent-safe" as
justification for an unlocked read of the growable resolver specifically (not the eager, never-mutated
one `server.go` holds) has missed this distinction.

### Pitfall 2: `ParseFilesX`'s dormant non-canonical-pointer branch waking up

**What goes wrong:** `compiler/lib/parser/parser.go:123-132`'s current second-branch logic, on a
`FilesResolver.FindFileByPath` hit, calls `desc.WrapFile` (falling back to `desc.CreateFileDescriptor`)
— minting a `*desc.FileDescriptor` that is a different Go object than whatever the registry's
`FileRegistry` map already holds for that same logical file. Today this branch essentially never fires
for anything `ParseOne` loaded, because `FilesResolver` is small and frozen and branch 1
(`registry.FileDescriptor`) already covers it. **Growing `FilesResolver` makes this branch reachable for
real** — any request whose literal filename string doesn't happen to match `FileRegistry`'s key
(computed from `fd.GetName()`) but does match something already registered in the growable resolver will
now hit this path and get a second, non-canonical descriptor for a file the registry already has a
canonical entry for.

**Why it happens:** The branch was originally written for a world where `FilesResolver` held everything
(`GetProtoRegistry()` eager, pre-Phase-11) and `registry.FileDescriptor`'s equivalent lookup would have
hit first in virtually every real case — the `WrapFile` path existed for edge cases involving files
outside the registry's own construction-time `FileRegistry` (e.g. compiled-in well-known types reachable
only via `protoregistry.GlobalFiles`-derived sources), not as a routine second chance for the registry's
own growing set.

**How to avoid:** Pattern 2 above — on a resolver hit, route back through `registry.FileDescriptor` by
the resolved file's own canonical path before falling back to minting a new descriptor. A test asserting
`require.Same(t, fdFromFirstLoad, fdFromReReference)` (pointer identity, not just equal JSON output) is
the only assertion shape that actually guards this — RSLV-03's own text says as much: "a re-parse that
mints a second descriptor for A can produce correct materialized JSON, so output-only assertions do not
guard the criterion."

**Warning signs:** A plan whose RSLV-03 test only checks that the materialized JSON for a config
referencing A twice looks correct, without a `require.Same`/pointer-identity assertion on the descriptor
itself, has not actually tested this pitfall — `desc.WrapFile`'s output can be semantically equivalent to
the canonical descriptor even though it is a different object, so an output-only test can pass while the
contract is broken.

### Pitfall 3: The `ParseAll` fallback silently diverging `FileRegistry` from `filesResolver`

**What goes wrong:** `ParseAll` (`utils/utils.go:352-373`) already computes a `before`/`after` diff
against `FileRegistry` to populate `lazyLoaded` when the D-03 whole-tree eager fallback fires. If growth
into `filesResolver` is implemented as a *separate* pass (e.g., iterating `d.FileRegistry` fresh after
`Import` returns, rather than reusing the identical diff), a bug in the second pass's dedup logic could
register a file into `filesResolver` that was already registered by an earlier `ParseOne` call —
`RegisterFile` returns an error on exact-path duplicate registration
`[VERIFIED: $GOMODCACHE/google.golang.org/protobuf@v1.36.12/reflect/protoregistry/registry.go:126-133]`,
so this manifests as a logged error on every subsequent `ParseAll` fallback, not a silent divergence — but
it is still a bug budget this phase should not spend.

**Why it happens:** `ParseAll`'s `Import(d.Parse, ...)` call populates `FileRegistry` through `Parse()`
(`utils/utils.go:210-222`), which writes directly to the map rather than going through
`recordFileLocked` — so the natural single-insert-point growth hook (`recordFileLocked`) does not fire
for this path at all today. Growth for `ParseAll`'s files needs its own explicit registration step, and
that step's dedup must be backed by the exact same `before`/`after` diff already computed for
`lazyLoaded`, not a second, independently-written one.

**How to avoid:** After `Import(d.Parse, ...)` returns inside `ParseAll`, loop over the same
`after \ before` diff already used for `lazyLoaded` and call `filesResolver.RegisterFile(fd.UnwrapFile())`
for each name in that diff — reusing the exact keys, not recomputing a second set.

**Warning signs:** Two separate loops over `d.FileRegistry` inside `ParseAll` (one for `lazyLoaded`, one
for `filesResolver`) that could each independently drift from what the other considers "already present."

## Runtime State Inventory

Not applicable — this is not a rename/refactor/migration phase.

## Code Examples

### Locked accessor replacing `ParseFilesX`'s direct field read

```go
// utils/utils.go — new method, mirrors the existing FileDescriptor(name) shape
// (utils/utils.go:106-114)
func (d *DescriptorRegistry) FindFileByPath(path string) (protoreflect.FileDescriptor, error) {
    d.mu.RLock()
    defer d.mu.RUnlock()
    if d.filesResolver == nil {
        return nil, protoregistry.NotFound
    }
    return d.filesResolver.FindFileByPath(path)
}
```

### Growth call inside the existing single-writer insert point

```go
// Source: utils/utils.go:333-343, read this session — extended here
func (d *DescriptorRegistry) recordFileLocked(fd *desc.FileDescriptor) {
    if _, ok := d.FileRegistry[fd.GetName()]; ok {
        return
    }
    d.FileRegistry[fd.GetName()] = fd
    d.lazyLoaded[fd.GetName()] = struct{}{}
    d.MessageRegistry.AddFile("type.googleapis.com", fd)
    if d.filesResolver != nil {
        if err := d.filesResolver.RegisterFile(fd.UnwrapFile()); err != nil {
            slog.Error("failed to register file in growable resolver", "file", fd.GetName(), "error", err)
        }
        d.registrationCount++
    }
    for _, dep := range fd.GetDependencies() {
        d.recordFileLocked(dep)
    }
}
```

### `RegisterFile`'s actual cost shape (why this is O(1 file), not O(registry))

```go
// Source: $GOMODCACHE/google.golang.org/protobuf@v1.36.12/reflect/protoregistry/registry.go:114-176,
// read this session — appends to filesByPath/descsByName, iterates only THIS
// file's own top-level declarations (rangeTopLevelDescriptors), never touches
// entries already present from a prior call.
func (r *Files) RegisterFile(file protoreflect.FileDescriptor) error {
    // ... path/package/symbol conflict checks against existing entries (map
    // lookups, not a rebuild) ...
    p.files = append(p.files, file)
    rangeTopLevelDescriptors(file, func(d protoreflect.Descriptor) {
        r.descsByName[d.FullName()] = d
    })
    r.filesByPath[path] = append(r.filesByPath[path], file)
    r.numFiles++
    return nil
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| `parser.NewParserWithDescriptorRegistry` builds `FilesResolver` once from an eagerly-populated `FileRegistry` (864 files, 260ms) | Registry construction is lazy (Phase 11); `FilesResolver`'s one-shot build now runs over ~65 well-known types, already cheap | Phase 11 | The 260ms *measurement* is already gone; this phase's job is preventing the *shape* of that cost from returning via a naive "rebuild on every miss" fix to `RSLV-01`'s staleness |
| `ParseFilesX`'s `FilesResolver.FindFileByPath` branch is dormant (frozen, small resolver rarely hit) | Once `FilesResolver` grows, this branch becomes live and, unmodified, mints non-canonical descriptors | This phase | Requires the Pattern 2 fix above — a pre-existing code path changes from "rarely exercised" to "routinely exercised" purely as a side effect of fixing RSLV-01 |

**Deprecated/outdated:** None specific to this phase's stack — no library version changes. Phase 11's own
`11-RESEARCH.md` cited `google.golang.org/protobuf v1.34.1`; that citation is stale — `go.mod` pins
`v1.36.12` as of this session, and `RegisterFile`'s implementation is unchanged between the two (both read
this session).

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The growable `*protoregistry.Files` should live as a new field on `DescriptorRegistry` (guarded by the existing `d.mu`), not as a separate parser-owned type with its own lock | Architecture Patterns Pattern 1, Alternatives Considered | If the planner instead chooses a parser-owned wrapper, the lock-ordering discipline ("never hold the Parser's lock while calling into the registry, which holds `d.mu`") becomes a new invariant this phase must document and test for, on top of everything `recordFileLocked` already needs. Not fatal, but more surface than the registry-owned recommendation |
| A2 | `ParseFilesX`'s branch-2 fix (route a `FindFileByPath` hit back through `registry.FileDescriptor` before falling back to `ParseOne`) is sufficient to close the RSLV-03 hazard, without also needing to change what `desc.WrapFile`/`desc.CreateFileDescriptor` do for genuinely-external files (i.e., files reachable only via `protoregistry.GlobalFiles`, never via this registry's own `FileRegistry`) | Common Pitfalls #2, Architecture Patterns Pattern 2 | If some caller legitimately needs `ParseFilesX` to resolve a file that is in the growable `FilesResolver` but was never independently recorded in `FileRegistry` (a state D-02 says should not occur, since both are written at the same locked insert point) — the fallback to `ParseOne` in the code example above should still make forward progress, just via a slower path, rather than silently misresolving |
| A3 | Deleting `config.protoResolver` (zero confirmed readers, `compiler/lib/config.go:27`, `compiler/lib/compiler.go:356`) is the right disposition, versus pointing it at the growable view "for completeness" | Integration Points (below), user's own Specific Ideas section | If a future phase (or an external caller via some reflection path not grepped this session) turns out to read `config.protoResolver`, deleting it is a compile-time break, easily caught; pointing it at a growing view "just in case" risks exactly the "frozen snapshot in a struct this phase is making grow" hazard the CONTEXT explicitly warns against. Deletion is the safer default given the zero-reader finding is itself `[VERIFIED: grep -rn "\.protoResolver\b" --include="*.go" compiler/, read this session, zero matches]` |

## Open Questions

1. **Does the D-05 registration counter count `RegisterFile` *calls* or distinct *files*?**
   - What we know: `recordFileLocked`'s existing guard means `RegisterFile` is called at most once per
     distinct `fd.GetName()`, so under the recommended design the two numbers are identical in practice.
   - What's unclear: Whether the planner wants the counter to also assert on **zero duplicate-registration
     errors** (a stronger invariant: not just "the count matches," but "no `RegisterFile` call ever hit
     the `already registered` branch"), which would need the error return from `RegisterFile` surfaced to
     the test, not just logged.
   - Recommendation: Have the growth call site return (or the registry track) whether any registration
     attempt errored, and assert `0` in the RSLV-02 test — a silently-logged, non-zero duplicate-registration
     count would indicate a divergence bug (Pitfall 3) even if the final `FileRegistry`/`filesResolver`
     sets happen to end up correct.

2. **Should the SAFE-01 race test extend `TestConcurrentCompile` or add a dedicated lower-level test?**
   - What we know: `TestConcurrentCompile` (`compiler/lib/concurrent_compile_test.go:25`) already races 8
     goroutines, each compiling a config that references one private proto plus one shared proto, against
     one shared `*lib.Compiler` — this is already literally "two concurrent compiles ... each reaching a
     proto the other hasn't touched" (roadmap success criterion 3). Adding a post-`g.Wait()` assertion
     (e.g., `RangeFiles` count matches `LoadedFileCount()`-derived expectations) would prove the resolver
     ended up correct, but proving the *race itself* was exercised needs the interleaving to happen
     between `RegisterFile` and `FindFileByPath`/`RangeFiles`, not just between two `RegisterFile` calls.
   - What's unclear: Whether `-race` alone (without a deliberately-adversarial interleaving, e.g. one
     goroutine calling `RangeFiles` in a tight loop while others call `CompileFile`) will reliably surface
     an unlocked read/write race on a machine/CI-runner combination, or whether the plan needs a dedicated,
     lower-level `utils` package test that races `RegisterFile` against `FindFileByPath`/`RangeFiles`
     directly (mirroring `TestParseMemoization`'s errgroup shape) to guarantee the interleaving happens.
   - Recommendation: Do both — extend `TestConcurrentCompile` with a resolver-view assertion (cheap,
     proves end-to-end correctness) and add a dedicated `utils` package test that explicitly races
     `RegisterFile`(from N `ParseOne` calls on distinct paths) against a concurrent `RangeFiles`/`FindFileByPath`
     reader goroutine — the second one is what actually forces the interleaving `-race` needs to catch a
     regression if the lock is ever dropped from one side.

## Environment Availability

No external tools, services, or runtimes beyond what's already required to build/test this Go module
(`go build`, `go test -race`, both already used in CI). Skipping the full table — this phase is pure
in-repo Go code with zero new environment dependencies.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` + `github.com/stretchr/testify` (already used project-wide) |
| Config file | none — `go test ./...` |
| Quick run command | `go test ./compiler/... ./utils/... -race -run 'TestConcurrentCompile\|TestParseMemoization\|TestFilesResolver'` |
| Full suite command | `go test -race -coverprofile=coverage.txt -covermode=atomic ./...` (matches `.github/workflows/go.yml`'s existing step) |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| RSLV-01 | `FilesResolver` reflects a file `ParseOne` loaded after construction (a `FindFileByPath`/`RangeFiles` call sees it) | unit | New test in `utils` or `compiler/lib/parser`: `ParseOne` a path, then assert the registry's `FindFileByPath`/`RangeFiles` (or `Parser.FilesResolver.RangeFiles` for the compiler's own lazy registry) includes it | ❌ Wave 0 |
| RSLV-02 | A newly-parsed file's registration cost is proportional to its own closure, not the total registry size so far | unit | New test: load N protos one at a time (mirrors `TestParseMemoization`'s shape, `compiler/lib/parser/lazy_parse_test.go:20`), track the D-05 registration counter after each load, assert the counter's cumulative total equals the sum of each step's own new-file count (no re-registration of already-present files) | ❌ Wave 0 |
| RSLV-03 | `load()` A, then B, then re-reference A resolves all three correctly, with A's descriptor pointer identical across both references | unit | New test in `compiler/lib` (a `.pconf`/generated-corpus config with two `load()`s of the same proto path, or a lower-level `ParseFilesX` test): `require.Same(t, fdFromFirstLoad, fdFromReReference)` — pointer identity, not output-only (Common Pitfall #2) | ❌ Wave 0 |
| SAFE-01 | Concurrent compiles against one shared compiler, each reaching a proto the other hasn't touched, race-free | race/integration | Extend `TestConcurrentCompile` (`compiler/lib/concurrent_compile_test.go:25`) with a post-`g.Wait()` resolver-view assertion, **and** add a dedicated `utils` package test explicitly racing `RegisterFile` (via `ParseOne` on distinct paths) against a concurrent `RangeFiles`/`FindFileByPath` reader — both under `go test -race` | ❌ Wave 0 (extension); ❌ Wave 0 (new dedicated test) |

### Sampling Rate
- **Per task commit:** `go test ./compiler/... ./utils/... -race`
- **Per wave merge:** `go test -race ./...` (full suite, matches CI)
- **Phase gate:** Full suite green under `-race` before `/gsd-verify-work` — this phase's entire point is
  concurrency correctness over a growable structure, so skipping `-race` at the gate defeats the phase

### Wave 0 Gaps
- [ ] RSLV-01's "grows after construction" assertion — does not exist today (Phase 11's tests assert on
  `FileRegistry`/`LoadedFileCount`, not on `FilesResolver`/`RangeFiles` specifically)
- [ ] RSLV-02's registration-counter test (D-05) — new method, new test, both absent today
- [ ] RSLV-03's pointer-identity re-reference test — `TestParser_ParseFilesX` (`parser_test.go:20`)
  exists but only exercises the eager (`ImportPaths`-unset) path with a single filename per case, never
  two `load()`s of the same file across a lazy registry
- [ ] SAFE-01's resolver-growth-specific race test — `TestConcurrentCompile` exists and already races
  goroutines against a shared compiler, but has no assertion touching `FilesResolver`/`RangeFiles` today;
  a dedicated lower-level `RegisterFile`-vs-`RangeFiles` race test in `utils` does not exist

## Security Domain

`security_enforcement` is not set to `false` in `.planning/config.json`
`[VERIFIED: .planning/config.json, read this session — no security_enforcement key present]`, so this
section is included per protocol. As with Phase 11, this is an internal descriptor-registry change with
no new network input and no new auth surface.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|----------------|---------|-------------------|
| V2 Authentication | No | No auth surface touched |
| V3 Session Management | No | N/A |
| V4 Access Control | No | N/A |
| V5 Input Validation | Marginal | Same `.proto` files under `src/`, already a trusted, repo-local input; this phase changes *how* their resolvers are exposed to readers (growable vs. frozen), not what is trusted or parsed |
| V6 Cryptography | No | N/A — unchanged from Phase 11 |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Unsynchronized concurrent map mutation inside `protoregistry.Files` (a Go fatal error — "concurrent map writes" — not a soft race) if `RegisterFile` is ever called without holding `d.mu` | Denial of Service | `d.mu.Lock()` around every `RegisterFile` call, `d.mu.RLock()` around every `FindFileByPath`/`RangeFiles` call against the growable instance — this is the concrete, verified-this-session risk (Common Pitfall #1): the upstream type provides zero internal protection for a non-`GlobalFiles` instance |
| A double-registration error (`RegisterFile` returning "already registered") being silently swallowed by a bare `slog.Error`, masking a real `FileRegistry`/`filesResolver` divergence bug (Pitfall 3) | Tampering (of internal state consistency, not external input) | Surface the error to the D-05 counter/test (Open Question 1) rather than only logging it — a logged-and-ignored error here is exactly the kind of silent-wrong-answer risk `PITFALLS.md`'s framing ("a correctness change disguised as a performance change") warns about generally |

## Sources

### Primary (HIGH confidence — read this session)
- `utils/utils.go` (full file) — `DescriptorRegistry` struct (47-71), `NewDescriptorRegistry` (73-95),
  `FileDescriptor`/`GetFileDescriptorSet`/`GetFilesResolver`/`GetTypesResolver` (106-156), `Import`/`Parse`
  (158-222), `ParseOne` (236-326), `recordFileLocked` (333-343), `ParseAll` (352-373),
  `LoadedFileCount`/`LocalFileCount`/`FellBackToEager` (378-413)
- `compiler/lib/parser/parser.go` (full file) — `Parser` struct and `NewParserWithDescriptorRegistry`
  (23-46), `RegistryTypeResolver` (48-116), `ParseFilesX` (117-156), `ReadConfig` (158-164)
- `compiler/lib/compiler.go` (full file) — `NewCompiler` (53-83), `Compiler` struct, `c.load` (345-366,
  including the `messageRegistry: &c.ModuleService.GetProtoRegistry().MessageRegistry` pointer fix already
  shipped by Phase 11)
- `compiler/lib/config.go` (full file) — `config` struct including the `protoResolver` field (line 27),
  confirmed zero readers in package via `grep -rn "\.protoResolver\b" --include="*.go" compiler/`
- `compiler/lib/module_service.go` (lines 1-70, 395-464) — `ModuleService` struct, `NewLazyModuleService`,
  `GetProtoRegistry`'s `lazyRegistry`/`ImportPaths` gate (450-461), the dead `GetProtoFilesRegistry` (403-406)
- `compiler/lib/starlark_loader.go` (full file) — `starlarkLoader.cache`/`Load` memoization (23-110),
  `loadProto` (210-224) — confirms per-compile `load()` deduplication happens above `ParseFilesX`, and that
  RSLV-03's hazard is about descriptor-pointer identity across compiles/lookup-path variance, not about
  the loader re-invoking `loadProto` for a literal repeated `load()` string within one compile
- `compiler/lib/concurrent_compile_test.go` (full file) — `TestConcurrentCompile`, the existing SAFE-01-shaped
  production concurrency test (8 goroutines, one shared `*lib.Compiler`, each touching a private + shared proto)
- `compiler/lib/parser/lazy_parse_test.go` (full file) — `TestParseMemoization`, `TestLazyParseDoesNotMutateLocalFiles`
- `compiler/lib/parser/parser_test.go` (full file) — `TestParser_ParseFilesX` (eager-only today),
  `TestParser_ReadConfig`
- `compiler/lib/startup_bench_test.go` (lines 100-167) — `TestCompilerStartupScaling`, confirms GATE-01's
  `t.Skipf`-to-`require.LessOrEqual` flip is explicitly Phase 15's job, not this phase's
- `server/server.go` (lines 275-436) — `NewProtoconfMutationServer` (285-303), `Init()`'s discovery scan and
  reflection wiring (325-436), confirming `s.parser` is always built from the non-lazy `GetProtoRegistry()` path
- `server/init_prohibitions_test.go` (lines 115-144) — `TestDiscoveryScanDoesNotBackReflection`, an existing
  regression test that directly asserts `s.parser.FilesResolver.FindFileByPath` stays `NotFound` for an
  eager, `ImportPaths`-unset registry — a must-not-break test for this phase
- `go.mod` (relevant lines) — `google.golang.org/protobuf v1.36.12` (line 48), `github.com/jhump/protoreflect
  v1.16.0` (line 19), `golang.org/x/sync v0.22.0` (line 45), `go 1.25.8` (line 3)
- `$GOMODCACHE/google.golang.org/protobuf@v1.36.12/reflect/protoregistry/registry.go` (lines 82-176, 219-232,
  315-402) — `Files` struct doc comment, `RegisterFile`, `FindFileByPath`, `RangeFiles`, all confirmed to gate
  locking exclusively on `r == GlobalFiles`
- `$GOMODCACHE/google.golang.org/protobuf@v1.36.12/reflect/protodesc/desc.go` (lines 39-263) — `FileOptions`,
  `NewFile`, `NewFiles`/`FileOptions.NewFiles` — confirms `GetFilesResolver()`'s existing batch-rebuild cost
  shape and that incremental `RegisterFile` does not require this machinery
- `$GOMODCACHE/github.com/jhump/protoreflect@v1.16.0/desc/descriptor.go` (lines 40-90) — `FileDescriptor`
  struct, `Unwrap`/`UnwrapFile` — confirms the bridge from `*desc.FileDescriptor` to the already-linked
  `protoreflect.FileDescriptor` `RegisterFile` needs
- `.planning/phases/11-concurrency-safe-lazy-registry-core/11-RESEARCH.md`,
  `11-VERIFICATION.md`, `11-PATTERNS.md` — Phase 11's shipped design and its own flagged note that Phase
  12 will change `NewParserWithDescriptorRegistry`'s snapshot behavior
- `.planning/research/compiler-performance/{BASELINE,OPTIONS,PITFALLS}.md` — the milestone's own
  measurement and design-space record, cited throughout for the origin and correct interpretation of the
  "260ms" figure
- `.planning/phases/12-growable-resolver-views-race-safety/12-CONTEXT.md` — the user's locked decisions
  and discretion items, reproduced verbatim above
- `.planning/REQUIREMENTS.md`, `.planning/STATE.md` — project decision records read this session

### Secondary (MEDIUM confidence)
- WebSearch, cross-checked against the direct source read above: confirms community understanding that
  `protoregistry.Files.RegisterFile` is not safe for concurrent registration in practice ("file already
  registered" panics reported when multiple goroutines register the same file), consistent with — and
  subordinate to — the direct source read, which is the load-bearing citation

### Tertiary (LOW confidence)
- WebSearch results on generic Go "singleflight + RWMutex growable cache" patterns — general community
  idiom, not specific to this codebase; not load-bearing since the exact pattern is already shipped and
  verified in this repo (`ParseOne`, Phase 11)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependencies; `google.golang.org/protobuf`'s actual pinned version
  corrected from Phase 11's stale citation and re-verified directly against `go.mod` and `$GOMODCACHE`
- Architecture: HIGH — every code path cited was read this session with line numbers; the locking and
  field-type recommendations are backed by a full-tree grep of every `FilesResolver` reader, not inference
- Pitfalls: HIGH — all three pitfalls trace to source reads this session, including one (the doc-comment
  vs. actual-locking gap in `protoregistry.Files`) not previously documented anywhere in this project's
  research history

**Research date:** 2026-09-07
**Valid until:** Stable until the next phase touches `utils/utils.go`, `compiler/lib/parser/parser.go`,
or `server/server.go` — recommend re-checking after Phase 13 lands, since Phase 13's exact symbol index
and shared type-URL resolution path will add a third resolution mechanism alongside the two (`FilesResolver`
growth, `LocalResolver`'s miss-fallthrough) this phase establishes.
