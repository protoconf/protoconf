# Phase 4: Dead Code Removal - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-03-27
**Phase:** 04-dead-code-removal
**Areas discussed:** Scope of init() scan, filekv dead code extent, Verification approach
**Mode:** --auto (all decisions auto-selected)

---

## Scope of init() scan

| Option | Description | Selected |
|--------|-------------|----------|
| Only inserter/inserter.go | Per REFC-09, only the specific init() identified in requirements | :heavy_check_mark: |
| Scan all packages | Broader sweep for unnecessary init() functions across codebase | |

**User's choice:** [auto] Only inserter/inserter.go (recommended default)
**Notes:** REFC-09 is specific to inserter. Broader scan would risk scope creep beyond phase boundary.

---

## filekv dead code extent

| Option | Description | Selected |
|--------|-------------|----------|
| Remove dead error check only | Lines 146-148 per REFC-10 | :heavy_check_mark: |
| Also clean up stale TODO comments | Remove "TODO implement me" on line 112 and similar | |

**User's choice:** [auto] Remove dead error check only (recommended default)
**Notes:** TODO comments are documentation markers, not dead code. REFC-10 is specific to the dead error check.

---

## Verification approach

| Option | Description | Selected |
|--------|-------------|----------|
| Run existing tests only | Verify no regressions with current test suite | :heavy_check_mark: |
| Add new tests for changed code | Write tests before/after removal | |

**User's choice:** [auto] Run existing tests only (recommended default)
**Notes:** New test coverage is Phase 9's scope. This phase's success criteria only require existing tests to pass.

---

## Claude's Discretion

- Whether to remove unused `runtime` import after init() deletion
- Exact formatting after dead code removal

## Deferred Ideas

None — discussion stayed within phase scope.
