package agent

import (
	"context"
	"errors"
	"testing"
	"time"

	protoconf_agent_config "github.com/protoconf/protoconf/agent/config/v1"
)

func TestRunAgent(t *testing.T) {
	type args struct {
		ctx    context.Context
		config *protoconf_agent_config.AgentConfig
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "run dev server",
			args: args{
				ctx: func() context.Context {
					ctx, _ := context.WithTimeoutCause(context.Background(), time.Second*5, errors.New("time out"))
					return ctx
				}(),
				config: &protoconf_agent_config.AgentConfig{
					GrpcAddress: ":0",
					HttpAddress: ":0",
					DevRoot:     "/some/fake/root",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := RunAgent(tt.args.ctx, tt.args.config); (err != nil) != tt.wantErr {
				t.Errorf("RunAgent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
