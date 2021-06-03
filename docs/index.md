---
title: protoconf
---
![Protoconf](assets/protoconf_new.svg){ align=left }
#### Codify configuration, Instant delivery

![Test](https://github.com/protoconf/protoconf/workflows/Bazel/badge.svg)
![Release](https://github.com/protoconf/protoconf/workflows/Release/badge.svg)
[![codecov](https://codecov.io/gh/protoconf/protoconf/branch/master/graph/badge.svg)](https://codecov.io/gh/protoconf/protoconf)

## Introduction
Modern services are comprised of many dynamic variables, that need to be changed regularly. Today, the process is unstructured and error prone. From ML model variables, kill switches, gradual rollout configuration, A/B experiment configuration and more - developers want their code to allow to be configured to the finer details.

**Protoconf is a modern approach to software configuration**, inspired by [Facebook's Configerator](https://research.fb.com/publications/holistic-configuration-management-at-facebook/).

Using Protoconf enables:

* **Code review for configuration changes**
  Enables the battle tested flow of pull-request & code-review. Configuration auditing out of the box (who did what, when?). The repository is the source of truth for the configuration deployed to production.
* **No service restart required to pick up changes**
  Instant delivery of configuration updates. Encourages writing software that doesn't know downtime.
* **Clear representation of complex configuration**
  Write configuration in Starlark (a Python dialect), no more copying & pasting from huge JSON files.
* **Automated validation**
  Config follows a fully-typed (Protobuf) schema. This allows writing validation code in Starlark, to verify your configuration before it is committed.

#### Configuration update flow

<div align="center">
  <img src="https://lior2b.github.io/temp/protoconf_flow.png">
</div>

#### How this looks from the service's eyes

This is roughly how configuration is consumed by a service. This paradigm encourages you to write software that can reconfigure itself in runtime rather than require a restart:

<div align="center">
  <img src="https://lior2b.github.io/temp/protoconf_api.png" width="400">
</div>

As Protoconf uses Protobuf and gRPC, it supports delivering configuration to [all major languages](https://github.com/protocolbuffers/protobuf/blob/master/docs/third_party.md). See also: [Protobuf overview](https://developers.google.com/protocol-buffers/docs/overview).
