# Phase 6: Token Auth & Script Security - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-03-28
**Phase:** 06-token-auth-script-security
**Areas discussed:** Auth token format, Credential forwarding, Script path validation, Auth configuration
**Mode:** Auto (--auto flag, all recommended defaults selected)

---

## Auth Token Format

| Option | Description | Selected |
|--------|-------------|----------|
| Static API key (bearer token) | Simple shared secret via gRPC metadata, constant-time comparison | :heavy_check_mark: |
| JWT with signature validation | Industry standard, supports claims/expiry, more complex setup | |
| mTLS client certificates | Strongest auth, already deferred to v2 (SECR-08) | |

**User's choice:** [auto] Static API key (bearer token)
**Notes:** Matches PROJECT.md decision "Token-based auth over mTLS — simpler to implement and forward to scripts as env vars". JWT deferred as overkill for current needs.

---

## Credential Forwarding

| Option | Description | Selected |
|--------|-------------|----------|
| Raw token as PROTOCONF_AUTH_TOKEN env var | Scripts receive the token directly, can validate/forward as needed | :heavy_check_mark: |
| Parsed claims as multiple env vars | Would require JWT; not applicable for static tokens | |
| Token file path | Write token to temp file, pass path — adds complexity | |

**User's choice:** [auto] Raw token as PROTOCONF_AUTH_TOKEN env var
**Notes:** Also forwards script_metadata from ConfigMutationRequest as PROTOCONF_SCRIPT_METADATA. Fixes existing bug: missing `=` in PROTOCONF_COMPILER_ADDR env var.

---

## Script Path Validation

| Option | Description | Selected |
|--------|-------------|----------|
| Validate at server startup | Fail fast if scripts missing/non-executable, defense-in-depth recheck at runtime | :heavy_check_mark: |
| Validate per-request only | Lazy validation, allows script hot-swapping but delays error discovery | |
| No validation | Current behavior — exec.Command fails at runtime with opaque error | |

**User's choice:** [auto] Validate at server startup
**Notes:** Aligns with project core value "no runtime surprises". Includes path traversal check (reject `..` in paths).

---

## Auth Configuration

| Option | Description | Selected |
|--------|-------------|----------|
| --auth-token flag (static secret) | Consistent with existing flag pattern, simple to implement | :heavy_check_mark: |
| Auth config file (JSON/YAML) | More flexible, supports rotation — overkill for v1 | |
| Environment variable only | Secure (not in CLI history) but inconsistent with current flag patterns | |

**User's choice:** [auto] --auth-token flag (static secret)
**Notes:** Follows established pattern from cliConfig/newFlagSet. Auth implemented as gRPC unary interceptor for clean separation.

---

## Claude's Discretion

- Interceptor implementation details (file placement, ConstantTimeCompare wrapping)
- Warning message format consistency with Phase 5 TLS warnings
- Whether to support both unary and stream interceptors (mutation service is unary-only)

## Deferred Ideas

- mTLS (SECR-08) — v2
- Role-based auth (SECR-09) — v2
- JWT/OIDC — future if needed
- Agent auth — agent is read-only, not in scope
