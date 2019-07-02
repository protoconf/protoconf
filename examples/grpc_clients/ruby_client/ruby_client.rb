#!/usr/bin/env ruby

require 'google/protobuf/well_known_types'
require 'grpc'
require 'agent/api/proto/v1/protoconf_service_services_pb'
require 'examples/protoconf/src/crawler/crawler_pb'

def main
    stub = V1::ProtoconfService::Stub.new("localhost:4300", :this_channel_is_insecure)
    request = V1::ConfigSubscriptionRequest.new(path: "crawler/text_crawler")
    resps = stub.subscribe_for_config(request)
    first_read = true
    resps.each do |r|
        crawler_service = r.value.unpack CrawlerService
        if first_read
            first_read = false
            puts "Config initial value: " + CrawlerService.encode_json(crawler_service)
        else
            puts "Config changed, new value: " + CrawlerService.encode_json(crawler_service)
        end
    end
end
  
main
