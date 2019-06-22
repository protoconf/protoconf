#!/usr/bin/env ruby

require 'google/protobuf/well_known_types'
require 'grpc'
require 'agent/api/proto/v1/protoconf_service_services_pb'
require 'examples/protoconf/src/my_client/client_config_pb'

def main
    stub = V1::ProtoconfService::Stub.new("localhost:4300", :this_channel_is_insecure)
    request = V1::ConfigSubscriptionRequest.new(path: "/my_client/my_client_config")
    resps = stub.subscribe_for_config(request)
    first_read = true
    resps.each do |r|
        client_config = r.value.unpack ClientConfig
        if first_read
            first_read = false
            puts "Config initial value: " + client_config.value
        else
            puts "Config changed, new value: " + client_config.value
        end
    end
end
  
main
