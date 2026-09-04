---
status: testing
phase: 11-concurrency-safe-lazy-registry-core
source: [11-VERIFICATION.md]
started: 2026-09-04T17:50:00Z
updated: 2026-09-04T17:50:00Z
---

## Current Test

number: 1
name: Construction against a src/ tree with zero .proto files, and with exactly one
expected: |
  NewCompiler / NewLazyModuleService succeed in both cases rather than panicking or
  erroring on an empty/near-empty tree.
awaiting: user response

## Tests

### 1. Construction against a src/ tree with zero .proto files, and with exactly one

expected: NewCompiler / NewLazyModuleService succeed in both cases rather than panicking or erroring on an empty/near-empty tree.
why_human: No committed test exercises either edge case — `utils/testdata/corpus.go`'s `GenerateCorpus` refuses `n<5`, and no other fixture with 0 or 1 proto file appears anywhere in the diffed test files. This is a structured `verification: backstop` truth in 11-01's frontmatter; the honest-verifier contract requires it to abstain rather than be inferred from the general case reading correctly.
result: [pending]

### 2. Init registers an order-independent service set across repeated runs

expected: The registered service set does not vary with `protoregistry.Files.RangeFiles`' map iteration order; only registration *order* differs between runs, never *which* services end up registered.
why_human: No test runs `Init` multiple times or forces distinct map iteration orders. `grpc.Server.GetServiceInfo()` being map-backed is suggestive but is not executed evidence, which is what the `backstop` tag requires.
result: [pending]

### 3. mod sync's .fds is immune to a concurrent in-process lazy compile

expected: Running `mod sync` concurrently with an in-process lazy `CompileFile` does not change the `.fds` bytes `mod sync` writes.
why_human: `TestModSyncFdsByteIdentical` exercises the eager mod-sync path in isolation, sequentially. No test runs a lazy compile and a mod-sync `Store()` concurrently in the same process to observe the claimed non-interference directly.
result: [pending]

### 4. WR-02 — ParseOne can return a non-canonical descriptor pointer when racing ParseAll

expected: Every caller of `ParseOne` for a given path — regardless of interleaving with a concurrent `ParseAll` — observes the one canonical `FileRegistry` pointer, matching `ParseOne`'s documented pointer-identity contract.
why_human: A genuine code-review-identified race between `ParseOne`'s singleflight closure (which parses outside `d.mu`, per its own documented lock discipline) and `ParseAll` (which holds `d.mu` across its whole-tree parse, per WR-01). `TestParseMemoization`'s concurrency subtest only races `ParseOne` against `ParseOne`, never against `ParseAll`, so it cannot catch this interleaving. **Decision needed:** fix now (return the post-`recordFileLocked` canonical value — the reviewer's suggested one-line fix) or accept as scoped-out follow-up.
result: [pending]

### 5. WR-01 — ParseAll holds d.mu across its entire whole-tree parse

expected: A decision on whether this is acceptable given D-03's "fires at most once per registry" bound, or whether it should be fixed to match `ParseOne`'s discipline before the phase is considered closed.
why_human: Not deadlock-prone (confirmed: `Import`'s `LookupImport`/`Accessor` closures never re-enter `d.mu`), but it is a real lock-hygiene inconsistency *within the same file*, with a concrete stall scenario — one goroutine's D-03 miss blocking every concurrently-compiling file on a shared `*lib.Compiler` for the whole-tree parse's duration. No must-have truth asserts anything about `ParseAll`'s lock duration, so this fails no stated truth, but it goes directly to the phase's own concurrency-safety framing.
result: [pending]

### 6. WR-04 — ModuleService.cachedRegistry is read/written without a lock

expected: A decision on whether this unenforced construction-order invariant needs a lock now (`module_service.go` already carries an unused `m.mutex` for exactly this purpose) or is acceptable as documented risk for a future direct-construction caller.
why_human: Not a live bug today (confirmed: `NewCompiler`'s single synchronous priming call happens-before every later concurrent reader), but it is precisely the class of implicit invariant this phase's own threat model (T-11-01) treats as high severity elsewhere. No must-have truth covers it.
result: [pending]

### 7. Seven plan-declared prohibitions tagged `verification: manual`

expected: Each holds under close reading, not merely under the tests that happen to pass — a proto that fails to parse on the lazy path is not silently swallowed into a successful compile; the D-03 eager fallback never fires invisibly; pre-existing fixtures/assertions were not weakened; a service `Init` cannot register does not vanish without a trace; the discovery scan does not back gRPC reflection; `mod sync` does not write a truncated `.fds`; a data race was not quieted by removing `-race` or coarsening a lock.
why_human: All seven are tagged `verification: manual` in the PLAN frontmatter — the plan authors themselves deferred these to human judgment. Independent evidence gathered during verification supports most (the D-03 fallback is logged via the `eagerFallback` field on the `compile finished` line, confirmed live; reflection registrations still read `s.parser.FilesResolver`/`LocalResolver`, confirmed by grep; `TestModSyncFdsByteIdentical` targets the truncated-`.fds` prohibition directly; no new lock or `-race`-stripping appears in the diff) — but none were exercised by a dedicated adversarial test, e.g. actually feeding a broken `.proto` through the lazy path and asserting the compile fails rather than silently materializing wrong output.
result: [pending]

## Summary

total: 7
passed: 0
issues: 0
pending: 7
skipped: 0
blocked: 0

## Gaps
