---
phase: 260902-erj
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - agent/kv_agent_impl.go
  - agent/kv_agent_rollout_impl.go
  - agent/store_probe_test.go
autonomous: true
requirements: [QUICK-260902-erj]

estimate:
  tokens: 35000
  raw_tokens: 23000
  tasks: 2
  confidence: low

must_haves:
  truths:
    - "The agent starts successfully against an etcd store that is reachable but empty"
    - "The agent still refuses to start when the KV store is unreachable (fail-fast preserved, NOT deleted)"
    - "Both NewProtoconfKVAgent and NewProtoconfKVAgentRollout probe with the same valid key"
    - "The probe key is non-empty after the backend strips its leading slash, for any value of config.Prefix including empty"
    - "A key that is merely absent is treated as success — a fresh store legitimately holds nothing"
  artifacts:
    - "agent/kv_agent_impl.go — one unexported checkStoreAvailable helper, called by both constructors"
    - "agent/store_probe_test.go — table test over both constructors using a local store.Store fake"
  key_links:
    - "path.Join(config.GetPrefix(), storeProbeKey) — the sentinel is always appended, so the result is never bare \"/\" and never normalizes to the empty string"
    - "store.Exists maps store.ErrKeyNotFound to (false, nil); only a transport error yields a non-nil error, so `if err != nil` is already the correct absent-is-success predicate"
---

<objective>
Fix agent startup against etcd at the root: the store-reachability probe in both agent
constructors passes the key `"/"`, which `kvtools/etcdv3`'s `normalize` (a bare
`strings.TrimPrefix(key, "/")`) reduces to the EMPTY STRING. etcd rejects an empty key with
`rpctypes.ErrEmptyKey` ("etcdserver: key is not provided"). That is a transport-level error,
not `store.ErrKeyNotFound`, so `Exists` propagates it and the constructor fails on a
perfectly healthy store.

Purpose: upstream PR #496 "fixed" this by commenting the whole check out, which trades a
false startup failure for a real one — an unreachable store would then surface as a failure
on every subscription instead of once at boot. We keep the fail-fast check and give it a
valid key.

Verified before writing this plan (do not re-derive):
- `etcdv3@v1.0.3` `Exists` -> `Get`; `Get` returns `store.ErrKeyNotFound` only when
  `result.Count == 0`, and `Exists` swallows exactly that into `(false, nil)`. Every other
  error propagates. So **key-absent is already success** and the existing `if err != nil`
  predicate is correct as written — only the key is wrong.
- `agent/filekv` and `agent/configmaps` hardcode `Exists` to `return true, nil`;
  `kvtools/consul` maps its empty-key miss to `ErrKeyNotFound`. etcd is the only backend
  that trips, which is why this went unnoticed.
- `agent/otelkv` `Exists` is a pass-through wrapper, so the fix applies through it.
- `AgentConfig.GetPrefix()` is the generated nil-safe getter.
- Callers of the two constructors: `agent/agent.go:116,118`, `devserver/command.go:53`,
  `test/e2e_test.go:275,296`, plus `agent/kv_agent_impl_test.go`. None need changes —
  the constructor signatures are unchanged.

Output: one shared `checkStoreAvailable` helper wired into both constructors, and a
regression test that provably fails against the unfixed code. Two commits.
</objective>

<execution_context>
@/Users/smintz/go/src/github.com/protoconf/protoconf/.claude/gsd-core/workflows/execute-plan.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.claude/gsd-core/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md
@CLAUDE.md
@agent/kv_agent_impl.go
@agent/kv_agent_rollout_impl.go
@agent/kv_agent_impl_test.go
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Failing regression test for the store-availability probe (RED)</name>
  <files>agent/store_probe_test.go</files>
  <behavior>
    New file, `package agent`. Follow the style of `agent/kv_agent_impl_test.go`:
    testify `require`/`assert`, table-driven subtests.

    Define a local fake, NOT a change to `agent/dummykv` — `dummykv.Exists` hardcodes
    `return true, nil`, so it cannot express either case, and making it real would require
    also changing `dummykv.Get` (which returns a plain `fmt.Errorf`, not
    `store.ErrKeyNotFound`) and would break every existing dummykv-backed test:

      type probeStore struct {
          store.Store   // embedded so only Exists needs a body; nothing else is called
          gotKey string
          exists bool
          err    error
      }
      func (p *probeStore) Exists(_ context.Context, key string, _ *store.ReadOptions) (bool, error)

    `Exists` records `key` into `gotKey` and returns the canned `exists`/`err`.

    Table over BOTH constructors so neither can regress independently. Give the table a
    `newAgent func(store.Store, *protoconf_agent_config.AgentConfig) error` column with two
    rows — one adapting `NewProtoconfKVAgent`, one adapting `NewProtoconfKVAgentRollout` —
    each discarding the concrete agent and returning only the error.

    For each constructor, three assertions:
    - Test 1 (THE regression assertion, the only one that goes red before Task 2):
      probe key validity. With `exists:false, err:nil`, run the constructor for
      `Prefix: ""` and for `Prefix: "some/prefix"`, and in both cases assert
      `strings.TrimPrefix(store.gotKey, "/")` is NOT the empty string. Assert the
      invariant, not a hardcoded sentinel string, so renaming the sentinel later does not
      break the test. This is exactly etcd's `normalize`, reproduced in the assertion.
    - Test 2 (absent key succeeds): `exists:false, err:nil` -> constructor returns nil
      error. A fresh, reachable, empty store must not block startup.
    - Test 3 (transport error fails): `err: errors.New("etcdserver: key is not provided")`
      -> constructor returns a non-nil error whose message contains
      "store is not available". Fail-fast is preserved.

    Expected RED behavior, so the executor is not surprised: the file COMPILES against the
    current unfixed constructors (their signatures are unchanged), and Tests 2 and 3 PASS
    before the fix. Only Test 1 fails, because the current probe key normalizes to "".
    A compile error here means the fake or the adapters are wrong, not that RED was reached.
  </behavior>
  <action>
    Create `agent/store_probe_test.go` implementing the behavior above. Do not touch
    `agent/dummykv`, do not touch either constructor in this task, and do not add any
    dependency — `store`, `errors`, `strings`, `context`, `testing` and testify are all
    already in use in this package.
  </action>
  <verify>
    <automated>cd /Users/smintz/go/src/github.com/protoconf/protoconf && go vet ./agent/ && if go test ./agent/ -run TestStoreAvailabilityProbe -count=1 >/dev/null 2>&1; then echo "UNEXPECTED PASS - test does not detect the bug"; exit 1; else echo "RED confirmed"; fi</automated>
  </verify>
  <done>`agent/store_probe_test.go` compiles and vets clean; `go test ./agent/ -run TestStoreAvailabilityProbe` FAILS on the probe-key assertion (not on a compile error) for both constructors.</done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: Extract one shared checkStoreAvailable helper and wire both constructors (GREEN)</name>
  <files>agent/kv_agent_impl.go, agent/kv_agent_rollout_impl.go</files>
  <behavior>
    After this task `go test ./agent/ -run TestStoreAvailabilityProbe` passes all three
    assertions for both constructors, and the rest of `./agent/...` is unaffected.
  </behavior>
  <action>
    In `agent/kv_agent_impl.go` add a package-level sentinel const and one unexported
    helper. Put both in this file rather than a new file — it already imports `context`,
    `errors`, `path`, `store` and `protoconf_agent_config`, so the change adds zero imports
    anywhere. Keep it to a const plus a function: no interface, no options struct, no new
    package.

    The const is a dot-prefixed sentinel name such as `.protoconf-agent-healthcheck`,
    chosen so it will not collide with a real config path. The helper takes a
    `context.Context`, a `store.Store` and the `*protoconf_agent_config.AgentConfig`, builds
    the probe key with `path.Join(config.GetPrefix(), storeProbeKey)`, calls
    `Exists(ctx, key, nil)`, and returns the existing `errors.Join(errors.New("store is not
    available"), err)` on a non-nil error, nil otherwise. Use the generated `GetPrefix()`
    getter, not direct field access, so a nil config cannot panic. `path.Join` collapses an
    empty prefix correctly, and because the sentinel is always appended the result is never
    bare "/" and therefore never normalizes to the empty string on etcd.

    Document WHY in a comment on the helper: the key must stay non-empty after the
    backend's leading-slash normalization, because etcd rejects an empty key with a
    transport error that `Exists` does not swallow; and an absent key is deliberately
    success, because `Exists` maps `store.ErrKeyNotFound` to `(false, nil)` so only a
    transport error reaches the caller.

    Then replace the two byte-identical probe blocks — `agent/kv_agent_impl.go` line 28 and
    `agent/kv_agent_rollout_impl.go` line 54 — with a call to the helper, passing
    `context.Background()` to preserve current behavior. Do NOT delete or comment out the
    check, and do not change either constructor's signature. Confirm `errors` and `context`
    are still used elsewhere in the rollout file (they are, at lines 236/304 and throughout)
    so no import goes stale.
  </action>
  <verify>
    <automated>cd /Users/smintz/go/src/github.com/protoconf/protoconf && gofmt -l agent/ | (! grep .) && go build ./... && go test ./agent/... -count=1 && grep -c checkStoreAvailable agent/kv_agent_impl.go agent/kv_agent_rollout_impl.go && (! grep -rn 'Exists(context.Background(), "/"' agent/)</automated>
  </verify>
  <done>`go build ./...` and `go test ./agent/...` both pass; `checkStoreAvailable` is referenced in both constructor files (count >= 2 in kv_agent_impl.go, >= 1 in kv_agent_rollout_impl.go); the old empty-normalizing probe call appears nowhere under `agent/`; gofmt reports no diffs.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| agent -> KV store (etcd/consul/zookeeper/configmaps/file) | Network call to an external datastore at process startup |

## STRIDE Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation Plan |
|-----------|----------|-----------|----------|-------------|-----------------|
| T-260902erj-01 | Denial of Service | `checkStoreAvailable` in `agent/kv_agent_impl.go` | medium | mitigate | Retain the fail-fast probe rather than deleting it (the PR #496 approach). An unreachable store must fail once at boot, not on every client subscription. Task 1 Test 3 asserts a transport error still aborts the constructor. |
| T-260902erj-02 | Information Disclosure | probe sentinel key | low | accept | The probe issues a read-only `Exists` against a dot-prefixed sentinel under the operator's own configured prefix. It never writes, and reveals nothing a client subscription would not. |

No package-manager installs and no dependency changes in this plan, so the supply-chain
threat row does not apply. No `.pb.go` file is touched and `go generate` is not run.
</threat_model>

<verification>
1. `go build ./...` — whole module still builds; no caller of either constructor broke.
2. `go test ./agent/... -count=1` — the new probe test passes and no existing
   dummykv/filekv-backed agent test regressed.
3. `grep -rn 'Exists(context.Background(), "/"' agent/` returns nothing — the
   empty-normalizing probe is gone from both files, at the root, not patched in one.
4. Sanity: `go test ./test/... -count=1 -run TestE2E` still passes (e2e constructs both
   agents against filekv/dummykv, whose `Exists` hardcodes `true, nil`, so it should be
   unaffected — if it fails, the helper signature or nil-config handling is wrong).
</verification>

<success_criteria>
- One shared `checkStoreAvailable` helper exists; neither constructor carries its own copy
  of the probe, and the fix is not duplicated.
- The probe key is non-empty after leading-slash normalization for both an empty and a
  non-empty `config.Prefix`, asserted by test for both constructors.
- Key-absent returns success; a transport error still aborts startup with
  "store is not available".
- No behavior change for consul, zookeeper, filekv, configmaps, or the otelkv wrapper.
- Scope held: no Any resolver, no GetConfig RPC, no TLS, no OTEL toggles from PR #496.
- Two atomic commits: `test(260902-erj): ...` then `fix(260902-erj): ...`.
</success_criteria>

<output>
Create `.planning/quick/260902-erj-fix-etcd-agent-startup-invalid-store-hea/260902-erj-SUMMARY.md` when done
</output>
