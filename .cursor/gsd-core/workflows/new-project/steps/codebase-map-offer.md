**If `needs_codebase_map` is true** (from init — existing code detected but no codebase map):

**Text mode (`workflow.text_mode: true` in config or `--text` flag):** Set `TEXT_MODE=true` if `--text` is present in `{{GSD_ARGS}}` OR `text_mode` from init JSON is `true`. When TEXT_MODE is active, replace every `conversational prompting` call with a plain-text numbered list and ask the user to type their choice number. This is required for non-Claude runtimes (OpenAI Codex, Gemini CLI, etc.) where `conversational prompting` is not available.
Use conversational prompting:

- header: "Codebase"
- question: "I detected existing code in this directory. Would you like to map the codebase first?"
- options:
  - "Map codebase first" — Run /gsd-map-codebase to understand existing architecture (Recommended)
  - "Skip mapping" — Proceed with project initialization

**If "Map codebase first":**

```
Run `/gsd-map-codebase` first, then return to `/gsd-new-project`
```

Exit command.
