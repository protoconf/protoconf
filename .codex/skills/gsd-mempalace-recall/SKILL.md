---
name: "gsd-mempalace-recall"
description: "Recall decisions, patterns, and surprises from MemPalace before planning"
metadata:
  short-description: "Recall decisions, patterns, and surprises from MemPalace before planning"
---

<codex_skill_adapter>
## A. Skill Invocation
- This skill is invoked by mentioning `$gsd-mempalace-recall`.
- Treat all user text after `$gsd-mempalace-recall` as `{{GSD_ARGS}}`.
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

**STOP -- DO NOT READ THIS FILE. You are already reading it. This prompt was injected into your context by the command system. Using the Read tool on this file wastes tokens. Begin executing Step 0 immediately.**

## Step 0 -- Banner

**Before ANY tool calls**, display this banner:

```
GSD > MEMPALACE RECALL
```

Then proceed to Step 1.

## Step 1 -- Config Gate

Check whether the MemPalace capability is enabled by reading `.planning/config.json` directly with the Read tool.

**DO NOT use `node "$HOME/.codex/gsd-core/bin/gsd-tools.cjs" config get-value`** -- it hard-exits on missing keys.

1. Read `.planning/config.json` with the Read tool.
2. If the file does not exist: write the "unavailable" stub (Step 4) and **STOP**.
3. Parse the JSON. Proceed to Step 2 only if `config.mempalace && config.mempalace.enabled === true` **and** `config.mempalace.recall_on_plan !== false`. Otherwise display the disabled message and **STOP** (`recall_on_plan: false` turns plan-time recall off while leaving the rest of the capability enabled).

**Disabled message:**

```
GSD > MEMPALACE RECALL

MemPalace memory is disabled. To activate:

  node <runtime-home>/gsd-core/bin/gsd-tools.cjs config-set mempalace.enabled true

Recall is opt-in; the loop proceeds normally without it.
```

This step is `onError: skip` at `plan:pre` -- recall never blocks planning.

## Step 2 -- Resolve wing, mode, and transport

1. **Wing.** Use `config.mempalace.wing` if non-empty; otherwise derive from `config.project_code`; otherwise fall back to the repository directory name.
2. **Mode.** Read `config.mempalace.memory_mode` (`augment` | `kg_backend` | `replace`, default `augment`). It sets how authoritative the palace is during recall:
   - `augment` — the palace is an *additional* layer; read native memory (`.planning/graphs/`, STATE) too and treat the palace as supplementary.
   - `kg_backend` — for knowledge-graph facts, query the palace's temporal KG *first, as the primary source*; fall back to `.planning/graphs/` when the palace is unreachable. Non-KG drawer recall stays additive.
   - `replace` — resolve recall *through the palace as the source of truth*; consult native artifacts only as a fallback when the palace is unreachable.
   In every mode an unreachable palace degrades to native memory — recall never blocks (`onError: skip`).
3. **Transport.** Prefer the **MCP tools** (`mempalace_*`) in interactive runs *when your MemPalace MCP server is registered and your runtime permits those tools*. Otherwise — headless/cron/autonomous runs, or runtimes that don't grant the MemPalace MCP tools — use the **CLI** (`mempalace wake-up`, `mempalace search`), which this skill's `Bash` allow-tool always covers. If neither is reachable, go to Step 4.
4. **Topic.** Read the phase `CONTEXT.md` (the consumed artifact). Derive a short search query from its title, goal, and key decisions.

## Step 3 -- Retrieve (read-only)

All calls in this step are side-effect-free. On any error or timeout, stop retrieving and write whatever was gathered (or the stub) -- never raise. This skill reads only the palace; GSD's planner reads native memory (`.planning/graphs/`, STATE) regardless. So under `kg_backend`/`replace` an unreachable palace falls back to that native memory automatically — reflect that in the stub (Step 4) rather than implying memory is gone.

1. **Wake up** (cheap, ~600--900 tokens):
   - Interactive: read the wing identity/summary, then `mempalace_search`.
   - Headless: `mempalace wake-up --wing <wing>`.
2. **Targeted search:**
   - Interactive: `mempalace_search(query=<topic>, wing=<wing>)`.
   - Headless: `mempalace search "<topic>" --wing <wing>`.
3. **Knowledge-graph facts** (unless `config.mempalace.mirror_kg !== false` — registry default is true, an absent key means enabled): `mempalace_kg_query` / `mempalace_kg_timeline` for decisions relevant to the topic and their validity windows. Under `augment` the palace KG *supplements* GSD's native `.planning/graphs/` — combine both, do not treat the palace as the sole source. Under `kg_backend` or `replace` the palace KG is the *primary* graph source — query it first and use `.planning/graphs/` only as a fallback when the palace is unreachable.
4. **Dedup** the returned drawers/facts; keep the top results.

## Step 4 -- Write MEMORY-RECALL.md

Write `MEMORY-RECALL.md` in the current phase directory. The planner consumes it.

When recall succeeded, structure it as:

```markdown
# Memory Recall (MemPalace)

_Wing: <wing> · Mode: <mode> · Transport: <mcp|cli>_

## Prior decisions
- <decision> — <provenance: drawer id / kg fact, valid_from>

## Patterns
- <pattern> — <provenance>

## Surprises / gotchas
- <surprise> — <provenance>
```

When MemPalace is unreachable, write the stub and continue. Under `kg_backend`/`replace`, name the native fallback so the planner knows memory is not gone — only un-augmented by the palace this run:

```markdown
# Memory Recall (MemPalace)

_MemPalace unavailable at recall time — falling back to GSD native memory (`.planning/graphs/`, STATE). No palace recall this run._
```

## Anti-Patterns

1. DO NOT let any MemPalace error fail the step -- recall is `onError: skip`.
2. DO NOT write to the palace from this skill -- recall is read-only; capture is a separate skill.
3. DO NOT paste raw search output into the file -- distil to decisions/patterns/surprises with provenance.
4. DO NOT skip the config gate.
