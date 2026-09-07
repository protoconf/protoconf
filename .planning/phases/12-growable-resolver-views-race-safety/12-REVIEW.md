---
phase: 12-growable-resolver-views-race-safety
reviewed: 2026-09-08T00:00:00Z
depth: standard
files_reviewed: 9
files_reviewed_list:
  - compiler/lib/compiler.go
  - compiler/lib/concurrent_compile_test.go
  - compiler/lib/config.go
  - compiler/lib/parser/canonical_identity_test.go
  - compiler/lib/parser/growable_resolver_test.go
  - compiler/lib/parser/parser.go
  - utils/growable_resolver_race_test.go
  - utils/growable_resolver_test.go
  - utils/utils.go
findings:
  critical: 1
  warning: 3
  info: 2
  total: 6
status: issues_found
---

# Phase 12: Code Review Report

**Reviewed:** 2026-09-08T00:00:00Z
**Depth:** standard
**Files Reviewed:** 9
**Status:** issues_found

## Summary

The core lock-discipline work is solid: `GetFilesResolver`, `registerFileLocked`, `FindFileByPath`,
and `RangeFiles` in `utils/utils.go` correctly gate every access to the growable
`*protoregistry.Files` behind `d.mu`, `recordFileLocked`/`ParseAll` call `registerFileLocked` from
the same locked insert point that writes `FileRegistry` (so D-02 non-divergence genuinely holds for
the lazy/compiler path), and the new tests (`TestRegisterFileRacesRangeFiles`,
`TestConcurrentCompile`'s extended assertions) do exercise real concurrent interleavings under
`-race`, not just single-threaded happy paths.

However, the fix to `ParseFilesX`'s "resolver hit" branch (`compiler/lib/parser/parser.go`) — the
one piece of the phase explicitly flagged as risky in the phase context — is broken: routing the
resolver read through `p.registry.FindFileByPath` instead of the raw `p.FilesResolver` field makes
the canonical-lookup-then-`desc.WrapFile`-fallback block unreachable for every caller, in both the
eager and lazy registry shapes, for opposite reasons. That silently regresses exactly the
external-file scenario (the mutation server's six hand-registered well-known files) the phase's own
docs said had to keep working, and no test in the reviewed set actually exercises it — the phase's
own planning docs cite a test (`TestDiscoveryScanDoesNotBackReflection`) as verification, but that
test never calls `ParseFilesX`. Additionally, the growable resolver's raw pointer is still exposed
through a public field (`Parser.FilesResolver`) and read directly, unlocked, by both production code
(`server/server.go`) and the phase's own test suite, which is exactly the anti-pattern the phase's
locking work was supposed to eliminate (currently latent/safe only because every concrete caller of
that field happens to sit on an eager, never-growing registry — nothing in the type system enforces
that).

## Critical Issues

### CR-01: `ParseFilesX`'s canonical-lookup/`desc.WrapFile` fallback branch is unreachable dead code, regressing resolution of files present only in the resolver

**File:** `compiler/lib/parser/parser.go:125, 142-171`
**Issue:**

The phase changed `ParseFilesX`'s resolver read from the raw field to the locked accessor:

```go
resolvedFd, resolverErr := p.registry.FindFileByPath(filename)   // was: p.FilesResolver.FindFileByPath(filename)
if resolverErr != nil {
    parsed, parseErr := p.registry.ParseOne(filename)
    ...
}
if canonical, ok := p.registry.FileDescriptor(resolvedFd.Path()); ok {
    results = append(results, canonical)
    continue
}
// "The file is in the resolver but genuinely absent from FileRegistry —
// the mutation server's six hand-registered well-known files
// (server/server.go). Fall back to wrapping the resolver's own
// descriptor, unchanged from before this phase."
fd := resolvedFd
d, err := desc.WrapFile(fd)
...
```

`p.registry.FindFileByPath` (`utils/utils.go:468-475`) returns `ErrNoGrowableResolver` whenever
`d.filesResolver == nil` — which is *every eager registry, forever* (`d.filesResolver` is only ever
assigned inside `GetFilesResolver`'s lazy branch, `utils/utils.go:182-193`; the eager branch always
returns a fresh, un-cached build and never touches the field). So for any `Parser` built over an
eager registry, `resolverErr != nil` on every call, and execution always falls into the `ParseOne`
branch — which itself always fails with `ErrLazyParseDisabled` for an eager registry
(`ImportPaths` empty). The comment's named scenario — "the mutation server's six hand-registered
well-known files" — is registered directly onto `parser.FilesResolver`
(`server/server.go:292-297`), a raw snapshot object that is *not* `d.filesResolver` for that (eager)
registry. Before this phase, `ParseFilesX` read `p.FilesResolver` directly and would have found
those six files; after this phase, the same lookup is rerouted through `p.registry.FindFileByPath`,
which can never see them, and the request now fails outright with
`errors.Join(ErrNoGrowableResolver, ErrLazyParseDisabled)`.

Symmetrically, for a *lazy* registry, `recordFileLocked` (`utils/utils.go:413-424`) and `ParseAll`'s
diff loop (`utils/utils.go:446-457`) are the only two places that ever grow `filesResolver`, and both
always pair every `registerFileLocked` call with a `FileRegistry` write at the same locked
insertion point. So whenever `FindFileByPath` succeeds for a lazy registry, the following
`p.registry.FileDescriptor(resolvedFd.Path())` canonical lookup is *guaranteed* to also succeed —
the `desc.WrapFile` fallback below it can never run either.

Net effect: lines 146-171 (the `desc.WrapFile`/`desc.CreateFileDescriptor` fallback, and the
associated comment claiming this is "unchanged from before this phase") are dead code under both
registry shapes, and the specific external-file case the phase context called out to protect
("Verify the canonical lookup was inserted ahead of it additively and did not change behavior for
that external-file case") is not additive — it silently breaks that case for any `ParseFilesX`
caller backed by an eager registry.

This is not caught by any test in the reviewed set. `compiler/lib/parser/growable_resolver_test.go`
and `canonical_identity_test.go` only exercise the lazy path. `compiler/lib/parser/parser_test.go`'s
`TestParser_ParseFilesX` only ever hits the direct-`FileRegistry`-hit branch or the
`ErrNoGrowableResolver`+`ErrLazyParseDisabled` not-found branch (both files it uses are either
already in `FileRegistry` or genuinely absent from everything) — it never constructs the
"present in resolver, absent from `FileRegistry`" state this fallback exists for. The phase's own
plan/summary docs (`12-02-SUMMARY.md:103`, `12-02-PLAN.md:354`) cite
`go test -race ./server/... (TestDiscoveryScanDoesNotBackReflection)` as verification that "the
mutation server's six hand-registered well-known files... still resolve" — but that test
(`server/init_prohibitions_test.go:123-144`) never calls `ParseFilesX`; it only calls
`s.parser.FilesResolver.FindFileByPath` directly (the raw field, bypassing the exact code path that
changed). The claimed verification does not cover the regression.

In production this is currently latent rather than actively triggered, because the only caller of
`ParseFilesX` (`compiler/lib/starlark_loader.go:211`) always runs against the compiler's lazy
registry (`NewLazyModuleService`), and `server.go`'s eager parser never calls `ParseFilesX`. But it
is a live defect in the general `Parser.ParseFilesX` contract that any future caller — or any change
that starts routing compiler-adjacent lookups through an eager registry, or a starlark load of one
of the mutation server's six well-known files via a shared parser — will hit.

**Fix:** Fall back to the raw field when the locked accessor reports no growable resolver, instead
of treating `ErrNoGrowableResolver` as "not found":

```go
resolvedFd, resolverErr := p.registry.FindFileByPath(filename)
if errors.Is(resolverErr, utils.ErrNoGrowableResolver) {
    // Eager registry: p.FilesResolver is the fixed, non-growing snapshot
    // (possibly hand-extended, e.g. server.go's six well-known files) —
    // read it directly, since d.filesResolver is never armed for it.
    resolvedFd, resolverErr = p.FilesResolver.FindFileByPath(filename)
}
if resolverErr != nil {
    parsed, parseErr := p.registry.ParseOne(filename)
    ...
}
```
Add a test that builds an eager registry, registers a file directly onto `p.FilesResolver` (mirroring
`server.go`'s pattern) without adding it to `FileRegistry`, and asserts `ParseFilesX` still resolves
it via the `desc.WrapFile` fallback — the state this whole branch exists for is currently untested.

## Warnings

### WR-01: `Parser.FilesResolver` is a public field holding the live, unsynchronized growable resolver — nothing stops an unlocked concurrent read/write through it

**File:** `compiler/lib/parser/parser.go:24-46`; exercised unlocked at `server/server.go:292-297,426,433` and `compiler/lib/parser/growable_resolver_test.go:31,41`
**Issue:** `GetFilesResolver()` documents that for a lazy registry it returns "the SAME object on
every later call" and that object "grows in place via `registerFileLocked`"
(`utils/utils.go:160-170`), and the whole point of `d.mu` is that a locally-constructed
`*protoregistry.Files` has no synchronization of its own. `NewParserWithDescriptorRegistry`
(`parser.go:36-46`) stores exactly that object, unguarded, into the exported `Parser.FilesResolver`
field. Every internal production read inside this package was correctly rerouted to the locked
`p.registry.FindFileByPath`/`RangeFiles` accessors this phase — but the field itself is still public,
and two call sites read/write it directly, unlocked: `server/server.go:292-297` (`RegisterFile` ×6,
plus two `DescriptorResolver: s.parser.FilesResolver` reflection wiring sites) and this phase's own
`growable_resolver_test.go:31,41` (`p.FilesResolver.FindFileByPath(...)` on a *lazy* registry's
parser). The race test's own doc comment (`utils/growable_resolver_race_test.go:28-31`) explicitly
calls this the exact anti-pattern to avoid ("reaching into the field would bypass `d.mu`"), yet the
parser package's own growable-resolver test does precisely that against a lazy registry, just
without a concurrent second goroutine to trip the race detector on it.
Today this is safe only because every concrete caller of `Parser.FilesResolver` happens to be backed
by an eager registry (`server.go`'s `lib.NewModuleService`, D-03) where `d.filesResolver` never
grows — that safety is a property of the current call graph, not of the API. Nothing in the type or
field visibility prevents a future caller from taking a lazy-registry-backed `Parser` (e.g. via
`compiler.parser`, if it were ever exposed, or a new consumer of
`NewParserWithDescriptorRegistry` over `NewLazyModuleService`) and reading/writing `.FilesResolver`
directly from a second goroutine while `ParseOne` is growing it — which per the phase's own analysis
is a Go fatal "concurrent map writes" crash, not a soft race.
**Fix:** Either unexport `FilesResolver` and force every reader through `Parser`-level locked
accessor methods, or wrap the field type so direct field access is impossible without going through
`registry.FindFileByPath`/`RangeFiles`. Failing that, at minimum: fix
`growable_resolver_test.go:31,41` to read through `dr.FindFileByPath` (as
`canonical_identity_test.go` and `TestConcurrentCompile` already correctly do) so the test suite
does not itself model the unlocked-access pattern it exists to guard against.

### WR-02: `FileRegistry` writes outside `recordFileLocked`/`ParseOne` still bypass `d.mu` entirely

**File:** `utils/utils.go:125-130` (`Merge`), `219-269` (`Import`, including the unlocked
read-modify-write in the `LookupImport` closure at 250-261), `271-283` (`Parse`), `551-558`
(`MergeFileDescriptorSet`)
**Issue:** The phase's whole design rests on "every access to the registry's own instance must be
under `d.mu`" for the growable resolver, and by extension for `FileRegistry` itself, since
`recordFileLocked`/`ParseOne` now take `d.mu.Lock()` specifically to write it safely. `Merge`,
`Import` (and its `LookupImport` closure, which does an unlocked read-then-write of
`d.FileRegistry[s]`), `Parse`, and `MergeFileDescriptorSet` all mutate the exact same map with no
locking at all. `ParseAll` is safe because it takes `d.mu.Lock()` for its whole body before calling
`Import`/`Parse` (documented at `utils/utils.go:443-445`), but every other call site of these four
methods (`module_service.go:361,456`, `server.go`'s discovery registry, `mod` command, `parser_test.go`)
relies entirely on the caller never invoking them concurrently with `ParseOne`/`recordFileLocked` on
the same registry instance. Today that holds by construction (verified: the eager-registry setup
path in `GetProtoRegistry()` calls `Import` only before the registry is published via
`m.cachedRegistry`, and `Sync()`'s `Import` runs on a private registry) — but it is an invariant
enforced by caller discipline across several packages, not by the registry itself, and it is easy to
violate silently since none of these four methods document the requirement.
**Fix:** Take `d.mu.Lock()` inside `Merge`, `Import`, `Parse`, and `MergeFileDescriptorSet`
themselves (or clearly document "caller must hold d.mu" the way `recordFileLocked` and
`fileDescriptorSetLocked` already do, and audit call sites), rather than relying on every current and
future caller independently knowing these four methods are only safe single-threaded.

### WR-03: `registerFileLocked`'s success/error counters can lag behind non-incremental registrations without any external signal beyond a log line

**File:** `utils/utils.go:396-406`
**Issue:** This is by design per the extensive doc comments (D-05/D-06: test-only counters, no CLI
surface, best-effort `slog.Error`), and the tests (`TestFilesResolverRegistrationErrorsStayZero`,
`TestRegistrationCountIsIncremental`) do correctly assert zero errors along both `ParseOne`/`ParseAll`
orderings. Flagging only because the design explicitly accepts "a duplicate-registration error...
only logged" as an acceptable production posture (no test, no metric, no alert wired to
`registrationErrors` outside this test suite) — a real `FileRegistry`/`filesResolver` divergence in
production would surface only as a single `slog.Error` line unless an operator is specifically
watching for it. Given this is an explicit, documented tradeoff rather than an oversight, treat this
as a forward-looking robustness note rather than something blocking this phase.
**Fix:** Consider a Prometheus counter or health-check surface for `registrationErrors` in a later
phase (already flagged in the phase's own research as "Open Question 1").

## Info

### IN-01: Duplicate import of the same package under two names

**File:** `compiler/lib/config.go:11-12`
**Issue:** `"google.golang.org/protobuf/proto"` is imported twice, once unaliased (`proto`) and once
aliased (`pbproto "google.golang.org/protobuf/proto"`). Both aliases are used in the file (`proto.Unmarshal`
at line 97, `pbproto.Message` at line 79), which is legal Go but confusing — a reader has to check
which alias resolves to which import to know they're the same package. Pre-existing, not introduced
by this phase's diff (only the `protoregistry` import and `protoResolver` field were removed here),
but present in a file under review.
**Fix:** Drop one alias and use a single name for the package throughout the file.

### IN-02: `err` shadowed inside `ParseFilesX`'s fallback block

**File:** `compiler/lib/parser/parser.go:151`
**Issue:** `d, err := desc.WrapFile(fd)` is a short variable declaration inside the `for` loop's body
block, which shadows the function's named return `err` rather than reusing it. Harmless today because
every error path in this function uses an explicit `return nil, err`/`return nil, errors.Join(...)`
(no bare `return` relies on the named value), but it is a `go vet -shadow` finding and a maintenance
hazard if a future edit adds a bare `return` expecting the named `err` to have been set. Pre-existing,
not introduced by this phase.
**Fix:** Rename the inner variable (e.g. `wrapErr`) or restructure to avoid shadowing the named return.

---

_Reviewed: 2026-09-08T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
