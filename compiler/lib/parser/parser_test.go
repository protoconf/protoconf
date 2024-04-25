package parser

import (
	"path/filepath"
	"regexp"
	"testing"

	_ "github.com/bufbuild/protovalidate-go"
	_ "github.com/bufbuild/protovalidate-go/legacy"

	"github.com/protoconf/protoconf/consts"
	v1 "github.com/protoconf/protoconf/datatypes/proto/v1"
	"github.com/protoconf/protoconf/utils"
	"github.com/protoconf/protoconf/utils/testdata"
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
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dr := utils.NewDescriptorRegistry()
			dr.Import(utils.Parse, []*regexp.Regexp{}, tt.fields.protoconfRoot)
			p := NewParser(dr.GetFilesResolver())
			_, err := p.ParseFilesX(tt.args.filenames...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parser.ParseFilesX() error = %v, wantErr %v", err, tt.wantErr)
				return
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
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:   "test",
			fields: fields{filepath.Join(testdata.SmallTestDir())},
			args: args{
				filename: "materialized_config/test.materialized_JSON",
				msg:      &v1.ProtoconfValue{},
			},
			wantErr: false,
		},
		{
			name:   "test not found",
			fields: fields{filepath.Join(testdata.SmallTestDir())},
			args: args{
				filename: "materialized_config/test.materialized_JSON1",
				msg:      &v1.ProtoconfValue{},
			},
			wantErr: true,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dr := utils.NewDescriptorRegistry()
			dr.Import(utils.Parse, []*regexp.Regexp{}, tt.fields.protoconfRoot)
			p := NewParser(dr.GetFilesResolver())
			if err := p.ReadConfig(filepath.Join(tt.fields.protoconfRoot, tt.args.filename), tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("Parser.ReadConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
