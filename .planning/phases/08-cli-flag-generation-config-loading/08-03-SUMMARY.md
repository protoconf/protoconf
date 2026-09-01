---
phase: 08-cli-flag-generation-config-loading
plan: 03
subsystem: cli
tags: [protobuf-reflection, config-precedence, libprotoconf, server-cli, tdd]

requires:
  - phase: 08-cli-flag-generation-config-loading
    provides: proto-generated CLI flags and -config-file loading from 08-01/08-02
provides:
  - "command.LayerConfigFile — shared, reusable precedence-layering helper for all five CLI components"
  - "PCLI-09 precedence (flags > env vars > config file > proto defaults) proven end-to-end for protoconf serve"
affects: [08-04, agent/command.go, mutate/mutate.go, inserter, compiler CLI config-file handlers]

actuals:
  tokens: 5700
  tasks: 2
  commits: 3

tech-stack:
  added: []
  patterns:
    - "proto.Message layering via protoreflect.Message.Range + a base/prev snapshot pair, instead of proto.Merge direction flips, to implement multi-source config precedence"
    - "setFieldReplacing as the single routine for repeated/map 'replace not append' semantics, called from both the file-fold and the override steps"

key-files:
  created:
    - command/configfile.go
    - command/configfile_test.go
  modified:
    - server/server.go
    - server/server_test.go

key-decisions:
  - "Superseded D-03's original proto.Merge(orig, config) file-loading mechanism with command.LayerConfigFile — the one-line merge-direction reversal was wrong in two independent ways (defaults get merged back on top of the file, and reassigning c.config orphans post-config-file flags); see the plan's why_not_the_one_line_reversal for the full analysis"
  - "base accumulates only the config-file layer (not env/flags), so a later -config-file can be told apart from an earlier one without also swallowing env values into the comparison baseline"
  - "matchesBase treats a value equal to the factory default, and a proto3 zero value, as indistinguishable from unset — documented as an accepted limitation rather than solved, since libprotoconf's Has()-based detection has no other signal available"

patterns-established:
  - "LayerConfigFile(live, base, preFile proto.Message): live is the message flag.Parse/Environment() are bound to and must never be reassigned; base is a running accumulator across all files loaded so far; preFile is captured immediately before each file's Unmarshal call. 08-04 replicates this exact three-argument shape across agent, mutate, inserter, and compiler."

requirements-completed: [PCLI-05, PCLI-06, PCLI-07, PCLI-08, PCLI-09]

coverage:
  - id: D1
    description: "PROTOCONF_SERVER_GRPC_ADDRESS overrides a grpc-address value supplied by -config-file (the PCLI-09 gap)"
    requirement: "PCLI-09"
    verification:
      - kind: unit
        ref: "server/server_test.go#Test_cliCommand_ConfigPrecedence/env_overrides_config_file"
        status: pass
    human_judgment: false
  - id: D2
    description: "An explicit -grpc-address flag wins over both env var and config file, from either argv position"
    requirement: "PCLI-09"
    verification:
      - kind: unit
        ref: "server/server_test.go#Test_cliCommand_ConfigPrecedence/flag_overrides_env_and_file_flag_last"
        status: pass
      - kind: unit
        ref: "server/server_test.go#Test_cliCommand_ConfigPrecedence/flag_overrides_env_and_file_flag_first"
        status: pass
    human_judgment: false
  - id: D3
    description: "A -config-file value overrides the factory default when no env var/flag sets that field"
    requirement: "PCLI-09"
    verification:
      - kind: unit
        ref: "server/server_test.go#Test_cliCommand_ConfigPrecedence/config_file_overrides_proto_default"
        status: pass
    human_judgment: false
  - id: D4
    description: "command.LayerConfigFile is the single implementation of flags > env vars > config file > proto defaults, and setFieldReplacing is the single implementation of repeated/map replacement, ready for 08-04 to reuse across the remaining four components"
    requirement: "PCLI-08"
    verification:
      - kind: unit
        ref: "command/configfile_test.go#TestLayerConfigFile (12 subtests, including 4 repeated-field rows)"
        status: pass
    human_judgment: false
  - id: D5
    description: "server.go no longer reassigns c.config inside the config-file handler; LayerConfigFile updates the live message in place"
    requirement: "PCLI-09"
    verification:
      - kind: other
        ref: "grep -c 'c.config, _ = orig' server/server.go == 0; grep -c 'command.LayerConfigFile(c.config, base, preFile)' server/server.go == 1"
        status: pass
    human_judgment: false

duration: 20min
completed: 2026-09-01
status: complete
---

# Phase 8 Plan 03: Env-Over-File Config Precedence for `protoconf serve` Summary

**`command.LayerConfigFile` implements flags > env vars > config file > proto defaults for the server CLI, replacing the reversed `proto.Merge(orig, c.config)` pattern that made config files beat env vars, and closes it with a test proven to fail against the pre-fix code.**

## Performance

- **Duration:** 20 min
- **Tasks:** 2
- **Files modified:** 4 (2 new, 2 modified)

## Accomplishments

- Built `command.LayerConfigFile`, a shared, reusable layering helper (`live`, `base`, `preFile` proto.Message arguments) that implements PCLI-09's stated precedence for any component's config message, not just the server's
- `setFieldReplacing` neutralizes `proto.Merge`'s repeated-field append semantics in the one place both the file-fold step and the override step call it, so a later `-config-file` replaces (not accumulates) an earlier file's list
- `matchesBase` distinguishes "explicitly supplied by env/flag" from "just came from the config file", handling scalars, enums, and lists, with documented (not silently swallowed) limitations for values that equal the factory default or a proto3 zero value
- Rewired `server/server.go`'s `-config-file` handler to call `LayerConfigFile(c.config, base, preFile)` in place, deleting the `c.config, _ = orig.(*ServerConfig)` reassignment that previously orphaned every flag parsed after `-config-file`
- `Test_cliCommand_ConfigPrecedence` (10 subtests) proves the full PCLI-09 chain end-to-end for `protoconf serve`, and was run red-first against the unmodified `server.go` to confirm it actually catches the regression
- `TestLayerConfigFile` (12 subtests) pins the helper's layering rules independent of any CLI, including four repeated-field rows using `InserterConfig.store_address` that fail without `setFieldReplacing`

## Task Commits

Each task was committed atomically (both tasks were `tdd="true"`, producing RED/GREEN cycles):

1. **Task 1: End-to-end env-over-file precedence for the server** — `e1f4c3a` (test, RED) + `789590b` (feat, GREEN)
2. **Task 2: Unit-test the layering helper** — `44fcc78` (test — all 12 rows passed against the Task 1 helper on first run, no fix needed)

**Plan metadata:** commit pending (this SUMMARY + STATE/ROADMAP/REQUIREMENTS update)

## Red-First Proof (Task 1)

Ran `go test ./server -run Test_cliCommand_ConfigPrecedence -count=1 -v` against the **unmodified** `server/server.go` (verbatim from `git show HEAD:server/server.go` before any edit in this plan). Four of ten subtests failed, exactly the ones the plan predicted:

```
--- FAIL: Test_cliCommand_ConfigPrecedence (0.00s)
    --- FAIL: Test_cliCommand_ConfigPrecedence/env_overrides_config_file (0.00s)
        Error: Not equal: expected ":9999", actual ":8888"
    --- FAIL: Test_cliCommand_ConfigPrecedence/flag_overrides_env_and_file_flag_last (0.00s)
        Error: Not equal: expected ":7777", actual ":8888"
    --- FAIL: Test_cliCommand_ConfigPrecedence/flag_overrides_env_and_file_flag_first (0.00s)
        Error: Not equal: expected ":7777", actual ":8888"
    --- PASS: Test_cliCommand_ConfigPrecedence/config_file_overrides_proto_default (0.00s)
    --- PASS: Test_cliCommand_ConfigPrecedence/empty_config_file_keeps_default (0.00s)
    --- PASS: Test_cliCommand_ConfigPrecedence/empty_env_var_is_treated_as_unset (0.00s)
    --- PASS: Test_cliCommand_ConfigPrecedence/env_and_file_agree (0.00s)
    --- FAIL: Test_cliCommand_ConfigPrecedence/later_config_file_wins (0.00s)
        Error: Not equal: expected ":7777", actual ":8888"
    --- PASS: Test_cliCommand_ConfigPrecedence/same_config_file_twice_is_idempotent (0.00s)
    --- PASS: Test_cliCommand_ConfigPrecedence/unparsable_config_file_extension_errors (0.00s)
FAIL
```

`env_overrides_config_file` failing is the PCLI-09 gap itself. `flag_overrides_env_and_file_flag_last`/`flag_overrides_env_and_file_flag_first` failing confirms the second bug identified in `<why_not_the_one_line_reversal>`: reassigning `c.config` to a clone orphans every flag parsed after `-config-file`, in either argv position. `later_config_file_wins` failing shows the reversed-merge pattern also broke "second file overrides first" for the same reason `env_overrides_config_file` fails — the file's value always wins once merged over the clone.

After implementing `command.LayerConfigFile` and rewiring `server.go`, all ten subtests pass (GREEN, verified again in this session).

## Files Created/Modified

- `command/configfile.go` — `LayerConfigFile`, `setFieldReplacing`, `matchesBase`: the shared precedence-layering implementation for PCLI-09
- `command/configfile_test.go` — `TestLayerConfigFile`, 12 subtests covering scalar and repeated-field layering rules independent of any CLI
- `server/server.go` — `Command()` now snapshots `base` before `lpc.Environment()`, and the `-config-file` closure calls `command.LayerConfigFile(c.config, base, preFile)` instead of reassigning `c.config`; the `-config-file` flag usage text now documents the precedence; the stale "outside this phase" NOTE comment is replaced with one stating the actual implemented ordering
- `server/server_test.go` — `Test_cliCommand_ConfigPrecedence` (10 subtests) and the `writeConfigJSON` test helper

## Decisions Made

- Superseded D-03's `proto.Merge(orig, config)` mechanism entirely rather than attempting the "one-line reversal" the plan explicitly warns against — the reversal is wrong in two independent, verified ways (factory defaults in `orig` would beat the file; the `c.config` reassignment orphans post-`-config-file` flags)
- `base` accumulates only the config-file layer across multiple `-config-file` flags, never env/flag values — this is what lets a second file be told apart from the first without also needing to exclude env-supplied fields from that comparison
- Documented (in code comments and in the `env_value_equal_to_default_is_indistinguishable` test) rather than attempted to fix the inherent limitation that a value equal to the factory default, or a proto3 zero value, is indistinguishable from "unset" to libprotoconf's `Has()`-based detection

## Deviations from Plan

None — plan executed exactly as written. The gofmt pre-existing import-ordering issue noted in the plan (`utils` and `grpc/status` out of sorted order in `server/server.go`) was corrected automatically by `gofmt -w` as instructed, with no additional manual reordering needed.

## Issues Encountered

None. `pre-commit`/git hooks ran normally on both commits with no failures. The `dangerouslyDisableSandbox`-free `cp`/redirect commands in this sandbox prompt for overwrite confirmation on existing files and silently no-op when declined, which required using the `Write` tool (with content read via `git show`) instead of shell `cp`/redirect to swap `server/server.go` between its pre-fix and post-fix states for the red-first proof — noted here only because it changed the mechanics of proving RED, not the result.

## Next Phase Readiness

- `command.LayerConfigFile` and `setFieldReplacing` are ready for `08-04` to apply to the remaining four components (`agent`, `mutate`, `inserter`, `compiler`) without restating the layering logic
- PCLI-09 is now fully satisfied for `protoconf serve`; `08-04` closes the same gap for the other four components to satisfy PCLI-09 project-wide
- `go test ./...` still has the pre-existing, unrelated `compiler/lib` test failure noted in `08-VERIFICATION.md` — deliberately not gated on here per the plan's scope note, unaffected by this plan's changes

---
*Phase: 08-cli-flag-generation-config-loading*
*Completed: 2026-09-01*
