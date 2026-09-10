---
name: gsd-forensics
description: "Post-mortem investigation for failed GSD workflows — diagnoses what went wrong."
---

<cursor_skill_adapter>
## A. Skill Invocation
- This skill is invoked when the user mentions `gsd-forensics` or describes a task matching this skill.
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

<objective>
Investigate what went wrong during a GSD workflow execution. Analyzes git history, `.planning/` artifacts, and file system state to detect anomalies and generate a structured diagnostic report.

Purpose: Diagnose failed or stuck workflows so the user can understand root cause and take corrective action.
Output: Forensic report saved to `.planning/forensics/`, presented inline, with optional issue creation.
</objective>

<execution_context>
@/Users/smintz/go/src/github.com/protoconf/protoconf/.cursor/gsd-core/workflows/forensics.md
</execution_context>

<context>
**Data sources:**
- `git log` (recent commits, patterns, time gaps)
- `git status` / `git diff` (uncommitted work, conflicts)
- `.planning/STATE.md` (current position, session history)
- `.planning/ROADMAP.md` (phase scope and progress)
- `.planning/phases/*/` (PLAN.md, SUMMARY.md, VERIFICATION.md, CONTEXT.md)
- `.planning/reports/SESSION_REPORT.md` (last session outcomes)

**User input:**
- Problem description: {{GSD_ARGS}} (optional — will ask if not provided)
</context>

<process>
Execute end-to-end.
</process>

<success_criteria>
- Evidence gathered from all available data sources
- At least 4 anomaly types checked (stuck loop, missing artifacts, abandoned work, crash/interruption)
- Structured forensic report written to `.planning/forensics/report-{timestamp}.md`
- Report presented inline with findings, anomalies, and recommendations
- Interactive investigation offered for deeper analysis
- GitHub issue creation offered if actionable findings exist
</success_criteria>

<critical_rules>
- **Read-only investigation:** Do not modify project source files during forensics. Only write the forensic report and update STATE.md session tracking.
- **Redact sensitive data:** Strip absolute paths, API keys, tokens from reports and issues.
- **Ground findings in evidence:** Every anomaly must cite specific commits, files, or state data.
- **No speculation without evidence:** If data is insufficient, say so — do not fabricate root causes.
</critical_rules>
