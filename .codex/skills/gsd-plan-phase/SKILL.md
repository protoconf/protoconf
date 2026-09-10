---
name: "gsd-plan-phase"
description: "Create detailed phase plan (PLAN.md) with verification loop"
metadata:
  short-description: "Create detailed phase plan (PLAN.md) with verification loop"
---

<codex_skill_adapter>
## A. Skill Invocation
- This skill is invoked by mentioning `$gsd-plan-phase`.
- Treat all user text after `$gsd-plan-phase` as `{{GSD_ARGS}}`.
- If no arguments are present, treat `{{GSD_ARGS}}` as empty.

## B. AskUserQuestion → request_user_input Mapping
GSD workflows use `AskUserQuestion` (Claude Code syntax). Translate to Codex `request_user_input`:

Parameter mapping:
- `header` → `header`
- `question` → `question`
- Options formatted as `"Label" — description` → `{label: "Label", description: "description"}`
- Generate `id` from header: lowercase, replace spaces with underscores

Batched calls:
- `AskUserQuestion([q1, q2])` → single `request_user_input` with multiple entries in `questions[]`

Multi-select workaround:
- Codex has no `multiSelect`. Use sequential single-selects, or present a numbered freeform list asking the user to enter comma-separated numbers.

Execute mode fallback:
- When `request_user_input` is rejected or unavailable, activate TEXT_MODE: append `--text` to `{{GSD_ARGS}}` so the workflow's built-in text-mode branching takes over. Present every `AskUserQuestion` call as a plain-text numbered list, then stop and wait for the user's reply. Do NOT pick a default and continue (#3018 / #3808).
- You may only proceed without a user answer when one of these is true:
  (a) the invocation included an explicit non-interactive flag (`--auto` or `--all`),
  (b) the user has explicitly approved a specific default for this question, or
  (c) the workflow's documented contract says defaults are safe (e.g. autonomous lifecycle paths).
- Do NOT write workflow artifacts (CONTEXT.md, DISCUSSION-LOG.md, PLAN.md, checkpoint files) until the user has answered the plain-text questions or one of (a)-(c) above applies. Surfacing the questions and waiting is the correct response — silently defaulting and writing artifacts is the #3018 failure mode.

## C. Task() → spawn_agent Mapping
GSD workflows use `Task(...)` (Claude Code syntax). Translate to Codex collaboration tools:

**Schema detection (required first step):** Before spawning, inspect the `spawn_agent`
tool's visible parameter schema (via `tool_search` or the tool list). Use the presence
of `agent_type` only to choose typed dispatch versus the generic-agent workaround.
Detect optional fields independently: `model`, `reasoning_effort`, `task_name`,
`fork_turns`, and `fork_context` may be added or removed without `agent_type` changing.
Never infer one field from a schema/version label or from the presence of another field.

- **agent_type-capable schema:** `spawn_agent` advertises `agent_type` — typed GSD agent dispatch is available.
- **Generic schema:** `spawn_agent` does not advertise `agent_type` — typed GSD agent dispatch is unavailable in this session, even if other optional fields are present.

Typed mapping (agent_type-capable schema only):
- `Task(subagent_type="X", prompt="Y")` → `spawn_agent(agent_type="X", message="Y")`
- `Agent(subagent_type="X", prompt="Y")` → `spawn_agent(agent_type="X", message="Y")`
- `Task(model="{resolved_model}")` → pass `model="{resolved_model}"` when the
  visible `spawn_agent` schema advertises `model` and the resolved value is explicit.
  This is how `model_profile` tier routing, including `adaptive`, reaches the child agent.
  Omit `model` only when the schema does not advertise `model`, or when the value is
  missing, empty, or `"inherit"`; omission deliberately inherits the session/static agent
  configuration. Explicit `model_overrides` may also be embedded in agent `.toml` files,
  but ordinary profile-resolved models are not, so a TOML file is not a reason to discard
  an available inline value.
- Before each typed spawn, obtain the paired effort for its role with
  `gsd_run query resolve-model <subagent_type> --pick effort` when the workflow has not
  already exposed it. The resolver's unified `effort` field maps to the Codex spawn argument
  `reasoning_effort`; do not look for a resolver field named `reasoning_effort`.
  Pass it when the visible `spawn_agent` schema advertises `reasoning_effort`. Omit the
  field when it is not advertised, or when the value is missing, empty, `"inherit"`, or
  unsupported; do not invent one-off effort literals in workflow prose.
- `fork_context: false` by default — GSD agents load their own context via `<required_reading>` blocks
- `task_name` — when advertised, provide a descriptive name for each spawned task
- `fork_turns` — when advertised, controls turn-forking depth; coexists with `fork_context` (not a replacement)
- `Task(isolation="worktree")` / `Agent(isolation="worktree")` → no direct `spawn_agent` mapping,
  but Codex declares `dispatch.isolation: orchestrator-worktree` (#2584). Codex
  `spawn_agent` still does not create or bind a git worktree; instead GSD itself
  creates the worktree and process-spawns the executor into it with
  `codex exec --cd <dir>`, performing every git operation on the executor's behalf
  (its `workspace-write` sandbox makes `.git` read-only). Workflows must therefore
  never fabricate a manual worktree protocol — route through the negotiated
  isolation adapter, which still fails closed for hosts declaring `none` (#3360).

Generic-agent workaround (multi_agent_v1 schema — NO agent_type field):
When only the generic `multi_agent_v1` schema is available, typed GSD agent dispatch
(`gsd-planner`, `gsd-executor`, etc.) is NOT possible. This is a known Codex limitation
(openai/codex#15250). **This workaround is NOT equivalent to typed gsd-planner/gsd-executor
execution** — GSD agents carry project-aware prompts, audit logging, and workflow context
that a generic subagent lacks. Use the following fallback:
1. Resolve your active Codex config root — the directory that contains your `config.toml`.
   This directory is determined in priority order: `$CODEX_HOME` (if set), the path given
   by `--config-dir` (if passed on invocation), a local `.codex` directory in the current
   project (if `--local` was used), or the default global config directory. Read
   `agents/<agent-name>.toml` relative to that config root to extract the agent's system
   instructions.
2. Inject those instructions as a role-preamble into a generic `spawn_agent(message=...)` call.
3. Label results and logs clearly as "generic-agent workaround" so the orchestrator and user
   know full typed-agent guarantees are not in effect.
4. Where typed dispatch is mandatory for correctness (e.g. worktree isolation), fail closed
   and report the schema limitation rather than silently degrading.

Spawn restriction:
- Codex restricts `spawn_agent` to cases where the user has explicitly
  requested sub-agents. When automatic spawning is not permitted, do the
  work inline in the current agent rather than attempting to force a spawn.
- In some Codex sessions, multi-agent tooling can be deferred. If `spawn_agent`
  is not currently visible, discover tools first via `tool_search` before
  defaulting to inline execution.

Parallel fan-out:
- Spawn multiple agents → collect agent IDs → `collaboration.wait_agent(timeout_ms=...)` for each to complete
- Do NOT use `functions.wait(cell_id=...)` — that is an unrelated exec-cell tool, not the collaboration wait

Result parsing:
- Look for structured markers in agent output: `CHECKPOINT`, `PLAN COMPLETE`, `SUMMARY`, etc.
- `close_agent(id)` after collecting results — but only if `close_agent` is visible in the current
  tool schema (check via `tool_search` first, same schema-detection gate as `spawn_agent` above)
</codex_skill_adapter>

<objective>
Create executable phase prompts (PLAN.md files) for a roadmap phase with integrated research and verification.

**Default flow:** Research (if needed) → Plan → Verify → Done

**Research-only mode (`--research-phase <N>`):** Spawn `gsd-phase-researcher` for phase `N`, write `RESEARCH.md`, then exit before the planner runs. Useful for cross-phase research, doc review before committing to a planning approach, and correction-without-replanning loops where iterating on research alone is dramatically cheaper than re-spawning the planner. Replaces the deleted research-phase command (#3042).

**Research-only modifiers:**
- **No flag** — when `RESEARCH.md` already exists, auto-uses it: emits a one-line notice and exits cleanly, no prompt.
- **`--research`** — force-refresh: re-spawn the researcher unconditionally, no prompt. Bypasses the existing-RESEARCH.md auto-use path.
- **`--view`** — view-only: print existing `RESEARCH.md` to stdout. Does not spawn the researcher. Cheapest mode for the correction-without-replanning loop. If no `RESEARCH.md` exists yet, errors with a hint to drop `--view`.

**Orchestrator role:** Parse arguments, validate phase, research domain (unless skipped), spawn gsd-planner, verify with gsd-plan-checker, iterate until pass or max iterations, present results.
</objective>

<execution_context>
@/Users/smintz/go/src/github.com/protoconf/protoconf/.codex/gsd-core/workflows/plan-phase.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.codex/gsd-core/references/ui-brand.md
</execution_context>

<runtime_note>
**Copilot (VS Code):** Use `vscode_askquestions` wherever this workflow calls `AskUserQuestion`. They are equivalent — `vscode_askquestions` is the VS Code Copilot implementation of the same interactive question API. Do not skip questioning steps because `AskUserQuestion` appears unavailable; use `vscode_askquestions` instead.
</runtime_note>

<context>
Phase number: {{GSD_ARGS}} (optional — when omitted, the orchestrating workflow reads ROADMAP.md and selects the next unplanned phase; `gsd-tools.cjs` itself has no auto-detect feature and requires an explicit phase number)

**Flags:**
- `--research` — Force re-research even if RESEARCH.md exists
- `--skip-research` — Skip research, go straight to planning
- `--gaps` — Gap closure mode (reads VERIFICATION.md, skips research)
- `--skip-verify` — Skip verification loop
- `--prd <file>` — Use a PRD/acceptance criteria file instead of discuss-phase. Parses requirements into CONTEXT.md automatically. Skips discuss-phase entirely.
- `--ingest <path-or-glob>` — Use one or more ADR files instead of discuss-phase. Parses locked decisions + scope fences into CONTEXT.md automatically. Skips discuss-phase entirely.
- `--ingest-format <auto|nygard|madr|narrative>` — Optional ADR parser format override (`auto` default).
- `--reviews` — Replan incorporating cross-AI review feedback from REVIEWS.md (produced by `$gsd-review`)
- `--text` — Use plain-text numbered lists instead of TUI menus (required for `/rc` remote sessions)
- `--mvp` — MVP enrichment on top of the default tracer-first ordering: frames the phase goal as a user story and, on Phase 1 of a new project, also emits `SKELETON.md` (Walking Skeleton). Vertical slicing itself is now the default (see `--no-tracer`); `--mvp` no longer *turns it on*. Can be persisted on a phase via `**Mode:** mvp` in ROADMAP.md.
- `--no-tracer` — Opt out of the default **tracer-first** decomposition and plan horizontal layers (the legacy default). By default every plan LEADS with one production-quality end-to-end `tracer` slice that is verified before any expansion task.
- `--no-reversibility-gates` — Suppress the human checkpoint that a **one-way-door** decision normally earns, for runs you intend to leave unattended. By default a decision rated `one-way` (undo needs a migration, breaks a published contract, or is impossible) gets a `checkpoint:decision` before the task implementing it. Ratings are still recorded on tasks and `costly` items still flagged — the flag changes what stops the run, not what the plan remembers.

Normalize phase input in step 2 before any directory lookups.
</context>

<process>
Execute end-to-end.
Preserve all workflow gates (validation, research, planning, verification loop, routing).
</process>
