# Protoconf Architecture Reference

## Infrastructure Requirements

1. A git repository hooked to your favorite CI/CD pipeline (We recommend Github/Gitlab)
2. A cluster of either consul, etcd or zookeeper. Consult the production guides of the respected software.

## Infrastructure Diagram

![Image: Protoconf Reference Architecture](https://raw.githubusercontent.com/protoconf/protoconf/master/docs/Protoconf_Reference_Architecture.png)

### protoconf insert

The `protoconf insert` command is used to read the configs from the `materialized_config` directory and insert them into your storage.

Its recommend that you will use a CI/CD pipeline to insert the configs after they have been pushed to the `master` branch.

The command has a `-prefix` flag which can help you push the configs to different prefixes. We recommend that you will use the `production` prefix for production and use other prefixes testing a Pull Request.
The machine which runs the `protoconf insert` command must have access to the storage.
If you have a significant number of configs and you want to limit your insertions to only what changed since last insertion, we recommend that you make sure to clone the repo with enough history to calculate the diff.

An example insertion script:

```sh
#!/bin/bash
set -e
set -x

echo "VARS: $*"
STORE_TYPE=consul
STORE_ADDRESS="http://127.0.0.1:8500"
PREFIX="production"
SHOULD_DELETE="false"
CHANGES_ONLY="false"
BULK_SIZE="10"

export GIT_TERMINAL_PROMPT=0
git config --global credential.helper "store --file=/local/git-creds"
mkdir -p /tmp/repo
git clone $GIT_REPO /tmp/repo || cd /tmp/repo
cd /tmp/repo
git fetch -a
git reset --hard $AFTER_COMMIT
if [ "${CHANGES_ONLY}" == "true" ]; then
  CONFIGS_TO_INSERT=$(git diff --name-only --diff-filter=ACMR ${AFTER_COMMIT:-HEAD} ${BEFORE_COMMIT:-HEAD~1} | sed -n 's/^materialized_config\///p')
else
  CONFIGS_TO_INSERT=$(find . -name *.materialized_JSON | sed -n 's/^.\/materialized_config\///p')
fi
if [ -z "$CONFIGS_TO_INSERT" ]; then
    exit 0
fi
DELETE=""
if [ "${SHOULD_DELETE}" == "true" ]; then
  DELETE="-d"
fi

echo $CONFIGS_TO_INSERT | xargs -n${BULK_SIZE} protoconf insert \
  -store "${STORE_TYPE}" \
  -store-address "${STORE_ADDRESS}" \
  -prefix "${PREFIX}/" \
  ${DELETE} \
  "$GITHUB_WORKSPACE"
```

### protoconf agent

The `protoconf agent` is the grpc server used to subscribe for the config changes. There are many options to deploy it.

1. Central agent or pool of agents behind a grpc aware load balancer
2. One agent per host
3. A sidecar agent to each task requires config updates from protoconf.
4. Implement `libprotoconf` inside your code (official Go implementation is available) and read directly from your storage.

Here are few considerations:

1. Are you planning to use protoconf to retrieve secrets from a secret store (Vault, KMS etc)? In future versions, we are planning to support dynamic values plugins that will allow the agent to replace static values with dynamic ones like secrets from a secret store. In this case, you should run the agent as close as possible to the context of the task requires the secret, so you can manage the access to the secret appropriately.
2. Does most of your workload run on PaaS/FaaS? If so, injecting sidecars might be hard and you should use a centralized cluster of agents behind a load balancer.

## Datacenter Design

### Single datacenter

### Multiple datacenters

## Network Connectivity

## Summary
