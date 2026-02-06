package agent

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"

	protoconfagent "github.com/protoconf/protoconf/agent/api/proto/v1"
	protoconf_pb "github.com/protoconf/protoconf/pb/protoconf/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
)

type legacyProtoconfServer struct {
	protoconfagent.UnimplementedProtoconfServiceServer
	stub protoconf_pb.ProtoconfServiceClient
}

func newLegacyProtoconfServer(srv protoconf_pb.ProtoconfServiceServer) *legacyProtoconfServer {
	buffer := 1024 * 1024
	lis := bufconn.Listen(buffer)
	ctx := context.Background()
	rpcServer := grpc.NewServer()
	protoconf_pb.RegisterProtoconfServiceServer(rpcServer, srv)
	go func() {
		context.AfterFunc(ctx, func() {
			slog.Info("shutting down rpc server")
			rpcServer.GracefulStop()
		})
		if err := rpcServer.Serve(lis); err != nil {
			log.Printf("error serving server: %v", err)
		}
	}()

	conn, err := grpc.DialContext(ctx, "",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("error connecting to server: %v", err)
	}
	stub := protoconf_pb.NewProtoconfServiceClient(conn)
	return &legacyProtoconfServer{
		stub: stub,
	}
}

func upgrade(in proto.Message, out proto.Message) error {
	b, err := proto.Marshal(in)
	if err != nil {
		return err
	}
	err = proto.Unmarshal(b, out)
	if err != nil {
		return err
	}
	return nil
}

func (s *legacyProtoconfServer) GetConfig(ctx context.Context, request *protoconfagent.ConfigRequest) (*protoconfagent.ConfigUpdate, error) {
	next := &protoconf_pb.ConfigRequest{}
	err := upgrade(request, next)
	if err != nil {
		return nil, err
	}
	resp, err := s.stub.GetConfig(ctx, next)
	if err != nil {
		return nil, err
	}
	result := &protoconfagent.ConfigUpdate{}
	err = upgrade(resp, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *legacyProtoconfServer) GetJsonConfig(ctx context.Context, request *protoconfagent.ConfigRequest) (*protoconfagent.ConfigUpdate, error) {
	next := &protoconf_pb.ConfigRequest{}
	err := upgrade(request, next)
	if err != nil {
		return nil, err
	}
	resp, err := s.stub.GetConfig(ctx, next)
	if err != nil {
		return nil, err
	}
	result := &protoconfagent.ConfigUpdate{}
	err = upgrade(resp, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *legacyProtoconfServer) GetJsonConfigHttp(w http.ResponseWriter, r *http.Request) {
	config := r.URL.Query().Get("config")
	if config == "" {
		http.Error(w, "Missing required 'config' query parameter", http.StatusBadRequest)
		return
	}
	cfg, err := s.GetJsonConfig(context.Background(), &protoconfagent.ConfigRequest{
		Path: config,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("Unable to fetch config [%s]", config), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(cfg.GetRaw())
}

func (s *legacyProtoconfServer) SubscribeForConfig(request *protoconfagent.ConfigSubscriptionRequest, srv protoconfagent.ProtoconfService_SubscribeForConfigServer) error {
	next := &protoconf_pb.ConfigSubscriptionRequest{}
	err := upgrade(request, next)
	if err != nil {
		return err
	}

	ctx := srv.Context()
	stream, err := s.stub.SubscribeForConfig(ctx, next)
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msg, err := stream.Recv()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return err
			}
			result := &protoconfagent.ConfigUpdate{}
			err = upgrade(msg, result)
			if err != nil {
				return err
			}
			srv.Send(result)
		}
	}

	return nil
}
