package utils

import (
	"testing"

	protoconfvalue "github.com/protoconf/protoconf/datatypes/proto/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
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

func TestReadConfig(t *testing.T) {
	type args struct {
		protoconfRoot string
		configName    string
	}
	tests := []struct {
		name    string
		args    args
		want    *protoconfvalue.ProtoconfValue
		wantErr bool
	}{
		{
			name: "read config",
			args: args{
				protoconfRoot: "testdata/large",
				configName:    "example/instance_0",
			},
			want: &protoconfvalue.ProtoconfValue{
				ProtoFile: "terraform/v1/terraform.proto",
				Value: &anypb.Any{
					TypeUrl: "type.googleapis.com/terraform.v1.Terraform",
					Value:   []byte("\nM\xa2%J\n\ninstance_0\x12<\n\x0cami-830c94e3\x9a\x01\x08t2.micro\xb2\x02 \n\x04Name\x12\x18ExampleAppServerInstance"),
				},
			},
			wantErr: false,
		},
		{
			name: "config not found",
			args: args{
				protoconfRoot: "testdata/large",
				configName:    "example/not_found",
			},
			wantErr: true,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadConfig(tt.args.protoconfRoot, tt.args.configName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !proto.Equal(got, tt.want) {
				t.Errorf("ReadConfig() = %v, want %v", got, tt.want)
			}
		})
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
			args:    args{root: "testdata/small", link: false},
			wantErr: false,
		},
		{
			name:    "test linked",
			args:    args{root: "testdata/small", link: true},
			wantErr: false,
		},
		{
			name:    "test broken proto",
			args:    args{root: "testdata/bad_proto", link: false},
			wantErr: true,
		},
		{
			name:    "test linked broken proto",
			args:    args{root: "testdata/bad_proto", link: true},
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
