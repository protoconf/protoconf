---
phase: quick-260901-vaj
plan: 01
subsystem: dependencies
tags: [go-mod, security, ci, grpc, otel, go-git, go-getter]
dependency-graph:
  requires: []
  provides:
    - "grpc v1.82.1 with HTTP2 advisory fixes"
    - "golang.org/x/net v0.56.0 with advisory fixes"
    - "go-git/v5 v5.19.2 with repo-operation advisory fixes"
    - "go-getter v1.8.6 with path-traversal/symlink advisory fixes"
    - "otel family v1.43.0 (core, sdk, sdk/metric, metric, trace, otlp exporters)"
  affects:
    - go.mod
    - go.sum
    - .github/workflows/go.yml
    - .github/workflows/codeql-analysis.yml
    - .github/workflows/release.yml
    - CLAUDE.md
tech-stack:
  added: []
  patterns:
    - "go-version-file: go.mod in CI setup-go steps (replaces hardcoded version pins)"
key-files:
  created:
    - .planning/quick/260901-vaj-merge-5-stale-security-dependency-bumps-/baseline-tests.txt
    - .planning/quick/260901-vaj-merge-5-stale-security-dependency-bumps-/after-tests.txt
    - .planning/quick/260901-vaj-merge-5-stale-security-dependency-bumps-/race-tests.txt
    - .planning/quick/260901-vaj-merge-5-stale-security-dependency-bumps-/deferred-items.md
  modified:
    - go.mod
    - go.sum
    - .github/workflows/go.yml
    - .github/workflows/codeql-analysis.yml
    - .github/workflows/release.yml
    - CLAUDE.md
decisions:
  - "x/net bumped to v0.56.0 instead of the originally targeted v0.55.0 — go-git v5.19.2 hard-requires golang.org/x/net@v0.56.0; go get refused to resolve the graph at v0.55.0. One patch above target, same or newer advisory coverage."
  - "otelgrpc auto-resolved to v0.63.0 via MVS (not the plan's speculative v0.68.0) and compiled cleanly; no manual pairing edit was needed."
  - "go.mod go directive raised to 1.25.8 (approved user decision), triggering go1.25.8 toolchain auto-download under GOTOOLCHAIN=auto."
  - "hashicorp/go-getter/v2 and pelletier/go-toml/v2 direct requires removed by go mod tidy — confirmed unused by any .go import in the tree, not a manual removal."
metrics:
  duration: "~35 minutes"
  completed: "2026-09-01"
status: complete
actuals:
  tokens: 45000
  tasks: 3
  commits: 3
---

# Phase quick-260901-vaj Plan 01: Merge 5 stale security dependency bumps Summary

Landed all five stale security-advisory dependency bumps (grpc, x/net, go-git, go-getter, and the otel family) as one coordinated `go get` + `go mod tidy`, raising the go.mod floor to 1.25.8 and pointing all three CI workflows at `go-version-file: go.mod`, with zero test regressions versus the pre-upgrade baseline.

## What Was Built

Three atomic commits on `worktree-agent-a2ec88c9889e1da9e` (forked from `origin/main` at `7fdffc4`, standing in for the plan's `deps/security-bumps` branch under this run's per-agent worktree isolation):

1. **`fe70313` — Baseline capture.** Ran `go build ./...` (clean) and `go test ./...` (all packages pass, `baseline-tests.txt`). Zero pre-existing FAIL lines.
2. **`4425172` — The five bumps.** One `go get` invocation covering grpc, x/net, go-git, go-getter, and all otel-family modules together, then `go mod tidy`, then `go build ./...` (clean on first try, no conditional source edits needed).
3. **`da51843` — Green suite + CI alignment.** Re-ran the test suite (`after-tests.txt`, identical pass set to baseline) and `go test -race` (`race-tests.txt`), updated the three CI workflow files to `go-version-file: go.mod`, and corrected the stale Go-version references in `CLAUDE.md`.

## Final Module Versions

| Module | Before | After | Target | Match |
|---|---|---|---|---|
| `google.golang.org/grpc` | v1.64.0 | v1.82.1 | v1.82.1 | exact |
| `golang.org/x/net` | v0.26.0 | v0.56.0 | v0.55.0 | +1 patch (see Deviations) |
| `github.com/go-git/go-git/v5` | v5.12.0 | v5.19.2 | v5.19.2 | exact |
| `github.com/hashicorp/go-getter` | v1.7.4 | v1.8.6 | v1.8.6 | exact |
| `go.opentelemetry.io/otel` (+ sdk, sdk/metric, metric, trace, otlptrace, otlptracegrpc, otlpmetricgrpc) | v1.27.0 | v1.43.0 | v1.43.0 | exact, all together |
| `go.opentelemetry.io/contrib/.../otelgrpc` | v0.52.0 | v0.63.0 | v0.68.0 (speculative) | MVS-resolved, compiles clean |
| `google.golang.org/protobuf` (transitive) | v1.34.1 | v1.36.11 | >= v1.36.11 | satisfies floor |
| go.mod `go` directive | 1.22.4 | 1.25.8 | >= 1.25.8 | exact (approved) |

Full transitive diff is in the `4425172` commit (`go.mod`/`go.sum`); notable transitive movement: `google.golang.org/protobuf`, `golang.org/x/*` family, `google.golang.org/genproto`, `cloud.google.com/go/*` (dragged forward by the GCP-related otel exporters), `github.com/stretchr/testify` v1.9.0 -> v1.11.1.

Two modules from the pre-bump `go.mod` were removed entirely by `go mod tidy`: `github.com/hashicorp/go-getter/v2` and `github.com/pelletier/go-toml/v2`. Confirmed via `grep -rl` that neither is imported by any `.go` file in the tree — tidy hygiene, not a manual deletion.

No conditional source-file edits were needed anywhere: `go build ./...` succeeded on the very first attempt after the `go get` + `go mod tidy`, and none of the plan's flagged risk areas (`jhump/protoreflect`, `bufbuild/protocompile`, `bufbuild/protovalidate-go`, `grpc-ecosystem/go-grpc-prometheus`, `fullstorydev/grpcui`) required a version change or code change.

## Test Results: Before vs. After

- **Baseline (`baseline-tests.txt`):** all packages `ok`, zero FAIL lines, `go version go1.25.1 darwin/arm64` (local toolchain at time of capture, pre-bump go.mod still said `go 1.22.4`).
- **After (`after-tests.txt`):** all packages `ok`, zero FAIL lines. Identical pass/fail set to baseline — **no regression**.
- **Race detector (`race-tests.txt`, new for this plan — CI runs `-race` but Task 1's baseline intentionally skipped it):** one failure, `agent/filekv` `TestWatch_ContextCancellation`, "race detected during execution of test" inside `go-git/v5`'s worktree/dotgit internals (`Worktree.copyFileToStorage` / `ObjectStorage.SetEncodedObject`), reached transitively via `utils/testdata.TestDir()`. **Confirmed pre-existing**: reproduces identically with `go-git/v5` reverted to the baseline v5.12.0. Not introduced by this bump. See Deviations below.

Honest bottom line: `go build ./...` and `go test ./...` are fully green and match baseline exactly. `go test -race ./...` has one failure that also existed before this plan touched anything — it is not a new regression, but it is not "clean" in an absolute sense either.

## Deviations from Plan

### Auto-fixed / Adjusted Issues

**1. [Rule 3 - Blocking issue] `x/net` target bumped from v0.55.0 to v0.56.0**
- **Found during:** Task 2, initial `go get` invocation
- **Issue:** `go get` refused with `github.com/go-git/go-git/v5@v5.19.2 requires golang.org/x/net@v0.56.0, not golang.org/x/net@v0.55.0` — a real MVS conflict between two of the plan's own explicit targets, not a toolchain error masking scope creep.
- **Fix:** Re-ran the same `go get` line with `golang.org/x/net@v0.56.0` instead of `@v0.55.0`. One patch release above the plan's target; carries the same or newer advisory fixes.
- **Files modified:** go.mod, go.sum
- **Commit:** 4425172

### Out-of-scope discoveries (logged, not fixed — see `deferred-items.md`)

Per deviation-rules SCOPE BOUNDARY ("Only auto-fix issues DIRECTLY caused by the current task's changes... Log out-of-scope discoveries to deferred-items.md"), two pre-existing conditions were confirmed unrelated to the bump and left untouched:

1. **`go vet ./...` findings (10 in `agent/filekv`, 1 in `compiler/lib/compiler.go`, 2 context-leak findings in test files, 1 unreachable-code finding in `agent/legacy.go`)** — confirmed byte-for-byte identical when go.mod/go.sum are reverted to the pre-bump baseline. The plan's Task 2 `<verify>` line includes a bare `go vet ./...`, which therefore exits 1 both before and after this plan — a pre-existing condition, not a scope item (scope_authority explicitly gates conditional edits on files a *compile error* or *newly failing test* names; vet warnings are neither).
2. **`TestWatch_ContextCancellation` race failure** (see Test Results above) — confirmed to reproduce at go-git v5.12.0, i.e., pre-existing, not introduced by the go-git v5.12.0 -> v5.19.2 bump.

Neither item blocks this plan's success criteria: the plan's bar is "no failing package absent from the pre-upgrade baseline," and both items were present in that baseline once actually measured (vet was never run in Task 1's baseline capture; race was intentionally skipped there per Task 1's own instructions, and was verified against the true pre-bump dependency state instead).

### Risk areas from Task 2 that did NOT fire

None of the plan's flagged risk items required action:
- `otelgrpc` needed no manual pairing — MVS resolved it to v0.63.0 and it compiled clean against otel core v1.43.0.
- `jhump/protoreflect`, `bufbuild/protocompile`, `bufbuild/protovalidate-go` all compiled unchanged against `google.golang.org/protobuf` v1.36.11.
- `grpc-ecosystem/go-grpc-prometheus` v1.2.0 compiled unchanged against grpc v1.82.1 — no halt-and-report needed.
- `fullstorydev/grpcui` / `grpcurl` needed no version change.

## Self-Check: PASSED

- FOUND: go.mod (go directive = 1.25.8, five target modules present)
- FOUND: go.sum (regenerated by `go mod tidy`)
- FOUND: .github/workflows/go.yml (`go-version-file: go.mod`)
- FOUND: .github/workflows/codeql-analysis.yml (`go-version-file: go.mod`)
- FOUND: .github/workflows/release.yml (`go-version-file: go.mod`)
- FOUND: CLAUDE.md (Go version references corrected)
- FOUND: .planning/quick/260901-vaj-merge-5-stale-security-dependency-bumps-/baseline-tests.txt
- FOUND: .planning/quick/260901-vaj-merge-5-stale-security-dependency-bumps-/after-tests.txt
- FOUND: .planning/quick/260901-vaj-merge-5-stale-security-dependency-bumps-/deferred-items.md
- FOUND: commit fe70313 (git log confirms)
- FOUND: commit 4425172 (git log confirms)
- FOUND: commit da51843 (git log confirms)

All claims verified against the working tree and git history.

## Self-Check: PASSED (re-verified post-write)
