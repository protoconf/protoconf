package com.protoconf.examples;

import com.google.protobuf.InvalidProtocolBufferException;
import com.protoconf.agent.api.v1.ProtoconfServiceGrpc;
import com.protoconf.agent.api.v1.ProtoconfServiceOuterClass;
import com.protoconf.examples.address_book.AddressBookOuterClass;
import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;
import io.grpc.stub.StreamObserver;

public class JavaClient {
    public static void main(String[] args) {
        ManagedChannel channel = ManagedChannelBuilder.forAddress("localhost", 4300).usePlaintext().build();
        ProtoconfServiceGrpc.ProtoconfServiceStub asyncStub = ProtoconfServiceGrpc.newStub(channel);
        ProtoconfServiceOuterClass.ConfigSubscriptionRequest request = ProtoconfServiceOuterClass.ConfigSubscriptionRequest.newBuilder().setPath("/address_book/my_address_book").build();
        asyncStub.subscribeForConfig(request, new StreamObserver<ProtoconfServiceOuterClass.ConfigUpdate>() {
            boolean firstRead = true;

            @Override
            public void onNext(ProtoconfServiceOuterClass.ConfigUpdate value) {
                try {
                    AddressBookOuterClass.AddressBook config = value.getValue().unpack(AddressBookOuterClass.AddressBook.class);
                    if (firstRead) {
                        firstRead = false;
                        System.out.println("Config initial value: " + config);
                    } else {
                        System.out.println("Config changed, new value: " + config);
                    }
                } catch (InvalidProtocolBufferException e) {
                    e.printStackTrace();
                }
            }

            @Override
            public void onError(Throwable t) {
                System.err.println("Error reading config, error: " +  t);
            }

            @Override
            public void onCompleted() {
            }
        });

        // Program code runs here while config asynchronously arrives
        try {
            Thread.sleep(1000 * 60 *60);
        } catch (InterruptedException e) {
        }
    }
}
