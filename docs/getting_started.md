# Getting Started

Prerequisite for this guide:

- Knowledge in Python
- Familiarity with [Protobuf](https://developers.google.com/protocol-buffers/) and [gRPC](https://grpc.io/)

### Step 1

The developer will define the config struct in `protobuf`. The `protobuf` file will be used to generate a marshaler of in the language of choice which can be used alongside the gRPC client to pull configs from the `protoconf agent` gRPC endpoint.

```protobuf
// file: ./src/myproject/myconfig.proto
syntax = "proto3";

message MyConfig {
    uint32 connection_timeout = 1;
    uint32 max_retries = 2;
    NestedStruct another_struct = 3;
}

message NestedStruct {
    string hello_world = 1;
}
```

### Step 2

The developer will then create a `.pconf` file to populate the config struct with the required values.

```python
"""
file: ./src/myproject/myconfig.pconf
"""
load("myconfig.proto", "MyConfig", "NestedStruct")

def main():
    return MyConfig(
        connection_timeout=5,
        max_retries=5,
        another_struct=NestedStruct(
            hello_world="Hello World!"
        )
    )
```

### Step 3

Our working directory is now ready to be compiled. when the developer will run `protoconf compile .`, The compiler will create a new file under `materialized_configs/myproject/myconfig.materialized_JSON` which can be used to validate the result of the config.

```json
// file: materialized_configs/myproject/myconfig.materialized_JSON
{
  "protoFile": "myproject/myconfig.proto",
  "value": {
    "@type": "type.googleapis.com/MyConfig",
    "connectionTimeout": 5,
    "maxRetries": 5,
    "anotherStruct": {
      "helloWorld": "Hello World!"
    }
  }
}
```

### Step 4

The developer might want to make sure no one can accidentally reduce the `connection_timeout` config below `3`. If he wish to do so, he can add a validator to the `MyConfig` struct:

```py
"""
file: ./src/myproject/myconfig.proto-validator
"""
load("myconfig.proto", "MyConfig")

def validate_connection_timeout(config):
    if config.connection_timeout < 3:
        fail("connection_timeout must be 3 or higher, got: %d" % config.connection_timeout)

add_validator(MyConfig, validate_connection_timeout)
```

### Step 5

The developer is now ready to test his configs locally. He can do so by running `protoconf agent -dev .`
The agent is now running and listening on `0.0.0.0:4300` and ready to accept gRPC calls

```shell
$ pip install grpcio-tools
$ python -m grpc_tools.protoc -Isrc --python_out=. --grpc_python_out=. ./src/myproject/myconfig.proto
$ git clone https://github.com/protoconf/protoconf.git
$ python -m grpc_tools.protoc -Iprotoconf/agent/api/proto/ --python_out=. --grpc_python_out=. protoconf/agent/api/proto/v1/protoconf_service.proto
```

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

Now the developer can make a change to the `./src/myproject/myconfig.pconf` and run `protoconf compile .` and see the config changes in her running software.

### Step 6

Use a supported KV store to release the config to production. The supported storages are: `consul`, `etcd` and `zookeeper`.

```sh
$ consul agent -dev &
$ protoconf insert -store consul -store-address localhost:8500 . myproject/myconfig
```

### Step 7

Run the agent in production mode

```sh
$ protoconf agent -store consul -store-address localhost:8500
```

### Step 8

Run your code the same way as step 5. Then make a change, compile and run the `protoconf insert` command from step 6 again.
