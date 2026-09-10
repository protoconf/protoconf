---
name: gsd-eval-review
description: "Audit an executed AI phase's evaluation coverage and produce an EVAL-REVIEW.md remediation plan."
argument-hint: "[phase number]"
allowed-tools:
  - Read
  - Write
  - Bash
  - Glob
  - Grep
  - Agent
  - AskUserQuestion
---

<objective>
Conduct a retroactive evaluation coverage audit of a completed AI phase.
Checks whether the evaluation strategy from AI-SPEC.md was implemented.
Produces EVAL-REVIEW.md with score, verdict, gaps, and remediation plan.
</objective>

<execution_context>
@/Users/smintz/go/src/github.com/protoconf/protoconf/.qwen/gsd-core/workflows/eval-review.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.qwen/gsd-core/references/ai-evals.md
</execution_context>

<context>
Phase: $ARGUMENTS — optional, defaults to last completed phase.
</context>

<process>
Execute end-to-end.
Preserve all workflow gates.
</process>
