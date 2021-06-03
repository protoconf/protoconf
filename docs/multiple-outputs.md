# Multiple outputs

Sometimes, we want to generate multiple configs from a single file. for this, we can use the `.mpconf`.

`protoconf` expects `.mpconf`'s `main()` function to return a `dict` with a `string` as the key and a `proto.Message` as the value.

Example:

```proto
// file: src/myservice/myconfig.proto
syntax = "proto3";
message MyConfig {
  string name = 1;
  uint32 timeout = 2;
}
```

```python
"""
file: ./src/myservice/outputs.mpconf
"""
load("myconfig.proto", "MyConfig")

out = {}

for i in range(5):
    out["config%d" % i] = MyConfig(name="config%d" % i, timeout=i * 3)

def main():
    return out
```

Now, when you compile your configs, you will see multiple outputs

```shell
$ protoconf compile .
$ find materialized_config/myservice
materialized_config/myservice
materialized_config/myservice/outputs
materialized_config/myservice/outputs/config4.materialized_JSON
materialized_config/myservice/outputs/config1.materialized_JSON
materialized_config/myservice/outputs/config0.materialized_JSON
materialized_config/myservice/outputs/config3.materialized_JSON
materialized_config/myservice/outputs/config2.materialized_JSON
```
