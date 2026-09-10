# Phase 11: Concurrency-Safe Lazy Registry Core - Pattern Map

**Mapped:** 2026-09-04
**Files analyzed:** 6 edited files + 5 new test files
**Analogs found:** 11 / 11 (all in-repo; no external analogs needed)

All line numbers below were re-verified against the live tree this session (drifted by a few
lines from RESEARCH.md's citations in some cases — corrected here).

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|-----------------|---------------|
| `utils/utils.go` (add `mu sync.RWMutex`, `group singleflight.Group`, `lazyLoaded map[string]struct{}`, `ParseOne` method) | model/service (shared cache) | CRUD (keyed cache read/write) | `compiler/lib/module_service.go` (`ModuleService.mutex sync.RWMutex` field pattern) + itself (`Parse`/`Import`, same file) | exact (self-file edit, mutex pattern borrowed from sibling) |
| `compiler/lib/parser/parser.go` (`ParseFilesX` miss branch calls `ParseOne` instead of erroring) | utility (parse dispatch) | request-response | itself — existing map-lookup fast path (lines 34-43) | exact |
| `compiler/lib/module_service.go` (`GetProtoRegistry()` stop eager `Import` for compiler path) | service | CRUD | itself — `Sync()`'s separate eager instance (lines ~399-408, see below) as the "keep this shape, don't touch it" reference | exact (self, plus internal precedent) |
| `compiler/lib/compiler.go` (fix `MessageRegistry` struct-copy at line 355; add loaded-file-count log) | service (registry wiring) | request-response | itself — existing `slog.Info("module service loaded", "took", ...)` at line 66 | exact |
| `compiler/lib/config.go` (`messageRegistry msgregistry.MessageRegistry` → `*msgregistry.MessageRegistry`) | model (value object) | CRUD | itself — struct field declaration, paired 1:1 with compiler.go:355's assignment | exact |
| `server/server.go` (`Init()` gets its own eager discovery registry) | controller (gRPC service wiring) | request-response | `compiler/lib/module_service.go` `Sync()` (lines ~399-408) — the "fresh throwaway `utils.NewDescriptorRegistry()` + `Import`" shape | role-match (cross-file, same shape) |
| New: `compiler/lib/parse_memoization_test.go` (or similar, LAZY-02) | test | request-response | `compiler/lib/startup_bench_test.go` (table/setup style) | role-match |
| New: concurrent-compile race test (LAZY-02 concurrency) | test | event-driven (goroutines) | `compiler/service.go:31-56` (`errgroup.Go` per file over one shared `*lib.Compiler`) — production pattern to replicate in test form | exact (mirrors production concurrency shape) |
| New: `utils/lazy_parse_no_mutate_test.go` (LAZY-03) | test | CRUD | `utils/utils.go`'s own `Parse`/`Store` tests (if any) or `startup_bench_test.go` style | role-match |
| New: `.fds` byte-identity test (LAZY-04) | test | file-I/O | `compiler/lib/module_service.go` `Sync()`/`GenFileDescriptorSet` + `Store()` (`utils/utils.go:188-205`) | role-match |
| New: `server/server_test.go` addition, `TestInitRegistersCustomService` (CONS-01) | test | request-response | `server/server_test.go:142-158` `TestProtoconfMutationServer_GenReflectionUI` (same `NewProtoconfMutationServer(testdata.SmallTestDir())` + `Init(rpcServer)` construction) | exact |

## Pattern Assignments

### `utils/utils.go` — add lock + singleflight-guarded `ParseOne` (LAZY-02, LAZY-03)

**Analog:** itself. Current struct and both existing write paths, verified this session:

**Struct today** (`utils/utils.go:32-36`):
```go
type DescriptorRegistry struct {
	MessageRegistry msgregistry.MessageRegistry
	FileRegistry    map[string]*desc.FileDescriptor
	localFiles      map[string]struct{}
}
```
Add `mu sync.RWMutex`, `group singleflight.Group`, `lazyLoaded map[string]struct{}` — initialize the last one in `NewDescriptorRegistry()` alongside the existing `FileRegistry`/`localFiles` init (same function, ~line 38-59).

**`Import`'s existing `LookupImport` closure** (`utils/utils.go:135-144`) — copy this exact shape for `ParseOne`'s `LookupImport`, it already does read-then-fallback-to-`desc.LoadFileDescriptor`:
```go
LookupImport: func(s string) (*desc.FileDescriptor, error) {
	if fd, ok := d.FileRegistry[s]; ok {
		return fd, nil
	}
	fd, err := desc.LoadFileDescriptor(s)
	if err == nil {
		d.FileRegistry[s] = fd
		return fd, nil
	}
	return nil, fmt.Errorf("failed to find descriptor for file: %s", s)
},
```
Wrap `d.FileRegistry` reads/writes in `d.mu.RLock()`/`d.mu.Lock()` for the new lazy path only — do not add locking to `Import`/`Parse` themselves (they are single-shot, whole-tree callers on their own dedicated registry instances; adding locks there is unnecessary surface).

**`Parse` today** (`utils/utils.go:158-170`, exact text, confirms LAZY-03's target):
```go
func (d *DescriptorRegistry) Parse(parser *protoparse.Parser, files []string) error {
	d.localFiles = map[string]struct{}{}
	descriptors, err := parser.ParseFiles(files...)
	for _, fd := range descriptors {
		d.MessageRegistry.AddFile("type.googleapis.com", fd)
		d.FileRegistry[fd.GetName()] = fd
		d.localFiles[fd.GetName()] = struct{}{}
	}
	if err != nil {
		return errors.Join(errors.New("failed to parse files"), err)
	}
	return nil
}
```
`ParseOne` must **never** touch `d.localFiles` — write only to `d.FileRegistry` and `d.lazyLoaded`, call `d.MessageRegistry.AddFile` (already internally mutex-protected — confirmed via `$GOMODCACHE` read in RESEARCH.md, no extra lock needed around that call itself, only around the map inserts).

**Error-wrapping idiom to copy** (used throughout this file): `errors.Join(errors.New("failed to <verb> files"), err)` — use the same shape for any new error path in `ParseOne`.

**singleflight usage:** `golang.org/x/sync/singleflight` — zero new import lines needed beyond `"golang.org/x/sync/singleflight"`; `errgroup` from the same module is already imported in `compiler/service.go` and `compiler/command.go`, confirming the module is already in `go.sum`.

---

### `compiler/lib/parser/parser.go` — `ParseFilesX` miss branch (LAZY-02)

**Analog:** itself, exact current code (`compiler/lib/parser/parser.go:34-47`):
```go
func (p *Parser) ParseFilesX(filenames ...string) (results []*desc.FileDescriptor, err error) {
	for _, filename := range filenames {
		if fd, ok := p.FileDescriptors[filename]; ok {
			results = append(results, fd)
			continue
		}
		fd, err := p.FilesResolver.FindFileByPath(filename)
		if err != nil {
			return nil, err
		}
		d, err := desc.WrapFile(fd)
		if err != nil {
```
Note `p.FileDescriptors` is the same map object as `DescriptorRegistry.FileRegistry` (assigned at construction, `NewParserWithDescriptorRegistry`, line ~29: `FileDescriptors: registry.FileRegistry`). The miss branch currently calls `p.FilesResolver.FindFileByPath(filename)` and errors if not found — that is the exact spot to replace with a call to the registry's new `ParseOne(importPaths, filename)`. The `Parser` struct needs a reference back to the owning `*utils.DescriptorRegistry` (or `ParseOne` needs to be reachable) to make this call — check whether `NewParserWithDescriptorRegistry` should retain the `*utils.DescriptorRegistry` pointer as a new unexported field rather than only unpacking its three public views, since `ParseFilesX`'s only registry access today is via `FileDescriptors`/`FilesResolver`.

**Imports pattern** (lines 1-16, current file) — keep the existing blank-import registration block unchanged:
```go
import (
	"os"

	_ "github.com/bufbuild/protovalidate-go"
	_ "github.com/bufbuild/protovalidate-go/legacy"
	_ "github.com/protoconf/protoconf/pb/protoconf/v1"

	"github.com/jhump/protoreflect/desc"
	"github.com/protoconf/protoconf/utils"
	...
)
```

---

### `compiler/lib/module_service.go` — `GetProtoRegistry()` scoping (LAZY-01)

**Current eager call** (`compiler/lib/module_service.go`, verified this session, function starts ~line 348):
```go
func (m *ModuleService) GetProtoRegistry() *utils.DescriptorRegistry {
	if m.cachedRegistry != nil {
		return m.cachedRegistry
	}
	registry := utils.NewDescriptorRegistry()
	m.Walk(func(r *module.RemoteRepo) error {
		...
		err := registry.Load(filepath.Join(m.getCacheDir(), r.Label+".fds"), r.FileDescriptorSetSum)
		...
	})
	err := registry.Import(registry.Parse, []*regexp.Regexp{}, filepath.Join(m.getProtoconfPath(), consts.SrcPath))
	// ^ THIS is the line to remove/guard for the compiler's construction path (LAZY-01)
	if err != nil {
		slog.Error("failed to parse proto files", slog.String("error", err.Error()))
	}
	m.cachedRegistry = registry
	return registry
}
```
Per RESEARCH.md's Open Question 1 recommendation: keep the `.Load(...)` calls (loading cached `.fds` from `.protoconf_cache/`, populated by `mod sync`) but drop the `registry.Import(registry.Parse, ...)` whole-`src/`-walk line for the compiler's path. The four other consumers (`server/server.go`, `inserter/inserter.go`, `agent/filekv/filekv.go`, `mutate/mutate.go`) all call this same function — confirm with the plan whether a second entry point (`GetLazyProtoRegistry()`) is needed so those four keep receiving the eager-Import behavior unchanged this phase (their tests, e.g. `inserter/inserter_test.go`, depend on it).

**`Sync()`'s separate eager instance — the pattern to copy for CONS-01** (`compiler/lib/module_service.go`, `Sync()` body, verified):
```go
registry := utils.NewDescriptorRegistry()
err = m.Walk(func(r *module.RemoteRepo) error {
	if r.Url == "." {
		return nil
	}
	err := m.GenFileDescriptorSet(registry, r)
	return err
})
return err
```
This is a fresh, throwaway registry, never shared with `m.cachedRegistry` — exactly the shape `server/server.go`'s `Init()` fix should copy (see below), and exactly what must NOT change for LAZY-04.

---

### `compiler/lib/compiler.go` + `compiler/lib/config.go` — `MessageRegistry` pointer fix + loaded-file-count log

**Struct-copy site, current exact code** (`compiler/lib/compiler.go`, `load()` method, verified):
```go
func (c *Compiler) load(filename string) (*config, error) {
	loader := c.GetLoader()
	locals, validators, err := loader.loadConfig(filepath.ToSlash(filename))
	if err != nil {
		return nil, err
	}

	return &config{
		filename:        filename,
		locals:          locals,
		validators:      validators,
		protoResolver:   c.parser.LocalResolver,
		messageRegistry: c.ModuleService.GetProtoRegistry().MessageRegistry,
		// ^ value copy — change to &c.ModuleService.GetProtoRegistry().MessageRegistry
		protoValidator:  c.validator,
	}, nil
}
```

**Receiving field, current exact code** (`compiler/lib/config.go`):
```go
type config struct {
	filename        string
	locals          starlark.StringDict
	validators      map[string]*starlark.Function
	messageRegistry msgregistry.MessageRegistry
	// ^ change to *msgregistry.MessageRegistry
	protoResolver   protoregistry.MessageTypeResolver
	protoValidator  *protovalidate.Validator
}
```
Any other read site of `c.messageRegistry` in `config.go` (e.g. `FindMessageTypeByUrl` call) needs no change beyond the type — pointer methods work the same via `.` on a pointer field.

**Existing `slog.Info` convention to extend (LAZY-05)** (`compiler/lib/compiler.go:66`, exact):
```go
slog.Info("module service loaded", "took", time.Since(t))
```
Add a second, analogous line where `CompileFile`/`CompileFileAsync` finishes (not at construction — the registry is still near-empty then), reading `len(registry.lazyLoaded)` (new field) via an exported accessor, e.g. `slog.Info("compile finished", "protoFilesLoaded", registry.LoadedFileCount())`. Keep the same key-value `slog` call shape (string message, then alternating string/value pairs) already used throughout this file.

---

### `server/server.go` — `Init()` eager discovery fix (CONS-01)

**Current discovery loop, exact code verified this session** (`server/server.go`, `Init()`):
```go
func (s *ProtoconfMutationServer) Init(rpcServer *grpc.Server) {
	protoconfmutation.RegisterProtoconfMutationServiceServer(rpcServer, &legacyProtoconfMutationServer{srv: s})
	protoconf_pb.RegisterProtoconfMutationServiceServer(rpcServer, s)
	protoconf_pb.RegisterProtoconfMutationReportServiceServer(rpcServer, s)

	s.exampleMaker = map[string]exampleFunc{}
	s.parser.FilesResolver.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		_, err := protoregistry.GlobalFiles.FindFileByPath(fd.Path())
		if !errors.Is(err, protoregistry.NotFound) {
			return true
		}
		if fd.Services().Len() < 1 {
			return true
		}
		for i := 0; i < fd.Services().Len(); i++ {
			svc := fd.Services().Get(i)
			... // registration body, unchanged
		}
		return true
	})
	...
}
```
Fix: build a throwaway `discoveryRegistry := utils.NewDescriptorRegistry()`, call `discoveryRegistry.Import(discoveryRegistry.Parse, nil, filepath.Join(s.protoconfRoot, consts.SrcPath))` (same signature `Sync()` uses), get `discoveryFiles := discoveryRegistry.GetFilesResolver()`, and range over `discoveryFiles` instead of `s.parser.FilesResolver` in this one loop only. Leave `s.parser` untouched everywhere else in the file (used by `Put()`/`MutateConfig`, per RESEARCH.md Open Question 3 — stays eager/unchanged this phase since `s.parser` is still built via the unmodified `GetProtoRegistry()` four-consumer path).

**Construction site for context** (`NewProtoconfMutationServer`, verified):
```go
ms, err := lib.NewModuleService(protoconfRoot)
...
parser := parser.NewParserWithDescriptorRegistry(ms.GetProtoRegistry())
```

---

### Test analogs

**LAZY-02 memoization test** — model the setup on `compiler/lib/startup_bench_test.go:20` (`TestGeneratedCorpusCompiles`): use `testdata.GenerateCorpus(t.TempDir(), n)` fixture generation, `require.NoError`/`require.Same` from testify (already imported project-wide). Pointer-identity assertion: call the parse path twice with the same path, `require.Same(t, fd1, fd2)`.

**LAZY-02 concurrency test** — the production concurrency shape to mirror is `compiler/service.go:31-56`'s `errgroup.Go` per file against one shared `*lib.Compiler` (confirmed present in that file and in `compiler/command.go`'s `runLocally`, ~line 140). New test: one `*lib.Compiler` from `NewCompiler(dir, false)`, `errgroup.Group` with N `Go()` calls each invoking `c.CompileFile(distinctFile)` on a `testdata.GenerateCorpus`-produced multi-config dir, run the whole test file under `go test -race`.

**LAZY-03 test** — assert on `DescriptorRegistry.localFiles`'s length being unchanged after a `ParseOne` call; since `localFiles` is unexported, this test must live in package `utils` (not `utils_test`), following the existing convention of internal test access (check for an existing `utils/utils_test.go` — if present, add there; if absent, create `utils/utils_test.go` following the package name of `utils.go` itself, i.e. `package utils`).

**LAZY-04 `.fds` byte-identity test** — reuses `ModuleService.GenFileDescriptorSet`/`Sync()` and `DescriptorRegistry.Store()` (`utils/utils.go:188-205` region, serializes `d.localFiles`'s current contents) — write/compare `.fds` bytes before/after the phase's changes on the same fixture corpus (e.g. `testdata.SmallTestDir()` or a `GenerateCorpus` output). Confirm no existing test already asserts on `.fds` byte content before adding (RESEARCH.md flags this as unverified — a `grep -rn "GenFileDescriptorSet\|\.fds" **/*_test.go` should be run again at plan/execute time).

**CONS-01 test — exact analog to copy** (`server/server_test.go:142-152`, `TestProtoconfMutationServer_GenReflectionUI`, confirmed verbatim):
```go
func TestProtoconfMutationServer_GenReflectionUI(t *testing.T) {
	protoconfRoot := testdata.SmallTestDir()
	server, err := NewProtoconfMutationServer(protoconfRoot)
	...
	rpcServer := grpc.NewServer()
	server.Init(rpcServer)
	...
	err = server.GenReflectionUI(ctx, rpcServer, httpServer)
	...
}
```
New test `TestInitRegistersCustomService` should follow this exact construction (`NewProtoconfMutationServer(testdata.SmallTestDir())` → `grpc.NewServer()` → `server.Init(rpcServer)`), then assert `rpcServer.GetServiceInfo()` contains `"test.v1.TestService"` (fixture at `utils/testdata/small/src/test.proto:71-74`, confirmed verbatim):
```protobuf
service TestService {
    rpc PutTestMessage(TestMessage) returns (protoconf.v1.ConfigMutationResponse);
    rpc PutValidateMe(ValidateMe) returns (protoconf.v1.ConfigMutationResponse);
}
```
Critically, this new test must **not** call `CompileFile` before asserting — the whole point is proving `Init()` alone (no compile) discovers the service.

## Shared Patterns

### Structured logging (`slog`)
**Source:** `compiler/lib/compiler.go:66`, `slog.Info("module service loaded", "took", time.Since(t))`; also `utils/utils.go`'s `slog.Debug`/`slog.Error` calls throughout `Import`.
**Apply to:** `utils/utils.go` (`ParseOne` failures), `compiler/lib/compiler.go` (LAZY-05 counter), `compiler/lib/module_service.go` (any new error path in the trimmed `GetProtoRegistry()`).
**Convention:** `slog.<Level>("<lowercase message>", "<key>", value, ...)` — message text is a short lowercase phrase, no trailing punctuation, key-value pairs follow.

### Error wrapping
**Source:** `utils/utils.go` throughout: `errors.Join(errors.New("failed to <verb> files"), err)`.
**Apply to:** Any new error return in `ParseOne` (`utils/utils.go`) — match this exact `errors.Join(errors.New(...), err)` shape rather than `fmt.Errorf("...: %w", err)`, to stay consistent with this file's existing idiom (note: `fmt.Errorf` with `%w` is used elsewhere in the codebase per CLAUDE.md conventions — but within `utils/utils.go` specifically, `errors.Join` is the local convention already established by every sibling function in that file).

### Test construction with `testdata.SmallTestDir()` / `testdata.GenerateCorpus()`
**Source:** `utils/testdata/embed.go:109` (`SmallTestDir()`), `utils/testdata/corpus.go:26` (`GenerateCorpus(dir string, n int) error`, requires `n >= 5`).
**Apply to:** All new tests in this phase (LAZY-02, LAZY-02-concurrency, CONS-01) — reuse these fixtures rather than authoring new `.proto`/`.pconf` files. `TestService` (CONS-01's fixture) already exists in `utils/testdata/small/src/test.proto:71-74`; no new fixture proto is needed for any requirement in this phase.

### `errgroup` for concurrent compiles (production pattern to replicate in tests)
**Source:** `compiler/service.go:31-56` (`CompileFiles` gRPC handler, `errgroup.Go` per file against shared `*lib.Compiler`); `compiler/command.go` `runLocally` (~line 140) does the identical thing for the CLI path.
**Apply to:** The new LAZY-02 concurrency test — construct the shared `*lib.Compiler` once, then use `golang.org/x/sync/errgroup` (already imported elsewhere in the module) to drive N goroutines, exactly mirroring these two production call sites.

## No Analog Found

None — every file in scope has a same-file (self) analog (the existing pre-lazy code being edited in place) or a strong cross-file structural analog (`Sync()`'s eager-instance shape for `Init()`'s fix; `GenReflectionUI`'s test-construction shape for the new CONS-01 test).

## Metadata

**Analog search scope:** `utils/utils.go`, `compiler/lib/parser/parser.go`, `compiler/lib/module_service.go`, `compiler/lib/compiler.go`, `compiler/lib/config.go`, `server/server.go`, `server/server_test.go`, `inserter/inserter_test.go`, `compiler/lib/startup_bench_test.go`, `utils/testdata/embed.go`, `utils/testdata/corpus.go`, `utils/testdata/small/src/test.proto`, `compiler/service.go`, `compiler/command.go`
**Files scanned:** 13
**Pattern extraction date:** 2026-09-04
