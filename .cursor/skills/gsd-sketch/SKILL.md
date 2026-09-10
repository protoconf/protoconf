---
name: gsd-sketch
description: "Sketch UI/design ideas with throwaway HTML mockups, or propose what to sketch next (frontier mode)"
---

<cursor_skill_adapter>
## A. Skill Invocation
- This skill is invoked when the user mentions `gsd-sketch` or describes a task matching this skill.
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
Explore design directions through throwaway HTML mockups before committing to implementation.
Each sketch produces 2-3 variants for comparison. Sketches live in `.planning/sketches/` and
integrate with GSD commit patterns, state tracking, and handoff workflows. Loads spike
findings to ground mockups in real data shapes and validated interaction patterns.

Two modes:
- **Idea mode** (default) — describe a design idea to sketch
- **Frontier mode** (no argument or "frontier") — analyzes existing sketch landscape and proposes consistency and frontier sketches

Does not require prior new-project setup — auto-creates `.planning/sketches/` if needed.
</objective>

<execution_context>
@/Users/smintz/go/src/github.com/protoconf/protoconf/.cursor/gsd-core/workflows/sketch.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.cursor/gsd-core/workflows/sketch-wrap-up.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.cursor/gsd-core/references/ui-brand.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.cursor/gsd-core/references/sketch-theme-system.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.cursor/gsd-core/references/sketch-interactivity.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.cursor/gsd-core/references/sketch-tooling.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.cursor/gsd-core/references/sketch-variant-patterns.md
</execution_context>

<runtime_note>
**Copilot (VS Code):** Use `vscode_askquestions` wherever this workflow calls `conversational prompting`.
</runtime_note>

<context>
Design idea: {{GSD_ARGS}}

**Available flags:**
- `--quick` — Skip mood/direction intake, jump straight to decomposition and building. Use when the design direction is already clear.
- `--wrap-up` — Package sketch design findings into a persistent project skill for future build conversations. Runs the sketch-wrap-up workflow.
</context>

<process>
Parse the first token of {{GSD_ARGS}}:
- If it is `--wrap-up`: strip the flag, execute the sketch-wrap-up workflow end-to-end.
- Otherwise: execute the sketch workflow end-to-end.

Preserve all workflow gates (intake, decomposition, target stack research, variant evaluation, MANIFEST updates, commit patterns).
</process>
