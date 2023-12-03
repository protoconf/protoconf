package parser

import (
	"path/filepath"
	"testing"

	_ "github.com/protoconf/proto-validate-reflect/validate"
	"github.com/protoconf/protoconf/consts"
	"github.com/protoconf/protoconf/utils"
	"github.com/protoconf/protoconf/utils/testdata"
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
			dr.Import(utils.Parse, tt.fields.protoconfRoot)
			p := NewParser(dr.GetFilesResolver())
			_, err := p.ParseFilesX(tt.args.filenames...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parser.ParseFilesX() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
