---
title: protoconf
---
![Protoconf](assets/protoconf_new.svg)

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

### What is protoconf

Protoconf is a configuration management framework. We call it a framework because it provides a platform to manage the entire life cycle of configuration files (configs).
Protoconf is a tool to aid in the specification and distribution of configs.
The goals of specification are to be robust, composable, and less error-prone.
The goals of distribution are to reach all of your machines quickly, reliably, and to be highly available even in disaster scenarios.

### When should you use Protoconf?

- [x] You need to update and distribute configuration dynamically, often to a large number of hosts or services.
- [x] You want to write your configuration with code.
- [x] You need change history.
- [x] You want to code review config changes.
- [x] You want validation that config changes conform to a schema and do not violate invariants that you define.
- [x] You want to canary (test) config changes before distributing them to production.
- [x] You can tolerate eventual consistency; config updates are not atomic w.r.t. different consumers.
- [x] You don't need config updates to propagate to your consumers within a certain SLA.
- [x] Your configs are reasonably small
- [x] You don't need very frequent config updates, more than about once every 5 mins.

### How Protoconf works

#### Configuration update flow

<div align="center">
  <img src="https://lior2b.github.io/temp/protoconf_flow.png">
</div>

#### How this looks from the service's eyes

This is roughly how configuration is consumed by a service. This paradigm encourages you to write software that can reconfigure itself in runtime rather than require a restart:

=== "Python"

	```python
	#!/usr/bin/env python
	import grpc
	from v1.protoconf_service_pb2_grpc import ProtoconfServiceStub
	from v1.protoconf_service_pb2 import ConfigSubscriptionRequest
	from myproject.myconfig_pb2 import MyConfig

	channel = grpc.insecure_channel("localhost:4300")
	stub = ProtoconfServiceStub(channel)

	config = MyConfig()
	for update in stub.SubscribeForConfig(ConfigSubscriptionRequest(path="myproject/myconfig")):
		update.value.Unpack(config)
		print(config)
	```

As Protoconf uses Protobuf and gRPC, it supports delivering configuration to [all major languages](https://github.com/protocolbuffers/protobuf/blob/master/docs/third_party.md). See also: [Protobuf overview](https://developers.google.com/protocol-buffers/docs/overview).
