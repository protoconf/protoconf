---
name: gsd-ui-review
description: "Retroactive 6-pillar visual audit of implemented frontend code"
---

<objective>
Conduct a retroactive 6-pillar visual audit. Produces UI-REVIEW.md with
graded assessment (1-4 per pillar). Works on any project.
Output: {phase_num}-UI-REVIEW.md
</objective>

<execution_context>
@/Users/smintz/go/src/github.com/protoconf/protoconf/.opencode/gsd-core/workflows/ui-review.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.opencode/gsd-core/references/ui-brand.md
</execution_context>

<context>
Phase: $ARGUMENTS — optional, defaults to last completed phase.
</context>

<process>
Execute end-to-end.
Preserve all workflow gates.
</process>
