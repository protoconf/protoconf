# Stack Research — Lazy Proto Descriptor Resolution

**Domain:** Go proto descriptor resolution (jhump/protoreflect + protobuf-go), for making `protoconf`'s compiler startup lazy/load-driven
**Researched:** 2026-09-04
**Confidence:** HIGH — every claim below is verified against the pinned module source (file:line), not recalled. Quotes are copied from the actual doc comments/code.

No new dependency is proposed. This document is an API-contract map of the stack already pinned in `go.mod`: `github.com/jhump/protoreflect v1.16.0`, `github.com/bufbuild/protocompile v0.10.0` (indirect), `google.golang.org/protobuf v1.34.1`.

## Confirmed shared-state hazard (why this matters more than usual)

`compiler/service.go` builds **one** `*lib.Compiler` in `NewCompilerService` and reuses it for the life of the process (`compiler/service.go:21-29`). `CompileFiles` spawns a goroutine per request via `CompileFileAsync` (`compiler/lib/compiler.go:122,152`), and every goroutine calls into the **same** `c.parser` (`c.parser.LocalResolver` at `compiler/lib/compiler.go:295,354`). Today `Parser.FilesResolver`/`LocalResolver` are built once, eagerly, and never mutated again — so this is safe by accident (read-only after construction). The moment resolution becomes lazy, `FileRegistry`, `FilesResolver`, and `LocalResolver` start mutating **after** concurrent goroutines already hold references to them. Every contract below is read through that lens.

## Recommended Stack (APIs to use, as they already exist)

### Core Technologies

| API | Version | Purpose | Why Recommended |
|---|---|---|---|
| `protoparse.Parser.LookupImport` | jhump/protoreflect v1.16.0 | Intercept an import the source `Accessor` can't resolve from disk, and hand back a pre-built `*desc.FileDescriptor` | Already the exact seam `utils.go`'s `Import()` uses (`utils/utils.go:141-152`). It is the correct place to serve a lazily-growing cache — see contract below. |
| `protoregistry.Files.RegisterFile` + `protoregistry.Types.RegisterMessage`/`RegisterEnum` | protobuf-go v1.34.1 | Incrementally add ONE already-linked file/message to a long-lived resolver, instead of rebuilding from a `FileDescriptorSet` | `desc.FileDescriptor.UnwrapFile()` (jhump) already returns a `protoreflect.FileDescriptor` — no round-trip through `protodesc.NewFiles` is needed per file. This directly eliminates the "260ms resolver rebuild" cost measured in BASELINE.md. |
| `protoregistry.Types` embedding the 4-method interface (`FindMessageByName`, `FindMessageByURL`, `FindExtensionByName`, `FindExtensionByNumber`) | protobuf-go v1.34.1 | The exact contract `protojson.MarshalOptions.Resolver` / `UnmarshalOptions.Resolver` require | `parser.go:74`'s `protojson.UnmarshalOptions{Resolver: p.LocalResolver}` only ever calls these 4 methods — see "Interface to implement" below. |

### Supporting APIs

| API | Version | Purpose | When to Use |
|---|---|---|---|
| `desc.FileDescriptor.UnwrapFile()` | jhump v1.16.0, `desc/descriptor.go:74-76` | Get the underlying `protoreflect.FileDescriptor` out of jhump's wrapper | Every time a newly-parsed `*desc.FileDescriptor` needs to go into a `*protoregistry.Files`/`*protoregistry.Types` — call this, don't round-trip through descriptor protos. |
| `msgregistry.MessageRegistry` (already in `utils.DescriptorRegistry.MessageRegistry`) | jhump v1.16.0, `dynamic/msgregistry/message_registry.go:42-46` | Type-URL → `*desc.MessageDescriptor` resolution for `google.protobuf.Any` (jhump's own Any-handling, not protojson's) | Already internally guarded by its own `sync.RWMutex` (field `mu`) — `AddFile`/`FindMessageTypeByUrl`/`Resolve` are already safe to call concurrently with each other, unlike `protoregistry.Types`. Keep using it for jhump-side Any resolution; it needs no extra locking. |
| `protodesc.Resolver` (2-method: `FindFileByPath`, `FindDescriptorByName`) | protobuf-go v1.34.1, `reflect/protodesc/desc.go:33-36` | What `protodesc.FileOptions{}.New`/`.NewFiles` need to resolve dependencies while building descriptors from raw `FileDescriptorProto` | Only relevant if linking from unlinked `FileDescriptorProto` (e.g. a persisted `.fds` warm-start, explicitly deferred). Not needed for the jhump `ParseFiles` path — those results are already fully linked. |

### What NOT to add

| Avoid | Why | Use Instead |
|---|---|---|
| `github.com/bufbuild/protoresolve` | Does not exist anywhere in this dependency graph — not in `go.sum`, not in the module cache, not a transitive dep of `protocompile@v0.10.0` or `protobuf@v1.34.1`. Adding it would be a new dependency, out of scope for this milestone. | The 2-method `protodesc.Resolver` and the 4-method `protojson` Resolver interface (both already shipped in `protobuf-go@v1.34.1`) cover every seam this project needs. |
| `linker.Resolver` / `linker.File` (protocompile internal) | `protocompile@v0.10.0/linker/files.go:176-193` defines `Resolver`/`ResolverFromFile`, but jhump's public API (`protoparse.Parser`) never exposes a `linker.File` to callers — it only returns `*desc.FileDescriptor` wrapping the final `protoreflect.FileDescriptor`. There is no way to reach this type through the pinned jhump version without vendoring protocompile directly. | `protoreflect.FileDescriptor` via `.UnwrapFile()`, which is the actual public output type. |
| Rolling your own eager symbol index across all files | Every eager index variant measured on the 799-proto corpus blows the 200ms budget by itself (`ParseFilesButDoNotLink`: 1,286ms; lexical scan: 623ms) — see OPTIONS.md. Not an API limitation, a workload-shape one. | Path-driven lazy resolution (Option A, already chosen) plus loud fallback for the type-URL case, per PITFALLS.md. |
| Migrating `jhump/protoreflect` → `dynamicpb`/`protoregistry` wholesale for this milestone | Explicitly deferred v2 item (PROJECT.md "Key Decisions"). Also empirically not implicated: `desc.WrapFiles` is 6.2ms and `desc.wrapFile` is 0.16s cumulative inside a 4.6s stage — the cost is in `protocompile`'s parse+link, which both APIs sit on top of. | Keep `jhump/protoreflect` for now; the fix is *how many files* get parsed and linked, not *which wrapper* sits on top of the result. |

## Exact API shapes (verified against pinned source)

### 1. `protoparse.Parser.LookupImport` / `Accessor` / `ParseFiles` vs `ParseFilesButDoNotLink`

`desc/protoparse/parser.go:54-131` (the `Parser` struct):

```go
type Parser struct {
    ImportPaths       []string
    InferImportPaths  bool
    LookupImport      func(string) (*desc.FileDescriptor, error)       // line 86
    LookupImportProto func(string) (*descriptorpb.FileDescriptorProto, error) // line 90
    Accessor          FileAccessor                                     // line 96
    IncludeSourceCodeInfo bool
    ValidateUnlinkedFiles bool
    InterpretOptionsInUnlinkedFiles bool
    ErrorReporter  ErrorReporter
    WarningReporter WarningReporter
}
```

Doc comment on `LookupImport` (`parser.go:74-86`), quoted verbatim: *"LookupImport is a function that accepts a filename and returns a file descriptor, which will be consulted when resolving imports... In the event of a filename collision, Accessor is consulted first, then LookupImport is consulted, and finally the well-known protos are used."*

Traced through `getResolver` (`parser.go:520-570`), this precedence is implemented as a `protocompile.CompositeResolver` of exactly two entries, tried in order:
1. `sourceResolver` — a `protocompile.SourceResolver{Accessor: ..., ImportPaths: p.ImportPaths}` that reads files off disk (or wherever `Accessor` points).
2. A fallback `protocompile.ResolverFunc` wrapping `LookupImport`/`LookupImportProto`, itself wrapped again by `protocompile.WithStandardImports(...)` so well-known protos (`google/protobuf/*.proto`) are the final fallback.

**Answer to "can `LookupImport` satisfy imports from a lazily-growing cache?" — yes, and it is already exactly what `utils.go`'s `Import()` does today** (`utils/utils.go:141-152`):

```go
LookupImport: func(s string) (*desc.FileDescriptor, error) {
    if fd, ok := d.FileRegistry[s]; ok {   // cache hit
        return fd, nil
    }
    fd, err := desc.LoadFileDescriptor(s)  // falls back to compiled-in Go protos
    if err == nil {
        d.FileRegistry[s] = fd
        return fd, nil
    }
    return nil, fmt.Errorf("failed to find descriptor for file: %s", s)
},
```

The gap for lazy-by-path resolution is not `LookupImport` — it is that `d.FileRegistry[s]` here is a **plain, unguarded `map[string]*desc.FileDescriptor` read+write**, and (per the shared-state hazard above) it will now be hit from concurrent goroutines post-construction. `LookupImport`'s own contract does not save you from this — see Concurrency Contracts below.

**`ParseFiles` vs `ParseFilesButDoNotLink`:** `ParseFiles` (`parser.go:149-201`) builds a `protocompile.Compiler{Resolver: res, MaxParallelism: 1, ...}` and calls `c.Compile(...)`, returning fully linked `[]*desc.FileDescriptor`. `ParseFilesButDoNotLink` (`parser.go:247-293`) skips linking entirely and returns `[]*descriptorpb.FileDescriptorProto` — type references unresolved, options uninterpreted. For the lazy path (`ParseFilesX` parsing one `load()`'d file plus its transitive imports), keep using `ParseFiles`: it's what already measures at 2.7ms for 7 files (BASELINE.md), and `ParseFilesButDoNotLink` buys nothing here — you still need linking eventually, and OPTIONS.md already measured `ParseFilesButDoNotLink` over the full 799-file corpus at 1,286ms (rejected as an eager index strategy, not proposed for the lazy per-file path either).

**Important, easy to miss:** `ParseFiles` hard-codes `MaxParallelism: 1` (`parser.go:185`). A single `ParseFiles` call will not invoke `LookupImport` from multiple goroutines *within that call*. But `protocompile.Resolver`'s own doc comment (`protocompile@v0.10.0/resolver.go:37-38`) says: *"Resolver implementations must be thread-safe as a single compilation operation could invoke FindFileByPath from multiple goroutines."* — and separately, the compiler *service* runs multiple concurrent `ParseFiles`/`ParseFilesX` calls (one goroutine per `CompileFileAsync`), all sharing the same `LookupImport` closure and the same `FileRegistry` map. **Thread-safety is required at the shared-cache layer regardless of `MaxParallelism`.**

### 2. `protoregistry.Files` / `protoregistry.Types` — mutation-after-construction and concurrency

Doc comment, `reflect/protoregistry/registry.go:82-85`: *"Files is a registry for looking up or iterating over files and the descriptors contained within them. The Find and Range methods are safe for concurrent use."* Identically worded for `Types` at line 476-478.

**That sentence is scoped to `protoregistry.GlobalFiles`/`GlobalTypes`, not to a custom instance.** Verified directly in the implementation — every mutating and reading method starts with the same pattern (`registry.go:114-118`, `registry.go:319-322`, `registry.go:350-357`, `registry.go:501-504`, `registry.go:619-622`, `registry.go:642-645`):

```go
func (r *Files) RegisterFile(file protoreflect.FileDescriptor) error {
    if r == GlobalFiles {
        globalMutex.Lock()
        defer globalMutex.Unlock()
    }
    // ... unguarded map read/write on r.filesByPath, r.descsByName ...
}
```

`var globalMutex sync.RWMutex` (`registry.go:67`) is a **package-level lock that only ever guards the two package-level singletons** `protoregistry.GlobalFiles` / `protoregistry.GlobalTypes`. A `*protoregistry.Files` or `*protoregistry.Types` you construct yourself (exactly what `utils.DescriptorRegistry.GetFilesResolver()`/`GetTypesResolver()` do) gets **zero internal locking**. `RegisterFile` mutates `r.filesByPath` and `r.descsByName` — plain Go maps — with no mutex at all when `r != GlobalFiles`. Same for `RegisterMessage`/`RegisterEnum`/`RegisterExtension` on a custom `*Types`.

Concretely, for a project-owned instance:
- Concurrent `FindFileByPath`/`RangeFiles`/`FindMessageByName`/`FindMessageByURL` calls among themselves ARE safe (concurrent map reads are fine in Go).
- **Concurrent `RegisterFile` while any goroutine is calling `FindFileByPath`/`RangeFiles`/`RangeFilesByPackage` on the same instance is a data race** — Go's race detector will flag it, and in the worst case a concurrent map write during a read panics at runtime (`fatal error: concurrent map read and map write`).
- **Concurrent `RegisterFile` calls from two goroutines is also a race** (two unguarded writers to the same map).

**Verdict for this project: a lazy `*protoregistry.Files`/`*protoregistry.Types` wrapper must own its own `sync.RWMutex`, taken as a write-lock around every `Register*` call and a read-lock around every `Find*`/`Range*` call it exposes.** Do not assume the doc comment's "safe for concurrent use" extends to your instance — it explicitly does not, and the source proves it (locking is conditioned on `r == GlobalFiles`/`r == GlobalTypes`, an identity check on the package singleton pointer).

**Registering an already-parsed `desc.FileDescriptor` incrementally (avoids the 260ms rebuild):**

```go
fd, _ := parser.ParseFiles(path)   // []*desc.FileDescriptor, already linked
files.RegisterFile(fd[0].UnwrapFile())  // protoreflect.FileDescriptor — no protodesc.NewFiles round-trip
```

`desc.FileDescriptor.UnwrapFile()` (`desc/descriptor.go:74-76`) returns the `protoreflect.FileDescriptor` the wrapper holds internally (`desc/descriptor.go:52`, field `wrapped protoreflect.FileDescriptor`) — this is exactly the type `RegisterFile` wants. Today's `utils.GetFilesResolver()` (`utils/utils.go:78-85`) instead serializes the *entire* `FileRegistry` map to a `descriptorpb.FileDescriptorSet` and calls `protodesc.FileOptions{AllowUnresolvable:true}.NewFiles(fds)` — a brand-new `*protoregistry.Files` built from scratch, every time it's called. That whole-registry rebuild is the "260ms / 864 files" line item in BASELINE.md, and it is **not required** for lazy operation: register the single new file directly, once, and never rebuild the resolver wholesale again on the hot path.

**One gotcha on `RegisterFile`'s duplicate-registration error:** it returns a plain formatted error (`errors.New("file %q is already registered", ...)`, `registry.go:128`), wrapped through `internal/errors.New` (`google.golang.org/protobuf@v1.34.1/internal/errors/errors.go:18-20`), whose doc comment says outright: *"Deliberately introduce instability into the error message string to discourage users from performing error string comparisons."* (`errors.go:26-32` — the prefix literally alternates between an ASCII space and a non-breaking space at random). There is **no exported sentinel** for "already registered" specifically — `Unwrap()` only gets you back to a generic `proto: ` sentinel shared by every error in the package. **Do not string-match or `errors.Is()`-match for duplicate registration.** Instead, check `FindFileByPath`/`FindMessageByName` for existence *under your own lock* before calling `RegisterFile`/`RegisterMessage`, so double-registration (a file reachable via two independent `load()` chains, or two concurrent compiles racing to cache the same file) never reaches `RegisterFile` in the first place.

### 3. Interface a lazy resolver must implement to drop into `protojson`

`encoding/protojson/decode.go:44-49` and `encode.go:105-109` both declare (identically) the `Resolver` field type as:

```go
Resolver interface {
    protoregistry.ExtensionTypeResolver
    protoregistry.MessageTypeResolver
}
```

Expanded from `reflect/protoregistry/registry.go:435-469`, that's exactly 4 methods:

```go
FindMessageByName(message protoreflect.FullName) (protoreflect.MessageType, error)
FindMessageByURL(url string) (protoreflect.MessageType, error)
FindExtensionByName(field protoreflect.FullName) (protoreflect.ExtensionType, error)
FindExtensionByNumber(message protoreflect.FullName, field protoreflect.FieldNumber) (protoreflect.ExtensionType, error)
```

`parser.go:74`'s `protojson.UnmarshalOptions{Resolver: p.LocalResolver}.Unmarshal(...)` (`p.LocalResolver` is `*protoregistry.Types`) only ever calls into these 4 methods. **A hand-written lazy resolver does not need to reimplement `*protoregistry.Types` wholesale** — it just needs a type satisfying this 4-method interface, e.g. a thin wrapper that, on a miss, triggers on-demand parse-and-register into a real `*protoregistry.Types` (guarded per the concurrency contract above) and then delegates. `protodesc.Resolver` (2-method: `FindFileByPath`, `FindDescriptorByName`, `reflect/protodesc/desc.go:33-36`) is the separate, narrower contract needed only if building single `protoreflect.FileDescriptor`s from raw `FileDescriptorProto` via `protodesc.FileOptions{}.New` — not needed on the jhump `ParseFiles` path, since jhump already hands back fully linked descriptors.

`linker.Resolver` (`protocompile@v0.10.0/linker/files.go:176-180`) is the union of `protodesc.Resolver` + both protoregistry interfaces, but it is internal to `protocompile`'s linking step — jhump's public `protoparse` API never returns a `linker.File`/exposes `linker.Resolver` to callers, only the wrapped `*desc.FileDescriptor`. Not reachable at the pinned jhump version without vendoring `protocompile` directly, and not needed: the two interfaces above already cover the call sites in `parser.go`.

### 4. `protoregistry.Files.RangeFiles` semantics while the set grows

`registry.go:347-365`:

```go
func (r *Files) RangeFiles(f func(protoreflect.FileDescriptor) bool) {
    ...
    for _, files := range r.filesByPath {
        for _, file := range files {
            if !f(file) { return }
        }
    }
}
```

It is a direct `range` over the live `filesByPath` map, **iteration order is explicitly undefined** (doc comment, line 348), and it takes no snapshot. Two consequences for lazy resolution:
1. **Do not call `RegisterFile` from inside the callback passed to `RangeFiles`** — Go forbids adding keys to a map while ranging over it (undefined behavior, not merely a lint warning) if it's the same map instance being mutated concurrently with the range's own goroutine, and it is an outright data race if a *different* goroutine calls `RegisterFile` concurrently with an in-flight `RangeFiles` on a non-`Global` instance (see Concurrency Contracts above — no internal lock).
2. `utils.GetTypesResolver()` (`utils/utils.go:87-108`) already does exactly this pattern today (`reg.RangeFiles(func(fd protoreflect.FileDescriptor) bool {... RegisterMessage ...})`) — but today it's safe only because it runs once, synchronously, at eager-construction time, over a `*protoregistry.Files` nobody else can see yet. Once file registration becomes ongoing/lazy, this function's contract changes: it must either (a) take a lock across the entire range-then-register operation, or (b) be restructured to register a `Types` entry at the same moment a `Files` entry is registered (single critical section), rather than deriving `Types` from a `RangeFiles` sweep after the fact.

### 5. Does jhump/protoreflect or protocompile ship a lazy/incremental resolver?

**No.** Verified two ways:
- `grep -rln "lazy\|incremental\|Lazy\|Incremental"` across both module trees (excluding tests) surfaces nothing related to descriptor caching — only unrelated hits (proto3-optional-field handling, a `.proto` test fixture literally named "descriptor-*", a validation helper). No lazy-resolver type exists in either package.
- `protocompile.Compiler` (`protocompile@v0.10.0/compiler.go:52-92`) has exactly 5 fields (`Resolver`, `MaxParallelism`, `Reporter`, `SourceInfoMode`, `RetainASTs`) — **no cache field**. And `protoparse.Parser.ParseFiles` (`parser.go:183-189`) constructs a brand-new `protocompile.Compiler{}` on every single call — there is no cross-call memoization anywhere in the jhump/protocompile stack itself. **All caching for lazy resolution must be hand-rolled at the `LookupImport`/`FileRegistry` layer**, which is exactly the shape `utils.go` already has (Option A in OPTIONS.md) — it just needs the concurrency guard described above, and to route the resolver-build path through incremental `RegisterFile` instead of whole-registry `protodesc.NewFiles` rebuilds.

## Concurrency Contracts — Summary Table

| Component | Safe for concurrent Find/Range among themselves? | Safe for concurrent Register while Find/Range in flight? | Verdict |
|---|---|---|---|
| `protoregistry.GlobalFiles` / `GlobalTypes` | Yes (`globalMutex`) | Yes (`globalMutex`) | Fine as-is; used only for compiled-in Go protos via `desc.LoadFileDescriptor`. |
| A project-owned `*protoregistry.Files` | Yes (plain map reads) | **No — data race, no internal lock** | Must wrap in your own `sync.RWMutex`. |
| A project-owned `*protoregistry.Types` | Yes (plain map reads) | **No — data race, no internal lock** | Must wrap in your own `sync.RWMutex`. |
| `msgregistry.MessageRegistry` (jhump) | Yes (own `sync.RWMutex`) | Yes (own `sync.RWMutex`) | Already safe; no change needed. Keep using for jhump-side Any resolution. |
| `d.FileRegistry map[string]*desc.FileDescriptor` (current `utils.go`) | No — plain map | No — plain map | Currently safe only because it's built once, eagerly, before any concurrent reader exists. Lazy mode breaks this invariant; needs a mutex. |
| `protoparse.Parser.LookupImport` closure | N/A (caller-supplied) | N/A | Must itself be safe for concurrent invocation — `protocompile.Resolver`'s doc comment requires it, and the compiler service's one-goroutine-per-`CompileFileAsync` model guarantees concurrent invocation in practice. |

## Version Compatibility

| Package A | Compatible With | Notes |
|---|---|---|
| `jhump/protoreflect@v1.16.0` | `bufbuild/protocompile@v0.10.0` (indirect) | jhump v1.16.0's `go.mod` pins this exact protocompile version; `ParseFiles`'s reliance on `protocompile.CompositeResolver`/`SourceResolver`/`WithStandardImports` (all v0.10.0 APIs) means bumping protocompile independently is not possible without bumping jhump — and PROJECT.md/OPTIONS.md already found v0.14.1 byte-identical for the relevant code path, so there is no reason to. |
| `jhump/protoreflect@v1.16.0` `desc.FileDescriptor.UnwrapFile()` | `google.golang.org/protobuf@v1.34.1` `protoregistry.Files.RegisterFile` | `UnwrapFile()` returns `protoreflect.FileDescriptor` from the exact protobuf-go version jhump v1.16.0 depends on (`go.mod` pins `google.golang.org/protobuf v1.34.1`) — matches this project's pinned version exactly, no adapter needed. |

## Sources

- `$(go env GOMODCACHE)/github.com/jhump/protoreflect@v1.16.0/desc/protoparse/parser.go` (Parser struct, `ParseFiles`, `ParseFilesButDoNotLink`, `getResolver`) — HIGH confidence, primary source, read directly.
- `$(go env GOMODCACHE)/github.com/jhump/protoreflect@v1.16.0/desc/descriptor.go`, `desc/load.go` (`UnwrapFile`, `LoadFileDescriptor`) — HIGH confidence, primary source.
- `$(go env GOMODCACHE)/github.com/jhump/protoreflect@v1.16.0/dynamic/msgregistry/message_registry.go` (`MessageRegistry` mutex, `AddFile`) — HIGH confidence, primary source.
- `$(go env GOMODCACHE)/google.golang.org/protobuf@v1.34.1/reflect/protoregistry/registry.go` (`Files`, `Types`, `globalMutex` scoping, `RegisterFile`, `RangeFiles`, `MessageTypeResolver`/`ExtensionTypeResolver`) — HIGH confidence, primary source.
- `$(go env GOMODCACHE)/google.golang.org/protobuf@v1.34.1/reflect/protodesc/desc.go` (`Resolver`, `FileOptions.New`/`.NewFiles`) — HIGH confidence, primary source.
- `$(go env GOMODCACHE)/google.golang.org/protobuf@v1.34.1/encoding/protojson/{encode,decode}.go` (`Resolver` field contract) — HIGH confidence, primary source.
- `$(go env GOMODCACHE)/google.golang.org/protobuf@v1.34.1/internal/errors/errors.go` (deliberately unstable error string) — HIGH confidence, primary source.
- `$(go env GOMODCACHE)/github.com/bufbuild/protocompile@v0.10.0/resolver.go`, `compiler.go`, `linker/files.go` (`Resolver` thread-safety doc comment, `Compiler` struct fields, `linker.Resolver`) — HIGH confidence, primary source.
- `/Users/smintz/go/src/github.com/protoconf/protoconf/utils/utils.go`, `compiler/lib/parser/parser.go`, `compiler/service.go`, `compiler/lib/compiler.go` — HIGH confidence, this repository's own code, read directly to establish the concurrency hazard.
- `.planning/research/compiler-performance/BASELINE.md`, `OPTIONS.md` — HIGH confidence, this project's own prior measurements, used for cost figures cited above.

---
*Stack research for: lazy proto descriptor resolution (Protoconf v2.0)*
*Researched: 2026-09-04*
