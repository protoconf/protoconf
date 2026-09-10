---
name: gsd-review
description: "Request cross-AI peer review of phase plans from external AI CLIs"
---

<augment_skill_adapter>
## A. Skill Invocation
- This skill is invoked when the user mentions `gsd-review` or describes a task matching this skill.
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
Invoke external AI CLIs (Gemini, Claude, Codex, OpenCode, Qwen Code, Cursor) to independently review phase plans.
Produces a structured REVIEWS.md with per-reviewer feedback that can be fed back into
planning via /gsd-plan-phase --reviews.

**Flow:** Detect CLIs → Build review prompt → Invoke each CLI → Collect responses → Write REVIEWS.md
</objective>

<execution_context>
@/Users/smintz/go/src/github.com/protoconf/protoconf/.augment/gsd-core/workflows/review.md
</execution_context>

<context>
Phase number: extracted from {{GSD_ARGS}} (required)

**Flags:**
- `--gemini` — Include Gemini CLI review
- `--claude` — Include Claude CLI review (uses separate session)
- `--codex` — Include Codex CLI review
- `--opencode` — Include OpenCode review (uses model from user's OpenCode config)
- `--qwen` — Include Qwen Code review (Alibaba Qwen models)
- `--cursor` — Include Cursor agent review
- `--agy` / `--antigravity` — Include Antigravity CLI review
- `--all` — Include all available CLIs

**No flags** — if `review.default_reviewers` is set, review with only those configured
reviewers that are detected; otherwise review with all available CLIs. Configured
`review.reviewer_instances` names may appear in `review.default_reviewers`; each runs as an
independent reviewer identity backed by its configured adapter+model (see
`docs/CONFIGURATION.md`). Instance names are not valid as flags.
</context>

<process>
Execute end-to-end.
</process>
