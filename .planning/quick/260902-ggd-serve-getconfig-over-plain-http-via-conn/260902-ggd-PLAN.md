---
phase: quick-260902-ggd
plan: 01
type: execute
wave: 1
depends_on: []
autonomous: true
requirements: [QUICK-260902-ggd]

files_modified:
  - third_party/google/api/httpbody.proto      # NEW — already on disk, UNTRACKED; must be `git add`ed
  - buf.yaml
  - buf.lock                                   # NEW — written by `buf dep update`
  - generate.go
  - go.mod
  - go.sum
  - pb/protoconf/v1/protoconf.proto
  - pb/protoconf/v1/protoconf.pb.go            # REGENERATED — do not hand-edit
  - pb/protoconf/v1/protoconf_grpc.pb.go       # REGENERATED (expected byte-identical) — do not hand-edit
  - agent/kv_agent_rollout_impl.go
  - agent/kv_agent_impl.go
  - agent/kv_agent_impl_test.go
  - agent/agent.go
  - agent/http_test.go                         # NEW

estimate:
  tokens: 40000
  raw_tokens: 40000
  tasks: 3
  confidence: low

must_haves:
  truths:
    - "`curl http://<agent-http-address>/v1/config/<path>` returns 200 with `Content-Type: application/json` and the config JSON **verbatim** — not base64, not wrapped in a `{\"raw\": ...}` envelope"
    - "`curl` for a config that does not exist returns 404, not 200-with-empty-body and not 500"
    - "`/metrics` and `/debug/pprof` still resolve to their own handlers after the transcoder is mounted at `/`"
    - "gRPC clients calling `GetConfig` are unaffected: `ConfigUpdate.value` still carries the Any, and a missing `config.json` sibling does NOT fail the RPC"
    - "The agent never attempts to protojson-marshal `ConfigUpdate.value` — no resolver is registered, and no JSON gRPC codec is registered, because either would fail on the unresolvable user-defined Any"
    - "`buf build` and `buf breaking` still pass with the new `google/api/httpbody.proto` import"
    - "`go mod tidy` leaves no diff"
  artifacts:
    - third_party/google/api/httpbody.proto
    - pb/protoconf/v1/protoconf.proto
    - agent/agent.go
    - agent/http_test.go
    - buf.lock
  key_links:
    - "`ConfigUpdate.raw` MUST be `google.api.HttpBody`, never a plain `bytes` field. Vanguard's `restIsHTTPBody` (protocol_rest.go:446) only takes the verbatim-write path when the response_body field's message full-name is exactly `google.api.HttpBody`; any other field type routes through `JSONCodec.MarshalAppendField`, which base64-encodes bytes into a JSON string"
    - "`vanguardgrpc.NewTranscoder` reads `server.GetServiceInfo()`, so it MUST be called AFTER both `RegisterProtoconfServiceServer` calls in `RunAgent`. Called earlier it silently registers zero services and every REST path 404s"
    - "Do NOT call `encoding.RegisterCodec(vanguardgrpc.NewCodec(&vanguard.JSONCodec{...}))`. `vanguardgrpc.NewTranscoder` (vanguardgrpc.go:55-59) adds `json` to the target codec set iff `encoding.GetCodec(\"json\") != nil`; the gRPC handler would then protojson-marshal `ConfigUpdate` and fail on the unresolvable Any, breaking the very RPC this plan exposes"
    - "`pb/protoconf/v1/protoconf.proto` MUST be generated with `-I=pb` (plus the new `-I=third_party`). Generating with `-I=.` renames the symbol to `File_pb_protoconf_v1_protoconf_proto` and breaks `server/server.go:285`"
    - "`buf build` FAILS today if the import is added without a buf fix — the vendored file's module-relative path is `third_party/google/api/httpbody.proto`, which never matches the import path `google/api/httpbody.proto`. Reproduced in a scratch module both with and without `third_party` in `build.excludes`. The fix is `deps: [buf.build/googleapis/googleapis]` + `buf.lock`"
    - "`generate.go`'s `find` would now sweep up `third_party/google/api/httpbody.proto` and emit a stray `httpbody.pb.go` under the googleapis go_package. It needs `-not -path '*third_party/*'`"
---

<objective>
Serve `GetConfig` to clients that cannot speak gRPC — curl, a shell script, a browser, a legacy service — by wrapping the agent's existing `*grpc.Server` with a `connectrpc.com/vanguard` transcoder mounted on the agent's existing HTTP mux.

Purpose: `GetConfig` landed in the parent branch but is reachable only over gRPC. A one-line `curl` is the whole point of a one-shot read.
Output: `curl http://agent:4380/v1/config/some/config/path` returns 200, `application/json`, and the config JSON byte-for-byte.
</objective>

<execution_context>
@/Users/smintz/go/src/github.com/protoconf/protoconf/.claude/gsd-core/workflows/execute-plan.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.claude/gsd-core/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md
@CLAUDE.md
@.planning/quick/260902-f8i-add-getconfig-one-shot-rpc-to-protoconfs/260902-f8i-SUMMARY.md

@pb/protoconf/v1/protoconf.proto
@agent/kv_agent_impl.go
@agent/kv_agent_rollout_impl.go
@agent/agent.go
@buf.yaml
@generate.go
</context>

<why_httpbody>
**Read this before writing any code. Every design choice below follows from it.**

`ConfigUpdate.value` is a `google.protobuf.Any` holding a **user-defined** message. Those messages have no generated Go code and are absent from `protoregistry.GlobalTypes`. The agent holds only bytes from the KV store — it has no descriptors for them. `protojson` (and therefore vanguard's `JSONCodec`) cannot marshal that Any; it fails with `unable to resolve "type.googleapis.com/...": not found`. There is no resolver we can hand it. **The agent can never render `value` to JSON.**

Vanguard's escape hatch (verified by reading `connectrpc.com/vanguard@v0.4.0`): when a REST rule's `response_body` resolves to a field whose message type is `google.api.HttpBody`, `restClientProtocol.prepareMarshalledResponse` (protocol_rest.go:227-245) writes that message's `data` bytes to the response **verbatim** and sets `Content-Type` from its `content_type` field. No protojson, no resolver, no base64.

A plain `bytes` field does NOT work — non-message response bodies route through `JSONCodec.MarshalAppendField`, which base64-encodes bytes into a JSON string.

Where the bytes come from: `inserter.XXXinsertVersion` (inserter/inserter.go:344-387) already writes a `config.json` key next to `config.data` — the protojson rendering, done at insert time by a process that *does* have the descriptors via `parser.LocalResolver`. `GetConfig` fills the HttpBody from that key, and the Any-resolution problem never has to be solved in the agent at all.
</why_httpbody>

<decisions>
These questions were raised in the task brief and are answered here. Do not re-litigate them; implement them.

**D-01 — The legacy proto (`agent/api/proto/v1/protoconf_service.proto`) does NOT get `raw`.**
The transcoder auto-registers every service on the `*grpc.Server`, but only methods carrying an HTTP rule get a REST route. Exactly one rule is supplied, selector `protoconf.v1.ProtoconfService.GetConfig`. The legacy `v1.ProtoconfService` has no rule, so it gets no REST route and never needs `response_body: "raw"`. Adding a second rule would also want the same `/v1/config/{path=**}` URL and collide. The legacy service is a wire-compat shim for old `protoconfservice` clients whose generated code has no `raw` field.
Consequence, accepted knowingly: `agent/legacy.go`'s `upgrade()` marshal round-trip drops field 3 into the legacy message's unknown fields. Invisible to legacy clients, harmless. Do NOT "fix" it, and do NOT add a `reserved 3;` placeholder — that is speculative.

**D-02 — `raw` is filled from `path.Join(prefix, request.Path, "config.json")` in BOTH agent impls, via one shared helper.**
That is exactly the sibling key the inserter writes. The rollout agent's primary key is `prefix/path/config.data`, so `config.json` is its literal sibling. The non-rollout agent gets the identical lookup: same helper, no behavioral asymmetry between two implementations of the same RPC, and harmless when the key is absent. One helper, two call sites.

**D-03 — A missing `config.json` leaves `raw` unset; the RPC still succeeds.**
`GetConfig`'s contract is the `value` field. A gRPC caller must not be broken because a JSON sibling written by a different producer is absent. Only `store.ErrKeyNotFound` is tolerated — any *other* store error still fails the RPC with `codes.Internal`, because a store outage must not silently degrade into an empty HTTP body.
Consequence, verified: with `raw` unset, `prepareMarshalledResponse`'s `!msg.IsValid()` branch returns the empty base buffer, so the HTTP client gets a 200 with an empty body. Mark that ceiling with a `ponytail:` comment.

**D-04 — No `filekv` change; dev mode gets no `raw`.**
`filekv`'s parser *could* resolve descriptors and render JSON, but teaching a generic valkeyrie store that "a key ending in `/config.json` means render the inner Any" puts config semantics in the wrong layer and duplicates the inserter's `dynamicpb` + `protojson` block. Dev users already have the materialized JSON on disk to `cat`. Name the ceiling and the upgrade path in the same `ponytail:` comment as D-03.

**D-05 — Do NOT register a JSON gRPC codec.** This is a correctness matter, not efficiency. See `key_links`.

**D-06 — buf needs `deps: [buf.build/googleapis/googleapis]`, not just an exclude.** Reproduced experimentally; see `key_links` and Task 1.

**D-07 — `generate.go` gets the `third_party` exclusion but NOT `-I=third_party`.** No proto outside `pb/` imports `google/api/*` (only `pb/protoconf/v1/protoconf.proto` will), so the include path buys nothing, while the exclusion is mandatory to stop a stray generated file.
</decisions>

<scope_fence>
IN scope: plain-HTTP access to `GetConfig`, and only what it needs to work.

OUT of scope — do NOT port from PR #496, do NOT touch:
- `GetJsonConfig` / `GetJsonConfigHttp` and the hand-rolled `/getJsonConfig` `net/http` handler — vanguard replaces both
- TLS changes of any kind, `OTEL_SDK_DISABLED` toggles, the noop stats handler, `conf.yaml`
- `inserter/`, the store health probe, `devserver/`, `server/`
- `agent/filekv/`, `agent/legacy.go`, `agent/api/proto/v1/*` (see D-01, D-04)
- `Test_cliCommand_Run/run_consul_server` — it hangs on this machine because a real Consul listens on 127.0.0.1:8500. Pre-existing and environmental. It is skipped in the verify command. Do not try to fix it.
- `/debug/pprof/heap` and friends 404ing: `agent.go` registers `/debug/pprof` as an *exact* pattern today, so subpaths already 404. No regression, not this plan's problem.
- `buf lint`'s "Category DEFAULT is deprecated, use STANDARD" warning — pre-existing, not in CI, leave it.
</scope_fence>

<toolchain_setup>
`protoc` and both plugins are installed and pinned but `$(go env GOPATH)/bin` is NOT on PATH in a fresh shell. A standalone `export PATH=...` statement was blocked by the sandbox in the parent task — prefix the invocation on one command line instead:

```
PATH="$PATH:$(go env GOPATH)/bin" protoc ...
```

Installed: protoc 34.0, protoc-gen-go v1.36.11, protoc-gen-go-grpc 1.6.2, buf 1.61.0.
</toolchain_setup>

<tasks>

<task type="auto">
  <name>Task 1: Vendored HttpBody, proto field, build plumbing, codegen, vanguard dependency</name>
  <files>third_party/google/api/httpbody.proto, buf.yaml, buf.lock, generate.go, pb/protoconf/v1/protoconf.proto, pb/protoconf/v1/protoconf.pb.go, pb/protoconf/v1/protoconf_grpc.pb.go, go.mod, go.sum</files>

  <read_first>
    - `third_party/google/api/httpbody.proto` — already on disk, the official googleapis file (Apache-2.0, 80 lines, unmodified), but **untracked**. Its `go_package` is `google.golang.org/genproto/googleapis/api/httpbody`, so protoc emits the correct Go import without generating Go for it.
    - `pb/protoconf/v1/protoconf.proto` — package `protoconf.v1`, `ConfigUpdate` currently has fields 1 (`value`) and 2 (`error`).
    - `buf.yaml` — `version: v1`, `build.excludes` currently `utils/testdata`, `examples`, `node_modules`. No `deps`, no `buf.lock`.
    - `generate.go` — one `go:generate` line whose `find` excludes `*pb/*` and `*utils/*` only.
    - `.github/workflows/lint.yml:32-46` — the `buf-breaking` job runs `buf breaking --against '.git#branch=main' --against-config buf.yaml`.
  </read_first>

  <action>
`git add third_party/google/api/httpbody.proto` — it exists on disk but is untracked, and everything below depends on it being in the tree. Do not modify its contents.

Add to `pb/protoconf/v1/protoconf.proto`: an `import "google/api/httpbody.proto";` alongside the existing imports, and a third field on `ConfigUpdate` named `raw`, number 3, of type `google.api.HttpBody`. Nothing else in that file changes. Do NOT touch `agent/api/proto/v1/protoconf_service.proto` (D-01).

Fix `buf.yaml` — this is mandatory, not cosmetic. A scratch reproduction confirmed `buf build` fails with `import "google/api/httpbody.proto": file does not exist` **both** with and without `third_party` in `build.excludes`, because the vendored file's module-relative path never matches the import path. Add `third_party` to `build.excludes` AND add a top-level `deps:` list containing `buf.build/googleapis/googleapis`, then run `buf dep update` to write `buf.lock`. Commit `buf.lock`. The same scratch reproduction confirms this combination makes `buf build` pass.

Fix `generate.go`: add `-not -path '*third_party/*'` to the `find` predicate so the vendored googleapis file is never handed to protoc for Go generation. Do NOT add `-I=third_party` to that line — no proto outside `pb/` imports `google/api/*`, so it buys nothing (D-07).

Regenerate. Use this command EXACTLY — the pb module is generated with `-I=pb` because `generate.go` deliberately excludes it, and `-I=third_party` is new:

    PATH="$PATH:$(go env GOPATH)/bin" protoc -I=pb -I=third_party --go_out=pb --go_opt=paths=source_relative --go-grpc_out=pb --go-grpc_opt=paths=source_relative protoconf/v1/protoconf.proto

This was dry-run in a scratch copy of `pb/` and `third_party/`: it produces **46 insertions / 32 deletions** in `protoconf.pb.go` and leaves `protoconf_grpc.pb.go` byte-identical. If your diff is hundreds of lines, the flags are wrong — stop and investigate rather than committing it. Do NOT hand-edit any `.pb.go` file. Confirm the generated file gained `httpbody "google.golang.org/genproto/googleapis/api/httpbody"` and `Raw *httpbody.HttpBody`, and that `grep -c File_protoconf_v1_protoconf_proto pb/protoconf/v1/protoconf.pb.go` is still non-zero (that symbol is what `server/server.go:285` resolves against).

Add the dependency with `go get connectrpc.com/vanguard` and let the resolver do its job — do not hand-edit `go.mod`. It resolves to v0.4.0. Its own go.mod requires protobuf v1.36.11 and grpc v1.79.3; this repo has v1.36.12 and v1.83.1, both newer, so it resolves cleanly. `google.golang.org/genproto/googleapis/api` is already an indirect dep and will be promoted to direct by the generated import. Finish with `go mod tidy` and confirm a second `go mod tidy` leaves no diff.

Commit message: `chore(proto): add ConfigUpdate.raw HttpBody, vendor googleapis, add vanguard`
  </action>

  <verify>
    <automated>go build ./... &amp;&amp; buf build &amp;&amp; buf breaking --against '.git#branch=main' --against-config buf.yaml &amp;&amp; go mod tidy &amp;&amp; git diff --exit-code go.mod go.sum</automated>
  </verify>

  <done>`third_party/google/api/httpbody.proto` is tracked; `ConfigUpdate` has `google.api.HttpBody raw = 3`; `protoconf.pb.go` is regenerated with a ~46/32 diff and still exports `File_protoconf_v1_protoconf_proto`; `buf build` and `buf breaking` pass with `deps` + `buf.lock`; `generate.go` skips `third_party`; `connectrpc.com/vanguard` is a direct dependency and `go mod tidy` is a no-op.</done>
</task>

<task type="tracer" tdd="true">
  <name>Task 2: End-to-end `curl` — fill `raw`, mount the transcoder, prove one HTTP GET</name>
  <files>agent/kv_agent_rollout_impl.go, agent/kv_agent_impl.go, agent/agent.go, agent/http_test.go</files>

  <read_first>
    - `agent/kv_agent_rollout_impl.go` — the `parseProtoconfValue(kvPair)` helper near the bottom of the file. This is where package-shared agent helpers live; put the new one beside it. Both agent files' `ConfigUpdate` is the SAME Go type (`kv_agent_rollout_impl.go` aliases `pb/protoconf/v1` as `protoconfservice`, `kv_agent_impl.go` as `protoconf_pb`), so one helper returning `*httpbody.HttpBody` serves both.
    - `agent/kv_agent_impl.go:96-126` and `agent/kv_agent_rollout_impl.go:278-308` — the two existing `GetConfig` bodies. Each already ends by returning `&ConfigUpdate{Value: result.Value}`.
    - `agent/agent.go:156-180` — `rpcServer` construction, the two `RegisterProtoconfServiceServer` calls, `grpc_prometheus.Register`, the three-line `mux`, and the orchestra `Conductor` that serves that mux on `config.HttpAddress`.
    - `agent/kv_agent_impl_test.go:33-80` — `newAny(msg)` and the `testServer(ctx, srv)` bufconn helper, plus the `dummykv` + `base64` + `proto.Marshal(&protoconfvalue.ProtoconfValue{...})` seeding pattern. Reuse all of it.
  </read_first>

  <behavior>
    - `GET /v1/config/some/config` against a store seeded with a primary config key and a sibling `config.json` returns 200, `Content-Type: application/json`, and a body byte-identical to the seeded `config.json` value — not base64, not `{"raw": ...}`.
    - A gRPC `GetConfig` for the same path still returns the Any in `value`, and now also returns `raw.data` equal to the seeded bytes with `raw.content_type == "application/json"`.
  </behavior>

  <action>
Add ONE shared helper next to `parseProtoconfValue` in `agent/kv_agent_rollout_impl.go`. It takes a context, a `store.Store`, the prefix, and the request path; reads `path.Join(prefix, configPath, "config.json")` via `store.Get` with `&store.ReadOptions{}`; returns `(nil, nil)` when the error `errors.Is` `store.ErrKeyNotFound` or the returned kvPair is nil; returns the store error otherwise; and on success returns an `*httpbody.HttpBody` with `ContentType` set to the literal `application/json` and `Data` set to the kvPair's `Value` bytes **unmodified** — no decode, no re-marshal, the inserter already wrote protojson there.

Document the helper with a short comment explaining WHY the agent cannot render this itself (the Any's descriptor is not in `protoregistry.GlobalTypes`), and add a `ponytail:` comment naming both ceilings from D-03/D-04: an absent `config.json` yields a nil HttpBody which vanguard renders as an empty 200 body, and dev-mode `filekv` has no `config.json` key at all — upgrade path is to render it in `filekv.Get`, or to 404 the REST route specifically, if that empty body ever bites.

Call the helper from BOTH `GetConfig` bodies, immediately before their final return, and set the result on the new `Raw` field of the returned `ConfigUpdate`. A helper error maps to `status.Error(codes.Internal, ...)` — a store outage must not silently degrade into an empty body. Do not restructure anything else in either method.

In `agent/agent.go`, extract the mux construction into a small unexported function taking the `*grpc.Server` and returning `(*http.ServeMux, error)`. It builds the transcoder via `vanguardgrpc.NewTranscoder(rpcServer, vanguard.WithRules(...))` with exactly one `*annotations.HttpRule`:
  - `Selector`: `protoconf.v1.ProtoconfService.GetConfig` — must match a real method or `NewTranscoder` errors at construction. `protoconf.v1` is the package declared on line 2 of `pb/protoconf/v1/protoconf.proto`; verify the selector resolves rather than assuming.
  - `Pattern`: an `HttpRule_Get` with `/v1/config/{path=**}`. The `**` capture spans slashes, and `path` names the `ConfigRequest.path` field.
  - `ResponseBody`: `raw`.
Handle the returned error with `errors.Join` in the house style; do not ignore it. Then build the `http.ServeMux`, keep the existing `/debug/pprof` and `/metrics` registrations verbatim, and add the transcoder at `/`.

Rewire `RunAgent` to call that function instead of building the mux inline, returning its error. The call site MUST stay where the current mux construction is — after both `RegisterProtoconfServiceServer` calls — because `vanguardgrpc.NewTranscoder` snapshots `server.GetServiceInfo()`. Also add one `logger.Warn` at that point recording that the HTTP config API is now served unauthenticated and without TLS on `config.HttpAddress`, so an operator who only hardened the gRPC port finds out.

Do NOT call `encoding.RegisterCodec(vanguardgrpc.NewCodec(&vanguard.JSONCodec{...}))` anywhere. See D-05 and `key_links` — it would make vanguard ask the gRPC handler for JSON, which fails on the unresolvable Any and breaks this feature outright.

Create `agent/http_test.go` covering the two behaviors above. Build a `dummykv` store, a `NewProtoconfKVAgent` over it with an empty-prefix `AgentConfig`, a bare `grpc.NewServer()` with `protoconf_pb.RegisterProtoconfServiceServer`, then the new mux function, then `httptest.NewServer(mux)` — plain HTTP/1.1 is correct here, no TLS and no `EnableHTTP2`, because vanguard rewrites the target request to HTTP/2 itself before handing it to the gRPC handler. Seed the primary key with a base64'd marshalled `ProtoconfValue` exactly as `kv_agent_impl_test.go` does, and seed `<path>/config.json` with a small literal JSON byte slice. Assert status, `Content-Type`, and byte-equality of the body against the seeded literal. If `Content-Type` comes back carrying parameters, compare with `mime.ParseMediaType` rather than loosening the assertion — the verbatim body assertion is the entire point of the HttpBody design and must not be weakened.

Commit message: `feat(agent): serve GetConfig over plain HTTP via vanguard transcoder`
  </action>

  <verify>
    <automated>go build ./... &amp;&amp; go test ./agent/ -count=1 -run 'HTTP|GetConfig'</automated>
  </verify>

  <done>A plain HTTP/1.1 `GET /v1/config/<path>` against the agent's real mux returns 200, `application/json`, and the seeded `config.json` bytes verbatim; both `GetConfig` impls populate `raw` from one shared helper; the transcoder is constructed after service registration and its error is handled; no JSON codec is registered anywhere.</done>
</task>

<task type="auto" tdd="true">
  <name>Task 3: Edge cases — 404, absent config.json, /metrics precedence, rollout agent, full verification</name>
  <files>agent/http_test.go, agent/kv_agent_impl_test.go, agent/kv_agent_rollout_impl_test.go</files>

  <read_first>
    - `agent/kv_agent_rollout_impl_test.go` — the existing rollout table test: `newProtoconfKVAgentRollout` + `testServer` + `dummykv` + `inserter.NewProtoconfInserter(testdata.SmallTestDir(), kvStore)` setup, and the existing `GetConfig` test added in the parent task.
    - `agent/http_test.go` — the harness written in Task 2; extend it, do not create a second file.
    - `agent/kv_agent_impl_test.go` — the existing `TestProtoconfKVAgent_GetConfig` three-case test; extend it rather than duplicating its scaffolding.
  </read_first>

  <behavior>
    - `GET /v1/config/does/not/exist` returns 404. (`codes.NotFound` from `GetConfig` maps through connect to HTTP 404.)
    - `GET /metrics` still returns 200 from the Prometheus handler, not the transcoder, even though the transcoder owns `/`.
    - A path whose primary key exists but whose `config.json` sibling does not: the gRPC `GetConfig` still succeeds with a non-nil `value` and a nil `raw`; the HTTP GET returns 200 with an empty body. Assert this explicitly — it is the accepted ceiling from D-03, and pinning it stops a future change from silently turning it into a 500.
    - `ProtoconfKVAgentRollout.GetConfig` populates `raw` from `prefix/path/config.json` after a real `inserter.InsertConfig`, which writes both `config.data` and `config.json`.
  </behavior>

  <action>
Extend `agent/http_test.go` with the 404, `/metrics`-precedence, and absent-`config.json` cases as subtests of the Task 2 harness — reuse the one `httptest.Server`, do not stand up three more. For `/metrics`, asserting 200 plus a body containing a Prometheus exposition marker is enough; the point is only that the exact pattern beats the `/` subtree pattern. Go's `http.ServeMux` documents most-specific-pattern-wins, but assert it rather than trusting it, since mounting a catch-all is the risky part of this change.

Extend `agent/kv_agent_impl_test.go`'s existing `GetConfig` test with the raw-present and raw-absent gRPC-level assertions, in its existing style.

Extend `agent/kv_agent_rollout_impl_test.go`'s existing `GetConfig` test to assert `raw.content_type` is `application/json` and that `raw.data` parses as JSON describing the inserted config — the inserter writes it, so this proves the helper's key shape matches the production layout rather than only the hand-seeded one. Keep the test sequential; a unary call has no delivery race.

Then run the full verification below and record the output in the summary.

Commit message: `test(agent): cover REST 404, empty raw, metrics precedence, rollout raw`
  </action>

  <verify>
    <automated>go build ./... &amp;&amp; go test ./agent/... ./test/... ./server/... -count=1 -skip Test_cliCommand_Run &amp;&amp; go mod tidy &amp;&amp; git diff --exit-code go.mod go.sum</automated>
  </verify>

  <done>REST returns 404 for a missing config; `/metrics` still wins over the catch-all; the absent-`config.json` behavior is pinned by a test at both the gRPC and HTTP levels; the rollout agent's `raw` is proven against a real `inserter.InsertConfig`; the full verify command passes and `go mod tidy` leaves no diff.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| unauthenticated HTTP client → agent `http_address` | NEW. This port previously served only pprof and metrics; it now serves config reads |
| HTTP client → `ConfigRequest.path` | Attacker-controlled path is joined onto a store key (and, in dev mode, a filesystem path) |
| KV store → agent | `config.json` bytes are written to the HTTP response verbatim, with an agent-chosen Content-Type |

## STRIDE Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation Plan |
|-----------|----------|-----------|----------|-------------|-----------------|
| T-ggd-01 | Information disclosure | vanguard transcoder on `config.HttpAddress` | high | accept | The brief mandates plain HTTP and forbids TLS changes. The exposure is real: the gRPC port can carry TLS and a bearer-token interceptor while this port carries neither, so an operator who hardened gRPC may not realise configs are now readable unauthenticated. Task 2 emits a startup `logger.Warn` naming the address so this cannot happen silently. Authn/authz on the HTTP read path is a deliberate follow-up |
| T-ggd-02 | Information disclosure | `ConfigRequest.path` → `prefix/path/config.json` store key | medium | mitigate | No new mitigation needed, but confirm it holds: dev-mode `filekv.Get` already rejects any key where `key != filepath.ToSlash(filepath.Clean(key))` before building a path, which is what stops `../` escaping `protoconfRoot`. The new sibling-key lookup goes through that same `store.Get`. KV backends (consul/etcd/zookeeper) are not filesystem-backed |
| T-ggd-03 | Tampering | `config.json` bytes written verbatim to the HTTP response | medium | mitigate | `Content-Type` is set by the agent to the literal `application/json`, never taken from stored data, so a poisoned store value cannot select `text/html` and turn a config read into stored XSS. Task 2 hardcodes it; do not make it configurable |
| T-ggd-04 | Denial of service | transcoder mounted at `/` shadowing operational endpoints | medium | mitigate | Task 3 asserts `/metrics` still returns 200 from the Prometheus handler after the catch-all is mounted, rather than trusting `http.ServeMux` precedence |
| T-ggd-05 | Denial of service | store error on the `config.json` lookup silently degrading the response | low | mitigate | D-03: only `store.ErrKeyNotFound` yields a nil `raw`; every other store error fails the RPC with `codes.Internal` |
| T-ggd-06 | Tampering | `connectrpc.com/vanguard` is alpha per its own README | medium | accept | Buf-maintained, Apache-2.0, pinned in `go.sum`. The blast radius is the HTTP port only — gRPC clients hit `rpcServer` on `grpc_address` and never traverse the transcoder. API instability is the real risk; a major-version bump will need a re-read of `WithRules` and the HttpBody path |
| T-ggd-SC | Tampering | npm/pip/cargo installs | n/a | accept | No package-manager installs. The one new Go module (`connectrpc.com/vanguard`, buf's own org) is added via `go get` with `go.sum` pinning; protoc and both plugins are already installed and pinned |
</threat_model>

<verification>
```
go build ./...
go test ./agent/... ./test/... ./server/... -count=1 -skip Test_cliCommand_Run
go mod tidy && git diff --exit-code go.mod go.sum
buf build
buf breaking --against '.git#branch=main' --against-config buf.yaml
```

`Test_cliCommand_Run/run_consul_server` is skipped deliberately: it hangs on this dev machine because a real Consul is listening on 127.0.0.1:8500. Pre-existing and environmental — do not attempt to fix it.

Manual spot checks:
- `git ls-files third_party/google/api/httpbody.proto` is non-empty (the file was untracked before this task).
- `grep -c File_protoconf_v1_protoconf_proto pb/protoconf/v1/protoconf.pb.go` is non-zero, proving the `-I=pb` generation was used and `server/server.go:285` still resolves.
- `git show --stat HEAD~2 -- pb/protoconf/v1/protoconf.pb.go` shows roughly 46 insertions / 32 deletions, not hundreds.
- `grep -rn 'RegisterCodec' agent/` returns nothing (D-05).
</verification>

<success_criteria>
- `curl http://<http-address>/v1/config/<path>` returns 200, `Content-Type: application/json`, and the config JSON byte-for-byte — asserted by an `httptest`-based test, not by eyeball.
- A missing config returns 404 over HTTP.
- `/metrics` still resolves to the Prometheus handler with the transcoder mounted at `/`.
- `ConfigUpdate` gained exactly one field, `google.api.HttpBody raw = 3`, in `pb/protoconf/v1/protoconf.proto` only; the legacy `agent/api/proto/v1` proto is untouched.
- Both agent `GetConfig` impls fill `raw` from one shared helper reading `prefix/path/config.json`; a missing sibling leaves `raw` unset without failing the RPC, and that behavior is pinned by a test.
- No JSON gRPC codec is registered; the agent never attempts to protojson-marshal `ConfigUpdate.value`.
- No `.pb.go` file was hand-edited; `buf build`, `buf breaking`, and `go mod tidy` are all clean.
- `GetJsonConfig`, `GetJsonConfigHttp`, TLS changes, `conf.yaml`, and inserter/filekv/devserver/legacy edits are all absent.
</success_criteria>

<output>
Create `.planning/quick/260902-ggd-serve-getconfig-over-plain-http-via-conn/260902-ggd-SUMMARY.md` when done
</output>
