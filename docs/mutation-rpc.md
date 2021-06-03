# Mutation RPC usage

One of the core principals of `protoconf` is the ability to mutate configs via an API (or RPC). This allows humans and machines work together on the same configuration codebase. `protoconf` allow humans to code the logic, while machines can only change values via the RPC.

### Create a dummy config

```proto
// file: ./src/myservice/myconfig.proto
syntax = "proto3";
message MyConfig {
  string name = 1;
  uint32 timeout = 2;
}
```

```python
"""
file: ./src/myservice/default.pconf
"""
load("myconfig.proto", "MyConfig")

config = MyConfig(name="config")

def main():
    return config
```

```shell
$ protoconf compile .
```

### Create a post-mutation script

```shell
$ echo '#!/bin/bash\nprotoconf compile .' > post.sh
$ chmod +x post.sh
```

### Run the mutation server in the background

```shell
$ protoconf serve -post ./post.sh . &
```

Now we will use the `protoconf mutate` command to hit the mutation RPC
```shell
$ protoconf mutate -path myservice/mutation -proto myservice/myconfig.proto -msg MyConfig -field timeout=3
```

You will now notice a new file created under `mutable_config`

```shell
 find mutable_config
$ mutable_config
mutable_config/myservice
mutable_config/myservice/mutation.materialized_JSON
```

### load the mutated values
```python
"""
file: ./src/myservice/default.pconf
"""
load("myconfig.proto", "MyConfig")
load("mutate:myservice/mutation", "value")

config = MyConfig(name="config", timeout=value.timeout)

def main():
    return config
```

Run the `protoconf mutate` command again with different value and watch how your configs changes.

## Next steps

### Running in production

The `protoconf serve` command accepts `-pre` and `-post` scripts which should be used for preparing the ground for writing the mutation (`-pre`) and followup actions to run after writing the mutation (`-post`). 

Both scripts will run by `protoconf serve` on every mutation. The scripts will be receiveing a `metadata` string as its first argument (`$1` in `bash`) and can be used to pass metadata from the initiator of the rpc to the script, This can be used to pass a token for github to validate the initiator or to pass more context to be added to the commit message.

These scripts should handle the `git` lifecycle of the mutation (setting the workspace to latest ref, and push the result after the writing done.)

compiling the configs should be part of the `-post` script.

When running in HA, you can use these scripts to acquire a lock from `consul`/`etcd`.

### Using gRPC

The mutation proto is available [here](https://github.com/protoconf/protoconf/blob/v0.1.3/server/api/proto/v1/protoconf_mutation.proto).
