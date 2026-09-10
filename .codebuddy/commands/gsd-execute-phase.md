---
description: "SDD phase execution — execute all plans in a phase with dependency-aware wave parallelization"
argument-hint: "<phase-number> [--wave N] [--gaps-only] [--interactive] [--tdd]"
---
<objective>
Execute all plans in a phase using wave-based parallel execution.

Orchestrator stays lean: discover plans, analyze dependencies, group into waves, spawn subagents, collect results. Each subagent loads the full execute-plan context and handles its own plan.

Optional wave filter:
- `--wave N` executes only Wave `N` for pacing, quota management, or staged rollout
- phase verification/completion still only happens when no incomplete plans remain after the selected wave finishes

Flag handling rule:
- The optional flags documented below are available behaviors, not implied active behaviors
- A flag is active only when its literal token appears in `{{GSD_ARGS}}`
- If a documented flag is absent from `{{GSD_ARGS}}`, treat it as inactive

Context budget: ~15% orchestrator, 100% fresh per subagent.
</objective>

<execution_context>
@/Users/smintz/go/src/github.com/protoconf/protoconf/.codebuddy/gsd-core/workflows/execute-phase.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.codebuddy/gsd-core/references/ui-brand.md
</execution_context>

<runtime_note>
**Copilot (VS Code):** Use `vscode_askquestions` wherever this workflow calls `AskUserQuestion`. They are equivalent — `vscode_askquestions` is the VS Code Copilot implementation of the same interactive question API.
</runtime_note>

<context>
Phase: {{GSD_ARGS}}

**Available optional flags (documentation only — not automatically active):**
- `--wave N` — Execute only Wave `N` in the phase. Use when you want to pace execution or stay inside usage limits.
- `--gaps-only` — Execute only gap closure plans (plans with `gap_closure: true` in frontmatter). Use after verify-work creates fix plans.
- `--interactive` — Execute plans sequentially inline (no subagents) with user checkpoints between tasks. Lower token usage, pair-programming style. Best for small phases, bug fixes, and verification gaps.

**Active flags must be derived from `{{GSD_ARGS}}`:**
- `--wave N` is active only if the literal `--wave` token is present in `{{GSD_ARGS}}`
- `--gaps-only` is active only if the literal `--gaps-only` token is present in `{{GSD_ARGS}}`
- `--interactive` is active only if the literal `--interactive` token is present in `{{GSD_ARGS}}`
- If none of these tokens appear, run the standard full-phase execution flow with no flag-specific filtering
- Do not infer that a flag is active just because it is documented in this prompt

Context files are resolved inside the workflow via `gsd-tools query init.execute-phase` and per-subagent `<required_reading>` blocks.
</context>

<process>
Execute end-to-end.
Preserve all workflow gates (wave execution, checkpoint handling, verification, state updates, routing).
</process>
