---
name: gsd-ns-context
description: "codebase intel | map graphify docs learnings mempalace"
allowed-tools:
  - Read
  - Skill
---


Route to the appropriate codebase-intelligence skill based on the user's intent.
`gsd-scan` and `gsd-intel` were folded into `gsd-map-codebase` flags by #2790.

| User wants | Read |
|---|---|
| Map the full codebase structure | Read `skills/map-codebase/SKILL.md` |
| Quick lightweight codebase scan | Read `skills/map-codebase/SKILL.md` (--fast) |
| Query mapped intelligence files | Read `skills/map-codebase/SKILL.md` (--query) |
| Generate a knowledge graph | Read `skills/graphify/SKILL.md` |
| Update project documentation | Read `skills/docs-update/SKILL.md` |
| Extract learnings from a completed phase | Read `skills/extract-learnings/SKILL.md` |
| Recall prior decisions and patterns before planning | Read `skills/mempalace-recall/SKILL.md` |
| File a phase artifact into MemPalace | Read `skills/mempalace-capture/SKILL.md` |

Read the matched sub-skill's SKILL.md and follow its instructions. The `skills/<name>/SKILL.md` paths in the right column are relative to this skill's own directory.
