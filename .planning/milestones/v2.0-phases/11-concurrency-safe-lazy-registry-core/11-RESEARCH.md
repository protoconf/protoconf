# Phase 11: Concurrency-Safe Lazy Registry Core - Research

**Researched:** 2026-09-04
**Domain:** Go proto descriptor lazy loading, concurrency-safe caching, gRPC dynamic service registration
**Confidence:** HIGH

## Summary

This phase has no CONTEXT.md — there was no `/gsd-discuss-phase` pass, so there are no user
locked-decisions to honor beyond what `REQUIREMENTS.md`, `ROADMAP.md`, `STATE.md`, and the
already-committed `.planning/research/compiler-performance/*` docs establish. That prior research
bundle (`SUMMARY.md`, `BASELINE.md`, `OPTIONS.md`, `PITFALLS.md`, `TESTING.md`, all dated
2026-09-04) already did the measurement and design-space work for the whole v2.0 milestone and is
authoritative — this document does not repeat it, it **extends it with source-verified specifics
for Phase 11's exact requirements** (LAZY-01..05, CONS-01) gathered by reading the actual files
this phase touches.

The eager parse+link happens at `compiler/lib/module_service.go:348-373`
(`ModuleService.GetProtoRegistry()`), specifically the call
`registry.Import(registry.Parse, []*regexp.Regexp{}, filepath.Join(protoconfPath, "src/"))` at
line 366, which walks every `.proto` under `src/` via `utils.DescriptorRegistry.Import`
(`utils/utils.go:110-160`) and parses+links all of them via `.Parse` (`utils/utils.go:162-174`).
The lazy replacement is **not a new library or a `protoregistry`/`protodesc` rewrite** — it is an
inversion already scoped in `OPTIONS.md` (Option A, chosen): make `parser.ParseFilesX`
(`compiler/lib/parser/parser.go:34-67`) actually parse a file the first time it's asked for by
path, and let `GetProtoRegistry()` return near-empty. `ParseFilesX` already has the map-lookup
shape LAZY-02 wants (`p.FileDescriptors[filename]` first, `p.FilesResolver.FindFileByPath`
second) — today both are always pre-filled because `GetProtoRegistry()` did the eager walk first.

**The one thing this phase's plan must get right that the prior research didn't verify against
source: `GetProtoRegistry()` is not just the compiler's function — it is called at construction
by five other components** (`server/server.go:290`, `inserter/inserter.go:220`,
`agent/filekv/filekv.go:80`, `mutate/mutate.go:73`, plus `mod sync`'s own separate eager instance).
Two already-passing tests — `inserter/inserter_test.go:21` and `:80` — assert on
`type.googleapis.com/test.v1.TestMessage` resolving through exactly this construction-time
snapshot. Making `GetProtoRegistry()` unconditionally lazy for all six consumers, as LAZY-01's
literal wording could be read to require, breaks these two tests today, before Phase 12
(growable resolvers) or Phase 13 (symbol index) exist to fix the fallback. See "Scope boundary"
below — this is the load-bearing design decision for this phase's plan.

Concurrency safety has two concrete, source-verified problems: (1) the future per-file
memoization map, and (2) a **struct-copy race that already exists in `go vet copylocks`
(BUG-03) and is currently benign only because nothing mutates the copied struct after
construction** — the lazy design directly removes that "nothing mutates after construction"
invariant, converting a cosmetic lint finding into a real, `-race`-catchable data race. CI
already runs `go test -race ./...` (`.github/workflows/go.yml`), so this is not hypothetical —
it will be caught, just possibly by CI rather than by design.

**Primary recommendation:** Scope the lazy path narrowly to the compiler's construction and
`ParseFilesX`, guard the shared descriptor map with a `sync.RWMutex` plus
`golang.org/x/sync/singleflight` (already a transitive dependency — no new package), fix the
`MessageRegistry` struct-copy at `compiler/lib/compiler.go:355` to a pointer as a
correctness-prerequisite of laziness (not a separate BUG-03 cleanup), and fix CONS-01 by giving
`ProtoconfMutationServer.Init()` its own one-time eager filesystem walk for service discovery —
mirroring the pattern `mod sync` already uses to stay eager on purpose.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| LAZY-01 | `GetProtoRegistry()` returns without bulk-walking, parsing or linking `src/` | "Summary", "Architecture Patterns" (Pattern 1), Open Question 1 — eager call site located at `compiler/lib/module_service.go:348-373`, line 366 is the exact call to remove/replace for the compiler's construction path |
| LAZY-02 | A proto file is parsed and linked on first request, memoised so a second request is a map lookup | "Architecture Patterns" Pattern 1 (`ParseOne` example), Common Pitfalls #1 and #4, Validation Architecture's LAZY-02 rows — existing half-built map-lookup shape at `compiler/lib/parser/parser.go:34-43` |
| LAZY-03 | On-demand single-file parsing never mutates `localFiles` | Common Pitfalls #2 — exact line (`utils/utils.go:163`) and mechanism identified; "Anti-Patterns to Avoid" third bullet |
| LAZY-04 | `mod sync` still parses and links the whole module tree, writing identical `.fds` files | "Architecture Patterns" Pattern 2, "Code Examples" second example, Validation Architecture's LAZY-04 row — confirms `Sync()`'s separate eager `DescriptorRegistry` instance (`module_service.go:399-408`) must stay untouched |
| LAZY-05 | Operator can see how many proto files a compile actually loaded | "Don't Hand-Roll" table, "Code Examples" third example, Validation Architecture's LAZY-05 row — existing `slog.Info` convention at `compiler/lib/compiler.go:66` identified as the pattern to extend, with the caveat that the count must be captured post-compile, not at construction |
| CONS-01 | Mutation server registers every custom gRPC service under `src/`, independent of what a config loaded | Common Pitfalls #3, "Architecture Patterns" Pattern 2 (full code example), Open Question 2, Validation Architecture's CONS-01 row — exact failure mechanism (`server/server.go:324-390`) and reusable fixture (`utils/testdata/small/src/test.proto:71-74`, `TestService`) both located |

</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Proto descriptor parse+link (registry construction) | Backend / Library (`compiler/lib`, `utils`) | — | Pure in-process Go library code; no network, no client tier involved |
| Per-file memoization / cache | Backend / Library (`utils.DescriptorRegistry`) | — | Shared mutable state consumed by every registry caller in-process |
| `.fds` cache serialization (`mod sync`) | Backend / Library (`compiler/lib/module_service.go`) | Filesystem / Storage (`.protoconf_cache/`) | Writes a cache artifact to disk; content lives in the library layer, persistence in storage |
| Compiler output / loaded-file-count logging | Backend / Library (structured `slog`) → CLI stdout | CLI (`cmd/protoconf`, `compiler/command.go`) | `slog` emits from the library; the CLI process is what an operator actually watches |
| Mutation server gRPC service registration | Backend / API (`server/server.go` `Init()`) | — | gRPC service catalog is server-tier startup wiring, not client-visible until `RegisterService` runs |
| Concurrent compile safety | Backend / Library | — | All concurrency in scope (errgroup over configs/files) is in-process goroutines sharing one `*lib.Compiler`, no cross-process concern |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/jhump/protoreflect` | v1.16.0 (pinned in `go.mod`) `[VERIFIED: go.mod:1]` | `protoparse.Parser`, `desc.FileDescriptor` — the existing parse+link engine | Already the project's only proto-parsing dependency; `OPTIONS.md` explicitly rejected migrating off it for this milestone |
| `google.golang.org/protobuf` | v1.34.1 (`go.mod`) | `protoregistry.Files`/`protoregistry.Types`, `protodesc` | Already in use for the resolver layer (`utils/utils.go:78-108`) |
| `golang.org/x/sync` | v0.22.0 `[VERIFIED: go.mod:45, and confirmed present at $GOMODCACHE/golang.org/x/sync@v0.22.0/singleflight]` | `singleflight.Group` for collapsing concurrent identical-path parse requests; `errgroup` already used by `compiler/service.go` and `compiler/command.go` | Already a direct dependency (used today for `errgroup`) — `singleflight` is the same module, zero new dependency to add |

**No new external package is required for this phase.** The concurrency primitives needed
(`sync.RWMutex`, `sync.Map`, `singleflight.Group`) are either stdlib or already vendored.

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `log/slog` | stdlib | Loaded-file-count instrumentation (LAZY-05) | Already the project's structured-logging convention (`compiler/lib/compiler.go:66`: `slog.Info("module service loaded", "took", ...)`) |
| `github.com/stretchr/testify` (`require`/`assert`) | v1.9.0 (`go.mod`) | New tests for LAZY-01..05, CONS-01 | Already the project's test assertion library everywhere |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `sync.RWMutex` + plain map | `sync.Map` | `sync.Map` avoids a custom lock but is optimized for "mostly-reads, stable key set" or "disjoint key sets per goroutine" access patterns; here every goroutine can read *and* write the *same* set of keys during a cold cache, which is `sync.Map`'s documented weak case. A `sync.RWMutex`-guarded map with accessor methods is simpler to reason about here and matches the project's existing pattern (`ModuleService.mutex sync.RWMutex`, `module_service.go:39`) |
| `singleflight.Group` | Per-key `sync.Once` stored in a `sync.Map` | Both dedupe concurrent identical requests. `singleflight` needs zero new per-key bookkeeping (no `sync.Once` object to allocate, store, and never garbage-collect); a per-key `sync.Once` map grows unboundedly and is exactly the kind of hand-rolled cache `singleflight` exists to replace. Use `singleflight` |
| Fixing `Init()`'s service discovery with a filesystem walk over a fresh eager `DescriptorRegistry` | Keeping `s.parser` (the mutation server's main registry) eager entirely, sidestepping the fix | Would satisfy CONS-01 trivially but silently keeps the mutation server on the eager 4.6s-shaped path indefinitely, and contradicts LAZY-01's requirement that `GetProtoRegistry()` stop bulk-walking. The one-time-eager-scan-for-service-discovery-only approach (below) gets both: fast lazy resolution for `Put()`/mutation type resolution, and correct eager discovery for `Init()` |

**Installation:** none — no new packages.

**Version verification:** `github.com/jhump/protoreflect v1.16.0` and `golang.org/x/sync v0.22.0`
confirmed present in `go.mod` and `$GOMODCACHE` this session (see Sources). No registry lookup
needed since nothing new is installed.

## Package Legitimacy Audit

Not applicable — this phase introduces zero new external packages. All concurrency primitives
(`sync.RWMutex`, `sync.Map`) are Go stdlib; `singleflight` is part of `golang.org/x/sync`, already
a direct dependency of this module (`go.mod:45`, used today by `compiler/service.go` and
`compiler/command.go` for `errgroup`).

## Architecture Patterns

### System Architecture Diagram

```
                         ┌─────────────────────────────────────────┐
                         │        ModuleService (per-process)       │
                         │                                           │
  cmd/protoconf compile  │  GetProtoRegistry() ──► cachedRegistry ── │───┐
  (errgroup over files)  │       (now returns near-empty registry,  │   │
        │                │        seeded with well-known types only)│   │
        │                └─────────────────────────────────────────┘   │
        ▼                                                               │
  NewCompiler() ──► Compiler{parser: *parser.Parser} (SHARED across     │
        │            concurrent CompileFile calls in one process)      │
        ▼                                                               │
  CompileFile(f) ──► starlarkLoader.Load() ──► load("//x/y.proto")      │
        │                          │                                    │
        │                          ▼                                    │
        │              starlarkLoader.loadProto(path)                   │
        │                          │                                    │
        │                          ▼                                    │
        │              parser.ParseFilesX(path)  ◄── LAZY-02 boundary   │
        │                          │                                    │
        │           ┌──────────────┴───────────────┐                    │
        │           │ 1. RLock, map lookup by path  │                    │
        │           │    HIT  → return cached FD    │  (2nd+ request:    │
        │           │    MISS → singleflight.Do(path)│   pure map lookup)│
        │           │           → parse+link ONE file│                    │
        │           │           → Lock, insert into  │                    │
        │           │             shared registry map│◄───────────────────┘
        │           └───────────────────────────────┘
        ▼
  writeConfig() / c.messageRegistry.FindMessageTypeByUrl()
        (must read through the SAME lock/pointer as the writer —
         see "MessageRegistry struct-copy race" pitfall)


  ─── separate, deliberately-still-eager paths (do not touch) ───

  protoconf mod sync ──► ModuleService.Sync() ──► fresh utils.NewDescriptorRegistry()
                          (own instance, walks src/ once, writes .fds — LAZY-04)

  ProtoconfMutationServer.Init() ──► fresh, one-time eager filesystem walk of src/
                          (own instance, for gRPC service discovery only — CONS-01 fix,
                           independent of s.parser which stays lazy for Put()/MutateConfig)
```

### Recommended Project Structure

No new packages or directories — this phase edits existing files in place:

```
utils/utils.go                     # DescriptorRegistry: add lock + lazy single-file parse method
compiler/lib/parser/parser.go      # ParseFilesX: parse-on-miss instead of assume-prefilled
compiler/lib/module_service.go     # GetProtoRegistry(): stop the eager Import() call (line 366)
compiler/lib/compiler.go           # fix MessageRegistry struct-copy at line 355; add LAZY-05 counter/log
server/server.go                   # Init(): eager filesystem-walk-based service discovery (CONS-01)
utils/testdata/small/src/test.proto # already has service TestService — reuse as CONS-01 fixture
```

### Pattern 1: Lazy-by-path parse with singleflight-guarded memoization

**What:** Replace the "always pre-filled" assumption in `ParseFilesX` with an explicit
check-then-parse-then-store sequence, safe under concurrent callers requesting the same or
different files.

**When to use:** Any time a shared, mutable, keyed cache is populated on demand by concurrent
goroutines (exactly `DescriptorRegistry.FileRegistry` under the new design).

**Example (shape only — the planner should verify field/method names against the actual
`DescriptorRegistry`/`Parser` structs read this session, `utils/utils.go:32-36` and
`compiler/lib/parser/parser.go:19-32`):**

```go
// utils/utils.go — new fields/methods on DescriptorRegistry
type DescriptorRegistry struct {
    MessageRegistry msgregistry.MessageRegistry
    FileRegistry    map[string]*desc.FileDescriptor
    localFiles      map[string]struct{}

    mu    sync.RWMutex          // guards FileRegistry reads/writes from the lazy path
    group singleflight.Group    // collapses concurrent requests for the same path
    lazyLoaded map[string]struct{} // NEW: files added by the lazy path, for LAZY-05's count —
                                    // deliberately separate from the well-known-type seed
}

// ParseOne parses and links a single file by path on first request, memoised thereafter.
// Never touches localFiles (LAZY-03) — that stays exclusively Parse()'s job for mod sync.
func (d *DescriptorRegistry) ParseOne(importPaths []string, path string) (*desc.FileDescriptor, error) {
    d.mu.RLock()
    if fd, ok := d.FileRegistry[path]; ok {
        d.mu.RUnlock()
        return fd, nil // LAZY-02: second request is a map lookup
    }
    d.mu.RUnlock()

    v, err, _ := d.group.Do(path, func() (interface{}, error) {
        // double-check under singleflight: another goroutine may have finished first
        d.mu.RLock()
        if fd, ok := d.FileRegistry[path]; ok {
            d.mu.RUnlock()
            return fd, nil
        }
        d.mu.RUnlock()

        parser := &protoparse.Parser{
            ImportPaths: importPaths,
            Accessor:    func(f string) (io.ReadCloser, error) { return os.Open(f) },
            LookupImport: func(s string) (*desc.FileDescriptor, error) {
                d.mu.RLock()
                fd, ok := d.FileRegistry[s]
                d.mu.RUnlock()
                if ok {
                    return fd, nil
                }
                return desc.LoadFileDescriptor(s) // compiled-in well-known types only
            },
        }
        fds, err := parser.ParseFiles(path) // protoparse resolves this file's own
        if err != nil {                     // transitive imports itself
            return nil, err
        }

        d.mu.Lock()
        for _, fd := range fds {
            d.FileRegistry[fd.GetName()] = fd
            d.lazyLoaded[fd.GetName()] = struct{}{}
        }
        d.mu.Unlock()
        d.MessageRegistry.AddFile("type.googleapis.com", fds[0]) // already internally
                                                                  // mutex-protected — see PITFALLS
        return fds[0], nil
    })
    if err != nil {
        return nil, err
    }
    return v.(*desc.FileDescriptor), nil
}
```

### Pattern 2: Separate eager instance for a startup-only correctness need (CONS-01)

**What:** `mod sync` already proves the pattern — build a *fresh*, throwaway
`utils.NewDescriptorRegistry()`, run its eager `Import()` once, and never let a lazy caller
share that instance (`compiler/lib/module_service.go:399-408`, `Sync()`).

**When to use:** Any consumer that structurally needs "everything under `src/`, right now,
once" rather than "whatever has been demanded so far" — `Init()`'s gRPC service discovery is
exactly this shape, and it is a startup cost with no 200ms budget (same justification `OPTIONS.md`
gives for keeping `mod sync` eager).

**Example — fixing CONS-01 at `server/server.go:324-390`:**

```go
// Init still registers built-in services first, unchanged (lines 325-327).
func (s *ProtoconfMutationServer) Init(rpcServer *grpc.Server) {
    protoconfmutation.RegisterProtoconfMutationServiceServer(rpcServer, &legacyProtoconfMutationServer{srv: s})
    protoconf_pb.RegisterProtoconfMutationServiceServer(rpcServer, s)
    protoconf_pb.RegisterProtoconfMutationReportServiceServer(rpcServer, s)

    s.exampleMaker = map[string]exampleFunc{}

    // CONS-01 fix: service discovery needs "every proto under src/, right now" — the same
    // shape mod sync already justifies staying eager for. Build a throwaway registry rather
    // than relying on s.parser.FilesResolver, which is now a lazy, load-driven view that is
    // near-empty at startup (nothing has been "load()"ed by any config yet).
    discoveryRegistry := utils.NewDescriptorRegistry()
    _ = discoveryRegistry.Import(discoveryRegistry.Parse, nil,
        filepath.Join(s.protoconfRoot, consts.SrcPath))
    discoveryFiles := discoveryRegistry.GetFilesResolver()

    discoveryFiles.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
        // ... identical body to today's loop (server/server.go:330-390), unchanged —
        // only the resolver being ranged over changes, from s.parser.FilesResolver
        // (now lazy) to discoveryFiles (still eager, scoped to this one startup call).
        return true
    })

    // s.parser.FilesResolver / LocalResolver (used below for reflection, and by Put()/
    // MutateConfig elsewhere) are left as the lazy, load-driven views — unaffected.
    ...
}
```

### Anti-Patterns to Avoid

- **Making `GetProtoRegistry()` unconditionally lazy for all six consumers in one step.**
  `inserter/inserter_test.go:21-71` and `:80-95` already assert on
  `type.googleapis.com/test.v1.TestMessage` resolving through the inserter's
  construction-time `parser.LocalResolver`, itself built from `ms.GetProtoRegistry()`
  (`inserter/inserter.go:220`). Nothing in the inserter's code path ever calls a
  parse-on-demand function, so under a blanket-lazy `GetProtoRegistry()` these tests fail
  immediately, mid-milestone, before Phase 12/13 exist to add the fallback. `OPTIONS.md`'s own
  "Blast radius" table already flags the compiler as "the only path-driven consumer, and the
  only one on the 200ms hot path" — treat that as the scope boundary, not just a comment.
- **Copying `msgregistry.MessageRegistry` by value after this phase lands.** See Common
  Pitfalls — this was silently safe only because nothing mutated the registry after
  construction; that invariant is exactly what this phase removes.
- **Reusing `DescriptorRegistry.Parse()` (the function, not a new method) for the lazy path.**
  `Parse()` unconditionally does `d.localFiles = map[string]struct{}{}` as its first line
  (`utils/utils.go:163`) — calling it per lazy request wipes `localFiles` down to whatever was
  most recently parsed, which is exactly the LAZY-03 violation and the CONS-01-adjacent
  `mod sync` truncation risk `PITFALLS.md` item 4 already describes. The lazy path needs its
  own method (`ParseOne` above, or equivalent) that never touches `localFiles`.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Collapsing N concurrent requests for the same not-yet-parsed file into one parse | A custom per-key mutex map, or a `map[string]*sync.Once` | `golang.org/x/sync/singleflight.Group` | Already a dependency (`go.mod:45`); purpose-built for exactly this "duplicate-suppression" shape; a hand-rolled `sync.Once` map leaks one `Once` object per distinct path forever, for the lifetime of the process |
| Eager, one-time, whole-tree proto parse for a startup correctness need | New parsing/walking code for `Init()` | `utils.DescriptorRegistry.Import`/`.Parse`, the exact machinery `mod sync` already uses for the identical shape | The functions already exist, are tested, and correctly resolve transitive imports; writing a second walker for `Init()` duplicates the first and doubles the surface for import-resolution bugs |
| Detecting "how many files did this compile actually load" | A new counter type or metrics wrapper | A plain `map[string]struct{}` (or the existing `FileRegistry` map's post-construction growth) tracked on `DescriptorRegistry`, logged via `slog` at the point `CompileFileAsync` finishes, matching the existing `slog.Info("module service loaded", "took", ...)` convention at `compiler/lib/compiler.go:66` | `slog` is already the project's structured-logging tool; no new dependency, no new logging pattern to learn |

**Key insight:** every mechanism this phase needs — memoized parsing, eager-scoped-instance
correctness, and file-count instrumentation — already has a working precedent somewhere in this
codebase (`ParseFilesX`'s map-lookup fast path, `mod sync`'s deliberately-separate eager
instance, `compiler.go:66`'s `slog.Info`). The job is composing existing shapes correctly, not
introducing new ones.

## Common Pitfalls

### Pitfall 1: `MessageRegistry` struct-copy race (BUG-03) turns live under laziness

**What goes wrong:** `compiler/lib/compiler.go:355` does
`messageRegistry: c.ModuleService.GetProtoRegistry().MessageRegistry,` — this is a **value copy**
of `msgregistry.MessageRegistry`
`[VERIFIED: $GOMODCACHE/github.com/jhump/protoreflect@v1.16.0/dynamic/msgregistry/message_registry.go:42-46, quoted: "type MessageRegistry struct {" ... "mu             sync.RWMutex"]`.
`compiler/lib/config.go:26` declares the receiving field as `messageRegistry msgregistry.MessageRegistry`
(value, not pointer) `[VERIFIED: compiler/lib/config.go:22-27, quoted: "type config struct {" ... "messageRegistry msgregistry.MessageRegistry" ... "protoResolver   protoregistry.MessageTypeResolver"]`,
read later at `config.go:68` via `c.messageRegistry.FindMessageTypeByUrl(...)`. Copying a struct
containing a `sync.RWMutex` gives the copy its **own, independent, zero-value mutex** — reads via
the copy are not actually synchronized against writes to the original, even though both
ultimately touch the same underlying map (Go maps are reference types, so `types`/`baseUrls`
inside `msgregistry.MessageRegistry` are shared between original and copy; the *mutex* is not).
This is exactly the pattern `go vet -copylocks` exists to catch (`REQUIREMENTS.md` BUG-03,
`compiler/lib/compiler.go:355`).

**Why it happens today without symptoms:** Today `GetProtoRegistry()` runs its entire eager
parse+link once, synchronously, inside `NewCompiler()`, before the `Compiler` is ever returned to
a caller. `MessageRegistry.AddFile` (the only writer, `utils/utils.go:166`) is never called again
after that point — every subsequent `c.load()` copy is a snapshot of an already-frozen structure.
No concurrent writer means the duplicated mutex never matters.

**Why laziness makes it a real bug:** LAZY-01/02 mean `MessageRegistry.AddFile` now runs
**during** compiles, every time a not-yet-seen proto is lazily parsed (see Pattern 1 above,
`d.MessageRegistry.AddFile(...)`). Two facts already verified this session make the race live: (1)
`compiler/service.go:31-56`'s `CompileFiles` gRPC handler runs `errgroup.Go` per file against
**one shared `*lib.Compiler`**, and (2) `compiler/command.go:140-151`'s `runLocally` (the plain
`protoconf compile a.pconf b.pconf` CLI path) does the identical thing. Both are real,
already-shipped call sites, not hypothetical test scaffolding. Once one goroutine's `c.load()`
takes its `messageRegistry` copy (an unsynchronized struct-field read) at the same moment another
goroutine's `loadProto` call is inside `AddFile`'s `r.mu.Lock()` section mutating the shared
maps, `go test -race` will flag it — CI already runs `go test -race ./...`
(`.github/workflows/go.yml`, `Run coverage` step).

**How to avoid:** Change `config.messageRegistry` to `*msgregistry.MessageRegistry` and
`compiler.go:355`'s assignment to `&c.ModuleService.GetProtoRegistry().MessageRegistry` — a
pointer means every reader and writer shares the *same* mutex instance, which
`msgregistry.MessageRegistry`'s own methods (`AddFile`, `FindMessageTypeByUrl`) already lock
correctly. This is a small, mechanical fix, but it is a **prerequisite of LAZY-02's correctness**,
not an optional cleanup. `REQUIREMENTS.md` lists BUG-03 as "deferred beyond this milestone" —
that framing predates this phase's design; flag this tension explicitly to the user (see
Assumptions Log) rather than silently overriding a recorded scope decision.

**Warning signs:** `go vet ./...` already flags this today (copylocks); `go test -race` will
newly flag it as an actual data race only once concurrent compiles start writing to
`MessageRegistry` mid-compile — meaning a plan that fixes the map/mutex around `FileRegistry` but
skips this one will pass a single-file, single-goroutine test suite and then fail intermittently
under `-race` with concurrent multi-file compiles, which is exactly what CI already runs weekly
and what LAZY-02's own test (see Validation Architecture) is designed to exercise.

### Pitfall 2: `Parse()`'s unconditional `localFiles` reset (LAZY-03's exact target)

**What goes wrong:** `utils.DescriptorRegistry.Parse` starts with
`d.localFiles = map[string]struct{}{}` `[VERIFIED: utils/utils.go:162-163, quoted: "func (d *DescriptorRegistry) Parse(parser *protoparse.Parser, files []string) error {" / "\td.localFiles = map[string]struct{}{}"]`
on **every call**, then repopulates it only with files from *this* call. `Store()`
(`utils/utils.go:188-205`) serializes exactly `d.localFiles`'s current contents to the `.fds`
cache file consumed by `mod sync`. If the lazy on-demand path calls `Parse()` per file (instead
of a dedicated method that skips this line), each lazy parse **wipes** the record of every
previously-loaded file, and any later `mod sync` sharing that registry instance would write a
truncated `.fds` — silently wrong, loads clean, breaks downstream (`PITFALLS.md` item 4,
already documented; this session locates the exact line the requirement maps to).

**Why it happens:** `Parse()` is designed for exactly one caller today —
`Import()` (`utils/utils.go:110-160`), called once per `GenFileDescriptorSet`/`Sync()` pass — so
resetting `localFiles` at the top is correct for that single-shot, whole-tree use.

**How to avoid:** Give the lazy per-file path its own method (Pattern 1's `ParseOne`) that
writes to `FileRegistry` and `MessageRegistry` but never touches `localFiles`. Keep `mod sync`
routed through the existing `Import`/`Parse` pair on its own **separate** `DescriptorRegistry`
instance (`module_service.go:399`, `Sync()` already does this — do not change it).

### Pitfall 3: `Init()`'s service discovery goes dark, not loud (CONS-01)

**What goes wrong:** `ProtoconfMutationServer.Init()` discovers custom gRPC services by ranging
`s.parser.FilesResolver` — a snapshot built once at `NewProtoconfMutationServer` construction from
`ms.GetProtoRegistry()` (`server/server.go:290,330`)
`[VERIFIED: server/server.go:324-390, quoted: "func (s *ProtoconfMutationServer) Init(rpcServer *grpc.Server) {" ... "s.parser.FilesResolver.RangeFiles(func(fd protoreflect.FileDescriptor) bool {"]`.
Once `GetProtoRegistry()` is lazy, this snapshot is near-empty at `Init()` time — `Init()` runs at
server startup, strictly before any config has been compiled, so nothing has triggered a lazy
parse of any `src/` file yet. A custom service defined in, say, `src/myservice/api.proto` is
simply never seen, and never registered — for the **entire process lifetime** (there is no
retry: `Init()` runs exactly once). This fails silently: the server starts fine, logs nothing
wrong, and the service is just unreachable.

**Why it happens:** `Init()`'s discovery mechanism (`RangeFiles` over a resolver) assumes the
resolver already contains everything, which was true only because `GetProtoRegistry()` was
eager. There is no code path that asks "did I find zero custom services — is that expected or a
regression?"

**How to avoid:** Pattern 2 above — give `Init()`'s discovery its own one-time eager scan,
independent of `s.parser`'s (now lazy) resolvers. `utils/testdata/small/src/test.proto:71-74`
already defines a matching fixture — `service TestService { rpc PutTestMessage(TestMessage)
returns (protoconf.v1.ConfigMutationResponse); rpc PutValidateMe(ValidateMe) returns
(protoconf.v1.ConfigMutationResponse); }`
`[VERIFIED: utils/testdata/small/src/test.proto:71-74, quoted verbatim above]`, and
`server/server_test.go:142-158`'s existing `TestProtoconfMutationServer_GenReflectionUI` already
calls `server.Init(rpcServer)` against `testdata.SmallTestDir()`. No new fixture proto needs to be
authored — success criterion 5's "fixture service and a test that fails without the fix" can be
built by adding an assertion (e.g. `rpcServer.GetServiceInfo()["test.v1.TestService"]` is present
and non-empty) to a **new** test that constructs the server the same way, since no existing test
currently asserts on service registration at all (only `GenReflectionUI`'s side effects are
checked today).

### Pitfall 4: Concurrent compiles are not a hypothetical for this phase

**What goes wrong:** A plan that treats "concurrency safety" as "add a mutex somewhere and move
on" without a test that actually launches concurrent compiles will not catch anything — Go's
race detector only flags races that are *exercised*.

**Why it happens:** The existing test suite (`compiler/lib/*_test.go`,
`compiler/lib/startup_bench_test.go`) exercises `NewCompiler`+`CompileFile` sequentially, one at a
time, per test. Nothing today constructs one `*lib.Compiler` and calls `CompileFile`/
`CompileFileAsync` from multiple goroutines within a single test — even though that pattern
already exists in production code (`compiler/service.go:31-56`, `compiler/command.go:140-151`).

**How to avoid:** The plan must add a test that does this explicitly: one shared `*lib.Compiler`
over a generated multi-config corpus, N goroutines each compiling a different config file that
overlaps some proto imports with the others, run under `go test -race`. See Validation
Architecture below for the concrete shape, reusing `testdata.GenerateCorpus`.

**Warning signs:** A plan whose only new tests are single-goroutine assertions ("second call
returns the same pointer") has verified LAZY-02's *correctness* but not its *concurrency safety*
— those are two different properties and need two different tests.

## Runtime State Inventory

Not applicable — this is not a rename/refactor/migration phase.

## Code Examples

### Existing map-lookup fast path already half-built (`compiler/lib/parser/parser.go:34-43`)

```go
// Source: compiler/lib/parser/parser.go, read this session
func (p *Parser) ParseFilesX(filenames ...string) (results []*desc.FileDescriptor, err error) {
    for _, filename := range filenames {
        if fd, ok := p.FileDescriptors[filename]; ok {
            results = append(results, fd)
            continue
        }
        fd, err := p.FilesResolver.FindFileByPath(filename)
        // ... today this branch always succeeds because GetProtoRegistry() pre-filled
        // both maps eagerly. The lazy design's job is: when this branch's lookup
        // MISSES, parse+link filename on demand instead of returning a "not found" error.
```

### `mod sync`'s deliberately-separate eager instance — the pattern to copy for CONS-01

```go
// Source: compiler/lib/module_service.go:399-408, read this session
registry := utils.NewDescriptorRegistry()
err = m.Walk(func(r *module.RemoteRepo) error {
    if r.Url == "." {
        return nil
    }
    err := m.GenFileDescriptorSet(registry, r)
    return err
})
```

### Loaded-file-count logging convention to extend for LAZY-05

```go
// Source: compiler/lib/compiler.go:65-66, read this session
registry := ms.GetProtoRegistry()
slog.Info("module service loaded", "took", time.Since(t))
// LAZY-05: this line runs at construction, when the lazy registry is still near-empty —
// the count that matters is captured AFTER CompileFile finishes, not here. Track it on
// DescriptorRegistry (a set of lazily-loaded paths, distinct from the well-known-type
// seed already in FileRegistry at construction — see utils/utils.go:38-59's
// globalRegexMatcher seed) and log it where CompileFileAsync's goroutine finishes
// (compiler/lib/compiler.go, near the `cancel(nil)` at the end of the closure).
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| `GetProtoRegistry()` eagerly walks and parses all of `src/` | `ParseFilesX` parses on demand, memoised, guarded by lock + singleflight | This phase | Registry construction: 4,639ms → ~3ms on the 799-proto corpus (`BASELINE.md`) |
| `add_validator` validators discovered by ranging the (now-lazy, incomplete) registry | Validators discovered by walking the filesystem for `*.proto-validator` | Already shipped, quick task 260904-f5j, `starlark_loader.go:112-147` `[VERIFIED, read this session]` | Prerequisite for this phase — already done, do not re-touch |

**Deprecated/outdated:** None specific to this phase's stack — no library version changes.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | LAZY-01's "`GetProtoRegistry()` returns without bulk-walking" should be scoped to the compiler's construction path only, not literally every one of the six callers, in Phase 11 | Summary, Architecture Patterns, Anti-Patterns | If the user actually wants all six consumers lazy in this phase (accepting the inserter test breakage as a known, temporarily-red state until Phase 13/14), the plan needs to explicitly say so and probably needs to also (a) skip or update `inserter/inserter_test.go:21` and `:80` for the duration, and (b) get sign-off that GATE-04 ("every pre-existing test stays green") is being read as "green at milestone close" rather than "green after every phase" |
| A2 | BUG-03's fix (pointer instead of value for `MessageRegistry`) should be folded into this phase as a correctness prerequisite of LAZY-02, despite `REQUIREMENTS.md` listing BUG-03 as "deferred beyond this milestone" | Common Pitfalls #1 | If the user wants BUG-03 truly untouched, the plan needs an alternative way to make `config.messageRegistry` reads safe against concurrent `AddFile` writes — e.g., taking the copy under `d.mu` alongside the `FileRegistry` lock, which is more contorted but avoids touching the field declared out-of-scope. Worth raising explicitly rather than assuming |
| A3 | Whether the mutation server's `s.parser` (used by `Put()`/`MutateConfig`, not just `Init()`) should also become lazy in this phase, or should it stay eager like today, given only `Init()`'s CONS-01 failure mode is in this phase's explicit success criteria | Architecture Patterns Pattern 2 | If `s.parser` stays eager, the mutation server keeps paying the full 4.6s-shaped startup cost this phase doesn't fix (acceptable — "Measured impact" for Phase 11 is scoped to the compile path only, per `ROADMAP.md`). If it goes lazy, `Put()`/`MutateConfig`'s type-URL resolution for custom (non-well-known) types breaks the same way the inserter's does (A1), since nothing yet triggers a lazy parse from a type-URL lookup — that mechanism is Phase 13's exact-symbol-index / shared-resolution-path work, not this phase's |

## Open Questions (RESOLVED — see D-01/D-02 in 11-01-PLAN.md and 11-02-PLAN.md)

1. **Does LAZY-01 apply to all six `GetProtoRegistry()` consumers in this phase, or only the compiler?**
   - What we know: The milestone's own sequencing (`STATE.md` decision, 2026-09-04: "Phases
     13-14 are correctness-only, protecting non-compiler-hot-path consumers from regressing
     under the change") implies non-compiler consumers are expected to still need protection
     *after* this phase — meaning they should not yet be exposed to the fully-lazy behavior, or
     they will regress before their protection exists.
   - What's unclear: `LAZY-01`'s literal wording names `GetProtoRegistry()` itself, which is
     currently one function shared by all six callers — REQUIREMENTS.md does not spell out a
     per-consumer scoping.
   - Recommendation: Scope Phase 11's laziness to the compiler's construction path (`NewCompiler`
     → `GetProtoRegistry()` → `ParseFilesX`), and have `server/server.go`, `inserter/inserter.go`,
     `agent/filekv/filekv.go`, `mutate/mutate.go` continue to receive an eagerly-populated
     registry for now (their call to `ms.GetProtoRegistry()` can either keep the eager Import
     internally via a parameter/second method, or the phase can introduce
     `GetLazyProtoRegistry()` as the new compiler-only entry point while `GetProtoRegistry()`
     keeps today's eager behavior for the other four). This satisfies LAZY-01's *intent* (the
     "GetProtoRegistry()" the milestone's `BASELINE.md`/`OPTIONS.md` measured 4,639ms against is
     specifically the one called from `NewCompiler`), keeps GATE-04 green throughout, and defers
     the harder problem (making the other four consumers correct under laziness) to where the
     roadmap already scheduled it (Phase 13/14). Confirm this scoping with the user before
     planning, since it is a load-bearing interpretation of a requirement, not a pure
     implementation detail.

2. **Should `Init()`'s new eager discovery scan also feed `s.parser.FilesResolver`/`LocalResolver` (used for reflection at `server/server.go:392-403`), or stay scoped to service registration only?**
   - What we know: Success criterion 5 only requires service *registration* to survive; it says
     nothing about gRPC server reflection (`grpcurl`/`grpcui` browsing) showing every `src/`
     type at startup.
   - What's unclear: If reflection's `DescriptorResolver` stays on the lazy `s.parser.FilesResolver`,
     an operator browsing reflection at startup (before any mutation has touched a given type)
     would see an incomplete schema — arguably a UX regression, but not one of this phase's
     five listed success criteria, and CONS-04 (Phase 14) explicitly owns "reports rather than
     silently skips" correctness for a related but distinct consumer (`GenReflectionUI`'s
     `mutable_config/` walk, not startup schema reflection).
   - Recommendation: Leave reflection on the lazy resolver for this phase (smallest diff that
     satisfies CONS-01 exactly as written); flag the reflection-completeness question as a
     candidate for Phase 14 rather than solving it here.

## Environment Availability

No external tools, services, or runtimes beyond what's already required to build/test this
Go module (`go build`, `go test -race`, both already used in CI). Skipping the full table — this
phase is pure in-repo Go code with zero new environment dependencies.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` + `github.com/stretchr/testify` (already used project-wide) |
| Config file | none — `go test ./...` |
| Quick run command | `go test ./compiler/... ./server/... -run TestLazy -race` (once new tests are named) |
| Full suite command | `go test -race -coverprofile=coverage.txt -covermode=atomic ./...` (matches `.github/workflows/go.yml`'s existing step) |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| LAZY-01 | Compiler construction cost proportional to config's own deps, not `src/` size | unit/scaling | `go test ./compiler/lib/... -run TestCompilerStartupScaling -short=false` | ✅ exists (`compiler/lib/startup_bench_test.go:108`) — Phase 11 should make its allocation ratio drop toward ≤2.0x, but flipping the `t.Skipf` to `require.LessOrEqual` is explicitly Phase 15's job (GATE-01), not this phase's |
| LAZY-02 | Second request for the same proto costs a map lookup, not a re-parse | unit | New test: call `ParseFilesX`/`ParseOne` twice with the same path, assert `require.Same(t, fd1, fd2)` (pointer identity proves no re-parse) | ❌ Wave 0 |
| LAZY-02 (concurrency) | Concurrent compiles requesting overlapping and disjoint protos, race-free | race/integration | New test: one shared `*lib.Compiler`, `errgroup` over N goroutines each calling `CompileFile` on a distinct generated config, `go test -race` | ❌ Wave 0 — reuse `testdata.GenerateCorpus`/a small multi-config variant |
| LAZY-03 | On-demand parse never mutates `localFiles` | unit | New test: call the lazy parse path, then call `Store()` on a *separately eager-populated* registry, assert its `.fds` output is unaffected by the lazy call happening on a different instance; or, more directly, assert `len(registry.localFiles)` (via a package-internal test) is unchanged after a lazy call | ❌ Wave 0 |
| LAZY-04 | `mod sync` still writes a byte-identical `.fds` | regression | Golden-file or checksum comparison: run `mod sync` before and after this phase's changes on the same fixture corpus, diff the `.fds` bytes | ❌ Wave 0 — no existing test exercises `GenFileDescriptorSet`'s output content directly; check for one before assuming absence |
| LAZY-05 | Operator can see loaded-file count from compiler output | unit | New test: capture `slog` output (or read the new counter/accessor directly, preferred — avoids brittle log-string matching) after `CompileFile`, assert it reflects only the config's own transitive deps, not `src/`'s total | ❌ Wave 0 |
| CONS-01 | Custom gRPC service registered and reachable at server startup, before any compile | integration | New test: `NewProtoconfMutationServer(testdata.SmallTestDir())`, `Init(rpcServer)`, assert `rpcServer.GetServiceInfo()["test.v1.TestService"]` is present, **without** calling `CompileFile` first | ❌ Wave 0 — fixture (`test.proto`'s `TestService`) already exists; only the test is missing |

### Sampling Rate
- **Per task commit:** `go test ./compiler/... ./server/... ./utils/... -race`
- **Per wave merge:** `go test -race ./...` (full suite, matches CI)
- **Phase gate:** Full suite green under `-race` before `/gsd-verify-work` — this phase's whole
  point is concurrency correctness, so skipping `-race` at the gate defeats the phase

### Wave 0 Gaps
- [ ] A concurrent-compile race test (Pitfall 4) — does not exist today; this is the
  highest-value new test in the set, since it is the only one that actually exercises the
  concurrency claim in the phase name
- [ ] A `ParseFilesX`/`ParseOne` pointer-identity memoization test (LAZY-02)
- [ ] A CONS-01 service-registration assertion on the existing `TestService` fixture (no new
  `.proto` needed)
- [ ] A `.fds` byte-identity regression check for `mod sync` (LAZY-04) — verify no existing test
  already does this before adding one; a quick `grep -rn "GenFileDescriptorSet\|\.fds" **/*_test.go`
  came back empty for assertions on file *content* (only on file *existence* via `mod sync`'s
  CLI tests, unverified this session — check before writing a duplicate)

## Security Domain

`security_enforcement` is not set to `false` in `.planning/config.json` (absent = enabled per
this project's own convention `[VERIFIED: .planning/config.json — no security_enforcement key present]`),
so this section is included per protocol. In practice this phase has minimal security surface:
it is an internal descriptor-cache and startup-wiring change with no new network input, no new
auth surface, and no new untrusted-input parsing beyond what `protoparse` already does on
repository-local `.proto` files (a trust boundary that already exists today, unchanged by this
phase).

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | No auth surface touched by this phase |
| V3 Session Management | No | N/A |
| V4 Access Control | No | N/A |
| V5 Input Validation | Marginal | `.proto` files under `src/` are already a trusted, repo-local input parsed by `protoparse` today; this phase changes *when* a given file is parsed (on demand vs. upfront), not *what* is trusted. No new validation control needed |
| V6 Cryptography | No | N/A — `.fds` cache uses MD5 for a content-identity checksum (`utils/utils.go:225-231`, pre-existing, not a security boundary — it's a cache-invalidation key, not a signature), unchanged by this phase |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Concurrent map read/write panic (Go fatal error, not a Go data race per se, but the failure mode of an unguarded `map[string]*desc.FileDescriptor` under concurrent lazy writes) | Denial of Service | `sync.RWMutex` around `FileRegistry` (Pattern 1) — this is the actual, concrete DoS-shaped risk this phase must close: an unguarded concurrent map write crashes the whole process, not just the current request |
| Unbounded cache growth from adversarial `load()` paths (a config `load()`ing a huge number of distinct, attacker-influenced proto paths to force unbounded memory growth) | Denial of Service | Out of scope for this phase — `src/` paths are project-controlled, not user/network input, in every current deployment shape (compiler reads local files, mutation server config source is the same repo). Note for SAFE-03 (Phase 14): "the lazy set stays proportional to what was actually demanded" is the relevant future control, not this phase's job |

## Sources

### Primary (HIGH confidence — read this session)
- `compiler/lib/module_service.go` (full file) — `GetProtoRegistry()` eager walk at lines 348-373, `GenFileDescriptorSet`/`Sync()`'s separate eager instance at 302-341, 379-408
- `compiler/lib/parser/parser.go` (full file) — `ParseFilesX`'s existing map-lookup shape, lines 34-67
- `utils/utils.go` (full file) — `DescriptorRegistry` struct (32-36), `Import`/`Parse`/`Store`/`Load` (110-250), the well-known-type seed (`globalRegexMatcher`, line 68)
- `compiler/lib/compiler.go` (full file) — `NewCompiler` (53-83), the `MessageRegistry` struct-copy at line 355, existing `slog.Info` convention at line 66
- `compiler/lib/starlark_loader.go` (full file) — confirms `loadValidators` fix already shipped (112-147), `loadProto`'s call into `ParseFilesX` (210-224), `loadMutable`'s type-URL resolution path (166-208)
- `compiler/lib/config.go` (lines 22-27, 68) — `config.messageRegistry` field type
- `server/server.go` (lines 1-70, 276-405) — `NewProtoconfMutationServer` (284-302), `Init()`'s service discovery loop (324-390)
- `server/server_test.go` (lines 1-40, 140-160) — existing `TestProtoconfMutationServer_GenReflectionUI`, no existing service-registration assertion
- `utils/testdata/small/src/test.proto` (lines 1-20, 55-75) — `TestService` fixture already present
- `inserter/inserter_test.go` (lines 1-100) — `TestProtoconfInserter_InsertConfig` and `TestProtoconfInserter_InsertConfig_AnyResolution`, both dependent on eager `GetProtoRegistry()`
- `inserter/inserter.go`, `agent/filekv/filekv.go`, `mutate/mutate.go` (grep + targeted reads) — confirmed identical `parser.NewParserWithDescriptorRegistry(ms.GetProtoRegistry())` construction pattern in all three
- `compiler/service.go`, `compiler/command.go` (full/partial) — confirmed the two existing production call sites that run concurrent `CompileFile`/`CompileFileAsync` against one shared `*lib.Compiler`
- `go.mod` (relevant lines) — `github.com/jhump/protoreflect v1.16.0`, `golang.org/x/sync v0.22.0`
- `$GOMODCACHE/golang.org/x/sync@v0.22.0/` (directory listing) — confirmed `singleflight` package present
- `$GOMODCACHE/github.com/jhump/protoreflect@v1.16.0/dynamic/msgregistry/message_registry.go` (lines 42-46, 149-161, 185-197) — confirmed `MessageRegistry`'s internal `sync.RWMutex` and that `AddFile`/`FindMessageTypeByUrl` correctly lock it
- `$GOMODCACHE/github.com/jhump/protoreflect@v1.16.0/desc/protoparse/parser.go` (grep) — confirmed `protoparse.Parser` has no internal synchronization of its own
- `.github/workflows/go.yml` — confirmed CI already runs `go test -race ./...`
- `.planning/config.json` — confirmed `nyquist_validation: true`, no `security_enforcement` override
- `compiler/lib/startup_bench_test.go` (full file) — existing scaling test/benchmark infra reusable for LAZY-01 verification
- `utils/testdata/embed.go`, `utils/testdata/corpus.go` (grep) — `SmallTestDir()`, `GenerateCorpus(dir, n)` signatures

### Secondary (MEDIUM confidence)
- `.planning/research/compiler-performance/{SUMMARY,BASELINE,OPTIONS,PITFALLS,TESTING}.md` — the already-committed prior research for this whole milestone, dated 2026-09-04. Treated as authoritative design-space work this document extends, not re-verifies (its own measurements were made by directly instrumenting `NewCompiler`, per its own text)
- `.planning/REQUIREMENTS.md`, `.planning/ROADMAP.md`, `.planning/STATE.md`, `.planning/PROJECT.md` — project decision records read this session

### Tertiary (LOW confidence)
- None — every claim above traces to a file read this session or the already-committed research bundle.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependencies; existing dependency versions confirmed in `go.mod` and `$GOMODCACHE` directly
- Architecture: HIGH — every code path cited was read this session with line numbers; the scope-boundary recommendation (Open Question 1) is the one genuinely judgment-based call, flagged explicitly rather than asserted as settled
- Pitfalls: HIGH — all four pitfalls trace to source reads, including two (the `MessageRegistry` struct-copy race and the inserter test breakage) that are new findings not present in the prior `.planning/research/compiler-performance/` bundle

**Research date:** 2026-09-04
**Valid until:** Stable until the next phase touches `compiler/lib/module_service.go`,
`compiler/lib/parser/parser.go`, `utils/utils.go`, or `server/server.go` — recommend re-checking
after Phase 12 lands, since Phase 12's growable resolvers will change some of the exact call
shapes described here (particularly `NewParserWithDescriptorRegistry`'s snapshot behavior).
