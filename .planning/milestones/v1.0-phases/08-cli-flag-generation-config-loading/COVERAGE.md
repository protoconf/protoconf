# Phase 08 — External API Coverage

No external API integration: CLI config precedence is resolved in-process using vendored libprotoconf and Go stdlib only — no external service, SDK, endpoint, webhook or MCP server is called.

## Why the detector fired

The `plan:pre` detector returned `detected: true` on two signals, both re-read against the phase scope and both false positives:

| Signal | Where it came from | Verdict |
|---|---|---|
| noun `api` | The prose phrase "the `ConfigLayerer` API" in `08-05-PLAN.md` / `08-06-PLAN.md`, describing an internal Go package's own exported surface. | False positive — an in-repo package API, not an external one. |
| verb `wire` + noun `grpc` | The reversibility rationale sentence "no proto, wire-format or gRPC service definition changes" in both plans, which asserts the ABSENCE of any such change. | False positive — a negation matched as if it were an integration. |

Protoconf does serve gRPC APIs (`agent`, `server`), but this phase touches none of them: its scope is `command/configfile.go` plus the five `Command()` factories, and no `.proto` file, service definition, or wire format is modified. Both plans gate on `git diff --exit-code go.mod go.sum`, so no new dependency — external or otherwise — can enter through this work.

No capability matrix is applicable. Per the API-coverage contribution, a reasoned declaration stands in place of a matrix and satisfies the `api-coverage.verify-pre` seal gate.
