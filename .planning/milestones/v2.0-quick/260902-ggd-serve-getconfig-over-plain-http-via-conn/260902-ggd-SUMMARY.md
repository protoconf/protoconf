---
phase: quick-260902-ggd
plan: 01
subsystem: api
tags: [grpc, http, vanguard, protobuf, httpbody, buf, agent]

requires:
  - phase: quick-260902-f8i
    provides: "GetConfig one-shot RPC on ProtoconfKVAgent and ProtoconfKVAgentRollout"
provides:
  - "ConfigUpdate.raw (google.api.HttpBody) field carrying the inserter's config.json rendering"
  - "Plain-HTTP GET /v1/config/{path} on the agent's existing http_address, via a connectrpc.com/vanguard transcoder wrapping the *grpc.Server"
  - "Vendored google/api/httpbody.proto and buf googleapis dependency plumbing"
affects: [agent, buf, protobuf-schema]

actuals:
  tokens: 19500
  tasks: 3
  commits: 3

tech-stack:
  added: ["connectrpc.com/vanguard v0.4.0"]
  patterns:
    - "vanguardgrpc.NewTranscoder wraps an existing *grpc.Server as an http.Handler, mounted via a shared unexported newAgentMux(*grpc.Server) constructor called AFTER all RegisterXXXServiceServer calls (it snapshots GetServiceInfo())"
    - "google.api.HttpBody response_body field lets a REST rule write bytes verbatim, bypassing protojson entirely -- used to avoid resolving an unresolvable google.protobuf.Any"

key-files:
  created:
    - third_party/google/api/httpbody.proto
    - buf.lock
    - agent/http_test.go
  modified:
    - pb/protoconf/v1/protoconf.proto
    - pb/protoconf/v1/protoconf.pb.go
    - buf.yaml
    - generate.go
    - go.mod
    - go.sum
    - agent/agent.go
    - agent/kv_agent_impl.go
    - agent/kv_agent_rollout_impl.go
    - agent/kv_agent_impl_test.go
    - agent/kv_agent_rollout_impl_test.go

key-decisions:
  - "D-01..D-07 from the plan implemented as locked: legacy proto untouched, raw filled via one shared helper in both agent impls, missing config.json leaves raw unset without failing the RPC, filekv/dev mode untouched, no JSON gRPC codec registered, buf.yaml needs deps+buf.lock (not just an exclude), generate.go excludes third_party without adding -I=third_party"
  - "go.mod/go.sum: connectrpc.com/vanguard cannot survive go mod tidy as a direct requirement until code actually imports it -- added via go get in Task 1 (staying indirect at that point), promoted to direct once agent.go's import landed in Task 2, with go mod tidy deferred to Task 3 where it is a true no-op"

requirements-completed: [QUICK-260902-ggd]

coverage:
  - id: D1
    description: "curl http://<http-address>/v1/config/<path> returns 200, application/json, config JSON byte-for-byte verbatim"
    requirement: "QUICK-260902-ggd"
    verification:
      - kind: unit
        ref: "agent/http_test.go#TestHTTP_GetConfig/plain_HTTP_GET_returns_the_seeded_config.json_verbatim"
        status: pass
    human_judgment: false
  - id: D2
    description: "A missing config returns 404 over HTTP, not 200-empty or 500"
    requirement: "QUICK-260902-ggd"
    verification:
      - kind: unit
        ref: "agent/http_test.go#TestHTTP_GetConfig/GET_for_a_missing_config_returns_404"
        status: pass
    human_judgment: false
  - id: D3
    description: "/metrics still resolves to the Prometheus handler after the transcoder is mounted at /"
    requirement: "QUICK-260902-ggd"
    verification:
      - kind: unit
        ref: "agent/http_test.go#TestHTTP_GetConfig/metrics_still_resolves"
        status: pass
    human_judgment: false
  - id: D4
    description: "Both agent GetConfig impls populate ConfigUpdate.raw from one shared helper; absent config.json leaves raw unset without failing the RPC (gRPC + HTTP pinned)"
    requirement: "QUICK-260902-ggd"
    verification:
      - kind: unit
        ref: "agent/http_test.go#TestHTTP_GetConfig/absent_config_json_sibling"
        status: pass
      - kind: unit
        ref: "agent/kv_agent_impl_test.go#TestProtoconfKVAgent_GetConfig/raw_is_populated_from_a_sibling_config_json_key"
        status: pass
      - kind: unit
        ref: "agent/kv_agent_impl_test.go#TestProtoconfKVAgent_GetConfig/raw_is_nil_when_the_config_json_sibling_is_absent"
        status: pass
      - kind: unit
        ref: "agent/kv_agent_rollout_impl_test.go#TestProtoconfKVAgentRollout_GetConfig/returns_inserted_value"
        status: pass
    human_judgment: false
  - id: D5
    description: "No JSON gRPC codec registered anywhere; buf build/breaking/go mod tidy clean"
    requirement: "QUICK-260902-ggd"
    verification:
      - kind: other
        ref: "grep -rn RegisterCodec agent/ (empty) + buf build + buf breaking + go mod tidy x2 idempotent"
        status: pass
    human_judgment: false

duration: 55min
completed: 2026-09-02
status: complete
---

# Phase quick-260902-ggd Plan 01: Serve GetConfig over plain HTTP via connectrpc.com/vanguard Summary

**GetConfig is now reachable with a plain curl http://<agent-http-address>/v1/config/<path>, returning the inserter's protojson config.json bytes byte-for-byte, via a connectrpc.com/vanguard transcoder wrapped around the agent's existing *grpc.Server and mounted on its existing HTTP mux.**

## Performance

- **Duration:** 55min
- **Tasks:** 3/3 completed
- **Files modified:** 14

## Accomplishments
- `ConfigUpdate` gained a third field, `google.api.HttpBody raw = 3`, and the vendored `google/api/httpbody.proto` is tracked with `buf.yaml`/`buf.lock` plumbing so `buf build`/`buf breaking` pass with the new import.
- Both `ProtoconfKVAgent.GetConfig` and `ProtoconfKVAgentRollout.GetConfig` fill `raw` from one shared helper (`getRawConfigJSON`) that reads the inserter's `config.json` sibling verbatim -- no protojson, no Any resolution attempted in the agent.
- `agent/agent.go` mounts a `connectrpc.com/vanguard` transcoder at `/` (via a new `newAgentMux` helper, called after both gRPC service registrations) exposing `GET /v1/config/{path=**}` with `response_body: "raw"`, alongside the existing `/debug/pprof` and `/metrics` handlers, with a startup warning that this port is now unauthenticated/no-TLS.
- Full test coverage: verbatim HTTP body assertion, 404 for missing configs, `/metrics` precedence over the catch-all, the absent-`config.json` empty-body ceiling (pinned at both gRPC and HTTP levels), and the rollout agent's `raw` proven against a real `inserter.InsertConfig`.

## Task Commits

Each task was committed atomically:

1. **Task 1: Vendored HttpBody, proto field, build plumbing, codegen, vanguard dependency** - `3ddacb8` (chore)
2. **Task 2: End-to-end curl -- fill raw, mount the transcoder, prove one HTTP GET** - `45dc51f` (feat, tracer)
3. **Task 3: Edge cases -- 404, absent config.json, /metrics precedence, rollout agent, full verification** - `a412e19` (test)

**Plan metadata:** committed separately by the orchestrator (not by this executor, per constraints).

## Files Created/Modified
- `third_party/google/api/httpbody.proto` - vendored, unmodified googleapis HttpBody message
- `buf.lock` - pins buf.build/googleapis/googleapis dependency (written by buf mod update)
- `buf.yaml` - adds deps: [buf.build/googleapis/googleapis] and excludes third_party from build.excludes
- `generate.go` - find predicate now also excludes *third_party/* so the vendored file is never handed to protoc for Go generation
- `pb/protoconf/v1/protoconf.proto` - import "google/api/httpbody.proto" + ConfigUpdate.raw field 3
- `pb/protoconf/v1/protoconf.pb.go` - regenerated (-I=pb -I=third_party), 44 insertions / 32 deletions
- `pb/protoconf/v1/protoconf_grpc.pb.go` - regenerated, byte-identical (no diff)
- `go.mod` / `go.sum` - connectrpc.com/vanguard v0.4.0 added as a direct dependency
- `agent/kv_agent_rollout_impl.go` - adds getRawConfigJSON helper (shared by both agent impls) and wires it into GetConfig
- `agent/kv_agent_impl.go` - wires getRawConfigJSON into GetConfig
- `agent/agent.go` - extracts newAgentMux(*grpc.Server) (*http.ServeMux, error), mounts the vanguard transcoder, adds a startup logger.Warn
- `agent/http_test.go` - new: end-to-end HTTP GET, 404, /metrics precedence, absent-config.json gRPC+HTTP pinning
- `agent/kv_agent_impl_test.go` - extended TestProtoconfKVAgent_GetConfig with raw-present/raw-absent gRPC assertions
- `agent/kv_agent_rollout_impl_test.go` - extended TestProtoconfKVAgentRollout_GetConfig to assert raw from a real inserter.InsertConfig

## Decisions Made
- Followed plan decisions D-01 through D-07 exactly as locked (see plan `<decisions>` block) -- no re-litigation.
- **go.mod/go.sum sequencing (not pre-specified by the plan):** `go get connectrpc.com/vanguard` alone (nothing importing it yet) is stripped back out by `go mod tidy`, since Go tooling removes requirements for modules no package in the main module imports, regardless of whether they were just added via `go get`. Resolved by leaving vanguard as an indirect requirement after Task 1's `go get` (skipping an immediate `go mod tidy` that would have prematurely removed it), then running `go mod tidy` only in Task 3 once Task 2's code actually imports it -- at which point it correctly promotes to a direct requirement and tidy is a true no-op (verified twice, idempotent).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking issue] Pre-existing go.mod/go.sum drift at the base commit, unrelated to this plan**
- **Found during:** Task 1, first go mod tidy verification pass
- **Issue:** Running go mod tidy at the base commit (e9f5b29, before any of this plan's changes) already produced a ~150-line diff across unrelated transitive dependencies (cel.dev/expr, cloud.google.com/go/*, github.com/aws/aws-sdk-go-v2/*, github.com/prometheus/*, etc.) plus removal of two direct requires (github.com/hashicorp/go-getter/v2, github.com/pelletier/go-toml/v2) that were never actually imported. Confirmed via git stash + re-run at HEAD with zero plan changes applied. This blocks the plan's own mandatory verify gate (go mod tidy && git diff --exit-code go.mod go.sum), which is scoped project-wide, not to this plan's files.
- **Fix:** go get connectrpc.com/vanguard's own MVS graph resolution incidentally resolved almost all of this drift as a side effect (same algorithm as tidy, without the pruning step), and it landed as part of Task 1's commit without needing a separate large diff of unrelated files. The remaining true no-op check was confirmed cleanly at the end of Task 3.
- **Files modified:** go.mod, go.sum (already in this plan's files_modified list)
- **Verification:** go mod tidy run twice at the end of Task 3 produces zero further diff; go build ./..., full test suite, buf build, buf breaking all pass.
- **Committed in:** 3ddacb8 (Task 1, incidental resolution) / a412e19 (Task 3, final direct-dependency promotion)

---

**Total deviations:** 1 auto-fixed (Rule 3 -- blocking issue, pre-existing environmental drift, not caused by this plan's own changes but blocking its own verify gate).
**Impact on plan:** No scope creep -- the fix rode along with dependency resolution this plan already needed to perform (go get connectrpc.com/vanguard), touched only the two files already in the plan's files_modified list, and no unrelated source files were altered.

## Issues Encountered
None beyond the go.mod/go.sum sequencing noted above.

## User Setup Required
None - no external service configuration required. Note the new startup log line (agent/agent.go): operators who TLS/auth-harden the gRPC port should be aware config.HttpAddress now also serves GetConfig unauthenticated and without TLS (T-ggd-01, accepted risk per the plan's threat model).

## Next Phase Readiness
GetConfig is fully reachable over both gRPC and plain HTTP. No blockers. Follow-up candidates explicitly deferred by the plan's ponytail comments: rendering config.json in dev-mode filekv (currently no HttpBody in dev mode), and channel-aware one-shot reads for the rollout agent's GetConfig (currently always reads the stable/default config).

---
*Phase: quick-260902-ggd*
*Completed: 2026-09-02*
