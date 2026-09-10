---
name: gsd-ns-ideate
description: "exploration capture | explore sketch spike spec capture"
allowed-tools:
  - Read
  - Skill
---


Route to the appropriate exploration / capture skill based on the user's intent.
`gsd-note`, `gsd-add-todo`, `gsd-add-backlog`, and `gsd-plant-seed` were folded
into `gsd-capture` (with `--note`, default, `--backlog`, `--seed` modes) by
#2790. The capture target lists pending todos via `--list`.

| User wants | Read |
|---|---|
| Explore an idea or opportunity | Read `skills/explore/SKILL.md` |
| Sketch out a rough design or plan | Read `skills/sketch/SKILL.md` |
| Time-boxed technical spike | Read `skills/spike/SKILL.md` |
| Write a spec for a phase | Read `skills/spec-phase/SKILL.md` |
| Capture a thought (todo / note / backlog / seed) | Read `skills/capture/SKILL.md` |

Read the matched sub-skill's SKILL.md and follow its instructions. The `skills/<name>/SKILL.md` paths in the right column are relative to this skill's own directory.
