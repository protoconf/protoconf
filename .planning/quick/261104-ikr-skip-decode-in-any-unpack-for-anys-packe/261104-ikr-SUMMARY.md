---
phase: quick-261104-ikr
plan: 01
subsystem: compiler/starproto
tags: [performance, any, starlark, benchmark, not-shipped]
status: complete
commits: 2
plan_head_before: a5a7d208f190a5384392b37070eea141f87dc38b
key-files:
  modified:
    - compiler/starproto/any_test.go
decisions:
  - "Per-thread any memo NOT shipped: ns/op improved 17.8% against a 25% gate. Production files reverted."
actuals:
  tokens: 30000
  tasks: 2
  commits: 2
---

# Quick 261104-ikr: Skip decode in any.unpack - Summary

**Outcome: not shipped.** The per-thread content-keyed memo worked and was equivalent to decoding on every test, but missed the 25% ns/op gate (17.8%), so `any.go` and `message.go` were reverted to origin/main. The equivalence tests and `BenchmarkAnyRoundTrip` stay as regression and measurement tooling.

## Benchmark (BenchmarkAnyRoundTrip, 5 chain steps per iteration, count=5, Apple M4 Pro)

| Metric | Before (mean) | After memo (mean) | Delta |
|--------|---------------|-------------------|-------|
| ns/op | 1,733,728 | 1,425,397 | -17.8% |
| B/op | 726,566 | 680,332 | -6.4% |
| allocs/op | 16,515.4 | 10,579.8 | -35.9% |

Raw runs: `bench-before.txt`, `bench-after.txt` in this directory.

Gate: ns/op at least 25% lower with allocs/op not higher. allocs/op passed; ns/op failed (17.8% < 25%). Decision: revert production code.

## What was done

- Task 1 (commit 0cc87d1b): Starlark harness (`execStar`, `anyChainPrelude`), `TestAnyUnpackMemo_*` equivalence tests (hit matches decode, source mutated after new including a nested repeated element, value bytes replaced, merge into populated target for repeated/scalar and singular nested, type mismatch, Any in parent field and nested round trip), and `BenchmarkAnyRoundTrip`. All pass on unmodified production code; baseline recorded.
- Task 2 (commit d1489e31): implemented the memo (`anyMemoKey` thread local, `cloneDynamic`, error-returning `toDynamicPb`) test-first with memo hit/miss assertions. `go test -race -skip Test_cliCommand_Run ./compiler/...` was green and no package-level map existed. The benchmark missed the gate, so production files were reverted via `git checkout` and the memo-internal assertions (`anyMemoLen`) removed. The `DoubleUnpack` equivalence test was kept.

The reverted implementation is preserved (uncommitted) as a patch at `/private/tmp/claude-501/-Users-smintz-go-src-github-com-protoconf-protoconf/02422f07-0b42-4fd4-83f1-f7148589e1fb/scratchpad/261104-ikr-memo.patch` in case the gate is revisited.

## Why the win was smaller than expected

The memo removes the jhump wire decode but still pays for `ToDynamicPb` + `anypb.New` (marshal), a deep `cloneDynamic` snapshot at `any.new`, and `MergeFrom` at `any.unpack`, plus the Starlark interpretation and field mutation in the benchmark chain. About 6k allocations per 5-step chain were saved, but only about 300 microseconds of 1.7 ms.

## Deviations from Plan

None in execution. One plan inconsistency: the AnyInParentField behavior said both `out` and `out2` hit the memo, but entries are one-shot and keyed by bytes, so one Any cannot serve both. The test was split into two scripts (parent-field read-back; nested Any), each asserting a hit. Moot after revert, since memo assertions were removed.

## Known Stubs

None.

## Self-Check: PASSED

- any_test.go has TestAnyUnpackMemo_* and BenchmarkAnyRoundTrip; any.go and message.go are identical to origin/main.
- Commits 0cc87d1b and d1489e31 exist; `go test -race ./compiler/starproto/` green after revert.
