#!/usr/bin/env ruby

require 'google/protobuf/well_known_types'
require 'grpc'
require 'agent/api/proto/v1/protoconf_service_services_pb'
require 'examples/protoconf/src/address_book/address_book_pb'

def main
    stub = V1::ProtoconfService::Stub.new("localhost:4300", :this_channel_is_insecure)
    request = V1::ConfigSubscriptionRequest.new(path: "/address_book/my_address_book")
    resps = stub.subscribe_for_config(request)
    first_read = true
    resps.each do |r|
        address_book = r.value.unpack AddressBook
        if first_read
            first_read = false
            puts "Config initial value: " + AddressBook.encode_json(address_book)
        else
            puts "Config changed, new value: " + AddressBook.encode_json(address_book)
        end
    end
end
  
main
