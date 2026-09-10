---
name: gsd-ns-manage
description: "config workspace | workstreams thread update ship inbox"
---

<augment_skill_adapter>
## A. Skill Invocation
- This skill is invoked when the user mentions `gsd-ns-manage` or describes a task matching this skill.
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

Route to the appropriate management skill based on the user's intent.
`gsd-config` (settings + advanced + integrations + profile) and `gsd-workspace`
(new + list + remove) are post-#2790 consolidated entries.

| User wants | Read |
|---|---|
| Configure GSD settings (basic / advanced / integrations / profile) | Read `skills/config/SKILL.md` |
| Manage workspaces (create / list / remove) | Read `skills/workspace/SKILL.md` |
| Manage parallel workstreams | Read `skills/workstreams/SKILL.md` |
| Continue work in a fresh context thread | Read `skills/thread/SKILL.md` |
| Pause current work | Read `skills/pause-work/SKILL.md` |
| Resume paused work | Read `skills/resume-work/SKILL.md` |
| Update the GSD installation | Read `skills/update/SKILL.md` |
| Ship completed work | Read `skills/ship/SKILL.md` |
| Process inbox items | Read `skills/inbox/SKILL.md` |
| Create a clean PR branch | Read `skills/pr-branch/SKILL.md` |
| Undo the last GSD action | Read `skills/undo/SKILL.md` |
| Archive accumulated phase directories | Read `skills/cleanup/SKILL.md` |
| Diagnose planning directory health | Read `skills/health/SKILL.md` |
| Open the interactive command center | Read `skills/manager/SKILL.md` |
| Configure workflow toggles and model profile | Read `skills/settings/SKILL.md` |
| Show project statistics | Read `skills/stats/SKILL.md` |
| Toggle which skills are surfaced | Read `skills/surface/SKILL.md` |
| Show the GSD command guide | Read `skills/help/SKILL.md` |

Read the matched sub-skill's SKILL.md and follow its instructions. The `skills/<name>/SKILL.md` paths in the right column are relative to this skill's own directory.
