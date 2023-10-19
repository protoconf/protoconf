package server

import (
	"context"
	"os"
	"reflect"
	"testing"

	v1 "github.com/protoconf/protoconf/datatypes/proto/v1"
	protoconfmutation "github.com/protoconf/protoconf/server/api/proto/v1"
	"github.com/protoconf/protoconf/utils/testdata"
)

func Test_server_MutateConfig(t *testing.T) {
	type fields struct {
		config        *cliConfig
		protoconfRoot string
	}
	type args struct {
		ctx context.Context
		in  *protoconfmutation.ConfigMutationRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *protoconfmutation.ConfigMutationResponse
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
				in: &protoconfmutation.ConfigMutationRequest{
					Path:  "test",
					Value: &v1.ProtoconfValue{},
				},
			},
			want:    &protoconfmutation.ConfigMutationResponse{},
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
				in: &protoconfmutation.ConfigMutationRequest{
					Path: "test",
					Value: &v1.ProtoconfValue{
						ProtoFile: "test.proto",
					},
				},
			},
			want:    &protoconfmutation.ConfigMutationResponse{},
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
				in: &protoconfmutation.ConfigMutationRequest{
					Path: "test",
					Value: &v1.ProtoconfValue{
						ProtoFile: "test.proto",
					},
				},
			},
			want:    &protoconfmutation.ConfigMutationResponse{},
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
				in: &protoconfmutation.ConfigMutationRequest{
					Path: "test",
					Value: &v1.ProtoconfValue{
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
				in: &protoconfmutation.ConfigMutationRequest{
					Path: "test",
					Value: &v1.ProtoconfValue{
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
			s := server{
				config:        tt.fields.config,
				protoconfRoot: tt.fields.protoconfRoot,
			}
			got, err := s.MutateConfig(tt.args.ctx, tt.args.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("server.MutateConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("server.MutateConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}
