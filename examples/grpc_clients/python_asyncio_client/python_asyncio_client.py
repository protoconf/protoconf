import asyncio
from contextlib import closing
from grpclib.client import Channel

from agent.api.proto.v1.protoconf_service_pb2 import ConfigSubscriptionRequest
from agent.api.proto.v1.protoconf_service_grpc import ProtoconfServiceStub
from examples.protoconf.src.crawler.crawler_pb2 import CrawlerService


async def main():
    with closing(Channel("127.0.0.1", 4300)) as channel:
        protoconf_service = ProtoconfServiceStub(channel)
        first_read = True
        crawler_service = CrawlerService()

        async with protoconf_service.SubscribeForConfig.open() as stream:
            await stream.send_message(
                ConfigSubscriptionRequest(path="crawler/text_crawler")
            )
            while True:
                config_update = await stream.recv_message()
                config_update.value.Unpack(crawler_service)
                if first_read:
                    first_read = False
                    print("Config initial value:", crawler_service)
                else:
                    print("Config changed, new value:", crawler_service)


if __name__ == "__main__":
    asyncio.get_event_loop().run_until_complete(main())
