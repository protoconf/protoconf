---
phase: 08-cli-flag-generation-config-loading
plan: 05
subsystem: cli
tags: [protobuf-reflection, config-precedence, provenance-tracking, libprotoconf, cli, tdd, security]

requires:
  - phase: 08-cli-flag-generation-config-loading
    provides: "command.LayerConfigFile, matchesBase, setFieldReplacing from 08-03/08-04, and the two reproduced gaps recorded in 08-VERIFICATION.md / 08-REVIEW.md CR-01"
provides:
  - "command.ConfigLayerer / NewConfigLayerer / (*ConfigLayerer).LayerConfigFile — a provenance-tracking replacement for the value-comparison layering model, closing both VERIFICATION.md BLOCKER gaps (#7 env coincidence, #8 message-typed later-file-wins)"
  - "agent/command.go rewired onto command.NewConfigLayerer, proving the fix end-to-end for protoconf agent before 08-06 generalizes it"
  - "setFieldReplacing's message/group arm now deep-copies via proto.Clone instead of aliasing the source submessage, closing 08-REVIEW.md IN-01"
  - "Test_cliCommand_MultiConfigFilePrecedence (CLI level) and nine new TestLayerConfigFile rows (unit level) pinning both gaps, recursive-merge semantics, flag provenance, and the IN-01 deep copy"
affects: [08-06, any future CLI component adding a repeated -config-file handler]

actuals:
  tokens: 9464
  tasks: 2
  commits: 3

tech-stack:
  added: []
  patterns:
    - "ConfigLayerer owns three separate messages (defaults, fileLayer, lastResult) plus an explicit-field-number set, replacing the single mutating base accumulator that could not distinguish 'explicitly supplied' from 'carried over from an earlier file'"
    - "Provenance recorded from two independent sources per LayerConfigFile call: flag.FlagSet.Visit (exact, ordering-based) for flags parsed before this -config-file, and a field-level diff against lastResult (value-based, but against a non-accumulating baseline) for env vars and newly-parsed flags"
    - "fieldEqual delegates every field kind, including message-typed, to proto.Equal via two single-field scratch messages, replacing matchesBase's kind-specific short-circuits"

key-files:
  created: []
  modified:
    - command/configfile.go
    - command/configfile_test.go
    - agent/command.go
    - agent/command_test.go

key-decisions:
  - "Retained the old LayerConfigFile free function and matchesBase in command/configfile.go rather than deleting them: compiler, server, inserter, and mutate still call the free function, and 08-06 is the plan responsible for migrating them and removing it"
  - "Added a comment above matchesBase recording exactly why 08-REVIEW.md CR-01's suggested patch (proto.Equal in the composite arm) would only have fixed the message-field gap while leaving the env-coincidence gap open — the baseline it compares against still accumulates config-file values"
  - "markExplicitFlags matches flag names against top-level JSONName only, per library fact 4: PopulateFlagSet routes MessageKind fields' flags (e.g. tls-config-cert-file) onto a detached dynamicpb message with no corresponding top-level field number, so those flag names are correctly ignored rather than mis-mapped"
  - "Provenance entries are never removed once set (idempotent across repeated -config-file loads and across markExplicitFlags calls), matching the plan's explicit requirement that provenance persists across every subsequent -config-file"

requirements-completed: [PCLI-05, PCLI-06, PCLI-07, PCLI-08, PCLI-09]

coverage:
  - id: D1
    description: "A later -config-file overrides an earlier one for message-typed fields (AgentConfig.tls_config.cert_file and AgentConfig.store_tls.ca_file) instead of the first file winning (VERIFICATION.md gap #8)"
    requirement: "PCLI-09"
    verification:
      - kind: unit
        ref: "agent/command_test.go#Test_cliCommand_MultiConfigFilePrecedence/later_file_wins_for_tls_cert_file"
        status: pass
      - kind: unit
        ref: "agent/command_test.go#Test_cliCommand_MultiConfigFilePrecedence/later_file_wins_for_store_tls_ca_file"
        status: pass
      - kind: unit
        ref: "command/configfile_test.go#TestLayerConfigFile/later_file_wins_for_message_field"
        status: pass
      - kind: unit
        ref: "command/configfile_test.go#TestLayerConfigFile/later_file_wins_for_store_tls_message_field"
        status: pass
    human_judgment: false
  - id: D2
    description: "An env var whose value coincidentally equals an earlier config file's value still wins over a later config file (VERIFICATION.md gap #7)"
    requirement: "PCLI-09"
    verification:
      - kind: unit
        ref: "agent/command_test.go#Test_cliCommand_MultiConfigFilePrecedence/env_wins_over_two_files_when_first_file_coincides"
        status: pass
      - kind: unit
        ref: "command/configfile_test.go#TestLayerConfigFile/env_wins_over_two_files_on_value_coincidence"
        status: pass
      - kind: unit
        ref: "command/configfile_test.go#TestLayerConfigFile/env_wins_over_two_files_for_message_field"
        status: pass
    human_judgment: false
  - id: D3
    description: "An explicit flag parsed before -config-file wins even when its value equals the compiled-in factory default, via flag.FlagSet.Visit provenance rather than value comparison"
    requirement: "PCLI-09"
    verification:
      - kind: unit
        ref: "agent/command_test.go#Test_cliCommand_MultiConfigFilePrecedence/flag_before_config_file_wins_when_flag_equals_factory_default"
        status: pass
      - kind: unit
        ref: "command/configfile_test.go#TestLayerConfigFile/flag_provenance_beats_file_when_flag_equals_factory_default"
        status: pass
      - kind: unit
        ref: "command/configfile_test.go#TestLayerConfigFile/unknown_flag_name_is_ignored"
        status: pass
    human_judgment: false
  - id: D4
    description: "Pre-existing later-file-wins and env-wins guarantees for scalar/list fields are preserved unbroken by the provenance rewrite, and setFieldReplacing's message arm deep-copies instead of aliasing (IN-01)"
    requirement: "PCLI-09"
    verification:
      - kind: unit
        ref: "agent/command_test.go#Test_cliCommand_ConfigPrecedence (all 6 subtests)"
        status: pass
      - kind: unit
        ref: "command/configfile_test.go#TestLayerConfigFile (all 12 original subtests)"
        status: pass
      - kind: unit
        ref: "command/configfile_test.go#TestLayerConfigFile/set_field_replacing_deep_copies_message_values"
        status: pass
      - kind: integration
        ref: "go test ./agent ./command ./server ./compiler ./inserter ./mutate -count=1"
        status: pass
    human_judgment: false

duration: 30min
completed: 2026-09-01
status: complete
---

# Phase 8 Plan 05: Provenance-tracking ConfigLayerer for the agent Summary

Replaced `command.LayerConfigFile`'s value-comparison provenance guess with a new `command.ConfigLayerer` that records field-level provenance from `flag.FlagSet.Visit` and env-supplied differences instead of inferring it from an accumulating baseline, closing both `08-VERIFICATION.md` BLOCKER gaps for `protoconf agent` and proving the fix end-to-end before 08-06 generalizes it to the other four components.

## Performance

- **Duration:** ~30 min
- **Tasks:** 2 completed
- **Files modified:** 4 (`command/configfile.go`, `command/configfile_test.go`, `agent/command.go`, `agent/command_test.go`)

## Accomplishments

- Closed VERIFICATION.md gap #8: a later `-config-file` now correctly overrides an earlier one for message-typed fields (`AgentConfig.tls_config.cert_file`, `AgentConfig.store_tls.ca_file`), where previously the first file silently won.
- Closed VERIFICATION.md gap #7: an env var whose value coincidentally equals an earlier config file's value no longer gets silently discarded when a second `-config-file` is loaded.
- Added exact flag provenance via `flag.FlagSet.Visit`, so a `-grpc-address` flag equal to the compiled-in factory default still beats a later config file (previously invisible to value comparison).
- Closed `08-REVIEW.md` IN-01: `setFieldReplacing`'s new message/group arm deep-copies via `proto.Clone` instead of installing the source submessage by reference, now load-bearing because `ConfigLayerer.fileLayer` is a long-lived accumulator.
- Preserved every pre-existing behavioral guarantee: all 6 `Test_cliCommand_ConfigPrecedence` subtests and all 12 original `TestLayerConfigFile` subtests still pass unchanged, and `command.LayerConfigFile`/`matchesBase` are retained (commented as superseded) so the four not-yet-migrated components still compile and pass.

## Task Commits

Each task was committed atomically. Task 1 (`type="tracer" tdd="true"`) used two commits (RED then GREEN); Task 2 (`type="auto" tdd="true"`) is test-only against already-implemented production code and needed one commit.

1. **Task 1 (RED): failing `Test_cliCommand_MultiConfigFilePrecedence`** - `843325c` (test)
2. **Task 1 (GREEN): `ConfigLayerer` + agent wiring** - `9f09ab2` (feat)
3. **Task 2: port + extend `TestLayerConfigFile`** - `168851b` (test)

## Files Created/Modified

- `command/configfile.go` - Adds `ConfigLayerer`, `NewConfigLayerer`, `(*ConfigLayerer).LayerConfigFile`, `(*ConfigLayerer).markExplicitFlags`, `fieldEqual`, and a message/group arm in `setFieldReplacing`. Retains `LayerConfigFile` (free function) and `matchesBase`, now commented as superseded, for the four components 08-06 has not yet migrated.
- `command/configfile_test.go` - All 12 original `TestLayerConfigFile` subtests ported to the `ConfigLayerer` API (same names, same expected values), plus 9 new subtests covering both reproduced gaps, recursive submessage merge, flag provenance, and the IN-01 deep-copy fix.
- `agent/command.go` - `base` is now the pristine defaults snapshot handed to `command.NewConfigLayerer`; the `-config-file` handler calls `layerer.LayerConfigFile(c.config, preFile)` instead of the superseded three-argument free function.
- `agent/command_test.go` - Adds `Test_cliCommand_MultiConfigFilePrecedence` (7 subtests) and the `writeAgentConfigRaw` helper. `Test_cliCommand_ConfigPrecedence`, `Test_cliCommand_Run`, `Test_cliCommand_Help`, `writeAgentConfigJSON`, and the `init()` fixture are unmodified.

## Red-First Verification (recorded verbatim)

`go test ./agent -run Test_cliCommand_MultiConfigFilePrecedence -count=1 -v` against the unmodified `command/configfile.go` and `agent/command.go` (before any Task 1 production change):

```
=== RUN   Test_cliCommand_MultiConfigFilePrecedence/later_file_wins_for_tls_cert_file
    command_test.go:387: got "a.pem", want "b.pem"
=== RUN   Test_cliCommand_MultiConfigFilePrecedence/later_file_wins_for_store_tls_ca_file
    command_test.go:387: got "ca1.pem", want "ca2.pem"
=== RUN   Test_cliCommand_MultiConfigFilePrecedence/env_wins_over_two_files_when_first_file_coincides
    command_test.go:387: got ":8888", want ":9999"
=== RUN   Test_cliCommand_MultiConfigFilePrecedence/flag_before_config_file_wins_when_flag_equals_factory_default
    command_test.go:387: got ":8888", want ":4300"
=== RUN   Test_cliCommand_MultiConfigFilePrecedence/later_file_wins_for_scalar_without_env
=== RUN   Test_cliCommand_MultiConfigFilePrecedence/env_wins_over_two_files_when_neither_coincides
=== RUN   Test_cliCommand_MultiConfigFilePrecedence/three_files_last_one_wins
--- FAIL: Test_cliCommand_MultiConfigFilePrecedence (0.01s)
    --- FAIL: .../later_file_wins_for_tls_cert_file (0.00s)
    --- FAIL: .../later_file_wins_for_store_tls_ca_file (0.00s)
    --- FAIL: .../env_wins_over_two_files_when_first_file_coincides (0.00s)
    --- FAIL: .../flag_before_config_file_wins_when_flag_equals_factory_default (0.00s)
    --- PASS: .../later_file_wins_for_scalar_without_env (0.00s)
    --- PASS: .../env_wins_over_two_files_when_neither_coincides (0.00s)
    --- PASS: .../three_files_last_one_wins (0.00s)
FAIL
```

Exactly the two named gaps (plus the flag-provenance row, a stronger corollary of gap 1) failed, and the two pre-existing regression-guard rows already passed — confirming the test targets the real defect before any fix was applied. After Task 1's implementation, the same command prints `--- PASS` for all seven subtests.

## Tracer Feedback Gate

Task 1 is `type="tracer"` with no `gate` attribute and a `<verify>` block containing only `<automated>` checks (no `<human-check>`). Per the executor's tracer feedback gate (checkpoints.md #3299), with `workflow.auto_advance` and `workflow._auto_chain_active` both `false` in `.planning/config.json` and `HUMAN_VERIFY_MODE` defaulting to `end-of-phase`, the automated-only branch applies: the tracer's `<verify>` was re-run end-to-end (all `go build`, `go test`, shape-check, and `gofmt` commands in Task 1's `<verify>` block — see Deviations/Verification below) and passed, so execution proceeded directly to Task 2 with no synthesized checkpoint.

## Decisions Made

See `key-decisions` in frontmatter. No decisions required user input; all fell within Rules 1-3 (bug fix / missing critical functionality — the security-relevant TLS precedence defect) rather than Rule 4 (architectural).

## Deviations from Plan

None - plan executed exactly as written. All `<library_facts>` in the plan were confirmed by execution with no discrepancies:

- `flag.(*FlagSet).Visit` inside the `-config-file` closure enumerated exactly the flags parsed strictly before that occurrence, confirmed by `flag_before_config_file_wins_when_flag_equals_factory_default` and `flag_provenance_beats_file_when_flag_equals_factory_default` passing.
- `protojson` honored nested `json_name` for `tls-config.cert-file` and `store-tls.ca-file`, confirmed by `later_file_wins_for_tls_cert_file` / `later_file_wins_for_store_tls_ca_file`.
- `proto.Merge`'s recursive submessage merge (setting only the oneof case the source populated, leaving other oneofs on the same message untouched) was confirmed by `message_field_recursive_merge_keeps_untouched_submessage_fields`.
- No module dependency was added: `git diff --exit-code go.mod go.sum` reported no changes throughout.

## Issues Encountered

None.

## Verification (plan-level, all passed)

1. `go build ./...` — exit 0.
2. `go test ./agent ./command ./server ./compiler ./inserter ./mutate -count=1` — `ok` for all six packages.
3. `go test ./agent -run Test_cliCommand_MultiConfigFilePrecedence -count=1 -v` — `--- PASS` for all 7 subtests.
4. `go test ./agent -run Test_cliCommand_ConfigPrecedence -count=1 -v` — `--- PASS` for all 6 pre-existing subtests.
5. `go test ./command -run TestLayerConfigFile -count=1 -v` — 21 `--- PASS: TestLayerConfigFile/` lines; all 12 original names present verbatim.
6. `agent/command.go` — zero occurrences of the superseded three-argument call, zero `c.config =` assignments.
7. `git diff --exit-code go.mod go.sum` — exit 0, no changes.
8. `git diff --exit-code .planning/phases/08-cli-flag-generation-config-loading/08-0[1-4]-PLAN.md` — exit 0, prior plans untouched.
9. `.planning/REQUIREMENTS.md` PCLI-09 text unchanged (still unmarked, `Gaps Found` — this plan does not itself flip that status; it strengthens PCLI-07/08 and closes PCLI-09 for the agent only, pending 08-06 for the remaining four components).

Additional shape/security checks (Task 1 `<verify>`): `LAYERER_SHAPE_OK`, `PROVENANCE_TWO_SOURCES_OK`, `AGENT_WIRING_OK`, `NO_CONFIG_REASSIGNMENT_OK`, `agent --help` lists all 4 required flags, `gofmt -l` clean on all touched files — all passed.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

`command.ConfigLayerer` is implemented, proven end-to-end for `protoconf agent`, and both `08-VERIFICATION.md` BLOCKER gaps are closed for that component. `PCLI-05` through `PCLI-09` remain a shared requirement set with `08-06-PLAN.md` (which also declares all five in its frontmatter and migrates `compiler`, `server`, `inserter`, and `mutate` onto `ConfigLayerer`, removing the superseded `LayerConfigFile`/`matchesBase`), so none of the five requirements flip to `Complete` in `.planning/REQUIREMENTS.md` until 08-06 also finishes (shared-ID gate, #2388) — this is expected, not a blocker. No concerns carried forward.

---
*Phase: 08-cli-flag-generation-config-loading*
*Completed: 2026-09-01*

## Self-Check: PASSED

All key files confirmed present on disk (`command/configfile.go`, `agent/command.go`, `agent/command_test.go`, `command/configfile_test.go`, this SUMMARY.md), and all four task/summary commit hashes (`843325c`, `9f09ab2`, `168851b`, `5cc5c41`) confirmed present in `git log --oneline --all`.
