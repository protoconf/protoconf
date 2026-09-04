# Testing: How to Measure and How to Not Regress

The repo has exactly one benchmark today (`agent/kv_agent_impl_test.go`) and no
compiler performance coverage at all. That is the first gap to close — a
performance change with no regression test decays back to 6.9s within a year.

## Pin a corpus first

Every number in BASELINE.md came from `protoconf-terraform/example`, which is a
live working tree. Its proto count will drift, and the measurement drifts with it.
A benchmark against an unpinned corpus measures the corpus.

Options, cheapest first:

- **Generate it.** A `testdata` generator that emits N synthetic protos with a
  configurable import fan-out. Deterministic, versioned with the code, and lets
  the benchmark parameterise on N — which is the property that actually matters
  here, since the bug is "cost scales with repo size, not config size". Recommended.
- **Vendor a slice.** Copy a fixed subset of the terraform protos into `testdata/`.
  Realistic shapes, but bulky in-repo and still a snapshot.
- **Submodule / fetch-on-demand.** Faithful, but makes the benchmark
  network-dependent and CI-fragile. Not worth it.

The generator is the right call: the assertion worth making is *scaling*, and only
a generator lets you assert it.

## The assertion that matters

Absolute wall-clock in CI is noisy and machine-dependent — a 200ms threshold on a
shared runner will flake. The defensible assertion is **scaling behaviour**:

> Compiling a config that loads 5 protos costs the same whether the repo contains
> 50 protos or 5,000.

Concretely: run the compile at N=50 and N=5000 and assert the ratio stays under a
small constant (say 2x). That is exactly the property being fixed, it is robust to
machine speed, and it fails loudly if anyone reintroduces an eager walk.

Keep a wall-clock benchmark too, but as a **reported number, not a gate** — same
posture the project already took with codecov thresholds (quick task 260902-cov).

## Benchmark shape

```go
func BenchmarkCompilerStartup(b *testing.B) {
    for _, n := range []int{50, 500, 5000} {
        b.Run(fmt.Sprintf("protos=%d", n), func(b *testing.B) {
            root := generateCorpus(b, n)   // b.TempDir(), N protos, config loads 5
            b.ResetTimer()
            for i := 0; i < b.N; i++ {
                c, err := lib.NewCompiler(root, false)
                if err != nil { b.Fatal(err) }
                if err := c.CompileFile("main.mpconf"); err != nil { b.Fatal(err) }
            }
        })
    }
}
```

Run with `-benchmem`. Allocation count is the leading indicator here — the profile
showed ~40% of time in GC, so `allocs/op` will move before `ns/op` does and is far
less noisy. Track it.

## Stage-level instrumentation

The stage table in BASELINE.md was produced by timing each constructor
independently. Worth keeping as a throwaway harness rather than shipped code:

```go
t := time.Now(); ms, _ := lib.NewModuleService(root); ms.LoadFromLockFile()
fmt.Println("NewModuleService+lock:", time.Since(t))

t = time.Now(); reg := ms.GetProtoRegistry()
fmt.Println("GetProtoRegistry:", time.Since(t), "files:", len(reg.FileRegistry))

t = time.Now(); _ = parser.NewParserWithDescriptorRegistry(reg)
fmt.Println("NewParserWithDescriptorRegistry:", time.Since(t))
```

`NewCompiler` already logs `slog.Info("module service loaded", "took", ...)`
(`compiler/lib/compiler.go:66`). Extending that one line to also report the file
count would make the problem visible in the field without any harness at all —
"loaded 864 files in 4.6s" is self-evidently wrong to anyone reading it.

## Profiling recipe

```go
f, _ := os.Create("cpu.out")
pprof.StartCPUProfile(f); defer pprof.StopCPUProfile()
// ... construct compiler, compile ...
```

```
go tool pprof -top -nodecount=25 cpu.out
go tool pprof -peek='FindImportByPath|linker.Link' cpu.out
```

`-peek` is what separated "linear scan over 864" from "many short scans" and
killed the upstream-bug theory (BASELINE.md, "Negative result"). Reach for it
before drawing conclusions from a flat `-top`.

Add `-memprofile` alongside. Given the GC share, the allocation profile is
probably more actionable than the CPU one for the remaining work.

## Correctness guards — the part that actually matters

The pitfalls in PITFALLS.md are all silent-wrong-answer risks. Speed tests will
not catch any of them. Each needs a test that fails *before* the fix:

| Guard | Test |
|-------|------|
| Validators still fire (PITFALLS 1) | Corpus where a `.proto-validator` sits on a proto **not** reachable from `load()`. Assert the compile fails. This is the highest-value test in the set — it fails today's design the moment the registry goes lazy. |
| `Any` round-trip (PITFALLS 2) | Materialize a config with an `Any` payload whose type is not in any `load()`; assert inserter/agent still resolve it. |
| No silent eager fallback (PITFALLS 2) | Assert the eager path is not taken on the happy path — count parsed files and assert it stays small. Otherwise a fallback masks the regression. |
| Resolver freshness (PITFALLS 3) | Load proto A, then proto B, then look A up again through `FilesResolver`. Catches the stale-snapshot bug. |
| `mod sync` completeness (PITFALLS 4) | Run `mod sync`, assert the written `.fds` contains every proto in the module, not just demanded ones. |

## CI

`go.yml` already runs tests with Codecov. Add the benchmark as a **separate
non-blocking job** that reports numbers; gate only on the scaling-ratio assertion,
which belongs in the normal test suite as an ordinary `go test` (it is a ratio, so
it is machine-independent).

Do not gate CI on absolute milliseconds. It will flake, someone will bump the
threshold to make it green, and the gate will be worthless.
