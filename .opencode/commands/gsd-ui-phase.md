---
description: Generate UI design contract (UI-SPEC.md) for frontend phases
argument-hint: "[phase]"
requires: [phase]
tools:
  read: true
  write: true
  bash: true
  glob: true
  grep: true
  agent: true
  webfetch: true
  question: true
  mcp__context7__*: true
---
<objective>
Create a UI design contract (UI-SPEC.md) for a frontend phase.
Orchestrates gsd-ui-researcher and gsd-ui-checker.
Flow: Validate → Research UI → Verify UI-SPEC → Done
</objective>

<execution_context>
@/Users/smintz/go/src/github.com/protoconf/protoconf/.opencode/gsd-core/workflows/ui-phase.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.opencode/gsd-core/references/ui-brand.md
</execution_context>

<context>
Phase number: $ARGUMENTS — optional, auto-detects next unplanned phase if omitted.
</context>

<process>
Execute end-to-end.
Preserve all workflow gates.
</process>
