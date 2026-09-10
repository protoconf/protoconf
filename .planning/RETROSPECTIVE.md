# Project Retrospective

*A living document updated after each milestone. Lessons feed forward into future planning.*

## Milestone: v2.0 — Compiler Startup Performance

**Shipped:** 2026-09-10
**Phases:** 5 | **Plans:** 25 | **Commits:** 190

### What Was Built

- On-demand, singleflight-guarded proto parsing wired end to end through the descriptor registry, parser, and compiler construction path. A config that reaches 5 protos now loads 5, not the whole tree.
- Growable resolver views that register files in place, replacing the construction-time snapshot and its O(everything-loaded-so-far) rebuild.
- An exact symbol-to-declaring-file index built by parsing without linking, persisted content-keyed under `.protoconf_cache`, backing the third tier of one shared resolution chain that every type-URL lookup delegates to.
- Lazy construction across all six registry consumers: compiler, mutation server, inserter, agent filekv, `mutate`, and the reflection UI walk.
- Two CI gates that assert rather than measure. The allocation-scaling ratio dropped from 7.24x to 1.02x, and a 160ms wall-clock budget runs in its own non-race step.

### What Worked

- **Deleting the fallback instead of hiding it behind a warning.** The whole-tree eager path was removed outright in Phase 13, at a one-way checkpoint taken only after two phases of evidence showed no uncovered resolution case. Leaving it reachable would have preserved the exact 4.6-second cost the milestone existed to remove.
- **One shared resolution chokepoint.** Routing every lookup through a single tiered chain meant one place to fix and one place to observe. The per-consumer lookup pattern is what let a mutable-config load silently bypass resolution in the first place.
- **Demonstrating that a test can fail before citing it as a control.** Phase 14 shipped a traversal test that could not fail, and two threat-model rows then cited it as a mitigation. The correction was to record the red failing output verbatim in each commit body, and verification reproduced it independently against the pre-fix commits.
- **Calibrating thresholds from CI observations, never from a laptop.** The 160ms budget came from two live runner measurements. A laptop-derived number either never fires or fires constantly.

### What Was Inefficient

- **Verification digests are fragile under later cleanup.** All five phases went stale at once when a lint and deprecation sweep touched 38 files. None of that changed behavior, but every phase then required an override to close. Verification coverage keyed to file digests treats formatting churn as a behavioral risk.
- **Deferred items outlived their findings.** Three items describing pre-existing vet findings stayed open across two phases, then all three turned out to be fixed by unrelated cleanup. Nothing rechecked them until milestone close.
- **A phase split that made criteria unobservable had to be merged.** Building the symbol index and consulting it were originally two phases. The index phase's success criteria could only assert that an artifact existed, since nothing read it until the next phase. Merging them anchored every criterion in an observable round-trip.

### Patterns Established

- A test cited as a security control must be demonstrated capable of failing, with the red output recorded, before its guard lands.
- Performance thresholds are calibrated from the environment that enforces them.
- A deliberate behavior removal gets written into the changelog and readme with the operator's remedy named, not quietly absorbed.
- Race tests prove their own failure-detection capability by temporary lock removal rather than assuming it.

### Key Lessons

1. Removing a slow fallback is only safe after evidence rules out uncovered cases, but keeping it "just in case" guarantees the cost you set out to eliminate stays reachable.
2. A resolution path that exists in more than one place will diverge. Consolidate first, then optimize.
3. Deferred items need a recheck trigger. Without one they accumulate as debt that may already be paid.
4. Substituting a synthetic corpus for a real one is fine, but the substitution and its unsourced sizing multiplier belong in the record, not in the presentation of the number.

### Cost Observations

- 25 plans across 5 phases in 6 days, 2026-09-04 to 2026-09-09.
- 73 Go files changed, roughly 6,450 lines added and 294 removed.
- Model profile: balanced.

---

## Cross-Milestone Trends

### Process Evolution

| Milestone | Phases | Plans | Key Change |
|-----------|--------|-------|------------|
| v1.0 Quality & Consistency Overhaul | 10 | 23 | Established proto-defined CLI config and real test coverage |
| v2.0 Compiler Startup Performance | 5 | 25 | Moved from measuring performance to asserting it in CI |

### Cumulative Quality

| Milestone | Whole-repo `go vet` | Race suite | CI performance gates |
|-----------|---------------------|------------|----------------------|
| v1.0 | findings present | green | none |
| v2.0 | silent | green | 2 (alloc ratio, wall clock) |
