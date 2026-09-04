---
phase: "08"
plan: "01"
subsystem: server, compiler
tags: [cli, flags, libprotoconf, proto-config, env-vars]
dependency_graph:
  requires:
    - "07-01: Proto-defined CLI config definitions for ServerConfig and CompilerConfig"
  provides:
    - "server CLI with libprotoconf-generated flags from ServerConfig proto"
    - "compiler CLI with libprotoconf-generated flags from CompilerConfig proto"
    - "PROTOCONF_SERVER_* env var support for server component"
    - "PROTOCONF_COMPILER_* env var support for compiler component"
    - "--config-file flag for JSON/YAML/protobuf config loading on both components"
  affects:
    - "08-02: inserter and mutate CLI flag migration (same pattern)"
tech_stack:
  added:
    - "configtool (github.com/protoconf/libprotoconf) imported in server/server.go and compiler/command.go"
    - "protoconf_server_config package imported in server/server.go and server/server_test.go"
    - "protoconf_compiler_config package imported in compiler/command.go"
  patterns:
    - "configtool.NewConfig + SetEnvKeyPrefix + Environment + PopulateFlagSet for CLI flag generation"
    - "flag.ContinueOnError FlagSet with c.flag.Func for config-file"
    - "proto.Clone + lpc.Unmarshal + proto.Merge for config file loading"
key_files:
  modified:
    - path: "server/server.go"
      change: "Replaced cliConfig struct and newFlagSet() with protoconf_server_config.ServerConfig and libprotoconf pattern"
    - path: "compiler/command.go"
      change: "Replaced cliConfig struct and newFlagSet() with protoconf_compiler_config.CompilerConfig and libprotoconf pattern"
    - path: "server/server_test.go"
      change: "Updated cliConfig references to protoconf_server_config.ServerConfig; rewrote Test_cliCommand_Run to use Command() factory"
    - path: "compiler/command_test.go"
      change: "Fixed Test_cliCommand_Help to use Command() factory instead of bare struct (nil flag panic)"
decisions:
  - "Preserve PROTOCONF_COMPILER_ADDR legacy env var name in runScript() for backward compat with existing scripts"
  - "env var for compiler-address is now PROTOCONF_COMPILER_COMPILER_ADDRESS (libprotoconf auto-generated); PROTOCONF_COMPILER_ADDR still set for script env"
  - "proto.Merge direction matches agent pattern: file values override env vars (intentional, consistent with agent)"
metrics:
  duration_seconds: 900
  completed_date: "2026-03-29"
  tasks_completed: 2
  files_modified: 4
---

# Phase 8 Plan 01: CLI Flag Generation — Server and Compiler Summary

Migrated server/server.go and compiler/command.go from hand-written flag parsing with manual cliConfig structs to libprotoconf-generated flags driven by Phase 7 ServerConfig and CompilerConfig proto definitions, with full PROTOCONF_SERVER_* / PROTOCONF_COMPILER_* env var support and --config-file loading.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Migrate server/server.go to libprotoconf-generated flags | 058ce5b | server/server.go |
| 2 | Migrate compiler/command.go to libprotoconf-generated flags | 0f44211 | compiler/command.go, compiler/command_test.go, server/server_test.go |

## Changes Made

### Task 1: server/server.go

- Removed `type cliConfig struct` and `func newFlagSet()`
- Added imports: `configtool "github.com/protoconf/libprotoconf"` and `protoconf_server_config "github.com/protoconf/protoconf/server/config/v1"`
- Changed `cliCommand` struct to hold `config *protoconf_server_config.ServerConfig` and `flag *flag.FlagSet`
- Changed `ProtoconfMutationServer.config` from `*cliConfig` to `*protoconf_server_config.ServerConfig`
- Rewrote `Command()` using `configtool.NewConfig` with `PROTOCONF_SERVER` env prefix, `PopulateFlagSet`, and `--config-file` support
- Updated `run()` to use `c.flag.Parse(args)` and proto field accessors (`c.config.GrpcAddress`, `c.config.TlsCert`, etc.)
- Updated `runScript()`: `s.config.GrpcAddress` with comment preserving `PROTOCONF_COMPILER_ADDR` legacy name
- Updated `MutateConfig()` calls to use `s.config.AuthToken`

### Task 2: compiler/command.go

- Removed `type cliConfig struct` and `func newFlagSet()`
- Added imports: `configtool`, `protoconf_compiler_config`, `proto`
- Changed `cliCommand` struct to hold `config *protoconf_compiler_config.CompilerConfig` and `flag *flag.FlagSet`
- Rewrote `Command()` using `configtool.NewConfig` with `PROTOCONF_COMPILER` env prefix
- Removed manual `os.Getenv("PROTOCONF_COMPILER_ADDR")` — libprotoconf handles env vars automatically
- Updated `Run()`, `runRemote()`, `runLocally()` to use proto field accessors (`config.CompilerAddress`, `config.VerboseLogging`, etc.)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed server/server_test.go using deleted cliConfig type**
- **Found during:** Task 1 verification (`go vet ./server/...`)
- **Issue:** `server_test.go` referenced `&cliConfig{}` and its fields `preMutationScript`, `postMutationScript`, `authToken` which were removed
- **Fix:** Updated all `&cliConfig{...}` to `&protoconf_server_config.ServerConfig{...}` with correct proto field names; rewrote `Test_cliCommand_Run` to use `Command()` factory instead of `newFlagSet()`
- **Files modified:** `server/server_test.go`
- **Commit:** 0f44211

**2. [Rule 1 - Bug] Fixed compiler/command_test.go nil flag panic in Help() test**
- **Found during:** Task 2 test run
- **Issue:** `Test_cliCommand_Help` created `&cliCommand{}` (bare struct, nil `flag` field) and called `Help()`, causing nil pointer dereference panic
- **Fix:** Updated `Test_cliCommand_Help` to use `c, err := Command()` factory which properly initializes `flag` field
- **Files modified:** `compiler/command_test.go`
- **Commit:** 0f44211

## Known Stubs

None — all config fields are wired from proto definitions through libprotoconf to actual CLI flags.

## Self-Check: PASSED

Files exist:
- server/server.go: FOUND
- compiler/command.go: FOUND
- server/server_test.go: FOUND
- compiler/command_test.go: FOUND

Commits exist:
- 058ce5b: FOUND (feat(08-01): migrate server/server.go to libprotoconf-generated flags)
- 0f44211: FOUND (feat(08-01): migrate compiler/command.go to libprotoconf-generated flags)
