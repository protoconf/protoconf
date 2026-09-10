# GSD Core — Git. Ship. Done.

- GSD workflows live in `gsd-core/workflows/`. Load the relevant workflow when
  the user runs a `/gsd-*` command.
- GSD agents live in `agents/`. Use the matching agent when spawning subagents.
- GSD tools are at `gsd-core/bin/gsd-tools.cjs`. Run with `node`.
- Planning artifacts live in `.planning/`. Never edit them outside a GSD workflow.
- Do not apply GSD workflows unless the user explicitly asks for them.
- When a GSD command triggers a deliverable (feature, fix, docs), offer the next
  step to the user using Cline's ask_user tool after completing it.
