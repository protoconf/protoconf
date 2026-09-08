# Phase 13: Exact Symbol Index & Shared Type-URL Resolution - Pattern Map

**Mapped:** 2026-09-08
**Files analyzed:** 3 modified (no new files — RESEARCH.md's "Recommended Project Structure" is explicit: extend existing files, no new packages)
**Analogs found:** 3 / 3 (all self-referential — the closest analog to each file is the adjacent code already in that same file, extended in place)

## File Classification

| Modified File | Role | Data Flow | Closest Analog | Match Quality |
|----------------|------|-----------|-----------------|----------------|
| `utils/utils.go` (new index build/cache methods on `DescriptorRegistry`) | service (registry) | batch (parse) + file-I/O (cache) | same file: `ParseOne` (:297-395, singleflight+lock discipline), `Store`/`Load` (:561-623, checksum-gated cache) | exact — same struct, same file, same lock |
| `compiler/lib/parser/parser.go` (`RegistryTypeResolver.FindMessageByURL`/`FindMessageByName`) | service (resolver chokepoint) | request-response (miss-fallthrough chain) | same file: current `FindMessageByURL` (:66-85) is itself the tier-insertion template | exact — editing in place, not porting a pattern from elsewhere |
| `compiler/lib/starlark_loader.go` (`loadMutable`, CONS-05 fix) | service (config loader) | request-response | same file, same function: `:177`'s existing `l.parser.TypeResolver.FindMessageByURL(...)` call, one line above the divergent `:187` call to replace | exact — the correct call already exists two lines away |

No `Glob`/`Grep` search across the broader tree was needed: RESEARCH.md's
"Recommended Project Structure" and "Architectural Responsibility Map"
already name the exact three files and insertion points, verified this
session against the live source (line numbers below match what is on disk
now, not what RESEARCH.md quoted, since a couple of doc comments/lines have
shifted slightly — re-verified by direct `Read`/`grep` in this pass).

## Pattern Assignments

### `utils/utils.go` — index build, persisted cache, build counter (TYPE-01/02/04/05/06/07, D-04)

**Analog A — `ParseOne`** (lines 297-395): the lock-discipline and
singleflight-collapsing template for both the scan tier's per-candidate parse
and the index build itself.

```go
func (d *DescriptorRegistry) ParseOne(path string) (*desc.FileDescriptor, error) {
	d.mu.RLock()
	importPaths := d.ImportPaths
	d.mu.RUnlock()
	if len(importPaths) == 0 {
		return nil, errors.Join(ErrLazyParseDisabled, fmt.Errorf("path=%s", path))
	}
	...
	v, err, _ := d.group.Do(path, func() (interface{}, error) {
		// re-check under the group before doing real work
		d.mu.RLock()
		if fd, ok := d.FileRegistry[path]; ok {
			d.mu.RUnlock()
			return fd, nil
		}
		d.mu.RUnlock()
		// Lock discipline: never hold d.mu while calling parser.ParseFiles.
		parser := &protoparse.Parser{ImportPaths: importPaths, Accessor: ..., LookupImport: ...}
		fds, err := parser.ParseFiles(path)
		...
		d.mu.Lock()
		d.recordFileLocked(fds[0])
		canonical, ok := d.FileRegistry[fds[0].GetName()]
		d.mu.Unlock()
		...
	})
	return v.(*desc.FileDescriptor), nil
}
```

Apply this shape to a new `d.group.Do("__symbol_index__", ...)` call for the
index build (same `singleflight.Group` field `d.group`, already present on
`DescriptorRegistry` — reuse it, do not add a second `Group`), and reuse the
identical `RLock`-read-config → unlock → do filesystem/parse work → `Lock`
only for the insert shape for the scan tier's per-candidate `ParseOne` calls
(which is just calling the existing `ParseOne`, not new lock code).

**Analog B — `Store`/`Load`** (lines 561-623): the checksum-gated persisted
cache shape TYPE-05/06 must follow.

```go
func (d *DescriptorRegistry) Store(path string) (string, error) {
	...
	b, err := proto.Marshal(fds)
	...
	h := fileDescriptorSetSum(fds)
	return h, os.WriteFile(path, b, 0644)
}

func (d *DescriptorRegistry) Load(path, checksum string) error {
	b, err := os.ReadFile(path)
	...
	fds := &descriptorpb.FileDescriptorSet{}
	err = proto.Unmarshal(b, fds)
	...
	md5sum := fileDescriptorSetSum(fds)
	if checksum != md5sum {
		return fmt.Errorf("failed to validate file content: %s (expected: %s, got: %s)", path, checksum, md5sum)
	}
	d.MergeFileDescriptorSet(fds)
	return nil
}
```

The index cache's equivalent: unmarshal the persisted `symbol -> path` map,
recompute the content key, compare, refuse (rebuild) on mismatch — never
serve stale. Never invent a second cache-trust idiom.

**Analog C — `dirhash.HashDir` call site** (`compiler/lib/module_service.go:273`,
not `utils.go`, but the only precedent in the repo for this exact primitive):

```go
h, err := dirhash.HashDir(repoCacheDir, "", hash1)
if errors.Is(err, os.ErrNotExist) {
	return "", ErrorRemoteRepoNotDownloaded
}
if r.Integrity != "dummy" && r.Integrity != h {
	return "", errors.Join(ErrorRemoteRepoValidationFailed, ...)
}
```

If `dirhash.HashDir(srcPath, ...)` is chosen for TYPE-05/06's key (open per
CONTEXT.md discretion), this is the exact call shape and the `hash1` function
value (`module_service.go`, check its `import ( ... "golang.org/x/mod/sumdb/dirhash" )`
block for `hash1`'s definition) to reuse rather than re-derive.

**Analog D — `LoadedFileCount`** (lines 512-518): the exact shape D-04's
index-build counter should mirror.

```go
// LoadedFileCount reports how many proto files have been loaded on the lazy
// path ... Safe to call concurrently with ParseOne.
func (d *DescriptorRegistry) LoadedFileCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.lazyLoaded)
}
```

D-04's counter: same doc-comment shape ("test-only ... Safe to call
concurrently"), same `RLock`/`defer RUnlock`/return-int body, no log line, no
CLI surface — e.g. `IndexBuildCount() int`.

**Deletion target — `ParseAll`** (lines 433-460) and its two call sites in
`parser.go` (`_ = r.registry.ParseAll()`, both branches): D-02 deletes this
method's role as the resolution-chain last tier. Note it is also called from
`command/` CLI paths for the *whole-tree compile* use case (verify via grep
before deleting the method itself — D-02 only removes it as a *type-URL
resolution* fallback, not necessarily the method if something else still
calls it for a different purpose). `FellBackToEager()` (lines 543-548) is the
paired observable RESEARCH.md's Pitfall 1 says must be replaced in kind, not
just deleted.

---

### `compiler/lib/parser/parser.go` — tier insertion (TYPE-08, D-01/D-02)

**Analog — current `FindMessageByURL`** (lines 66-85 as read this session):

```go
func (r *RegistryTypeResolver) FindMessageByURL(url string) (protoreflect.MessageType, error) {
	mt, err := r.snapshot.FindMessageByURL(url)          // Tier 0 — unchanged
	if err == nil {
		return mt, nil
	}
	if !errors.Is(err, protoregistry.NotFound) {
		return nil, err
	}
	if md, mErr := r.registry.MessageRegistry.FindMessageTypeByUrl(url); mErr == nil && md != nil {
		return dynamicpb.NewMessageType(md.UnwrapMessage()), nil // Tier 1 — unchanged
	}
	// <-- D-01's scan tier (Tier 2) and index tier (Tier 3) insert HERE
	_ = r.registry.ParseAll()  // <-- D-02 deletes this call and its FindMessageByName twin
	if md, mErr := r.registry.MessageRegistry.FindMessageTypeByUrl(url); mErr == nil && md != nil {
		return dynamicpb.NewMessageType(md.UnwrapMessage()), nil
	}
	return nil, fmt.Errorf("%w: %s", protoregistry.NotFound, url)  // <-- D-02's hard error needs more context
}
```

`FindMessageByName` (lines 87-104) is the exact structural twin — apply the
same tier insertion, deriving the URL form (`"type.googleapis.com/" + string(name)`,
already the pattern at line ~93) for the scan/index lookups so both methods
share one underlying tier implementation rather than duplicating the scan+
index logic twice. `FindExtensionByName`/`FindExtensionByNumber` (lines
106-115) stay untouched — snapshot-only, confirmed by RESEARCH.md Assumption A3.

**Error-wrapping convention to reuse for the hard error** (same file, same
lines): `fmt.Errorf("%w: %s", protoregistry.NotFound, url)` — extend this,
don't replace the sentinel, when adding dirs-searched/index-state context
(D-02).

---

### `compiler/lib/starlark_loader.go` — CONS-05 fix (Pitfall 4)

**Analog — the correct call, one line above the one to fix** (lines ~177-187
as read this session):

```go
mt, err := l.parser.TypeResolver.FindMessageByURL(protoconfValue.Value.TypeUrl)
if err != nil {
	return nil, err
}
new := dynamicpb.NewMessage(mt.Descriptor())
err = protoconfValue.Value.UnmarshalTo(new)
...

// THIS is the CONS-05 divergence — bypasses TypeResolver entirely:
d, err := l.moduleService.GetProtoRegistry().MessageRegistry.FindMessageTypeByUrl(protoconfValue.Value.TypeUrl)
```

Fix: route the second lookup through `l.parser.TypeResolver` too, or
determine (per RESEARCH.md Pitfall 4) whether the `*desc.MessageDescriptor`
this second call obtains for `dynamic.NewMessage` can instead be derived from
`mt` (line 177's result) directly, collapsing two resolutions of the same
`TypeUrl` into one call through the one shared chokepoint.

---

## Shared Patterns

### Lock discipline (Phase 11 D-precedent, binding on both new tiers)
**Source:** `utils/utils.go` `ParseOne` comment block (lines ~285-296) and
`utils/parse_all_deadlock_test.go`
**Apply to:** the scan tier's per-candidate `ParseOne` calls and the index
build's file walk — both do filesystem work then parse; never hold `d.mu`
across either.

### Singleflight collapsing
**Source:** `utils/utils.go:297` `d.group.Do(path, ...)` — the existing
`*singleflight.Group` field on `DescriptorRegistry`
**Apply to:** the index build (one key, e.g. a fixed string), so two
concurrent compiles pay the build once (Claude's Discretion item, resolved
by reusing the existing field rather than adding a second `Group`).

### Checksum-gated cache trust
**Source:** `utils/utils.go` `Store`/`Load` (561-623), `fileDescriptorSetSum`
**Apply to:** the index's on-disk artifact (TYPE-05/06) — validate before
trusting, refuse and rebuild on mismatch, never serve stale.

### Error-wrapping / hard-error convention
**Source:** `compiler/lib/parser/parser.go` — `fmt.Errorf("%w: %s", protoregistry.NotFound, url)`
**Apply to:** D-02's replacement for `ParseAll`, extended with dirs-searched
and index-cache-state context, still wrapping the same sentinel so existing
`errors.Is(err, protoregistry.NotFound)` callers keep working.

### Test-only observable counters
**Source:** `utils/utils.go` `LoadedFileCount`/`FilesResolverRegistrationCount`
(492-518) — `RLock`/return-int, doc comment states "Safe to call concurrently", no log line, no CLI surface.
**Apply to:** D-04's index-build counter, and (per Pitfall 1) whatever
replaces `FellBackToEager()`'s intent — distinguishing resolved-via-scan /
resolved-via-index-cold / resolved-via-index-warm / hard-error.

## No Analog Found

| File/Concern | Role | Data Flow | Reason |
|------|------|-----------|--------|
| Scoped lexical scan (D-01/D-05, new helper — location TBD: `utils/utils.go` sibling method or a new file in `compiler/lib/parser/`) | utility (filesystem scan) | transform (regex candidate extraction) | No prior lexical/regex-over-`.proto`-source scan exists in this codebase; every existing search is proto-structural (`ParseFiles`/`ParseFilesButDoNotLink`), not textual. Build from RESEARCH.md's Pattern 1/D-05 spec directly — package-prefix→directory narrowing, indented-declaration match, never trusted without a follow-up `ParseOne` verify. |
| Recursive nested-symbol name accumulation for the index walk (TYPE-01, RESEARCH.md Pattern 1 / Pitfall 3) | transform | batch | No existing code in this repo walks `DescriptorProto.NestedType` recursively to build a symbol→path map (the closest relative, `msgregistry.AddFile`'s internal `addMessageTypesLocked`, is vendored third-party code, not a local pattern to copy — reuse it by calling it, per "Don't Hand-Roll", rather than imitating its recursion by hand). |

## Metadata

**Analog search scope:** `utils/utils.go`, `compiler/lib/parser/parser.go`,
`compiler/lib/starlark_loader.go`, `compiler/lib/module_service.go`,
`compiler/lib/config.go` — all identified directly from RESEARCH.md's
verified line references and confirmed by direct `Read`/`grep` this session
(no broader `Glob`/`Grep` sweep needed; RESEARCH.md's own scouting already
named every touch point with verified line numbers).
**Files scanned:** 5
**Pattern extraction date:** 2026-09-08
