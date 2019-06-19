import asyncio
from contextlib import closing
from grpclib.client import Channel

from agent.api.proto.v1.protoconf_service_pb2 import ConfigSubscriptionRequest
from agent.api.proto.v1.protoconf_service_grpc import ProtoconfServiceStub
from examples.protoconf.src.my_client.client_config_pb2 import ClientConfig


async def main():
    with closing(Channel("127.0.0.1", 4300)) as channel:
        protoconf_service = ProtoconfServiceStub(channel)
        first_read = True
        client_config = ClientConfig()

        async with protoconf_service.SubscribeForConfig.open() as stream:
            await stream.send_message(
                ConfigSubscriptionRequest(path="/my_client/my_client_config")
            )
            while True:
                config_update = await stream.recv_message()
                config_update.value.Unpack(client_config)
                if first_read:
                    first_read = False
                    print("Config initial value:", client_config.value)
                else:
                    print("Config changed, new value:", client_config.value)


if __name__ == "__main__":
    asyncio.get_event_loop().run_until_complete(main())
