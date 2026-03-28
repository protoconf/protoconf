---
phase: 08-cli-flag-generation-config-loading
plan: "02"
subsystem: inserter, mutate, command
tags: [cli, libprotoconf, config-loading, proto-flags, dead-code-removal]
dependency_graph:
  requires:
    - "07-01: Proto-defined CLI config types (InserterConfig, MutateConfig)"
  provides:
    - "inserter CLI with libprotoconf-generated flags from InserterConfig proto"
    - "mutate CLI with libprotoconf-generated flags from MutateConfig proto"
    - "command package without KVStoreConfig dead code"
  affects:
    - "cmd/protoconf (binary still builds cleanly)"
    - "inserter tests (updated to use Command() factory)"
tech_stack:
  added: []
  patterns:
    - "configtool.NewConfig + lpc.PopulateFlagSet pattern applied to inserter and mutate"
    - "InserterConfig.StoreType enum replaces string constants for store type switching"
key_files:
  created: []
  modified:
    - inserter/inserter.go
    - inserter/inserter_test.go
    - mutate/mutate.go
    - command/command.go
decisions:
  - "Use default consul address 127.0.0.1:8500 when StoreAddress is empty (parity with etcd/zookeeper defaults)"
  - "Update inserter tests to use Command() factory — test must match new struct shape"
  - "Update inserter help test expectations to match proto-generated flag descriptions"
metrics:
  duration_seconds: 343
  completed_date: "2026-03-29"
  tasks_completed: 2
  files_modified: 4
---

# Phase 08 Plan 02: Inserter and Mutate CLI Migration Summary

Migrated inserter and mutate CLI flag parsing from manual cliConfig structs to libprotoconf-generated flags, and removed dead KVStoreConfig code from the command package.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Migrate inserter to libprotoconf flags, remove KVStore dead code | 8a1a448 | inserter/inserter.go, command/command.go |
| 2 | Migrate mutate to libprotoconf flags; fix inserter tests | 6791b63 | mutate/mutate.go, inserter/inserter.go, inserter/inserter_test.go |

## What Was Built

Both inserter and mutate CLI commands now follow the same pattern established by the agent:

**inserter/inserter.go:**
- `cliCommand` struct holds `config *protoconf_inserter_config.InserterConfig` and `flag *flag.FlagSet`
- `Command()` factory creates config, calls `configtool.NewConfig`, sets `PROTOCONF_INSERTER` env prefix, populates flag set
- Store type switching uses `InserterConfig.StoreType` enum values instead of string comparisons against `command.KVStore*` constants
- `--config-file` flag supports json/yaml/pb config file loading
- Default consul address added when `store-address` is empty (matching etcd/zookeeper behavior)

**mutate/mutate.go:**
- `cliCommand` struct holds `config *protoconf_mutate_config.MutateConfig`, `flag *flag.FlagSet`, and `ui cli.Ui`
- `fieldsArray` custom type deleted — libprotoconf handles `repeated string` natively
- `cliConfig` struct and `newFlagSet()` function deleted
- `Command()` factory sets defaults `ProtoconfRoot="./src"` and `ServerAddress="localhost:4301"`
- `PROTOCONF_MUTATE` env prefix for all config fields
- `--config-file` flag supports json/yaml/pb config file loading

**command/command.go:**
- Removed `KVStoreConsul`, `KVStoreEtcd`, `KVStoreZookeeper`, `KVStoreConfigMaps` constants
- Removed `KVStoreConfig` struct
- Removed `AddKVStoreFlags()` function
- Removed unused `"flag"` import

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed consul empty address panic**
- **Found during:** Task 2 (test run)
- **Issue:** Old code passed `[]string{kVConfig.Address}` = `[""]` to consul which handled empty string. New code passes `c.config.StoreAddress` = `[]string{}` (empty slice) which panics in consul client
- **Fix:** Added default address `127.0.0.1:8500` when `StoreAddress` empty for consul case, matching pattern of etcd/zookeeper
- **Files modified:** inserter/inserter.go
- **Commit:** 6791b63

**2. [Rule 1 - Bug] Fixed test breakage after struct shape change**
- **Found during:** Task 2 (test run)
- **Issue:** `inserter_test.go` created `&cliCommand{}` directly (no flag set) causing nil pointer dereference in `Run()` and `Help()`
- **Fix:** Updated `Test_cliCommand_Run` and `Test_cliCommand_Help` to use `Command()` factory; updated `TestCommand` to remove broken `reflect.DeepEqual` check against empty struct
- **Files modified:** inserter/inserter_test.go
- **Commit:** 6791b63

**3. [Rule 1 - Bug] Updated stale help text expectations**
- **Found during:** Task 2 (test run)
- **Issue:** `Test_cliCommand_Help` expected old hand-written help strings like `"Key-value store type (consul/zookeeper/etcd) (default \"consul\")"` that no longer match proto-generated output
- **Fix:** Updated expectations to match proto-generated flag descriptions (flag names and enum values present, old human-written descriptions removed)
- **Files modified:** inserter/inserter_test.go
- **Commit:** 6791b63

## Verification Results

- `go build ./inserter/... ./mutate/... ./command/... ./compiler/... ./cmd/...` — PASSED
- `go build ./...` — PASSED (full project)
- `go vet ./inserter/... ./mutate/... ./command/...` — PASSED
- `go test ./inserter/... ./mutate/... ./command/... -count=1` — PASSED

## Self-Check: PASSED
