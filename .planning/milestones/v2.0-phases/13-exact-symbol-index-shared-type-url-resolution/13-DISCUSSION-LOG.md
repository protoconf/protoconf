# Phase 13: Exact Symbol Index & Shared Type-URL Resolution - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-09-08
**Phase:** 13-exact-symbol-index-shared-type-url-resolution
**Areas discussed:** Index scope & payload, Cache key & invalidation, Shared path & fallback tier, ReadConfig & the criterion-3 guard, Lexical scan tier (user-raised)

---

## Index scope & payload

| Option | Description | Selected |
|--------|-------------|----------|
| Messages only, nested included | Literal TYPE-01. Smallest index, cheapest build. Enums/extensions keep resolving through the construction-time snapshot. | |
| Messages + enums, nested included | Mirrors what `GetTypesResolver` already registers, so the index is a strict superset of the resolver it backs. | |
| All named symbols | Messages, enums, services, extensions. Matches PROJECT.md's 138,554-symbol figure; would also house CONS-01's service enumeration. | |
| You decide | Pick on evidence from what the six consumers look up and what a no-link parse yields for free. | ✓ |

**User's choice:** You decide → recorded under Claude's Discretion.
**Notes:** The adjacent question — whether the index covers `.protoconf_cache` module repos as well as `src/` — was **not** asked, because it was settled by code during the session. On the lazy path `registry.ImportPaths = []string{srcPath}` (`compiler/lib/module_service.go:453`), and remote-module protos arrive instead via `registry.Load(<label>.fds, sum)` → `MergeFileDescriptorSet`, so their symbols are already in `MessageRegistry`. `src/`-only scope is derived from an existing invariant.

---

## Cache key & invalidation

| Option | Description | Selected |
|--------|-------------|----------|
| `dirhash.HashDir` over `src/` | Already a direct dep, already keys module repo caches (`module_service.go:273`). Literally content-keyed; costs a full read of `src/`. | |
| Stat manifest, hash on mismatch | `(path, size, mtime)` warm check, content hash on mismatch. Cheapest warm path; mtime lies on a fresh clone, so must be one-way. | |
| md5 sum mirroring the `.fds` idiom | Reuse `registry.Load`'s checksum gate. Catch: that sum is over the *parsed* set, so it requires the parse the cache exists to skip. | |
| You decide | Pick on measured evidence — hash cost vs. rebuild cost, recorded. | ✓ |

**User's choice:** You decide → recorded under Claude's Discretion with a measurement constraint.
**Notes:** Framing offered before the question: because the index is built on first need (TYPE-07), the validity check is also only paid on first need, so a `google.protobuf.Value`-only compile hashes nothing. That materially weakens the cost argument against a strict content hash.

---

## Shared path & fallback tier — blast radius

| Option | Description | Selected |
|--------|-------------|----------|
| Compiler only; Ph14 migrates rest | Build the shared path, wire the compiler onto it. `inserter:369`, `server:607`, `mutate:76` wait for Phase 14. Honors Phase 12's D-03 fence. | ✓ |
| Rewire all consumers now | Criterion 1's grep passes when Phase 13 closes; Phase 14 becomes pure verification. Larger blast radius in one phase. | |
| One shared function, opt-in call sites | Extract the implementation now; other consumers call it on their eager registry where it degrades to today's behavior. | |

**User's choice:** Compiler only; Phase 14 migrates the rest. → **D-03**
**Notes:** Verified during the session that this composes with the fallback decision below rather than conflicting with it: `ParseAll` already short-circuits on `len(d.ImportPaths) == 0` (`utils/utils.go:436`), which is every eager registry, so "the index replaces `ParseAll`" is automatically compiler-scoped. Consequence recorded in CONTEXT.md: ROADMAP criterion 1's grep clause completes in Phase 14, not this phase.

---

## Shared path & fallback tier — unresolvable type URL

| Option | Description | Selected |
|--------|-------------|----------|
| Index replaces `ParseAll`; hard error | Index is the last tier; an unreachable symbol is an error, not a 799-file parse. Deletes the 4.6s path outright. | |
| Index before `ParseAll`; loud warn | `ParseAll` still fires but warns. Safest for correctness; leaves the 4.6s path reachable in production. | |
| Index replaces `ParseAll`; error names the symbol | Same as the first, but the error is the diagnostic — unresolved URL, directories searched, index cold/warm. Folds the "loud fallback" requirement into this phase. | ✓ |

**User's choice:** Index replaces `ParseAll`; error names the symbol. → **D-02**
**Notes:** Accepted risk stated explicitly in CONTEXT.md — an index or scan blind spot turns a slow compile into a failed one. Rated **one-way** reversibility: restoring the fallback later means re-introducing `ParseAll`'s whole-tree parse and its `eagerFallback` latch, and any repo that compiled only because of that fallback will have been failing in the interim.

---

## ReadConfig & the criterion-3 guard

| Option | Description | Selected |
|--------|-------------|----------|
| Exported build counter | Mirrors Phase 12's D-05/D-06 exactly: exported method beside `LoadedFileCount()`, test-only, no log line or CLI surface. Test asserts count == 0 on a `Value`-only compile. | ✓ |
| Counter plus cold-cache assertion | Counter plus asserting no index file appears under `.protoconf_cache`. Two observables, at the cost of coupling the test to cache layout. | |
| Timing assertion in the scaling gate | Fold into `TestCompilerStartupScaling` under a wall-clock bound. Phase 12 already rejected this shape for RSLV-02 as the flakiest gate. | |
| You decide | Pick the observable on evidence, following D-05/D-06 precedent. | |

**User's choice:** Exported build counter. → **D-04**
**Notes:** The companion question — whether `parser.ReadConfig` needs a `proto_file` pre-pass at all, given that `protojson` consults its `Resolver` for every nested `Any` — was **not** asked. It is answerable from code and was recorded under Claude's Discretion with a verify-don't-assume constraint.

---

## Lexical scan tier (raised by the user, not offered as an option)

The user interrupted the "ready for context?" gate with a proposal:

> `grep -e "^package packagename.v1" -e "^message MessageName" src/**/*.proto` should give back a pretty fast resolution path. Let's explore this option as a fallback to the index before eager loading.

Measured against the real 799-proto corpus at `../protoconf-terraform/example/src` (37.8 MB, warm page cache):

| Measurement | Value |
|---|---|
| Message declarations, top-level (`^message`) | 4,054 |
| Message declarations, **nested** (indented) | **126,530** — 96.9% |
| Whole-corpus un-anchored scan | **~11 ms** |
| OPTIONS.md's "naive lexical scan" (Go, read+regexp per file) | 623 ms |
| `ParseFilesButDoNotLink` over the same tree | 1,286 ms |
| Candidate files for `message AwsS3Bucket` / `Tags` / `Timeouts` | 4 / 15 / **351** |

Three findings were put back to the user: the speed instinct is right and OPTIONS.md's 623ms figure undersells a scan by ~50x; the `^message` anchor finds only 3.1% of symbols and structurally misses the nested-`Any` case TYPE-03 exists for; and a scan returns candidates, not answers.

| Option | Description | Selected |
|--------|-------------|----------|
| Scan BEFORE index build | snapshot → MessageRegistry → scan + verify → build/cache index only if the scan can't answer → hard error. Keeps every TYPE-* requirement; index built rarely rather than merely cached. | ✓ |
| Scan replaces the index entirely | Drop the persisted index; scan + verify-after-parse always. Revises PROJECT.md's key design decision and guts TYPE-01/02/05/06/07. | |
| Index only, as currently specified | Drop the scan; record the 11ms measurement for a future revisit. | |
| Scan before index, index built in background | Fastest steady state; adds concurrency to the phase whose lock discipline is already the delicate part. | |

**User's choice:** Scan BEFORE index build. → **D-01**
**Notes:** The placement the user originally named — scan *after* the index — was pushed back on and not adopted: if the index is exact and complete over `src/`, an index miss means the symbol is not in `src/`, so a scan of the same tree misses too. The follow-up question (when does the chain escalate from scan to index build?) returned "You decide" and is recorded under Claude's Discretion with a break-even measurement constraint, plus an explicit note that if the rule never fires, TYPE-01/02/05/06 ship as dead code and that must be visible rather than silent.

---

## Claude's Discretion

- Index symbol-kind coverage — messages only / + enums / all named symbols.
- Cache key and invalidation mechanism, with the measured hash cost recorded.
- The scan → index escalation rule, with the break-even candidate count derived.
- On-disk index format and in-memory shape (~138K symbols).
- Whether the index build is singleflighted like `ParseOne`.
- The scan's and index build's concurrency contract under Phase 11's lock discipline.
- Whether `parser.ReadConfig` needs a pre-pass at all.

## Deferred Ideas

- Replace the symbol index entirely with scan + verify-after-parse — plausible on the 11ms measurement, but a roadmap amendment via `/gsd-phase`, not a phase decision.
- Rewiring `inserter` / `server` / `mutate` / `agent/filekv` onto the shared path — CONS-02/03/04, Phase 14.
- `compiler/lib/config.go:68`'s silent `ErrUnexpectedType` skip — a fourth in-compiler resolution site whose silent-nil branch sits awkwardly beside D-02's loud-failure decision.
