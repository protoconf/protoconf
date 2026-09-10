---
name: gsd-spec-phase
description: "Clarify WHAT a phase delivers with ambiguity scoring; produces a SPEC.md before discuss-phase."
---

<cursor_skill_adapter>
## A. Skill Invocation
- This skill is invoked when the user mentions `gsd-spec-phase` or describes a task matching this skill.
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
Clarify phase requirements through structured Socratic questioning with quantitative ambiguity scoring.

**Position in workflow:** `spec-phase → discuss-phase → plan-phase → execute-phase → verify`

**How it works:**
1. Load phase context (PROJECT.md, REQUIREMENTS.md, ROADMAP.md, STATE.md)
2. Scout the codebase — understand current state before asking questions
3. Run Socratic interview loop (up to 6 rounds, rotating perspectives)
4. Score ambiguity across 4 weighted dimensions after each round
5. Gate: ambiguity ≤ 0.20 AND all dimensions meet minimums → write SPEC.md
6. Commit SPEC.md — discuss-phase picks it up automatically on next run

**Output:** `{phase_dir}/{padded_phase}-SPEC.md` — falsifiable requirements that lock "what/why" before discuss-phase handles "how"
</objective>

<execution_context>
@/Users/smintz/go/src/github.com/protoconf/protoconf/.cursor/gsd-core/workflows/spec-phase.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.cursor/gsd-core/templates/spec.md
</execution_context>

<runtime_note>
**Copilot (VS Code):** Use `vscode_askquestions` wherever this workflow calls `conversational prompting`. They are equivalent.
</runtime_note>

<context>
Phase number: {{GSD_ARGS}} (required)

**Flags:**
- `--auto` — Skip interactive questions; Claude selects recommended defaults and writes SPEC.md
- `--text` — Use plain-text numbered lists instead of TUI menus (required for `/rc` remote sessions)

Context files are resolved in-workflow using `init phase-op`.
</context>

<process>
Execute end-to-end.

**MANDATORY:** Read the workflow file BEFORE taking any action. The workflow contains the complete step-by-step process including the Socratic interview loop, ambiguity scoring gate, and SPEC.md generation. Do not improvise from the objective summary above.
</process>

<success_criteria>
- Codebase scouted for current state before questioning begins
- All 4 ambiguity dimensions scored after each interview round
- Gate passed: ambiguity ≤ 0.20 AND all dimension minimums met
- SPEC.md written with falsifiable requirements, explicit boundaries, and acceptance criteria
- SPEC.md committed atomically
- User knows they can now run /gsd-discuss-phase which will load SPEC.md automatically
</success_criteria>
