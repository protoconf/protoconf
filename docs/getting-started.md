# Getting Started

Prerequisite for this guide:

- Knowledge in Python
- Familiarity with [Protobuf](https://developers.google.com/protocol-buffers/) and [gRPC](https://grpc.io/)

### Define your first config

The first step will be to define the config struct in `protobuf`. The `protobuf` file will be used to generate a marshaler of in the language of choice which can be used alongside the gRPC client to pull configs from the `protoconf agent` gRPC endpoint.

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

### Code your config

Create a `.pconf` file to populate the config struct with the required values.

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

### Compile and check results

Our working directory is now ready to be compiled. Run `protoconf compile .`. The compiler will create a new file under `materialized_configs/myproject/myconfig.materialized_JSON` which can be used to validate the result of the config.

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

### Add validators

You might want to make sure no one can accidentally reduce the `connection_timeout` config below `3`. If he wish to do so, he can add a validator to the `MyConfig` struct:

```python
"""
file: ./src/myproject/myconfig.proto-validator
"""
load("myconfig.proto", "MyConfig")

def validate_connection_timeout(config):
    if config.connection_timeout <= 3:
        fail("connection_timeout must be 3 or higher, got: %d" % config.connection_timeout)

add_validator(MyConfig, validate_connection_timeout)
```

### Consume your config locally

To test his configs locally, you can run `protoconf agent -dev .`
The agent is now running and listening on `0.0.0.0:4300` and ready to accept gRPC calls.

Install the `grpc` and `protobuf` tools to generate the `stub` code to communicate with the protoconf gRPC agent.

```shell
$ pip install grpcio-tools
$ python -m grpc_tools.protoc -Isrc --python_out=. --grpc_python_out=. ./src/myproject/myconfig.proto
$ git clone https://github.com/protoconf/protoconf.git
$ python -m grpc_tools.protoc -Iprotoconf/agent/api/proto/ --python_out=. --grpc_python_out=. protoconf/agent/api/proto/v1/protoconf_service.proto
```

Write a simple python code that will use the generated code to communicate with the agent.
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

Run the python code and make a change to the `./src/myproject/myconfig.pconf`. After running `protoconf compile .` again, you will see the config changes in your running software.

### Prepare for Production

Use a supported KV store to release the config to production. The supported storages are: [Consul](https://www.consul.io), [Etcd](https://www.etcd.io) or [Zookeeper](https://zookeeper.apache.org/).

```shell
$ consul agent -dev &
$ protoconf insert -store consul -store-address localhost:8500 . myproject/myconfig
```

### Run the agent in production mode

```shell
$ protoconf agent -store consul -store-address localhost:8500
```

Run your code the same way as step 5. Then make a change, compile and run the `protoconf insert` command from step 6 again.