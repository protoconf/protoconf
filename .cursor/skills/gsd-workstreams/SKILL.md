---
name: gsd-workstreams
description: "Manage parallel workstreams — list, create, switch, status, progress, complete, and resume"
---

<cursor_skill_adapter>
## A. Skill Invocation
- This skill is invoked when the user mentions `gsd-workstreams` or describes a task matching this skill.
- Treat all user text after the skill mention as `{{GSD_ARGS}}`.
- If no arguments are present, treat `{{GSD_ARGS}}` as empty.

## B. User Prompting
When the workflow needs user input, prompt the user conversationally:
- Present options as a numbered list in your response text
- Ask the user to reply with their choice
- For multi-select, ask for comma-separated numbers

## C. Tool Usage
Use these Cursor tools when executing GSD workflows:
- `Shell` for running commands (terminal operations)
- `StrReplace` for editing existing files
- `Read`, `Write`, `Glob`, `Grep`, `Task`, `WebSearch`, `WebFetch`, `TodoWrite` as needed

## D. Subagent Spawning
When the workflow needs to spawn a subagent:
- Use `Task(subagent_type="generalPurpose", ...)`
- The `model` parameter maps to Cursor's model options (e.g., "fast")
</cursor_skill_adapter>

# /gsd-workstreams

Manage parallel workstreams for concurrent milestone work.

## Usage

`/gsd-workstreams [subcommand] [args]`

### Subcommands

| Command | Description |
|---------|-------------|
| `list` | List all workstreams with status |
| `create <name>` | Create a new workstream |
| `status <name>` | Detailed status for one workstream |
| `switch <name>` | Set active workstream |
| `progress` | Progress summary across all workstreams |
| `complete <name>` | Archive a completed workstream |
| `resume <name>` | Resume work in a workstream |

## Step 1: Parse Subcommand

Parse the user's input to determine which workstream operation to perform.
If no subcommand given, default to `list`.

## Step 2: Execute Operation

### list
Run: `gsd-tools query workstream.list --raw --cwd "$CWD"`
Display the workstreams in a table format showing name, status, current phase, and progress.

### create
Run: `gsd-tools query workstream.create <name> --raw --cwd "$CWD"`
After creation, display the new workstream path and suggest next steps:
- `/gsd-new-milestone --ws <name>` to set up the milestone

### status
Run: `gsd-tools query workstream.status <name> --raw --cwd "$CWD"`
Display detailed phase breakdown and state information.

### switch
Run: `gsd-tools query workstream.set <name> --raw --cwd "$CWD"`
Also set `GSD_WORKSTREAM` for the current session when the runtime supports it.
If the runtime exposes a session identifier, GSD also stores the active workstream
session-locally so concurrent sessions do not overwrite each other.

### progress
Run: `gsd-tools query workstream.progress --raw --cwd "$CWD"`
Display a progress overview across all workstreams.

### complete
Run: `gsd-tools query workstream.complete <name> --raw --cwd "$CWD"`
Archive the workstream to milestones/.

### resume
Set the workstream as active and suggest `/gsd-resume-work --ws <name>`.

## Step 3: Display Results

Format the JSON output from gsd-tools query into a human-readable display.
Include the `${GSD_WS}` flag in any routing suggestions.
