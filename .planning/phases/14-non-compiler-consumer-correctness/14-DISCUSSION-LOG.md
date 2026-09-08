# Phase 14: Non-Compiler Consumer Correctness - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-09-08
**Phase:** 14-non-compiler-consumer-correctness
**Areas discussed:** Lazy flip blast radius, Reflection completeness, CONS-04 failure shape, How criteria 4 & 5 get proven

---

## Area Selection

| Option | Description | Selected |
|--------|-------------|----------|
| Lazy flip blast radius | Which of the four consumers construct via `NewLazyModuleService` | ✓ |
| Reflection completeness | What reflection and `MutateConfig` see once `LocalResolver` is a near-empty snapshot | ✓ |
| CONS-04 failure shape | What `GenReflectionUI` does with an unresolvable mutable config | ✓ |
| How 4 & 5 get proven | Race-test shape, and what makes "proportional to demand" concrete | ✓ |

**User's choice:** All four areas.

---

## Lazy Flip Blast Radius

### Q1 — Which consumers construct via `NewLazyModuleService`?

| Option | Description | Selected |
|--------|-------------|----------|
| All four (Recommended) | inserter, mutation server, agent/filekv, mutate CLI all lazy; `mod sync` stays eager. Uniform, no carve-out, every consumer stops paying the 4.6s whole-tree parse | ✓ |
| The three criteria name | inserter, filekv, server lazy; `mutate` CLI stays eager as a one-shot CLI doing a single `FindMessageByName`. Smaller blast radius, one consumer permanently on the old path | |
| Long-lived only | server + filekv lazy; inserter and mutate stay eager. Smallest change, but criterion 1 becomes unassertable and would need a roadmap amendment | |

**User's choice:** All four.
**Notes:** Grounding presented before the question — the flip is a one-call change per site because `GetProtoRegistry`'s lazy branch already sets `ImportPaths` and `CacheDir`, which is exactly what arms the Phase 13 scan and index tiers. Also established before asking: ROADMAP criteria 1, 2 and 5 are unobservable while these consumers stay eager, so some flip was structurally required.

### Q2 — When should an unresolvable type surface on a long-lived consumer?

| Option | Description | Selected |
|--------|-------------|----------|
| At first request, loudly | No startup validation pass. Phase 13 D-02's hard error returned to caller and logged. Preserves criterion 5 exactly | ✓ |
| Startup smoke-check | Resolve every config the process is already committed to serving, bounded by existing work not repo size | |
| Startup validates everything | Pre-resolve the whole repo at startup. Loudest, most familiar, but contradicts criterion 5 outright | |

**User's choice:** At first request, loudly.
**Notes:** The tradeoff was stated explicitly — under eager, a missing type failed at construction; under lazy the same repo starts clean and fails at first request. Accepted deliberately. Recorded in CONTEXT as D-02, with the clarification that this does not soften CONS-04: `GenReflectionUI` reports because it already walks `mutable_config/` on a ticker, not because a validation pass was added.

---

## Reflection Completeness

### Q1 — What should gRPC reflection see on a lazy mutation server?

| Option | Description | Selected |
|--------|-------------|----------|
| Everything (reuse discovery) | Keep `Init()`'s already-paid `discoveryRegistry` alive and hand its files resolver to reflection. Zero regression, no extra parse. Requires scoping criterion 5's counter to the serving registry | ✓ |
| Only what was demanded | Leave reflection on the growable `FilesResolver`. Purest read of criterion 5, but grpcui lists a service then fails to describe it — a user-visible regression | |
| Resolve on demand | Wrap `DescriptorResolver` so a miss triggers `ParseOne`. Satisfies both criteria honestly, but new machinery on an untested path with filesystem work inside reflection requests | |

**User's choice:** Everything (reuse discovery).
**Notes:** Verified before asking that `Init()` already performs a full eager `src/` parse into a throwaway registry (`server.go:351-360`, the Phase 11 CONS-01 fix) and discards it. This is the explicit resolution of the question Phase 11's D-02 deferred to "a later phase." The rejected on-demand option is preserved in Deferred Ideas as the principled upgrade.

### Q2 — What resolves types on the write path?

| Option | Description | Selected |
|--------|-------------|----------|
| TypeResolver everywhere (Recommended) | All type-URL and nested-`Any` resolution through `parser.TypeResolver`; discovery registry is reflection's backstop only. Closes Phase 13 D-03's outstanding grep clause | ✓ |
| Discovery resolver on the server | Server uses the eager discovery registry for both reflection and `MutateConfig`. Server behavior unchanged, but two resolution implementations survive and criterion 1's grep clause never closes | |
| You decide | Resolve from the code during planning under the constraint that it must close the grep clause | |

**User's choice:** TypeResolver everywhere.
**Notes:** Raised because keeping an eager discovery registry alive makes it tempting to point everything at it. Also surfaced that `MutateConfig` (`server.go:466`) currently marshals with `LocalResolver`, which under lazy would hard-fail every custom-type mutation. Four concrete call sites recorded in CONTEXT D-03, including `inserter.go:379`'s marshal resolver, whose output the agent serves verbatim to REST clients.

---

## CONS-04 Failure Shape

### Q1 — How should `GenReflectionUI` report a config it cannot resolve?

| Option | Description | Selected |
|--------|-------------|----------|
| Aggregate, log on change | Complete the whole walk, collect every failure with its diagnostic, log one aggregate, re-log only when the failure set changes. Loud without spam | ✓ |
| Aggregate, log every pass | Simplest — no state, no dedupe. But a permanently-broken config logs every 5 seconds forever | |
| Surface in the UI too | Aggregate + log-on-change, plus render broken configs as visible grpcui entries. Strongest read of "reports rather than silently skips" | |

**User's choice:** Aggregate, log on change.
**Notes:** Confirmed by reading the callers before asking: `GenReflectionUI` runs on a **5-second ticker** at four call sites, all of which **discard its returned error**. That made "fail startup" meaningless from iteration 2 onward and made per-pass logging ~720 lines/hour. The user's stated reason for rejecting per-pass logging: that volume is loud enough to be ignored. Fixed regardless of the choice and stated as such: the walk must stop aborting on first failure, since `return err` today silently loses every remaining example — exactly what CONS-04 forbids. The rejected UI-rendering option is preserved in Deferred Ideas.

---

## How Criteria 4 & 5 Get Proven

### Q1 — What shape proves criterion 4 (concurrent requests, race-free)?

| Option | Description | Selected |
|--------|-------------|----------|
| Both: e2e + tight-loop | Bufconn e2e with N concurrent clients, plus a dedicated unpaced tight-loop test sanity-checked by removing the lock and confirming the race is detected | ✓ |
| End-to-end only | Real gRPC over bufconn under `-race`. Matches the criterion's wording, but is the shape Phase 12 found too rare to force the interleaving | |
| Tight-loop only | Drive the impl structs directly. Sharpest detection, but leaves handler and interceptor layers untested | |

**User's choice:** Both.
**Notes:** Phase 12's SAFE-01 finding was presented as the deciding evidence — `TestConcurrentCompile`'s goroutines spend most of their time inside `ParseFiles` with no lock held, so only a dedicated continuously-running-reader test reliably forces the window, and Phase 12 sanity-checked its own test by temporarily removing the lock.

### Q2 — What makes criterion 5's "proportional to demand" concrete?

| Option | Description | Selected |
|--------|-------------|----------|
| Both clauses (Recommended) | (a) escalation guard: one config resolvable only via the index tier, assert the count grows by its transitive closure alone; (b) sequence guard: N distinct configs, bounded per-config delta, ends below corpus size | ✓ |
| Escalation guard only | The actual regression this criterion exists to catch. Cheapest and sharpest, but leaves the "over time" clause unasserted | |
| Sequence guard only | Reads most directly as "long-running process," but catches a single all-loading request only by luck | |

**User's choice:** Both clauses.
**Notes:** Uses the existing exported-counter observable (`LoadedFileCount()`), per the Phase 12 D-05/D-06 and Phase 13 D-04 precedent — no new observable, no log line, no CLI surface. CONTEXT D-06 records the necessary scoping: the counter is read on the *serving* registry, never on D-06's retained discovery registry.

---

## Claude's Discretion

Deferred to Claude with measurement or correctness constraints attached (full text in CONTEXT.md):

- How the discovery registry is retained (must not become a general resolver; must not be read by a criterion-5 assertion)
- Shape of the aggregated CONS-04 error and its change-detection (a changed failure *reason* must still re-log)
- Whether the four `GenReflectionUI` call sites start checking the returned error (the ticker must keep running regardless)
- Whether `agent/filekv` needs any change beyond the construction flip (verify with a test, do not assume)
- Test placement and fixtures for the D-07/D-08 tests (the escalation fixture must be built against the real Phase 13 tier boundary)
- Whether `mutate`/`inserter` per-config failure aborts or skips (do not silently change CLI exit-code behavior)

## Deferred Ideas

- On-demand `DescriptorResolver` for reflection — the principled upgrade if the retained discovery parse's footprint ever matters
- Rendering unresolvable configs as visible broken entries in the grpcui example list
- `compiler/lib/config.go:68`'s silent `ErrUnexpectedType` skip — carried forward from Phase 13; now the last silent-skip branch in the milestone
- `devserver`'s three registries after the flip (lazy compiler + lazy mutation server + retained discovery) — reviewed and accepted; no criterion covers it and sharing a registry is a real design change, not a cleanup
