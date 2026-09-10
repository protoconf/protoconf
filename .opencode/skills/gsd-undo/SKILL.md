---
name: gsd-undo
description: "Safe git revert. Roll back phase or plan commits using the phase manifest with dependency checks."
---

<objective>
Safe git revert — roll back GSD phase or plan commits using the phase manifest, with dependency checks and a confirmation gate before execution.

Three modes:
- **--last N**: Show recent GSD commits for interactive selection
- **--phase NN**: Revert all commits for a phase (manifest + git log fallback)
- **--plan NN-MM**: Revert all commits for a specific plan
</objective>

<execution_context>
@/Users/smintz/go/src/github.com/protoconf/protoconf/.opencode/gsd-core/workflows/undo.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.opencode/gsd-core/references/ui-brand.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.opencode/gsd-core/references/gate-prompts.md
</execution_context>

<context>
$ARGUMENTS
</context>

<process>
Execute end-to-end.
</process>
