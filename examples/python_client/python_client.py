#!/usr/bin/env python3

from protoconf import ProtoconfSync

# Protobuf generated files use absolute imports
import os
import sys
sys.path.append(os.path.join(os.path.dirname(__file__), "."))

from crawler.crawler_pb2 import CrawlerService


class Example(object):
    def run(self):
        protoconf = ProtoconfSync()
        crawler_service = protoconf.get_and_subscribe(
            "crawler/text_crawler", CrawlerService, self.got_config
        )
        print("initial config:", crawler_service)

        self.got_config(crawler_service)
        protoconf.close()

    def got_config(self, new_crawler_service):
        self.crawler_service = new_crawler_service


if __name__ == "__main__":
    Example().run()
