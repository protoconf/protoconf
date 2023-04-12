# Using protoconf exec

It's very likely that your infra relies on many components which does not have native `protoconf` integration. You can still use protoconf to code their config and use some wrappers around the process you are running in order to write and reload the config. `protoconf exec` aims to be a general purpose way to do so (still WIP and many features are not implemented)

### Import the exec config to your workspace

```shell
$ mkdir -p src/exec
curl -Lo src/exec/exec_config.proto https://raw.githubusercontent.com/protoconf/protoconf/v0.1.3/exec/config/exec_config.proto
```

### Create a dummy proto and config

```protobuf
// file: ./src/myservice/myconfig.proto
syntax = "proto3";
message MyConfig {
  string name = 1;
  uint32 timeout = 2;
}
```

```python
"""
file: ./src/myservice/myconfig.pconf
"""
load("myconfig.proto", "MyConfig")

config = MyConfig(name="test", timeout=3)

def main():
    return config
```

### Create an `exec` config

```python
"""
Generates tf.json files under ./tfconfigs
"""
load(
    "//exec/exec_config.proto",
    "Config",
    "WatcherConfig",
    "Action",
    "ActionTypeWriteToFile",
)

configs = [
    "mysservice/myconfig"
]


def main():
    return Config(
        items=[
            WatcherConfig(
                path=path,
                proto_file="myservice/myconfig.proto",
                actions=[
                    Action(file=ActionTypeWriteToFile(path="tfconfigs/%s.json" % path))
                ],
            )
            for path in configs
        ]
    )
```

