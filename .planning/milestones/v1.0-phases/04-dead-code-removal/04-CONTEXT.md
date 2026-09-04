# Phase 4: Dead Code Removal - Context

**Gathered:** 2026-03-27
**Status:** Ready for planning

<domain>
## Phase Boundary

Remove unnecessary init functions and dead error checks from the codebase. Two specific items: the runtime.GOMAXPROCS init() in inserter/inserter.go and the unreachable error check in filekv.Watch. All tests must pass after removal.

</domain>

<decisions>
## Implementation Decisions

### init() Function Removal
- **D-01:** Remove only the `init()` function in `inserter/inserter.go` (lines 47-53) that sets `runtime.GOMAXPROCS(runtime.NumCPU())` — this is a no-op since Go 1.5 already defaults to NumCPU
- **D-02:** Scope limited to inserter/inserter.go per REFC-09 — do not scan or modify init() functions in other packages

### Dead Error Check Removal
- **D-03:** Remove the unreachable `if err != nil { return }` block at `agent/filekv/filekv.go` lines 146-148 — this checks `err` from line 137 which was already handled at lines 138-141 and cannot be non-nil at that point
- **D-04:** Do not modify surrounding code, TODO comments, or other logic in filekv.go — only remove the dead check per REFC-10

### Claude's Discretion
- Whether to also remove the `runtime` import if it becomes unused after init() removal
- Exact formatting after dead code removal (gofmt will handle this)

</decisions>

<canonical_refs>
## Canonical References

No external specs — requirements fully captured in decisions above and in REQUIREMENTS.md (REFC-09, REFC-10).

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- None needed — this is pure removal, no new code

### Established Patterns
- `inserter/inserter.go` uses standard Go flag parsing and CLI patterns consistent with other packages
- `agent/filekv/filekv.go` implements the valkeyrie `store.Store` interface with fsnotify-based file watching

### Integration Points
- `inserter/inserter.go` init() runs at package load time — removal affects no callers since GOMAXPROCS already defaults to NumCPU
- `agent/filekv/filekv.go` Watch() goroutine — the dead check is unreachable, so removing it changes no behavior

</code_context>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches. This is a straightforward dead code cleanup.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 04-dead-code-removal*
*Context gathered: 2026-03-27*
