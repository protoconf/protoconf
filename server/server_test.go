package server

import (
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"github.com/protoconf/protoconf/compiler/lib"
	protoconf_pb "github.com/protoconf/protoconf/pb/protoconf/v1"
	"github.com/protoconf/protoconf/utils/testdata"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
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
		wantErr error
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
			wantErr: nil,
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
			wantErr: ErrInternalCompilerError,
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
			wantErr: ErrInternalCompilerError,
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
			wantErr: ErrPreMutationScriptError,
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
			wantErr: os.ErrNotExist,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := lib.NewCompiler(tt.fields.protoconfRoot, false)
			s := NewProtoconfMutationServer(tt.fields.protoconfRoot, WithCompiler(compiler))
			s.config = tt.fields.config
			s.PreMutationScript = tt.fields.config.preMutationScript
			s.PostMutationScript = tt.fields.config.postMutationScript
			// TODO(smintz): assert the response
			_, err := s.MutateConfig(tt.args.ctx, tt.args.in)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("server.MutateConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
func TestProtoconfMutationServer_GenReflectionUI(t *testing.T) {
	protoconfRoot := testdata.SmallTestDir()
	server := NewProtoconfMutationServer(protoconfRoot)

	ctx := context.Background()
	rpcServer := grpc.NewServer()
	server.Init(rpcServer)
	httpServer := &http.Server{}

	err := server.GenReflectionUI(ctx, rpcServer, httpServer)
	if err != nil {
		t.Errorf("GenReflectionUI returned an error: %v", err)
	}

	// Add assertions here to verify the expected behavior of GenReflectionUI
}
func TestProtoconfMutationServer_ReportProgress(t *testing.T) {
	protoconfRoot := testdata.SmallTestDir()
	compiler := lib.NewCompiler(protoconfRoot, false)

	s := NewProtoconfMutationServer(protoconfRoot, WithCompiler(compiler))
	ctx := context.Background()
	in := &protoconf_pb.ConfigMutationResponse{
		Uuid: "test-uuid",
		// Set other fields as needed
	}
	s.reports.Store("test-uuid", in)

	// Call the function
	got, err := s.ReportProgress(ctx, in)
	if err != nil {
		t.Errorf("ReportProgress() error = %v", err)
		return
	}
	assert.NotNil(t, got)

	// Assert the response
	// Add your assertions here to verify the expected behavior of ReportProgress
}
func Test_cliCommand_Run(t *testing.T) {
	// Create a temporary directory for the protoconfRoot
	protoconfRoot := testdata.SmallTestDir()
	defer os.RemoveAll(protoconfRoot)

	// Create a temporary file for the preMutationScript
	preMutationScript, err := ioutil.TempFile("", "preMutationScript")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(preMutationScript.Name())

	// Create a temporary file for the postMutationScript
	postMutationScript, err := ioutil.TempFile("", "postMutationScript")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(postMutationScript.Name())

	// Set up the command and flags
	command := &cliCommand{}
	flags, _ := newFlagSet()
	flags.SetOutput(ioutil.Discard)
	flags.Parse([]string{
		"-grpc-address", "localhost:50051",
		"-pre", preMutationScript.Name(),
		"-post", postMutationScript.Name(),
		protoconfRoot,
	})

	// Run the command
	exitCode := command.Run(flags.Args())

	// Check the exit code
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}

	// Add assertions here to verify the expected behavior of the Run function
}
func Test_cliCommand_Synopsis(t *testing.T) {
	command := &cliCommand{}
	got := command.Synopsis()
	want := "Runs a server"
	if got != want {
		t.Errorf("cliCommand.Synopsis() = %q, want %q", got, want)
	}
}
