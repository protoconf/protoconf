---
phase: 260902-hp5
plan: 01
subsystem: infra
tags: [filekv, concurrency, data-race, fsnotify, mutex]

requires: []
provides:
  - "Lock-guarded, idempotent closeWatchers in agent/filekv/filekv.go"
  - "readEvents no longer holds w.lock across a blocking channel send"
  - "readEvents terminates on fsnotifyWatcher.Errors close instead of spinning"
  - "All ten Store methods use pointer receivers (go vet copylocks clean)"
  - "TestClose_Idempotent regression test"
affects: [agent, devserver]

actuals:
  tokens: 1366
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns: ["copy-under-lock-then-release-then-send for channel fan-out"]

key-files:
  created: []
  modified:
    - agent/filekv/filekv.go
    - agent/filekv/filekv_test.go

key-decisions:
  - "closeWatchers resets w.watches to an empty map (not nil) so addWatch/removeWatch never write to a nil map"
  - "readEvents copies the per-path watcher slice under w.lock, releases, then sends — holding the lock across the blocking send would deadlock every store operation behind one slow Watch consumer"
  - "Left a ponytail: comment on the residual send-on-closed-channel race between the copy and the send during a concurrent Close, with the documented upgrade path (shared done channel) — explicitly out of scope per plan"

patterns-established:
  - "Pattern: copy shared-slice-of-channels under the lock, release, then do blocking sends over the local copy — avoids deadlocking the lock behind a slow consumer"

requirements-completed: [QUICK-260902-hp5]

coverage:
  - id: D1
    description: "closeWatchers is lock-guarded and idempotent (empty-map reset, no double-close)"
    requirement: "QUICK-260902-hp5"
    verification:
      - kind: unit
        ref: "go test -race ./agent/filekv/... -count=3 (run 3x back to back)"
        status: pass
      - kind: unit
        ref: "agent/filekv/filekv_test.go#TestClose_Idempotent"
        status: pass
    human_judgment: false
  - id: D2
    description: "readEvents copies watcher slice under lock, sends outside lock; Errors case returns instead of spinning"
    requirement: "QUICK-260902-hp5"
    verification:
      - kind: unit
        ref: "go test -race ./agent/filekv/... -count=3 (run 3x back to back)"
        status: pass
    human_judgment: false
  - id: D3
    description: "All ten Store methods use pointer receivers; go vet copylocks clean"
    requirement: "QUICK-260902-hp5"
    verification:
      - kind: other
        ref: "go vet ./agent/filekv/..."
        status: pass
    human_judgment: false

duration: 25min
completed: 2026-09-02
status: complete
---

# Quick Task 260902-hp5: Fix filekv data race and double-close Summary

**Guarded `closeWatchers` with `w.lock`, reset its map to empty (not nil) instead of setting it nil, moved the blocking channel send in `readEvents` outside the lock via a slice copy, added a missing `return` after the `Errors` case, and switched all ten `Store` methods to pointer receivers — eliminating a race the detector caught at `filekv.go:256` vs `:261` on every `main` run.**

## Performance

- **Duration:** 25 min
- **Tasks:** 2/2 completed
- **Files touched:** 2 (agent/filekv/filekv.go, agent/filekv/filekv_test.go)

## Accomplishments

- `closeWatchers` now acquires `w.lock` for its entire body (matching `addWatch`/`removeWatch`) and resets `w.watches` to a fresh empty map instead of `nil`, making it safe to call twice from any goroutine without panicking on a subsequent write to a nil map.
- `readEvents`'s `fsnotify.Write` branch copies the watcher slice for `event.Name` under `w.lock`, releases the lock, then performs the blocking `channel <- struct{}{}` sends over the local copy — so one slow `Watch` consumer can never block every other store operation behind the mutex.
- `readEvents`'s `Errors` case now `return`s after `closeWatchers()`, matching the `Events` `!ok` branch, so a closed `fsnotifyWatcher.Errors` channel no longer spins the goroutine hot.
- All ten `Store` value receivers (`Put`, `Get`, `Delete`, `Exists`, `WatchTree`, `NewLock`, `List`, `DeleteTree`, `AtomicPut`, `AtomicDelete`) changed to pointer receivers — `go vet ./agent/filekv/...` went from 10 `copylocks` findings to 0. Method bodies (including the `panic("implement me")` stubs) are unchanged.
- Added `TestClose_Idempotent`, asserting `Close()` can be called twice with `NoError` both times.

## Deviations from Plan

None — plan executed exactly as written, including the `ponytail:` comment documenting the residual send-on-closed-channel window (out of scope per plan; upgrade path recorded inline).

## Verification (repetition gate)

- `go build ./...` — succeeded.
- `go vet ./agent/filekv/...` — exit 0 (was 10 copylocks findings before this fix).
- `go test -race ./agent/filekv/... -count=2 -timeout 10m` (Task 1 gate) — `ok` in 103.9s.
- `go test -race ./agent/filekv/... -run TestClose_Idempotent -count=1` — `PASS` in 7.4s.
- Repetition gate `for i in 1 2 3; do go test -race ./agent/filekv/... -count=3 -timeout 20m; done` — all three iterations passed (`ok` in 174.9s, 172.1s, 173.2s respectively), zero `WARNING: DATA RACE` across all 9 total package test runs.
- `go test -race ./agent/... -count=1 -skip Test_cliCommand_Run -timeout 20m` — all packages `ok` (agent 31.1s, configmaps 2.6s, dummykv 8.7s, filekv 61.2s, otelkv 4.5s).
- `git diff --stat` against base confirms exactly `agent/filekv/filekv.go` and `agent/filekv/filekv_test.go` changed — no `.pb.go`, nothing under `.claude/worktrees/`.

## Known Stubs

None introduced. The eight `panic("implement me")` stubs (`Get`, `Delete`, `WatchTree`, `NewLock`, `List`, `DeleteTree`, `AtomicPut`, `AtomicDelete`) are pre-existing and intentionally untouched per plan scope — their receivers changed from value to pointer, their bodies did not.

## Self-Check: PASSED

- FOUND: agent/filekv/filekv.go (modified, verified via git diff --stat)
- FOUND: agent/filekv/filekv_test.go (modified, verified via git diff --stat)
- FOUND commit 81ff31d (fix: closeWatchers lock/idempotent/receivers)
- FOUND commit ffda1bb (test: TestClose_Idempotent)
