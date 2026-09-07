# Phase 12: Growable Resolver Views & Race Safety - Pattern Map

**Mapped:** 2026-09-07
**Files analyzed:** 4 edited files + 2 extended test files (no brand-new files required)
**Analogs found:** 6 / 6 (all self-file: every touched file already carries the pattern to extend, inherited
from Phase 11)

All line numbers below were re-read against the live tree this session (`utils/utils.go`,
`compiler/lib/parser/parser.go`, `compiler/lib/config.go`, `compiler/lib/compiler.go`,
`compiler/lib/concurrent_compile_test.go`, `compiler/lib/parser/lazy_parse_test.go`,
`compiler/lib/parser/parser_test.go`) — they have drifted a little from RESEARCH.md's own citations
(RESEARCH.md cites `recordFileLocked` at `utils/utils.go:333-343`; current tree: same, confirmed) and from
11-PATTERNS.md's citations (Phase 11 already landed: `ParseOne`, `recordFileLocked`, `LoadedFileCount`,
`RegistryTypeResolver`, `TestConcurrentCompile`, `TestParseMemoization` all exist now — this phase extends
them, it does not create them).

**Where this diverges from 11-PATTERNS.md:** Phase 11's map was for code that did not yet exist (`ParseOne`,
`recordFileLocked`, etc. were new in Phase 11). This phase's targets are those *same functions*, now shipped
— so every analog below is "itself, current shipped form" rather than a sibling file. There is no new
package, no new file role, and (per D-04) no new constructor — the closest analog for every edit is the
function being extended.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|-----------------|---------------|
| `utils/utils.go` (+`filesResolver *protoregistry.Files` field, +`FindFileByPath`/`RangeFiles` locked accessors, +registration counter, growth call in `recordFileLocked` and `ParseAll`) | model/service (shared cache) | CRUD (keyed cache read/write) | itself — `LoadedFileCount()` (`utils/utils.go:378-382`) for the locked-accessor shape; `recordFileLocked` (`utils/utils.go:333-343`) and `ParseAll` (`utils/utils.go:352-373`) for the two growth insert points | exact (self-file edit) |
| `compiler/lib/parser/parser.go` (`ParseFilesX` branch 2 rewritten to route through registry's locked accessor + canonical `FileDescriptor` lookup instead of `p.FilesResolver.FindFileByPath` + `desc.WrapFile`) | utility (parse dispatch) | request-response | itself — current branch (`compiler/lib/parser/parser.go:117-156`) | exact |
| `compiler/lib/config.go` (delete dead `protoResolver protoregistry.MessageTypeResolver` field, line 27) | model (value object) | CRUD | itself — paired 1:1 with the compiler.go:356 assignment being deleted | exact |
| `compiler/lib/compiler.go` (delete dead `protoResolver: c.parser.LocalResolver` assignment, line 356) | service (registry wiring) | request-response | itself — same `load()` method, paired with config.go's field | exact |
| `compiler/lib/concurrent_compile_test.go` (extend `TestConcurrentCompile` with post-`g.Wait()` resolver-view assertions; SAFE-01) | test | event-driven (goroutines) | itself, current shipped test (`compiler/lib/concurrent_compile_test.go:25-58`) | exact |
| `compiler/lib/parser/lazy_parse_test.go` or new `utils` package test (dedicated `RegisterFile`-vs-`FindFileByPath` race, RSLV-03/SAFE-01) | test | CRUD / event-driven | `TestParseMemoization`'s concurrency block (`compiler/lib/parser/lazy_parse_test.go:44-60`, `errgroup` + shared registry) for the goroutine-fan-out shape; `compiler/lib/parser/parser_test.go:20` `TestParser_ParseFilesX` for the RSLV-03 pointer-identity assertion shape | exact / role-match |

## Pattern Assignments

### `utils/utils.go` — growable `filesResolver` field + locked accessors + growth hooks (RSLV-01, RSLV-02, D-05)

**Analog:** itself. Current struct (`utils/utils.go:47-71`, verified this session — note this already
includes Phase 11's `mu`/`group`/`lazyLoaded`/`eagerFallback`/`afterParseHook`, so this phase adds to an
already-locked struct, not a bare one):

```go
type DescriptorRegistry struct {
	MessageRegistry msgregistry.MessageRegistry
	FileRegistry    map[string]*desc.FileDescriptor
	localFiles      map[string]struct{}

	ImportPaths []string

	mu            sync.RWMutex
	group         singleflight.Group
	lazyLoaded    map[string]struct{}
	eagerFallback bool

	afterParseHook func()
}
```
Add `filesResolver *protoregistry.Files` and `registrationCount int` here — both guarded by the existing
`mu`, per D-01's rejection of a second lock. Initialize `filesResolver` in `NewDescriptorRegistry()`
(`utils/utils.go:73-95`) via the existing `GetFilesResolver()` call (cheap: near-empty seed for a lazy
registry, per RESEARCH.md's "260ms is already gone" finding) — cache the result on the struct instead of
recomputing it, which is the one behavior change `NewParserWithDescriptorRegistry` (below) must pick up.

**Locked-accessor shape to copy exactly** (`utils/utils.go:378-382`, `LoadedFileCount`, D-05's own named
model):
```go
func (d *DescriptorRegistry) LoadedFileCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.lazyLoaded)
}
```
New `FindFileByPath`/`RangeFiles` methods on `DescriptorRegistry` follow this exact `RLock`/`defer
RUnlock`/single-field-read shape; a new registration-count getter (D-05's observable) follows it too — the
same three-line pattern, no new idiom needed.

**Growth insert point 1 — `recordFileLocked`, current shipped form** (`utils/utils.go:333-343`, exact,
already the single writer and already idempotent — this is the "one extra line" RESEARCH.md's Pattern 1
describes):
```go
func (d *DescriptorRegistry) recordFileLocked(fd *desc.FileDescriptor) {
	if _, ok := d.FileRegistry[fd.GetName()]; ok {
		return
	}
	d.FileRegistry[fd.GetName()] = fd
	d.lazyLoaded[fd.GetName()] = struct{}{}
	d.MessageRegistry.AddFile("type.googleapis.com", fd)
	for _, dep := range fd.GetDependencies() {
		d.recordFileLocked(dep)
	}
}
```
Add the `filesResolver.RegisterFile(fd.UnwrapFile())` call (+ `registrationCount++`) inside the
`if _, ok := ...; ok { return }` guard's else-path — i.e. right after `d.MessageRegistry.AddFile(...)`,
before the dependency recursion, exactly per RESEARCH.md Pattern 1. This is called under `d.mu.Lock()`
already (caller discipline documented at line 332: "Callers must hold d.mu for writing") — no new lock
needed. Use `slog.Error` on a `RegisterFile` error (matches this file's existing convention, see Shared
Patterns below), and do not let that error abort recursion into dependencies.

**Growth insert point 2 — `ParseAll`'s before/after diff, current shipped form** (`utils/utils.go:352-373`,
exact — this is Pitfall 3's target: reuse this diff, do not write a second one):
```go
func (d *DescriptorRegistry) ParseAll() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.eagerFallback || len(d.ImportPaths) == 0 {
		return nil
	}
	before := make(map[string]struct{}, len(d.FileRegistry))
	for k := range d.FileRegistry {
		before[k] = struct{}{}
	}
	err := d.Import(d.Parse, []*regexp.Regexp{}, d.ImportPaths...)
	for k := range d.FileRegistry {
		if _, ok := before[k]; !ok {
			d.lazyLoaded[k] = struct{}{}
		}
	}
	d.eagerFallback = true
	return err
}
```
`Import`/`Parse` write `FileRegistry` directly (not through `recordFileLocked` — confirmed, this is
Pitfall 3's root cause), so the second loop over "names newly present" is exactly where the matching
`filesResolver.RegisterFile(d.FileRegistry[k].UnwrapFile())` call belongs — same diff, same keys, same
loop, per D-02's "uniform insert point" and Pitfall 3's explicit warning against a second independent pass.

**Error-wrapping / logging idiom already established in this file** (used throughout — e.g.
`GetFilesResolver`, line 130): `slog.Error("failed to <verb>", "error", err.Error())` for non-fatal
best-effort operations (a `RegisterFile` duplicate/conflict error should not abort the calling parse); the
sentinel-error + `errors.Join` idiom (lines 39, 241, 245, 292) is reserved for errors returned to the
caller, not for background bookkeeping like resolver growth.

---

### `compiler/lib/parser/parser.go` — `ParseFilesX` branch 2 rewrite (RSLV-03)

**Analog:** itself, current shipped code (`compiler/lib/parser/parser.go:117-156`, exact — already differs
from RESEARCH.md's citation of `WrapFile`/`CreateFileDescriptor` on a bare `p.FilesResolver` hit; the
`ParseOne` fallback already exists for the *miss* case, this phase's job is fixing the *hit* case):
```go
func (p *Parser) ParseFilesX(filenames ...string) (results []*desc.FileDescriptor, err error) {
	for _, filename := range filenames {
		if fd, ok := p.registry.FileDescriptor(filename); ok {
			results = append(results, fd)
			continue
		}
		resolvedFd, resolverErr := p.FilesResolver.FindFileByPath(filename)
		if resolverErr != nil {
			parsed, parseErr := p.registry.ParseOne(filename)
			if parseErr != nil {
				return nil, errors.Join(resolverErr, parseErr)
			}
			results = append(results, parsed)
			continue
		}
		fd := resolvedFd
		d, err := desc.WrapFile(fd)
		// ... falls back to desc.CreateFileDescriptor on WrapFile error ...
		results = append(results, d) // <-- RSLV-03 hazard: non-canonical pointer on this path
	}
	return results, nil
}
```
Two things need fixing, both already flagged in RESEARCH.md Pattern 2 / Pitfall 2:
1. `p.FilesResolver.FindFileByPath` (line 123) reads the field directly, unlocked — once `filesResolver`
   grows, this is the "only unlocked reader of a growable `FilesResolver` in the tree" RESEARCH.md
   identifies. Replace with the new locked `p.registry.FindFileByPath(filename)` accessor added to
   `DescriptorRegistry` above.
2. On a hit, do not call `desc.WrapFile`/`desc.CreateFileDescriptor` at all — route back through
   `p.registry.FileDescriptor(resolvedFd.Path())` (the exact method already used at the top of this loop,
   line 119, which is `ParseOne`'s own trusted canonical-pointer accessor) before falling back to
   `p.registry.ParseOne(filename)` if that second lookup also misses. This reuses the loop's own existing
   `ParseOne` fallback call (lines 125-129) rather than introducing a third code path.

**Imports** (`compiler/lib/parser/parser.go:1-20`, unchanged — no new import needed; `desc.WrapFile`/
`desc.CreateFileDescriptor`/`protodesc.ToFileDescriptorProto` become dead in this function and should be
checked for removal from the import block if no longer used elsewhere in the file — `protodesc` and
`desc.LoadFileDescriptor` are still used by `NewParserWithDescriptorRegistry`/other functions in this file,
verify per-symbol before deleting an import line).

**`RegistryTypeResolver`** (`compiler/lib/parser/parser.go:57-115`) — D-01 leaves this untouched; it is the
reference miss-fallthrough shape (snapshot → `MessageRegistry` → `ParseAll` retry) if any part of this
phase's discretion items end up wanting a fallthrough rather than growth, but no edit is expected here.

---

### `compiler/lib/config.go` + `compiler/lib/compiler.go` — delete dead `protoResolver` field

**Analog:** itself, both current shipped sites, exact:

`compiler/lib/config.go:27`:
```go
type config struct {
	...
	protoResolver   protoregistry.MessageTypeResolver
	...
}
```
`compiler/lib/compiler.go:356` (inside `load()`):
```go
return &config{
	...
	protoResolver: c.parser.LocalResolver,
	...
}, nil
```
Delete both lines together in the same edit (they are a paired field/assignment, same shape 11-PATTERNS.md
used for the `messageRegistry` pointer fix). Confirm zero readers before deleting — RESEARCH.md's A3
assumption already verified this via `grep -rn "\.protoResolver\b" --include="*.go" compiler/` returning
zero matches; re-run that grep at execute time as a final check, since new code from this phase's own edits
must not introduce a new reader. If `protoregistry` becomes an unused import in `config.go` after deletion,
remove that import line too.

---

### `compiler/lib/concurrent_compile_test.go` — extend `TestConcurrentCompile` (SAFE-01)

**Analog:** itself, current shipped test (`compiler/lib/concurrent_compile_test.go:25-58`, exact — already
races 8 goroutines via `errgroup.Group`, each compiling a `.pconf` that loads one private + one shared
proto, against one shared `*lib.Compiler`, then asserts `LoadedFileCount()` bounds):
```go
func TestConcurrentCompile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, testdata.GenerateCorpus(dir, 30))
	...
	c, err := NewCompiler(dir, false)
	require.NoError(t, err)

	g := new(errgroup.Group)
	for k := 0; k < n; k++ {
		k := k
		g.Go(func() error {
			return c.CompileFile(fmt.Sprintf("concurrent%d.pconf", k))
		})
	}
	require.NoError(t, g.Wait())
	...
	loadedCount := c.ModuleService.GetProtoRegistry().LoadedFileCount()
	require.Greater(t, loadedCount, n, ...)
	require.Less(t, loadedCount, 30, ...)
}
```
Add a resolver-view assertion after `g.Wait()`, in the same style as the existing `loadedCount` checks:
range `c.ModuleService.GetProtoRegistry()`'s new `RangeFiles` accessor, count entries, and assert that
count matches (or is consistent with) `LoadedFileCount()` — proving the growable `filesResolver` ended up
with the same set as `FileRegistry`/`lazyLoaded` post-concurrency (D-02's "never diverge" invariant, made
observable). This is a same-test extension, not a new test function — follow `require.Greater`/
`require.Less`/`require.Equal` from `testify`, already imported (`github.com/stretchr/testify/require`).

---

### `compiler/lib/parser/lazy_parse_test.go` or new dedicated race test — SAFE-01 / RSLV-03 pointer identity

**Analog for the goroutine-fan-out shape:** `TestParseMemoization`'s concurrency block, current shipped
code (`compiler/lib/parser/lazy_parse_test.go:44-60`, exact):
```go
dr2 := utils.NewDescriptorRegistry()
dr2.ImportPaths = []string{src}

const n = 8
var mu sync.Mutex
pointers := make([]interface{}, 0, n)

g := new(errgroup.Group)
for i := 0; i < n; i++ {
	g.Go(func() error {
		fd, err := dr2.ParseOne("pkg7/msg7.proto")
		...
```
For a dedicated `RegisterFile`-vs-`FindFileByPath`/`RangeFiles` race (RESEARCH.md's Open Question 2
recommendation: "do both" — extend `TestConcurrentCompile` *and* add a lower-level dedicated test), mirror
this exact shape but in package `utils` (unexported `filesResolver`/`registrationCount` fields need
package-internal test access, matching the existing convention noted in 11-PATTERNS.md for `localFiles`):
one goroutine loop calling `dr.ParseOne(distinctPath)` for N distinct paths, one concurrent reader goroutine
calling the new `dr.RangeFiles`/`dr.FindFileByPath` in a tight loop, run under `go test -race`
(`.github/workflows/go.yml`'s existing gate). Place this in `utils/utils_test.go` if it exists (check at
execute time), else a new `utils/growable_resolver_test.go`, `package utils`.

**Analog for the RSLV-03 pointer-identity assertion:** `compiler/lib/parser/parser_test.go:20`
`TestParser_ParseFilesX` — read its current body at execute time for exact construction, and use
`require.Same(t, fdFromFirstLoad, fdFromReReference)` (not output/JSON equality) per Pitfall 2's explicit
warning that output-only assertions do not guard the canonical-pointer contract. `TestParseMemoization`
already uses `require.Same(t, fd1, fd2, ...)` at this exact shape (`lazy_parse_test.go`, confirmed above) —
copy that assertion form for the new "load A, load B, re-reference A" sequence through `ParseFilesX`
specifically (not `ParseOne` directly, since RSLV-03's hazard is in `ParseFilesX`'s own branch 2).

## Shared Patterns

### Lock discipline: never hold `d.mu` while calling into `parser.ParseFiles`
**Source:** `utils/utils.go:265-269`, comment + code already shipped in `ParseOne`, guarded by
`utils/parse_all_deadlock_test.go` (existing regression test — do not remove or weaken).
**Apply to:** Any new code touching `filesResolver` inside `recordFileLocked` or `ParseAll` — both already
run entirely inside `d.mu.Lock()`/`d.mu.RLock()` sections that never call back into `parser.ParseFiles`, so
adding `RegisterFile` calls there introduces no new inversion risk, but any *new* function must be checked
against this same rule before being written.

### Locked-accessor three-line shape
**Source:** `utils/utils.go:378-382` (`LoadedFileCount`), `utils/utils.go:109-114` (`FileDescriptor`).
**Apply to:** Every new read method on `DescriptorRegistry` this phase adds (`FindFileByPath`, `RangeFiles`,
the D-05 registration-count getter) — `RLock`/`defer RUnlock`/single operation, nothing more.

### `slog.Error` for best-effort background bookkeeping vs. `errors.Join` for caller-facing errors
**Source:** `utils/utils.go:130` (`slog.Error("failed to generate files resolver", "error", err.Error())`)
vs. `utils/utils.go:241,245,292` (`errors.Join(ErrX, fmt.Errorf(...))`).
**Apply to:** A `RegisterFile` failure inside `recordFileLocked`/`ParseAll` should log via `slog.Error` and
continue (matches `GetFilesResolver`'s own precedent for a `protodesc.NewFiles`/registration failure) rather
than propagate as a caller-visible error — `RegisterFile` failing on one file must not fail the whole parse.

### `testify` + `errgroup` concurrency test shape
**Source:** `compiler/lib/concurrent_compile_test.go:25-58`, `compiler/lib/parser/lazy_parse_test.go:1-60`.
**Apply to:** All new/extended tests in this phase — `t.TempDir()` + `testdata.GenerateCorpus(dir, n)` for
fixtures, `errgroup.Group` for goroutine fan-out, `require.NoError`/`require.Same`/`require.Greater` from
`github.com/stretchr/testify/require`, run the whole file under `go test -race`.

## No Analog Found

None. Every file this phase touches already contains, in its current shipped form, the exact function this
phase extends (Phase 11 having created `ParseOne`/`recordFileLocked`/`ParseAll`/`LoadedFileCount`/
`RegistryTypeResolver`/`TestConcurrentCompile`/`TestParseMemoization` one phase prior) — there is no
brand-new file, package, or role in this phase's scope, consistent with CONTEXT.md's "no new constructor,
no new flag, no new package" decisions (D-01, D-04).

## Metadata

**Analog search scope:** `utils/utils.go`, `compiler/lib/parser/parser.go`, `compiler/lib/config.go`,
`compiler/lib/compiler.go`, `compiler/lib/concurrent_compile_test.go`, `compiler/lib/parser/lazy_parse_test.go`,
`compiler/lib/parser/parser_test.go`
**Files scanned:** 7 (all git-tracked, confirmed via `git ls-files`)
**Pattern extraction date:** 2026-09-07
