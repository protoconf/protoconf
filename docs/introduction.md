# Introduction

Modern services are comprised of many dynamic variables, that need to be changed regularly. Today, the process is unstructured and error prone. From ML model variables, kill switches, gradual rollout configuration, A/B experiment configuration and more - developers want their code to allow to be configured to the finer details.
Protoconf is a modern approach to software configuration, inspired by [Facebook's Configerator](https://research.fb.com/publications/holistic-configuration-management-at-facebook/).
Using Protoconf enables:

- Code review for configuration changes Enables the battle tested flow of pull-request & code-review. Configuration auditing out of the box (who did what, when?). The repository is the source of truth for the configuration deployed to production.
- No service restart required to pick up changes Instant delivery of configuration updates. Encourages writing software that doesn't know downtime.
- Clear representation of complex configuration Write configuration in Starlark (a Python dialect), no more copying & pasting from huge JSON files.
- Automated validation Config follows a fully-typed (Protobuf) schema. This allows writing validation code in Starlark, to verify your configuration before it is committed.

Protoconf is a configuration management framework. We call it a framework because it provides a platform to manage the entire life cycle of configuration files (configs).
Protoconf is a tool to aid in the specification and distribution of configs.
The goals of specification are to be robust, composable, and less error-prone.
The goals of distribution are to reach all of your machines quickly, reliably, and to be highly available even in disaster scenarios.
When should you use Protoconf?

- You need to update and distribute configuration dynamically, often to a large number of hosts or services.
- You want to write your configuration with code.
- You need change history.
- You want to code review config changes.
- You want validation that config changes conform to a schema and do not violate invariants that you define.
- You want to canary (test) config changes before distributing them to production.
- You can tolerate eventual consistency; config updates are not atomic w.r.t. different consumers.
- You don't need config updates to propagate to your consumers within a certain SLA.
- Your configs are reasonably small
- You don't need very frequent config updates, more than about once every 5 mins.
