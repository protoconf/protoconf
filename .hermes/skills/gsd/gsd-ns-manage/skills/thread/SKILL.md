---
name: gsd-thread
description: "Manage persistent context threads for cross-session work"
version: "1.13.0"
argument-hint: "[list [--open | --resolved] | close <slug> | status <slug> | name | description]"
allowed-tools:
  - Read
  - Write
  - Bash
---


<objective>
Create, list, close, or resume persistent context threads. Threads are lightweight
cross-session knowledge stores for work that spans multiple sessions but
doesn't belong to any specific phase.
</objective>

<execution_context>
@/Users/smintz/go/src/github.com/protoconf/protoconf/.hermes/gsd-core/workflows/thread.md
</execution_context>

<process>
Execute end-to-end.
</process>
