---
phase: quick-260925-k4y
plan: 01
subsystem: cli
tags: [slog, protobuf, cli-flags, compiler]

# Dependency graph
requires: []
provides:
  - "-log-level flag / PROTOCONF_COMPILER_LOG_LEVEL env var on protoconf compile"
  - "CompilerConfig.LogLevel string field (field 7, additive, wire-compatible)"
affects: [compiler/command.go, compiler/config/v1]

# Actuals (#2632)
actuals:
  tokens: 1492
  tasks: 1
  commits: 2
plan_head_before: a5cebb5bbabebd526573f6210428401b5235528c

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "log-level flags use a plain string field parsed with stdlib slog.Level.UnmarshalText, not a copied protobuf enum, to avoid libprotoconf v0.1.0's flaggable.String() enum-by-index bug when env vars set a non-default value"

key-files:
  created: []
  modified:
    - compiler/config/v1/compiler_config.proto
    - compiler/config/v1/compiler_config.pb.go
    - compiler/command.go
    - compiler/command_test.go

key-decisions:
  - "Used a plain string field for CompilerConfig.LogLevel instead of mirroring agent/config/v1's LogLevel enum, because libprotoconf v0.1.0's flaggable.String() (flagset.go:36) looks up enum values by number as a list index; PopulateFlagSet calls String() after Environment() has applied env vars, so any non-INFO enum value set via PROTOCONF_COMPILER_LOG_LEVEL would panic at startup (reproduced with the analogous agent bug: PROTOCONF_AGENT_LOG_LEVEL=DEBUG protoconf agent -h -> index out of range [-4]). slog.Level.UnmarshalText gives the same DEBUG/INFO/WARN/ERROR names with case-insensitive parsing for free."
  - "Level is applied in cliCommand.Run directly after flag.Parse and before the NArg check, so it also covers the cpuprofile setup and the runRemote path, not just runLocally."

requirements-completed: [QUICK-260925-k4y]

coverage:
  - id: D1
    description: "protoconf compile -log-level DEBUG enables slog.Debug output (default logger otherwise drops below Info)"
    requirement: "QUICK-260925-k4y"
    verification:
      - kind: unit
        ref: "compiler/command_test.go#Test_cliCommand_LogLevel/debug_flag_enables_debug_output"
        status: pass
      - kind: unit
        ref: "compiler/command_test.go#Test_cliCommand_LogLevel/lowercase_accepted"
        status: pass
      - kind: manual_procedural
        ref: "protoconf compile -h shows -log-level (binary build + CLI check)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Invalid -log-level value returns exit code 2 with an stderr error, default (unset) behavior is unchanged"
    requirement: "QUICK-260925-k4y"
    verification:
      - kind: unit
        ref: "compiler/command_test.go#Test_cliCommand_LogLevel/invalid_level_is_usage_error"
        status: pass
      - kind: manual_procedural
        ref: "protoconf compile -log-level LOUD /nonexistent -> exit 2, 'invalid -log-level: ...'"
        status: pass
    human_judgment: false
  - id: D3
    description: "PROTOCONF_COMPILER_LOG_LEVEL=DEBUG does not panic (unlike the agent's enum-typed log-level)"
    requirement: "QUICK-260925-k4y"
    verification:
      - kind: unit
        ref: "compiler/command_test.go#Test_cliCommand_LogLevel/env_var_does_not_panic"
        status: pass
      - kind: manual_procedural
        ref: "PROTOCONF_COMPILER_LOG_LEVEL=DEBUG protoconf compile -h -> no goroutine trace"
        status: pass
    human_judgment: false
  - id: D4
    description: "CompilerConfig wire format is unchanged; log_level is additive at field number 7"
    requirement: "QUICK-260925-k4y"
    verification:
      - kind: other
        ref: "git diff compiler/config/v1/compiler_config.proto (only an appended field, no renumbering/removal)"
        status: pass
    human_judgment: false

duration: ~20min
completed: 2026-09-25
status: complete
---

# Phase quick-260925-k4y Plan 01: Add -log-level flag to protoconf compile Summary

**Added a string `log_level` field (proto field 7, additive) to `CompilerConfig`, wired into `cliCommand.Run` via `slog.Level.UnmarshalText` and `slog.SetLogLoggerLevel`, so `protoconf compile -log-level DEBUG` now surfaces the existing `slog.Debug` calls in the compile path.**

## Performance

- **Duration:** ~20 min
- **Completed:** 2026-09-25T07:41:08Z
- **Tasks:** 1 (tracer, TDD)
- **Files modified:** 4

## Accomplishments
- `protoconf compile -log-level DEBUG <root>` (and `PROTOCONF_COMPILER_LOG_LEVEL=DEBUG`) enables `slog.Debug` output; case-insensitive, unset stays at Info as before.
- Invalid `-log-level` values return exit code 2 with `invalid -log-level: <err>` on stderr instead of silently falling back.
- The env var path no longer risks the agent's `flaggable.String()` enum-by-index panic, because the new field is a plain `string`, not an enum.
- `Test_cliCommand_LogLevel` (4 subtests) locks in all four `must_haves.truths` from the plan.

## Task Commits

Each task was committed atomically (TDD RED -> GREEN, single task):

1. **Task 1 RED: add failing test for compile -log-level** - `66eb3d6a` (test)
2. **Task 1 GREEN: add -log-level flag to protoconf compile** - `c837a364` (feat)

_No plan-metadata commit: per this session's execution constraints, PLAN.md/STATE.md/SUMMARY.md are not committed here — the orchestrator commits docs._

## Files Created/Modified
- `compiler/config/v1/compiler_config.proto` - added `string log_level = 7 [json_name = "log-level"]` with a doc comment
- `compiler/config/v1/compiler_config.pb.go` - regenerated (single-file `protoc --go_out`) for the new field, `GetLogLevel()` getter, updated raw descriptor
- `compiler/command.go` - `cliCommand.Run` parses `c.config.LogLevel` via `slog.Level.UnmarshalText` and calls `slog.SetLogLoggerLevel`, placed after `flag.Parse` and before the `NArg() < 1` check; invalid values return 2
- `compiler/command_test.go` - added `Test_cliCommand_LogLevel` with 4 subtests (debug flag, lowercase, invalid value, env var no-panic)

## Decisions Made
- String field over enum for `LogLevel`, to sidestep the libprotoconf v0.1.0 `flaggable.String()` enum-index bug (see key-decisions above and the follow-up note below).
- Level application lives in `Run`, not `runLocally`, so it also covers `cpuprofile` setup errors and the `runRemote` path.

## Deviations from Plan

None - plan executed exactly as written. The regenerated `.pb.go` picked up a protoc version bump in its header comment (`protoc v7.34.0` -> `v7.36.2`, matching the `protoc` binary actually installed at `/opt/homebrew/bin/protoc`, libprotoc 36.2) — the plan explicitly called this acceptable.

## Issues Encountered
None. RED failed as predicted (`flag provided but not defined: -log-level`, exit 2); GREEN passed on first attempt for all 4 subtests, the pinned `./compiler/...` suite, `go vet`, and all manual CLI checks.

## User Setup Required
None - no external service configuration required.

## Follow-up (not fixed here, per plan's `<output>` instruction)

The agent's enum-typed `-log-level` panics when `PROTOCONF_AGENT_LOG_LEVEL` is set to a non-INFO value. Root cause: `libprotoconf v0.1.0` `flagset.go:36` uses `values.Get(int(number))` to render an enum flag's default, which indexes by ordinal position rather than by protobuf enum number; `agent/config/v1/agent_config.proto`'s `LogLevel` enum has `DEBUG = -4`, so `Get(-4)` panics with `index out of range [-4]`. The fix is `values.ByNumber(number)`. This is a `libprotoconf` upstream bug, out of scope for this task, and not something `protoconf` itself can patch without a dependency change.

## Next Phase Readiness
`compile -log-level` is shippable as-is. No blockers for other work in this repo.

---
*Phase: quick-260925-k4y*
*Completed: 2026-09-25*

## Self-Check: PASSED

All four modified files exist on disk (`compiler/config/v1/compiler_config.proto`, `compiler/config/v1/compiler_config.pb.go`, `compiler/command.go`, `compiler/command_test.go`), this SUMMARY.md exists, and both commit hashes (`66eb3d6a`, `c837a364`) are present in `git log --oneline --all`.
