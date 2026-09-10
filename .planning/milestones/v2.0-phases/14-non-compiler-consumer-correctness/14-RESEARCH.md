# Phase 14: Non-Compiler Consumer Correctness - Research

**Researched:** 2026-09-08
**Domain:** Go concurrency-safe lazy proto resolution across four non-compiler consumers (mutation server, inserter, agent/filekv, mutate CLI)
**Confidence:** HIGH — every claim below is grounded in a file:line read at HEAD this session; no external library research was needed (no new dependencies, no unfamiliar frameworks — this is entirely in-repo verification of code CONTEXT.md already scouted).

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** All four consumers construct via `NewLazyModuleService`. `inserter.go:192`, `server.go:295`, `filekv.go:70`, and `mutate.go:67` all flip. `mod/command.go:81` (`mod sync`) stays eager. `NewLazyModuleService`'s doc comment (`module_service.go:48-53`) is now wrong and must be rewritten by this phase.
- **D-02:** Failure surfaces at first request, loudly — no startup validation pass. This does not soften CONS-04: `GenReflectionUI` reports its failures (D-05) because it is already walking `mutable_config/` on a ticker, not because a validation pass was added.
- **D-03:** `parser.TypeResolver` resolves every type URL and every nested `Any`, in all four consumers: `inserter/inserter.go:369` and `:379`, `server/server.go:466`, `mutate/mutate.go:74-76`. `server/server.go:607-635` already on `TypeResolver` from Phase 13; no change. "No `LocalResolver` remains as a type-URL or `Any` resolution source" is a checkable end-state, distinct from the `ExtensionResolver` uses.
- **D-04:** The kept-alive discovery registry (D-06) is reflection's completeness backstop and nothing else. It must not become a second general-purpose resolver.
- **D-05:** `GenReflectionUI` completes the whole walk, aggregates failures, and logs on change. Stop aborting the walk (today a `FindMessageByURL`/`UnmarshalTo` failure does `return err`, terminating `filepath.WalkDir`, whose return is discarded at all four call sites: `server.go:196`, `:200`, `devserver/command.go:87`, `:96`). Aggregate every unresolvable path with its type URL and the Phase 13 D-02 diagnostic into one error, returned from `GenReflectionUI`. Log on change, not per pass (5-second ticker; re-log only when the failure set changes).
- **D-06:** gRPC reflection sees everything: keep `Init()`'s discovery registry alive. Retain the discovery registry past `Init()` and hand its files resolver to `reflection.NewServerV1` and `reflection.NewServer` (`server.go:433-444`). Costs no extra parse. Criterion 5's counter is measured on the *serving* registry (`s.parser`'s), never on the discovery registry.
- **D-07:** SAFE-02 gets both an end-to-end test (bufconn, N concurrent clients driving `MutateConfig`/`SubscribeForConfig` under `-race`, real interceptors) and a dedicated unpaced tight-loop test per consumer, sanity-checked by temporarily removing the lock and confirming `WARNING: DATA RACE` before restoring it.
- **D-08:** SAFE-03 gets both of criterion 5's clauses as separate assertions, using `LoadedFileCount()` (`utils/utils.go:512`). (a) Escalation guard: serve one config whose type only the symbol-index tier can resolve; assert `LoadedFileCount()` grows by that config's transitive closure alone. (b) Sequence guard: serve N distinct configs against one long-lived process; assert the count grows by a bounded delta per config and ends strictly below corpus size.

### Claude's Discretion

- How the discovery registry is retained (D-06) — a field on `ProtoconfMutationServer`, a second `*parser.Parser`, or handing only its `*protoregistry.Files` forward. Constraint: must not become reachable as a general type-URL resolver (D-04), and must not be read by any criterion-5 assertion (D-06).
- The shape of the aggregated CONS-04 error and its change-detection (D-05) — `errors.Join` over per-file errors vs. a typed aggregate carrying paths; failure-set comparison by sorted path list vs. a hash. Constraint: a config that changes from one failure reason to a *different* failure reason must re-log.
- Whether the four `GenReflectionUI` call sites start checking the returned error (`server.go:196/200`, `devserver/command.go:87/96`). Constraint: the ticker must keep running regardless — a failed pass may never stop the server or the loop.
- Whether `agent/filekv` needs any code change at all beyond D-01's construction flip. Constraint: verify with a test that exercises a type not resolved at construction (criterion 2), rather than assuming it.
- Test placement and fixtures for D-07/D-08 — which package, whether the synthetic corpus (`testdata.GenerateCorpus`) or a hand-built fixture backs the sequence guard. Constraint: the escalation guard needs a type the scan tier cannot resolve but the index tier can, so the fixture must be built against the Phase 13 tier boundary, not guessed.
- Whether `mutate`'s and `inserter`'s per-config failure aborts the run or skips the item. Current behavior is abort; D-02 makes the error loud either way. Constraint: do not silently change a CLI's exit-code behavior as a side effect of the resolver rewiring. **(Research correction: verified at HEAD, inserter's current behavior is skip-and-log-continue with exit 0, not abort — see Common Pitfalls #2. Only mutate currently aborts. Preserve each consumer's own current shape.)**

### Deferred Ideas (OUT OF SCOPE)

- On-demand `DescriptorResolver` for reflection (wrapping reflection's resolver so a miss triggers `ParseOne` and retries) — deferred in favor of D-06 because it is new machinery on an untested path; the principled upgrade if the retained discovery registry's memory footprint ever matters.
- Rendering unresolvable configs as visible broken entries in the grpcui example list — touches the `standalone.Example` rendering path, no criterion requires it, no test covers it.
- `compiler/lib/config.go:68`'s silent `ErrUnexpectedType` skip — in the *compiler*, not this phase's four consumers; stays deferred. Not a criterion; may be folded in if cheap.
- `devserver`'s three registries (lazy compiler + lazy mutation server + D-06's retained discovery parse, all in one process) — reviewed and accepted this session; `devserver` is a dev convenience, no criterion covers it.
- The scaling-gate flip and milestone numbers (Phase 15); `mod sync` (stays eager by definition).

</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-------------------|
| CONS-02 | The inserter reads and inserts materialized configs, resolving their types correctly | D-01/D-03 rewiring at `inserter.go:192,369,379`; existing regression test `inserter_test.go:80-95` becomes the real lazy-path proof (Code Examples) |
| CONS-03 | The agent's filekv store serves configs to subscribed clients, resolving their types correctly | Verified `filekv.go:111` already on `TypeResolver` via `ReadConfig` (TYPE-09) — only `filekv.go:70`'s construction flip (D-01) is required; Wave 0 gap: a new test exercising a type not resolved at construction |
| CONS-04 | `GenReflectionUI`'s periodic `mutable_config/` walk resolves every file's type, and reports rather than silently skips a file it cannot resolve | Pitfall 1 (the `filepath.WalkDir` return-value discard at `server.go:603`) and D-05's aggregate-and-log-on-change design in Architecture Patterns / Code Examples |
| SAFE-02 | Concurrent requests against a long-lived mutation server or agent are race-free under `go test -race` | D-07 e2e harness precedents (`test/e2e.go:16`, `agent/kv_agent_impl_test.go:214`) and dedicated tight-loop precedent (`utils/growable_resolver_race_test.go:32-89`) in Code Examples |
| SAFE-03 | A long-running process serving many different configs does not accumulate the full registry | D-08(a)/(b) fixture design in Code Examples, grounded in `scanCandidateLimit` (`utils/symbol_scan.go:41`) and `LoadedFileCount()` per-registry scoping (`utils/utils.go:505-512`) |

</phase_requirements>

## Summary

Phase 14 has one job: flip four already-identified construction sites to lazy
(`D-01`), rewire five already-identified resolver call sites onto the shared
`TypeResolver` (`D-03`), fix `GenReflectionUI` to report instead of silently
skip (`D-05`), keep gRPC reflection complete under laziness by retaining the
`Init()` discovery parse (`D-06`), and prove SAFE-02/SAFE-03 with two kinds of
tests per D-07/D-08. Every call site CONTEXT.md named was re-read this session
and confirmed unchanged at HEAD — **no drift since context-gathering.**

This research resolves the seven open questions CONTEXT.md left for the
planner, all by direct code reading, no guessing:

1. **The scan-vs-index tier boundary is `scanCandidateLimit = 32`** (`utils/symbol_scan.go:41`), and there is already a committed fixture pattern that forces it — `TestSymbolScanRespectsCandidateLimit` (`utils/symbol_scan_test.go:91-103`). D-08(a)'s escalation-guard fixture is this pattern, adapted so the query resolves to a real, unambiguous answer via the index once the scan bails.
2. **`LoadedFileCount()` is per-`*DescriptorRegistry` instance**, never global (`utils/utils.go:505-512`: `d.mu.RLock(); return len(d.lazyLoaded)` — a receiver method reading a receiver field). Criterion 5 must read `s.parser`'s registry, i.e. `ms.GetProtoRegistry().LoadedFileCount()` off the *serving* `ModuleService`, never the D-06 discovery registry, which is a second, separate `*DescriptorRegistry` instance the D-06 fix constructs.
3. **The dedicated tight-loop technique already exists twice** as a directly-adaptable pattern: `utils/growable_resolver_race_test.go:32-89` (`TestRegisterFileRacesRangeFiles`) and `utils/index_build_deadlock_test.go` (`TestIndexBuildRacesParseOne`) — both run unpaced reader goroutines in a `for { select { case <-done: return; default: ... } }` loop for the full duration of a concurrent writer `errgroup`, which is exactly D-07's second test shape.
4. **No pre-built N-concurrent-client bufconn harness exists for both RPCs at once**, but two single-purpose analogs do: `test.TestServer` (`test/e2e.go:16-47`, generic `RegServer`/`AddConnectionToClient`, used by `test/e2e_test.go`'s `TestMutationWithScripts`) for the mutation server, and the package-local `testServer` (`agent/kv_agent_impl_test.go:214-243`) for the agent. Neither drives real interceptors by default — `test/e2e_test.go`'s `TestAuthFlow` (lines 189-247) is the closer precedent for that, building its own inline bufconn+interceptor server rather than using `test.TestServer`.
5. **`agent/filekv` needs zero code change beyond the D-01 construction flip.** `filekv.go:111`'s `s.parser.ReadConfig` already calls `parser.go:201-207`'s `ReadConfig`, which already uses `p.TypeResolver` (not `LocalResolver`) — confirmed by direct read, not assumed. Only `filekv.go:70`'s `lib.NewModuleService` → `lib.NewLazyModuleService` needs to change.
6. **D-06's retained discovery registry needs no new struct field.** `discoveryFiles` (`server.go:359`) is already a local variable in scope through `server.go:433-444` where `reflection.NewServerV1`/`NewServer` are constructed, in the same function (`Init`). Swapping `DescriptorResolver: s.parser.FilesResolver` → `DescriptorResolver: discoveryFiles` at both call sites is sufficient — the reflection server's own reference keeps the `*protoregistry.Files` alive for the process lifetime with no extra plumbing.
7. **`GenReflectionUI`'s WalkDir return value is discarded even before reaching its four callers** — `server.go:603`, `filepath.WalkDir(...)` is called with no `err :=` capture at all, so a failure inside the closure that returns non-nil never even reaches `GenReflectionUI`'s own `return` statement (which is `return nil` at line 686 unconditionally, unless a later step like `standalone.WithExamples` fails). This is a stricter finding than CONTEXT.md's framing ("discarded at all four call sites") — the discard happens one level earlier, inside `GenReflectionUI` itself.

**Primary recommendation:** Do the mechanical D-01/D-03 rewiring first (five files, ten line-level edits, each independently regression-tested by an *existing* test — see Code Examples), then D-06's two-line reflection-resolver swap, then D-05's `GenReflectionUI` rewrite (capture WalkDir's error, aggregate with `errors.Join`, add a `lastFailures` comparison field), then the two D-07/D-08 test suites last, since they exercise the finished wiring.

**Correction to CONTEXT.md, verified this session:** the inserter CLI's current per-file failure behavior is **not** abort — `inserter/inserter.go:107-118` runs each file's `InsertConfigFile` in its own goroutine, logs the error on failure, and the command still returns `0`. Only `mutate` (`mutate/mutate.go:78-81`, `return 1`) currently aborts. The Claude's Discretion constraint ("do not silently change a CLI's exit-code behavior") therefore means: preserve inserter's per-file skip-and-log-continue, and preserve mutate's abort — do not unify them.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Type-URL / nested-`Any` resolution (D-03) | Backend / Library (`compiler/lib/parser.RegistryTypeResolver`) | — | In-process resolver chain; no network, no client tier |
| Inserter's KV write path | Backend / API (`inserter/inserter.go`) | Storage (Consul/etcd/ZooKeeper/ConfigMaps via valkeyrie) | Reads materialized JSON, resolves types, writes to a KV store |
| Agent filekv `Get`/`Watch` | Backend / API (`agent/filekv`) | Storage (local filesystem, dev-mode KV backend) | Serves subscribed gRPC clients from on-disk materialized configs |
| Mutation server `MutateConfig`/`GenReflectionUI` | Backend / API (`server/server.go`) | — | gRPC service; reflection UI is server-tier introspection, not client code |
| gRPC reflection completeness (D-06) | Backend / API (`server/server.go` `Init()`) | — | Server-startup wiring; the retained discovery parse never leaves this tier |
| Concurrency safety proofs (D-07/D-08) | Backend / Library + API test suites | — | All concurrency in scope is in-process goroutines against one shared server/registry; no cross-process concern |

## Standard Stack

No new external dependencies. Every package this phase touches is already a
direct dependency: `google.golang.org/grpc` (bufconn, reflection),
`golang.org/x/sync/errgroup` (test concurrency), `github.com/stretchr/testify`
(assertions), `google.golang.org/protobuf/encoding/protojson`. No `go get`
required.

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Reusing `discoveryFiles` local var directly (D-06) | A new `discoveryRegistry *utils.DescriptorRegistry` field on `ProtoconfMutationServer` | The field buys nothing here — `Init()` runs once and `discoveryFiles` already outlives it via the reflection server's own reference; a field is unrequested state that a future reader would have to convince themselves is never mutated (D-04's narrowness constraint is easier to prove for a captured local than a struct field). |
| `errors.Join` for D-05's aggregate | A typed `[]FailedConfig{Path, TypeURL, Err}` slice | `errors.Join` matches this codebase's established idiom (CLAUDE.md "Error Handling" convention, and 9+ existing call sites in `inserter.go`, `server.go`, `utils.go`, `agent.go`). A typed slice is more structured but nothing in this phase's criteria needs structured access — only aggregation and a change-detection comparison, both of which `errors.Join` plus a sorted-path-string comparison already provide. |

## Package Legitimacy Audit

No new packages. This phase adds zero dependencies — audit not applicable.

## Architecture Patterns

### System Architecture Diagram

```
                    ┌─────────────────────────────────────────┐
                    │   parser.RegistryTypeResolver (shared)    │
                    │  Tier0 snapshot → Tier1 growable Message  │
                    │  Registry → Tier2 scan (D-01, ≤32 cand.)  │
                    │  → Tier3 exact index (13-02) → hard error │
                    └───────────────┬────────────────────────────┘
                                    │ FindMessageByURL/ByName
        ┌───────────────┬──────────┼──────────────┬───────────────┐
        │               │          │              │               │
        ▼               ▼          ▼              ▼               ▼
  inserter.go:369   server.go:466  filekv.go:111  mutate.go:76  server.go:618
  (LocalResolver     (LocalResolver (already        (LocalResolver (already
   → TypeResolver)    → TypeResolver) TypeResolver)  → TypeResolver) TypeResolver)
        │               │                              │               │
        ▼               ▼                              ▼               ▼
  inserter.go:379   MutateConfig                  mutate CLI      GenReflectionUI
  (marshal resolver)  marshal                       Set command    walk (D-05: must
        │              resolver                                    aggregate, not
        ▼                                                           abort on 1st miss)
  KV store write
  (config.json,                                              ┌─────────────────┐
   config.data)                                               │ D-06: reflection │
                                                                │ DescriptorResolver│
                                                                │ = discoveryFiles  │
                                                                │ (Init()'s eager   │
                                                                │ src/ parse, kept  │
                                                                │ alive, never a    │
                                                                │ general resolver) │
                                                                └─────────────────┘
```

### Recommended Change Shape (no new files)

```
inserter/inserter.go       — 2 line-level swaps (D-01 construction, D-03 resolver x2)
server/server.go           — D-01 construction, D-03 marshal resolver, D-05 GenReflectionUI
                              rewrite, D-06 reflection wiring swap (2 lines)
agent/filekv/filekv.go     — D-01 construction only (verified: no D-03 change needed)
mutate/mutate.go           — D-01 construction, D-03 resolver swap
devserver/command.go       — no change required by any criterion (D-05 discretion: may add
                              error logging at the two discarded call sites, optional)
compiler/lib/module_service.go — rewrite NewLazyModuleService's stale doc comment (D-01)
```

### Pattern 1: The five-tier resolver rewiring (D-03)

**What:** Replace `i.parser.LocalResolver` / `parser.LocalResolver` /
`s.parser.LocalResolver` with `i.parser.TypeResolver` /
`resolver.TypeResolver` / `s.parser.TypeResolver` at exactly five call
sites.
**When to use:** Any resolution site that must see a type parsed after
construction — true for every consumer once D-01 makes construction produce
a near-empty snapshot.
**Example (verified at HEAD, `inserter/inserter.go:369,379`):**
```go
// Source: inserter/inserter.go:369-379 (current, pre-Phase-14)
mt, err := i.parser.LocalResolver.FindMessageByURL(protoconfValue.Value.TypeUrl)
...
data, err = protojson.MarshalOptions{Multiline: true, Resolver: i.parser.LocalResolver}.Marshal(new)
```
Both `LocalResolver` occurrences become `TypeResolver`. `TypeResolver` is
`*parser.RegistryTypeResolver` (`compiler/lib/parser/parser.go:60-116`),
which implements the same `protojson.MarshalOptions.Resolver` /
`protoregistry.MessageTypeResolver` interfaces `LocalResolver`
(`*protoregistry.Types`) does — the swap is source-compatible, no
call-site signature change.

### Pattern 2: D-06's reflection resolver swap needs no new state

**What:** `server.go:354-360`'s `Init()`-local `discoveryRegistry`/
`discoveryFiles` already exist; they simply need to be the value passed to
`reflection.NewServerV1`/`NewServer` instead of `s.parser.FilesResolver`.
**Example (verified at HEAD, `server/server.go:354-444`):**
```go
// server.go:354-359 (unchanged — already runs today, output currently discarded
// for reflection purposes)
discoveryRegistry := utils.NewDescriptorRegistry()
srcPath := filepath.Join(s.protoconfRoot, consts.SrcPath)
if err := discoveryRegistry.Import(discoveryRegistry.Parse, []*regexp.Regexp{}, srcPath); err != nil {
    logger.Error("failed to parse proto files for service discovery", "path", srcPath, "error", err)
}
discoveryFiles := discoveryRegistry.GetFilesResolver()
// ... service registration loop using discoveryFiles.RangeFiles (unchanged) ...

// server.go:433-444 (D-06 change: s.parser.FilesResolver -> discoveryFiles,
// both reflection.NewServerV1 and reflection.NewServer; ExtensionResolver
// stays s.parser.LocalResolver per D-04/the untouched-clause note below)
reflectionServer := reflection.NewServerV1(reflection.ServerOptions{
    Services:           rpcServer,
    DescriptorResolver: discoveryFiles,       // was: s.parser.FilesResolver
    ExtensionResolver:  s.parser.LocalResolver,
})
grpc_reflection_v1.RegisterServerReflectionServer(rpcServer, reflectionServer)
grpc_reflection_v1alpha.RegisterServerReflectionServer(rpcServer, reflection.NewServer(reflection.ServerOptions{
    Services:           rpcServer,
    DescriptorResolver: discoveryFiles,       // was: s.parser.FilesResolver
    ExtensionResolver:  s.parser.LocalResolver,
}))
```
`reflection.ServerOptions.DescriptorResolver` is typed `protodesc.Resolver`
(`go doc google.golang.org/grpc/reflection ServerOptions`, verified this
session) — `*protoregistry.Files` satisfies it, same as `s.parser.FilesResolver`
did, so this is a drop-in type-compatible swap.

**Do not touch `ExtensionResolver`.** `RegistryTypeResolver.FindExtensionByName`
(`parser.go:118-121`) delegates straight to the same construction-time
snapshot `LocalResolver` already is — D-03/D-04 explicitly exclude this from
the "single resolution path" cleanup because there is nothing growable to
wire in.

### Anti-Patterns to Avoid

- **Adding a struct field for the discovery registry "just in case."** The
  local-variable capture already satisfies D-06's requirement and D-04's
  narrowness constraint more cheaply — a field is reachable from every method
  on `ProtoconfMutationServer` and invites a future reader to start resolving
  through it. Only add a field if a later criterion needs the discovery
  registry after `Init()` returns for something reflection-unrelated (none
  does today).
- **Rewiring `server.go:601-660`'s `WalkDir` closure error handling without
  first capturing `filepath.WalkDir`'s own return value.** The closure's
  `return err` paths already abort the walk early; fixing that alone (making
  every branch `return nil` and aggregating separately) is necessary, but the
  outer `filepath.WalkDir(...)` call at line 603 also needs `err := ` — today
  its return value isn't even assigned to `_`.
- **Treating `agent/filekv` as needing a D-03-style resolver swap.** It
  doesn't — `filekv.go:111` is already on `TypeResolver` via `ReadConfig`
  (TYPE-09, closed in Phase 13). Adding an unnecessary second change here
  is scope creep against a file the phase should touch exactly once (the
  `New()` constructor call).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Unpaced concurrent reader/writer race forcing | A custom `sync.WaitGroup` + manual timing loop | The `for { select { case <-done: return; default: <call> } }` pattern already in `utils/growable_resolver_race_test.go:44-68` | Proven (Phase 12 SAFE-01) to reliably force the interleaving `TestConcurrentCompile`-style e2e tests miss; reinventing it risks a flaky or non-forcing test |
| In-process bufconn gRPC client/server wiring | A new dial/listen helper per test file | `test.TestServer` (`test/e2e.go:16-47`) for generic services, or the agent-local `testServer` (`agent/kv_agent_impl_test.go:214-243`) pattern for `ProtoconfServiceServer` | Both already exist, are exercised in CI, and match this repo's established `passthrough:///bufnet` + `insecure.NewCredentials()` convention (STATE.md Phase 01 decision) |
| A synthetic corpus generator for D-08(b)'s sequence guard | A new corpus builder | `utils/testdata.GenerateCorpus(dir, n)` (`utils/testdata/corpus.go:26`) — already produces N distinct, uniquely-named messages per file, exactly what "serve N distinct configs" needs | Existing generator is deterministic (fixed-seed `rand`), already used by `TestConcurrentCompile` and the scaling-gate benchmark; distinct per-file symbol names (`Msg0`, `Msg1`, ...) make it directly usable for the sequence guard without modification |

**Key insight:** Every mechanism D-07/D-08 need — the tight-loop race
technique, the bufconn harness, the synthetic corpus, the `LoadedFileCount()`
observable — already exists in this codebase from Phases 11-13. This phase's
test work is composition, not invention.

## Runtime State Inventory

Not applicable — this phase is a correctness/rewiring pass over Go code
paths, not a rename, refactor, or migration of stored/external state. No
datastore keys, external service configs, OS registrations, secrets, or
build artifacts carry the strings this phase touches (`LocalResolver`,
`TypeResolver`, `NewModuleService`, `NewLazyModuleService` are Go identifiers
internal to this repository's binary, not externally-persisted names).
**None found in every category — verified by reading all five call sites
this phase touches; none reference a KV key, env var name, or OS
registration.**

## Common Pitfalls

### Pitfall 1: Believing `GenReflectionUI`'s outer `return err` sites are the whole fix

**What goes wrong:** Fixing only `server.go:605/621/628` (the closure's early
`return err` paths) without also capturing `filepath.WalkDir`'s own return
value at line 603 leaves the aggregated error unreachable — `WalkDir`'s
return is currently thrown away entirely, not even into `_`.
**Why it happens:** The closure's `return err` reads like the failure path;
it's easy to miss that the enclosing `filepath.WalkDir(...)` statement never
captures what the closure caused `WalkDir` to return.
**How to avoid:** Change the closure to always `return nil` (never abort the
walk) and accumulate failures into a local slice/aggregate instead; separately
capture `filepath.WalkDir(...)`'s own return (still relevant for a directory
I/O error, distinct from a per-file resolution failure) and fold it in.
**Warning signs:** A test that plants a second, resolvable config after a
broken one and asserts the second one still appears in `examples` — if this
test passes before the fix, the closure abort was already masked by
something else and the real bug is elsewhere.

### Pitfall 2: Assuming the inserter's current per-file behavior is "abort"

**What goes wrong:** CONTEXT.md's discretion note frames inserter and mutate
as sharing "abort" as the current behavior. Verified at HEAD: the inserter
CLI (`inserter/inserter.go:107-118`) already skips-and-logs per file inside
a goroutine and always returns `0`; only `mutate` (`mutate/mutate.go:78-81`)
aborts with `return 1`. A plan that "preserves current abort behavior" for
both would silently change the inserter's exit code — a regression the D-02
loud-failure decision does not require and Claude's Discretion explicitly
forbids ("do not silently change a CLI's exit-code behavior as a side effect
of the resolver rewiring").
**Why it happens:** Both consumers loop over multiple configs, and it's easy
to conflate "logs a loud error" (true for both, and what D-02 requires) with
"aborts the run" (true only for mutate).
**How to avoid:** Preserve each consumer's current control-flow shape exactly;
only the resolver call sites and error content change, not the loop structure.
**Warning signs:** A test asserting the inserter CLI's exit code changes from
0 to 1 on a single bad file among several good ones.

### Pitfall 3: Building a new synthetic corpus for D-08(a) instead of reusing the committed pattern

**What goes wrong:** `utils/testdata.GenerateCorpus` gives every message a
unique per-file name (`Msg0`, `Msg1`, ...), so it can never trigger the
scan tier's `scanCandidateLimit` bailout — no two files ever share a
declaration the scan's regex would match. A planner reaching for it to prove
D-08(a) will get a scan-tier *hit*, not the miss the criterion needs.
**Why it happens:** `GenerateCorpus` is the corpus fixture named in
`13-VERIFICATION.md`/`TESTING.md`, and it's natural to assume it covers every
tier boundary.
**How to avoid:** Use the `TestSymbolScanRespectsCandidateLimit` shape
(`utils/symbol_scan_test.go:91-103`) instead — write `scanCandidateLimit+1`
(33) `.proto` files that each contain a textually-matching `message`/`enum`
declaration for the same short name, but keep the true target's declaration
under a distinct, disambiguating fully-qualified package name so the index
tier resolves it exactly once the scan bails. See Code Examples below for the
concrete shape.
**Warning signs:** `d.ScanResolutionCount()` is nonzero after the "escalation"
lookup — that means the scan tier answered it, not the index tier, and the
fixture doesn't test what D-08(a) needs.

## Code Examples

### D-08(a) escalation-guard fixture (built from the `TestSymbolScanRespectsCandidateLimit` precedent)

```go
// Source: utils/symbol_scan_test.go:91-103 (existing, verified at HEAD) is the
// scan-tier-bails half. The escalation guard extends it: after the scan
// bails, resolve through the shared TypeResolver chain (parser.go's
// resolveTiers) and assert (a) it succeeds via the index tier, not the scan
// tier, and (b) LoadedFileCount() grew by exactly the resolved file's own
// transitive closure — not the whole 33-file corpus.
root := t.TempDir()
// 32 decoy files anywhere under root, each textually declaring "message Thing {"
// under a DIFFERENT package than the real target, so declPattern (which matches
// on message/enum name only, not package) sees 33 total candidates and
// symbolScanCandidates bails (utils/symbol_scan.go:126-129: len > scanCandidateLimit -> nil).
for i := 0; i < scanCandidateLimit; i++ {
    writeSymbolScanProto(t, root, fmt.Sprintf("decoy%d/f.proto", i), fmt.Sprintf("decoy%d", i), "Thing")
}
// The real target: package "real.v1", message "Thing" -- fully-qualified
// "real.v1.Thing" is unique across the whole tree (no other file uses this
// exact package+name pair), so the index tier answers it unambiguously
// (utils/symbol_index.go's exact fully-qualified-name keying).
writeSymbolScanProto(t, root, "target/f.proto", "real.v1", "Thing")

d := NewDescriptorRegistry()
d.ImportPaths = []string{root}
d.CacheDir = t.TempDir()

before := d.LoadedFileCount()
require.False(t, d.LoadSymbolByScan("real.v1.Thing"), "scan tier must bail at 33 candidates")
require.Equal(t, 0, d.ScanResolutionCount())
require.True(t, d.LoadSymbolByIndex("real.v1.Thing"), "index tier must resolve it")
// D-08(a): grows only by this lookup's own transitive closure (here, 1 file
// with no imports), never jumps toward the 33-file corpus.
require.Equal(t, before+1, d.LoadedFileCount())
```

### D-07 dedicated tight-loop precedent to adapt

```go
// Source: utils/growable_resolver_race_test.go:32-89 (existing, verified at
// HEAD) — TestRegisterFileRacesRangeFiles. Adapt this shape for the mutation
// server / agent: replace dr.RangeFiles/dr.FindFileByPath readers with
// concurrent calls into s.parser.TypeResolver.FindMessageByURL (or the
// agent's Get/SubscribeForConfig path) run continuously in a tight loop
// alongside N concurrent MutateConfig/ParseOne-triggering writers.
done := make(chan struct{})
var wg sync.WaitGroup
wg.Add(1)
go func() {
    defer wg.Done()
    for {
        select {
        case <-done:
            return
        default:
            dr.RangeFiles(func(protoreflect.FileDescriptor) bool { return true })
        }
    }
}()
// ... second reader goroutine, then N-way errgroup of writers, then close(done); wg.Wait() ...
```
**Required sanity-check step (not itself a committed test, a plannable task):**
temporarily comment out the lock this new test depends on, run
`go test -race -run <TestName>`, confirm the output contains
`WARNING: DATA RACE`, then restore the lock. This is how Phase 12 validated
`TestRegisterFileRacesRangeFiles`'s own failure-detection capability
(STATE.md Phase 12 entry) and is the same validation D-07 requires here.

### D-07 e2e harness precedent to adapt

```go
// Source: test/e2e_test.go:189-247 (existing, verified at HEAD) — TestAuthFlow.
// This is the closer precedent for "real interceptors" than test.TestServer,
// since test.TestServer's grpc.NewServer() (test/e2e.go:19) takes no
// interceptor options. Build the mutation-server e2e test on this inline
// bufconn+interceptor shape; N concurrent goroutines call client.MutateConfig
// (or client.SubscribeForConfig, agent side) instead of TestAuthFlow's
// three sequential t.Run subtests.
lis := bufconn.Listen(1024 * 1024)
rpcServer := grpc.NewServer(grpc.ChainUnaryInterceptor(makeTokenInterceptor(secretToken)))
protoconf_pb.RegisterProtoconfMutationServiceServer(rpcServer, srv)
go func() { rpcServer.Serve(lis) }()
conn, _ := grpc.NewClient("passthrough:///bufnet",
    grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
    grpc.WithTransportCredentials(insecure.NewCredentials()))
client := protoconf_pb.NewProtoconfMutationServiceClient(conn)
```

### Existing regression tests that already prove CONS-02 once D-01/D-03 land

```go
// Source: inserter/inserter_test.go:80-95 (existing, verified at HEAD) —
// TestProtoconfInserter_InsertConfig_AnyResolution. Exercises exactly the
// nested-Any-in-config.json path D-03's inserter.go:379 swap protects
// (the 260902-eie backport). Under today's EAGER construction this test
// already passes trivially (everything is resolved at construction); once
// D-01 flips the inserter lazy, this same test becomes the natural CONS-02
// regression proof, no new fixture required — the type MUST resolve
// on-demand for it to keep passing.
err := i.InsertConfigFile("field_type_any_test.materialized_JSON")
require.NoError(t, err)
got := string(v.Value)
assert.Contains(t, got, "type.googleapis.com/test.v1.TestMessage")
assert.Contains(t, got, "test_any")
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| Four consumers construct via eager `lib.NewModuleService` | All four construct via `lib.NewLazyModuleService`, `mod sync` stays eager | This phase (D-01) | Each consumer stops paying its own whole-`src/`-tree parse+link at startup |
| `LocalResolver` used directly at 5 call sites for type-URL/`Any` resolution | `TypeResolver` (the Phase 13 shared tiered chain) used everywhere except `ExtensionResolver` | This phase (D-03) | Closes ROADMAP criterion 1's grep clause: exactly one type-URL resolution implementation left in the codebase |
| `GenReflectionUI` aborts the whole walk on the first unresolvable config, error discarded at every caller | Walk completes, failures aggregated, logged only when the failure set changes | This phase (D-05) | An operator sees every broken mutable config, not just "the process before the first broken one" |
| Reflection's `DescriptorResolver` is `s.parser.FilesResolver` (near-empty once lazy) | `Init()`'s existing discovery parse (`discoveryFiles`) is retained and wired in instead | This phase (D-06) | gRPC reflection stays byte-identical to today's eager behavior; costs nothing extra since the parse already runs |

**Deprecated/outdated:** `NewLazyModuleService`'s doc comment
(`compiler/lib/module_service.go:48-53`) claiming "every other consumer ...
must keep using `NewModuleService`" — false after this phase; only `mod sync`
still does. Must be rewritten as part of this phase's D-01 work.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `errors.Join` (over a typed aggregate struct) is the right shape for D-05's aggregated error, based on codebase convention rather than a hard requirement | Alternatives Considered, Don't Hand-Roll | Low — this is Claude's Discretion per CONTEXT.md, not a locked decision; if the planner wants per-file structured access (path + type URL + diagnostic) for the failure-set comparison, a typed slice is a reasonable alternative that still satisfies every criterion. The comparison mechanism (sorted path list vs. hash) is independently flagged as discretion in CONTEXT.md already. |

**All other claims in this research were verified by direct file reads at
HEAD this session (file:line citations throughout) — no other assumed
claims.**

## Open Questions

None outstanding for planning. All seven questions the phase brief posed
(tier boundary, `LoadedFileCount()` scoping, race technique, bufconn harness,
`agent/filekv`'s dependency, reflection wiring type, and the four
`GenReflectionUI` call sites) were answered by direct code reads this
session, cited above.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | Everything | ✓ | go1.25.8 darwin/arm64 | — |
| `go test -race` | SAFE-02/SAFE-03 proofs, CI gate | ✓ | Go 1.25.8 built-in | — |
| `protoc` | Not needed this phase (no new `.proto` changes to generated code) | ✓ | present at `/opt/homebrew/bin/protoc` | — |
| CI `go test -race ./...` | Ongoing regression gate | ✓ | `.github/workflows/go.yml:37` | — |

No missing dependencies; nothing blocks this phase.

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go standard `testing` + `github.com/stretchr/testify` (assert/require) |
| Config file | none — `go test` via module, no framework config file |
| Quick run command | `go test ./inserter/... ./agent/filekv/... ./server/... ./mutate/... ./utils/... -run <TestName> -v` |
| Full suite command | `go test -race -count=1 ./...` (matches CI's `.github/workflows/go.yml:37`) |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CONS-02 | Inserter resolves an on-demand type correctly | unit | `go test ./inserter/... -run TestProtoconfInserter_InsertConfig_AnyResolution -v` | ✅ existing (`inserter/inserter_test.go:80`), becomes a real lazy-path proof once D-01 lands |
| CONS-03 | filekv resolves an on-demand type correctly | unit | new test needed | ❌ Wave 0 — existing `TestGet_ValidKey` (`agent/filekv/filekv_test.go:100`) doesn't exercise a type requiring scan/index tier |
| CONS-04 | `GenReflectionUI` reports rather than skips | unit | new test needed, extends `TestProtoconfMutationServer_GenReflectionUI` (`server/server_test.go:144`) | ❌ Wave 0 — needs a fixture with two mutable configs, one broken, asserting the good one still appears and the returned error names the broken one |
| SAFE-02 | Concurrent mutation-server/agent requests race-free | integration (e2e) + dedicated tight-loop | `go test -race -run <e2e> ./server/... ./agent/...` and `go test -race -run <tightloop> ./server/... ./agent/...` | ❌ Wave 0 — both need new test files per the D-07 patterns above |
| SAFE-03 | `LoadedFileCount()` stays proportional | unit, both clauses | `go test -race -run <escalation-guard\|sequence-guard>` | ❌ Wave 0 — both need new tests per D-08(a)/(b) patterns above |

### Sampling Rate

- **Per task commit:** targeted `go test -run <TestName> -race ./<package>/...`
- **Per wave merge:** `go test -race -count=1 ./...`
- **Phase gate:** full suite green (matching CI) before `/gsd-verify-work`

### Wave 0 Gaps

- [ ] `agent/filekv/filekv_test.go` — a test resolving a type only reachable via the scan or index tier (not the global well-known seed, not construction-time), covering CONS-03
- [ ] `server/server_test.go` (or a new `server/gen_reflection_ui_test.go`) — a two-config fixture (one resolvable, one not) proving CONS-04's aggregate-and-continue behavior
- [ ] `server/mutate_config_race_test.go` (new) and/or `agent/*_race_test.go` (new) — SAFE-02's e2e and dedicated tight-loop pair, per consumer
- [ ] `utils/symbol_scan_test.go` or a new `utils/loaded_file_count_test.go` — SAFE-03's escalation-guard (D-08a) and sequence-guard (D-08b) tests
- Framework install: none — testify and errgroup are already direct/transitive dependencies (`go.mod`)

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-------------------|
| V2 Authentication | no (unchanged this phase) | Bearer token interceptor already exists (`server.go` `bearerTokenInterceptor`), untouched |
| V3 Session Management | no | N/A — stateless gRPC requests |
| V4 Access Control | no (unchanged this phase) | Same auth path as today |
| V5 Input Validation | yes, indirectly | The D-02 hard-error path (Phase 13) already validates that an unresolvable type URL fails loudly with a diagnostic rather than silently constructing a wrong/empty message — this phase extends that same contract to four more call sites, not a new validation surface |
| V6 Cryptography | no | Unaffected — no crypto/TLS code touched this phase |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|-----------------------|
| Data race on shared `*utils.DescriptorRegistry` state under concurrent gRPC requests | Tampering (undefined-behavior corruption) | `d.mu` lock discipline already established (Phase 11/12); D-07's job is to *prove* it holds for these four consumers' new lazy call paths, not invent new locking |
| Silent resolution failure masking a real proto-schema mismatch (a type genuinely absent, not just "not yet loaded") | Denial of Service / Information Disclosure (an operator ships a broken config unaware) | D-02's loud, named-symbol hard error (already shipped, Phase 13) plus this phase's D-05 aggregate-and-report extend the same "loud, never silent" mitigation to every consumer |

## Sources

### Primary (HIGH confidence — direct file reads at HEAD this session)
- `compiler/lib/parser/parser.go:24-207` — `Parser`, `RegistryTypeResolver`, `ReadConfig`, `ParseFilesX`
- `utils/utils.go:36-540` — `DescriptorRegistry`, `LoadedFileCount`, lock discipline comments
- `utils/symbol_scan.go:1-197`, `utils/symbol_scan_test.go:1-160` — scan tier and its candidate-limit test
- `utils/symbol_index.go:1-350`, `utils/symbol_index_test.go:1-180` — index tier, duplicate-symbol tie-break, link-zero-files proof
- `server/server.go:160-687` — `Run`, `NewProtoconfMutationServer`, `Init`, reflection wiring, `MutateConfig`, `GenReflectionUI`
- `devserver/command.go:1-120` — the four `GenReflectionUI` call sites' devserver half
- `inserter/inserter.go:1-420`, `inserter/inserter_test.go:1-100` — construction, resolver call sites, existing regression coverage
- `agent/filekv/filekv.go:1-130` — construction and `Get`/`ReadConfig`
- `mutate/mutate.go:1-100` — construction and resolver call site
- `compiler/lib/module_service.go:36-464` — `NewLazyModuleService`, `GetProtoRegistry`'s lazy branch
- `test/e2e.go:1-47`, `test/e2e_test.go:1-250`, `agent/kv_agent_impl_test.go:1-245` — bufconn harness precedents
- `utils/growable_resolver_race_test.go:1-89` — dedicated tight-loop race precedent
- `utils/testdata/corpus.go:1-169` — synthetic corpus generator, why it doesn't fit D-08(a)
- `go doc google.golang.org/grpc/reflection ServerOptions` — `DescriptorResolver` field type confirmation
- `.planning/phases/13-.../13-CONTEXT.md`, `.planning/phases/12-.../12-VERIFICATION.md`, `.planning/phases/11-.../11-RESEARCH.md` — predecessor decisions and the SAFE-01 evidence behind D-07

### Secondary (MEDIUM confidence)
None — no web/docs lookups were needed this session.

### Tertiary (LOW confidence)
None.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependencies, every pattern already in-repo
- Architecture: HIGH — every call site read at HEAD, all seven open questions answered by direct code reading
- Pitfalls: HIGH — each pitfall grounded in a specific line read this session, not inferred

**Research date:** 2026-09-08
**Valid until:** Next commit that touches `server/server.go`, `inserter/inserter.go`, `agent/filekv/filekv.go`, `mutate/mutate.go`, `utils/symbol_scan.go`, or `utils/symbol_index.go` — this research pins exact line numbers and will drift on any of those files changing. Given this is the next phase to execute with no other work queued against these files, expected valid through Phase 14's execution.
