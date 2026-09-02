---
phase: quick-260902-eie
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - inserter/inserter.go
  - inserter/inserter_test.go
autonomous: true
requirements: [QUICK-260902-EIE]

estimate:
  tokens: 26000
  raw_tokens: 13000
  tasks: 2
  confidence: low

must_haves:
  truths:
    - "Inserting a materialized config whose message embeds a populated google.protobuf.Any of a user-defined type succeeds instead of aborting the whole insert."
    - "The config.json written to the KV store contains the resolved Any payload (type URL plus inner fields) for singular, repeated, and map Any fields."
    - "go test ./inserter/... and go build ./... both pass."
  artifacts:
    - inserter/inserter.go
    - inserter/inserter_test.go
  key_links:
    - "XXXinsertVersion's config.json protojson.Marshal must use i.parser.LocalResolver, the same resolver parser.ReadConfig already uses on the read side."
---

<objective>
Fix the missing `Resolver` on the config.json `protojson.Marshal` in
`(*ProtoconfInserter).XXXinsertVersion`, so materialized configs containing
`google.protobuf.Any` values of user-defined types can be inserted. Backport of the
Any-resolution part of upstream PR #496 only.

Purpose: today any config built with the `any.new()` Starlark builtin fails to insert entirely —
protojson falls back to `protoregistry.GlobalTypes`, which has never heard of the user's protos.
Output: a one-line production fix plus a regression test that fails without it.
</objective>

<execution_context>
@/Users/smintz/go/src/github.com/protoconf/protoconf/.claude/gsd-core/workflows/execute-plan.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.claude/gsd-core/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md
@CLAUDE.md
@inserter/inserter.go
@inserter/inserter_test.go
</context>

<prior_investigation>
The planner reproduced the bug and validated the fix against the real code before writing this
plan. The executor does not need to rediscover any of the following.

**1. The bug reproduces with an EXISTING fixture — no new proto surface is needed.**

`utils/testdata/small/materialized_config/field_type_any_test.materialized_JSON` (produced by
`utils/testdata/small/src/field_type_any_test.pconf` via `any.new()`) already carries all three
Any shapes: singular `anyField`, `anyRepeated`, and `anyMap`. Every inner Any is
`type.googleapis.com/test.v1.TestMessage`, declared in `utils/testdata/small/src/test.proto`,
which has no generated `.pb.go` and therefore exists ONLY in the parser's `LocalResolver`. That
is exactly the condition the bug needs. `testdata.SmallTestDir()` — already used throughout
`inserter/inserter_test.go` — materializes it into a temp git repo.

**2. Observed failure on today's `main`:**

```
proto: google.protobuf.Any: unable to resolve "type.googleapis.com/test.v1.TestMessage": not found
error marshaling ProtoconfValue to json, value=proto_file:"test.proto" ...
```

**3. Observed output AFTER applying the fix** (all three Any shapes serialize):

```
{
  "stringValue": "recovered_from_any",
  "anyField":    { "@type": "type.googleapis.com/test.v1.TestMessage", "stringValue": "test_any" },
  "anyRepeated": [ { "@type": "...TestMessage", "stringValue": "test_any_repeated" } ],
  "anyMap":      { "hello": { "@type": "...TestMessage", "stringValue": "test_any_map" } }
}
```

**4. Sibling-call-site audit — already performed repo-wide. Result: `inserter.go:379` is the
only gap. Do not touch anything else.**

| Call site | Message marshaled | Verdict |
|---|---|---|
| `inserter/inserter.go:379` | `dynamicpb` built from `LocalResolver` — carries the user's config | **THE FIX.** Only site missing a resolver on a user-typed path |
| `inserter/inserter.go:308` | `*protoconf_pb.ProtoconfValue_ConfigRollout` | **Leave alone.** Generated Go type in GlobalTypes; fields are only Duration / Timestamp / scalars / `map<string,string>`. No Any reachable |
| `inserter/inserter.go:390` | `*protoconf_pb.Metadata` | **Leave alone.** Generated Go type in GlobalTypes; strings + Timestamps only. No Any reachable |
| `server/server.go:416` | user value | Already passes `Resolver:` — correct, out of scope |
| `compiler/lib/compiler.go:294` | `ProtoconfValue` | Already passes `Resolver: c.parser.LocalResolver` — correct, out of scope |
| `compiler/lib/module_service.go:202` | module lock head | protoconf-owned, no Any. Correct as-is, out of scope |
| `compiler/lib/parser/parser.go` `ReadConfig` | read side | Already passes `Resolver: p.LocalResolver` — this plan is the write side catching up |

The inserter was the last holdout. This is the root-cause fix, not a symptom patch.
</prior_investigation>

<tasks>

<task type="tracer">
  <name>Task 1: Pass LocalResolver to the config.json protojson.Marshal</name>
  <files>inserter/inserter.go</files>
  <action>
In `(*ProtoconfInserter).XXXinsertVersion`, at the "Writing config json" step (currently line
379), add `Resolver: i.parser.LocalResolver` to the `protojson.MarshalOptions` literal that
marshals the `new` dynamicpb message, so it reads
`protojson.MarshalOptions{Multiline: true, Resolver: i.parser.LocalResolver}.Marshal(new)`.

This is the entire production change. Do NOT add a resolver to the two sibling marshals in this
same file (the rollout one a few lines above at 308, and the metadata one immediately below at
390) — both marshal generated protoconf-owned Go types that are registered in
`protoregistry.GlobalTypes` and whose field sets cannot reach a `google.protobuf.Any`. The
repo-wide audit in the prior_investigation section is authoritative; do not re-audit and do not widen
the diff. Do NOT run `go generate` and do NOT touch any `.pb.go` file — no proto change is
needed. Do NOT pull in any other part of upstream PR #496 (etcd health check, GetConfig RPC,
TLS, OTEL toggles are separate scheduled PRs).
  </action>
  <verify>
    <automated>go build ./... && grep -v '^[[:space:]]*//' inserter/inserter.go | grep -c 'Resolver: i.parser.LocalResolver' | grep -qx 1 && go test ./inserter/... -count=1</automated>
  </verify>
  <done>`inserter/inserter.go` contains the resolver on exactly one marshal — the config-json one — and neither sibling marshal gained it (the count-is-exactly-1 gate is what proves the rollout and metadata marshals were left alone); `go build ./...` and `go test ./inserter/... -count=1` pass; `git diff --stat` shows one changed file and one changed line.</done>
  <reversibility rating="reversible">One-line change to marshal options; revert is a single-line edit.</reversibility>
</task>

<task type="auto" tdd="true">
  <name>Task 2: Regression test proving Any values survive the config.json write</name>
  <files>inserter/inserter_test.go</files>
  <behavior>
    - Inserting `field_type_any_test.materialized_JSON` returns no error (today it returns `unable to resolve "type.googleapis.com/test.v1.TestMessage": not found`).
    - The KV key `field_type_any_test/config.json` exists and its value mentions the inner Any type URL `type.googleapis.com/test.v1.TestMessage`.
    - That value carries the payload of all three Any shapes: `test_any` (singular `any_field`), `test_any_repeated` (repeated), `test_any_map` (map).
  </behavior>
  <action>
Add ONE new top-level test function to `inserter/inserter_test.go`, e.g.
`TestProtoconfInserter_InsertConfig_AnyResolution`, following the file's existing fixture style:
`dummykv.New(context.Background(), []string{}, &dummykv.Config{})` for the store,
`testdata.SmallTestDir()` for the root, `NewProtoconfInserter(dir, kvStore)`, then
`InsertConfigFile("field_type_any_test.materialized_JSON")` asserted with `require.NoError`, then
`kvStore.Get(ctx, "field_type_any_test/config.json", &store.ReadOptions{})` and
`assert.Contains` over the four substrings listed in the behavior block.

Every import this needs (`context`, `testing`, `store`, `dummykv`, `testdata`, `assert`,
`require`) is already present in the file — add no new imports and no new fixture files.

Use a dedicated function rather than a row in `TestProtoconfInserter_InsertConfig`'s table on
purpose: that table asserts with `strings.HasPrefix` against `"{"`, which cannot distinguish a
resolved Any from any other JSON, and protojson deliberately randomizes its indentation
whitespace, so a longer literal prefix would be flaky. Substring assertions are whitespace-proof.
Leave the existing table and its two rows untouched.

Prove the test is a real regression test before finishing: `git stash push inserter/inserter.go`,
run `go test ./inserter/... -count=1 -run AnyResolution` and confirm it FAILS with the resolver
error, then `git stash pop` and confirm it passes. Record both outcomes in the SUMMARY.
  </action>
  <verify>
    <automated>go test ./inserter/... -count=1 && go build ./...</automated>
  </verify>
  <done>`go test ./inserter/... -count=1` passes with the fix and the new test fails without it (red/green both observed and recorded); no new imports, no new fixture files, existing test functions unmodified.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| inserter → KV store | Compiled config JSON written to Consul/etcd/ZooKeeper/ConfigMaps and later served to applications |

## STRIDE Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation Plan |
|-----------|----------|-----------|----------|-------------|-----------------|
| T-eie-01 | Information Disclosure | `XXXinsertVersion` config.json write | low | accept | The resolver only widens type resolution to the project's own descriptors — the same set `parser.ReadConfig` already trusts on the read side, and the same binary `config.data` payload is already written unconditionally. No new data is exposed |
| T-eie-02 | Denial of Service | `InsertConfigFile` | low | mitigate | This change removes a DoS-shaped failure: today one Any-bearing config aborts its entire insert. Regression test in Task 2 locks the behavior |
| T-eie-SC | Tampering | package installs | n/a | accept | No package-manager install in this plan; no dependency added or changed |
</threat_model>

<verification>
1. `go build ./...` — clean.
2. `go test ./inserter/... -count=1` — all green, including the new Any-resolution test.
3. `git diff --stat` — exactly two files touched (`inserter/inserter.go`, `inserter/inserter_test.go`); the production diff is one line; no `.pb.go` file appears.
4. Red-check recorded: the new test fails on stashed-fix with `unable to resolve "type.googleapis.com/test.v1.TestMessage"`.
</verification>

<success_criteria>
- Configs containing `google.protobuf.Any` of user-defined types insert successfully.
- The written `config.json` contains the resolved Any payload for singular, repeated, and map fields.
- The two sibling marshals in `inserter.go` (rollout, metadata) are unchanged.
- No proto regeneration, no dependency change, nothing else from PR #496.
</success_criteria>

<output>
Create `.planning/quick/260902-eie-fix-any-resolution-in-inserter-json-mars/260902-eie-SUMMARY.md` when done.
</output>
