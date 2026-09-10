---
description: Generate an AI-SPEC.md design contract for phases that involve building AI systems.
argument-hint: "[phase number]"
requires: [phase]
tools:
  read: true
  write: true
  bash: true
  glob: true
  grep: true
  agent: true
  webfetch: true
  websearch: true
  question: true
  mcp__context7__*: true
---
<objective>
Create an AI design contract (AI-SPEC.md) for a phase involving AI system development.
Orchestrates gsd-framework-selector → gsd-ai-researcher → gsd-domain-researcher → gsd-eval-planner.
Flow: Select Framework → Research Docs → Research Domain → Design Eval Strategy → Done
</objective>

<execution_context>
@/Users/smintz/go/src/github.com/protoconf/protoconf/.kilo/gsd-core/workflows/ai-integration-phase.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.kilo/gsd-core/references/ai-frameworks.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.kilo/gsd-core/references/ai-evals.md
</execution_context>

<context>
Phase number: $ARGUMENTS — optional; when omitted, the orchestrating workflow reads ROADMAP.md and selects the next unplanned phase. This is not a `gsd-tools.cjs` CLI feature — the CLI's phase-lookup primitives require an explicit phase number.
</context>

<process>
Execute end-to-end.
Preserve all workflow gates.
</process>
