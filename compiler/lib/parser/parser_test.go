package parser

import (
	"path/filepath"
	"regexp"
	"testing"

	_ "github.com/bufbuild/protovalidate-go"
	_ "github.com/bufbuild/protovalidate-go/legacy"
	protoconf_pb "github.com/protoconf/protoconf/pb/protoconf/v1"

	"github.com/protoconf/protoconf/consts"
	"github.com/protoconf/protoconf/utils"
	"github.com/protoconf/protoconf/utils/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestParser_ParseFilesX(t *testing.T) {
	type fields struct {
		protoconfRoot string
	}
	type args struct {
		filenames []string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:   "test",
			fields: fields{filepath.Join(testdata.SmallTestDir(), consts.SrcPath)},
			args:   args{filenames: []string{"test.proto"}},
		},
		{
			name:    "test not found",
			fields:  fields{filepath.Join(testdata.SmallTestDir(), consts.SrcPath)},
			args:    args{filenames: []string{"test1.proto"}},
			wantErr: true,
		},
		{
			name:   "test with package verification",
			fields: fields{filepath.Join(testdata.SmallTestDir(), consts.SrcPath)},
			args:   args{filenames: []string{"test.proto"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dr := utils.NewDescriptorRegistry()
			dr.Import(dr.Parse, []*regexp.Regexp{}, tt.fields.protoconfRoot)
			p := NewParserWithDescriptorRegistry(dr)
			got, err := p.ParseFilesX(tt.args.filenames...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parser.ParseFilesX() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				require.NotEmpty(t, got)
				assert.Equal(t, tt.args.filenames[0], got[0].GetName())
			}
		})
	}
}

func TestParser_ReadConfig(t *testing.T) {
	type fields struct {
		protoconfRoot string
	}
	type args struct {
		filename string
		msg      proto.Message
	}
	tests := []struct {
		name          string
		fields        fields
		args          args
		wantErr       bool
		wantProtoFile string
	}{
		{
			name:   "test",
			fields: fields{filepath.Join(testdata.SmallTestDir())},
			args: args{
				filename: "materialized_config/test.materialized_JSON",
				msg:      &protoconf_pb.ProtoconfValue{},
			},
			wantErr:       false,
			wantProtoFile: "test.proto",
		},
		{
			name:   "test not found",
			fields: fields{filepath.Join(testdata.SmallTestDir())},
			args: args{
				filename: "materialized_config/test.materialized_JSON1",
				msg:      &protoconf_pb.ProtoconfValue{},
			},
			wantErr:       true,
			wantProtoFile: "",
		},
		{
			name:   "with rollout config",
			fields: fields{filepath.Join(testdata.SmallTestDir())},
			args: args{
				filename: "materialized_config/with_config_rollout.materialized_JSON",
				msg:      &protoconf_pb.ProtoconfValue{},
			},
			wantErr:       false,
			wantProtoFile: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dr := utils.NewDescriptorRegistry()
			dr.Import(dr.Parse, []*regexp.Regexp{}, tt.fields.protoconfRoot)
			p := NewParserWithDescriptorRegistry(dr)
			if err := p.ReadConfig(filepath.Join(tt.fields.protoconfRoot, tt.args.filename), tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("Parser.ReadConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				pcv, ok := tt.args.msg.(*protoconf_pb.ProtoconfValue)
				if ok && tt.wantProtoFile != "" {
					assert.Equal(t, tt.wantProtoFile, pcv.ProtoFile)
					assert.NotNil(t, pcv.Value)
				}
			}
		})
	}
}
