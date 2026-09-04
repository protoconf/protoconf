# Options: Getting `NewCompiler` Under Budget

Scope decision (2026-09-04): **lazy load-driven resolution**. This document
records why, and what was rejected.

## The shape of the fix

Today `ModuleService.GetProtoRegistry()` eagerly walks `src/`, parses every
`.proto`, links them all, and caches the result. `parser.ParseFilesX` — the
function the Starlark loader calls when it hits `load("//x/y.proto", "Msg")` —
is then a pure **map lookup** into that pre-filled registry. It never parses.

```go
// compiler/lib/parser/parser.go
func (p *Parser) ParseFilesX(filenames ...string) ([]*desc.FileDescriptor, error) {
    for _, filename := range filenames {
        if fd, ok := p.FileDescriptors[filename]; ok { ... }   // pre-filled
        fd, err := p.FilesResolver.FindFileByPath(filename)     // pre-filled
        ...
    }
}
```

The inversion: make `ParseFilesX` parse on demand, and let `GetProtoRegistry()`
return an empty-but-capable registry. The `load()` statements already name
exactly the files needed, and `protoparse` already resolves transitive imports
itself. The measured cost of that path is 2.7ms.

The public shape of `ParseFilesX` does not change, which matters — see
"Blast radius" below.

## Option A — Lazy by path, on-demand parse (chosen)

Parse and link a proto file the first time something asks for it by path,
memoised in the existing `FileRegistry` map.

- **Cost on the corpus:** 4,639ms -> ~3ms for registry construction.
- **Knock-on:** `NewParserWithDescriptorRegistry` (260ms) is derived from
  `FileRegistry` — `GetFilesResolver()` builds a `FileDescriptorSet` from every
  entry and runs `protodesc.NewFiles` over it. With 7 entries instead of 864
  that cost collapses proportionally. It cannot stay eager: a lazy registry with
  an eager resolver snapshot would be a stale-cache bug the first time a new file
  is parsed.
- **Projected total:** ~200-210ms, dominated by the existing 197ms `CompileFile`.
- **Risk:** lookups that are *not* by path. See PITFALLS.md — this is the real work.

## Option B — Lazy + persisted linked-descriptor cache

Option A plus an on-disk `FileDescriptorSet` cache keyed by content hash, so warm
compiles skip parsing entirely.

Deferred, not rejected. Rationale: Option A already lands at ~3ms for the load
path. A cache optimises 3ms; it is not where the remaining budget is. The
machinery also already half-exists (`registry.Store` / `registry.Load` write
`.fds` files per module into `.protoconf_cache`), so this is cheap to add *later*
if warm-start numbers ever justify it. Adding it now buys nothing measurable and
introduces a cache-invalidation surface.

Note it does become attractive if PITFALLS.md's symbol-index problem forces an
index build — an index is exactly the kind of thing worth persisting.

## Option C — Migrate off `jhump/protoreflect` to `protoregistry`/`dynamicpb`

Rejected for this milestone. Already recorded in PROJECT.md as a deferred v2
item ("large scope, touches compiler/starproto extensively"), and the profile
does not implicate the `desc` wrappers: `desc.WrapFiles` over the global registry
is 6.2ms, and `desc.wrapFile` is 0.16s cum inside a 4.6s stage. The cost is in
`protocompile`'s parse+link, which both APIs sit on top of. Migrating would move
the same work behind a different type.

## Rejected: optimise the linker lookup

`linker.Files.FindFileByPath` is 16% of profile time and is a linear scan. It is
still not worth touching — `r.deps` is a per-file dependency list, not the full
set, and v0.14.1 is byte-identical. See BASELINE.md "Negative result". Recorded
here so it is not re-proposed.

## Rejected: parallelise the eager parse

Would divide 4.6s by core count at best — call it 600-800ms on an 8-core box,
still 3-4x over budget, while burning every core on work that is ~99% discarded.
It also makes GC pressure worse per unit of wall time, and the corpus that
motivated this issue will keep growing. Parallelism is the wrong axis when the
right answer is to not do the work.

## Rejected: eager symbol index (any form)

Needed to answer type-URL lookups (PITFALLS.md), but every eager variant measured
over the corpus exceeds the whole budget by itself:

| Variant | Cost |
|---------|------|
| `ParseFilesButDoNotLink` over 799 | 1,286ms |
| Naive lexical scan (read + regex per file) | 623ms |

Parallelising the lexical scan could plausibly reach ~100ms, but that spends half
the total budget on an index most compiles never consult. If an index is
required, it must be **persisted and incrementally maintained**, not rebuilt per
invocation — which is Option B's machinery.

## Blast radius

`GetProtoRegistry()` has five consumers. All were checked; all are lookup-shaped,
which is what makes lazy viable:

| Consumer | Site | Lookup by |
|----------|------|-----------|
| compiler | `compiler/lib/compiler.go:65`, `:355` | path (`load()`), type URL |
| mutation server | `server/server.go:290` | type URL |
| inserter | `inserter/inserter.go:220` | type URL |
| agent filekv | `agent/filekv/filekv.go:80` | type URL |
| mutate CLI | `mutate/mutate.go:73` | type URL |

The compiler is the only path-driven consumer, and the only one on the 200ms hot
path. The other four resolve `google.protobuf.Any` type URLs — that is the case
Option A does not answer on its own, and it is the subject of PITFALLS.md.

One caller must stay eager: `ModuleService.Sync` / `GenFileDescriptorSet`
(`protoconf mod sync`) genuinely needs to parse everything to write `.fds` cache
files. That is a separate, explicitly-invoked command with no latency budget.
Keep its eager path intact rather than routing it through the lazy one.
