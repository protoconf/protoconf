---
name: gsd-mvp-phase
description: "Plan a phase as a vertical MVP slice — user story, SPIDR splitting, then plan-phase"
---

<augment_skill_adapter>
## A. Skill Invocation
- This skill is invoked when the user mentions `gsd-mvp-phase` or describes a task matching this skill.
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
Guide the user through MVP-mode planning for a phase. The command:

1. Prompts for an "As a / I want to / So that" user story (three structured questions)
2. Runs SPIDR splitting check — if the story is too large, walks through Spike/Paths/Interfaces/Data/Rules and offers to split into multiple phases
3. Writes `**Mode:** mvp` and the reformatted `**Goal:**` to the phase's ROADMAP.md section
4. Delegates to `/gsd plan-phase <N>` which auto-detects MVP mode via the roadmap field

Phase 1 of the vertical-mvp-slice PRD shipped the planner-side machinery; this command is the user entry point for it.
</objective>

<execution_context>
@/Users/smintz/go/src/github.com/protoconf/protoconf/.augment/gsd-core/workflows/mvp-phase.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.augment/gsd-core/references/spidr-splitting.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.augment/gsd-core/references/user-story-template.md
</execution_context>

<runtime_note>
**Copilot (VS Code):** Use `vscode_askquestions` wherever this workflow calls `conversational prompting`. Equivalent API.
</runtime_note>

<context>
Phase number: {{GSD_ARGS}} (required — integer or decimal like `2.1`)

The phase must already exist in ROADMAP.md (created via `/gsd new-project`, `/gsd add-phase`, or `/gsd insert-phase`). This command does not create new phases — it converts an existing phase to MVP mode.
</context>

<process>
Execute the mvp-phase workflow from @/Users/smintz/go/src/github.com/protoconf/protoconf/.augment/gsd-core/workflows/mvp-phase.md end-to-end.
Preserve all gates: phase existence, status guard (refuse in_progress/completed), user-story format validation, SPIDR splitting check, ROADMAP write confirmation, plan-phase delegation.
</process>
