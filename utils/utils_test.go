package utils

import (
	"context"
	"reflect"
	"testing"

	"github.com/kylelemons/godebug/diff"
	protoconfvalue "github.com/protoconf/protoconf/datatypes/proto/v1"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestLoadRemoteDescriptorsFromBuf(t *testing.T) {
	name := func(s string) *string { return &s }
	num := func(i int32) *int32 { return &i }
	b := func(i bool) *bool { return &i }
	type args struct {
		ctx  context.Context
		repo string
	}
	tests := []struct {
		name    string
		args    args
		want    *descriptorpb.FileDescriptorSet
		wantErr bool
	}{
		{
			name: "eliza",
			args: args{context.Background(), "buf.build/bufbuild/eliza"},
			want: &descriptorpb.FileDescriptorSet{
				File: []*descriptorpb.FileDescriptorProto{{
					Name:    name("buf/connect/demo/eliza/v1/eliza.proto"),
					Package: name("buf.connect.demo.eliza.v1"),
					MessageType: []*descriptorpb.DescriptorProto{
						{Name: name("SayRequest"), Field: []*descriptorpb.FieldDescriptorProto{{Name: name("sentence"), Number: num(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), JsonName: name("sentence")}}},
						{Name: name("SayResponse"), Field: []*descriptorpb.FieldDescriptorProto{{Name: name("sentence"), Number: num(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), JsonName: name("sentence")}}},
						{Name: name("ConverseRequest"), Field: []*descriptorpb.FieldDescriptorProto{{Name: name("sentence"), Number: num(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), JsonName: name("sentence")}}},
						{Name: name("ConverseResponse"), Field: []*descriptorpb.FieldDescriptorProto{{Name: name("sentence"), Number: num(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), JsonName: name("sentence")}}},
						{Name: name("IntroduceRequest"), Field: []*descriptorpb.FieldDescriptorProto{{Name: name("name"), Number: num(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), JsonName: name("name")}}},
						{Name: name("IntroduceResponse"), Field: []*descriptorpb.FieldDescriptorProto{{Name: name("sentence"), Number: num(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), JsonName: name("sentence")}}},
					},
					Service: []*descriptorpb.ServiceDescriptorProto{{
						Name: name("ElizaService"),
						Method: []*descriptorpb.MethodDescriptorProto{
							{Name: name("Say"), InputType: name(".buf.connect.demo.eliza.v1.SayRequest"), OutputType: name(".buf.connect.demo.eliza.v1.SayResponse"), Options: &descriptorpb.MethodOptions{}},
							{Name: name("Converse"), InputType: name(".buf.connect.demo.eliza.v1.ConverseRequest"), OutputType: name(".buf.connect.demo.eliza.v1.ConverseResponse"), Options: &descriptorpb.MethodOptions{}, ClientStreaming: b(true), ServerStreaming: b(true)},
							{Name: name("Introduce"), InputType: name(".buf.connect.demo.eliza.v1.IntroduceRequest"), OutputType: name(".buf.connect.demo.eliza.v1.IntroduceResponse"), Options: &descriptorpb.MethodOptions{}, ServerStreaming: b(true)},
						},
					}},
					Syntax: name("proto3"),
				}},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadRemoteDescriptorsFromBuf(tt.args.ctx, tt.args.repo)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadRemoteDescriptorsFromBuf() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !proto.Equal(got, tt.want) {
				t.Errorf("LoadRemoteDescriptorsFromBuf() got:\n%v\nwant:\n%v\ndiff:\n%v", prototext.Format(got), prototext.Format(tt.want), diff.Diff(prototext.Format(got), prototext.Format(tt.want)))
			}
		})
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
			name: "envoy",
			args: args{
				protoconfRoot: "/Users/smintz/go/src/github.com/protoconf/protoconf-envoy",
				configName:    "example/envoy/test-id",
			},
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadConfig(tt.args.protoconfRoot, tt.args.configName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReadConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}
