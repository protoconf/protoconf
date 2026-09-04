# Architecture Research

**Domain:** Retrofitting lazy, load-driven proto resolution into Protoconf's existing compiler/registry stack
**Researched:** 2026-09-04
**Confidence:** HIGH — every claim below is derived directly from the current source (file:line cited); nothing here is external-ecosystem speculation.

## Standard Architecture — current state (eager)

```
                         ┌───────────────────────────────────────┐
                         │            ModuleService               │
                         │  (compiler/lib/module_service.go)      │
                         │                                         │
                         │  GetProtoRegistry()                    │
                         │   - if cachedRegistry != nil: return   │
                         │   - else: NewDescriptorRegistry()      │
                         │     + Load() each dep's .fds           │
                         │     + Import(Parse, src/) ← EAGER:     │
                         │       walks + parses + links ALL       │
                         │       .proto under src/ (4.6s/799)     │
                         │   - cache in m.cachedRegistry           │
                         └───────────────┬─────────────────────────┘
                                         │ *utils.DescriptorRegistry
                                         ▼
                         ┌───────────────────────────────────────┐
                         │         DescriptorRegistry              │
                         │            (utils/utils.go)             │
                         │  FileRegistry  map[path]*desc.FileDesc  │◄── memoized by Parse()/Import()
                         │  MessageRegistry msgregistry.Registry   │◄── already incremental (AddFile per parse)
                         │  localFiles    map[path]struct{}        │◄── reset every Parse() call; Store() target
                         └───────────────┬─────────────────────────┘
                                         │ GetFilesResolver() / GetTypesResolver()
                                         │  — ONE-SHOT SNAPSHOT over every entry
                                         ▼
                         ┌───────────────────────────────────────┐
                         │              Parser                     │
                         │     (compiler/lib/parser/parser.go)     │
                         │  FilesResolver *protoregistry.Files     │◄── built once from FileDescriptorSet
                         │  LocalResolver *protoregistry.Types     │◄── built once by ranging FilesResolver
                         │  FileDescriptors = registry.FileRegistry│
                         │                                          │
                         │  ParseFilesX(paths...)  — pure lookup   │
                         │  ReadConfig(file, msg)  — protojson     │
                         └───────────────┬─────────────────────────┘
                                         │  constructed once per consumer, then reused
                    ┌────────────────────┼─────────────────────────────────┐
                    ▼                    ▼                                 ▼
             compiler (CLI+daemon)  mutation server (daemon)     inserter / agent-filekv / mutate CLI
```

Separately, `ModuleService.Sync()` / `GenFileDescriptorSet()` builds its **own**, disposable
`utils.DescriptorRegistry` (module_service.go:399), never touching `m.cachedRegistry`, and calls
the eager `registry.Import(registry.Parse, ...)` → `registry.Store()` to write `.fds` cache files.
This separation already exists structurally today — the retrofit must preserve it, not introduce it.

### Component Responsibilities

| Component | Responsibility | Current shape | Must become |
|-----------|----------------|----------------|-------------|
| `ModuleService.GetProtoRegistry()` | Entry point every consumer calls to get a registry | Eager bulk parse of `src/`, cached once per `ModuleService` | Returns an empty-but-capable registry immediately; no bulk walk |
| `DescriptorRegistry.FileRegistry` | File-path → parsed descriptor cache | Filled once via `Import`/`Parse` over the whole tree | Filled on demand, one file (+ its transitive imports) at a time |
| `DescriptorRegistry.MessageRegistry` (jhump `msgregistry`) | Type-URL → message descriptor | **Already incremental** — `Parse()` calls `AddFile` per parsed file | No change needed |
| `DescriptorRegistry.localFiles` / `Store()` | What `mod sync` serializes to `.fds` | Reset on every `Parse()` call | Must stay untouched by the lazy path — see "mod sync coexistence" |
| `Parser.FilesResolver` / `Parser.LocalResolver` | Path/symbol lookup views handed to `protojson` and the Starlark loader | One-shot snapshot built at `NewParserWithDescriptorRegistry` | Long-lived, incrementally grown via `RegisterFile`/`RegisterMessage`/`RegisterEnum` (already-public, already-mutable stdlib APIs — no new type needed) |
| `Parser.ParseFilesX` | Choke point the Starlark loader calls for `load("//x.proto")` | Pure map lookup into pre-filled state | Same public signature; on miss, parses + registers, then returns |
| `Parser.ReadConfig` | Choke point all 5 daemon/CLI consumers call to read a materialized `ProtoconfValue` off disk | Single `protojson.Unmarshal` call with a fixed `Resolver` | Gains a cheap raw-JSON pre-pass for `proto_file`, see below |

## `GetProtoRegistry()` Consumer Table

Every call site, verified against source, with lookup mode and sensitivity classification.

| # | Consumer | Site | Lookup mode | CLI vs daemon | Latency sensitivity |
|---|----------|------|-------------|----------------|----------------------|
| 1 | Compiler construction | `compiler/lib/compiler.go:65` (`NewCompiler`) | **Path** (`load("//x.proto")` via `ParseFilesX`) **and** type URL (loadMutable, writeConfig's `protojson` resolver) | Both — `protoconf compile` (CLI, one-shot) and `devserver`/`compiler/service.go` (daemon, `Compiler` reused across concurrent `CompileFiles` calls) | **YES — this is the 200ms budget itself.** The only consumer on the milestone's hot path (OPTIONS.md "Blast radius"). |
| 2 | Compiler per-config load | `compiler/lib/compiler.go:355` (`c.load` → `messageRegistry: c.ModuleService.GetProtoRegistry().MessageRegistry`) | Type URL (feeds `config.messageRegistry`, used by validators) | Same as #1 | Re-fetches the already-cached registry (`m.cachedRegistry` short-circuit) — cheap regardless of laziness, not itself a parse trigger |
| 3 | Mutation server construction | `server/server.go:290` (`NewProtoconfMutationServer`) | Type URL — `GenReflectionUI` (server.go:570) walks `mutable_config/` at startup, resolving each file's `Any` via `LocalResolver.FindMessageByURL`; `FilesResolver` also seeded with the server's own gRPC-reflection/health/mutation protos | Daemon | Startup-time only (`GenReflectionUI` walks a typically-small `mutable_config/` tree once); **not** per-RPC — but the resulting `Parser` is then shared across concurrent RPCs, so its later mutation (lazy fill) must be concurrency-safe |
| 4 | Inserter construction | `inserter/inserter.go:220` (`NewProtoconfInserter`) | Type URL — `InsertConfigFile` (inserter.go:250) reads one already-materialized config per invocation via `ReadConfig` | CLI (`protoconf insert`, used by CI/CD) | One-shot per process; may process many files per CI run but sequentially within one process — not the 200ms target, but should not regress to the old 6.9s eager cost either |
| 5 | Agent filekv construction | `agent/filekv/filekv.go:80` (`New`) | Type URL — `Store.Get` (filekv.go:111) calls `ReadConfig` **per gRPC client request/watch tick** | Daemon | **Highest per-request sensitivity of the 4 non-compiler consumers** — this is the live config-serving path (`agent/kv_agent_impl.go:72,131` calls `store.Watch`/`store.Get` once per subscribed client and per change event) |
| 6 | Mutate CLI construction | `mutate/mutate.go:73` | **Symbol name** (`FindMessageByName` on a CLI-flag-supplied `--proto-msg`, not a type URL read from a document) | CLI, one-shot | Not sensitive — single invocation. Notably has **no `proto_file` hint available** (it isn't reading a `ProtoconfValue` envelope, it's building one), so it is the one consumer that structurally cannot use the `proto_file` pre-pass and would need either the package→directory fallback chain or a new `--proto-file` flag as a fast-path hint |
| 7 | Starlark loader, mutable-config load | `compiler/lib/starlark_loader.go:183` (`loadMutable`) | Type URL, resolved twice — once via `l.parser.LocalResolver.FindMessageByURL` (line 177) to unwrap the `Any`, once via `l.moduleService.GetProtoRegistry().MessageRegistry.FindMessageTypeByUrl` (line 187) to get a jhump `desc.Message` for the dynamic message bridge | Compiler (both CLI and daemon) | **YES** — this fires whenever a config does `load("//mutable/x", "value")`, i.e. inside the same hot compile path as #1 |

**Conclusion matching OPTIONS.md:** the compiler (#1, #2, #7) is the only path-driven, latency-budgeted
consumer. The other four (#3–#6) are all type-URL/symbol lookups reading already-materialized
documents — correctness-under-laziness work, not further speed work. Treat them as a
"must not regress" phase, not a "must get faster" phase.

## The `ReadConfig` ordering problem, traced

```go
// compiler/lib/parser/parser.go:69
func (p *Parser) ReadConfig(filename string, msg proto.Message) error {
    configReader, err := os.ReadFile(filename)
    ...
    return protojson.UnmarshalOptions{Resolver: p.LocalResolver}.Unmarshal(configReader, msg)
}
```

Every one of its 5 call sites (`starlark_loader.go:172`, `server.go:570`, `inserter.go:250`,
`filekv.go:111` — 4 confirmed, plus `mutate` builds its message differently) passes
`msg = &protoconf_pb.ProtoconfValue{}`. `ProtoconfValue` is:

```proto
message ProtoconfValue {
  string proto_file = 1;                 // which .proto produced the top-level message
  google.protobuf.Any value = 2;         // the actual payload
  ...
}
```

`protojson.UnmarshalOptions.Resolver` is fixed **before** `Unmarshal` sees a single byte. When the
unmarshaler reaches the `value` field's `"@type"` key, it calls `resolver.FindMessageByURL(url)`
synchronously, mid-parse — it cannot be told "wait, re-resolve after you've also seen `proto_file`."
Textual field order in the emitted JSON (protojson marshals in field-number order, so `proto_file`
usually precedes `value`) does not help: the `Resolver` interface has no hook for "I've already
decoded this other field, use it." This is the real ordering hazard — not file-system ordering, but
that the resolver is a static argument to a single black-box call.

### Viable integration shapes

**Shape 1 — cheap raw-JSON pre-pass (fast path).**
Before calling `protojson.Unmarshal`, run a plain `encoding/json` unmarshal into a struct that only
declares `proto_file`:
```go
var peek struct{ ProtoFile string `json:"proto_file"` }
_ = json.Unmarshal(raw, &peek)   // encoding/json ignores "value" — no Any semantics needed
```
This works precisely because `encoding/json` (unlike `protojson`) does not need to understand `Any`
to skip over an unrecognized/uninteresting field — it just treats `value` as an opaque object it
never asked for. Then lazily parse `peek.ProtoFile` and register it before the real
`protojson.Unmarshal` call. Cheap, exact, and correct for the **top-level** message only — this
resolves consumer rows #3, #4, #5, #7 in the common case.

**Shape 2 — lazy-on-miss resolver (structural fallback, required regardless).**
Wrap or extend `Parser.LocalResolver`'s `FindMessageByURL` so that a cache miss triggers, in order:
package name → directory (13/13 exact on the measured corpus, PITFALLS.md item 2) → scoped lexical
scan of that directory → loud eager fallback. This is **required independent of Shape 1** because
`ProtoconfValue.proto_file` only ever names the file of the *top-level* message — a nested `Any`
(via `starproto.AnyModule`, exercised by `field_type_any_test.pconf`) is structurally unreachable
from `proto_file`. Since `protojson`'s recursive descent into the top-level message's own fields
calls the *same* `FindMessageByURL` for any nested `Any` it encounters, one resolver implementing
Shape 2 transparently answers both the top-level lookup (if Shape 1 didn't already warm it) and
every nested one, with no additional pre-pass machinery.

**Shape 3 — do both (recommended).** Shape 1 is a strict optimization on top of Shape 2's fallback
chain: it turns the common, cheap, exact case (top-level type, `proto_file` present and correct)
into a single direct parse instead of a directory scan, while Shape 2 remains the only mechanism
that can ever answer a nested `Any` or a document with an absent/stale `proto_file`. Building only
Shape 1 leaves nested `Any` (a tested, first-class feature) broken under laziness. Building only
Shape 2 works but pays a directory-scan cost even on the common top-level case that a one-line JSON
peek would have answered directly.

**Two non-negotiables, regardless of shape (carried over from PITFALLS.md):** always verify the
resolved symbol actually exists in the parsed file rather than trusting the heuristic, and treat any
hint (`proto_file`, or a revived `proto_files` repeated hint) as a hint, not a source of truth — a
wrong hint must degrade to slower-than-necessary, never resolve to the wrong type.

## Where the lazy cache lives

**No new component.** Two existing structures already are the cache; they only need to stop being
filled once, up front:

1. **File-parse cache (by path): `DescriptorRegistry.FileRegistry`** (utils.go:34). Already a
   `map[string]*desc.FileDescriptor`, already used as a memo by `Parse()`/`Import()`'s
   `LookupImport` closure (utils.go:141 — it already checks `d.FileRegistry[s]` before parsing).
   The only change is *when* it gets filled: today `GetProtoRegistry()` forces one giant `Import`
   call over all of `src/`; lazily, it needs a single-file entry point that reuses the same
   `LookupImport`-driven transitive-resolution machinery `protoparse` already provides, without
   re-walking the whole `src/` tree first.
2. **Symbol/type view cache: `Parser.FilesResolver` / `Parser.LocalResolver`** (parser.go:20-21).
   `protoregistry.Files` and `protoregistry.Types` are not build-once-only types — `RegisterFile`,
   `RegisterMessage`, `RegisterEnum` are public, incremental APIs (this is literally how every
   generated `.pb.go`'s `init()` populates `protoregistry.GlobalFiles`/`GlobalTypes`). The fix for
   pitfall #3 is not a new "lazy view" type — it is constructing these two fields once, near-empty,
   and calling the existing registration calls (the same ones `DescriptorRegistry.GetTypesResolver`
   already contains, utils.go:87-108) incrementally from inside `ParseFilesX` on every cache miss,
   instead of batching them all at `NewParserWithDescriptorRegistry` time.

Why this boundary and not `DescriptorRegistry` alone, `Parser` alone, or a new type: every one of
the 7 consumer call sites already funnels through `NewParserWithDescriptorRegistry` →
`Parser.ParseFilesX`/`ReadConfig`. Keeping the cache on the types already sitting at that boundary
means zero signature changes propagate outward — `ParseFilesX`'s public shape is explicitly called
out in OPTIONS.md as needing to stay stable, and it does under this shape. A new "LazyResolver"
component would duplicate the map+registration logic that `DescriptorRegistry`/`protoregistry.*`
already provide, for no boundary benefit (ladder rung 2: reuse what's already there).

`DescriptorRegistry.MessageRegistry` (jhump's `msgregistry.MessageRegistry`, used by
`loadMutable`'s `FindMessageTypeByUrl`) needs **no change** — `Parse()` already calls `AddFile` per
individual parsed file (utils.go:166), so it is already correct under on-demand, one-file-at-a-time
parsing.

## Eager `mod sync` coexistence

The isolation the milestone requires already exists structurally and must simply not be undone:

- `ModuleService.Sync()` (module_service.go:399) constructs its own local
  `registry := utils.NewDescriptorRegistry()` — never `m.cachedRegistry`, never shared with any
  `Parser`. `GenFileDescriptorSet` then does `registry.Import(registry.Parse, ...)` (bulk, eager)
  followed by `registry.Store(cacheFile)`. This registry is discarded when `Sync()` returns.
- `GetProtoRegistry()` (the lazy path) constructs and caches a **different** registry instance in
  `m.cachedRegistry`, used by every `Parser`.
- As long as this stays two separate instances (it already is), pitfall #4's truncation risk cannot
  occur: `Store()` only ever serializes a registry that was filled by `Sync`'s own dedicated eager
  `Import`/`Parse` call over the full path list for that module, never by a lazy per-file load.

**The landmine to avoid when implementing the lazy single-file entry point:** `DescriptorRegistry.Parse()`
resets `localFiles = map[string]struct{}{}` on **every** call (utils.go:163). If the new lazy,
on-demand parse method is implemented by calling the existing `Parse()` once per file, then loading
proto A then proto B against the *same* registry instance would leave `localFiles` holding only
`{B}` — silently dropping A. This is harmless for `GetProtoRegistry()`'s lazily-filled registry
(nothing ever calls `Store()` on it), but it means the lazy entry point must be a **new** method that
populates `FileRegistry`/`MessageRegistry` directly (mirroring what `Parse()` does to those two maps)
without touching `localFiles` at all — `localFiles` is `Store()`'s bookkeeping and is meaningless
outside the eager `mod sync` path. Do not let the two code paths share that field.

## Concurrency

Every daemon consumer serves more than one request against a **single, long-lived** `Parser`
instance:

- `compiler/service.go`'s `CompilerService.CompileFiles` fans a `errgroup` out over every file in
  the request, each goroutine calling `CompileFileAsync` → `c.load` → `starlarkLoader.Load` →
  `loadProto` → **the same `c.parser.ParseFilesX`**. `devserver` holds one such `Compiler` for its
  whole lifetime.
- `agent/filekv.Store.Get` is called once per subscribed gRPC client and once per file-change event
  (`agent/kv_agent_impl.go:72,131`) — an agent with N active client streams calls `ReadConfig`
  against the **same `Store.parser`** N-times-concurrently.
- `ProtoconfMutationServer` similarly holds one `Parser` across all mutation RPCs.

Today this is safe because `ParseFilesX`/`ReadConfig` only ever **read** pre-built, immutable
snapshots. The moment `ParseFilesX` or the `ReadConfig` pre-pass can **write** (parse + register) on
a cache miss, every one of the maps and `protoregistry.*` instances above becomes a shared mutable
structure accessed from concurrent goroutines: `DescriptorRegistry.FileRegistry`,
`DescriptorRegistry.MessageRegistry`, `Parser.FilesResolver`, `Parser.LocalResolver`. None of these
are internally synchronized today (`utils.go`'s map writes and `protoregistry.Files.RegisterFile`
are not safe for concurrent use without external locking).

**Required:** a `sync.RWMutex` (or single `sync.Mutex` — the write path already dominates cost, an
`RWMutex`'s benefit is marginal once cache hit-rate is high, but it's the correct default) guarding
the check-then-fill sequence in `ParseFilesX` and in whatever new `DescriptorRegistry` method backs
it. Also flag, while touching this code: `ModuleService.GetProtoRegistry()`'s
`if m.cachedRegistry != nil { return }` check (module_service.go:349) is itself an unsynchronized
read-then-write today — benign under the old eager design because the race window is "two goroutines
both eagerly parse everything, one result gets discarded," but worth locking properly rather than
carrying that latent race into a codebase that is about to get much more concurrency-sensitive in
this exact area.

Inserter (#4) and mutate CLI (#6) are effectively single-goroutine per process invocation — locking
costs them nothing measurable, so apply the same primitive uniformly rather than special-casing
CLI-vs-daemon.

## Component boundaries — what changes, what stays

| Component | Changes | Stays the same |
|-----------|---------|-----------------|
| `ModuleService.GetProtoRegistry()` | Stops bulk-parsing `src/`; returns a registry configured with import paths but empty of file content | Public signature; caching in `m.cachedRegistry`; separateness from `Sync()`'s registry |
| `DescriptorRegistry` | Gains a synchronized, single-file on-demand parse method that does not touch `localFiles`; `FileRegistry`/`MessageRegistry` gain a mutex | `Import`/`Parse`/`Store`/`Load` (used unchanged by `mod sync`); `MessageRegistry`'s existing per-file `AddFile` behavior |
| `Parser` | `FilesResolver`/`LocalResolver` become incrementally grown instead of snapshot-once; `ParseFilesX` gains a parse-and-register path on miss, guarded by a mutex; `ReadConfig` gains the `proto_file` pre-pass | Public signatures of `ParseFilesX` and `ReadConfig`; `NewParserWithDescriptorRegistry`'s call shape |
| `starlark_loader.go` | None required by this work (its `loadValidators` filesystem-walk fix already shipped separately) | `loadProto`, `loadMutable` call the same `Parser` methods, unaware of laziness underneath |
| 5 consumer packages (compiler, server, inserter, filekv, mutate) | None — they keep calling `ms.GetProtoRegistry()` / `parser.NewParserWithDescriptorRegistry` / `ReadConfig` exactly as today | Everything else |
| `ModuleService.Sync()` / `GenFileDescriptorSet` | None | Stays fully eager, on its own disposable registry instance |

## Data flow — compile path

**Before (eager):** `protoconf compile` → `NewCompiler` → `GetProtoRegistry()` blocks for 4.6s
parsing/linking all 799 protos → `NewParserWithDescriptorRegistry` blocks another 260ms snapshotting
resolvers over 864 files → `CompileFile` (184ms: Starlark eval, `ParseFilesX` = pure lookup, write).
Total 6.97s, dominated by work the specific config never asked for.

**After (lazy):** `protoconf compile` → `NewCompiler` → `GetProtoRegistry()` returns near-instantly
(registry configured, nothing parsed) → `NewParserWithDescriptorRegistry` builds near-empty
resolvers → `CompileFile`'s Starlark `load()` statements drive `ParseFilesX` to parse+register only
the ~5 directly-loaded + ~7 transitively-imported protos (measured at 2.7ms in isolation) →
`loadMutable` (if the config reads a mutable value) drives the `ReadConfig` pre-pass + fallback chain
to resolve that one `Any`. Target ~200-210ms total, dominated by the already-at-budget 184ms
Starlark/validate/write stage.

## Data flow — materialized-config read path

**Before (eager):** server/inserter/filekv/mutate each construct a `Parser` backed by a fully-linked
799-file registry, then call `ReadConfig` → `protojson.Unmarshal` resolves the `Any` against the
already-complete `LocalResolver` — correct, but paid the same 4.6s+260ms cost at construction that
the compiler paid, for a consumer that will only ever look at a handful of already-materialized
files.

**After (lazy):** construction is near-instant (same registry change as the compile path). Each
`ReadConfig` call: (1) raw-JSON pre-pass extracts `proto_file`, lazily parses+registers that one
file if not already cached; (2) `protojson.Unmarshal` proceeds; if it hits a nested `Any` the
pre-pass didn't cover, the resolver's on-miss fallback (package → directory → scoped scan → loud
eager fallback) resolves it inline, verifying the symbol before returning it.

## Suggested build order

Two prerequisite items from PITFALLS.md's own ordering are **already shipped** per PROJECT.md and
are not phases of this work: the `loadValidators` filesystem-walk fix (quick 260904-f5j) and the
pinned synthetic benchmark corpus + scaling gate (quick 260904-fwk). The remaining work:

1. **Concurrency-safe lazy `DescriptorRegistry` core.** Add locking to `FileRegistry`/
   `MessageRegistry`; add the new on-demand single-file parse method that does not touch
   `localFiles`; change `GetProtoRegistry()` to stop bulk-walking `src/`. Pure `utils.go` +
   `module_service.go` change. No dependency on anything else in this list — do this first, it is
   the foundation everything below calls into.

2. **Lazy resolver views on `Parser` (resolves pitfall #3).** Seed `FilesResolver`/`LocalResolver`
   near-empty at construction; make `ParseFilesX` parse-and-register on miss, under a mutex.
   *Depends on (1)* for the on-demand parse primitive. This phase alone delivers the milestone's
   headline win — the compiler is the only consumer on the 200ms hot path, and this phase is
   sufficient to get it there (path-based `load()` resolution, plus `loadMutable`'s type-URL
   resolution once the `ReadConfig` pre-pass from phase 3 lands — note `loadMutable` is compiler-side
   too, so phase 2 and phase 3's compiler-relevant slice are tightly coupled; sequence them together
   if the roadmap wants the full compile-path win in one phase).

3. **`ReadConfig` pre-pass + type-URL fallback chain for the 4 daemon/CLI consumers.** Add the
   raw-JSON `proto_file` peek to `Parser.ReadConfig`; add the package→directory→scoped-scan→loud-
   fallback chain as the `FindMessageByURL` miss behavior. *Depends on (2)* — this phase adds more
   *ways* to trigger the same growable-resolver mechanism, it doesn't need a different one. Frame
   this phase's success criterion as "no regression, no silent wrong answers" (server, inserter,
   filekv, mutate were never on the 200ms budget) rather than a further speed target.

4. **Decide and document the error-surface change; flip the scaling gate.** Decide whether
   `protoconf compile` should keep validating the whole `src/` tree for broken imports (optionally
   behind a flag) now that a lazy compile will never touch an unreached broken proto, and write the
   decision down rather than letting CI behavior silently change. Flip
   `TestCompilerStartupScaling`'s `t.Skipf` to `require.LessOrEqual` — this is the milestone's stated
   definition of done, and it can only happen once (1)-(3) exist to measure.

## Anti-patterns to avoid

### Anti-Pattern 1: Snapshot-plus-lazy-source

Building `Parser.FilesResolver`/`LocalResolver` once from whatever is in `DescriptorRegistry` at
construction time, then letting `DescriptorRegistry` keep growing underneath it. This is exactly
pitfall #3: "works the first time, wrong the second." The fix is not a refresh-on-read wrapper — it
is making the resolver instances themselves grow via their own public `RegisterFile`/`RegisterMessage`
API, so there is only ever one instance, never a stale copy.

### Anti-Pattern 2: Routing the lazy single-file parse through `DescriptorRegistry.Parse()`

`Parse()` resets `localFiles` on every call (pitfall #4). Reusing it for the lazy path silently
breaks nothing *today* (nothing calls `Store()` on the lazily-filled registry) but plants a landmine
for the next person who tries to share a registry instance between an eager and lazy caller. Give
the lazy path its own method that never touches `localFiles`.

### Anti-Pattern 3: Building a new "LazyResolver" interface/type

Everything needed already exists on `DescriptorRegistry` (a mutable map) and `protoregistry.Files`/
`Types` (public, incremental registration APIs already used by every generated `.pb.go`). A new
wrapper type would duplicate that state and add a translation layer for no boundary benefit, while
also being one more thing every one of the 7 consumer call sites would need to know about.

## Integration boundaries

| Boundary | Communication | Notes |
|----------|----------------|-------|
| `ModuleService` ↔ `DescriptorRegistry` | Direct struct field + method calls (`registry.Import`, `.Load`, `.Store`) | Two independent registry instances must keep existing side by side: `m.cachedRegistry` (lazy) and `Sync()`'s local one (eager) |
| `DescriptorRegistry` ↔ `Parser` | `NewParserWithDescriptorRegistry(registry)` at construction only | Must stop being a one-time snapshot call; the two need an ongoing relationship (registry does the parsing, `Parser` does the registering) for the resolver's lifetime |
| `Parser` ↔ `starlark_loader.go` | `l.parser.ParseFilesX(...)`, `l.parser.ReadConfig(...)`, `l.parser.LocalResolver.FindMessageByURL(...)` | Unaware of laziness underneath — this is the value of keeping `ParseFilesX`'s signature stable |
| `Parser` ↔ 5 consumer packages | `parser.NewParserWithDescriptorRegistry(ms.GetProtoRegistry())` then `.ReadConfig`/`.LocalResolver` | Identical call shape before and after; only construction cost and cache-miss behavior change |

## Sources

- `compiler/lib/module_service.go` (this repo)
- `utils/utils.go` (this repo)
- `compiler/lib/parser/parser.go` (this repo)
- `compiler/lib/starlark_loader.go` (this repo)
- `compiler/lib/compiler.go` (this repo)
- `compiler/service.go`, `server/server.go`, `inserter/inserter.go`, `agent/filekv/filekv.go`,
  `agent/kv_agent_impl.go`, `mutate/mutate.go` (this repo)
- `.planning/research/compiler-performance/OPTIONS.md`, `.planning/research/compiler-performance/PITFALLS.md` (prior milestone research, this repo)
- `.planning/PROJECT.md` (milestone definition, this repo)

---
*Architecture research for: Protoconf v2.0 lazy proto resolution retrofit*
*Researched: 2026-09-04*
