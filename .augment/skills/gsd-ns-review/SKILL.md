---
name: gsd-ns-review
description: "quality gates | code review debug audit security eval ui"
---

<augment_skill_adapter>
## A. Skill Invocation
- This skill is invoked when the user mentions `gsd-ns-review` or describes a task matching this skill.
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

Route to the appropriate quality / review skill based on the user's intent.
`gsd-code-review-fix` was absorbed by `gsd-code-review --fix` in #2790.

| User wants | Read |
|---|---|
| Review code for quality and correctness | Read `skills/code-review/SKILL.md` |
| Auto-fix code review findings | Read `skills/code-review/SKILL.md` (--fix) |
| Audit UAT / acceptance testing | Read `skills/audit-uat/SKILL.md` |
| Security review of a phase | Read `skills/secure-phase/SKILL.md` |
| Evaluate AI response quality | Read `skills/eval-review/SKILL.md` |
| Review UI for design and accessibility | Read `skills/ui-review/SKILL.md` |
| Validate phase outputs | Read `skills/validate-phase/SKILL.md` |
| Debug a failing feature or error | Read `skills/debug/SKILL.md` |
| Forensic investigation of a broken system | Read `skills/forensics/SKILL.md` |
| Autonomous audit-to-fix pipeline | Read `skills/audit-fix/SKILL.md` |
| Cross-AI peer review of plans | Read `skills/review/SKILL.md` |
| Generate a UI design contract | Read `skills/ui-phase/SKILL.md` |

Read the matched sub-skill's SKILL.md and follow its instructions. The `skills/<name>/SKILL.md` paths in the right column are relative to this skill's own directory.
