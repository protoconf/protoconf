---
name: gsd-docs-update
description: "Generate or update project documentation verified against the codebase"
---

<augment_skill_adapter>
## A. Skill Invocation
- This skill is invoked when the user mentions `gsd-docs-update` or describes a task matching this skill.
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
Generate and update up to 9 documentation files for the current project. Each doc type is written by a gsd-doc-writer subagent that explores the codebase directly — no hallucinated paths, phantom endpoints, or stale signatures.

Flag handling rule:
- The optional flags documented below are available behaviors, not implied active behaviors
- A flag is active only when its literal token appears in `{{GSD_ARGS}}`
- If a documented flag is absent from `{{GSD_ARGS}}`, treat it as inactive
- `--force`: skip preservation prompts, regenerate all docs regardless of existing content or GSD markers
- `--verify-only`: check existing docs for accuracy against codebase, no generation (full verification requires Phase 4 verifier)
- If `--force` and `--verify-only` both appear in `{{GSD_ARGS}}`, `--force` takes precedence
</objective>

<execution_context>
@/Users/smintz/go/src/github.com/protoconf/protoconf/.augment/gsd-core/workflows/docs-update.md
</execution_context>

<context>
Arguments: {{GSD_ARGS}}

**Available optional flags (documentation only — not automatically active):**
- `--force` — Regenerate all docs. Overwrites hand-written and GSD docs alike. No preservation prompts.
- `--verify-only` — Check existing docs for accuracy against the codebase. No files are written. Reports VERIFY marker count. Full codebase fact-checking requires the gsd-doc-verifier agent (Phase 4).

**Active flags must be derived from `{{GSD_ARGS}}`:**
- `--force` is active only if the literal `--force` token is present in `{{GSD_ARGS}}`
- `--verify-only` is active only if the literal `--verify-only` token is present in `{{GSD_ARGS}}`
- If neither token appears, run the standard full-phase generation flow
- Do not infer that a flag is active just because it is documented in this prompt
</context>

<process>
Execute end-to-end.
Preserve all workflow gates (preservation_check, flag handling, wave execution, monorepo dispatch, commit, reporting).
</process>
