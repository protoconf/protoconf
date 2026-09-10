---
name: gsd-ns-manage
description: "config workspace | workstreams thread update ship inbox"
allowed-tools:
  - Read
  - Skill
---


Route to the appropriate management skill based on the user's intent.
`gsd-config` (settings + advanced + integrations + profile) and `gsd-workspace`
(new + list + remove) are post-#2790 consolidated entries.

| User wants | Read |
|---|---|
| Configure GSD settings (basic / advanced / integrations / profile) | Read `skills/config/SKILL.md` |
| Manage workspaces (create / list / remove) | Read `skills/workspace/SKILL.md` |
| Manage parallel workstreams | Read `skills/workstreams/SKILL.md` |
| Continue work in a fresh context thread | Read `skills/thread/SKILL.md` |
| Pause current work | Read `skills/pause-work/SKILL.md` |
| Resume paused work | Read `skills/resume-work/SKILL.md` |
| Update the GSD installation | Read `skills/update/SKILL.md` |
| Ship completed work | Read `skills/ship/SKILL.md` |
| Process inbox items | Read `skills/inbox/SKILL.md` |
| Create a clean PR branch | Read `skills/pr-branch/SKILL.md` |
| Undo the last GSD action | Read `skills/undo/SKILL.md` |
| Archive accumulated phase directories | Read `skills/cleanup/SKILL.md` |
| Diagnose planning directory health | Read `skills/health/SKILL.md` |
| Open the interactive command center | Read `skills/manager/SKILL.md` |
| Configure workflow toggles and model profile | Read `skills/settings/SKILL.md` |
| Show project statistics | Read `skills/stats/SKILL.md` |
| Toggle which skills are surfaced | Read `skills/surface/SKILL.md` |
| Show the GSD command guide | Read `skills/help/SKILL.md` |

Read the matched sub-skill's SKILL.md and follow its instructions. The `skills/<name>/SKILL.md` paths in the right column are relative to this skill's own directory.
