---
name: gsd-phase
description: "Multi-phase management — add, insert, remove, or edit phases in ROADMAP.md (roadmap phase CRUD)"
---

<cursor_skill_adapter>
## A. Skill Invocation
- This skill is invoked when the user mentions `gsd-phase` or describes a task matching this skill.
- Treat all user text after the skill mention as `{{GSD_ARGS}}`.
- If no arguments are present, treat `{{GSD_ARGS}}` as empty.

## B. User Prompting
When the workflow needs user input, prompt the user conversationally:
- Present options as a numbered list in your response text
- Ask the user to reply with their choice
- For multi-select, ask for comma-separated numbers

## C. Tool Usage
Use these Cursor tools when executing GSD workflows:
- `Shell` for running commands (terminal operations)
- `StrReplace` for editing existing files
- `Read`, `Write`, `Glob`, `Grep`, `Task`, `WebSearch`, `WebFetch`, `TodoWrite` as needed

## D. Subagent Spawning
When the workflow needs to spawn a subagent:
- Use `Task(subagent_type="generalPurpose", ...)`
- The `model` parameter maps to Cursor's model options (e.g., "fast")
</cursor_skill_adapter>

<objective>
Manage phases in ROADMAP.md with a single consolidated command.

Mode routing:
- **default** (no flag): Add a new integer phase to the end of the current milestone → add-phase workflow
- **--insert**: Insert urgent work as a decimal phase (e.g., 72.1) between existing phases → insert-phase workflow
- **--remove**: Remove a future phase and renumber subsequent phases → remove-phase workflow
- **--edit**: Edit any field of an existing phase in place → edit-phase workflow
</objective>

<routing>

| Flag | Action | Workflow |
|------|--------|----------|
| (none) | Add new integer phase at end of milestone | add-phase |
| --insert | Insert decimal phase (e.g., 72.1) after specified phase | insert-phase |
| --remove | Remove future phase, renumber subsequent | remove-phase |
| --edit | Edit fields of existing phase in place | edit-phase |

</routing>

<execution_context>
@/Users/smintz/go/src/github.com/protoconf/protoconf/.cursor/gsd-core/workflows/add-phase.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.cursor/gsd-core/workflows/insert-phase.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.cursor/gsd-core/workflows/remove-phase.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.cursor/gsd-core/workflows/edit-phase.md
</execution_context>

<context>
Arguments: {{GSD_ARGS}}

Parse the first token of {{GSD_ARGS}}:
- If it is `--insert`: strip the flag, pass remainder (format: <after-phase-number> <description>) to insert-phase workflow
- If it is `--remove`: strip the flag, pass remainder (phase number) to remove-phase workflow
- If it is `--edit`: strip the flag, pass remainder (phase-number [--force]) to edit-phase workflow
- Otherwise: pass all of {{GSD_ARGS}} (phase description) to add-phase workflow

Roadmap and state are resolved in-workflow via `init phase-op` and targeted reads.
</context>

<process>
1. Parse the leading flag (if any) from {{GSD_ARGS}}.
2. Load and execute the appropriate workflow end-to-end based on the routing table above.
3. Preserve all validation gates from the target workflow.
</process>
