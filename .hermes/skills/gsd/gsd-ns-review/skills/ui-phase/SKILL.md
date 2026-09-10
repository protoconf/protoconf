---
name: gsd-ui-phase
description: "Generate UI design contract (UI-SPEC.md) for frontend phases"
version: "1.13.0"
argument-hint: "[phase]"
allowed-tools:
  - Read
  - Write
  - Bash
  - Glob
  - Grep
  - Agent
  - WebFetch
  - AskUserQuestion
  - mcp__context7__*
---

<objective>
Create a UI design contract (UI-SPEC.md) for a frontend phase.
Orchestrates gsd-ui-researcher and gsd-ui-checker.
Flow: Validate → Research UI → Verify UI-SPEC → Done
</objective>

<execution_context>
@/Users/smintz/go/src/github.com/protoconf/protoconf/.hermes/gsd-core/workflows/ui-phase.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.hermes/gsd-core/references/ui-brand.md
</execution_context>

<context>
Phase number: $ARGUMENTS — optional, auto-detects next unplanned phase if omitted.
</context>

<process>
Execute end-to-end.
Preserve all workflow gates.
</process>
