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

## 2. `Any` resolution by type URL — softer than first assessed

**Corrected 2026-09-04.** This section originally claimed a symbol index was
needed and called this "the one genuinely hard part of the work." That was wrong,
and it would have misdirected planning. The correction is below; the original
reasoning is kept at the end so the mistake is legible.

Four of the five `GetProtoRegistry()` consumers (server, inserter, filekv, mutate)
resolve `google.protobuf.Any` payloads through
`MessageRegistry.FindMessageTypeByUrl` / `LocalResolver.FindMessageByURL`. The
compiler does too, in `loadMutable` (`starlark_loader.go:173`, `:183`), and at
write time in `writeConfig`, which marshals a `ProtoconfValue` through
`protojson` with `c.parser.LocalResolver`.

A type URL names a symbol, not a file — but **the data model already carries the
file**. `ProtoconfValue.proto_file` is field 1 of the envelope
(`pb/protoconf/v1/protoconf.proto:12`) and is populated on every write by
`toProtoconfValue` (`compiler/lib/compiler.go:281`) with
`message.GetMessageDescriptor().GetFile().GetName()`.

Every consumer that resolves a type URL is reading a **materialized config**, and
that config states which `.proto` produced it. So the lookup is
`type URL -> proto_file -> lazy parse by path` — the same path-driven mechanism
Option A already builds. No symbol index, no convention-probe, no eager fallback.

Two cases split cleanly:

- **Types produced by the current compile** (`writeConfig`) are already in the
  lazy registry, because the config `load()`ed them to build the message. The
  resolver just has to be a lazy view rather than a snapshot — which is item 3,
  not a separate problem.
- **Types read from a materialized config** (`loadMutable`, and all four daemon
  consumers) come with `proto_file` attached. Read it, parse that file, resolve.

**The one real wrinkle is ordering.** `protojson` needs the `Any`'s type while it
is unmarshaling the envelope, and it cannot be relied on to read `proto_file`
first. So the read path needs a cheap pre-pass: pull `protoFile` out of the raw
JSON, lazily load that proto, then unmarshal properly. Small and well-defined,
but it is real work and it touches `parser.ReadConfig`, which every consumer uses.

**Residual risk.** `proto_file` is only as good as what wrote it. A config
materialized by an older protoconf, hand-edited, or produced by the
`message.MergeInto` branch of `toProtoconfValue` (which returns early without
setting `ProtoFile`) may carry an empty or stale value. The read path must handle
an absent `proto_file` explicitly rather than assuming it — falling back to the
eager parse is acceptable there, provided the fallback is **loud** (a one-line
warn or a counter surfaced under `-v`). A silent fallback lets a repo regress to
6.9s with the benchmark still green.

<details>
<summary>Original assessment, superseded</summary>

The first pass claimed you "cannot know which of 799 files declares
`pkg.Message` without having looked inside them", and proposed convention-probe
-> persisted symbol index -> eager fallback, with the index costing 623ms-1.29s
cold. That reasoning treated the type URL as the only available information and
missed `proto_file` entirely. The index options in OPTIONS.md ("Rejected: eager
symbol index") remain correctly rejected, now for a stronger reason: nothing
needs an index.

</details>

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
5. Type-URL resolution (item 2) — `proto_file`-driven; the work is the `ReadConfig` pre-pass and the absent-`proto_file` fallback, not an index.
6. Decide and document the error-surface change (item 5).
