package compiler

import (
	"context"
	"net"
	"testing"

	protoconf_pb "github.com/protoconf/protoconf/pb/protoconf/v1"
	"github.com/protoconf/protoconf/utils/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func Test_cliCommand_Run(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dir := testdata.SmallTestDir()
	svc := NewCompilerService(dir, true)
	rpcServer := grpc.NewServer()
	protoconf_pb.RegisterProtoconfCompileServer(rpcServer, svc)
	lis, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)

	go func() {
		context.AfterFunc(ctx, func() { rpcServer.GracefulStop() })
		if err := rpcServer.Serve(lis); err != nil {
			t.Logf("error serving server: %v", err)
		}
	}()

	type args struct {
		args []string
	}
	tests := []struct {
		name string
		c    *cliCommand
		args args
		want int
	}{
		{name: "Test Single file", c: &cliCommand{}, args: args{args: []string{dir, "test.pconf"}}, want: 0},
		{name: "Test All files", c: &cliCommand{}, args: args{args: []string{dir}}, want: 1},
		{name: "Test Profiling", c: &cliCommand{}, args: args{args: []string{"-cpuprofile=/dev/null", "-memprofile=/dev/null", dir, "test.pconf"}}, want: 0},
		{name: "Test Remote Single file", c: &cliCommand{}, args: args{args: []string{"-compiler-address", lis.Addr().String(), dir, "test.pconf"}}, want: 0},
		{name: "Test Remote All files", c: &cliCommand{}, args: args{args: []string{"-compiler-address", lis.Addr().String(), dir}}, want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := Command()
			if got := c.Run(tt.args.args); got != tt.want {
				t.Errorf("cliCommand.Run() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_cliCommand_Help(t *testing.T) {
	c := &cliCommand{}
	result := c.Help()
	assert.NotEmpty(t, result)
}
