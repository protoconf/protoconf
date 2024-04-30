package server

import (
	"context"
	"os"
	"testing"

	protoconf_pb "github.com/protoconf/protoconf/pb/protoconf/v1"
	"github.com/protoconf/protoconf/utils/testdata"
)

func Test_server_MutateConfig(t *testing.T) {
	type fields struct {
		config        *cliConfig
		protoconfRoot string
	}
	type args struct {
		ctx context.Context
		in  *protoconf_pb.ConfigMutationRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *protoconf_pb.ConfigMutationResponse
		wantErr bool
	}{
		{
			name: "test no workspace",
			fields: fields{
				config:        &cliConfig{},
				protoconfRoot: os.TempDir(),
			},
			args: args{
				ctx: context.Background(),
				in: &protoconf_pb.ConfigMutationRequest{
					Path:  "test",
					Value: &protoconf_pb.ProtoconfValue{},
				},
			},
			want:    &protoconf_pb.ConfigMutationResponse{},
			wantErr: false,
		},
		{
			name: "test",
			fields: fields{
				config:        &cliConfig{},
				protoconfRoot: testdata.SmallTestDir(),
			},
			args: args{
				ctx: context.Background(),
				in: &protoconf_pb.ConfigMutationRequest{
					Path: "test",
					Value: &protoconf_pb.ProtoconfValue{
						ProtoFile: "test.proto",
					},
				},
			},
			want:    &protoconf_pb.ConfigMutationResponse{},
			wantErr: false,
		},
		{
			name: "run scripts",
			fields: fields{
				config: &cliConfig{
					preMutationScript:  "true",
					postMutationScript: "true",
				},
				protoconfRoot: testdata.SmallTestDir(),
			},
			args: args{
				ctx: context.Background(),
				in: &protoconf_pb.ConfigMutationRequest{
					Path: "test",
					Value: &protoconf_pb.ProtoconfValue{
						ProtoFile: "test.proto",
					},
				},
			},
			want:    &protoconf_pb.ConfigMutationResponse{},
			wantErr: false,
		},
		{
			name: "run bad pre scripts",
			fields: fields{
				config: &cliConfig{
					preMutationScript: "false",
				},
				protoconfRoot: testdata.SmallTestDir(),
			},
			args: args{
				ctx: context.Background(),
				in: &protoconf_pb.ConfigMutationRequest{
					Path: "test",
					Value: &protoconf_pb.ProtoconfValue{
						ProtoFile: "test.proto",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "run bad post scripts",
			fields: fields{
				config: &cliConfig{
					postMutationScript: "false",
				},
				protoconfRoot: testdata.SmallTestDir(),
			},
			args: args{
				ctx: context.Background(),
				in: &protoconf_pb.ConfigMutationRequest{
					Path: "test",
					Value: &protoconf_pb.ProtoconfValue{
						ProtoFile: "test.proto",
					},
				},
			},
			wantErr: true,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewProtoconfMutationServer(tt.fields.protoconfRoot)
			s.config = tt.fields.config
			// TODO(smintz): assert the response
			_, err := s.MutateConfig(tt.args.ctx, tt.args.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("server.MutateConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
