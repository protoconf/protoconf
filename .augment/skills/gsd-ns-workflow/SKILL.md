---
name: gsd-ns-workflow
description: "workflow | discuss plan execute verify phase progress"
---

<augment_skill_adapter>
## A. Skill Invocation
- This skill is invoked when the user mentions `gsd-ns-workflow` or describes a task matching this skill.
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

Route to the appropriate phase-pipeline skill based on the user's intent.
Sub-skill names below are post-#2790 consolidated targets — `gsd-phase`
absorbs the former add/insert/remove/edit-phase commands and `gsd-progress`
absorbs the former next/do workflow-advance commands. The reclaimed
`gsd-next` target is the state-aware smart-entry launcher, not the retired
workflow-advance command.

| User wants | Read |
|---|---|
| Gather context before planning | Read `skills/discuss-phase/SKILL.md` |
| Clarify what a phase delivers | Read `skills/spec-phase/SKILL.md` |
| Create a PLAN.md | Read `skills/plan-phase/SKILL.md` |
| Execute plans in a phase | Read `skills/execute-phase/SKILL.md` |
| Verify built features through UAT | Read `skills/verify-work/SKILL.md` |
| Add / insert / remove / edit a phase | Read `skills/phase/SKILL.md` |
| Advance to the next logical step | Read `skills/progress/SKILL.md` |
| Open the state-aware smart-entry launcher | Read `skills/next/SKILL.md` |
| Offload planning to the ultraplan cloud | Read `skills/ultraplan-phase/SKILL.md` |
| Cross-AI plan review convergence loop | Read `skills/plan-review-convergence/SKILL.md` |
| Generate tests for a completed phase | Read `skills/add-tests/SKILL.md` |
| Design an AI-integration phase | Read `skills/ai-integration-phase/SKILL.md` |
| Run all remaining phases autonomously | Read `skills/autonomous/SKILL.md` |
| Execute a trivial task inline | Read `skills/fast/SKILL.md` |
| Plan a phase as a vertical MVP slice | Read `skills/mvp-phase/SKILL.md` |
| Execute a quick task with GSD guarantees | Read `skills/quick/SKILL.md` |
| Batch several quick-shaped tasks together | Read `skills/quick-batch/SKILL.md` |

Read the matched sub-skill's SKILL.md and follow its instructions. The `skills/<name>/SKILL.md` paths in the right column are relative to this skill's own directory.
