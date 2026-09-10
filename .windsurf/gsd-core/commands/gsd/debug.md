---
name: gsd-debug
description: Systematic debugging with persistent state across context resets
argument-hint: "[list | status <slug> | continue <slug> | --diagnose] [issue description]"
allowed-tools:
  - Read
  - Write
  - Bash
  - Agent
  - conversational prompting
---

<objective>
Debug issues using scientific method with subagent isolation.

**Orchestrator role:** Gather symptoms, spawn gsd-debugger agent, handle checkpoints, spawn continuations.

**Flags:**
- `--diagnose` — Diagnose only. Returns a Root Cause Report without applying a fix.

**Subcommands:** `list` · `status <slug>` · `continue <slug>`
</objective>

<available_agent_types>
Valid GSD subagent types (use exact names — do not fall back to 'general-purpose'):
- gsd-debug-session-manager — manages debug checkpoint/continuation loop in isolated context
- gsd-debugger — investigates bugs using scientific method
</available_agent_types>

<execution_context>
@/Users/smintz/go/src/github.com/protoconf/protoconf/.windsurf/gsd-core/workflows/debug.md
</execution_context>

<context>
User's input: {{GSD_ARGS}}

Parse subcommands and flags from {{GSD_ARGS}} BEFORE the active-session check:
- If {{GSD_ARGS}} starts with "list": SUBCMD=list, no further args
- If {{GSD_ARGS}} starts with "status ": SUBCMD=status, SLUG=remainder (trim whitespace)
- If {{GSD_ARGS}} starts with "continue ": SUBCMD=continue, SLUG=remainder (trim whitespace)
- If {{GSD_ARGS}} contains `--diagnose`: SUBCMD=debug, diagnose_only=true, strip `--diagnose` from description
- Otherwise: SUBCMD=debug, diagnose_only=false

Check for active sessions (used for non-list/status/continue flows):
```bash
ls .planning/debug/*.md 2>/dev/null | grep -v resolved | head -5
```
</context>

<process>
Execute end-to-end.
</process>
