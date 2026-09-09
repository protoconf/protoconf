package agent

import (
	"context"
	"errors"
	"testing"
	"time"

	protoconf_agent_config "github.com/protoconf/protoconf/agent/config/v1"
	"github.com/protoconf/protoconf/utils/testdata"
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
