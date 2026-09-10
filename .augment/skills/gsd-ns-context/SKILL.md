---
name: gsd-ns-context
description: "codebase intel | map graphify docs learnings mempalace"
---

<augment_skill_adapter>
## A. Skill Invocation
- This skill is invoked when the user mentions `gsd-ns-context` or describes a task matching this skill.
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

Route to the appropriate codebase-intelligence skill based on the user's intent.
`gsd-scan` and `gsd-intel` were folded into `gsd-map-codebase` flags by #2790.

| User wants | Read |
|---|---|
| Map the full codebase structure | Read `skills/map-codebase/SKILL.md` |
| Quick lightweight codebase scan | Read `skills/map-codebase/SKILL.md` (--fast) |
| Query mapped intelligence files | Read `skills/map-codebase/SKILL.md` (--query) |
| Generate a knowledge graph | Read `skills/graphify/SKILL.md` |
| Update project documentation | Read `skills/docs-update/SKILL.md` |
| Extract learnings from a completed phase | Read `skills/extract-learnings/SKILL.md` |
| Recall prior decisions and patterns before planning | Read `skills/mempalace-recall/SKILL.md` |
| File a phase artifact into MemPalace | Read `skills/mempalace-capture/SKILL.md` |

Read the matched sub-skill's SKILL.md and follow its instructions. The `skills/<name>/SKILL.md` paths in the right column are relative to this skill's own directory.
