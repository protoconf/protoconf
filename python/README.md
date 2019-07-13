Install with:
```bash
pip3 install -r requirements.txt ./
```

To regenerate Protobuf files:
```bash
pip3 install grpcio-tools
python3 -m grpc_tools.protoc --python_out=. --python_grpc_out=. -I../../ ../../{agent/api/proto/v1/protoconf_service.proto,server/api/proto/v1/protoconf_mutation.proto,types/proto/v1/protoconf_value.proto}
```
