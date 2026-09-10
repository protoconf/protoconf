---
description: Manage persistent context threads for cross-session work
argument-hint: "[list [--open | --resolved] | close <slug> | status <slug> | name | description]"
requires: [phase]
tools:
  read: true
  write: true
  bash: true
---

<objective>
Create, list, close, or resume persistent context threads. Threads are lightweight
cross-session knowledge stores for work that spans multiple sessions but
doesn't belong to any specific phase.
</objective>

<execution_context>
@/Users/smintz/go/src/github.com/protoconf/protoconf/.kilo/gsd-core/workflows/thread.md
</execution_context>

<process>
Execute end-to-end.
</process>
