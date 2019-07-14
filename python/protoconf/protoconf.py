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

from server.api.proto.v1.protoconf_mutation_pb2 import ConfigMutationRequest
from server.api.proto.v1.protoconf_mutation_grpc import ProtoconfMutationServiceStub

from datatypes.proto.v1.protoconf_value_pb2 import ProtoconfValue

AGENT_DEFAULT_PORT = 4300
SERVER_DEFAULT_PORT = 4301

class Protoconf(object):
    def __init__(self, host="127.0.0.1", port=AGENT_DEFAULT_PORT):
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
    def __init__(self, host="127.0.0.1", port=AGENT_DEFAULT_PORT, executor=None):
        self._asyncio_thread = None
        self._protoconf = Protoconf(host, port)
        self._executor = executor if executor != None else ThreadPoolExecutor()

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
            self._asyncio_thread = threading.Thread(target=run_loop) # should this use the executor?
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


class ProtoconfMutation(object):
    def __init__(self, host="127.0.0.1", port=SERVER_DEFAULT_PORT):
        self._host = host
        self._port = port
        self._clear_state()

    def _clear_state(self):
        self._channel = None

    async def mutate_config(self, path, value, script_metadata):
        if self._channel == None:
            self._channel = Channel(self._host, self._port)

        protoconf_service = ProtoconfMutationServiceStub(self._channel)

        config = ProtoconfValue()
        config.proto_file = value.DESCRIPTOR.file.name
        config.value.Pack(value)

        await protoconf_service.MutateConfig(ConfigMutationRequest(path=path, value=config, script_metadata=script_metadata))

    def close(self):
        self._channel.close()
        self._clear_state()

class ProtoconfMutationSync(object):
    def __init__(self, host="127.0.0.1", port=SERVER_DEFAULT_PORT):
        self._protoconf = ProtoconfMutation(host, port)

    def mutate_config(self, path, value, script_metadata):
        async def async_mutate():
            await self._protoconf.mutate_config(path, value, script_metadata)

        loop = asyncio.new_event_loop()
        def run_loop():
            asyncio.set_event_loop(loop)
            loop.run_until_complete(async_mutate())

        asyncio_thread = threading.Thread(target=run_loop)
        asyncio_thread.start()
        asyncio_thread.join()

    def close(self):
        self._protoconf.close()
