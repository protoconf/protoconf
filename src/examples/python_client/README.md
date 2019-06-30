Install the protoconf dependency with:
```bash
pip3 install -r ../../python/requirements.txt ../../python/
```

To regenerate Protobuf files:
```bash
pip3 install grpcio-tools
python3 -m grpc_tools.protoc --python_out=. -I../protoconf/src ../protoconf/src/crawler/crawler.proto
```
