# Requirements: Protoconf — Compiler Startup Performance (v2.0)

**Defined:** 2026-09-04
**Core Value:** Every component must be testable, consistent, and free of runtime surprises
**Milestone goal:** `protoconf compile` completes in under 200ms on a repository of any size, by parsing and linking only the protos a config actually reaches.

Milestone v1.0 (Quality & Consistency Overhaul) shipped complete — its 44 requirements are archived in `.planning/milestones/v1.0-phases/` and its roadmap history in git.

## v2.0 Requirements

### Lazy Registry Core

- [ ] **LAZY-01**: `GetProtoRegistry()` returns without bulk-walking, parsing or linking `src/` — a repository's proto count no longer determines compiler construction cost
- [ ] **LAZY-02**: A proto file is parsed and linked the first time it is requested by path, and memoised so a second request costs a map lookup
- [ ] **LAZY-03**: On-demand single-file parsing never mutates `localFiles`, so `registry.Store` continues to serialise a complete `.fds` for `mod sync`
- [ ] **LAZY-04**: `protoconf mod sync` still parses and links the whole module tree, writing `.fds` cache files identical to today's
- [ ] **LAZY-05**: An operator can see how many proto files a compile actually loaded, so an unexpectedly wide load is visible in the field rather than only under a profiler

### Resolver Views

- [ ] **RSLV-01**: `FilesResolver` and `LocalResolver` reflect files parsed after their construction, rather than a snapshot taken at construction time
- [ ] **RSLV-02**: A newly parsed file is registered into the resolvers incrementally, without rebuilding the full `FileDescriptorSet`
- [ ] **RSLV-03**: Compiling a config that `load()`s a proto, then a second proto, then re-reads the first, resolves all three correctly

### Type-URL Resolution

Design decision (2026-09-04, user): resolve type URLs through an **exact symbol
index built by parsing without linking**, and drop `ProtoconfValue.proto_file`
as a resolution mechanism. Measured on the 799-proto terraform corpus: unlinked
parse costs 1.41s against 4.64s linked (3.3x cheaper) and yields 138,554 message
symbols including nested types, resolving every probe exactly.

This supersedes the package-name heuristic, the scoped lexical scan, post-parse
symbol verification, and the revived `repeated proto_files` field — all four
existed only to *guess* a declaring file, and an exact index makes them
unnecessary.

Historical note: lazy loading was protoconf's original design, and `proto_file`
exists because of it. It failed when `Any` was introduced and the inserter could
not resolve nested types — lazy-by-path cannot answer a symbol it was never asked
to load. Going fully eager was the workaround, and it costs 4.6s per compile. The
index is the piece that was missing; this closes the original gap rather than
repeating it.

Roadmap note (2026-09-04, revision): the index (TYPE-01/02/04/05/06/07) and the
shared resolution path that consults it (TYPE-03/08/09) ship together in one
phase (Phase 13), not two — an index with nothing consulting it yet has no
observable success criteria, only an artifact.

- [ ] **TYPE-01**: A symbol index maps every message symbol under `src/`, nested types included, to the file that declares it
- [ ] **TYPE-02**: The index is built by parsing without linking, so its cost is proportional to parsing alone
- [ ] **TYPE-03**: A type URL nested inside an `Any` field, at any depth, resolves through the index — this is the case that broke the original lazy design
- [ ] **TYPE-04**: Only the files actually referenced by resolved symbols are linked; building the index links nothing
- [ ] **TYPE-05**: The index is persisted under `.protoconf_cache` and keyed by content, so a warm run does not rebuild it
- [ ] **TYPE-06**: A change to any `.proto` under `src/` invalidates the cached index, and a stale index is never served
- [ ] **TYPE-07**: The index is built on first need, so a consumer that resolves no unknown symbol never pays for it
- [ ] **TYPE-08**: Type-URL resolution is a single shared code path used by every consumer, not duplicated per call site
- [ ] **TYPE-09**: `parser.ReadConfig` resolves every nested `@type` in a materialized config, at any depth, through that path

### Consumer Correctness

- [ ] **CONS-01**: The mutation server registers every custom gRPC service declared under `src/` — service discovery does not depend on what a config happened to load
- [ ] **CONS-02**: The inserter reads and inserts materialized configs, resolving their types correctly
- [ ] **CONS-03**: The agent's filekv store serves configs to subscribed clients, resolving their types correctly
- [ ] **CONS-04**: `GenReflectionUI`'s periodic `mutable_config/` walk resolves every file's type, and reports rather than silently skips a file it cannot resolve
- [ ] **CONS-05**: Compiling a config that loads a mutable config resolves the mutable value's type, and every nested `Any` within it, correctly

### Safety Under Concurrency

- [ ] **SAFE-01**: Concurrent compiles against one shared compiler are race-free under `go test -race`
- [ ] **SAFE-02**: Concurrent requests against a long-lived mutation server or agent are race-free under `go test -race`
- [ ] **SAFE-03**: A long-running process serving many different configs does not accumulate the full registry — the lazy set stays proportional to what was actually demanded

### Verification

- [ ] **GATE-01**: `TestCompilerStartupScaling`'s allocation ratio at n=50 vs n=400 is at or below 2.0x, and its `t.Skipf` branch is replaced by `require.LessOrEqual`
- [ ] **GATE-02**: Compiling the in-repo 800-proto synthetic corpus completes end-to-end in under 200ms, asserted in CI
- [ ] **GATE-05**: The real protoconf-terraform corpus (799 protos, 6.97s at baseline) is measured once at milestone close and the number recorded as evidence — not a CI gate, since that repo is not checked in and drifts independently. Note the synthetic corpus is ~3x cheaper per file, so GATE-02 passing is a weaker claim than this one
- [ ] **GATE-03**: The behaviour change to compile-time validation — a broken proto that no config loads is no longer reported — is decided deliberately and documented for operators
- [ ] **GATE-04**: Every pre-existing test stays green, with the four validator cases and `field_type_any_test.pconf` unchanged in outcome

## Future Requirements

Acknowledged, deferred beyond this milestone.

### Caching

- **CACHE-01**: Persist *linked descriptors* on disk so warm compiles skip parsing entirely — distinct from TYPE-05's symbol-index cache, which is in scope; this optimises the ~3ms lazy path, not the 4.6s eager one

### Migration

- **MIGR-01**: Migrate from `jhump/protoreflect/dynamic` to `dynamicpb` — carried over from v1.0, large scope, touches `compiler/starproto` extensively

### Known Defects

- **BUG-01**: `add_validator` keeps only the last validator per message (`starlark_functions.go:36-63`), so `test.proto-validator`'s three registrations on `ValidateMe` collapse to one and three test fixtures pass for the wrong reason. Pre-existing, characterized by `starlark_loader_test.go`, out of scope here.
- **BUG-02**: `mutate/mutate.go` converts `TYPE_SINT32` to `uint32` instead of `int32` (08-REVIEW.md WR-02)
- **BUG-03**: `go vet` copylocks at `compiler/lib/compiler.go:355` (broken-windows ledger id 2)

## Out of Scope

| Feature | Reason |
|---------|--------|
| Persisted linked-descriptor cache | Optimises the ~3ms lazy path, not the 4.6s eager one; `.fds` machinery already half-exists if warm-start ever justifies it |
| Parallelising the eager parse | Divides 4.6s by core count at best — still 3-4x over budget, burning every core on ~99% discarded work |
| Patching or bumping `protocompile` for the linker lookup | `linker.Files.FindFileByPath` scans a file's direct deps, not all 864; v0.14.1 is byte-identical. There is no lookup to fix |
| Adding `protoresolve` | Not in the dependency graph; a new dependency for a problem hand-rolled caching already solves |
| Making `mod sync` lazy | It serialises the registry to `.fds`; a lazy registry would write a truncated cache that then loads clean and is wrong |
| Migrating off `jhump/protoreflect` | Deferred v1.0 item; the profile does not implicate the wrappers — cost is in protocompile's parse+link, which both APIs sit on |
| Fixing `add_validator`'s last-write-wins clobbering | Pre-existing and unrelated to laziness; a product decision of its own (BUG-01) |
| Making the global-registry seed lazy | Verified to be what keeps `buf.validate` extension interpretation correct; touching it is the one way to break something currently safe |
| `ProtoconfValue.proto_file` as a resolution mechanism | Superseded by the exact symbol index. The field stays populated for compatibility, but nothing resolves through it — one mechanism, not two |
| Package-name → directory heuristic, scoped lexical scan, symbol verification | All existed to guess a declaring file; the index answers exactly. Package→directory is convention, not spec — protobuf mandates no such relationship |
| Reviving `repeated string proto_files` | Its only job was nested-`Any` provenance, which the index answers exactly and without the staleness and version-skew failure modes that sank it before |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| LAZY-01 | Phase 11 | Pending |
| LAZY-02 | Phase 11 | Pending |
| LAZY-03 | Phase 11 | Pending |
| LAZY-04 | Phase 11 | Pending |
| LAZY-05 | Phase 11 | Pending |
| CONS-01 | Phase 11 | Pending |
| RSLV-01 | Phase 12 | Pending |
| RSLV-02 | Phase 12 | Pending |
| RSLV-03 | Phase 12 | Pending |
| SAFE-01 | Phase 12 | Pending |
| TYPE-01 | Phase 13 | Pending |
| TYPE-02 | Phase 13 | Pending |
| TYPE-03 | Phase 13 | Pending |
| TYPE-04 | Phase 13 | Pending |
| TYPE-05 | Phase 13 | Pending |
| TYPE-06 | Phase 13 | Pending |
| TYPE-07 | Phase 13 | Pending |
| TYPE-08 | Phase 13 | Pending |
| TYPE-09 | Phase 13 | Pending |
| CONS-05 | Phase 13 | Pending |
| CONS-02 | Phase 14 | Pending |
| CONS-03 | Phase 14 | Pending |
| CONS-04 | Phase 14 | Pending |
| SAFE-02 | Phase 14 | Pending |
| SAFE-03 | Phase 14 | Pending |
| GATE-01 | Phase 15 | Pending |
| GATE-02 | Phase 15 | Pending |
| GATE-03 | Phase 15 | Pending |
| GATE-04 | Phase 15 | Pending |
| GATE-05 | Phase 15 | Pending |

**Coverage:**
- v2.0 requirements: 30 total
- Mapped to phases: 30
- Unmapped: 0 ✓

---
*Requirements defined: 2026-09-04*
*Last updated: 2026-09-04 — Phase 13/14 merged into single Phase 13 per user revision (v2.0, Phases 11-15)*
