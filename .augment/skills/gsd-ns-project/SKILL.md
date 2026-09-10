---
name: gsd-ns-project
description: "project lifecycle | milestones audits summary"
---

<augment_skill_adapter>
## A. Skill Invocation
- This skill is invoked when the user mentions `gsd-ns-project` or describes a task matching this skill.
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

Route to the appropriate project / milestone skill based on the user's intent.
`gsd-plan-milestone-gaps` was deleted by #2790 — gap planning now happens
inline as part of `gsd-audit-milestone`'s output.

| User wants | Read |
|---|---|
| Start a new project | Read `skills/new-project/SKILL.md` |
| Onboard an existing codebase | Read `skills/onboard/SKILL.md` |
| Create a new milestone | Read `skills/new-milestone/SKILL.md` |
| Complete the current milestone | Read `skills/complete-milestone/SKILL.md` |
| Audit a milestone for issues | Read `skills/audit-milestone/SKILL.md` |
| Summarize milestone status | Read `skills/milestone-summary/SKILL.md` |
| Import an external plan | Read `skills/import/SKILL.md` |
| Bootstrap planning from existing docs | Read `skills/ingest-docs/SKILL.md` |
| Generate a developer profile | Read `skills/profile-user/SKILL.md` |
| Review and promote backlog items | Read `skills/review-backlog/SKILL.md` |

Read the matched sub-skill's SKILL.md and follow its instructions. The `skills/<name>/SKILL.md` paths in the right column are relative to this skill's own directory.
