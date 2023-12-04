package server

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mitchellh/cli"
	"github.com/protoconf/protoconf/consts"
	protoconfmutation "github.com/protoconf/protoconf/server/api/proto/v1"
	"github.com/protoconf/protoconf/utils"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
)

type cliCommand struct{}

type cliConfig struct {
	grpcAddress        string
	preMutationScript  string
	postMutationScript string
}

func newFlagSet() (*flag.FlagSet, *cliConfig) {
	flags := flag.NewFlagSet("", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintln(flags.Output(), "Usage: [OPTION]... protoconfRoot")
		flags.PrintDefaults()
	}

	config := &cliConfig{}
	flags.StringVar(&config.grpcAddress, "grpc-address", consts.ServerDefaultAddress, "Server gRPC address")
	flags.StringVar(&config.preMutationScript, "pre", "", "Pre mutation script")
	flags.StringVar(&config.postMutationScript, "post", "", "Post mutation script")

	return flags, config
}

func (c *cliCommand) Run(args []string) int {
	flags, config := newFlagSet()
	flags.Parse(args)

	if flags.NArg() < 1 {
		flags.Usage()
		return 1
	}

	protoconfRoot := strings.TrimSpace(flags.Args()[0])
	protoconfServer := &ProtoconfMutationServer{config: config, protoconfRoot: protoconfRoot}

	log.Printf("Starting Protoconf server at \"%s\", version %s", config.grpcAddress, consts.Version)
	log.Printf("Config: protoconf_root=\"%s\" pre-mutation-script=\"%s\" post-mutation-script=\"%s\"", protoconfRoot, config.preMutationScript, config.postMutationScript)

	listener, err := net.Listen("tcp", config.grpcAddress)
	if err != nil {
		log.Printf("Error listening on address=%s err=%s", config.grpcAddress, err)
		return 1
	}

	rpcServer := grpc.NewServer()
	protoconfmutation.RegisterProtoconfMutationServiceServer(rpcServer, protoconfServer)

	log.Println("Protoconf server running")
	err = rpcServer.Serve(listener)
	if err != nil {
		log.Printf("Error serving gRPC, err=%s", err)
		return 1
	}

	return 0
}

func (c *cliCommand) Help() string {
	var b bytes.Buffer
	b.WriteString(c.Synopsis())
	b.WriteString("\n")
	flags, _ := newFlagSet()
	flags.SetOutput(&b)
	flags.Usage()
	return b.String()
}

func (c *cliCommand) Synopsis() string {
	return "Runs a server"
}

// Command is a cli.CommandFactory
func Command() (cli.Command, error) {
	return &cliCommand{}, nil
}

type ProtoconfMutationServer struct {
	config        *cliConfig
	protoconfRoot string
}

func NewProtoconfMutationServer(protoconfRoot string) *ProtoconfMutationServer {
	return &ProtoconfMutationServer{protoconfRoot: protoconfRoot, config: &cliConfig{}}
}

func (s ProtoconfMutationServer) MutateConfig(ctx context.Context, in *protoconfmutation.ConfigMutationRequest) (*protoconfmutation.ConfigMutationResponse, error) {
	log.Printf("Mutating path=%s", in.Path)
	filename := filepath.Join(s.protoconfRoot, consts.MutableConfigPath, filepath.Clean(in.Path)+consts.CompiledConfigExtension)

	resolver := utils.LocalResolver(filepath.Join(s.protoconfRoot, consts.SrcPath))
	jsonData, err := protojson.MarshalOptions{Resolver: resolver, Multiline: true}.Marshal(in.Value)
	if err != nil {
		return nil, logError(fmt.Errorf("error marshaling ProtoconfValue to JSON, value=%s", in.Value))
	}

	if s.config.preMutationScript != "" {
		if err := runScript(s.config.preMutationScript, in.ScriptMetadata); err != nil {
			return nil, logError(fmt.Errorf("error running pre mutation script, err=%s", err))
		}
	}

	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return nil, logError(fmt.Errorf("error creating output directory %s, err: %s", filepath.Dir(filename), err))
	}

	if err := os.WriteFile(filename, []byte(jsonData), 0644); err != nil {
		return nil, logError(fmt.Errorf("error writing to file %s, err: %s", filename, err))
	}

	log.Printf("Written to %s", filename)

	if s.config.postMutationScript != "" {
		if err := runScript(s.config.postMutationScript, in.ScriptMetadata); err != nil {
			return nil, logError(fmt.Errorf("error running post mutation script, err=%s", err))
		}
	}

	return &protoconfmutation.ConfigMutationResponse{}, nil
}

func logError(err error) error {
	log.Printf("Error: %s", err)
	return err
}

func runScript(filename string, argument string) error {
	cmd := exec.Command(filename, argument)
	_, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			log.Printf("Script error: %s", string(ee.Stderr))
		}
		return err
	}
	return nil
}
