---
phase: "07"
plan: "01"
subsystem: proto-defined-cli-configs
tags: [proto, config, server, compiler, inserter, mutate]
dependency_graph:
  requires: []
  provides:
    - server/config/v1.ServerConfig
    - compiler/config/v1.CompilerConfig
    - inserter/config/v1.InserterConfig
    - mutate/config/v1.MutateConfig
  affects:
    - Phase 8 CLI flag generation (PopulateFlagSet wiring)
tech_stack:
  added:
    - "server/config/v1/server_config.proto (new package)"
    - "compiler/config/v1/compiler_config.proto (new package)"
    - "inserter/config/v1/inserter_config.proto (new package)"
    - "mutate/config/v1/mutate_config.proto (new package)"
  patterns:
    - "proto3 message definitions with json_name options matching CLI flag names"
    - "Nested enum for StoreType within InserterConfig"
    - "repeated string fields for multi-value CLI flags"
key_files:
  created:
    - server/config/v1/server_config.proto
    - server/config/v1/server_config.pb.go
    - compiler/config/v1/compiler_config.proto
    - compiler/config/v1/compiler_config.pb.go
    - inserter/config/v1/inserter_config.proto
    - inserter/config/v1/inserter_config.pb.go
    - mutate/config/v1/mutate_config.proto
    - mutate/config/v1/mutate_config.pb.go
  modified: []
decisions:
  - "TLS fields are flat strings in ServerConfig (not nested TLSConfig) to exactly match existing server CLI flags for Phase 8 compatibility"
  - "InserterConfig.StoreType enum excludes 'file' (agent-only dev mode) but includes configmaps; each component defines its own enum per D-10"
  - "InserterConfig.store_address is repeated string despite current CLI being single string, per D-07 for future multi-address support"
  - "compiler_address field added to CompilerConfig despite being absent from D-06 spec — discovered in compiler/command.go line 54"
metrics:
  duration: "164s"
  completed_date: "2026-03-28"
  tasks_completed: 2
  files_created: 8
---

# Phase 7 Plan 1: Proto-Defined CLI Configs Summary

**One-liner:** Proto3 message definitions for server, compiler, inserter, and mutate CLI configs with json_name options matching existing CLI flag names, including StoreType enum and repeated string fields.

## What Was Built

Four new proto packages extending the agent's existing proto-defined config pattern to all other protoconf components:

- **ServerConfig** (`server/config/v1`): 7 fields covering grpc-address, pre/post mutation scripts, TLS cert/key/CA, and auth-token
- **CompilerConfig** (`compiler/config/v1`): 6 fields covering repl, verbose-logging, process-templates, cpuprofile, memprofile, and compiler-address
- **InserterConfig** (`inserter/config/v1`): 5 fields with a nested StoreType enum (consul/etcd/zookeeper/configmaps), repeated store_address, prefix, namespace, and delete flag
- **MutateConfig** (`mutate/config/v1`): 11 fields covering protoconf root, proto file/message, server address, config path, metadata, repeated fields array, TLS cert/key/CA, and insecure flag

All four protos were compiled with `protoc` (using source_relative paths) to generate `.pb.go` files following the same pattern as `agent/config/v1/agent_config.pb.go`. The full project (`go build ./...`) builds without errors.

## Tasks Completed

| Task | Description | Commit |
|------|-------------|--------|
| 1 | Create proto config definitions for all four components | 2b4b006 |
| 2 | Generate .pb.go files and verify full project compilation | 3331e89 |

## Decisions Made

1. **Flat TLS strings in ServerConfig** — The server's existing flags (`tls-cert`, `tls-key`, `tls-ca`) are simple file path strings. Using a flat structure exactly mirrors the current `cliConfig` struct, making Phase 8 wiring straightforward without requiring nested message handling.

2. **InserterConfig owns its own StoreType enum** — Per D-10, the inserter does not import the agent's enum. The inserter's StoreType omits `file` (which is agent-specific for dev mode) and includes `configmaps`. Each component is self-contained.

3. **repeated string store_address** — The current inserter uses a single `string` for store-address, but D-07 explicitly requires `repeated string` for future multi-address support. Phase 8 will need to handle the string-to-repeated conversion.

4. **compiler_address field included** — The D-06 spec omitted this field but it exists as a real CLI flag in `compiler/command.go` line 54. Adding it ensures the CompilerConfig message is complete and Phase 8 can wire all flags.

## Deviations from Plan

None — plan executed exactly as written. The `buf.yaml` update was not needed since the existing `version: v1` with default roots already covers all new proto directories.

Note: `go vet ./...` reports pre-existing issues in `agent/filekv`, `compiler/lib`, `test/`, and `agent/` packages. These are out-of-scope and were present before this plan. All new packages (`server/config/v1`, `compiler/config/v1`, `inserter/config/v1`, `mutate/config/v1`) pass `go vet` cleanly.

## Known Stubs

None — this plan creates pure proto schema definitions with no runtime stubs or placeholder data.

## Self-Check: PASSED

Files created:
- server/config/v1/server_config.proto: FOUND
- server/config/v1/server_config.pb.go: FOUND
- compiler/config/v1/compiler_config.proto: FOUND
- compiler/config/v1/compiler_config.pb.go: FOUND
- inserter/config/v1/inserter_config.proto: FOUND
- inserter/config/v1/inserter_config.pb.go: FOUND
- mutate/config/v1/mutate_config.proto: FOUND
- mutate/config/v1/mutate_config.pb.go: FOUND

Commits:
- 2b4b006: feat(07-01): add proto config definitions for all four components
- 3331e89: feat(07-01): generate .pb.go files for all four component config protos
