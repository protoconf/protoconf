# Phase 5: TLS Support - Context

**Gathered:** 2026-03-28
**Status:** Ready for planning

<domain>
## Phase Boundary

Add TLS support for gRPC servers and clients. The agent already has proto-defined TLS config fields (`AgentConfig.TLSConfig`, `insecure` bool) but no Go implementation wiring them up. The mutation server and mutate CLI have no TLS config at all. When no TLS is configured, servers must log a visible warning. Existing insecure-mode usage continues without flag changes.

</domain>

<decisions>
## Implementation Decisions

### TLS Helper Package
- **D-01:** Create a shared TLS helper (e.g., `utils/tls.go` or similar) that constructs `tls.Config` from cert/key/CA file paths. Both agent and server will use this to avoid duplicating the `tls.LoadX509KeyPair` / `x509.CertPool` logic.
- **D-02:** The helper accepts file paths (not inline text) for the common case. The agent's proto `TLSConfig` has both `key_file`/`key_text` oneofs — the helper should handle both forms since the agent proto already defines them.

### Agent (gRPC Server) — SECR-01
- **D-03:** Wire the existing `AgentConfig.TlsConfig` proto fields to actual `grpc.Creds(credentials.NewTLS(...))` server option in `agent/agent.go` at `grpc.NewServer()`.
- **D-04:** Wire the existing `AgentConfig.Insecure` bool — when true (or when no TLS config provided), skip TLS. When false and TLS config is present, enforce TLS.
- **D-05:** The agent already has flags auto-generated from proto via `libprotoconf.PopulateFlagSet` — no new flag registration needed, just Go wiring.

### Mutation Server (gRPC Server) — SECR-01
- **D-06:** Add `--tls-cert` and `--tls-key` flags (and optional `--tls-ca` for client cert verification) to `server/server.go`'s `cliConfig` and `newFlagSet()`.
- **D-07:** Wire these flags to `grpc.Creds(credentials.NewTLS(...))` server option at `grpc.NewServer()` in `server/server.go`.

### Mutate CLI (gRPC Client) — SECR-02
- **D-08:** Add `--tls-cert`, `--tls-key`, `--tls-ca`, and `--insecure` flags to `mutate/mutate.go`'s CLI config.
- **D-09:** When TLS flags are provided, use `credentials.NewTLS(...)` instead of `insecure.NewCredentials()` for `grpc.NewClient()`.
- **D-10:** When `--insecure` (or no TLS flags), keep current `insecure.NewCredentials()` behavior.

### Insecure Warning — SECR-03
- **D-11:** At server startup (both agent and mutation server), if no TLS config is provided, log `slog.Warn("gRPC server running without TLS — connections are not encrypted")`.
- **D-12:** The warning is informational — no behavioral change, no failure, no env var to suppress. This matches the success criteria: "logs a visible warning."

### Out of Scope
- **D-13:** DevServer does NOT get TLS support — it's a local development tool. Adding TLS would add friction without benefit.
- **D-14:** Example clients in `examples/` are NOT modified — they are illustrative code, not production.
- **D-15:** Compiler service gRPC connection (`compiler/command.go`) does NOT get TLS — it's a local in-process connection.

### Claude's Discretion
- Exact file placement of the TLS helper (could be `utils/tls.go`, `internal/tls/`, or a new `tls/` package)
- Whether to use `AgentConfig_TLSConfig` type directly in the helper or abstract to a simpler struct
- Whether `--tls-ca` flag name should be `--tls-ca` or `--tls-ca-cert` — prefer `--tls-ca` for brevity

</decisions>

<canonical_refs>
## Canonical References

- `agent/config/v1/agent_config.proto` — Source of truth for TLS config structure (TLSConfig message with key/cert/ca oneofs)
- `agent/command.go` — Shows how `libprotoconf.PopulateFlagSet` auto-generates flags from proto
- `server/server.go` — Mutation server flag setup at `newFlagSet()` and server creation
- `mutate/mutate.go` — Mutate CLI gRPC client setup with `grpc.NewClient()`

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `agent/config/v1/agent_config.proto` TLSConfig message — already defines cert/key/CA with file+text oneofs
- `libprotoconf.PopulateFlagSet` — auto-generates CLI flags from proto fields, already used by agent
- Phase 1 migration to `grpc.WithTransportCredentials(insecure.NewCredentials())` — all call sites already use the modern API

### Established Patterns
- Agent uses proto-defined config with libprotoconf for flag generation
- Mutation server uses manual `flag.FlagSet` with `cliConfig` struct
- Mutate CLI uses manual `flag.FlagSet` with `cliConfig` struct
- gRPC server options are passed inline at `grpc.NewServer(opts...)`
- gRPC client credentials are passed inline at `grpc.NewClient(addr, opts...)`

### Integration Points
- `agent/agent.go:128` — `grpc.NewServer()` call site for agent
- `server/server.go:120` — `grpc.NewServer()` call site for mutation server
- `mutate/mutate.go:210` — `grpc.NewClient()` call site for mutate CLI
- All three are the primary insertion points for TLS credentials

</code_context>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches for Go TLS with gRPC.

</specifics>

<deferred>
## Deferred Ideas

- mTLS support (SECR-08) — deferred to v2 per REQUIREMENTS.md
- TLS for KV store connections (agent's `store_tls` field) — related but not in SECR-01/02/03 scope; could be a future phase

</deferred>

---

*Phase: 05-tls-support*
*Context gathered: 2026-03-28*
