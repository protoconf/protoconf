---
name: gsd-ultraplan-phase
description: "[BETA] Offload plan phase to Trae's ultraplan cloud; review in browser and import back."
stage: workflow
---


<objective>
Offload GSD's plan phase to Trae's ultraplan cloud infrastructure.

Ultraplan drafts the plan in a remote cloud session while your terminal stays free.
Review and comment on the plan in your browser, then import it back via /gsd-import --from.

⚠ BETA: ultraplan is in research preview. Use /gsd-plan-phase for stable local planning.
Requirements: Trae v2.1.91+, claude.ai account, GitHub repository.
</objective>

<execution_context>
@/Users/smintz/go/src/github.com/protoconf/protoconf/.trae/gsd-core/workflows/ultraplan-phase.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.trae/gsd-core/references/ui-brand.md
</execution_context>

<context>
{{GSD_ARGS}}
</context>

<process>
Execute the ultraplan-phase workflow end-to-end.
</process>
