package agent

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	protoconf_agent_config "github.com/protoconf/protoconf/agent/config/v1"
	"github.com/protoconf/protoconf/utils/testdata"
	"google.golang.org/grpc"
)

func TestRunAgent(t *testing.T) {
	type args struct {
		ctx    context.Context
		cancel context.CancelFunc
		config *protoconf_agent_config.AgentConfig
	}
	newArgs := func() args {
		ctx, cancel := context.WithTimeoutCause(context.Background(), time.Second*5, errors.New("time out"))
		return args{
			ctx:    ctx,
			cancel: cancel,
			config: &protoconf_agent_config.AgentConfig{
				GrpcAddress: ":0",
				HttpAddress: ":0",
				DevRoot:     testdata.SmallTestDir(),
			},
		}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "run dev server",
			args: newArgs(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer tt.args.cancel()
			if err := RunAgent(tt.args.ctx, tt.args.config); (err != nil) != tt.wantErr {
				t.Errorf("RunAgent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServeGRPCStoppedBeforeServe(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	rpcServer := grpc.NewServer()
	rpcServer.GracefulStop()
	if err := serveGRPC(rpcServer, lis); err != nil {
		t.Errorf("serveGRPC() on a stopped server = %v, want nil", err)
	}
}
