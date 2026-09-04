# Project Research Summary

**Project:** Protoconf — Lazy Proto Descriptor Resolution (v2.0 performance + correctness milestone)
**Domain:** Go proto-descriptor compiler internals (jhump/protoreflect, google.golang.org/protobuf)
**Researched:** 2026-09-04
**Confidence:** HIGH overall, with one explicitly LOW-by-policy but MEDIUM-grade section (type-URL mechanism options)

## Executive Summary

This milestone has one measured, unambiguous target: `TestCompilerStartupScaling` currently reports a 7.24x allocation-ratio blowup between n=50 and n=400 synthetic protos and `t.Skipf`s because the target is <=2.0x. Done means that ratio holds and the skip becomes `require.LessOrEqual`. The root cause is fully diagnosed and cheap to fix: `ModuleService.GetProtoRegistry()` eagerly parses and links every `.proto` under `src/` (4,639ms of a 6.97s compile, 94% of wall time, on the measured 799-file corpus) when a typical compile only needs 7 files reachable from its `load()` graph (2.7ms). A second, smaller cost (`NewParserWithDescriptorRegistry`'s 260ms resolver-snapshot rebuild) is caused by the same eagerness one layer up. Fixing both — lazy-by-path parsing plus incremental resolver registration — is projected to land the compile path at ~200-210ms, already dominated by the 197ms Starlark-eval/validate/write stage that is not itself in scope.

The catch is that "lazy" is a load-bearing assumption change, not a pure optimization. Making the registry incomplete-by-default breaks every piece of code that currently assumes it is complete. Three such assumptions have now been found, all structurally the same defect wearing different code: `loadValidators` ranging the registry for `.proto-validator` files (already fixed, pre-milestone), `ProtoconfMutationServer.Init()` ranging the registry once at startup to discover and register gRPC services before `Serve()` runs (unfixed — the most severe finding, since a missing service registration is permanent and silent, distinguishable from "was never built" only by absence), and `GenReflectionUI`'s 5-second ticker resolving `Any` type URLs against a resolver snapshot that goes stale the instant the registry starts growing (unfixed — fails silently and repeatedly, not just once). Treat "code that ranges a registry and assumes it enumerates everything" as a recurring defect class needing a single shared mitigation pattern, not three unrelated bugs to patch independently.

The recommended approach: fix the registry/resolver core first (this alone gets the compiler to budget and is what the scaling test measures), then separately handle the four non-compiler consumers (mutation server, inserter, filekv, mutate CLI) as a correctness-only pass with "no regression, no silent wrong answers" as their success criterion — they were never on the 200ms budget and must not be conflated with the speed work. The mutation server's `Init()` problem is a structural exception requiring its own explicit eager sub-path (or deferred init) rather than routing through the lazy mechanism at all. The type-URL resolution mechanism for `google.protobuf.Any` (needed by the four non-compiler consumers plus `loadMutable`) is deliberately left undecided by user instruction; this document presents the option space rather than a winner, for phase planning to settle.

## Key Findings

### Recommended Stack

No new dependency is proposed or needed. The pinned stack (`jhump/protoreflect v1.16.0`, `google.golang.org/protobuf v1.34.1`, `bufbuild/protocompile v0.10.0` indirect) already exposes every API the lazy design requires:

**Core technologies:**
- `protoparse.Parser.LookupImport` — the existing seam (`utils/utils.go:141-152`) that already serves a lazily-growing `FileRegistry` cache; needs a mutex, not a redesign.
- `protoregistry.Files.RegisterFile` / `protoregistry.Types.RegisterMessage`/`RegisterEnum` — incremental, public, already-used-by-generated-code APIs for growing `FilesResolver`/`LocalResolver` one file at a time instead of rebuilding from a `FileDescriptorSet` snapshot (eliminates the 260ms rebuild).
- `desc.FileDescriptor.UnwrapFile()` — gets the `protoreflect.FileDescriptor` needed for `RegisterFile` directly out of jhump's parse result, no round-trip through `protodesc.NewFiles`.

**Critical version/API finding:** `protoregistry.Files`/`Types`'s "safe for concurrent use" doc comment is scoped *only* to the package-level `GlobalFiles`/`GlobalTypes` singletons (locking is conditioned on `r == GlobalFiles` via pointer identity). A project-owned instance — exactly what this codebase constructs — has **zero internal locking**; concurrent `Register*` calls, or `Register*` concurrent with `Find*`/`Range*`, is a data race today waiting for lazy mode to expose it. This must be wrapped in a dedicated `sync.RWMutex`. No lazy/incremental resolver ships anywhere in this dependency stack — all caching must be hand-rolled at the `LookupImport`/`FileRegistry` layer, which is exactly the shape already chosen (Option A below).

**Explicitly rejected additions:** `github.com/bufbuild/protoresolve` (not in the dependency graph, would be a new dependency for no capability gap), `linker.Resolver` (internal to protocompile, never exposed through jhump's public API), migrating off `jhump/protoreflect` (explicitly deferred v2 item; profiling shows the cost is in protocompile's parse+link, not the jhump wrapper — migrating moves the same work behind a different type).

### Expected "Features" (this is a mechanism-option landscape, not a product feature set)

FEATURES.md researched one specific open decision: how to resolve `google.protobuf.Any` type URLs to source files without an eagerly-loaded registry. No existing ecosystem (buf, grpc reflection, Bazel, Go's own `protoregistry.GlobalTypes`) solves "resolve a symbol against a corpus that hasn't been fully loaded" — each one assumes the corpus is already fully materialized before symbol lookup starts. This validates leaving the mechanism open rather than importing a canonical answer.

**Must have (table stakes — correctness, no mechanism can skip these):**
- `proto_file`-driven resolution for the top-level message (already the data model; `ProtoconfValue.proto_file` is field 1, already populated on every write) — reduces directly to Option A's lazy-by-path parse.
- Post-candidate symbol verification on every resolution — free (it's the same parse the mechanism already performs), and non-negotiable because protobuf's spec has **no rule** tying `package` to file path (confirmed against the proto3 grammar directly). Any heuristic guess must be checked or it can silently resolve to the wrong type.
- Explicit, loud (never silent) fallback for absent/stale `proto_file` — a silent eager fallback regresses a repo to 6.9s while the scaling gate stays green, the exact failure mode this milestone must not reintroduce.
- Independent resolution path for **nested** `Any` — `proto_file` only names the top-level message's file; `any_field`/`any_repeated`/`any_map` reference types in arbitrary other files, structurally unreachable from `proto_file`. Treating "top-level solved" as "Any solved" is a documented repeat mistake.

**Should have / speed-only optimizations (should be gated behind verification, never bypass it):**
- Package-name -> directory heuristic: exact on 13/13 packages in the measured terraform corpus, but **not spec-guaranteed** — protobuf's own style guide discourages coupling package to directory path for deeply nested trees, while buf's `PACKAGE_DIRECTORY_MATCH` lint rule (MINIMAL/most-fundamental category) enforces the opposite convention. Read as: strong convention in generated/tooled repos, unenforceable assumption in hand-written ones.
- Scoped lexical scan of the resolved directory as a fallback when that directory holds >1 candidate (measured worst case: 256 files in one terraform package).
- Revived `repeated string proto_files` field (wire-compatible as field 5) as a precomputed hint. **User-supplied domain knowledge, unverified against this repo's history:** an earlier protoconf version reportedly carried this field and it was "very buggy." No trace exists in this repo's git history — reviving it is viable only as a verified hint, never as a source of truth, with documented failure modes (schema refactor moves a message and staleness goes undetected; `MergeInto`'s early-return path can leave the field silently partial; version skew from hand-edited `materialized_config/` files, a documented supported workflow).

**Defer (v2+ / not justified by current data):**
- Persisted symbol/descriptor-set index (Bazel-descriptor-set-cache-shaped). Every eager-index variant measured on the 799-proto corpus blows the 200ms budget alone (`ParseFilesButDoNotLink`: 1,286ms; naive lexical scan: 623ms) — not a viable hot-path mechanism regardless of persistence, only relevant if the hint/heuristic/scan chain measurably misses in production.

### Architecture Approach

No new component is warranted. The two structures that already exist ARE the lazy cache — they only need to stop being filled all-at-once: `DescriptorRegistry.FileRegistry` (file-parse-by-path cache, already the memo `LookupImport` checks) and `Parser.FilesResolver`/`LocalResolver` (symbol/type views, currently one-shot snapshots that must become incrementally grown via already-public `RegisterFile`/`RegisterMessage`/`RegisterEnum`). `DescriptorRegistry.MessageRegistry` (jhump's `msgregistry`) needs **no change** — it's already incremental (`AddFile` per parsed file) and already internally mutex-guarded.

**Major components and what changes:**
1. `ModuleService.GetProtoRegistry()` — stops bulk-parsing `src/`; returns a registry configured with import paths but empty of content. Caching in `m.cachedRegistry` stays, but its check-then-set is an unsynchronized race that must be locked properly now that concurrency actually matters.
2. `DescriptorRegistry` — gains a synchronized, single-file on-demand parse method that does **not** touch `localFiles` (that field is `Store()`'s bookkeeping for `mod sync` and is reset on every `Parse()` call — routing the lazy path through `Parse()` would silently drop earlier files from `localFiles`, a landmine for a feature nothing currently uses but that must not be planted).
3. `Parser` — `FilesResolver`/`LocalResolver` become incrementally grown; `ParseFilesX` gains a parse-and-register-on-miss path, guarded by a mutex; `ReadConfig` gains a cheap raw-JSON pre-pass that extracts `proto_file` via plain `encoding/json` (which doesn't need `Any` semantics to skip over the `value` field) before the real `protojson.Unmarshal` fires. Both `ParseFilesX` and `ReadConfig`'s public signatures stay stable — no consumer-facing changes propagate.
4. `ModuleService.Sync()`/`GenFileDescriptorSet()` (`mod sync`) — unchanged, stays fully eager on its own disposable registry instance, already structurally isolated from `m.cachedRegistry` today. This isolation must not be undone.
5. The 5 consumer packages named in PROJECT.md (compiler, mutation server, inserter, agent filekv, mutate CLI) — **plus a 6th call site not previously named**, `starlark_loader.go`'s `loadMutable`, which resolves type URLs twice (once via `LocalResolver`, once via `MessageRegistry`) on the compiler's own hot path. PROJECT.md's "five consumers" framing is therefore incomplete — treat it as six, with `loadMutable` tightly coupled to the compiler's speed-critical path rather than grouped with the four correctness-only consumers.

**Concurrency is the load-bearing constraint throughout:** every daemon consumer (compiler service via errgroup-fanned `CompileFiles`, agent filekv serving N concurrent client streams, the mutation server across all RPCs) shares one long-lived `Parser`/registry today, safely, only because nothing mutates those structures after construction. The moment lazy-fill writes into those maps mid-request, every one of `DescriptorRegistry.FileRegistry`, `MessageRegistry`, `Parser.FilesResolver`, `LocalResolver` becomes a concurrently-written structure with zero existing internal synchronization — a crash risk (`fatal error: concurrent map writes`), not merely a correctness bug. `golang.org/x/sync/singleflight` (dependency-free relative to the repo's existing `errgroup` import from the same module) is suggested as the natural mechanism, solving both the crash and redundant concurrent re-parsing of the same cold file in one move.

### Critical Pitfalls

1. **Mutation server `Init()` ranges the registry once, before `Serve()`, to discover and register gRPC services (`server/server.go:172,324-389,611`)** — under a lazy registry, at `Init()` time nothing under `src/` has been demanded yet, so every custom mutation service silently fails to register, for the process's entire lifetime, with no per-request recovery (`grpc.Server.RegisterService` panics if called after `Serve()`). This is orchestrator-verified as the most severe finding in the set, worse than the already-fixed `loadValidators` bug because it degrades an entire serving surface permanently rather than one config's validation. Must be fixed as a blocking co-requirement of the same phase that makes the registry lazy — either keep this one enumeration on an explicit eager filesystem walk (mirroring the `loadValidators` fix), or defer the registration decision until an eager one-time parse before `Serve()`.

2. **`GenReflectionUI`'s 5-second ticker is a second, recurring consumer of the exact same stale-resolver problem (`server/server.go:560-596`)** — walks `mutable_config/` and resolves each file's `Any` type URL through the same point-in-time `LocalResolver` snapshot. Unlike the startup-only mutation-server bug, this fails silently and *repeatedly* (an `Error`-level log easy to miss at lower log verbosity, and inconsistent error handling: one failure mode aborts the whole walk, the other silently skips just that file). Same fix as the `proto_file`-driven resolution work — this is a 6th named call site for that shared mechanism (5th of the daemon/CLI consumers plus `loadMutable`), not a separate problem, and should be explicitly enumerated so it isn't missed.

3. **`ModuleService.cachedRegistry` is a process-lifetime singleton — one eager fallback anywhere poisons every future compile silently.** If any lazy-fill fallback is implemented by calling the old eager `Import()` on the same cached registry, the entire registry becomes fully populated and stays that way for the rest of the process's life. Dangerously, a scaling benchmark that constructs a **fresh** `Compiler` per iteration (as the current benchmark shape does) will never observe this — the regression is invisible to exactly the test suite built to catch regressions, and only shows up in long-running `devserver`/mutation-server processes. Requires a dedicated long-running-process test as an explicit acceptance criterion, distinct from the scaling-ratio test.

4. **Concurrent compiles share one `Parser`/registry with zero synchronization, safe today only because nothing mutates after construction.** `go test -race` will reliably catch this once a concurrent-compile test exists — but no such test exists in this repo today (`CompileFiles`/`errgroup` currently have zero test coverage of concurrent lazy-load races). This ships as a production panic, not a CI failure, unless the phase's test suite adds the first-ever concurrent-compile test alongside the fix.

5. **`add_validator`'s last-write-wins map (`compiler/lib/starlark_functions.go:36-63`) is pre-existing and load-order-sensitive, but must not be re-coupled to registry order.** Not caused by, and out of scope to fix as part of, this milestone — but the risk specific to *this* milestone is a well-intentioned refactor "simplifying" the already-fixed `loadValidators` filesystem walk back into a registry range because it looks more consistent with the new lazy proto loading. Guard with a one-line comment/review checklist item, not a new phase.

**Verified NOT broken (carry forward to avoid over-scoping):**
- **`protovalidate-go` custom-option/extension interpretation (`buf.validate.field`) is unaffected by `src/` laziness.** Traced end to end: the global-registry seed (`buf/validate/*`, `google/*`, `protoconf/v1/*`) is unconditional, happens before any `src/` import, and is untouched by whatever lazy mechanism gets built. A field can't declare `(buf.validate.field)` without importing the file that defines it, so that file is always structurally present by the time the annotated file parses. The only real risk is a *future* change making this global seed itself lazy/conditional — recommend pinning with a regression test, not touching it as part of this work.
- **Compiled output determinism is already order-independent.** `DescriptorRegistry.Store()` explicitly sorts keys before serializing and re-sorts before hashing; `GetFileDescriptorSet()` iterates an already-unordered map today, eager or lazy makes no difference. The only load-order-sensitive correctness issue found anywhere is `add_validator` (already known, already characterized, pre-existing).

## Implications for Roadmap

**Reconciling build orders:** ARCHITECTURE.md proposes 4 phases; PITFALLS.md's pitfall-to-phase mapping largely folds into the same 4 but calls out pitfall 1 (mutation server `Init()`) as a "blocking co-requirement" of phase 1 rather than a separate phase, and pitfall 3 (`GenReflectionUI`) as belonging to phase 3 as a named 6th consumer. These are consistent, not conflicting — the recommendation below merges them into ARCHITECTURE.md's 4-phase shape with PITFALLS.md's blocking annotations folded in explicitly, since ARCHITECTURE.md's ordering is dependency-driven (each phase requires the primitive the previous phase built) and PITFALLS.md's annotations are about what must not be deferred within a phase, not a competing sequence.

**Two items are already shipped and are NOT phases of this milestone:** the `loadValidators` filesystem-walk fix (quick 260904-f5j) and the pinned synthetic benchmark corpus + scaling gate (quick 260904-fwk). Do not replan either.

**Correctness vs. speed, made explicit:** of the four phases below, only Phase 1 and Phase 2 move the measured numbers (the 4,639ms eager parse and the 260ms resolver rebuild, respectively — together the entire gap between 6.97s and ~200ms). Phase 3 is pure correctness work protecting the 4-6 non-compiler consumers from silently breaking under the change Phase 1/2 makes; its success criterion is "no regression, no silent wrong answers," never a further speed target. Phase 4 is decision-making and verification, not new mechanism.

### Phase 1: Concurrency-safe lazy `DescriptorRegistry` core
**Rationale:** Foundation everything else calls into; no dependency on anything else in this list. This is where the 4,639ms cost lives (94% of current wall time) and where the crash-risk concurrency hazard (Pitfall 3/4 above) must be closed before any consumer starts writing into shared state mid-request.
**Delivers:** `GetProtoRegistry()` stops bulk-walking `src/`; a new single-file on-demand parse method on `DescriptorRegistry` that never touches `localFiles`; `sync.RWMutex` (or `singleflight`) guarding `FileRegistry`/`MessageRegistry` writes and the `cachedRegistry` check-then-set.
**Avoids:** singleton poisoning (requires a long-running-process test, not just the scaling benchmark), concurrent writes (requires the first-ever concurrent-compile `-race` test in this repo).

### Phase 2: Lazy resolver views on `Parser` (resolves the compiler's own hot-path type-URL case)
**Rationale:** Depends on Phase 1's on-demand parse primitive. This phase alone delivers the milestone's headline win — the compiler is the only consumer on the 200ms budget (BASELINE.md), and path-based `load()` resolution plus `loadMutable`'s type-URL resolution together close the full 6.97s->~200ms gap.
**Delivers:** `FilesResolver`/`LocalResolver` seeded near-empty at construction, grown incrementally via `RegisterFile`/`RegisterMessage`/`RegisterEnum` on `ParseFilesX` miss, under the Phase 1 mutex.
**Implements:** the "Shape 3" resolver design from ARCHITECTURE.md — one resolver wrapper answering both direct-path lookups and any nested-`Any` lookup transparently.
**Note:** `loadMutable` is compiler-side and tightly coupled to this phase — consider sequencing it together with Phase 2 rather than deferring it to Phase 3's "other consumers" bucket, since it sits inside the same 200ms-budgeted call path as `load()`.

### Phase 3: `ReadConfig` pre-pass + type-URL fallback chain for the remaining consumers
**Rationale:** Depends on Phase 2 — adds more *triggers* for the same growable-resolver mechanism, not a new one. This phase must explicitly enumerate **six** call sites, not five: mutation server construction (`Init()`'s service-registration path — see below), inserter, agent filekv, mutate CLI, and `GenReflectionUI`'s 5-second ticker (newly identified, previously unnamed in PROJECT.md).
**Delivers:** raw-JSON `proto_file` peek added to `Parser.ReadConfig` (Shape 1); package->directory->scoped-lexical-scan->loud-eager-fallback chain as `FindMessageByURL`'s miss behavior (Shape 2); ideally factored into one shared helper (e.g. `parser.ResolveAnyType(protoFile, typeURL)`) rather than duplicated five times, since PITFALLS.md flags duplication as its own risk.
**Structural exception — must be called out separately in phase planning:** `ProtoconfMutationServer.Init()`'s gRPC service registration is **not** a lookup-shaped consumer like the others — it is a one-time, whole-repository service *enumeration* that must complete before `Serve()` starts and cannot use per-request lazy recovery. This needs either an explicit eager filesystem walk (mirroring the `loadValidators` pattern) or a deferred-`Init()`-until-first-compile design requiring one eager parse before `Serve()` — either way, decide this explicitly in this phase rather than letting it silently inherit the lazy path's semantics.
**Avoids:** silent service-registration loss (most severe finding), stale resolver snapshot including the recurring `GenReflectionUI` ticker case.

### Phase 4: Decide error-surface change, verify, and flip the gate
**Rationale:** Can only happen once Phases 1-3 exist to measure against; this is the milestone's actual definition of done.
**Delivers:** an explicit written decision on whether `protoconf compile` should keep validating the whole `src/` tree for broken imports now that lazy compiles never touch unreached broken protos (optionally behind a flag) — a CI-behavior change that must be documented, not silently absorbed; flip `TestCompilerStartupScaling`'s `t.Skipf` to `require.LessOrEqual`.
**Test note:** this is the one test in the whole milestone with a hard numeric pass/fail gate (<=2.0x ratio) already defined; everything else in Phases 1-3 is either fails-before-fix (mutation server Init, stale-resolver, concurrent-write pitfalls — write the test, watch it fail on today's tree once the lazy change lands without the fix) or characterization-only guards protecting already-correct behavior (extension interpretation, output determinism, `add_validator`'s existing pinned test) that should stay green throughout and need no new assertions, only inclusion in the regression suite.

### Phase Ordering Rationale

- Phase 1 before Phase 2 before Phase 3 is a strict dependency chain: Phase 2 needs Phase 1's single-file parse primitive; Phase 3 needs Phase 2's growable-resolver mechanism to attach new trigger points to.
- Phase 1 and Phase 2 together are the entire speed story (4,639ms + 260ms of the 6.97s baseline); Phase 3 is entirely correctness, protecting consumers that were never on the 200ms budget from regressing under the change.
- The mutation server's `Init()` problem is placed inside Phase 3 rather than Phase 1 because it is a distinct mechanism (explicit eager enumeration, not lazy resolution) — but it must not slip to a later milestone; it is a blocking correctness gap the moment Phase 1/2 ship, since a lazy registry with unfixed `Init()` silently drops every custom mutation service from that point forward.

### Research Flags

Needs deeper research during phase planning:
- **Phase 3 — type-URL resolution mechanism.** FEATURES.md's confidence is explicitly LOW-by-policy (uncurated web research, no verified-source integration) even though individual claims are MEDIUM-grade official-docs sourced. The package->directory heuristic, the revived `proto_files` field, and the scoped-lexical-scan fallback are all presented as a menu, not a decision — by explicit user instruction, this choice is deferred to phase planning. Do not let the roadmapper or a plan pick a "winner" without surfacing this trade-off table again at that time.
- **Phase 3 — mutation server `Init()` fix shape.** Two viable shapes (explicit eager filesystem walk vs. deferred-init-before-Serve) are presented without a clear winner; needs a design decision during planning, informed by how much startup latency an eager service-discovery walk actually costs (likely small — service declarations are a much smaller scan than full linking — but unmeasured).

Phases with well-documented, low-ambiguity patterns (standard implementation, skip research-phase):
- **Phase 1:** the concurrency primitive (mutex or singleflight around existing maps) and the single-file parse method are both fully specified against verified stdlib/jhump API contracts in STACK.md — no open design questions.
- **Phase 2:** the resolver-growth mechanism (`RegisterFile`/`RegisterMessage`/`RegisterEnum`) is a direct, already-precedented API usage (identical to how every generated `.pb.go` populates the global registry).
- **Phase 4:** mechanical — flip a skip to an assertion once the prior phases are measured.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Every claim verified against pinned module source with file:line citations, not recalled; no new dependency proposed |
| Features (type-URL mechanism) | LOW-by-policy / MEDIUM-per-claim | Uncurated web research (official docs only, single-pass, not cross-verified by a second independent source); the domain itself has no direct precedent in any ecosystem checked, which is a finding, not a gap |
| Architecture | HIGH | Derived directly from this repo's own source, file:line cited throughout; cross-checked against prior milestone research (BASELINE.md, OPTIONS.md) for consistency |
| Pitfalls | HIGH | Grounded in file:line references from this codebase; two "verified not broken" findings were actively investigated, not assumed, which increases confidence in the overall scope boundary |

**Overall confidence:** HIGH — the performance diagnosis and fix shape are unusually well-measured for a research phase (exact millisecond breakdowns, CPU profiles, negative results already ruled out). The one genuine open question (type-URL mechanism) is explicitly and deliberately left open by the user, not a research gap to close before planning.

### Gaps to Address

- **Type-URL resolution mechanism choice** (Phase 3): present the FEATURES.md option table to the user/roadmapper during phase planning rather than defaulting silently to one option; this is a user decision point, not a research gap.
- **Mutation server `Init()` fix shape** (Phase 3): needs a design decision (eager filesystem walk vs. deferred init) informed by measuring the cost of an eager service-discovery-only walk, which was not itself measured in this research round — only full parse+link was measured.
- **Revived `proto_files` field, viability as a verified hint:** the "very buggy" prior-version report is user-supplied domain knowledge with no trace in this repo's git history — treat as an unverified hint about design risk, not a specification of what went wrong; if this option is pursued, budget time to independently rediscover the failure modes rather than assuming the historical bug is fully characterized here.
- **`add_validator` last-write-wins bug:** confirmed pre-existing and out of scope, but flagged as a backlog item for a future, separate milestone decision (whether to error on duplicate registration) — not something this roadmap needs to solve, only something it must not accidentally destabilize by recoupling validator-discovery order to proto-registry order.

## Sources

### Primary (HIGH confidence)
- This repository's own source, read directly: `compiler/lib/module_service.go`, `utils/utils.go`, `compiler/lib/parser/parser.go`, `compiler/lib/starlark_loader.go`, `compiler/lib/compiler.go`, `compiler/service.go`, `server/server.go`, `inserter/inserter.go`, `agent/filekv/filekv.go`, `agent/kv_agent_impl.go`, `mutate/mutate.go`, `devserver/command.go`, `compiler/lib/starlark_functions.go`, `compiler/lib/starlark_loader_test.go`
- `$(go env GOMODCACHE)` pinned module source: `jhump/protoreflect@v1.16.0` (`desc/protoparse/parser.go`, `desc/descriptor.go`, `desc/load.go`, `dynamic/msgregistry/message_registry.go`), `google.golang.org/protobuf@v1.34.1` (`reflect/protoregistry/registry.go`, `reflect/protodesc/desc.go`, `encoding/protojson/{encode,decode}.go`, `internal/errors/errors.go`), `bufbuild/protocompile@v0.10.0` (`resolver.go`, `compiler.go`, `linker/files.go`)
- `.planning/research/compiler-performance/BASELINE.md`, `OPTIONS.md` — this project's own prior measurements (stage breakdown, CPU profile, rejected-option ledger)
- `.planning/PROJECT.md` — milestone definition

### Secondary (MEDIUM confidence)
- [Protocol Buffers Language Specification (Proto3)](https://protobuf.dev/reference/protobuf/proto3-spec/) — confirms no grammar rule ties `package` to file path
- [Protocol Buffers Style Guide](https://protobuf.dev/programming-guides/style/) — discourages package/directory coupling for deeply nested paths
- [Buf Docs — Compilation and descriptors](https://buf.build/docs/reference/compilation-and-descriptors), [Lint rules](https://buf.build/docs/lint/rules/) (`PACKAGE_DIRECTORY_MATCH`), [Files and packages](https://buf.build/docs/reference/protobuf-files-and-packages/), [Managing dependencies](https://buf.build/docs/bsr/module/dependency-management/)
- [GRPC Server Reflection Protocol](https://grpc.github.io/grpc/core/md_doc_server-reflection.html), [Bazel Protocol Buffer Rules](https://bazel.build/reference/be/protocol-buffer) / [ProtoInfo](https://bazel.build/versions/6.5.0/rules/lib/ProtoInfo)
- `grpc.Server.RegisterService` "must be called before Serve" — grpc-go's documented contract, not independently re-verified against grpc-go source this session (flagged MEDIUM on exact panic behavior, HIGH on the ordering requirement itself, since this codebase's own `Init()`-before-`Serve()` structure already assumes it)

### Tertiary (LOW confidence)
- User-supplied domain knowledge: an earlier protoconf version's `repeated string proto_files` field was "very buggy" — no corroborating trace in this repo's git history; treat as a design-risk hint, not a verified specification

---
*Research completed: 2026-09-04*
*Ready for roadmap: yes*
