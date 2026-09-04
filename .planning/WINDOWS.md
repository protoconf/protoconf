---
schema_version: 1
open_count: 2
waived_count: 0
fixed_count: 0
total_count: 2
last_updated: 2026-09-04T04:09:44.656Z
---

# Broken Windows Ledger

> Cross-phase defect register. With `workflow.windows_enforce` enabled, `/gsd-ship` blocks while `open_count > 0`.
> Waive with `gsd-tools windows waive <id> "<reason>"` (reason required).
> Mark fixed with `gsd-tools windows fixed <id>`.

| id | phase | kind | file | line | description | status | reason | recorded_at | resolved_at |
|----|-------|------|------|------|-------------|--------|--------|-------------|-------------|
| 1 | quick-260903-c93 | deviation | agent/command_test.go |  | Test_cliCommand_Run subtests run_consul_server and config-file_non_empty hang indefinitely (no real Consul/etcd/store backend available); confirmed pre-existing on both protovalidate-go v0.6.2 and v0.8.0, unrelated to this task | open |  | 2026-09-03T02:29:11.815Z |  |
| 2 | quick-260904-f5j | deviation | compiler/lib/compiler.go | 355 | go vet: literal copies lock value from c.ModuleService.GetProtoRegistry().MessageRegistry (sync.RWMutex) -- pre-existing, confirmed present at baseline 20d6521, unrelated to this task, out of scope | open |  | 2026-09-04T04:09:44.656Z |  |

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
  }
]
````
