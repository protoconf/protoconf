package utils

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	_ "github.com/bufbuild/protovalidate-go"
	_ "github.com/bufbuild/protovalidate-go/legacy"

	"github.com/protoconf/protoconf/consts"
	"github.com/protoconf/protoconf/utils/testdata"
	"github.com/stretchr/testify/require"
)

func BenchmarkFindEmpty(b *testing.B) {
	for i := 0; i < b.N; i++ {
		find("testdata/empty", ".proto")
	}
}
func BenchmarkFindSmall(b *testing.B) {
	for i := 0; i < b.N; i++ {
		find("testdata/small", ".proto")
	}
}

func BenchmarkFindLarge(b *testing.B) {
	for i := 0; i < b.N; i++ {
		find("testdata/large", ".proto")
	}
}

func BenchmarkLoadLocalProtoFilesEmpty(b *testing.B) {
	for i := 0; i < b.N; i++ {
		LoadLocalProtoFiles(false, "testdata/empty")
	}
}

func BenchmarkLoadLocalProtoFilesSmall(b *testing.B) {
	for i := 0; i < b.N; i++ {
		LoadLocalProtoFiles(false, "testdata/small")
	}
}

func BenchmarkLoadLocalProtoFilesLarge(b *testing.B) {
	for i := 0; i < b.N; i++ {
		LoadLocalProtoFiles(false, "testdata/large")
	}
}

func TestLoadLocalProtoFiles(t *testing.T) {
	type args struct {
		root string
		link bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "test",
			args:    args{root: "testdata/small/src", link: false},
			wantErr: false,
		},
		{
			name:    "test linked",
			args:    args{root: "testdata/small/src", link: true},
			wantErr: false,
		},
		{
			name:    "test broken proto",
			args:    args{root: "testdata/bad_proto/src", link: false},
			wantErr: true,
		},
		{
			name:    "test linked broken proto",
			args:    args{root: "testdata/bad_proto/src", link: true},
			wantErr: true,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadLocalProtoFiles(tt.args.link, tt.args.root)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadLocalProtoFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestNewDescriptorRegistry(t *testing.T) {
	registry := NewDescriptorRegistry()
	largePath := filepath.Join(testdata.SmallTestDir(), consts.SrcPath)
	err := registry.Import(Parse, []*regexp.Regexp{}, largePath)
	require.NoError(t, err)

	files := registry.GetFilesResolver()
	_, err = files.FindFileByPath("test.proto")
	require.NoError(t, err)

	types := registry.GetTypesResolver(files)
	_, err = types.FindMessageByName("TestMessage")
	require.NoError(t, err)

	storeFile := filepath.Join(os.TempDir(), "data.fds")
	h, err := registry.Store(storeFile)
	require.NoError(t, err)

	newReg := NewDescriptorRegistry()
	err = newReg.Load(storeFile, "baddd")
	require.Error(t, err)
	err = newReg.Load(storeFile, h)
	require.NoError(t, err)

}
