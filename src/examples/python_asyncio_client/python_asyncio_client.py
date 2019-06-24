import asyncio
from contextlib import closing
from grpclib.client import Channel

from agent.api.proto.v1.protoconf_service_pb2 import ConfigSubscriptionRequest
from agent.api.proto.v1.protoconf_service_grpc import ProtoconfServiceStub
from examples.protoconf.src.address_book.address_book_pb2 import AddressBook


async def main():
    with closing(Channel("127.0.0.1", 4300)) as channel:
        protoconf_service = ProtoconfServiceStub(channel)
        first_read = True
        address_book = AddressBook()

        async with protoconf_service.SubscribeForConfig.open() as stream:
            await stream.send_message(
                ConfigSubscriptionRequest(path="/address_book/my_address_book")
            )
            while True:
                config_update = await stream.recv_message()
                config_update.value.Unpack(address_book)
                if first_read:
                    first_read = False
                    print("Config initial value:", address_book)
                else:
                    print("Config changed, new value:", address_book)


if __name__ == "__main__":
    asyncio.get_event_loop().run_until_complete(main())
