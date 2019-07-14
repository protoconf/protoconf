#!/usr/bin/env python3

from contextlib import closing
from protoconf import ProtoconfSync, ProtoconfMutationSync
from time import time

# Protobuf generated files use absolute imports
import os
import sys
sys.path.append(os.path.join(os.path.dirname(__file__), "."))

from crawler.crawler_pb2 import CrawlerService, Crawler


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


class MutationExample(object):
    def run(self, script_metadata):
        with closing(ProtoconfMutationSync()) as protoconf:
            protoconf.mutate_config(
                "crawler/extra_crawler",
                Crawler(
                    user_agent="Linux/ time=%s"%int(time()),
                    http_timeout=30,
                ),
                script_metadata,
            )


if __name__ == "__main__":
    if len(sys.argv) > 2 and sys.argv[1] == "mutate":
        script_metadata = sys.argv[2]
        MutationExample().run(script_metadata)
    else:
        Example().run()
