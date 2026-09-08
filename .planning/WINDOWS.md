---
schema_version: 1
open_count: 4
waived_count: 0
fixed_count: 0
total_count: 4
last_updated: 2026-09-08T13:58:21.944Z
---

# Broken Windows Ledger

> Cross-phase defect register. With `workflow.windows_enforce` enabled, `/gsd-ship` blocks while `open_count > 0`.
> Waive with `gsd-tools windows waive <id> "<reason>"` (reason required).
> Mark fixed with `gsd-tools windows fixed <id>`.

| id | phase | kind | file | line | description | status | reason | recorded_at | resolved_at |
|----|-------|------|------|------|-------------|--------|--------|-------------|-------------|
| 1 | quick-260903-c93 | deviation | agent/command_test.go |  | Test_cliCommand_Run subtests run_consul_server and config-file_non_empty hang indefinitely (no real Consul/etcd/store backend available); confirmed pre-existing on both protovalidate-go v0.6.2 and v0.8.0, unrelated to this task | open |  | 2026-09-03T02:29:11.815Z |  |
| 2 | quick-260904-f5j | deviation | compiler/lib/compiler.go | 355 | go vet: literal copies lock value from c.ModuleService.GetProtoRegistry().MessageRegistry (sync.RWMutex) -- pre-existing, confirmed present at baseline 20d6521, unrelated to this task, out of scope | open |  | 2026-09-04T04:09:44.656Z |  |
| 3 | 14 | lint-warning | agent/agent_test.go | 27 | Pre-existing lostcancel vet finding (context.WithTimeoutCause discard), dates to 2024-03-17, unrelated to Phase 14 -- surfaced only because 14-05 is the first plan to run unscoped go vet ./... | open |  | 2026-09-08T13:58:21.860Z |  |
| 4 | 14 | lint-warning | agent/legacy.go | 97 | Pre-existing unreachable-code vet finding, dates to 2024-05-27, unrelated to Phase 14 -- surfaced only because 14-05 is the first plan to run unscoped go vet ./... | open |  | 2026-09-08T13:58:21.944Z |  |

````json
[
  {
    "id": 1,
    "kind": "deviation",
    "phase": "quick-260903-c93",
    "file": "agent/command_test.go",
    "line": null,
    "description": "Test_cliCommand_Run subtests run_consul_server and config-file_non_empty hang indefinitely (no real Consul/etcd/store backend available); confirmed pre-existing on both protovalidate-go v0.6.2 and v0.8.0, unrelated to this task",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-09-03T02:29:11.815Z",
    "resolved_at": null
  },
  {
    "id": 2,
    "kind": "deviation",
    "phase": "quick-260904-f5j",
    "file": "compiler/lib/compiler.go",
    "line": 355,
    "description": "go vet: literal copies lock value from c.ModuleService.GetProtoRegistry().MessageRegistry (sync.RWMutex) -- pre-existing, confirmed present at baseline 20d6521, unrelated to this task, out of scope",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-09-04T04:09:44.656Z",
    "resolved_at": null
  },
  {
    "id": 3,
    "kind": "lint-warning",
    "phase": "14",
    "file": "agent/agent_test.go",
    "line": 27,
    "description": "Pre-existing lostcancel vet finding (context.WithTimeoutCause discard), dates to 2024-03-17, unrelated to Phase 14 -- surfaced only because 14-05 is the first plan to run unscoped go vet ./...",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-09-08T13:58:21.860Z",
    "resolved_at": null
  },
  {
    "id": 4,
    "kind": "lint-warning",
    "phase": "14",
    "file": "agent/legacy.go",
    "line": 97,
    "description": "Pre-existing unreachable-code vet finding, dates to 2024-05-27, unrelated to Phase 14 -- surfaced only because 14-05 is the first plan to run unscoped go vet ./...",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-09-08T13:58:21.944Z",
    "resolved_at": null
  }
]
````
