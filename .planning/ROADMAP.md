# Roadmap: Protoconf

## Milestones

- ✅ **v1.0 Quality & Consistency Overhaul** — Phases 1-10 (shipped 2026-03-31)
- ✅ **v2.0 Compiler Startup Performance** — Phases 11-15 (shipped 2026-09-10)

## Phases

**Phase Numbering:**

- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)
- Phase numbering is continuous across milestones — v2.0 starts at Phase 11, not Phase 1

Decimal phases appear between their surrounding integers in numeric order.

<details>
<summary>✅ v1.0 Quality & Consistency Overhaul (Phases 1-10) — SHIPPED 2026-03-31</summary>

- [x] Phase 1: Deprecated API Migrations (1/1 plans) — completed 2026-03-23
- [x] Phase 2: os.Exit Refactoring (2/2 plans) — completed 2026-03-27
- [x] Phase 3: Observability & Global State Cleanup (2/2 plans) — completed 2026-03-27
- [x] Phase 4: Dead Code Removal (1/1 plans) — completed 2026-03-28
- [x] Phase 5: TLS Support (2/2 plans) — completed 2026-03-28
- [x] Phase 6: Token Auth & Script Security (2/2 plans) — completed 2026-03-31
- [x] Phase 7: Proto-Defined CLI Configs (1/1 plans) — completed 2026-03-28
- [x] Phase 8: CLI Flag Generation & Config Loading (6/6 plans) — completed 2026-09-01
- [x] Phase 9: Unit Test Coverage & Infrastructure (4/4 plans) — completed 2026-03-31
- [x] Phase 10: Placeholder Fixes & Integration Tests (2/2 plans) — completed 2026-03-31

Full phase detail archived under `.planning/milestones/v1.0-phases/`.

</details>

<details>
<summary>✅ v2.0 Compiler Startup Performance (Phases 11-15) — SHIPPED 2026-09-10</summary>

**Milestone Goal:** `protoconf compile` completes in under 200ms on a repository of any size, by parsing and linking only the protos a config actually reaches — without silently breaking the consumers that currently assume the registry is complete.

- [x] Phase 11: Concurrency-Safe Lazy Registry Core (5/5 plans) — completed 2026-09-07
- [x] Phase 12: Growable Resolver Views & Race Safety (4/4 plans) — completed 2026-09-08
- [x] Phase 13: Exact Symbol Index & Shared Type-URL Resolution (4/4 plans) — completed 2026-09-08
- [x] Phase 14: Non-Compiler Consumer Correctness (9/9 plans) — completed 2026-09-08
- [x] Phase 15: Verification, Decision & Gate Flip (3/3 plans) — completed 2026-09-09

Full phase detail archived under `.planning/milestones/v2.0-ROADMAP.md` and `.planning/milestones/v2.0-phases/`.

</details>

## Progress

| Milestone | Phases | Plans | Status | Shipped |
|-----------|--------|-------|--------|---------|
| v1.0 Quality & Consistency Overhaul | 1-10 | 23/23 | Complete | 2026-03-31 |
| v2.0 Compiler Startup Performance | 11-15 | 25/25 | Complete | 2026-09-10 |

Next milestone not yet scoped. Run `/gsd-new-milestone` to begin.
