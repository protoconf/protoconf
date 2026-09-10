# Phase 12: Growable Resolver Views & Race Safety - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-09-07
**Phase:** 12-Growable Resolver Views & Race Safety
**Areas discussed:** Growth mechanism + locking, Blast radius, ParseFilesX shape, RSLV-02 observable

---

## Growth mechanism + locking

### Q1 — How do FilesResolver and LocalResolver come to reflect files parsed after construction?

| Option | Description | Selected |
|--------|-------------|----------|
| Miss-fallthrough wrapper | Wrap both in a resolver reading the immutable snapshot then falling through to `DescriptorRegistry` — the `RegistryTypeResolver` shape. No mutation, no lock. Cost: enumeration only sees the snapshot. | |
| Real incremental registration | `recordFileLocked` pushes into `Files.RegisterFile` and `Types.RegisterMessage`. True protoregistry semantics. Cost: locking on every read, exported fields become methods, `server.go:292-297` changes. | |
| Hybrid: grow Files, wrap Types | Register files incrementally into `FilesResolver` behind a lock (where enumeration matters); leave `Types` as the Phase 11 fallthrough wrapper. | ✓ |

**User's choice:** Hybrid — grow Files, wrap Types.
**Notes:** Each mechanism is the cheapest one that serves its consumer; types are already correctly served by something that mutates nothing.

### Q2 — How do readers get synchronised against mid-compile mutation?

| Option | Description | Selected |
|--------|-------------|----------|
| Wrap in a locked type | New parser-owned type holding `*protoregistry.Files` + `RWMutex`, methods for Find/Range/Register; field type changes, `server.go` call sites adapt. | |
| Lock inside DescriptorRegistry | Keep the raw field; registry registers under the `d.mu` it already owns. One lock, but exported-field readers stay unsynchronised. | |
| You decide | Whichever holds up under `-race` with the smallest diff once call sites are traced. | ✓ |

**User's choice:** Claude's discretion.

### Q3 — What gets registered: the requested file, or its transitive closure?

| Option | Description | Selected |
|--------|-------------|----------|
| Whatever recordFileLocked records | Same insert points as the existing recursive dependency walk; closure comes free, already-present skip satisfies RSLV-02. | |
| Requested file only | Fewer entries; needs another path to satisfy `protodesc.NewFile`'s import requirement. | |
| You decide | Whichever keeps `protodesc.NewFile` satisfiable without a second resolution path. | ✓ |

**User's choice:** Claude's discretion.

### Q4 — Does the D-03 `ParseAll` eager fallback also land in `protoregistry.Files`?

| Option | Description | Selected |
|--------|-------------|----------|
| Yes — same insert point, no special case | Uniform and correct; the fallback pays a whole-tree registration, a cliff already known and slated for removal by Phase 13's index. | ✓ |
| No — fallback stays MessageRegistry-only | Path resolver never eats the whole-tree cost, but `FileRegistry` and `FilesResolver` diverge — a subtle invariant to carry forward. | |
| Make the fallback loud first | Surface fallback firing + file count alongside the existing `LoadedFileCount` instrumentation. | |

**User's choice:** Yes — same insert point, no special case.
**Notes:** A set divergence between `FileRegistry` and `FilesResolver` was judged the worse thing to carry forward than an already-slow path staying slow.

---

## Blast radius

### Q1 — Who gets the growable FilesResolver this phase?

| Option | Description | Selected |
|--------|-------------|----------|
| Everyone — growth is unconditional | One `Parser`, one behavior; eager consumers never fire `ParseOne` so never see growth. Phase 14 verifies rather than changes. | |
| Compiler only — opt in | Gate growth to the compiler's construction path; guarantees zero change for the other five, matching Phase 11's scope boundary. | ✓ |
| You decide | Whichever keeps existing inserter/server/mutate tests green with fewer moving parts. | |

**User's choice:** Compiler only — opt in.

### Q2 — How is the compiler-only opt-in expressed?

| Option | Description | Selected |
|--------|-------------|----------|
| Derive it from ImportPaths | No new constructor, no flag: grow only when `registry.ImportPaths` is non-empty — the exact condition Phase 11 already uses to gate `ParseOne`. | ✓ |
| Separate constructor | `NewGrowableParserWithDescriptorRegistry` / a `With...` option; explicit at the call site, one constructor for Phase 14 to delete. | |
| You decide | Whichever reads more honestly once `ImportPaths`' population is traced. | |

**User's choice:** Derive it from `ImportPaths`.
**Notes:** Verified in-session — `ImportPaths` is written only at `compiler/lib/module_service.go:454`, under `m.lazyRegistry`, set only by `NewLazyModuleService`, called only by `NewCompiler`. The gate is an existing invariant, not a new one.

### Q3 — Does `Parser.FilesResolver`'s exported field type change?

| Option | Description | Selected |
|--------|-------------|----------|
| Field type stays, growth hides behind it | Zero API churn; but exported-field readers sit outside the lock, so `-race` proof must show none exists on a lazy registry. | |
| Field becomes a locked interface | Readers structurally forced through the lock; type change ripples through four packages in a "compiler only" phase. | |
| You decide | Whichever is provable under `-race` without dragging non-compiler consumers in. | ✓ |

**User's choice:** Claude's discretion.

---

## ParseFilesX shape

### Q1 — What shape does `ParseFilesX` take?

| Option | Description | Selected |
|--------|-------------|----------|
| Collapse to registry → ParseOne | Delete the `FilesResolver` branch and its `WrapFile`/`CreateFileDescriptor` fallback; every descriptor is canonical, ~30 lines deleted. | |
| Keep the branch, return canonical | Keep the lookup but return the `FileRegistry` pointer on a hit; fixes identity, leaves a provably redundant branch behind. | |
| You decide | Trace what the branch actually resolves that the registry doesn't, then cut or keep on that evidence. | ✓ |

**User's choice:** Claude's discretion.

### Q2 — What must the RSLV-03 A → B → A test assert?

| Option | Description | Selected |
|--------|-------------|----------|
| Pointer identity plus correct output | Correct materialized JSON AND the same descriptor pointer for A both times; the pointer assertion is what catches the regression. | |
| Correct output only | Tests observable behavior; a re-parse minting a second descriptor for A still passes — the exact bug being guarded. | |
| You decide | Whichever fails loudly if the canonical-pointer contract regresses. | ✓ |

**User's choice:** Claude's discretion.

### Q3 — The dead `config.protoResolver` field

| Option | Description | Selected |
|--------|-------------|----------|
| Delete it | Two-line removal; a stale snapshot reference in the struct this phase makes grow is an invitation to wire it up wrong later. | |
| Leave it, note it | Out of stated scope; record as deferred cleanup. | |
| You decide | Delete if nothing in-tree or in Phase 13/14's shape wants it; otherwise point it at the growable view. | ✓ |

**User's choice:** Claude's discretion.

---

## RSLV-02 observable

### Q1 — What makes "no rebuild over previously-loaded files" observable and gate-able?

| Option | Description | Selected |
|--------|-------------|----------|
| Registration counter on the registry | Exported like `LoadedFileCount`; test asserts registrations grow by each file's own closure, not N×(files-so-far). | ✓ |
| Structural — resolver identity never changes | Cheap and unambiguous about rebuilds; says nothing about per-registration work. | |
| Timing, via the scaling gate | Measures what the user feels; flakiest gate shape, can't distinguish a rebuild from any other regression. | |

**User's choice:** Registration counter on the registry.

### Q2 — Is the counter operator-visible or test-only?

| Option | Description | Selected |
|--------|-------------|----------|
| Test-only, exported method | Read by tests; no log line, no CLI surface. Operators already get the LAZY-05 loaded-file count. | ✓ |
| Also in compiler output | Visible while debugging a slow compile, citable in Phase 15's numbers; one more number in every compile. | |
| You decide | Whichever Phase 15 can cite without adding noise. | |

**User's choice:** Test-only, exported method.

### Q3 — What does this phase add to `TestConcurrentCompile` for SAFE-01?

| Option | Description | Selected |
|--------|-------------|----------|
| Extend the existing test | Add resolver-view assertions to the existing 8-goroutine test; one fixture covers both parse and growth paths. | |
| New dedicated test | Separate test racing `RegisterFile` against `FindFileByPath`; isolates the new failure mode, two fixtures. | |
| You decide | Whichever makes a resolver-growth race fail loudly under `go test -race`. | ✓ |

**User's choice:** Claude's discretion.

---

## Claude's Discretion

- Locking / API shape for the growable `FilesResolver` (constraint: provable under `-race` without dragging non-compiler consumers in).
- Whether `Parser.FilesResolver`'s exported field type changes.
- Registration granularity — requested file vs. transitive closure (constraint: `protodesc.NewFile`'s dependency requirement satisfiable without a second resolution path).
- `ParseFilesX` shape — cut or keep the `FilesResolver` branch, on traced evidence.
- RSLV-03 test assertions (constraint: must fail if the canonical-pointer contract regresses).
- SAFE-01 test placement — extend `TestConcurrentCompile` or add a dedicated race test.
- The dead `config.protoResolver` field — delete or repoint.

## Deferred Ideas

- Operator-visible resolver-churn metric — revisit if Phase 15's numbers want it.
- Removing the D-03 `ParseAll` whole-tree fallback cliff — Phase 13's symbol index.
- `ModuleService.GetProtoFilesRegistry()` deletion — zero callers today; Phase 14 or a `/gsd-quick`.
- Collapsing the two resolver mechanisms into one — revisit if Phase 13's index makes the Types fallthrough redundant.

## Findings surfaced during the scout (not user decisions)

- `ParseFilesX` (`compiler/lib/parser/parser.go:123`) mints a non-canonical descriptor pointer via `desc.WrapFile` on a `FilesResolver` hit — latent today, reachable once the resolver grows.
- `config.protoResolver` (`compiler/lib/config.go:27`, assigned `compiler/lib/compiler.go:356`) is assigned and never read.
- `ModuleService.GetProtoFilesRegistry()` (`compiler/lib/module_service.go:403`) has zero callers in the tree.
