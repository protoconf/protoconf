#!/usr/bin/env python3

import asyncio
from protoconf import Protoconf

# Protobuf generated files use absolute imports
import os
import sys
sys.path.append(os.path.join(os.path.dirname(__file__), "."))

from crawler.crawler_pb2 import CrawlerService


class Example(object):
    async def run(self):
        protoconf = Protoconf()
        crawler_service = await protoconf.get_and_subscribe(
            "crawler/text_crawler", CrawlerService, self.got_config
        )
        print("initial config:", crawler_service)

        await self.got_config(crawler_service)
        await protoconf.close()

    async def got_config(self, new_crawler_service):
        self.crawler_service = new_crawler_service


if __name__ == "__main__":
    asyncio.get_event_loop().run_until_complete(Example().run())
