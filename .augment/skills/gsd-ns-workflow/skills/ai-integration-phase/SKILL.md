---
name: gsd-ai-integration-phase
description: "Generate an AI-SPEC.md design contract for phases that involve building AI systems."
---

<augment_skill_adapter>
## A. Skill Invocation
- This skill is invoked when the user mentions `gsd-ai-integration-phase` or describes a task matching this skill.
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
Create an AI design contract (AI-SPEC.md) for a phase involving AI system development.
Orchestrates gsd-framework-selector → gsd-ai-researcher → gsd-domain-researcher → gsd-eval-planner.
Flow: Select Framework → Research Docs → Research Domain → Design Eval Strategy → Done
</objective>

<execution_context>
@/Users/smintz/go/src/github.com/protoconf/protoconf/.augment/gsd-core/workflows/ai-integration-phase.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.augment/gsd-core/references/ai-frameworks.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.augment/gsd-core/references/ai-evals.md
</execution_context>

<context>
Phase number: {{GSD_ARGS}} — optional; when omitted, the orchestrating workflow reads ROADMAP.md and selects the next unplanned phase. This is not a `gsd-tools.cjs` CLI feature — the CLI's phase-lookup primitives require an explicit phase number.
</context>

<process>
Execute end-to-end.
Preserve all workflow gates.
</process>
