---
phase: 08-cli-flag-generation-config-loading
verified: 2026-09-01T14:09:47Z
status: passed
score: 10/10 must-haves verified
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 6/8
  gaps_closed:
    - "Multi-config-file precedence holds when an env var (or already-parsed flag) value coincidentally equals what an earlier -config-file set for the same scalar/list field — env still wins. Closed by command.ConfigLayerer (08-05), generalized to all five components (08-06). Independently re-run here: Test_cliCommand_MultiConfigFilePrecedence/env_wins_over_two_files_when_first_file_coincides passes on agent AND server; env_list_wins_over_two_files_when_first_file_coincides passes on inserter; TestLayerConfigFile/env_wins_over_two_files_on_value_coincidence passes at the unit level."
    - "Multi-config-file 'later file wins' guarantee holds for message-typed fields (AgentConfig.tls_config, AgentConfig.store_tls). Independently re-run here: Test_cliCommand_MultiConfigFilePrecedence/later_file_wins_for_tls_cert_file and /later_file_wins_for_store_tls_ca_file pass on agent; TestLayerConfigFile/later_file_wins_for_message_field and /later_file_wins_for_store_tls_message_field pass at the unit level."
  gaps_remaining: []
  regressions: []
---

# Phase 8: CLI Flag Generation & Config Loading Verification Report

**Phase Goal:** CLI flags are generated from proto definitions; all components accept env vars and config files
**Verified:** 2026-09-01T14:09:47Z
**Status:** passed
**Re-verification:** Yes — after gap closure (08-05 built `command.ConfigLayerer` with real provenance tracking and proved it end-to-end for `agent`; 08-06 generalized it to `serve`/`compile`/`insert`/`mutate` and removed the superseded `LayerConfigFile`/`matchesBase` value-comparison path)

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | `--help` on every component (`agent`, `serve`, `compile`, `insert`, `mutate`) shows flags generated from that component's proto config, annotated with an env key | ✓ VERIFIED | Ran all five `--help` commands directly (`go run cmd/protoconf/main.go <cmd> --help`). Each lists proto-derived flags (e.g. `-grpc-address`, `-auth-token`, `-tls-cert/-key/-ca`, `-pre`, `-post` for `serve`) with `env key: PROTOCONF_<COMPONENT>_*` annotations. |
| 2 | Existing flag names are preserved (backward compatible) via `json_name` proto annotations | ✓ VERIFIED | Re-sampled `server_config.proto`'s `json_name = "grpc-address"` etc. against live `--help` output — names match exactly. Unchanged since prior verification round; 08-05/08-06 touched only precedence resolution, not flag registration. |
| 3 | Setting a `PROTOCONF_<COMPONENT>_*` env var configures the corresponding option, for all five components | ✓ VERIFIED | `SetEnvKeyPrefix` + `lpc.Environment()` confirmed present in all five `Command()` factories; directly exercised by every `env_*` subtest below (env values reach `cc.config` after `flag.Parse`). |
| 4 | Passing `-config-file` loads configuration from JSON, YAML, or protobuf format, for all five components | ✓ VERIFIED | `-config-file` present in all five `--help` outputs with `available formats: json, yaml, pb` in the usage text; format dispatch unchanged from prior round. |
| 5 | Single `-config-file` precedence: flags > env vars > config file > proto defaults | ✓ VERIFIED | `go test ./agent ./command ./server ./compiler ./inserter ./mutate -count=1` — all six `ok`. `Test_cliCommand_ConfigPrecedence` (all five components) and `TestLayerConfigFile`'s 21 subtests pass, independently re-run here with `-v`. |
| 6 | "Later `-config-file` wins" holds for scalar and repeated-string fields | ✓ VERIFIED | `later_file_wins_for_scalar_without_env` (agent, server), `later_file_replaces_earlier_list_without_env` / `three_files_last_list_wins` (inserter), `second_file_overrides_first` / `second_file_replaces_first_list` (command unit suite) all pass, independently re-run. |
| 7 | Precedence still holds when 2+ `-config-file` flags are combined with an env var whose value coincidentally equals an earlier file's value (scalar/list fields) — PREVIOUSLY FAILED, gap #7 | ✓ VERIFIED | Independently re-run: `go test ./agent -run Test_cliCommand_MultiConfigFilePrecedence/env_wins_over_two_files_when_first_file_coincides -v` → PASS (env=:9999, file1=:9999, file2=:8888 → result :9999). Same scenario re-run and PASS on `./server` (scalar) and `./inserter` (`env_list_wins_over_two_files_when_first_file_coincides`, repeated-string). Unit-level `TestLayerConfigFile/env_wins_over_two_files_on_value_coincidence` also PASS. Root cause fixed: `ConfigLayerer` records provenance from `flag.FlagSet.Visit` and a diff against `lastResult` (a non-accumulating baseline), not from comparison against the mutating `base` accumulator that caused the original defect. |
| 8 | "Later `-config-file` wins" holds for message-typed fields (`AgentConfig.tls_config`, `store_tls`), independent of any env var — PREVIOUSLY FAILED, gap #8 | ✓ VERIFIED | Independently re-run: `go test ./agent -run Test_cliCommand_MultiConfigFilePrecedence/later_file_wins_for_tls_cert_file -v` → PASS (file1=a.pem, file2=b.pem, no env → result b.pem) and `/later_file_wins_for_store_tls_ca_file` → PASS. Unit-level `TestLayerConfigFile/later_file_wins_for_message_field`, `/later_file_wins_for_store_tls_message_field`, and `/message_field_recursive_merge_keeps_untouched_submessage_fields` (proves the fix doesn't turn later-file-wins into wholesale submessage replacement) all PASS. Root cause fixed: `setFieldReplacing`'s message/group arm now uses the same explicit-provenance re-apply step (step 5 of `LayerConfigFile`) as every other field kind, instead of `matchesBase`'s old unconditional `false` for message/map/bytes kinds. |
| 9 | An explicit flag parsed before `-config-file` wins over the config file even when byte-identical to the compiled-in factory default (flag provenance, not value-inequality inference) | ✓ VERIFIED | `flag_before_config_file_wins_when_flag_equals_factory_default` passes on `agent` and `server`; unit-level `flag_provenance_beats_file_when_flag_equals_factory_default` and its negative-control sibling `unknown_flag_name_is_ignored` both pass — confirms flag names that match no top-level field are correctly NOT treated as explicit. |
| 10 | `command/configfile.go` declares exactly one layering entry point — the superseded `LayerConfigFile` free function and `matchesBase` are gone | ✓ VERIFIED | `grep -c 'func LayerConfigFile(' command/configfile.go` = 0, `grep -c 'func matchesBase(' command/configfile.go` = 0, `grep -c 'func setFieldReplacing(' command/configfile.go` = 1, `grep -c 'func NewConfigLayerer(' command/configfile.go` = 1 — all directly re-run. `grep -rn 'command\.LayerConfigFile\b' --include='*.go' .` returns zero hits repo-wide (no dangling call sites). |

**Score:** 10/10 truths verified (both previously-failed truths independently re-confirmed closed by direct test execution, not by trusting SUMMARY claims)

### Independent Re-Verification Method

Both BLOCKER gaps from the prior verification round were closed by 08-05 (built `command.ConfigLayerer`, proved end-to-end for `agent`) and 08-06 (generalized to `server`/`compiler`/`inserter`/`mutate`, deleted the superseded free function). Rather than accept 08-05-SUMMARY.md/08-06-SUMMARY.md's claims, this verification:

1. Read `command/configfile.go` in full and hand-traced `ConfigLayerer.LayerConfigFile`'s six steps against the exact scenarios that broke the old `matchesBase`-based implementation — confirmed provenance is now recorded from two independent sources (`markExplicitFlags` via `fs.Visit`, and a `fieldEqual` diff against `lastResult`, a per-call snapshot that does NOT accumulate config-file values the way the old `base` did) and re-applied in a dedicated step (step 5) for every field kind including message-typed ones.
2. Independently re-ran (not merely inspected) every regression test naming the two prior gaps, with `-v` and `-run` scoped to the exact subtest, and read the actual PASS/FAIL output:
   - `go test ./agent -run Test_cliCommand_MultiConfigFilePrecedence -count=1 -v` → all 7 subtests PASS
   - `go test ./server -run Test_cliCommand_MultiConfigFilePrecedence -count=1 -v` → all 6 subtests PASS
   - `go test ./inserter -run Test_cliCommand_MultiConfigFilePrecedence -count=1 -v` → all 3 subtests PASS
   - `go test ./command -run TestLayerConfigFile -count=1 -v` → 21/21 subtests PASS (12 original + 9 new)
3. Ran the full six-package test suite (`go test ./agent ./command ./server ./compiler ./inserter ./mutate -count=1`) — all `ok`, confirming no regression in the pre-existing precedence guarantees while closing the two gaps.
4. Ran `go build ./...` (exit 0) and `gofmt -l` on every touched file (empty output — all gofmt-clean).
5. Confirmed the structural prohibition from 08-06: `grep -c 'func LayerConfigFile('` and `grep -c 'func matchesBase('` both return 0 in `command/configfile.go`, and a repo-wide grep for `command\.LayerConfigFile\b` returns no hits — exactly one layering entry point (`NewConfigLayerer`/`(*ConfigLayerer).LayerConfigFile`) survives.
6. Ran `go test ./... -count=1` once and confirmed the only two failures are the pre-established baseline failures (`compiler/lib` `TestCompiler_CompileFile/load_remote_with_load_local.pconf` and `test` `Test` e2e, both failing on a missing `.protoconf_cache/vizceral_repo/src/services/frontend.pinc` fixture) — identical to the documented pre-phase-08 baseline, not new regressions.

**Conclusion: both gaps are genuinely closed**, verified by direct execution against the committed code, not accepted on SUMMARY authority.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `command/configfile.go` | `ConfigLayerer`/`NewConfigLayerer`/`(*ConfigLayerer).LayerConfigFile` — provenance-tracking layering helper, sole entry point | ✓ VERIFIED | Declares `type ConfigLayerer struct`, `func NewConfigLayerer(...)` (1 occurrence), `func (l *ConfigLayerer) LayerConfigFile(...)`, `func (l *ConfigLayerer) markExplicitFlags()`, `func fieldEqual(...)`. Zero occurrences of `func LayerConfigFile(` (free function) or `func matchesBase(`. |
| `agent/command.go`, `server/server.go`, `compiler/command.go`, `inserter/inserter.go`, `mutate/mutate.go` | Config-file handlers wired to `command.NewConfigLayerer`/`layerer.LayerConfigFile`, no `c.config` reassignment | ✓ VERIFIED | `grep -c 'command.NewConfigLayerer(base, c.flag)'` = 1 and `grep -c 'layerer.LayerConfigFile(c.config, preFile)'` = 1 in each of the five files; `grep -c 'c\.config *='` = 0 in each. |
| `command/configfile_test.go` | `TestLayerConfigFile` ported to `ConfigLayerer` plus rows for both closed gaps | ✓ VERIFIED | 21 subtests, all pass; all 12 original names present verbatim (`file_overrides_factory_default`, `second_file_overrides_first`, etc.) alongside the 9 new rows. |
| `agent/command_test.go`, `server/server_test.go`, `inserter/inserter_test.go` | `Test_cliCommand_MultiConfigFilePrecedence` — CLI-level regression coverage | ✓ VERIFIED | 7 subtests (agent), 6 (server), 3 (inserter), all pass independently re-run. |
| `CHANGELOG.md` | Records the multi-config-file precedence correction, naming PCLI-09 | ✓ VERIFIED | Single `## Unreleased` heading; body names `PCLI-09`, `tls-config`, `-config-file`, and states the security-relevant TLS consequence explicitly. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `agent/command.go`, `server/server.go`, `compiler/command.go`, `inserter/inserter.go`, `mutate/mutate.go` | `command/configfile.go` | `command.NewConfigLayerer(base, c.flag)` + `layerer.LayerConfigFile(c.config, preFile)` | ✓ WIRED | One call site of each per file, confirmed by grep; all five packages compile and their precedence test suites pass. |
| `command/configfile.go`'s `ConfigLayerer.markExplicitFlags` | `flag.FlagSet` | `l.fs.Visit(...)` | ✓ WIRED | Directly exercised by `flag_before_config_file_wins_when_flag_equals_factory_default` (agent, server) and `flag_provenance_beats_file_when_flag_equals_factory_default` / `unknown_flag_name_is_ignored` (unit) — all pass. |
| Each component's proto config package | Component CLI file | `configtool.NewConfig` + `PopulateFlagSet` | ✓ WIRED | Confirmed via `--help` output showing proto-field-derived flags with correct env-key prefixes for all five components. |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| `go build ./...` | `go build ./...` | Exit 0, no diagnostics | ✓ PASS |
| Directly-touched packages' test suites pass | `go test ./agent ./command ./server ./compiler ./inserter ./mutate -count=1` | All six `ok` | ✓ PASS |
| Gap #7 (env-coincidence) closed, all affected components | `go test ./agent ./server -run Test_cliCommand_MultiConfigFilePrecedence/env_wins_over_two_files_when_first_file_coincides -v`; `go test ./inserter -run .../env_list_wins_over_two_files_when_first_file_coincides -v` | `--- PASS` on all three | ✓ PASS |
| Gap #8 (message-typed later-file-wins) closed | `go test ./agent -run Test_cliCommand_MultiConfigFilePrecedence/later_file_wins_for_tls_cert_file\|later_file_wins_for_store_tls_ca_file -v` | `--- PASS` for both | ✓ PASS |
| `TestLayerConfigFile` full unit suite | `go test ./command -run TestLayerConfigFile -count=1 -v` | 21/21 `--- PASS`, 0 `--- FAIL` | ✓ PASS |
| Single layering entry point structural prohibition | `grep -c 'func LayerConfigFile(' / 'func matchesBase('` in `command/configfile.go`; repo-wide `grep -rn 'command\.LayerConfigFile\b'` | `0`, `0`, no hits | ✓ PASS |
| All five components' `--help` still show `-config-file` | `go run cmd/protoconf/main.go <cmd> --help` for agent/serve/compile/insert/mutate | `-config-file value` present in all five | ✓ PASS |
| `gofmt -l` on all touched files | `gofmt -l command/configfile.go command/configfile_test.go agent/command.go agent/command_test.go server/server.go server/server_test.go compiler/command.go inserter/inserter.go inserter/inserter_test.go mutate/mutate.go` | Empty output | ✓ PASS |
| Full-tree regression check (baseline comparison) | `go test ./... -count=1` (run once) | Only the two documented pre-existing baseline failures (`compiler/lib` `load_remote_with_load_local.pconf`, `test` e2e `Test`), both on a missing `.protoconf_cache/vizceral_repo` fixture | ✓ PASS (no new regressions) |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| PCLI-05 | 08-01, 08-02, 08-03, 08-04, 08-05, 08-06 | CLI flag parsing generated from proto definitions for all components | ✓ SATISFIED | All five components use `configtool.NewConfig` + `PopulateFlagSet`; confirmed via `--help` for all five. |
| PCLI-06 | 08-01, 08-02, 08-03, 08-04, 08-05, 08-06 | Generated CLI matches current flag interface (backward compatible) | ✓ SATISFIED | `json_name` annotations preserve original flag names; all five `--help` outputs re-sampled and match. |
| PCLI-07 | 08-01, 08-02, 08-03, 08-04, 08-05, 08-06 | All components support config loading via env vars (`PROTOCONF_*` prefix) | ✓ SATISFIED | `SetEnvKeyPrefix` + `Environment()` present and functioning in all five `Command()` factories; env-coincidence tests directly exercise this. |
| PCLI-08 | 08-01, 08-02, 08-03, 08-04, 08-05, 08-06 | All components support config loading via config files (JSON/YAML/protobuf) | ✓ SATISFIED | `-config-file` present and functional in all five; idempotence, empty-file, and unparsable-extension edges pinned by named subtests (`same_config_file_twice_is_idempotent_across_two_flags`, `empty_second_config_file_does_not_erase_first`, `unparsable_config_file_extension_errors`), all re-run and passing. |
| PCLI-09 | 08-01, 08-02, 08-03, 08-04, 08-05, 08-06 | Config precedence: flags > env vars > config file > defaults | ✓ SATISFIED | Holds for the single-config-file case AND for 2+ `-config-file` flags combined with env-coincidence and message-typed fields, on all five components — both previously-BLOCKED scenarios independently re-verified by execution above. |

`.planning/REQUIREMENTS.md` traceability table maps exactly PCLI-05 through PCLI-09 to Phase 8 (all marked `[x]`/`Complete`), matching every plan's `requirements:` frontmatter across all six plans (08-01 through 08-06). No orphaned requirements found for this phase.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `command/configfile.go:36-41`, `143-164` | `setFieldReplacing`'s list arm / `LayerConfigFile` steps 3-4 | New finding from independent code review (08-REVIEW.md WR-01): a later `-config-file` explicitly setting a repeated/map field to an empty list cannot clear an earlier file's non-empty value for that field, because proto3 implicit presence makes `Range`/`Has` report a zero-length repeated field as absent regardless of whether the file explicitly set it | ℹ️ Info | Genuine edge-case gap, but NOT covered by any ROADMAP success criterion or PLAN must-have — no truth in this phase's scope asserts "a config file can reset a list to empty". Not gating; recommend a follow-up doc note or test row, not a phase-08 blocker. |
| `mutate/mutate.go:159-163` | `TYPE_SINT32` uses `uint32()` converter instead of `int32()` (08-REVIEW.md WR-02, formerly WR-01 in the prior review round) | ℹ️ Info | Pre-existing bug (predates phase 8, confirmed via `git diff` against parent commit), explicitly out of scope per 08-06-PLAN.md — not a phase-08 regression, not gating. |
| `agent/command.go:57-111`, `server/server.go:217-249`, `compiler/command.go:189-221`, `inserter/inserter.go:142-174`, `mutate/mutate.go:272-305` | ~20-line `ConfigLayerer` wiring block duplicated near-verbatim across all five components (08-REVIEW.md WR-03) | ℹ️ Info | Maintainability concern, not a correctness defect — every duplicate site passes its own precedence test suite. A shared `RegisterConfigFileFlag` helper is suggested but not required by any must-have. |

No `TBD`/`FIXME`/`XXX`(-as-marker)/`TODO`/`HACK`/`PLACEHOLDER` debt markers found in any of the phase's touched files. (Grep hits for the literal string `XXX` are pre-existing identifier names — `XXXinsertVersion`, `XXX_MessageName` — not debt-marker comments, and predate this phase.)

### Human Verification Required

None. Both previously-open gaps were confirmed closed by direct, independent execution against the real committed code (test runs with `-v` read line-by-line, not accepted on SUMMARY authority), and the structural single-layering-path prohibition was confirmed by grep with zero hits.

### Gaps Summary

None remaining. The phase's two BLOCKER gaps from the prior verification round — (1) env var silently lost when combined with two `-config-file` flags on a value coincidence, and (2) later-file-wins inverted for message-typed fields — are both genuinely closed:

- **Root cause fix:** `command.ConfigLayerer` replaced the value-comparison `matchesBase` heuristic with real provenance tracking: `markExplicitFlags` reads `flag.FlagSet.Visit` for exact flag-parse-ordering provenance, and a `fieldEqual` diff against `lastResult` (a per-call snapshot, not an accumulating baseline) captures env-var and newly-parsed-flag provenance. This directly addresses why the old approach was unfixable by a value-comparison patch alone (a single evolving baseline cannot distinguish "carried over from an earlier file" from "explicitly supplied").
- **Coverage:** Both gaps are now pinned by CLI-level regression tests on the exact components where they were reproduced (`agent` for message-typed fields and the original env-coincidence scenario; `server` for the scalar coincidence; `inserter` for the repeated-string coincidence) AND by nine new unit-level `TestLayerConfigFile` rows — all independently re-run in this verification and confirmed passing.
- **Generalization:** All five components (`agent`, `serve`, `compile`, `insert`, `mutate`) route through the identical `ConfigLayerer` mechanism — confirmed by grep across all five files — so the fix is not agent-only.
- **No parallel defective path survives:** the superseded `LayerConfigFile` free function and `matchesBase` were deleted entirely once every call site moved; a repo-wide grep confirms zero dangling references.
- **No regressions:** the full six-package test suite passes, `go build ./...` is clean, and a full `go test ./...` run shows only the two pre-established, environmental, phase-8-independent baseline failures.
- A new, non-gating finding from independent code review (WR-01: config files cannot reset a repeated field to empty) is recorded as an info-level anti-pattern above — it is a genuine edge case but is not covered by any ROADMAP success criterion or PLAN must-have for this phase, so it does not block phase completion.

**Phase goal achieved.** CLI flags are generated from proto definitions across all five components, and all five components correctly accept env vars and config files with the documented precedence — including under repeated `-config-file` flags and message-typed fields, which required two rounds of gap closure (08-05, 08-06) to get right.

---

_Verified: 2026-09-01T14:09:47Z_
_Verifier: Claude (gsd-verifier)_
