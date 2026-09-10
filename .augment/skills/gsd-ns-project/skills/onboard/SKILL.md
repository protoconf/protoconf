---
name: gsd-onboard
description: "Guide existing codebase onboarding through mapping, doc ingest, and planning setup"
---

<augment_skill_adapter>
## A. Skill Invocation
- This skill is invoked when the user mentions `gsd-onboard` or describes a task matching this skill.
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

<runtime_note>
**Copilot (VS Code):** Use `vscode_askquestions` wherever this workflow calls `conversational prompting`. They are equivalent — `vscode_askquestions` is the VS Code Copilot implementation of the same interactive question API.
</runtime_note>

<objective>
Guide brownfield onboarding for an existing codebase by routing through the existing GSD primitives in the safe order: codebase map → docs ingest → project initialization → onboarding summary.

**Creates or confirms:**
- `.planning/codebase/` — evidence-backed codebase map from `/gsd-map-codebase`
- `.planning/PROJECT.md`, `REQUIREMENTS.md`, `ROADMAP.md`, `STATE.md` — project setup from `/gsd-new-project` or `/gsd-ingest-docs`
- `.planning/onboarding/SUMMARY.md` — lightweight index of what was learned and the next command

**Non-goals:** This command does not execute phases, ship work, or overwrite existing planning artifacts without an explicit gate.
</objective>

<execution_context>
@/Users/smintz/go/src/github.com/protoconf/protoconf/.augment/gsd-core/workflows/onboard.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.augment/gsd-core/references/ui-brand.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.augment/gsd-core/references/gate-prompts.md
</execution_context>

<context>
Arguments: {{GSD_ARGS}}

Flags:
- `--fast` — prefer `/gsd-map-codebase --fast` for the mapping handoff; the complete map is still required before `/gsd-new-project`.
- `--text` — use plain-text numbered lists instead of TUI menus.
</context>

<process>
Execute the onboard workflow end-to-end. Preserve all safety gates, text-mode fallbacks, idempotency checks, and top-level handoff rules for nested interactive commands.
</process>
