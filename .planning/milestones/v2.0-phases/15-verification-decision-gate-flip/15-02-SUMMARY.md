---
phase: 15-verification-decision-gate-flip
plan: 02
subsystem: docs
tags: [changelog, readme, buf, operator-docs]

requires:
  - phase: 13-exact-symbol-index-shared-type-url-resolution
    provides: "D-02 (eager fallback deleted) — the underlying one-way behavior change this plan documents"
  - phase: 14-non-compiler-consumer-correctness
    provides: "D-02 (failure surfaces at first request, loudly; no startup validation pass) — GATE-03 is this decision's operator-facing half"
provides:
  - "CHANGELOG.md breaking-change bullet documenting that protoconf compile no longer parses/links unreferenced protos"
  - "README.md standing-behavior section documenting the same, for readers who never open the changelog"
affects: [15-03]

actuals:
  tokens: 3300
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns: []

key-files:
  created: []
  modified:
    - CHANGELOG.md
    - README.md

key-decisions:
  - "D-04/D-05 executed as written: the write-up lands in both CHANGELOG.md (## Unreleased / ### ⚠ BREAKING CHANGES) and a new README.md section, mutually consistent, both carrying the buf breaking / buf lint / buf.yaml caveat so no reader concludes protoconf already runs whole-tree proto validation for them."
  - "Confirmed against .github/workflows/lint.yml and buf.yaml before writing the caveat: CI's buf-breaking job runs only `buf breaking --against-config buf.yaml`, no buf lint step exists, and buf.yaml (buf.build/protoconf/protoconf) scopes to this repository's own protos, not a downstream config repository."
  - "D-06 honored: no test, fixture, or .pconf case added — git diff for this plan touches only CHANGELOG.md and README.md."

requirements-completed: [GATE-03]

coverage:
  - id: D1
    description: "CHANGELOG.md carries a new **compiler:** breaking-change bullet under ## Unreleased stating what changed, the consequence, the buf remedy, and the CI caveat"
    requirement: "GATE-03"
    verification:
      - kind: other
        ref: "grep -n -A 20 '^## Unreleased' CHANGELOG.md | grep -q '\\*\\*compiler:\\*\\*'"
        status: pass
      - kind: other
        ref: "grep -c '^\\* \\*\\*cli:\\*\\*' CHANGELOG.md == 2 (pre-existing bullets untouched)"
        status: pass
    human_judgment: false
  - id: D2
    description: "README.md gains a new ## section, positioned between Quick start and Production setup, documenting the same behavior as standing fact for readers who never open the changelog"
    requirement: "GATE-03"
    verification:
      - kind: other
        ref: "grep -n '^## ' README.md (new heading positioned correctly, all 7 pre-existing headings survive)"
        status: pass
    human_judgment: false
  - id: D3
    description: "CHANGELOG bullet and README section are mutually consistent on all four facts (behavior, consequence, remedy, caveat) — both name buf, buf breaking, buf lint, and buf.yaml"
    requirement: "GATE-03"
    verification:
      - kind: other
        ref: "grep -q 'buf breaking' README.md && grep -q 'buf.yaml' README.md && grep -q 'buf lint' CHANGELOG.md && echo CONSISTENT"
        status: pass
    human_judgment: false
  - id: D4
    description: "No test, fixture, or .pconf case pins the documented behavior (D-06, negative decision)"
    verification:
      - kind: other
        ref: "git diff --name-only HEAD~2 -- lists only CHANGELOG.md and README.md"
        status: pass
    human_judgment: false

duration: 12min
completed: 2026-09-09
status: complete
---

# Phase 15 Plan 2: Operator-Facing Broken-Proto Decision Write-Up Summary

**Documented, in CHANGELOG.md and README.md, that `protoconf compile` no longer parses or links a `.proto` no config reaches — with buf named as the operator's remedy and an explicit caveat that this repository's own CI does not run `buf lint` and its `buf.yaml` does not cover a downstream config repository.**

## Performance

- **Duration:** 12 min
- **Started:** 2026-09-09T05:01:38Z
- **Completed:** 2026-09-09T05:05:35Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- `CHANGELOG.md`'s `## Unreleased` / `### ⚠ BREAKING CHANGES` block gained a third bullet, `**compiler:**`, matching the voice of the two existing `**cli:**` bullets, stating what changed, the consequence (error surfaces on first load instead of at compile time), the upside (6.97s → sub-200ms on the 799-proto reference corpus), the remedy (run `buf` against your own config repository), and the caveat (this repo's CI runs `buf breaking` not `buf lint`; `buf.yaml` scopes to protoconf's own protos).
- `README.md` gained a new `## What \`protoconf compile\` validates` section, positioned between `## Quick start` and `## Production setup`, written for a reader who has never read the changelog, carrying the same four facts as the CHANGELOG bullet.
- Verified both documents against the actual CI wiring before writing the caveat: `.github/workflows/lint.yml`'s `buf-breaking` job runs `buf breaking --against-config buf.yaml` only; no `buf lint` step exists anywhere in the repo's CI. `buf.yaml` is `buf.build/protoconf/protoconf`, scoped to this repository's own protos.
- Cross-file consistency confirmed by grep: both files contain `buf breaking` and `buf.yaml`; `CHANGELOG.md` contains `buf lint`; `README.md` contains `buf lint` twice (prose + remedy instruction).

## Task Commits

Each task was committed atomically:

1. **Task 1: CHANGELOG breaking-change entry for the compile-time validation change** - `d934acd` (feat)
2. **Task 2: README standing-behavior section on what `protoconf compile` validates** - `6058738` (feat)

## Files Created/Modified
- `CHANGELOG.md` - added one `**compiler:**` bullet to the existing `### ⚠ BREAKING CHANGES` list under `## Unreleased`; the two pre-existing `**cli:**` bullets are byte-unchanged
- `README.md` - added one new `## What \`protoconf compile\` validates` section between `## Quick start` and `## Production setup`; all seven pre-existing headings survive unchanged

## Decisions Made
- D-04 executed: accepted, with buf named as the answer, and the honest caveat stated plainly (not softened) in both documents.
- D-05 executed: the write-up lands in both CHANGELOG.md and README.md, verified mutually consistent via the plan's cross-file grep.
- D-06 executed (negative decision): no test, fixture, or `.pconf` case added.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness

Side-by-side comparison of the two texts, for a later reviewer to check consistency without opening both files:

**CHANGELOG.md bullet (`## Unreleased` / `### ⚠ BREAKING CHANGES`):**

> * **compiler:** `protoconf compile` no longer parses or links every `.proto` file under `src/` — it now parses only the files a config transitively loads, and a proto that no config reaches is never parsed. Previously, a syntax or reference error in an unreferenced proto still failed the compile; it no longer does. That error now surfaces the first time a config actually loads that proto. This is deliberate: startup cost is now proportional to what a config demands rather than to repository size (6.97s to sub-200ms on a 799-proto reference corpus, per `.planning/research/compiler-performance/BASELINE.md`). If you need whole-tree proto validation, run `buf` against your own config repository — that is now buf's job, not the compiler's. Caveat: this repository's own CI runs `buf breaking` and does not run `buf lint`, and this repository's `buf.yaml` scopes to protoconf's own protos, not to a downstream config repository. Whole-tree proto validation is something an operator sets up and runs themselves; protoconf does not do it for them.

**README.md section (`## What \`protoconf compile\` validates`):**

> `protoconf compile` parses and links only the `.proto` files a config transitively loads. Protos under `src/` that no config reaches are not parsed, and errors in them — a syntax mistake, a broken reference — are not reported at compile time.
>
> This is deliberate, not an oversight: compile cost is proportional to what the config actually demands, not to how many protos happen to live in the repository. On a 799-proto reference corpus this took startup from 6.97s to under 200ms.
>
> The error still surfaces — the first time a config loads that proto.
>
> For whole-tree validation across every proto in your repository, run `buf lint` against your own config repository, as a step in your own CI. Protoconf does not do this for you: this repository's own CI runs `buf breaking` and does not run `buf lint`, and this repository's `buf.yaml` covers protoconf's own protos, not a downstream config repository. Nothing in protoconf runs whole-tree validation on an operator's behalf.

Both texts agree on all four facts: what changed, the consequence, the remedy, and the caveat. GATE-03 closes cleanly for the remaining Phase 15 plan (GATE-04/GATE-05 test-suite and real-corpus evidence). No blockers.

---
*Phase: 15-verification-decision-gate-flip*
*Completed: 2026-09-09*

## Self-Check: PASSED

- FOUND: CHANGELOG.md contains `**compiler:**` bullet under `## Unreleased`
- FOUND: README.md contains `## What \`protoconf compile\` validates` between `## Quick start` and `## Production setup`
- FOUND: commit d934acd (Task 1)
- FOUND: commit 6058738 (Task 2)
- Plan-level `<verification>` re-run: CHANGELOG.md carries 3 breaking-change bullets (third is compiler); README.md has the new section correctly positioned; both files contain `buf lint`, `buf breaking`, `buf.yaml`; `git diff --name-only HEAD~2` lists only `CHANGELOG.md` and `README.md`.
