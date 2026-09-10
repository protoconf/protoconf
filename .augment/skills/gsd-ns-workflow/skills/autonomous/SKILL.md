---
name: gsd-autonomous
description: "Run all remaining phases autonomously — discuss→plan→execute per phase"
---

<augment_skill_adapter>
## A. Skill Invocation
- This skill is invoked when the user mentions `gsd-autonomous` or describes a task matching this skill.
- Treat all user text after the skill mention as `{{GSD_ARGS}}`.
- If no arguments are present, treat `{{GSD_ARGS}}` as empty.

## B. User Prompting
When the workflow needs user input, prompt the user conversationally:
- Present options as a numbered list in your response text
- Ask the user to reply with their choice
- For multi-select, ask for comma-separated numbers

## C. Tool Usage
Use these Augment tools when executing GSD workflows:
- `launch-process` for running commands (terminal operations)
- `str-replace-editor` for editing existing files
- `view` for reading files and listing directories
- `save-file` for creating new files
- `grep` for searching code (or use MCP servers for advanced search)
- `web-search`, `web-fetch` for web queries
- `add_tasks`, `view_tasklist`, `update_tasks` for task management

## D. Subagent Spawning
When the workflow needs to spawn a subagent:
- Use the built-in subagent spawning capability
- Define agent prompts in `.augment/agents/` directory
</augment_skill_adapter>

<objective>
Execute all remaining milestone phases autonomously. For each phase: discuss → plan → execute. Pauses only for user decisions (grey area acceptance, blockers, validation requests).

Uses ROADMAP.md phase discovery and Skill() flat invocations for each phase command. After all phases complete: milestone audit → complete → cleanup.

**Creates/Updates:**
- `.planning/STATE.md` — updated after each phase
- `.planning/ROADMAP.md` — progress updated after each phase
- Phase artifacts — CONTEXT.md, PLANs, SUMMARYs per phase

**After:** Milestone is complete and cleaned up.
</objective>

<execution_context>
@/Users/smintz/go/src/github.com/protoconf/protoconf/.augment/gsd-core/workflows/autonomous.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.augment/gsd-core/references/ui-brand.md
</execution_context>

<context>
Optional flags:
- `--from N` — start from phase N instead of the first incomplete phase.
- `--to N` — stop after phase N completes (halt instead of advancing to next phase).
- `--only N` — execute only phase N (single-phase mode).
- `--interactive` — run discuss inline with questions (not auto-answered), then dispatch plan→execute as background agents. Keeps the main context lean while preserving user input on decisions.
- `--converge` — run each phase's planning step through `gsd-plan-review-convergence` instead of plain `gsd-plan-phase`. Requires `workflow.plan_review_convergence=true`.
- `--cross-ai` — compatibility alias for `--converge`.

When `--converge` or `--cross-ai` is set, reviewer selector flags supported by `gsd-plan-review-convergence` may be passed through: `--codex`, `--gemini`, `--claude`, `--opencode`, `--ollama`, `--lm-studio`, `--llama-cpp`, `--all`, and `--max-cycles N`.

Project context, phase list, and state are resolved inside the workflow using init commands (`gsd-tools query init.milestone-op`, `gsd-tools query roadmap.analyze`). No upfront context loading needed.
</context>

<process>
Execute end-to-end.
Preserve all workflow gates (phase discovery, per-phase execution, blocker handling, progress display).
</process>
