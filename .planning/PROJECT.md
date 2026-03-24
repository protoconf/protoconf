# Protoconf — Quality & Consistency Overhaul

## What This Is

Protoconf is a configuration management tool that uses Protocol Buffers as schema and Starlark as the configuration language. It compiles Starlark configs into materialized protobuf, distributes them via KV stores (Consul, etcd, ZooKeeper, Kubernetes ConfigMaps), and serves them to applications via gRPC streaming. This milestone focuses on comprehensive quality improvements: testing, consistency, security hardening, and modernizing deprecated patterns across the entire codebase.

## Core Value

Every component must be testable, consistent, and free of runtime surprises — no panics in production code, no os.Exit in libraries, no deprecated APIs, and proper test coverage across all packages.

## Requirements

### Validated

- ✓ Starlark-to-protobuf compilation pipeline — existing
- ✓ Multi-backend KV store distribution (Consul, etcd, ZooKeeper, ConfigMaps, file) — existing
- ✓ gRPC agent with streaming config subscriptions — existing
- ✓ Mutation server with pre/post script support — existing
- ✓ Module system for external protobuf dependencies — existing
- ✓ Dev server combining agent + compiler + mutation — existing
- ✓ OpenTelemetry + Prometheus observability — existing
- ✓ Starlark format command (`protoconf fmt`) — existing
- ✓ Rollout-aware config insertion — existing
- ✓ Proto validation (protovalidate + Starlark validators) — existing

### Active

- [ ] Comprehensive test coverage across all packages
- [x] Refactor os.Exit calls in library code to error returns — Validated in Phase 2: os.Exit Refactoring
- [ ] Extract shared OTel bootstrap to common package
- [x] Migrate all deprecated gRPC APIs (WithInsecure, v1alpha reflection) — Validated in Phase 1: Deprecated API Migrations
- [ ] Migrate from jhump/protoreflect/dynamic to dynamicpb
- [ ] Add TLS support for gRPC connections
- [ ] Token-based auth with credential forwarding to pre/post scripts
- [ ] Proto-defined CLI configuration (research + implement)
- [ ] Fix known bugs (missing `=` in env var, dummykv.Exists always true)
- [ ] Remove dead code and unnecessary init functions
- [ ] Fix global mutable state issues (Starlark resolver settings, mutate package)

### Out of Scope

- KV store unimplemented method implementations — panics are intentional interface stubs; they signal gaps if methods become needed
- Full CLI framework migration (mitchellh/cli to cobra) — the real problem is configs not being proto-defined, which is in scope
- Mobile or web client SDKs — focus is backend quality
- New feature development — this milestone is purely quality/consistency

## Context

- Brownfield Go project with ~15 packages, serving as a configuration management platform
- Codebase has grown organically with inconsistencies: mixed error handling (os.Exit vs error returns), duplicate code (OTel setup), deprecated APIs still in use
- Several packages have zero test files: mutate/, devserver/, fmt/, command/, KV stores
- Existing tests have placeholder assertions and TODO comments providing false coverage confidence
- The project uses mitchellh/cli (maintenance mode) but the deeper issue is that CLI configurations should be defined as protobuf messages and CLI flags generated from those definitions
- Agent already self-configures via protobuf (agent/config/v1/agent_config.proto) — this pattern should be extended to all components
- Pre/post mutation scripts need auth credentials forwarded as environment variables for git operations

## Constraints

- **Tech stack**: Go 1.22+, must maintain backward compatibility with existing config repos
- **Proto compatibility**: Cannot break existing protobuf wire formats or gRPC service definitions
- **KV store interface**: Must remain compatible with valkeyrie store.Store interface
- **Build**: CGO_ENABLED=0, must produce static binaries
- **Testing**: Must not break existing CI (GitHub Actions with Codecov)

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Keep KV store panic stubs | Intentional interface satisfaction; panics signal future needs | — Pending |
| Token-based auth over mTLS | Simpler to implement and forward to scripts as env vars | — Pending |
| Proto-defined CLI configs | Consistency with protoconf's own philosophy; agent already does this | — Pending |
| Research proto-to-CLI generation | Need to find the right approach before committing to implementation | — Pending |
| Migrate jhump/protoreflect to dynamicpb | Official package is the recommended replacement | — Pending |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd:transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd:complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-03-24 after Phase 2 completion*
