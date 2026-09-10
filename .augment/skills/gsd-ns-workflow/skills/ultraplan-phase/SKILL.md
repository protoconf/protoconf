---
name: gsd-ultraplan-phase
description: "[BETA] Offload plan phase to Augment's ultraplan cloud; review in browser and import back."
---

<augment_skill_adapter>
## A. Skill Invocation
- This skill is invoked when the user mentions `gsd-ultraplan-phase` or describes a task matching this skill.
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
Offload GSD's plan phase to Augment's ultraplan cloud infrastructure.

Ultraplan drafts the plan in a remote cloud session while your terminal stays free.
Review and comment on the plan in your browser, then import it back via /gsd-import --from.

⚠ BETA: ultraplan is in research preview. Use /gsd-plan-phase for stable local planning.
Requirements: Augment v2.1.91+, claude.ai account, GitHub repository.
</objective>

<execution_context>
@/Users/smintz/go/src/github.com/protoconf/protoconf/.augment/gsd-core/workflows/ultraplan-phase.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.augment/gsd-core/references/ui-brand.md
</execution_context>

<context>
{{GSD_ARGS}}
</context>

<process>
Execute the ultraplan-phase workflow end-to-end.
</process>
