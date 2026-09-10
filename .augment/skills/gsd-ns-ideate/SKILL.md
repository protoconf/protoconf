---
name: gsd-ns-ideate
description: "exploration capture | explore sketch spike spec capture"
---

<augment_skill_adapter>
## A. Skill Invocation
- This skill is invoked when the user mentions `gsd-ns-ideate` or describes a task matching this skill.
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

Route to the appropriate exploration / capture skill based on the user's intent.
`gsd-note`, `gsd-add-todo`, `gsd-add-backlog`, and `gsd-plant-seed` were folded
into `gsd-capture` (with `--note`, default, `--backlog`, `--seed` modes) by
#2790. The capture target lists pending todos via `--list`.

| User wants | Read |
|---|---|
| Explore an idea or opportunity | Read `skills/explore/SKILL.md` |
| Sketch out a rough design or plan | Read `skills/sketch/SKILL.md` |
| Time-boxed technical spike | Read `skills/spike/SKILL.md` |
| Write a spec for a phase | Read `skills/spec-phase/SKILL.md` |
| Capture a thought (todo / note / backlog / seed) | Read `skills/capture/SKILL.md` |

Read the matched sub-skill's SKILL.md and follow its instructions. The `skills/<name>/SKILL.md` paths in the right column are relative to this skill's own directory.
