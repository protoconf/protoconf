# Pitfalls Research: Retrofitting Lazy Loading into an Eager Descriptor Registry

**Domain:** Go / Protocol Buffers / Starlark config compiler — converting `ModuleService.GetProtoRegistry()` from an eager, filesystem-walk-at-startup registry to a lazy, load()-driven one.
**Researched:** 2026-09-04
**Confidence:** HIGH — every finding below is grounded in file:line references from this codebase, not hypothesized. Two items (protovalidate extension safety, output-order determinism) are "verified not broken" findings, included because the question asked to investigate them, not because they need fixing.

This extends `.planning/research/compiler-performance/PITFALLS.md`. It does not restate that file's items — it deepens the investigation the milestone context asked for: **where else does code assume the registry is complete, and what happens at the trust boundaries (extensions, ordering, concurrency, backward compat) once it isn't.**

**Status check on the existing doc:** its item 1 (`loadValidators` ranging the registry) is already fixed in this tree — `compiler/lib/starlark_loader.go:112-147` now does `filepath.WalkDir(l.srcDir, ...)` for `*.proto-validator` files, not `RangeFiles`. Its items 2-6 stand. This document adds five more registry-completeness assumptions and consumer-specific risks that the "make the registry lazy" work will hit.

## Critical Pitfalls

### Pitfall 1: The mutation server's dynamic gRPC service registration ranges the registry exactly once, at startup, before anything has loaded

**What goes wrong:**
`server/server.go:324-389`, `ProtoconfMutationServer.Init()`:

```go
s.parser.FilesResolver.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
    ...
    for i := 0; i < fd.Services().Len(); i++ {
        ...
        rpcServer.RegisterService(svcDesc, s)   // line 386
    }
    return true
})
```

This is how protoconf turns a `service` defined in a `src/*.proto` file into a live gRPC method the mutation server actually serves — it walks every file in the registry, finds services whose methods return `ConfigMutationResponse`, and calls `grpc.Server.RegisterService` for each. `Init()` is called exactly once, at process startup, in `devserver/command.go:82` and in the standalone `cmd/server` — **before any config has been compiled and before `grpc.Server.Serve()` runs.** `grpc.Server.RegisterService` panics if called after `Serve()` has started, so there is no natural place to call it again later.

Under a lazy registry, at the moment `Init()` runs, nothing under `src/` has been demanded yet — the registry holds only the handful of files explicitly pre-registered two lines above it (`grpc_reflection_v1`, `grpc_health_v1`, the mutation and protoconf-v1 protos themselves). **Every custom mutation service defined anywhere under `src/` silently fails to register**, for the entire lifetime of the process. This is worse than the already-fixed `loadValidators` bug: that one degrades one config's validation; this one degrades an entire serving surface, permanently, with no per-request recovery path.

**Why it happens:**
The existing eager design conflates two different lifecycles: "parse the protos this config needs" (per-compile, naturally lazy) and "discover every service definition in the repo so the server can expose it" (a one-time, whole-repository enumeration that has to happen *before* traffic starts). Making `GetProtoRegistry()` lazy fixes the first and breaks the second, because both currently read from the same registry instance.

**How to avoid:**
This is not "make it faster," it needs a different mechanism entirely: either (a) keep this one enumeration eager — walk `src/*.proto` explicitly (mirroring the `loadValidators` fix: filesystem walk, not registry range) and parse only the files needed to discover `service` declarations, which is a much smaller and structurally different scan than parsing every message for linking; or (b) defer `Init()`'s service-registration decision until first compile, requiring an eager one-time parse before `Serve()` (acceptable — this happens once at startup, not per-request, so it doesn't reintroduce the per-compile 4.6s cost). Do not let this consumer share a lazy, demand-driven registry silently — it has no way to "catch up" the way an interactive compile does.

**Warning signs:**
Devserver or mutation server logs "Registering service" for zero or fewer services than `src/` actually defines. `protoc --decode` / `grpcurl list` against a running server shows the reflection service exists but a known custom mutation RPC is absent. This will not error — grpc clients calling the missing method get a normal `Unimplemented` status, indistinguishable from "was never built."

**Phase to address:**
Same phase that makes the registry lazy, as a blocking co-requirement — not a follow-up. This consumer must be re-pointed at an explicit eager path *before* `GetProtoRegistry()` changes semantics, or the mutation server regresses silently the day the switch flips.

**Test:**
Fails-before-fix. Write a fixture `src/` with a custom `service` returning `ConfigMutationResponse`. Start a `ProtoconfMutationServer`, call `Init()`, then assert (via `grpc.Server`'s internal service info, or a live `grpcurl`-style dial) that the service is registered — **before compiling any config that references that proto.** Today (eager registry) this passes trivially because `Init()` sees everything. The moment the registry is made lazy without fixing this consumer, this test fails, because `Init()` runs before any config load.

---

### Pitfall 2: `GetFilesResolver`/`GetTypesResolver` are one-shot conversions of a `map[string]*desc.FileDescriptor` — and the "same map" illusion of safety

**What goes wrong:**
Extends compiler-performance PITFALLS.md item 3 with a subtlety that will mislead whoever implements the lazy path.

`compiler/lib/parser/parser.go:25-32`:
```go
func NewParserWithDescriptorRegistry(registry *utils.DescriptorRegistry) *Parser {
    files := registry.GetFilesResolver()
    return &Parser{
        FilesResolver:   files,
        LocalResolver:   registry.GetTypesResolver(files),
        FileDescriptors: registry.FileRegistry,   // <-- same map reference, not a copy
    }
}
```

`Parser.FileDescriptors` is the literal `registry.FileRegistry` map (Go maps are reference types). `ParseFilesX` (`parser.go:34-67`) checks this map *first*: `if fd, ok := p.FileDescriptors[filename]; ok { ... }`. Because it's the same map object, entries a lazy-fill mechanism adds directly to `registry.FileRegistry` after `Parser` construction *are* visible here — this one lookup path is accidentally safe under lazy loading, for exact-path hits.

But `FilesResolver` (`*protoregistry.Files`) and `LocalResolver` (`*protoregistry.Types`) are **not** aliases — they are separately-constructed objects built once from a `FileDescriptorSet` snapshot at that moment in time (`utils.go:78-108`). A file added to `registry.FileRegistry` after `Parser` construction is invisible to `p.FilesResolver.FindFileByPath` and to `p.LocalResolver.FindMessageByURL` — this is `ParseFilesX`'s fallback path (line 40) and the only path used by `loadMutable`'s type-URL resolution (`starlark_loader.go:177`) and `config.go`'s `protoResolver`/`messageRegistry` validation path.

The danger is specifically that the "safe" map-aliasing lookup will make ad hoc manual testing look fine (direct-path loads of already-known files work), while the resolver-based lookups — the ones that matter for `Any` and mutable-config type resolution — stay silently stale. Someone debugging this will find the map lookup working and wrongly conclude the resolver problem doesn't exist.

**Why it happens:**
Two different data representations (`desc.FileDescriptor` map for jhump/protoparse, `protoregistry.Files`/`protoregistry.Types` for google.golang.org/protobuf-based consumers) are synchronized only at one moment — `Parser` construction — and nothing keeps them in sync afterward.

**How to avoid:**
Whatever "resolvers become lazy views" mechanism is built for item 3 must wrap **both** `FilesResolver` and `LocalResolver`, not just fix the registry's own map. Prefer a thin wrapper type implementing `protodesc.Resolver` / the type-resolution interfaces that, on miss, triggers the same lazy-fill-by-path used by `ParseFilesX`, then retries — rather than trying to keep a `protoregistry.Files` "live" (it has no incremental-insert API; it's build-once by design in google.golang.org/protobuf).

**Warning signs:**
"Works when I `load()` it directly, fails when the same type is reached via `Any`/mutable config" is the exact symptom shape. Any bug report of the form "second lookup of the same type fails" or "works the first time, wrong the second" (already predicted in the base doc) confirms this.

**Phase to address:**
The "resolvers as lazy views" phase (already called out in PROJECT.md's active requirements). Explicitly scope it to cover `FilesResolver` and `LocalResolver` as a matched pair, and write the test against `LocalResolver.FindMessageByURL`, not the aliased map, or the fix will look complete while the real gap remains.

**Test:**
Fails-before-fix. Load proto A directly (`load("//a.proto", ...)`), then reference a type from proto B only via an `Any` payload without ever `load()`-ing B directly. Assert `parser.LocalResolver.FindMessageByURL` resolves B's type. Today this passes because the registry (and therefore the resolver snapshot) is fully eager. Under a naive lazy registry with unfixed resolvers, B is never parsed and this fails with "not found."

---

### Pitfall 3: `GenReflectionUI`'s periodic walk is a second, recurring consumer of the exact same stale-resolver problem — on a 5-second ticker

**What goes wrong:**
`server/server.go:560-596`, called from `devserver/command.go:89-97` every 5 seconds via `time.NewTicker(5 * time.Second)`:

```go
filepath.WalkDir(filepath.Join(s.protoconfRoot, consts.MutableConfigPath), func(path string, info fs.DirEntry, err error) error {
    ...
    err = s.parser.ReadConfig(path, value)                                  // protojson unmarshal using LocalResolver
    mt, err := s.parser.LocalResolver.FindMessageByURL(value.GetValue().GetTypeUrl())
    ...
})
```

This walks every file under `mutable_config/` and, for each one, resolves its `Any` payload's type URL through the same `s.parser.LocalResolver` that Pitfall 2 shows is a point-in-time snapshot. It is not documented anywhere in the base PITFALLS.md, and it's a *recurring* job in a long-running devserver process — meaning it doesn't just fail once at startup like Pitfall 1, it fails silently and repeatedly, every 5 seconds, logging `"error finding message"` at `Error` level (`server.go:575`) that nobody watching stdout at debug-off log levels will notice, since the loop swallows the error for that one file (`return nil` on `ReadConfig` failure at line 571, but `return err` on `FindMessageByURL` failure at line 576 — inconsistent handling, worth noting on its own: one failure mode aborts the whole `WalkDir`, the other skips just that file).

**Why it happens:**
Same root cause as Pitfall 2 — `s.parser` is a single shared object built once at `NewProtoconfMutationServer` construction, and this consumer was written assuming (correctly, today) that the registry backing it is complete.

**How to avoid:**
Route this through the same `proto_file`-driven lazy-load chain the base doc proposes for `loadMutable`/inserter/filekv — `ProtoconfValue.proto_file` is already being read off disk here via `ReadConfig`, so the pre-pass (pull `proto_file` before the type-dependent unmarshal, lazily load it, then resolve) fixes this consumer for free once built as a shared helper rather than duplicated per-consumer. Do not let this ship as a fifth bespoke implementation of the same pattern — factor it once (e.g., `parser.ResolveAnyType(protoFile, typeURL)`) and route all five `GetProtoRegistry()` consumers through it, since `server.go`, `inserter.go`, `filekv.go`, `mutate.go`, and `starlark_loader.go`'s `loadMutable` all currently duplicate the "read config, resolve type URL" sequence independently.

**Warning signs:**
`grep -c "error finding message"` in devserver logs climbing steadily; the reflection UI's example list silently shrinking as more mutable configs use types not directly reachable from what's been compiled since server start.

**Phase to address:**
Same phase as Pitfall 2/item 2 (the `proto_file`-driven resolution work) — this is a fifth call site for that fix, not a separate problem. Add it to the consumer inventory explicitly so it isn't missed; PROJECT.md's "type-URL resolution across all five registry consumers" line should be read as covering this, but it isn't named yet anywhere.

**Phase to address (test):**
Characterization-only as an isolated unit — the fix is identical to Pitfall 2's, so a dedicated failing test here is redundant with Pitfall 2's test *if* the fix is factored into one shared helper (recommended). If the fix is instead duplicated per-consumer (not recommended), then this needs its own fails-before-fix test: materialize a mutable config whose `Any` type isn't in the initial registry, run `GenReflectionUI`, assert no `"error finding message"` log and that the example appears.

---

### Pitfall 4: `ModuleService.cachedRegistry` is a process-lifetime singleton — one eager fallback anywhere poisons every future compile silently

**What goes wrong:**
`compiler/lib/module_service.go:348-362`:
```go
func (m *ModuleService) GetProtoRegistry() *utils.DescriptorRegistry {
    if m.cachedRegistry != nil {
        return m.cachedRegistry
    }
    registry := utils.NewDescriptorRegistry()
    ...
    m.cachedRegistry = registry
    return registry
}
```

`devserver` and `CompilerService` construct exactly **one** `Compiler` (and therefore one `ModuleService`, one `cachedRegistry`) for the lifetime of the process (`compiler/service.go:21-29`, `devserver/command.go:63`). If any lazy-fill mechanism ever falls back to an eager full-repository import — even once, even for one unusual config — and that fallback is implemented as "just call the old eager `Import()` path on this same registry," the *entire* registry becomes fully populated and stays that way for every subsequent compile the process ever handles, because it's cached. From that point on, every later compile looks correctly lazy in isolation (small alloc count, no new parsing needed — it's all already there) while the process as a whole silently paid the full 4.6s cost once and now perpetually holds ~864 parsed files in memory it may never need again.

This is the single most dangerous shape for the item-6 "silent eager fallback" concern in the base doc: a scaling benchmark that constructs a **fresh** `Compiler` per iteration (as `TESTING.md`'s proposed benchmark shape does — `lib.NewCompiler(root, false)` inside the loop) will never observe this, because each iteration gets a fresh, unpoisoned `ModuleService`. The regression is invisible to exactly the test suite designed to catch regressions, and only shows up in the long-running `devserver`/mutation-server processes the benchmark doesn't model.

**Why it happens:**
Caching the registry per-`ModuleService` was a correct optimization under the eager model (compute once, reuse forever, since "once" already meant "everything"). It becomes a trap the moment "the registry" can be partially populated, because now "cache it" means "cache whatever state it happened to be in when someone last touched it," and any code path that widens it widens it permanently.

**How to avoid:**
Instrument, don't just implement: expose a counter (`registry.ParsedFileCount()` or similar, already half-suggested in TESTING.md's "extend the one log line" idea) and add a **long-running-process test**, not just a construct-once benchmark: build one `Compiler`, compile config A (small `load()` graph), assert file count stays small, compile config B, assert it only grows by B's incremental needs — never jumps to "everything." This is the test the current benchmark shape cannot express and must be added alongside it, not instead of it.

**Warning signs:**
Devserver memory/file-count metrics that start low and jump to a plateau matching total `src/` proto count after one specific request, then never come back down; a scaling benchmark staying green while a live devserver's first-request latency (not steady-state) spikes.

**Phase to address:**
The lazy-by-path registry phase itself — add the "long-running process, multiple compiles, no single request re-widens the cache" test as an explicit acceptance criterion for that phase, distinct from the scaling-ratio test (which only exercises fresh-process-per-compile).

**Test:**
Characterization-only until the lazy registry exists (there's nothing to characterize yet — today's registry is always fully eager by design, so "poisoning" isn't meaningful pre-fix). Once lazy loading lands, this becomes fails-before-fix for any fallback implementation that reuses the shared registry: construct one `Compiler`, compile a config that forces the eager fallback (e.g., an `Any` type with an unreadable `proto_file`), then compile a second, unrelated small config, and assert the second compile's newly-parsed-file count is small — not "already satisfied because everything is loaded."

---

### Pitfall 5: Concurrent compiles share one `Parser`/registry with zero synchronization — safe today only because nothing mutates after construction

**What goes wrong:**
`compiler/service.go:31-57`, `CompilerService.CompileFiles`, runs one goroutine per file in the request via `errgroup`, all calling `s.Compiler.CompileFileAsync` concurrently against the **same** `*lib.Compiler`. `compiler/lib/compiler.go:360-370`'s `GetLoader()` allocates a fresh `starlarkLoader` (and its own `cache` map) per call — that part is already goroutine-safe — but every loader shares `c.parser` (`compiler/lib/parser/parser.go`'s single `*Parser`, holding the aliased `FileDescriptors` map, `FilesResolver`, `LocalResolver`) and `c.ModuleService` (single `cachedRegistry`).

Today this is safe by construction: `GetProtoRegistry()` runs once, synchronously, at `NewCompiler()` time, before any concurrent `CompileFileAsync` call exists — so every subsequent access from concurrent goroutines is a **read-only** access to fully-populated maps, and concurrent map reads without writes are safe in Go. The moment lazy loading requires writing into `registry.FileRegistry` / `registry.MessageRegistry` / `registry.localFiles` **during** `CompileFileAsync` (which is the entire point of "load-driven"), those writes happen from multiple goroutines with no mutex around `utils.DescriptorRegistry`'s maps (`utils/utils.go` — `Import`, `Parse`, `MergeFileDescriptorSet` all write `d.FileRegistry[...]` unguarded) or around `ModuleService.cachedRegistry`'s check-then-set (`module_service.go:349-350`, no lock, unlike the `mutex sync.RWMutex` that already exists on the struct — that mutex protects the module-dependency `Walk`/`Lock` path, not the registry).

This is Go's classic "fatal error: concurrent map writes" territory — a **crash**, not a silently-wrong answer, but a crash triggered by ordinary concurrent traffic (a single multi-file `CompileFiles` streaming RPC, or two concurrent RPCs against one long-lived `devserver`/`CompilerService` process) that the current single-threaded-eager design has never had to survive.

**Why it happens:**
The eager design's implicit invariant — "the registry never changes after `NewCompiler` returns" — is exactly what "load-driven" lazy loading has to violate. Nothing in `CompilerService.CompileFiles`'s errgroup fan-out was written with that invariant in mind because it never had to be.

**How to avoid:**
Whatever lazy-fill mechanism gets built needs its own synchronization independent of `ModuleService.mutex` (which serves a different purpose — module/dependency metadata, not the descriptor maps) — either a dedicated `sync.RWMutex` around `DescriptorRegistry`'s maps with a fast read-path check before taking the write lock (classic double-checked pattern, matching the existing `cachedRegistry != nil` early-return shape but made actually safe), or a `sync.Map`/`singleflight` per file path so concurrent demands for the same not-yet-loaded proto coalesce into one parse instead of racing. `singleflight` (already a dependency-free stdlib-adjacent choice — `golang.org/x/sync/singleflight`, and this repo already imports `golang.org/x/sync/errgroup` from the same module) is the natural fit: it solves both the crash and the "N goroutines redundantly parsing the same cold file" performance question at once.

**Warning signs / what `-race` catches vs. misses:**
`go test -race` **will** catch this reliably, but only in a test that actually drives two-plus concurrent `CompileFileAsync`/`CompileFile` calls against one shared `Compiler` where at least one of them needs to lazy-load a file the other hasn't already touched — no such test exists in this repo today (confirmed: no `_test.go` file references `CompileFiles` or `errgroup`). Absent that test, this ships as a runtime panic discovered first in production/devserver under real concurrent load, not in CI. `-race` will *not* help if the only tests exercising concurrency are single-file compiles or use a pre-warmed (already-eager) registry, since there'd be no concurrent write to detect.

**Phase to address:**
The lazy-by-path registry phase must include a concurrency-safety mechanism as a hard requirement, not an afterthought — and the phase's test suite must add the first-ever concurrent-compile test in this codebase.

**Test:**
Fails-before-fix (once lazy loading exists without synchronization) / does-not-yet-exist-to-fail (today, because the registry can't be mutated post-construction at all, so there's nothing to race on). Add: one `Compiler` over a corpus with N≥20 distinct protos, launch ≥2 goroutines each compiling a *different* config that `load()`s a disjoint proto not yet touched by the other, run under `go test -race`, assert no race and no panic. This test is meaningless against today's eager registry (trivially passes, proves nothing) — its value is entirely as a regression gate for the lazy implementation, so it should be added in the same commit that introduces lazy-fill, not before.

---

### Pitfall 6: `add_validator`'s last-write-wins map is a pre-existing, load-order-sensitive bug that the lazy-loading work must not accidentally re-randomize

**What goes wrong:**
`compiler/lib/starlark_functions.go:36-63`, `starAddValidator`: `(*mp)[messageName] = validator` — unconditional overwrite, no check for an existing entry. Two `.proto-validator` files registering a validator for the same message's fully-qualified name silently keep only the one that ran last; the other's cross-field validation logic is dropped with no warning, no error, no log.

This is **not new** and **not caused by making the registry lazy** — it already exists, and it's already been characterized (not fixed) in `compiler/lib/starlark_loader_test.go:34-63`, whose comment block explicitly documents it: "`add_validator`'s map keeps only the most recently registered function per message... Walk order is now deterministic... putting this fixture last guarantees its always-failing rule is the one actually enforced." That test pins today's *walk order* (lexical filesystem order, from the already-fixed `loadValidators`) as the mechanism that makes which-validator-wins deterministic — but "deterministic" here means "predictable," not "correct." Two legitimate validators for the same message will still silently clobber each other; the test only proves the clobbering is reproducible.

**Why this belongs in this document:** the milestone context explicitly calls this out as a latent bug "three test fixtures pass for the wrong reason," and the risk specific to *this* milestone is that lazy loading changes proto *file* load order (demand-driven, following each config's `load()` graph) while validator *file* discovery stays a separate, already-fixed filesystem walk (`loadValidators`, lexical order, independent of the proto registry). These are now two decoupled orderings where there used to be one. As long as nobody re-couples them (e.g., by "optimizing" `loadValidators` to iterate load()-order instead of filesystem order because it looks more consistent with the new lazy proto loading), the existing test's pinned behavior survives unchanged. The risk is a well-intentioned refactor re-introducing registry-order dependence into the one place that was just fixed to not have it.

**How to avoid:**
Don't fix the map-clobbering bug as part of this milestone (out of scope — it's a correctness bug independent of laziness, and the existing test currently asserts the clobbering as expected behavior, so "fixing" it would be a breaking behavior change requiring its own decision, not a side effect of a performance milestone). Do add a one-line guard against regression: keep `loadValidators`' filesystem walk order explicitly documented as intentionally decoupled from proto load order, so a future contributor doesn't "simplify" it back into a registry range.

**Warning signs:**
A refactor PR that touches `loadValidators` and reintroduces `RangeFiles`/registry iteration anywhere in its implementation.

**Phase to address:**
No new phase needed — a one-line comment in `loadValidators` (or a code-review checklist item for the lazy-registry phase) stating "do not couple validator discovery order to proto registry order" is sufficient. If `add_validator`'s clobbering itself is ever fixed (e.g., error on duplicate registration), that's a separate, out-of-milestone decision — flag it as a backlog item, not a v2.0 blocker.

**Test:**
Characterization-only. `TestLoadValidators_OrphanValidatorIsDiscovered` already exists and already encodes the "last-registered-wins, walk-order-is-lexical" behavior. No new test needed for the lazy-loading work itself; the existing test is the guard. (A hypothetical future "reject duplicate `add_validator` for the same message" fix would need its own new fails-before-fix test, but that's out of scope here.)

---

## Verified — Investigated and Found Not Broken (include so planning doesn't over-scope)

### Custom option / extension interpretation (`buf.validate.field`, `validate.field`) is unaffected by `src/` laziness — verified against this codebase, not assumed

The question raised a real concern: does `protovalidate-go` silently skip field constraints if the file declaring the `buf.validate.field` extension isn't in the registry at validation time? Traced end to end:

1. `utils/utils.go:38-45`, `NewDescriptorRegistry()` seeds `FileRegistry` from **`protoregistry.GlobalFiles`** (Go's process-wide compiled-in proto registry, populated by the blank imports `_ "github.com/bufbuild/protovalidate-go"`, `_ "github.com/bufbuild/protovalidate-go/legacy"`, `_ "github.com/protoconf/protoconf/pb/protoconf/v1"` present in both `utils.go` and `parser.go`), filtered by `globalRegexMatcher = (google|google/rpc|google/type|buf/validate|validate|protoconf/v1)/(.*)\.proto`. **This seeding is unconditional and happens before any `src/` import** — it does not depend on, and is not touched by, whatever lazy-by-path mechanism gets built for `src/`.
2. `protoparse`'s `LookupImport` (`utils.go:141-152`) finds `buf/validate/validate.proto` in `d.FileRegistry` for any `src/*.proto` file that `import`s it, regardless of whether the registry is otherwise lazy — because normal `import` statement following inside a single file's parse is unrelated to the "walk the whole repo" completeness problem that the rest of this document is about. A field with `(buf.validate.field) = {...}` cannot exist in a file that doesn't import `buf/validate/validate.proto`, so the file needed to interpret it is always structurally present by the time that file is parsed.
3. This generalizes to project-authored custom extensions too: a hypothetical `src/myoptions/options.proto` declaring its own `extend google.protobuf.FieldOptions` and used by another `src/` proto works under lazy loading for the same reason — the annotated file's own `import` pulls it in.

**The one real risk here is not about lazy loading — it's about never making step 1 conditional.** If a future change makes the global-registry seed itself lazy or config-gated (e.g., "only seed `buf/validate` if some `src/` file imports it, to shave a few files off empty-repo compiles"), that would silently disable *all* protovalidate constraint enforcement repo-wide the moment it missed one edge case, which is a far worse failure than anything else in this document. **Recommendation: pin this with a test and a comment, don't touch it as part of the performance work.**

**Test:** Characterization guard, not fails-before-fix (nothing is broken today). Add one test: a fixture proto with a `(buf.validate.field)` constraint, no direct interaction with the lazy-loading changes, asserting the constraint is still enforced after the lazy registry ships. This is a regression tripwire for the *next* five years, not a bug found today.

### Determinism of compiled output — the two order-sensitive spots are already sort-protected

Question 3 asked what breaks when load order becomes demand-driven instead of filesystem-walk order. Beyond `add_validator` (Pitfall 6), traced the two places that consume registry/file iteration order for output:

- `utils.go:188-205`, `DescriptorRegistry.Store()` (the `.fds` cache writer) — explicitly collects keys into a slice and calls `sort.Strings(keys)` before building the `FileDescriptorSet`, and separately runs the written set through `fileDescriptorSetSum` which itself re-sorts (`fdsSorter`) before hashing. **Already order-independent by construction**, regardless of insertion order. (This is also the file the base doc's item 4 says must stay on the eager path entirely — that recommendation is orthogonal to and doesn't conflict with this determinism finding.)
- `utils.go:70-76`, `GetFileDescriptorSet()` (feeds `GetFilesResolver()`) iterates `d.FileRegistry` — a plain Go map, **already unordered today**, eager or lazy makes no difference, and downstream `protodesc.NewFiles` doesn't depend on input order for a self-consistent set.

**Conclusion: compiled JSON output field ordering and `.fds` content are not at risk from demand-driven load order.** The only load-order-sensitive correctness issue found is Pitfall 6 (`add_validator`), which is pre-existing and already characterized.

---

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---|---|---|
| 1. Mutation server `Init()` ranges registry once, before anything loads | Same phase as making the registry lazy — hard blocking co-requirement | Fixture `service` in `src/`, assert `RegisterService` fires before any config compile |
| 2. `FilesResolver`/`LocalResolver` snapshot, map-aliasing false safety | "Resolvers as lazy views" phase (PROJECT.md active item) | `Any`-only-reachable-type resolves via `LocalResolver`, not just the aliased map |
| 3. `GenReflectionUI`'s recurring 5s walk, same stale-resolver issue | Same phase as `proto_file`-driven type resolution (base doc item 2) — add as 5th named consumer | Mutable config with type outside initial registry resolves without `"error finding message"` |
| 4. `cachedRegistry` singleton — one fallback poisons the process | Lazy-by-path registry phase | Long-running-process test: two sequential compiles on one `Compiler`, second doesn't re-widen |
| 5. Concurrent compiles, unsynchronized registry writes | Lazy-by-path registry phase | First-ever concurrent-compile test in this repo, run under `-race` |
| 6. `add_validator` last-write-wins (pre-existing, order-decoupling risk) | No new phase — code-review guard during the lazy-registry phase | Existing `TestLoadValidators_OrphanValidatorIsDiscovered` stays green |
| Extension/custom-option interpretation | Not a phase — pin with a regression test, do not touch the global-seed step | New characterization test for `buf.validate.field` enforcement post-lazy |
| Output determinism | Not a phase — already protected | None needed; documented so it isn't re-investigated later |

## Sources

- This codebase, read directly: `server/server.go`, `compiler/lib/module_service.go`, `compiler/lib/parser/parser.go`, `compiler/lib/compiler.go`, `compiler/lib/starlark_loader.go`, `compiler/lib/starlark_functions.go`, `compiler/service.go`, `devserver/command.go`, `utils/utils.go`, `compiler/lib/starlark_loader_test.go`.
- `.planning/research/compiler-performance/PITFALLS.md` and `TESTING.md` (extended, not duplicated).
- Go runtime behavior for concurrent map access (`fatal error: concurrent map writes`) and `-race` detection semantics — general Go knowledge, not sourced from an external doc; flagged here as MEDIUM confidence on the exact panic message text, HIGH confidence on the underlying hazard (unsynchronized concurrent map read+write is undefined behavior in Go and reliably instrumented by `-race`).
- `grpc.Server.RegisterService` "must be called before Serve, panics after" — grpc-go's documented contract; not independently re-verified against grpc-go source in this session, flagged MEDIUM confidence on the exact panic behavior, HIGH confidence on the ordering requirement (it is why `Init()` is structured to run before `ListenAndServe()` in this codebase's own `devserver/command.go`).

---
*Pitfalls research for: Protoconf v2.0 — lazy descriptor registry retrofit*
*Researched: 2026-09-04*
