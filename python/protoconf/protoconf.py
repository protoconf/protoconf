import asyncio
from grpclib.client import Channel

import threading
from concurrent.futures import ThreadPoolExecutor

# Protobuf generated files use absolute imports
import os
import sys
sys.path.append(os.path.join(os.path.dirname(__file__), "."))

from agent.api.proto.v1.protoconf_service_pb2 import ConfigSubscriptionRequest
from agent.api.proto.v1.protoconf_service_grpc import ProtoconfServiceStub


class Protoconf(object):
    def __init__(self, host="127.0.0.1", port=4300):
        self._host = host
        self._port = port
        self._clear_state()

    def _clear_state(self):
        self._channel = None
        self._streams = []
        self._updates_tasks = []

    async def get_and_subscribe(self, path, protobuf_type, callback):
        if self._channel == None:
            self._channel = Channel(self._host, self._port)

        stream = (
            await ProtoconfServiceStub(self._channel)
            .SubscribeForConfig.open()
            .__aenter__()
        )
        await stream.send_message(ConfigSubscriptionRequest(path=path))

        async def get_config():
            protoconf_value = await stream.recv_message()
            if protoconf_value == None:
                return None

            config = protobuf_type()
            protoconf_value.value.Unpack(config)
            return config

        config = await get_config()
        if config != None and callback != None:
            self._streams.append(stream)

            async def get_updates():
                while True:
                    config = await get_config()
                    if config == None:
                        break
                    asyncio.ensure_future(callback(config))

            self._updates_tasks.append(asyncio.ensure_future(get_updates()))
        else:
            await stream.cancel()
        return config

    async def close(self):
        for stream in self._streams:
            await stream.cancel()
            await stream.__aexit__(None, None, None)
        for task in self._updates_tasks:
            task.cancel()
        self._channel.close()
        self._clear_state()


class ProtoconfSync(object):
    def __init__(self, host="127.0.0.1", port=4300):
        self._asyncio_thread = None
        self._protoconf = Protoconf(host, port)
        self._executor = ThreadPoolExecutor()

    def get_and_subscribe(self, path, protobuf_type, callback):
        config = None
        config_set = threading.Event()

        async def async_subscribe():
            async def async_callback(config):
                self._executor.submit(callback, config)

            nonlocal config
            config = await self._protoconf.get_and_subscribe(
                path, protobuf_type, async_callback
            )
            config_set.set()

        if self._asyncio_thread == None:

            def run_loop():
                asyncio.set_event_loop(self._loop)
                self._loop.run_until_complete(self._should_close.wait())

            self._loop = asyncio.new_event_loop()
            self._should_close = asyncio.Event(loop=self._loop)
            self._asyncio_thread = threading.Thread(target=run_loop)
            self._asyncio_thread.start()

        asyncio.run_coroutine_threadsafe(async_subscribe(), self._loop)
        config_set.wait()
        return config

    def close(self):
        if self._asyncio_thread == None:
            return

        async def set_should_close():
            self._should_close.set()
            await self._protoconf.close()

        asyncio.run_coroutine_threadsafe(set_should_close(), self._loop)
        self._asyncio_thread.join()
