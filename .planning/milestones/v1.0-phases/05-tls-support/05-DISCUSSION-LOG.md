# Phase 5: TLS Support - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-03-28
**Phase:** 05-tls-support
**Areas discussed:** TLS helper location, Server TLS config, Client-side TLS, Insecure warning, DevServer treatment
**Mode:** --auto (all decisions auto-selected)

---

## TLS Helper Location

| Option | Description | Selected |
|--------|-------------|----------|
| Shared helper package | Create utils/tls.go or similar with reusable TLS config construction | :heavy_check_mark: |
| Inline per component | Duplicate TLS setup in each server/client | |

**User's choice:** [auto] Shared helper package (recommended default)
**Notes:** Agent, mutation server, and mutate CLI all need TLS config loading — a shared helper avoids 3x duplication.

---

## Server TLS Config Approach

| Option | Description | Selected |
|--------|-------------|----------|
| Add flags to server's cliConfig | --tls-cert, --tls-key, --tls-ca flags via flag.FlagSet | :heavy_check_mark: |
| Adopt agent's proto-defined config | Move server to proto-based config like agent | |

**User's choice:** [auto] Add flags to server's cliConfig (recommended default)
**Notes:** Proto-defined CLI config is Phase 7/8 scope. Keep server's existing flag pattern consistent for now.

---

## Client-side TLS

| Option | Description | Selected |
|--------|-------------|----------|
| mutate CLI gets TLS flags | --tls-cert/--tls-key/--tls-ca/--insecure for mutate CLI | :heavy_check_mark: |
| All clients get TLS | Also add to compiler service, examples | |

**User's choice:** [auto] mutate CLI only (recommended default)
**Notes:** Compiler service uses in-process bufconn. Examples are illustrative. Only mutate CLI is a real network client to mutation server.

---

## Insecure Warning Mechanism

| Option | Description | Selected |
|--------|-------------|----------|
| slog.Warn at server startup | Log warning when no TLS configured | :heavy_check_mark: |
| Require explicit --insecure flag | Fail if neither TLS nor --insecure is set | |

**User's choice:** [auto] slog.Warn at startup (recommended default)
**Notes:** Agent already has an `insecure` bool in proto. Success criteria says "logs a visible warning" not "requires acknowledgment."

---

## DevServer Treatment

| Option | Description | Selected |
|--------|-------------|----------|
| No TLS for devserver | DevServer is local-only, skip TLS | :heavy_check_mark: |
| Add TLS to devserver | Full TLS support for devserver too | |

**User's choice:** [auto] No TLS for devserver (recommended default)
**Notes:** DevServer is explicitly for local development. Adding TLS friction has no benefit.

---

## Claude's Discretion

- Exact file placement of TLS helper
- Whether to use agent's `AgentConfig_TLSConfig` proto type or abstract to simpler struct
- Flag naming (`--tls-ca` vs `--tls-ca-cert`)

## Deferred Ideas

- mTLS support (SECR-08) — v2 per REQUIREMENTS.md
- TLS for KV store connections (`store_tls` field)
