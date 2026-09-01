---
phase: 08-cli-flag-generation-config-loading
verified: 2026-09-01T00:00:00Z
status: gaps_found
score: 6/8 must-haves verified
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 12/13
  gaps_closed:
    - "Single -config-file precedence: flags > env vars > config file > proto defaults now holds for all five components (server, compiler, inserter, mutate, agent), proven by Test_cliCommand_ConfigPrecedence in each package and by TestLayerConfigFile's 12 subtests in command/configfile_test.go."
    - "server/server.go (and the other four components) no longer reassign c.config inside the config-file handler; command.LayerConfigFile updates the live message in place."
  gaps_remaining: []
  regressions:
    - "NEW gap discovered by independent code review (08-REVIEW.md CR-01) and confirmed here by executing throwaway reproduction tests against the real command.LayerConfigFile: the precedence guarantee silently breaks once two -config-file flags are combined with an env var (scalar/list fields, on value coincidence) or with any message-typed field (deterministically — AgentConfig.tls_config / store_tls). This was not covered by the 08-03/08-04 test suites and is a new finding, not a re-appearance of the original gap."
gaps:
  - truth: "Multi-config-file precedence holds when an env var (or already-parsed flag) value coincidentally equals what an earlier -config-file set for the same scalar/list field — env should still win"
    status: failed
    reason: "command.LayerConfigFile compares preFile against prev := proto.Clone(base) taken BEFORE the current file is folded into base, but base is a running accumulator across every file loaded so far, not a pristine snapshot. After file 1 sets a field to the same value the env var already holds, base absorbs that value. On file 2, prev now equals the (real) env value by coincidence, matchesBase() returns true, and the env override is treated as 'unchanged' and dropped — file 2's own value silently wins instead. Independently reproduced by running a throwaway test directly against command.LayerConfigFile (not committed): env=:9999, file1=:9999, file2=:8888 -> result is :8888 (file2), not :9999 (env) as PCLI-09 and ROADMAP Success Criterion #5 require."
    artifacts:
      - path: "command/configfile.go"
        issue: "matchesBase (lines 99-127) compares against prev, a moving target that absorbs each processed config file's raw value; it cannot distinguish 'value carried over from an earlier file' from 'value explicitly supplied by env/flag' when the two coincide. Root cause analyzed in 08-REVIEW.md CR-01."
    missing:
      - "Real provenance tracking for which fields were explicitly set by env vars/flags before config-file processing began (e.g., a field-number set captured once, immediately after lpc.Environment() runs and before any flag/config-file parsing, by diffing c.config against the pristine base snapshot) — value-equality comparison against any single evolving baseline cannot solve this in general, per 08-REVIEW.md's analysis of why a naive fix breaks other already-passing tests"
      - "A regression test exercising env var + 2 config files together (none of command/configfile_test.go's 12 rows, nor any component's Test_cliCommand_ConfigPrecedence table, combines an env var with two -config-file flags)"
  - truth: "Multi-config-file 'later file wins' guarantee holds for message-typed fields (no env var needed)"
    status: failed
    reason: "matchesBase hard-codes return false for fd.IsMap() / MessageKind / GroupKind / BytesKind (configfile.go:120-124), which is meant to mean 'always treat as explicitly supplied' but the caller's true meaning is inverted for this data flow — the field is being read from preFile (the env/pre-file-parse snapshot), and unconditionally re-applying preFile's message value after every subsequent config file means the FIRST config file's message-typed value always wins over any LATER config file's value for that same field. Independently reproduced by running a throwaway test directly against command.LayerConfigFile (not committed): file1 sets AgentConfig.TlsConfig.CertFile=a.pem, file2 sets it to b.pem, no env var involved -> result is a.pem (file1), not b.pem (file2) as the documented/tested 'later config file wins' contract requires for every other field kind in this same helper."
    artifacts:
      - path: "command/configfile.go"
        issue: "matchesBase lines 120-124 treat every message/map/bytes field as unconditionally 'explicitly supplied', bypassing the equality check that scalars and lists correctly get via the list arm above it"
      - path: "agent/config/v1/agent_config.proto"
        issue: "AgentConfig.tls_config (field 8) and AgentConfig.store_tls (field 9) are exactly the affected type (TLSConfig message), and agent/command.go was migrated onto command.LayerConfigFile by 08-04 — this is not a hypothetical, it is live in a component this phase modified"
    missing:
      - "matchesBase's message/map arm needs proto.Equal(v.Message().Interface(), prev.Get(fd).Message().Interface()) instead of an unconditional false, exactly as 08-REVIEW.md CR-01's suggested fix traces (verified there to produce the correct 'later file wins' result without disturbing the scalar/list paths)"
      - "A regression test loading two -config-file flags that both set a message-typed field (e.g. AgentConfig.TlsConfig.CertFile) to different values — command/configfile_test.go has zero rows using a message-typed fixture field, and neither agent/command_test.go nor any other component's test exercises two config files against a message field"
---

# Phase 8: CLI Flag Generation & Config Loading Verification Report

**Phase Goal:** CLI flags are generated from proto definitions; all components accept env vars and config files
**Verified:** 2026-09-01T00:00:00Z
**Status:** gaps_found
**Re-verification:** Yes — after gap closure (08-03, 08-04 closed the original single-file precedence gap; this run discovers a new, independently-confirmed gap surfaced by code review)

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | `--help` on every component (`agent`, `serve`, `compile`, `insert`, `mutate`) shows flags generated from that component's proto config, annotated with an env key | ✓ VERIFIED | Ran all five `--help` commands directly. `serve --help` shows `-grpc-address`, `-auth-token`, `-tls-cert/-key/-ca`, `-pre`, `-post`, each with `env key: PROTOCONF_SERVER_*`; `agent --help`, `compile --help`, `insert --help`, `mutate --help` show the equivalent for their own prefixes. |
| 2 | Existing flag names are preserved (backward compatible) via `json_name` proto annotations | ✓ VERIFIED | `server_config.proto` declares `json_name = "grpc-address"`, `"tls-cert"`, `"auth-token"`, etc.; `--help` output uses exactly those names. Matches prior verification's flag-by-flag check, re-sampled here. |
| 3 | Setting a `PROTOCONF_<COMPONENT>_*` env var configures the corresponding option, for all five components | ✓ VERIFIED | `SetEnvKeyPrefix("PROTOCONF_SERVER")` / `_COMPILER` / `_INSERTER` / `_MUTATE` / `_AGENT"` plus `lpc.Environment()` confirmed present in all five `Command()` factories (`server/server.go`, `compiler/command.go`, `inserter/inserter.go`, `mutate/mutate.go`, `agent/command.go`); `--help` prints an `env key:` / `[env: ...]` line per flag. |
| 4 | Passing `-config-file` loads configuration from JSON, YAML, or protobuf format, for all five components | ✓ VERIFIED | `-config-file` flag present in all five components' `--help` output, usage text lists `json, yaml, pb` (agent additionally lists `jsonnet`); `lpc.Unmarshal` dispatches by extension (confirmed in library facts cited by 08-03-PLAN.md and unchanged). |
| 5 | Single `-config-file` precedence: flags > env vars > config file > proto defaults | ✓ VERIFIED | `go test ./server ./compiler ./inserter ./mutate ./agent ./command -count=1` — all six packages `ok`. `Test_cliCommand_ConfigPrecedence` (in each of `server`, `compiler`, `inserter`, `mutate`, `agent`) and `TestLayerConfigFile`'s 12 subtests in `command/configfile_test.go` explicitly assert this ordering for one config file at a time and pass. |
| 6 | "Later `-config-file` wins" holds for scalar and repeated-string fields under the tested conditions (no coincidentally-equal env value in play) | ✓ VERIFIED | `later_config_file_wins`, `second_file_overrides_first`, `second_file_replaces_first_list` subtests pass in `server/server_test.go` and `command/configfile_test.go`. |
| 7 | Precedence still holds when 2+ `-config-file` flags are combined with an env var whose value coincidentally equals an earlier file's value (scalar/list fields) | ✗ FAILED | Independently reproduced by running a throwaway test directly against `command.LayerConfigFile` (built, executed, then deleted — not committed; source shown in Gaps section below). `env=:9999, file1=:9999, file2=:8888` → result `:8888` (file 2 silently wins), not `:9999` (env, as PCLI-09/ROADMAP SC#5 require). Root cause: `matchesBase` compares against a moving `base` snapshot that absorbs each processed file's value. See `08-REVIEW.md` CR-01 and gap #1 below. |
| 8 | "Later `-config-file` wins" holds for message-typed fields (e.g. `AgentConfig.tls_config`, `store_tls`), independent of any env var | ✗ FAILED | Independently reproduced the same way. Two `-config-file` flags setting `AgentConfig.TlsConfig.CertFile` to `a.pem` then `b.pem`, no env involved → result `a.pem` (the FIRST file), not `b.pem` (the later file, as the tested/documented contract for every other field kind requires). Root cause: `matchesBase` hard-codes `return false` (meaning "always re-apply preFile's value") for `MessageKind`/`GroupKind`/`BytesKind`/map fields, which inverts the intended semantics for this specific field kind. See `08-REVIEW.md` CR-01 and gap #2 below. |

**Score:** 6/8 truths verified (2 present-and-wired but behaviorally FAILED — not `PRESENT_BEHAVIOR_UNVERIFIED`, these were directly exercised and shown to produce wrong output)

### Independent Verification of Code Review Finding CR-01

`08-REVIEW.md`'s CR-01 claims the `command.LayerConfigFile` precedence guarantee breaks for (a) scalar/list fields when an env var value coincides with an earlier config file's value, and (b) deterministically for any message-typed field. This was evaluated independently — not accepted on the reviewer's authority — by:

1. Reading `command/configfile.go` line by line and hand-tracing both scenarios against the actual `matchesBase`/`setFieldReplacing`/`LayerConfigFile` implementation (reproduced in full below).
2. Confirming no existing test in `command/configfile_test.go`, `server/server_test.go`, `agent/command_test.go`, `inserter/inserter_test.go`, `mutate/mutate_test.go`, or `compiler/command_test.go` combines two `-config-file` loads with an env var, or exercises a message-typed field at all (`grep` for `TlsConfig`/`tls_config`/`store_tls` across all `*_test.go` in the phase's changed files returned zero matches).
3. Writing a standalone Go test file directly against `command.LayerConfigFile` (not part of the tracked diff — created, run, then deleted) reproducing both scenarios and running it:

```
$ go test ./command -run 'TestRepro_' -v -count=1
=== RUN   TestRepro_ScalarCoincidence
    zz_repro_test.go:27: BUG CONFIRMED: env value lost after second config file. got :8888 want :9999 (env)
--- FAIL: TestRepro_ScalarCoincidence (0.00s)
=== RUN   TestRepro_MessageField
    zz_repro_test.go:52: BUG CONFIRMED: second config file's message value lost. got a.pem want b.pem
--- FAIL: TestRepro_MessageField (0.00s)
FAIL
```

**Conclusion: CR-01 holds.** Both failure modes are real, reproducible against the committed code, and neither is covered by any test in the repository. This is not a hypothetical edge case — `AgentConfig.tls_config` / `store_tls` are live fields in a component (`agent`) this phase modified (08-04), and the "multiple `-config-file` flags" scenario is an explicitly documented, explicitly tested feature (`later_config_file_wins`, `second_file_overrides_first`) whose guarantee this bug breaks under adjacent, realistic conditions. `git status` confirms the reproduction file was not left in the tree.

This is recorded as two FAILED truths (#7, #8) and two BLOCKER gaps below, not a human-judgment item — the failure was directly observed by execution, not inferred from absence of coverage.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `command/configfile.go` | `LayerConfigFile` — shared precedence-layering helper implementing PCLI-09 | ⚠️ VERIFIED but contains a correctness bug | Exists, exported, wired into all five components identically (`grep -rn "command.LayerConfigFile(c.config, base, preFile)"` matches `server/server.go`, `compiler/command.go`, `inserter/inserter.go`, `mutate/mutate.go`, `agent/command.go` — one hit each). Passes its own 12-subtest unit suite, but see gaps #1/#2 for the scenarios it does not correctly handle. |
| `command/configfile_test.go` | `TestLayerConfigFile` — unit coverage of layering edges | ✓ VERIFIED (as far as it goes) | 12 subtests, all pass (`go test ./command -run TestLayerConfigFile -v` — confirmed via full package run). Does not cover message-typed fields or env+2-files combinations — the exact gap. |
| `server/server.go`, `compiler/command.go`, `inserter/inserter.go`, `mutate/mutate.go`, `agent/command.go` | Config-file handlers rewired to `command.LayerConfigFile`, no `c.config` reassignment | ✓ VERIFIED | `grep -c 'c.config, _ = orig'` and `grep -c 'proto.Merge(orig'` both return 0 across all five files; each calls `command.LayerConfigFile(c.config, base, preFile)` exactly once. |
| `CHANGELOG.md` | Records the agent's precedence-order behavior change | ✓ VERIFIED | `## Unreleased` / `### ⚠ BREAKING CHANGES` entry present, names PCLI-09 and the agent-specific behavior change explicitly. |
| `command/command.go` | KVStoreConfig/AddKVStoreFlags/KVStore* dead code removed | ✓ VERIFIED (carried over from initial verification, unaffected by 08-03/04) | Re-confirmed no regressions — package still builds and tests pass. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `server/server.go`, `compiler/command.go`, `inserter/inserter.go`, `mutate/mutate.go`, `agent/command.go` | `command/configfile.go` | `command.LayerConfigFile(c.config, base, preFile)` | ✓ WIRED | One call site per file, confirmed by grep; all five packages compile and their test suites pass. |
| Each component's proto config package (`server/config/v1`, `compiler/config/v1`, `inserter/config/v1`, `mutate/config/v1`, `agent/config/v1`) | Component CLI file | `configtool.NewConfig` + `PopulateFlagSet` | ✓ WIRED | Confirmed via `--help` output showing proto-field-derived flags with correct env-key prefixes for all five components. |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| `go build ./...` | `go build ./...` | Exit 0, no diagnostics | ✓ PASS |
| Directly-touched packages' test suites pass | `go test ./server ./compiler ./inserter ./mutate ./command ./agent -count=1` | All six `ok` | ✓ PASS |
| `serve`/`agent`/`compile`/`insert`/`mutate` `--help` show proto-generated flags + `-config-file` with precedence text | `go run cmd/protoconf/main.go <cmd> --help` (all five) | All show env-annotated flags and a `-config-file` usage string stating "Values are overridden by PROTOCONF_<X>_* environment variables and by command-line flags" | ✓ PASS |
| Pre-existing, environmental test failure reproduces identically off-phase | `go test ./compiler/lib -run TestCompiler_CompileFile/load_remote_with_load_local.pconf -count=1` | Fails with `open .../.protoconf_cache/vizceral_repo/src/services/frontend.pinc: no such file or directory` | Confirmed pre-existing/environmental, not a phase-08 regression — excluded from gaps per task instructions and independently re-run here |
| CR-01 reproduction — scalar/list env-coincidence and message-field precedence | Throwaway test against `command.LayerConfigFile` (created, run, deleted) | Both fail with `BUG CONFIRMED` (see above) | ✗ FAIL — recorded as gaps #1/#2, not a spot-check pass |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| PCLI-05 | 08-01, 08-02, 08-03, 08-04 | CLI flag parsing generated from proto definitions for all components | ✓ SATISFIED | All five components use `configtool.NewConfig` + `PopulateFlagSet`; confirmed via `--help` for all five. |
| PCLI-06 | 08-01, 08-02, 08-03, 08-04 | Generated CLI matches current flag interface (backward compatible) | ✓ SATISFIED | `json_name` annotations preserve original flag names across all five components; sampled and confirmed. |
| PCLI-07 | 08-01, 08-02, 08-03, 08-04 | All components support config loading via env vars (`PROTOCONF_*` prefix) | ✓ SATISFIED | `SetEnvKeyPrefix` + `Environment()` present and functioning in all five `Command()` factories. |
| PCLI-08 | 08-01, 08-02, 08-03, 08-04 | All components support config loading via config files (JSON/YAML/protobuf) | ✓ SATISFIED | `-config-file` present and functional in all five components; formats confirmed in usage text. |
| PCLI-09 | 08-01, 08-02, 08-03, 08-04 | Config precedence: flags > env vars > config file > defaults | ✗ BLOCKED | Holds for the single-config-file case (proven by tests). Provably fails — independently reproduced by execution — when 2+ `-config-file` flags combine with (a) an env var whose value coincides with an earlier file's value for a scalar/list field, or (b) any message-typed field (deterministic, no env needed — affects live `agent` fields `tls_config`/`store_tls`). REQUIREMENTS.md marks PCLI-09 `[x]` and traceability table says "Complete" — both are premature given this finding. |

REQUIREMENTS.md traceability table maps exactly PCLI-05 through PCLI-09 to Phase 8, matching every plan's `requirements:` frontmatter (08-01 through 08-04, all list `[PCLI-05, PCLI-06, PCLI-07, PCLI-08, PCLI-09]`). No orphaned requirements found for this phase.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `command/configfile.go` | 99-127 | `matchesBase` — logic bug, not a debt marker (see CR-01 / gaps #1, #2) | 🛑 Blocker | Breaks PCLI-09/ROADMAP SC#5 for a documented, tested multi-file scenario |
| `mutate/mutate.go` | 159-163 | `TYPE_SINT32` uses `uint32()` converter instead of `int32()` (08-REVIEW.md WR-01) | ℹ️ Info | Pre-existing bug (predates phase 8, confirmed via `git log -p`), unrelated to CLI flag generation/config loading goal — not a phase-08 regression, not gating this verification |
| `agent/command.go:64`, `compiler/command.go:195`, `server/server.go:223`, `inserter/inserter.go:148`, `mutate/mutate.go:279` | — | `lpc.Environment()` error return discarded (08-REVIEW.md WR-02) | ⚠️ Warning | Currently a no-op in vendored `libprotoconf@v0.1.0`, but silently swallows a future error surface; does not affect any of this phase's observable truths today |
| `command/configfile.go:69-88` | `setFieldReplacing` default branch aliases source `protoreflect.Value` for message fields instead of cloning (08-REVIEW.md IN-01) | ℹ️ Info | No observable bug today (`preFile` is a short-lived local), but a latent hazard if the helper is reused elsewhere |

No `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` markers found in any of the phase's touched files (`agent/command.go`, `command/command.go`, `command/configfile.go`, `compiler/command.go`, `inserter/inserter.go`, `mutate/mutate.go`, `server/server.go`).

### Human Verification Required

None. Both gaps were confirmed by direct execution against the real code, not left to inference or design-intent ambiguity.

### Gaps Summary

The phase closed its original verification gap (single-`-config-file` precedence, `flags > env vars > config file > proto defaults`) correctly and comprehensively across all five components — `command.LayerConfigFile` is a real, well-tested, shared implementation for the scenarios its own test suite covers, and re-running every directly-touched package's test suite plus a fresh `go build ./...` confirms no regressions.

However, an independent code review (`08-REVIEW.md` CR-01) identified — and this verification independently reproduced by executing throwaway tests against the actual `command.LayerConfigFile` — a genuine, unresolved correctness gap in the same helper: the precedence guarantee silently inverts once **two or more `-config-file` flags** are combined with either (1) an env var whose value coincidentally matches what an earlier file set for a scalar/repeated-string field, or (2) **any message-typed field**, deterministically, with no env var required at all. Case (2) is not a corner case: `AgentConfig.tls_config` and `AgentConfig.store_tls` are exactly this field kind, live in the `agent` component this phase modified in 08-04, and a user running `protoconf agent -config-file first.json -config-file second.json` with different TLS material in each file will silently get the FIRST file's certificate/key — a real, security-relevant footgun for a feature (`later_config_file_wins`) this same phase explicitly built and tested for every other field kind.

Both root causes trace to `matchesBase` in `command/configfile.go`: it compares candidate values against `prev`, a snapshot of `base` that itself accumulates every previously-processed config file's raw values rather than a value with clear provenance, and it hard-codes "always explicitly supplied" for message/map/bytes fields — the opposite of the correct behavior for this data flow. `08-REVIEW.md` traces a fix for the message-field case (`proto.Equal` instead of an unconditional `false`) and explains why the scalar/list coincidental-value case needs real provenance tracking rather than a value-comparison patch, since a naive fix breaks the already-passing `second_file_overrides_first` / `later_config_file_wins` tests.

**This looks like a genuine defect, not an intentional deviation** — there is no comment, design note, or CONTEXT.md decision accepting this behavior, and it directly contradicts the phase's own `must_haves` truth ("When two `-config-file` flags are passed, the later file's value overrides the earlier file's value for the same field") and PCLI-09/ROADMAP Success Criterion #5. No override is suggested; this must be closed with a new gap-closure plan (a natural `08-05`) before the phase goal can be considered achieved.

---

_Verified: 2026-09-01T00:00:00Z_
_Verifier: Claude (gsd-verifier)_
