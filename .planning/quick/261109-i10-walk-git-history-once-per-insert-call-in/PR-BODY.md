Stack: #557 → #554 → #552 → #553 → #555 → **6/6**

## Problem

`protoconf insert` takes 711-737 s on a repo with 3,050 commits and 122 configs. That is about 6 s per file. The etcd write takes about 30 ms and `git log -1` on the CLI takes about 9 ms. The numbers are 737 s on v0.2.0 and 711 s on the #555 build.

The goroutine dump from the report shows where the time goes:

`Run.func1 -> InsertConfigFile -> GatherMetadata -> commitPathIter.Next -> getNextFileCommit -> DiffTree -> merkletrie.DiffTreeContext`

`GatherMetadata` walks the whole git history once per file. The global `metaMutex` serializes those walks. The cost is files x commits.

The benchmark profile before the fix agrees. `GatherMetadata` holds 34.02 s of 55.80 s of samples (61%), almost all of it in `commitPathIter.getNextFileCommit` and `DiffTree`. After the fix, `prefetchLastCommits` holds 0.62 s of 16.55 s (3.75%), and that includes the one-off walk. Both profiles are focused on `insertConfigFiles`.

## Fix

`insertConfigFiles` now walks the history once for all requested paths. It builds a path-to-last-commit map and stops as soon as every path is found. `GatherMetadata` reads the map first. On a miss it runs the old per-file walk unchanged.

The walk diffs the same adjacent commit pairs in the same order as go-git's per-file walk, so the first hit per path is the same commit. A walk error is recorded as the same raw error for every path that is still pending.

Callers that use `InsertConfigFile` or `GatherMetadata` directly keep the per-file walk. The map lives for one `insertConfigFiles` call and is cleared before it returns. A reused inserter never sees stale commits.

## Before and after

`BenchmarkInsertManyConfigs`: 100 configs, 300 commits, `-count 5`, median of five runs.

| Metric | Before | After | Change |
| --- | --- | --- | --- |
| ns/op | 12,488,231,459 | 233,940,783 | -98.1% |
| B/op | 6,767,251,256 | 115,086,128 | -98.3% |
| allocs/op | 122,885,501 | 2,035,146 | -98.3% |

## Parallel against sequential

The per-file goroutines stay. After the fix, `BenchmarkInsertParallelism` gives these medians of three runs. The 2 ms row stands in for a remote etcd.

| Put latency | Parallel | Sequential |
| --- | --- | --- |
| 0 | 233.8 ms | 231.4 ms |
| 2 ms | 243.8 ms | 1,904.2 ms |

At 0 ms the two tie. At 2 ms parallel is about 8x faster. Parallel is not slower in either setting, so the goroutines stay.

## Equivalence

`TestPrefetchMatchesPerFileWalk` compares the map lookup with the per-file walk for every path. It checks the commit, both emails, both timestamps, the `Error()` text, and `errors.Is` for `io.EOF`, `plumbing.ErrObjectNotFound` and `plumbing.ErrReferenceNotFound`.

The fixtures cover:

- a merge of two branches
- a rename (source and target path)
- never-committed files: staged, untracked and missing
- an uncommitted edit in the worktree
- a simulated shallow clone whose cut hides some files
- an empty repo
- a non-git directory
- a 20-file, 60-commit repo

`TestGatherMetadataScenarios` pins what the old walk returns for those cases. It was written and run before the change.

The go-git quirks stay on purpose. Merge-side files get the merge commit. Files from the root commit get the last commit in iteration order. `CommittedAt` is reused for `AuthoredAt`.

## Tests and lint

- `go test -race -timeout 300s -skip 'Test_cliCommand_Run' ./...` passes. Bare `go test ./...` hangs on live Consul, etcd and ZooKeeper subtests, so it is not used.
- `go vet ./...` is clean.
- Local `golangci-lint` v2.8.0 will not run. It was built with Go 1.25 and the module targets Go 1.26. CI pins v2.14.0, so CI is the real lint check.

## Limits

I did not measure on the real 3,050-commit repo. The numbers above come from a generated repo.

I did not fix the unrelated `ensureConfigMaterialized` bug. It appends `consts.CompiledConfigPath` instead of `CompiledConfigExtension`.

🤖 Generated with [Claude Code](https://claude.com/claude-code)
