---
name: gsd-ultraplan-phase
description: "[BETA] Offload plan phase to Hermes Agent's ultraplan cloud; review in browser and import back."
version: "1.13.0"
argument-hint: "[phase-number]"
allowed-tools:
  - Read
  - Bash
  - Glob
  - Grep
---


<objective>
Offload GSD's plan phase to Hermes Agent's ultraplan cloud infrastructure.

Ultraplan drafts the plan in a remote cloud session while your terminal stays free.
Review and comment on the plan in your browser, then import it back via /gsd-import --from.

⚠ BETA: ultraplan is in research preview. Use /gsd-plan-phase for stable local planning.
Requirements: Hermes Agent v2.1.91+, claude.ai account, GitHub repository.
</objective>

<execution_context>
@/Users/smintz/go/src/github.com/protoconf/protoconf/.hermes/gsd-core/workflows/ultraplan-phase.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.hermes/gsd-core/references/ui-brand.md
</execution_context>

<context>
$ARGUMENTS
</context>

<process>
Execute the ultraplan-phase workflow end-to-end.
</process>
