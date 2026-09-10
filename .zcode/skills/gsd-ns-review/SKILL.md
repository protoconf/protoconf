---
name: gsd-ns-review
description: "quality gates | code review debug audit security eval ui"
allowed-tools:
  - Read
  - Skill
---


Route to the appropriate quality / review skill based on the user's intent.
`gsd-code-review-fix` was absorbed by `gsd-code-review --fix` in #2790.

| User wants | Read |
|---|---|
| Review code for quality and correctness | Read `skills/code-review/SKILL.md` |
| Auto-fix code review findings | Read `skills/code-review/SKILL.md` (--fix) |
| Audit UAT / acceptance testing | Read `skills/audit-uat/SKILL.md` |
| Security review of a phase | Read `skills/secure-phase/SKILL.md` |
| Evaluate AI response quality | Read `skills/eval-review/SKILL.md` |
| Review UI for design and accessibility | Read `skills/ui-review/SKILL.md` |
| Validate phase outputs | Read `skills/validate-phase/SKILL.md` |
| Debug a failing feature or error | Read `skills/debug/SKILL.md` |
| Forensic investigation of a broken system | Read `skills/forensics/SKILL.md` |
| Autonomous audit-to-fix pipeline | Read `skills/audit-fix/SKILL.md` |
| Cross-AI peer review of plans | Read `skills/review/SKILL.md` |
| Generate a UI design contract | Read `skills/ui-phase/SKILL.md` |

Read the matched sub-skill's SKILL.md and follow its instructions. The `skills/<name>/SKILL.md` paths in the right column are relative to this skill's own directory.
