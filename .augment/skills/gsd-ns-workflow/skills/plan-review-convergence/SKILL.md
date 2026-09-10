---
name: gsd-plan-review-convergence
description: "Cross-AI plan convergence - replan until review concerns are resolved."
---

<augment_skill_adapter>
## A. Skill Invocation
- This skill is invoked when the user mentions `gsd-plan-review-convergence` or describes a task matching this skill.
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
Cross-AI plan convergence loop — an outer revision gate around gsd-review and gsd-planner.
Repeatedly: review plans with external AI CLIs → if HIGH or actionable non-HIGH concerns remain → replan with --reviews feedback → re-review. Stops when no unresolved HIGH concerns or actionable MEDIUM/LOW findings remain outside PLAN.md, or when max cycles is reached.

**Flow:** Skill("gsd-plan-phase") → Agent→Skill("gsd-review") → check unresolved HIGH + actionable non-HIGH → Skill("gsd-plan-phase --reviews") → Agent→Skill("gsd-review") → ... → Converge or escalate

Replaces gsd-plan-phase's internal gsd-plan-checker with external AI reviewers (codex, gemini, etc.). Plan-phase runs **inline** (bare Skill at depth 0) so it can spawn gsd-planner/gsd-plan-checker at depth 1. Review runs inside an isolated Agent (gsd-review is a Bash leaf — no sub-agents needed). Orchestrator only does loop control.

**Orchestrator role:** Parse arguments, validate phase, run plan-phase inline (Skill at depth 0), spawn an Agent for gsd-review, check unresolved HIGH and actionable non-HIGH counts, stall detection, escalation gate.
</objective>

<execution_context>
@/Users/smintz/go/src/github.com/protoconf/protoconf/.augment/gsd-core/workflows/plan-review-convergence.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.augment/gsd-core/references/revision-loop.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.augment/gsd-core/references/gates.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.augment/gsd-core/references/agent-contracts.md
</execution_context>

<runtime_note>
**Copilot (VS Code):** Use `vscode_askquestions` wherever this workflow calls `conversational prompting`. They are equivalent — `vscode_askquestions` is the VS Code Copilot implementation of the same interactive question API. Do not skip questioning steps because `conversational prompting` appears unavailable; use `vscode_askquestions` instead.
</runtime_note>

<context>
Phase number: extracted from {{GSD_ARGS}} (required)

**Flags:**
- `--codex` — Use Codex CLI as reviewer (default if no reviewer flag given AND `review.default_reviewers` is unset; otherwise `review.default_reviewers` wins per ADR-0011 — #2315)
- `--gemini` — Use Gemini CLI as reviewer
- `--agy` / `--antigravity` — Use Antigravity CLI as reviewer (successor to the discontinued Gemini CLI)
- `--claude` — Use Claude CLI as reviewer (separate session)
- `--coderabbit` — Use CodeRabbit as reviewer (reviews the working-tree diff, not the source tree)
- `--opencode` — Use OpenCode as reviewer
- `--qwen` — Use Qwen Code CLI as reviewer (Alibaba Qwen models)
- `--cursor` — Use Cursor agent as reviewer
- `--ollama` — Use local Ollama server as reviewer (OpenAI-compatible, default host `http://localhost:11434`; configure model via `review.models.ollama`)
- `--lm-studio` — Use local LM Studio server as reviewer (OpenAI-compatible, default host `http://localhost:1234`; configure model via `review.models.lm_studio`)
- `--llama-cpp` — Use local llama.cpp server as reviewer (OpenAI-compatible, default host `http://localhost:8080`; configure model via `review.models.llama_cpp`)
- `--kimi-code` — Use Kimi Code CLI as reviewer (Moonshot AI)
- `--all` — Use all available CLIs and running local model servers
- `--max-cycles N` — Maximum replan→review cycles (default: 3)

**Feature gate:** This command requires `workflow.plan_review_convergence=true`. Enable with:
`gsd config-set workflow.plan_review_convergence true`
</context>

<process>
Execute end-to-end.
Preserve all workflow gates (pre-flight, revision loop, stall detection, escalation).
</process>
