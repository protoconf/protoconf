---
phase: 08-cli-flag-generation-config-loading
plan: 04
subsystem: cli
tags: [protobuf-reflection, config-precedence, libprotoconf, cli, tdd, changelog]

requires:
  - phase: 08-cli-flag-generation-config-loading
    provides: "command.LayerConfigFile, matchesBase, setFieldReplacing, and the server.go rewiring pattern from 08-03"
provides:
  - "PCLI-09 precedence (flags > env vars > config file > proto defaults) proven end-to-end for protoconf compile, insert, mutate and agent, completing it project-wide alongside 08-03's serve"
  - "Test_cliCommand_ConfigPrecedence for all four remaining components, including the inserter's repeated-field and enum coverage"
  - "CHANGELOG.md Unreleased / BREAKING CHANGES entry documenting the agent's changed config precedence"
affects: [09-*, any future CLI component adding a -config-file handler]

actuals:
  tokens: 7700
  tasks: 3
  commits: 5

tech-stack:
  added: []
  patterns:
    - "command.LayerConfigFile(c.config, base, preFile) replicated identically across compiler, inserter, mutate and agent — same three-argument shape 08-03 established for server.go"

key-files:
  created: []
  modified:
    - compiler/command.go
    - compiler/command_test.go
    - inserter/inserter.go
    - inserter/inserter_test.go
    - mutate/mutate.go
    - mutate/mutate_test.go
    - agent/command.go
    - agent/command_test.go
    - CHANGELOG.md

key-decisions:
  - "Followed the 08-03 pattern exactly rather than adapting it: base := proto.Clone(c.config) before lpc.Environment(), preFile := proto.Clone(c.config) before lpc.Unmarshal, then command.LayerConfigFile(c.config, base, preFile) in place of the old proto.Merge(orig, c.config) + c.config reassignment"
  - "agent/command.go gets an ADDED precedence comment (none existed before) rather than a replaced one, since it was the original of the reversed pattern the other four components copied — its comment additionally flags the behavior change for operators reading the source"
  - "Inserter's store enum tests use the exact PROTOCONF_INSERTER_STORE env key derived from the proto field name (not a guessed PROTOCONF_INSERTER_STORE_TYPE), verified against --help output before writing the rows"
  - "mutate/mutate.go's pre-existing gofmt import-order violation (datatypes/proto/v1 vs mutate/config/v1) was corrected as part of this task's import edit, per the plan's explicit heads-up, and noted here rather than silently folded in"
  - "Approved scope addition (per dispatch context, not a plan requirement): fixed inserter/inserter_test.go's Test_cliCommand_Help, which still asserted the pre-be1ebfc Synopsis() wording ('Insert a materialized config to the key-value store' / 'Insert a materialized config') against the current Synopsis() text introduced in be1ebfc"

requirements-completed: [PCLI-05, PCLI-06, PCLI-07, PCLI-08, PCLI-09]

coverage:
  - id: D1
    description: "PROTOCONF_COMPILER_COMPILER_ADDRESS overrides a compiler-address value supplied by -config-file"
    requirement: "PCLI-09"
    verification:
      - kind: unit
        ref: "compiler/command_test.go#Test_cliCommand_ConfigPrecedence/env_overrides_config_file"
        status: pass
    human_judgment: false
  - id: D2
    description: "PROTOCONF_INSERTER_PREFIX overrides a prefix value supplied by -config-file; explicit flags win over both from either argv position"
    requirement: "PCLI-09"
    verification:
      - kind: unit
        ref: "inserter/inserter_test.go#Test_cliCommand_ConfigPrecedence/env_overrides_config_file"
        status: pass
      - kind: unit
        ref: "inserter/inserter_test.go#Test_cliCommand_ConfigPrecedence/flag_overrides_env_and_file_flag_last"
        status: pass
      - kind: unit
        ref: "inserter/inserter_test.go#Test_cliCommand_ConfigPrecedence/flag_overrides_env_and_file_flag_first"
        status: pass
    human_judgment: false
  - id: D3
    description: "The repeated -store-address field replaces (never appends) both from an env-var list and from a second -config-file overriding an earlier one"
    requirement: "PCLI-09"
    verification:
      - kind: unit
        ref: "inserter/inserter_test.go#Test_cliCommand_ConfigPrecedence/env_list_replaces_config_file_list"
        status: pass
      - kind: unit
        ref: "inserter/inserter_test.go#Test_cliCommand_ConfigPrecedence/two_config_files_later_list_replaces_earlier"
        status: pass
    human_judgment: false
  - id: D4
    description: "The -store enum precedence path is exercised, including the documented consul=0 zero-value-enum limitation and its command-line escape hatch"
    requirement: "PCLI-09"
    verification:
      - kind: unit
        ref: "inserter/inserter_test.go#Test_cliCommand_ConfigPrecedence/env_overrides_config_file_store_enum"
        status: pass
      - kind: unit
        ref: "inserter/inserter_test.go#Test_cliCommand_ConfigPrecedence/flag_overrides_env_and_file_store_enum"
        status: pass
      - kind: unit
        ref: "inserter/inserter_test.go#Test_cliCommand_ConfigPrecedence/zero_value_enum_from_env_is_indistinguishable_from_unset"
        status: pass
      - kind: unit
        ref: "inserter/inserter_test.go#Test_cliCommand_ConfigPrecedence/flag_can_still_select_the_zero_value_enum"
        status: pass
    human_judgment: false
  - id: D5
    description: "PROTOCONF_MUTATE_SERVER_ADDRESS overrides an addr value from -config-file; a config file still overrides the localhost:4301 factory default"
    requirement: "PCLI-09"
    verification:
      - kind: unit
        ref: "mutate/mutate_test.go#Test_cliCommand_ConfigPrecedence/env_overrides_config_file"
        status: pass
      - kind: unit
        ref: "mutate/mutate_test.go#Test_cliCommand_ConfigPrecedence/config_file_overrides_proto_default"
        status: pass
    human_judgment: false
  - id: D6
    description: "PROTOCONF_AGENT_GRPC_ADDRESS overrides a grpc-address value from -config-file (the deliberate agent behavior change); a config file still overrides the :4300 factory default"
    requirement: "PCLI-09"
    verification:
      - kind: unit
        ref: "agent/command_test.go#Test_cliCommand_ConfigPrecedence/env_overrides_config_file"
        status: pass
      - kind: unit
        ref: "agent/command_test.go#Test_cliCommand_ConfigPrecedence/config_file_overrides_proto_default"
        status: pass
    human_judgment: false
  - id: D7
    description: "The agent's changed precedence is discoverable by an operator from CHANGELOG.md and from --help, without reading the source"
    requirement: "PCLI-09"
    verification:
      - kind: other
        ref: "grep -n 'Unreleased' CHANGELOG.md && grep -q 'flags > env vars > config file > proto defaults' CHANGELOG.md && grep -q 'PCLI-09' CHANGELOG.md"
        status: pass
      - kind: other
        ref: "go run cmd/protoconf/main.go agent --help | grep -A2 config-file"
        status: pass
    human_judgment: false
  - id: D8
    description: "No .go file in the repository still claims the config-file merge deviation from PCLI-09 is intentional and deferred; all five component files plus command/configfile.go carry the implemented-precedence comment"
    requirement: "PCLI-09"
    verification:
      - kind: other
        ref: "grep -rq 'outside this phase' --include='*.go' . (exit 1, confirming absence)"
        status: pass
      - kind: other
        ref: "grep -l 'flags > env vars > config file > proto defaults' server/server.go compiler/command.go inserter/inserter.go mutate/mutate.go agent/command.go command/configfile.go | wc -l == 6"
        status: pass
    human_judgment: false

duration: 25min
completed: 2026-09-01
status: complete
---

# Phase 8 Plan 04: PCLI-09 Config Precedence for compiler, inserter, mutate and agent Summary

**Replicated 08-03's `command.LayerConfigFile` rewiring across the four remaining CLI components — closing PCLI-09 project-wide, including the inserter's repeated-field and `store` enum precedence paths, and recording the agent's changed config-vs-env ordering in `CHANGELOG.md`.**

## Performance

- **Duration:** ~25 min
- **Tasks:** 3
- **Files modified:** 9 (8 Go files across 4 packages, 1 CHANGELOG.md)

## Accomplishments

- All four remaining `-config-file` handlers (`compiler/command.go`, `inserter/inserter.go`, `mutate/mutate.go`, `agent/command.go`) now call `command.LayerConfigFile(c.config, base, preFile)` in place of the old `proto.Merge(orig, c.config)` + `c.config` reassignment pattern, closing the same two bugs 08-03 fixed for `server.go`: config files no longer beat env vars, and flags parsed after `-config-file` are no longer orphaned
- `Test_cliCommand_ConfigPrecedence` added to all four packages (30 subtests total: 6 compiler + 12 inserter + 6 mutate + 6 agent), every one run red-first against the pre-fix `proto.Merge(orig, c.config)` handlers and confirmed to fail exactly where PCLI-09 predicts
- Inserter coverage extends beyond scalars: two repeated-field rows prove `-store-address` replaces rather than appends (both from an env-var list and from a second `-config-file`), and four `store` enum rows exercise the enum arm of `matchesBase`, including the documented `consul=0` zero-value-enum limitation and its command-line escape hatch — none of it required a `command/configfile.go` change, confirming the 08-03 helper already handles enums correctly
- `agent/command.go`'s precedence comment is newly ADDED (not replaced — none existed before), since the agent was the original pattern the other four components copied; the comment explicitly flags this as a behavior change from the agent's history
- `CHANGELOG.md` gained an `## Unreleased` / `### ⚠ BREAKING CHANGES` section naming `protoconf agent`, `PCLI-09`, and the full precedence chain, giving an upgrading operator a discoverable, non-source record of the change
- All five components' `--config-file` usage strings now state their env-var prefix and that flags/env vars override the file, visible in `--help`
- Repo-wide gates pass: `go build ./...`, exactly 5 top-level `Test_cliCommand_ConfigPrecedence` PASS lines plus `TestLayerConfigFile`, zero `.go` files still claiming the deviation is "outside this phase", and exactly 6 files carrying the `flags > env vars > config file > proto defaults` comment

## Task Commits

Each task was committed atomically; both Task 1 and Task 2 were `tdd="true"`, producing RED/GREEN cycles:

1. **Task 1: Compiler and inserter precedence + tests** — `3e0ed17` (test, RED) + `1d9e630` (feat, GREEN)
2. **Task 2: Mutate and agent precedence + tests, including the agent behavior change** — `67fee1c` (test, RED) + `0492b8f` (feat, GREEN)
3. **Task 3: CHANGELOG.md and repo-wide gate** — `e620139` (docs)

**Plan metadata:** commit pending (this SUMMARY + STATE/ROADMAP/REQUIREMENTS update)

## Red-First Proof

### Task 1 (compiler, inserter)

Ran `go test ./compiler ./inserter -run Test_cliCommand_ConfigPrecedence -count=1 -v` against the unmodified `compiler/command.go` and `inserter/inserter.go` (restored via `git checkout --` immediately before the run, then restored back to the fixed version afterward). Failures matched the plan's prediction exactly:

- **Compiler:** `env_overrides_config_file`, `flag_overrides_env_and_file_flag_last`, `flag_overrides_env_and_file_flag_first` failed (all produced `file:8888` instead of the expected env/flag value); `config_file_value_is_applied`, `empty_config_file_leaves_field_empty`, `empty_env_var_is_treated_as_unset` already passed.
- **Inserter scalar (`prefix`):** same three rows failed for the same reason.
- **Inserter repeated field:** `env_list_replaces_config_file_list` produced `["env1:1","env2:2","file1:1"]` instead of `["env1:1","env2:2"]` — the file's entry was appended, not replaced. `two_config_files_later_list_replaces_earlier` produced `["file1:1","file1:1"]` instead of `["file2:2"]` — this is the append-hazard fact 4 from 08-03's `why_not_the_one_line_reversal` made concrete at the CLI level: the pre-fix `proto.Merge(orig, c.config)` neither replaced the list nor let the second file win, it duplicated the first file's entry.
- **Inserter enum:** `env_overrides_config_file_store_enum` got `zookeeper` (2) instead of `etcd` (1); `flag_overrides_env_and_file_store_enum` got `zookeeper` (2) instead of `configmaps` (3); `flag_can_still_select_the_zero_value_enum` got `zookeeper` (2) instead of `consul` (0) — even an explicit flag was orphaned by the `c.config` reassignment bug. `zero_value_enum_from_env_is_indistinguishable_from_unset` already passed (it asserts the documented limitation, which existed under both the old and new code).

After restoring `command.LayerConfigFile(c.config, base, preFile)`, all 18 subtests (6 compiler + 12 inserter) passed.

### Task 2 (mutate, agent)

Same restore-run-restore procedure for `mutate/mutate.go` and `agent/command.go`:

- **Mutate:** `env_overrides_config_file`, `flag_overrides_env_and_file_flag_last`, `flag_overrides_env_and_file_flag_first` failed (all produced `file:8888`); `config_file_overrides_proto_default` and `empty_config_file_keeps_default` already passed — confirming the fix does not regress the "file overrides factory default" half of PCLI-09.
- **Agent:** identical shape — `env_overrides_config_file` got `:8888` instead of `:9999`; both flag-order rows got `:8888` instead of `:7777`; `config_file_overrides_proto_default` and `empty_config_file_keeps_default` already passed both before and after the fix, as the plan requires.

After restoring both handlers, all 12 subtests (6 mutate + 6 agent) passed.

## Files Created/Modified

- `compiler/command.go` — `Command()` now snapshots `base` before `lpc.Environment()`; the `-config-file` closure calls `command.LayerConfigFile(c.config, base, preFile)`; usage string documents the ordering; stale NOTE replaced
- `compiler/command_test.go` — `Test_cliCommand_ConfigPrecedence` (6 subtests) plus `writeConfigJSON` helper
- `inserter/inserter.go` — same five-part rewiring as compiler
- `inserter/inserter_test.go` — `Test_cliCommand_ConfigPrecedence` (12 subtests: 6 scalar `prefix`, 2 `store_address` list, 4 `store` enum), `writeConfigJSON` and `writeStoreAddressJSON` helpers; also fixes `Test_cliCommand_Help`'s stale Synopsis() string assertions (see Deviations)
- `mutate/mutate.go` — same five-part rewiring; also fixes the pre-existing gofmt import-order violation (`pv "datatypes/proto/v1"` moved before `protoconf_mutate_config "mutate/config/v1"`)
- `mutate/mutate_test.go` — `Test_cliCommand_ConfigPrecedence` (6 subtests) plus `writeConfigJSON` helper
- `agent/command.go` — same rewiring, but the precedence comment is newly ADDED (none existed before) and explicitly flags the behavior change
- `agent/command_test.go` — `Test_cliCommand_ConfigPrecedence` (6 subtests) written in the file's existing plain-`testing` style (no testify import added), using a distinctly-named `writeAgentConfigJSON` helper to avoid colliding with the file's shared `os.TempDir()` `jsonConfig` fixture convention; never calls `cc.Run`
- `CHANGELOG.md` — new `## Unreleased` / `### ⚠ BREAKING CHANGES` section

## Decisions Made

- Replicated the 08-03 pattern byte-for-byte in shape (`base`/`preFile`/`command.LayerConfigFile` call sites) across all four components rather than adapting per-component — the plan's `<component_reference>` table already verified every field name, env key, flag name and factory default, so there was no ambiguity to resolve locally
- `agent/command.go` gets an ADDED comment rather than a replaced one, since it carried no precedent NOTE — this preserves the historical-record framing the plan asked for ("the agent's changed ordering is stated in its own source comment")
- Fixed `mutate/mutate.go`'s pre-existing gofmt violation as part of the import-block edit the task already required, exactly as the plan's Task 2 action flagged in advance
- Approved deviation (per dispatch context, `<pre_existing_test_failures>`): updated `inserter/inserter_test.go`'s `Test_cliCommand_Help` stale string assertions to match the current `Synopsis()` text from commit `be1ebfc`. `Synopsis()` itself was not touched.

## Deviations from Plan

### Auto-fixed Issues

**1. [Approved scope addition — pre-existing test regression, not caused by this plan] Fixed stale `Test_cliCommand_Help` string assertions in `inserter/inserter_test.go`**
- **Found during:** dispatch context (`<pre_existing_test_failures>`), confirmed before Task 1
- **Issue:** commit `be1ebfc` changed `inserter/inserter.go`'s `Synopsis()` to "Insert materialized configs into a key-value store (Consul, etcd, ZooKeeper, or ConfigMaps)" but never updated the test, which still asserted the old "Insert a materialized config to the key-value store" / "Insert a materialized config" strings
- **Fix:** updated the two `want` string literals in `Test_cliCommand_Help` to match the current `Synopsis()` wording; `Synopsis()` itself unchanged
- **Files modified:** `inserter/inserter_test.go`
- **Verification:** `go test ./inserter -run Test_cliCommand_Help -count=1 -v` — both subtests PASS
- **Committed in:** `3e0ed17` (Task 1 RED commit, since the fix lives in the test file touched by that commit)

---

**Total deviations:** 1 approved scope addition (explicitly pre-authorized by the dispatch context, not a Rule 1-4 auto-fix of code this plan wrote).
**Impact on plan:** Restores a passing baseline for `go test ./inserter`; no functional code (`Synopsis()`) was touched.

## Issues Encountered

- The sandboxed `cp` command prompts for overwrite confirmation and silently no-ops when declined when swapping files between pre-fix and post-fix states for the red-first proof (same issue 08-03's SUMMARY documented) — worked around identically, using the `Write` tool with content captured via `Read` instead of shell `cp`
- `.planning/REQUIREMENTS.md` carries the pre-existing, uncommitted `TEST-07`..`TEST-13` checkbox edits flagged in this session's `<sequential_execution>` instructions as not mine to touch. Confirmed via `git log -1 -- .planning/REQUIREMENTS.md` that the file's last commit (`6e6d09b`) was 08-03's revert of its own accidental inclusion, and that this session never staged or committed `.planning/REQUIREMENTS.md`. As a direct consequence, the plan's Task 3 `<verify>` line `git diff --exit-code README.md mkdocs.yml go.mod go.sum .planning/REQUIREMENTS.md 08-01-PLAN.md 08-02-PLAN.md` reports a non-zero exit solely because of that pre-existing `REQUIREMENTS.md` diff — verified individually that `README.md`, `mkdocs.yml`, `go.mod`, `go.sum`, `08-01-PLAN.md` and `08-02-PLAN.md` are each untouched (`git diff --exit-code` on each passes independently). No `requirements.mark-complete` call in this session's `update_requirements` step will touch `TEST-*` IDs — only `PCLI-05`..`PCLI-09` are in scope, per this plan's frontmatter.

## Next Phase Readiness

- PCLI-09 is now satisfied for all five CLI components (`serve` from 08-03, `compile`/`insert`/`mutate`/`agent` from this plan)
- PCLI-05 through PCLI-08 remain satisfied — no flag name, env prefix, `-config-file` support, or generated-flag mechanism changed
- `go test ./...` still has the two pre-existing, unrelated failures noted in the dispatch context (`compiler/lib`'s `TestCompiler_CompileFile/load_remote_with_load_local.pconf`, missing `.protoconf_cache` fixture; and `test`'s e2e `Test`), both untouched by and unrelated to this plan's changes
- Phase 8 (CLI Flag Generation & Config Loading) is now complete: all four plans (08-01, 08-02, 08-03, 08-04) have SUMMARYs

---
*Phase: 08-cli-flag-generation-config-loading*
*Completed: 2026-09-01*

## Self-Check: PASSED

- `compiler/command.go` — FOUND, contains `command.LayerConfigFile(c.config, base, preFile)`
- `compiler/command_test.go` — FOUND, contains `Test_cliCommand_ConfigPrecedence`
- `inserter/inserter.go` — FOUND, contains `command.LayerConfigFile(c.config, base, preFile)`
- `inserter/inserter_test.go` — FOUND, contains `Test_cliCommand_ConfigPrecedence` and `writeStoreAddressJSON`
- `mutate/mutate.go` — FOUND, contains `command.LayerConfigFile(c.config, base, preFile)`, gofmt-clean
- `mutate/mutate_test.go` — FOUND, contains `Test_cliCommand_ConfigPrecedence`
- `agent/command.go` — FOUND, contains `command.LayerConfigFile(c.config, base, preFile)`
- `agent/command_test.go` — FOUND, contains `Test_cliCommand_ConfigPrecedence`, no testify import added
- `CHANGELOG.md` — FOUND, contains `## Unreleased`, `PCLI-09`, `flags > env vars > config file > proto defaults`
- Commit `3e0ed17` (test: compiler+inserter RED) — FOUND
- Commit `1d9e630` (feat: compiler+inserter GREEN) — FOUND
- Commit `67fee1c` (test: mutate+agent RED) — FOUND
- Commit `0492b8f` (feat: mutate+agent GREEN) — FOUND
- Commit `e620139` (docs: CHANGELOG) — FOUND
- `go build ./...` — exits 0
- `go test ./command ./server ./agent ./compiler ./inserter ./mutate -run 'TestLayerConfigFile|Test_cliCommand_ConfigPrecedence' -count=1 -v` — all PASS, 5 top-level `Test_cliCommand_ConfigPrecedence` PASS lines
- `grep -rq 'outside this phase' --include='*.go' .` — exit 1 (absent, as required)
- 6 files carry the precedence comment (server, compiler, inserter, mutate, agent, command/configfile.go)
- `git diff --exit-code README.md mkdocs.yml go.mod go.sum` — no changes
- `git diff --exit-code .planning/phases/08-cli-flag-generation-config-loading/08-01-PLAN.md .planning/phases/08-cli-flag-generation-config-loading/08-02-PLAN.md` — no changes
