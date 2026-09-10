---
name: gsd-capture
description: "Capture ideas, tasks, notes, and seeds to their destination"
---

<augment_skill_adapter>
## A. Skill Invocation
- This skill is invoked when the user mentions `gsd-capture` or describes a task matching this skill.
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
Capture ideas, tasks, notes, and seeds to their appropriate destination in the GSD system.

Mode routing:
- **default** (no flag): Capture as a structured todo for later work → add-todo workflow
- **--note**: Zero-friction idea capture (append/list/promote) → note workflow
- **--backlog**: Add an idea to the backlog parking lot (999.x numbering) → add-backlog workflow
- **--seed**: Capture a forward-looking idea with trigger conditions → plant-seed workflow
- **--list**: List pending todos and select one to work on → check-todos workflow
- **--list-seeds**: List/audit captured seeds (optional status filter) → list-seeds workflow
</objective>

<routing>

| Flag | Destination | Workflow |
|------|-------------|----------|
| (none) | Structured todo in .planning/todos/ | add-todo |
| --note | Timestamped note file, list, or promote | note |
| --backlog | ROADMAP.md backlog section (999.x) | add-backlog |
| --seed | .planning/seeds/SEED-NNN-slug.md | plant-seed |
| --list | Interactive todo browser + action router | check-todos |
| --list-seeds | Read-only seed list/audit (optional status filter) | list-seeds |

</routing>

<execution_context>
@/Users/smintz/go/src/github.com/protoconf/protoconf/.augment/gsd-core/workflows/add-todo.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.augment/gsd-core/workflows/note.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.augment/gsd-core/workflows/add-backlog.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.augment/gsd-core/workflows/plant-seed.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.augment/gsd-core/workflows/check-todos.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.augment/gsd-core/workflows/list-seeds.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.augment/gsd-core/references/ui-brand.md
</execution_context>

<context>
Arguments: {{GSD_ARGS}}

Parse the first token of {{GSD_ARGS}}:
- If it is `--note`: strip the flag, pass remainder to note workflow
- If it is `--backlog`: strip the flag, pass remainder to add-backlog workflow
- If it is `--seed`: strip the flag, pass remainder to plant-seed workflow
- If it is `--list-seeds`: strip the flag, pass remainder (optional status filter) to list-seeds workflow
- If it is `--list`: pass remainder (optional area filter) to check-todos workflow
- Otherwise: pass all of {{GSD_ARGS}} to add-todo workflow
</context>

<process>
1. Parse the leading flag (if any) from {{GSD_ARGS}}.
2. Load and execute the appropriate workflow end-to-end based on the routing table above.
3. Preserve all workflow gates from the target workflow (directory structure, duplicate detection, commits, etc.).
</process>
