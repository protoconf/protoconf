package test

import (
	"context"
	"log/slog"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

type RegServer func(s *grpc.Server)
type AddConnectionToClient func(conn *grpc.ClientConn)

func TestServer(ctx context.Context, regServer RegServer, addConnectionToClient AddConnectionToClient) func() {
	buffer := 101024 * 1024
	lis := bufconn.Listen(buffer)
	baseServer := grpc.NewServer()
	regServer(baseServer)
	go func() {
		context.AfterFunc(ctx, func() { baseServer.GracefulStop() })
		if err := baseServer.Serve(lis); err != nil {
			slog.Error("error serving server", "error", err)
		}
	}()

	conn, err := grpc.DialContext(ctx, "",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("error connecting to server", "error", err)
	}

	closer := func() {
		err := lis.Close()
		if err != nil {
			slog.Error("error closing listener", "error", err)
		}
		baseServer.Stop()
	}

	addConnectionToClient(conn)

	return closer
}
