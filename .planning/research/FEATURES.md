# Feature Research: Type-URL Resolution Strategies for Lazy Proto Loading

**Domain:** protobuf tooling — resolving `google.protobuf.Any` type URLs to source `.proto` files without an eagerly-loaded descriptor registry
**Researched:** 2026-09-04
**Confidence:** LOW per the project's `classify-confidence` seam (uncurated `websearch`/`webfetch`, no verified-source integration configured) — but every claim below is sourced from primary/official documentation (protobuf.dev, buf.build/docs, bazel.build, protocolbuffers/protobuf-go on GitHub), not third-party recall, and the package/directory question was checked directly against the proto3 grammar spec rather than a search snippet. Treat the *tier label* as LOW-by-policy, the *individual claims* as MEDIUM-grade (official docs, single-pass, not cross-verified by a second independent source).

This is not a product feature landscape — it is a mechanism-option landscape for one architectural decision (PITFALLS.md item 2's "chain to evaluate during phase planning"). The template's table-stakes / differentiators / anti-features framing maps directly: table stakes are correctness requirements no chosen mechanism can skip, differentiators are speed-only layers, anti-features are approaches that look like prior art but don't transfer.

## What other ecosystems actually do (and the gap this project sits in)

Checked: buf's module/workspace resolution and linking, grpc server reflection + grpcurl, Bazel's `proto_library`/`ProtoInfo`, Go's `protoregistry.GlobalTypes`/`GlobalFiles`.

None of them solve "resolve a symbol against a corpus that has not been fully loaded." Each sidesteps the problem a different way:

- **buf** resolves *imports* (file paths) against the local workspace then module deps — that's path resolution, the same shape as protoconf's already-solved `load()` case. Symbol-to-file mapping only happens during **linking**, and linking requires the full module already parsed — buf builds this table over a fully-loaded module, not incrementally against an unknown corpus.
- **grpc server reflection / grpcurl** resolve symbols against a `FileDescriptorSet` or descriptor pool the *server already built and holds entirely in memory at startup*. The reflection protocol has a "by symbol" request shape, but the server answers it from a complete pool — it never has to decide which of N files to open.
- **Bazel** resolves proto dependencies through an explicit, author-declared `BUILD`-file graph (`proto_library` deps), not runtime symbol search. `foo-descriptor-set.proto.bin` is generated per target from its declared `srcs`, and `ProtoInfo.transitive_descriptor_sets` propagates that up the explicit graph. This is path/target resolution wearing a different name — closer precedent for protoconf's `load()` case than for the type-URL case.
- **Go's `protoregistry.GlobalTypes`** is populated by generated `.pb.go` files registering themselves via `init()` side effects when their package is imported. `FindMessageByURL` then does an O(1) map lookup — but only because every schema the process will ever need was compiled into the binary in advance. This assumes static, compile-time knowledge of the full schema set.

**Conclusion for phase planning:** there is no off-the-shelf "lazy symbol resolver" precedent to adopt wholesale. Protoconf's constraint — a corpus of unknown-at-build-time protos, discovered from a config repo, resolved just-in-time — is genuinely outside what these tools were built for. This validates PITFALLS.md's decision to leave the mechanism open rather than importing someone else's answer; use the pieces below as a menu, not a template.

## Feature Landscape

### Table Stakes (Correctness — no mechanism can skip these)

| Feature | Why Required | Complexity | Notes |
|---------|--------------|------------|-------|
| `proto_file`-driven resolution for the top-level message | Already the data model (field 1, populated on every write); skipping it means re-deriving what's already carried | LOW | Solved per PITFALLS.md item 2 — reduces to "lazy parse by path," which is Option A |
| Post-candidate symbol verification | protobuf's spec has **no** rule tying `package` to file path (confirmed against the proto3 grammar directly — package is a dotted identifier, import is a string literal, no linkage between them). Any heuristic that guesses a file must be checked, or it can silently resolve to the wrong type | LOW | Free: it's the same parse the mechanism already performs to load the candidate file, then one `FindMessageByName`/`FindEnumByName` lookup against the result. Not a fourth pipeline stage, it's confirming what step 3 below already produced |
| Explicit absent/stale-`proto_file` handling | `message.MergeInto`'s branch of `toProtoconfValue` returns early without setting `ProtoFile` (PITFALLS.md item 2); older writers may not have set it at all | LOW | Must be a real code path, not an assumption — falling back to eager parse is acceptable *if* it is loud |
| Loud (non-silent) terminal fallback | A silent fallback to full eager parse regresses a repo to 6.9s with the scaling benchmark still green — the exact failure mode PROJECT.md's Active requirements name explicitly ("Loud (never silent) fallback when a type URL cannot be resolved cheaply") | LOW | One-line warn or a counter surfaced under `-v` is sufficient; the requirement is "never silent," not "never slow" |
| Independent resolution path for nested `Any` | `ProtoconfValue.proto_file` names only the top-level message's file. `any_field`/`any_repeated`/`any_map` (exercised by `field_type_any_test.pconf`) reference types in *any other file* — structurally unreachable from `proto_file`. Treating "top-level solved" as "Any solved" is the exact mistake PITFALLS.md corrected twice | MEDIUM | This is the actual open problem; everything below is candidate machinery for it |
| Never trust an unverified hint (package→directory guess or a revived `proto_files` field) as ground truth | Both are heuristics from *convention*, not from the protobuf language spec. Both must terminate at symbol verification, not stop one step early | LOW | See "safety of package→directory" and "envelope hint failure modes" below — this is the design invariant that keeps either from becoming a silent wrong-answer bug |

### Differentiators / Optimizations (speed only — none of these are required for correctness)

| Feature | Value Proposition | Complexity | Notes |
|---------|--------------------|------------|-------|
| Package-name → directory heuristic | On the terraform corpus: 13/13 packages match their directory exactly; median files-per-package is 1, so the median case resolves with a single targeted parse and zero scanning | LOW | **Not spec-guaranteed.** protobuf.dev's own style guide says the package "should not be coupled with the directory path, especially when the files are in a deeply nested path" — official guidance is agnostic-to-cool on the convention. Buf's `PACKAGE_DIRECTORY_MATCH` lint rule (MINIMAL category — buf's most fundamental hygiene tier, the one whose violations "usually cause real problems in downstream tooling") enforces the opposite: file path must match package. The convention exists because Go/Java-style per-package-per-directory code generation *needs* it, not because the wire format requires it. Read as: strong convention in generated/tooled repos, unenforceable assumption in hand-written ones. Must feed into verification, never bypass it. |
| Scoped lexical scan of the resolved directory | Reduces the worst case (256 files in `terraform.aws.resources.v6`) from "scan 799 files" (623ms, measured) to "scan ~256" (~200ms extrapolated) — still a full-corpus-scan-shaped cost in the worst directory, but bounded by the heuristic's blast radius instead of the whole tree | MEDIUM | Only a fallback *within* the package→directory heuristic when that directory holds >1 candidate file. Does not stand alone — depends on the heuristic already narrowing the search |
| Revived `repeated string proto_files` field (wire-compatible as field 5; fields 1-4 are taken) | Written by the compiler at marshal time, this is a precomputed hint that can skip the package→directory guess entirely when present and correct | MEDIUM | This is the field an earlier version reportedly shipped and that was "buggy" — see anti-feature analysis below for exactly how it can go wrong and what makes a revival of it safe |
| Persisted symbol index (buf/Bazel-style) | The only precedent found for a real "persisted, incrementally-maintained index" is Bazel's per-`proto_library`-target `foo-descriptor-set.proto.bin`, propagated transitively and invalidated by Bazel's normal content-hash/action-graph cache semantics. That is a file→descriptor-set cache keyed by build inputs, **not a symbol table** — buf's own linking-stage symbol table is, per buf's docs, an in-memory per-invocation structure with no documented cross-run persistence. Buf's on-disk module cache (`~/.cache/buf`) caches *downloaded remote module content* by commit, not parsed/linked results | HIGH | Deferred per OPTIONS.md Option B — the machinery (`registry.Store`/`Load` writing `.fds` per module) already half-exists and is directionally the right shape (content-keyed file cache, Bazel-style) if this is ever needed, but nothing examined justifies building a symbol table from scratch. Every eager-index variant measured on this corpus (1,286ms `ParseFilesButDoNotLink`, 623ms lexical scan) blows the 200ms budget alone |
| Parallelized lexical scan | Could plausibly get an unscoped 623ms scan to ~100ms | LOW | Still half the total budget on a mechanism most compiles never need — only worth combining with the scoped-directory optimization above, never as a substitute for it |

### Anti-Features (avoid, with reasons)

| Feature | Why It Looks Attractive | Why It's Wrong Here | Instead |
|---------|--------------------------|----------------------|---------|
| Eager full-corpus symbol index, rebuilt per invocation | "Just index everything once, then every lookup is O(1)" | Measured cost on this corpus: `ParseFilesButDoNotLink` over 799 files = 1,286ms; naive lexical scan = 623ms. Both blow the entire 200ms budget alone, before any config-specific work runs (OPTIONS.md, "Rejected: eager symbol index") | Path-driven lazy parse (Option A) plus the scoped, on-demand mechanisms above |
| Trusting package→directory mapping as ground truth (no verification) | It's exact on the sampled corpus (13/13) and would make resolution a single lookup | protobuf's spec permits *any* package/path relationship (confirmed from the grammar directly); the sampled corpus is machine-generated and "a favourable sample that hand-written repos will break" (PITFALLS.md's own conclusion). Buf's official style guide even discourages coupling deeply-nested paths to package names — the convention is tooling-ecosystem pressure, not a guarantee | Use it to narrow the search, always confirm with the symbol-verification step (which is free — see table stakes) |
| Treating a revived `proto_files` envelope field as authoritative | It would make nested-`Any` resolution nearly free when present | Concrete staleness/incompleteness modes: (1) a config materialized before a schema refactor moved a message to a different file — parse succeeds, symbol is absent, unverified code resolves the wrong type or crashes confusingly; (2) `toProtoconfValue`'s `MergeInto` branch returns early without populating fields (already documented in PITFALLS.md) — the list can be silently partial for a subset of nested `Any` values; (3) version skew — older protoconf binaries, hand-edited `materialized_config/` files (a documented, supported workflow in this project), or the pre-fork version of this exact field (reported buggy, no trace in this repo's history) may emit an empty, partial, or semantically different field. This is precisely the bug class that reportedly sank the original field | Treat as a *hint that narrows the search*, always run it through symbol verification. A stale hint should degrade to "no faster than not having it," never to "resolves to the wrong type" |
| Adopting Go's `protoregistry.GlobalTypes` init-time self-registration pattern | It's the idiomatic Go answer to "resolve a type URL," and this project already links `google.golang.org/protobuf` | It assumes every schema the process will ever need was known and compiled in at build time (generated `.pb.go` packages side-effect-importing themselves). Protoconf's whole premise is schemas discovered dynamically from an arbitrary config repo at runtime — there is no `.pb.go` to import in advance | Keep the dynamic, on-demand `desc`/`protoreflect` path already in use; this pattern doesn't transfer regardless of mechanism chosen for the symbol problem |
| Adopting grpc server reflection's or Bazel's resolution model as literal precedent for the symbol-lookup step | Both are real, working systems that "resolve types" | Both resolve against something already fully materialized — reflection against a descriptor pool the server built completely at startup, Bazel against an explicit author-declared dependency graph. Neither has ever had to choose among N unopened files by symbol name alone; they're evidence path-driven resolution is the normal shape (validating protoconf's already-solved `load()` case), not a solution to the type-URL case | Use them as confirmation that no mechanism here is "the industry standard" — the choice genuinely is protoconf's to make, which is why phase planning should treat it as such rather than searching for a canonical answer |

## Feature Dependencies

```
proto_file (top-level, already carried)
    └──already solved by──> Option A: lazy-by-path parse

symbol verification (parse candidate + FindMessageByName)
    └──requires──> a parsed candidate FileDescriptor
                      (i.e. requires Option A's lazy parser to exist first)

package→directory heuristic
    └──requires──> symbol verification (never trust unverified)
    └──enhanced by──> scoped lexical scan
                          (fallback when the resolved directory holds >1 file)

revived `proto_files` hint (field 5)
    └──requires──> symbol verification (never trust unverified)
    └──enhances──> package→directory heuristic
                      (skips the directory guess entirely when present and verified)

loud terminal fallback
    └──triggered when──> proto_file absent AND heuristic/hint/scan all fail to verify

persisted symbol index (Option B / Bazel-style descriptor-set cache)
    └──conflicts with──> the "no index needed" scope decision in PITFALLS.md/OPTIONS.md
    └──only relevant if──> hint + heuristic + scoped-scan chain proves insufficient in production
```

### Dependency Notes

- **Symbol verification requires the lazy parser to exist first:** this is not a new subsystem — it is the same `ParseFilesX`-shaped call Option A already makes for `load()` paths, invoked with a guessed path instead of a `load()`-named one. That's why it's free: the parse happens regardless of whether the guess is right, and checking the result costs one map lookup.
- **The heuristic and the hint both terminate in verification, never around it:** this is the single design invariant that makes either safe to use at all, given neither is guaranteed by the protobuf spec.
- **The persisted index conflicts with the current scope decision:** OPTIONS.md rejected an eager index outright and PITFALLS.md's correction found no index is needed for the common cases. Only reach for it if production data shows the hint/heuristic/scan chain missing often enough to matter — and if so, model it on Bazel's per-target descriptor-set cache (content-hash invalidated), not a from-scratch symbol table, since that's the only persisted design found with real invalidation semantics.

## MVP Definition

This maps directly onto PITFALLS.md's "chain to evaluate during phase planning" — restated here as a layered rollout rather than a single mechanism, since the decision is deliberately deferred:

### Launch With (v1 — must ship for correctness)

- [ ] `proto_file`-driven resolution for top-level messages — already designed, reduces to Option A
- [ ] Symbol verification on every candidate resolution, no exceptions — the one universal safety net
- [ ] Explicit absent/stale-`proto_file` handling with a loud fallback path
- [ ] Package→directory heuristic for nested `Any`, always gated by verification

### Add After Validation (v1.x — only if the heuristic's worst case shows up in practice)

- [ ] Scoped lexical scan for directories with >1 candidate file (covers the measured worst case: 256 files in one terraform package)
- [ ] Revived `proto_files` field (field 5), written at marshal time, consumed as a verified hint

### Future Consideration (v2+ — only if data justifies it)

- [ ] Persisted symbol/descriptor-set index, Bazel-descriptor-set-cache-shaped (content-hash keyed, not rebuilt per invocation) — nothing examined in this research or in OPTIONS.md's measurements currently justifies this; revisit only if the hint/heuristic/scan chain measurably misses often enough in production use to matter

## Feature Prioritization Matrix

| Feature | User Value (correctness/speed) | Implementation Cost | Priority |
|---------|-------------------------------|----------------------|----------|
| `proto_file` lazy-by-path resolution | HIGH (correctness, top-level case) | LOW (already designed) | P1 |
| Symbol verification | HIGH (correctness, all cases) | LOW (free — reuses the parse) | P1 |
| Loud fallback on unresolved/absent `proto_file` | HIGH (correctness — prevents silent regression) | LOW | P1 |
| Package→directory heuristic (verified) | MEDIUM (speed, median case is free) | LOW | P1 |
| Scoped lexical scan fallback | MEDIUM (speed, bounds the worst case) | MEDIUM | P2 |
| Revived `proto_files` hint (verified) | MEDIUM (speed, skips the guess) | MEDIUM | P2 |
| Persisted symbol/descriptor-set index | LOW at current measured scale | HIGH | P3 |
| Parallelized lexical scan | LOW (marginal, only matters if scan is on the hot path often) | LOW | P3 |

**Priority key:**
- P1: Correctness — required regardless of which optimization layers phase planning chooses
- P2: Should have if the median-case heuristic proves insufficient for this project's real corpora
- P3: Only if production data shows P1+P2 insufficient; not justified by anything measured so far

## Ecosystem Comparison (in place of a "competitor" table)

| Concern | buf | grpc reflection / grpcurl | Bazel | Go `protoregistry` | Protoconf's constraint |
|---------|-----|----------------------------|-------|----------------------|--------------------------|
| How types are found | Symbol table built during linking, over a fully-parsed module | Descriptor pool built once at server startup, queried by symbol | Explicit per-target dependency graph declared in `BUILD` files | Static compile-time `init()` self-registration | Corpus unknown at build time, discovered from a config repo, must resolve just-in-time |
| Assumes full corpus loaded? | Yes, at time of linking | Yes, at server startup | N/A — resolution is via declared deps, not runtime search | Yes, at process init | No — this is exactly what makes it lazy |
| Directly reusable technique | Path/import resolution (already mirrors protoconf's solved `load()` case) | None (assumes the hard part already done) | Path/target resolution (again mirrors `load()`) | None (assumes the hard part already done) | — |

## Sources

- [Buf Docs — Compilation and descriptors](https://buf.build/docs/reference/compilation-and-descriptors) — linking stage builds a symbol-to-file mapping over a fully-parsed module; no documented cross-run persistence of that table
- [Buf Docs — Lint rules and categories](https://buf.build/docs/lint/rules/) — `PACKAGE_DIRECTORY_MATCH` is in the MINIMAL category; violations "usually cause real problems in downstream Protobuf tooling"
- [Buf Docs — Files and packages](https://buf.build/docs/reference/protobuf-files-and-packages/)
- [Buf Docs — Managing dependencies](https://buf.build/docs/bsr/module/dependency-management/) — module cache stores downloaded remote module content, not parsed/linked results
- [Protocol Buffers — Style Guide](https://protobuf.dev/programming-guides/style/) — "The package should not be coupled with the directory path, especially when the files are in a deeply nested path"
- [Protocol Buffers Language Specification (Proto3)](https://protobuf.dev/reference/protobuf/proto3-spec/) — grammar for `package`/`import`; no rule linking package to file path or directory
- [GRPC Server Reflection Protocol](https://grpc.github.io/grpc/core/md_doc_server-reflection.html) — symbol-based `DescriptorDatabaseRequest`, answered from a descriptor pool the server already holds
- [grpcurl / protoreflect package docs](https://pkg.go.dev/github.com/Gitforxuyang/protoreflect/grpcurl) — resolves symbols against a `FileDescriptorSet` or a reflection-capable server, never against an unopened corpus
- [Bazel — Protocol Buffer Rules](https://bazel.build/reference/be/protocol-buffer) and [ProtoInfo](https://bazel.build/versions/6.5.0/rules/lib/ProtoInfo) — per-target descriptor sets, propagated transitively, invalidated by Bazel's standard content-hash/action-graph cache semantics
- [protoregistry package — google.golang.org/protobuf/reflect/protoregistry](https://pkg.go.dev/google.golang.org/protobuf/reflect/protoregistry) — `GlobalTypes`/`GlobalFiles`, `FindMessageByURL`, populated via generated-code `init()` self-registration
- [google.protobuf.Any / TypeResolver](https://github.com/protocolbuffers/protobuf/blob/main/src/google/protobuf/any.proto) and [type_resolver.h](https://protobuf.dev/reference/cpp/api-docs/google.protobuf.util.type_resolver/) — canonical `type_url` format and the abstract `TypeResolver` interface pattern

---
*Feature research for: type-URL resolution mechanism options, protoconf v2.0 lazy proto loading milestone*
*Researched: 2026-09-04*
