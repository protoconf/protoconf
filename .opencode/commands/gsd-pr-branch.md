---
description: Create a clean PR branch by filtering out .planning/ commits — ready for code review
argument-hint: "[target branch, default: main]"
requires: [review]
tools:
  bash: true
  read: true
  question: true
---

<objective>
Create a clean branch suitable for pull requests by filtering out .planning/ commits
from the current branch. Reviewers see only code changes, not GSD planning artifacts.

This solves the problem of PR diffs being cluttered with PLAN.md, SUMMARY.md, STATE.md
changes that are irrelevant to code review.
</objective>

<execution_context>
@/Users/smintz/go/src/github.com/protoconf/protoconf/.opencode/gsd-core/workflows/pr-branch.md
</execution_context>

<process>
Execute end-to-end.
</process>
