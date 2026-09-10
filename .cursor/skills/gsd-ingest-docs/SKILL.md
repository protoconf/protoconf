---
name: gsd-ingest-docs
description: "Bootstrap or merge a .planning/ setup from existing ADRs, PRDs, SPECs, and docs in a repo."
---

<cursor_skill_adapter>
## A. Skill Invocation
- This skill is invoked when the user mentions `gsd-ingest-docs` or describes a task matching this skill.
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
Build the full `.planning/` setup (or merge into an existing one) from multiple pre-existing planning documents — ADRs, PRDs, SPECs, DOCs — in one pass.

- **Net-new bootstrap** (`--mode new`, default when `.planning/` is absent): produces PROJECT.md + REQUIREMENTS.md + ROADMAP.md + STATE.md from synthesized doc content, delegating final generation to `gsd-roadmapper`.
- **Merge into existing** (`--mode merge`, default when `.planning/` is present): appends phases and requirements derived from the ingested docs; hard-blocks any contradiction with existing locked decisions.

Auto-synthesizes most conflicts using the precedence rule `ADR > SPEC > PRD > DOC` (overridable via manifest). Surfaces unresolved cases in `.planning/INGEST-CONFLICTS.md` with three buckets: auto-resolved, competing-variants, unresolved-blockers. The BLOCKER gate from the shared conflict engine prevents any destination file from being written when unresolved contradictions exist.

**Inputs:** directory-convention discovery (`docs/adr/`, `docs/prd/`, `docs/specs/`, `docs/rfc/`, root-level `{ADR,PRD,SPEC,RFC}-*.md`), or an explicit `--manifest <file>` YAML listing `{path, type, precedence?}` per doc.

**v1 constraints:** hard cap of 50 docs per invocation; `--resolve interactive` is reserved for a future release.
</objective>

<execution_context>
@/Users/smintz/go/src/github.com/protoconf/protoconf/.cursor/gsd-core/workflows/ingest-docs.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.cursor/gsd-core/references/ui-brand.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.cursor/gsd-core/references/gate-prompts.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.cursor/gsd-core/references/doc-conflict-engine.md
</execution_context>

<context>
{{GSD_ARGS}}
</context>

<process>
Execute the ingest-docs workflow end-to-end. Preserve all approval gates (discovery, conflict report, routing) and the BLOCKER safety rule.
</process>
