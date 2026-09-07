---
phase: "11"
slug: "concurrency-safe-lazy-registry-core"
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: "2026-09-07"
---

# Phase 11 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

Register origin: `register_authored_at_plan_time: true` — all five PLAN.md files carry a
`<threat_model>` block. Mitigations were verified against the implementation; no retroactive
STRIDE scan was performed and no new threats were sought (State B, ASVS L1 grep depth).

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| `.pconf`/`.mpconf` `load()` string → on-demand proto path | On-demand parsing turns a config-authored string into a filesystem path the process opens. Under the eager registry it could only select from an already-parsed set. | Repository-controlled path string |
| `src/**.proto` → `protoparse` | Repository-local proto source parsed in-process. Pre-existing; only *when* a file crosses changed. | Proto source text |
| Concurrent compile goroutines → shared `DescriptorRegistry` | `compiler/service.go` and `compiler/command.go`'s `runLocally` run `errgroup.Go` per file against one shared `*lib.Compiler`. | In-memory descriptor maps |
| `src/**.proto` → `Init`'s discovery scan → `rpcServer.RegisterService` | A repository proto determines, at startup, which gRPC services the mutation server exposes. | Service descriptors |
| `.protoconf_cache/*.fds` on disk → any later process that `Load`s it | Written by `mod sync`, consumed by a different process/machine; its md5 validates the bytes present, not their completeness. | Serialized FileDescriptorSet |
| `protoconf.lock` on disk → `LoadFromLockFile` | Unmarshalled by `protojson` into `m.head` from a checkout, CI cache, partial write, or editor crash. | Dependency pins + integrity hashes |
| `protoconf.lock` `label`/`sourcePath` → `filepath.Join(getCacheDir(), …)` | Lock-file-controlled strings become filesystem paths `getter.GetAny` writes into. | Path components |
| `CONFIGSPACE` → `starlark.ExecFile` | Arbitrary repository Starlark executed by `mod init` / `mod tidy`. Pre-existing, unchanged. | Executable Starlark |
| `mod sync` exit code → CI/CD pipelines | Pipelines gate on exit 0; a success code after a failed generation ships an empty cache downstream. | Process exit status |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-11-01 | Denial of Service | `DescriptorRegistry.FileRegistry` under concurrent `ParseOne` | high | mitigate | `sync.RWMutex` guards every lazy-path read/write (11 lock sites, `utils/utils.go`); `singleflight.Group` (`utils.go:60`) collapses duplicate in-flight parses. Verified by `TestConcurrentCompile` under `-race`. | closed |
| T-11-02 | Tampering | `ParseOne` path handling | medium | mitigate | `ErrUnsafeProtoPath` (`utils.go:45`) returned when `!filepath.IsLocal(filepath.FromSlash(path))` (`utils.go:244`). | closed |
| T-11-03 | Denial of Service | `RegistryTypeResolver` D-03 eager fallback | medium | mitigate | `eagerFallback` latch (`utils.go:62`, checked `utils.go:355`) makes `ParseAll` one-shot per registry. | closed |
| T-11-04 | Information Disclosure | `compile finished` log line | low | accept | Emits a file count and a boolean; names only a config file already in existing compiler logs. | closed |
| T-11-05 | Denial of Service | `ParseAll` holding the write lock across a whole-tree `Import` | low | accept | Blocking is correct here; the latch bounds it to once per process. Re-affirmed as accepted risk during UAT (see Accepted Risks Log). | closed |
| T-11-06 | Denial of Service | `Init` discovery scan over a malformed `.proto` | medium | mitigate | `Import` error logged via `logger.Error` (`server/server.go:348`); startup continues. | closed |
| T-11-07 | Tampering | duplicate service registration colliding with a built-in | medium | mitigate | `protoregistry.GlobalFiles.FindFileByPath` skip preserved (`server/server.go:353`); `TestInitRegistersCustomService` asserts one entry and no panic. | closed |
| T-11-08 | Elevation of Privilege | a `src/`-discovered service gaining a mutation handler | low | accept | Same service class the snapshot loop discovered; output-type filter and `HandlerType` unchanged. `src/` is repository-controlled — same trust level as the code being run. | closed |
| T-11-09 | Information Disclosure | discovery scan reading unloaded `src/` files | low | accept | Identical to today's eager `GetProtoRegistry` import. No newly-read file. | closed |
| T-11-10 | Denial of Service | `config.messageRegistry` value copy | high | mitigate | Shared by pointer (`compiler/lib/compiler.go:363`). `go vet ./compiler/...` reports no copylocks. Verified by `TestConcurrentCompile` under `-race -count=2`. | closed |
| T-11-11 | Tampering | a truncated `.fds` written by `mod sync` | high | mitigate | `TestModSyncFdsByteIdentical` (`compiler/lib/mod_sync_fds_test.go:29`) pins serialized bytes against an independently constructed eager registry — checksum validation cannot detect this class. | closed |
| T-11-12 | Repudiation | a race appearing only on a warm second run | medium | mitigate | `TestConcurrentCompile` (`compiler/lib/concurrent_compile_test.go:25`) runs with `-count=2`. | closed |
| T-11-13 | Denial of Service | `.pconf` fixtures in `t.TempDir()` | low | accept | Per-test temp dir removed by the Go runtime; no repository file written. | closed |
| T-11-14 | Denial of Service | `Init`'s `m.head.Deps[name] = msg` nil-map write | high | mitigate | Non-nil `Deps` invariant restored on every `LoadFromLockFile` return path (`module_service.go:248-249`). Pinned by `TestModInitLockFileShapes` (`mod/command_test.go:90`), 8 rows, mutation-verified. | closed |
| T-11-15 | Tampering | `Lock()` writing over an unparseable `protoconf.lock` | high | mitigate | `Init` returns the parse error before any write (`module_service.go:118-119`); the file is left byte-for-byte intact. Covered by the `zero_byte`, `truncated_json`, `unknown_key` rows. | closed |
| T-11-16 | Repudiation | `mod init`/`mod tidy` exiting 0 having persisted nothing | high | mitigate | `MergeLock` no longer reloads before writing (`module_service.go:254-261`), so `Init`'s `CONFIGSPACE` merge survives to disk. Isolated by the `explicit_empty_deps` row. | closed |
| T-11-17 | Tampering | lock-controlled `label` reaching `filepath.Join(getCacheDir(), label)` | medium | accept | Pre-existing, out of scope. Grants no privilege the same write access does not: anyone who can write `protoconf.lock` can write `CONFIGSPACE`, which `Init` executes as arbitrary Starlark. | closed |
| T-11-18 | Information Disclosure | non-`ErrNotExist` read failure returning nil | medium | accept | `LoadFromLockFile` still treats every `os.ReadFile` error as "no lock file". Narrowing it changes behavior for all 8 callers; belongs in its own change. Carried as prohibition 3, `verification: manual`. | closed |
| T-11-19 | Tampering | `Store()` reading `FileRegistry`/`localFiles` without `d.mu` | medium | mitigate | Instance separation: eager serializing registry and lazy compiler registry are distinct objects, asserted `require.NotSame` (`compiler/lib/mod_sync_fds_test.go:129`) and pinned behaviorally by byte-identity under `-race`. | closed |
| T-11-20 | Denial of Service | test fixtures in `t.TempDir()` | low | accept | Per-test temp dir; no repository file written, no network call. | closed |
| T-11-21 | Tampering | closing `Walk` assigning `FileDescriptorSetSum` from an empty `Store` | high | mitigate | `registry.LocalFileCount() == 0` guard returns before `Store` (`module_service.go:378`), so the empty md5 is never computed or assigned. Pinned by `unsynced_dep_no_getter_url` and `downloaded_dep_bad_source_path`, both mutation-verified. | closed |
| T-11-22 | Tampering | a zero-byte `.fds` left in `.protoconf_cache` | high | mitigate | Same guard, same placement — no `Store` call means no file. Asserted directly: no `.fds` exists after a failed generation. | closed |
| T-11-23 | Repudiation | `walk()` replacing rather than accumulating its recursive error | high | mitigate | `err = errors.Join(err, walk(deps[i], walkFn))` and `return errors.Join(err, walkFn(head))` (`module_service.go:541-554`). Pinned by `downloaded_dep_bad_source_path`'s exit-code assertion. | closed |
| T-11-24 | Denial of Service | widening `Import`/`find` so a non-existent path errors | medium | accept | Accepted by NOT doing it — `server/server.go`, `utils.go`'s D-03 fallback and `module_service.go`'s eager build all depend on today's tolerance. The guard sits at `GenFileDescriptorSet`, the only site that persists a checksum. | closed |
| T-11-25 | Denial of Service | hard-failing `mod sync` where it previously exited 0 | medium | mitigate | Deliberate; warn-and-skip was rejected as the same silent-success class as T-11-16. Error output names the dependency, every searched directory, and the fixing command. `good_path_control` (`mod/command_test.go:342`) proves the good path still exits 0. | closed |
| T-11-26 | Information Disclosure | absolute cache paths in the new error output | low | accept | The same paths already appear in the preceding "Could not load from cache" output; they are what make the message actionable. | closed |
| T-11-27 | Tampering | lock-controlled `label`/`sourcePath` in `filepath.Join` | medium | accept | Carried forward unchanged from T-11-17. This plan neither widens nor narrows what those paths can address. | closed |
| T-11-28 | Denial of Service | fixtures from `testdata.SmallTestDir()` | low | accept | Copies the embedded fixture into a fresh `os.MkdirTemp`, never the repository. Not auto-removed — pre-existing behavior shared with every current `compiler/lib` test. | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-11-01 | T-11-04 | Log line emits a file count and a boolean; no proto paths or config values. | plan 11-01 | 2026-09-07 |
| AR-11-02 | T-11-05 | `ParseAll` holds the write lock across the one-shot fallback import. Blocking is correct; the latch bounds it to once per process. Re-affirmed at UAT after code review carried it forward as WR-01. | user, UAT test 5 | 2026-09-07 |
| AR-11-03 | T-11-08 | Discovered services are the same class the snapshot loop discovered; `src/` is repository-controlled content at the same trust level as the running code. | plan 11-02 | 2026-09-07 |
| AR-11-04 | T-11-09 | No file is read that the existing eager import did not already read. | plan 11-02 | 2026-09-07 |
| AR-11-05 | T-11-13 | Per-test `t.TempDir()` fixtures; no repository write. | plan 11-03 | 2026-09-07 |
| AR-11-06 | T-11-17, T-11-27 | Lock-controlled path components. Pre-existing; grants no privilege beyond what writing `CONFIGSPACE` (executed as arbitrary Starlark) already grants. Whole `mod` input surface flagged for a dedicated review. | plans 11-04, 11-05 | 2026-09-07 |
| AR-11-07 | T-11-18 | Every `os.ReadFile` error still reads as "no lock file". Narrowing to `os.IsNotExist` changes behavior for all 8 callers; carried as prohibition 3 with `verification: manual`. | plan 11-04 | 2026-09-07 |
| AR-11-08 | T-11-20 | Per-test `t.TempDir()` fixtures; no repository write, no network call. | plan 11-04 | 2026-09-07 |
| AR-11-09 | T-11-24 | Not widening `Import`/`find`'s tolerance of a non-existent path — three eager consumers depend on it; D-01 fenced off exactly that blast radius. | plan 11-05 | 2026-09-07 |
| AR-11-10 | T-11-26 | Absolute cache paths already present in adjacent output on the preceding lines. | plan 11-05 | 2026-09-07 |
| AR-11-11 | T-11-28 | `testdata.SmallTestDir()` copies into `os.MkdirTemp`, not the repository; non-removal is pre-existing. | plan 11-05 | 2026-09-07 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-09-07 | 28 | 28 | 0 | /gsd-secure-phase (State B, ASVS L1) |

### Notes from this audit

- **Scope.** State B run: register taken from the five PLAN `<threat_model>` blocks, mitigations
  verified against the implementation. No SUMMARY carried a `## Threat Flags` section. Per the
  L1 short-circuit rule (`threats_open: 0`, register authored at plan time, ASVS 1) no auditor
  subagent was spawned — grep depth is the declared sufficient depth at this level.
- **Post-UAT code change.** `LocalFileCount()`'s `d.mu.RLock()` was removed after the register was
  authored (commit `3005e5a`, UAT test 8). This does not reopen T-11-19: that threat's mitigation
  is *instance separation*, not locking, and the separation assertion (`require.NotSame`) is
  untouched. The removal makes the code's synchronisation claim match reality — `localFiles` is
  absent from `mu`'s documented guarantee and `Parse`, its only writer, takes no lock — and the
  constraint is recorded at the call site. Not a new threat: no caller reaches it concurrently.
- **Observation, out of scope, not a threat.** `walk()` (`module_service.go:541`) sorts `keys` but
  indexes the unsorted `deps` slice with the sorted index, so the walk order is nondeterministic
  despite the sort. It does not affect T-11-23's error accumulation, which is order-independent by
  construction, but it is why 11-05's accumulator mutation test has an order-dependent red signal
  (documented in `11-05-SUMMARY.md`). Recorded here so it is not lost.

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-09-07
