---
phase: 04-dead-code-removal
plan: "01"
subsystem: inserter, agent/filekv
tags: [dead-code, cleanup, inserter, filekv]
dependency_graph:
  requires: []
  provides: [REFC-09, REFC-10]
  affects: [inserter/inserter.go, agent/filekv/filekv.go]
tech_stack:
  added: []
  patterns: []
key_files:
  created: []
  modified:
    - inserter/inserter.go
    - agent/filekv/filekv.go
decisions: []
metrics:
  duration: "3m"
  completed_date: "2026-03-27"
  tasks: 2
  files: 2
---

# Phase 04 Plan 01: Dead Code Removal — REFC-09 and REFC-10 Summary

**One-liner:** Removed dead `runtime.GOMAXPROCS` init() from inserter and unreachable error check from filekv Watch goroutine.

## What Was Built

Two targeted dead-code removals in Go source files:

1. **inserter/inserter.go**: Removed `func init()` that called `runtime.GOMAXPROCS(runtime.NumCPU())` — this was a no-op since Go runtime already uses all available CPUs by default since Go 1.5. Also removed the now-unused `"runtime"` import.

2. **agent/filekv/filekv.go**: Removed unreachable `if err != nil { return }` block that appeared after the `watchCh <- &store.KVPair{...}` channel send. The `err` variable at that point came from `proto.Marshal`, which had already been fully handled in the prior `if err != nil { slog.Error(...); return }` block. If execution reached the channel send, `err` was guaranteed nil, making the subsequent check dead code.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Remove dead init() and runtime import from inserter | 0474c6f | inserter/inserter.go |
| 2 | Remove dead error check from filekv Watch goroutine | 98b1445 | agent/filekv/filekv.go |

## Verification Results

- `grep -c 'func init()' inserter/inserter.go` returns 0
- `grep -c '"runtime"' inserter/inserter.go` returns 0
- `grep -c 'GOMAXPROCS' inserter/inserter.go` returns 0
- Dead `if err != nil { return }` removed from filekv.go Watch goroutine
- `grep -c 'func init()' agent/filekv/filekv.go` returns 1 (functional init preserved)
- `grep 'valkeyrie.Register' agent/filekv/filekv.go` matches (functional init intact)
- `go build ./...` exits 0
- `go test ./inserter/...` passes

## Deviations from Plan

None - plan executed exactly as written.

## Known Stubs

None introduced by this plan. Pre-existing stubs in filekv.go (panic("implement me") methods for Get, Delete, WatchTree, NewLock, List, DeleteTree, AtomicPut, AtomicDelete) are out of scope per PROJECT.md "Out of Scope" section: "KV store unimplemented method implementations — panics are intentional interface stubs."

## Self-Check: PASSED

- inserter/inserter.go: exists and contains no init(), runtime import, or GOMAXPROCS
- agent/filekv/filekv.go: exists and contains no dead error check after watchCh send
- Commit 0474c6f: exists
- Commit 98b1445: exists
