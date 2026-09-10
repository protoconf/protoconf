---
name: gsd-ns-project
description: "project lifecycle | milestones audits summary"
allowed-tools:
  - Read
  - Skill
---


Route to the appropriate project / milestone skill based on the user's intent.
`gsd-plan-milestone-gaps` was deleted by #2790 — gap planning now happens
inline as part of `gsd-audit-milestone`'s output.

| User wants | Read |
|---|---|
| Start a new project | Read `skills/new-project/SKILL.md` |
| Onboard an existing codebase | Read `skills/onboard/SKILL.md` |
| Create a new milestone | Read `skills/new-milestone/SKILL.md` |
| Complete the current milestone | Read `skills/complete-milestone/SKILL.md` |
| Audit a milestone for issues | Read `skills/audit-milestone/SKILL.md` |
| Summarize milestone status | Read `skills/milestone-summary/SKILL.md` |
| Import an external plan | Read `skills/import/SKILL.md` |
| Bootstrap planning from existing docs | Read `skills/ingest-docs/SKILL.md` |
| Generate a developer profile | Read `skills/profile-user/SKILL.md` |
| Review and promote backlog items | Read `skills/review-backlog/SKILL.md` |

Read the matched sub-skill's SKILL.md and follow its instructions. The `skills/<name>/SKILL.md` paths in the right column are relative to this skill's own directory.
