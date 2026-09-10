---
name: gsd-workspace
description: "Manage GSD workspaces — create, list, or remove isolated workspace environments"
---

<augment_skill_adapter>
## A. Skill Invocation
- This skill is invoked when the user mentions `gsd-workspace` or describes a task matching this skill.
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
Manage GSD workspaces with a single consolidated command.

Mode routing:
- **--new**: Create an isolated workspace with repo copies and independent .planning/ → new-workspace workflow
- **--list**: List active GSD workspaces and their status → list-workspaces workflow
- **--remove**: Remove a GSD workspace and clean up worktrees → remove-workspace workflow
</objective>

<routing>

| Flag | Action | Workflow |
|------|--------|----------|
| --new | Create workspace with worktree/clone strategy | new-workspace |
| --list | Scan ~/gsd-workspaces/, show summary table | list-workspaces |
| --remove | Confirm and remove workspace directory | remove-workspace |

</routing>

<execution_context>
@/Users/smintz/go/src/github.com/protoconf/protoconf/.augment/gsd-core/workflows/new-workspace.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.augment/gsd-core/workflows/list-workspaces.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.augment/gsd-core/workflows/remove-workspace.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.augment/gsd-core/references/ui-brand.md
</execution_context>

<context>
Arguments: {{GSD_ARGS}}

Parse the first token of {{GSD_ARGS}}:
- If it is `--new`: strip the flag, pass remainder (--name, --repos, --path, --strategy, --branch, --auto flags) to new-workspace workflow
- If it is `--list`: execute list-workspaces workflow (no argument needed)
- If it is `--remove`: strip the flag, pass remainder (workspace-name) to remove-workspace workflow
- Otherwise (no flag): show usage — one of --new, --list, or --remove is required
</context>

<process>
1. Parse the leading flag from {{GSD_ARGS}}.
2. Load and execute the appropriate workflow end-to-end based on the routing table above.
3. Preserve all workflow gates from the target workflow (validation, approvals, commits, routing).
</process>
