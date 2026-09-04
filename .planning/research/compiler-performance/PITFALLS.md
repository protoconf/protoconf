# Pitfalls: What Breaks Under Lazy Loading

Lazy resolution is a **correctness change disguised as a performance change**.
Today every proto in `src/` is in the registry, and several code paths quietly
depend on that. Each one below is a silent-wrong-answer risk, not a crash.

Ordered by how likely they are to ship undetected.

## 1. `loadValidators` iterates the registry, not the filesystem — BLOCKER

`compiler/lib/starlark_loader.go:116`:

```go
l.parser.FilesResolver.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
    protoFile := fd.Path()
    validatorFile := protoFile + consts.ValidatorExtensionSuffix
    validatorAbsPath := filepath.Join(l.srcDir, validatorFile)
    if exists, isDir, err := stat(validatorAbsPath); ... 
```

For every proto in the registry, it stats for a sibling `.proto-validator` and
loads it if present.

Under lazy loading the registry holds 7 files instead of 864, so
**cross-field validators attached to protos not reachable from `load()` are
silently skipped.** Configs that should fail validation would compile clean. This
is the worst possible failure mode for a config system: it does not error, it
approves.

It is also load-order dependent — `loadValidators` runs against whatever happens
to be in the registry at that moment, which under lazy loading is a moving target.

**Fix — and it is faster, not a compromise:** invert the loop. Walk the
filesystem for `*.proto-validator` files and load those, instead of walking 864
descriptors and stat-ing for each. Validators are rare (the terraform corpus has
**zero**, yet still pays 864 stat syscalls today, inside the 197ms `CompileFile`).
Walking a handful of real files is both correct under lazy loading and cheaper
than what happens now.

Do this **before** making the registry lazy. It is independently correct, it is a
speedup on its own, and it removes the ordering hazard rather than working around it.

## 2. `Any` resolution by type URL has no file to be lazy about

Four of the five `GetProtoRegistry()` consumers (server, inserter, filekv, mutate)
resolve `google.protobuf.Any` payloads through
`MessageRegistry.FindMessageTypeByUrl` / `LocalResolver.FindMessageByURL`. The
compiler does too, in `loadMutable` (`starlark_loader.go:173`, `:183`).

A type URL is `type.googleapis.com/pkg.Message`. It names a **symbol**, not a
file. Lazy-by-path cannot answer it: you cannot know which of 799 files declares
`pkg.Message` without having looked inside them.

This is the one genuinely hard part of the work. Options, in increasing cost:

- **Convention:** derive a candidate path from the package name and probe it.
  Cheap, works for repos where `pkg/v1/foo.proto` declares `pkg.v1.*`. Fails
  silently on repos that do not follow it — so it needs a fallback, not a bare bet.
- **Persisted symbol index:** build once, store in `.protoconf_cache`, invalidate
  on mtime/hash. Correct and fast warm. Costs 623ms-1.29s to build cold (see
  OPTIONS.md) and adds an invalidation surface.
- **Eager fallback:** on a miss, fall back to today's full parse. Correct by
  construction, and the slow path only fires when the fast path cannot answer.

The pragmatic combination is convention-probe, then index, then eager fallback —
but note that a *silent* fallback to eager parsing means a repo can regress to
6.9s without anyone noticing. Whatever is built, **make the fallback loud** (a
one-line warn, or a counter surfaced in `-v`), or the benchmark will pass while
real users stay slow.

Also worth checking: whether the mutation server and agent — which are long-lived
processes, not CLI invocations — actually want lazy at all. Paying 4.6s once at
daemon startup is very different from paying it per compile. Lazy may be correct
to scope to the compiler and leave the daemons eager.

## 3. `GetFilesResolver` / `GetTypesResolver` snapshot the registry

`parser.NewParserWithDescriptorRegistry` calls `registry.GetFilesResolver()`,
which builds a `FileDescriptorSet` from **every** entry in `FileRegistry` and runs
`protodesc.NewFiles` over it, then `GetTypesResolver` ranges that to register
every message and enum into a `protoregistry.Types`.

These are eager snapshots taken once at construction. With a lazy registry, any
file parsed *after* construction is absent from both resolvers — and
`ParseFilesX` consults `p.FilesResolver.FindFileByPath` as its second lookup, so
this is directly on the load path.

The resolvers must become lazy views over the registry, or be invalidated on
every insertion. A snapshot plus a lazy source is a stale-cache bug waiting to
happen, and it will present as "works the first time, wrong the second".

## 4. `registry.Store` writes whatever `localFiles` happens to hold

`utils.go` `Store()` serialises `d.localFiles` to a `.fds` cache file, and
`Parse()` resets `localFiles` on every call. Under lazy loading `localFiles`
becomes "whatever was demanded so far" rather than "everything in this module",
so a `mod sync` that shares a lazy registry would write a **truncated** cache file
— which then loads clean and is wrong on the next run.

Keep `mod sync` / `GenFileDescriptorSet` on the eager path (OPTIONS.md, "Blast
radius"). Do not let the two share a registry instance.

## 5. Import-cycle and error-surface changes

`protoparse` reports import errors when it parses. Eagerly, a broken import
anywhere in `src/` fails the compile immediately. Lazily, a broken proto that no
config loads is never parsed and never reported.

This is arguably an improvement — but it is a **behaviour change** that will
surface as "CI used to catch this". Decide deliberately whether `protoconf
compile` should keep validating the whole tree (perhaps behind a flag, or left to
`mod sync`), and write it down rather than letting it change by accident.

## 6. The benchmark corpus is a moving target

`protoconf-terraform` is a working tree, not a fixture. Its proto count will
drift, and every number in BASELINE.md drifts with it. See TESTING.md — pin a
corpus before optimising against one, or the regression test measures the corpus
rather than the compiler.

## Ordering

1. Fix `loadValidators` (item 1) — independently correct, independently faster.
2. Pin a benchmark corpus (item 6) — nothing else is measurable without it.
3. Make the resolvers lazy views (item 3) — prerequisite for a lazy registry.
4. Lazy-by-path registry — the actual win.
5. Type-URL resolution (item 2) — the hard part; scope may reduce to compiler-only.
6. Decide and document the error-surface change (item 5).
