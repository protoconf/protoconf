package mutate

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	dpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/ptypes"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/mitchellh/cli"
	"github.com/pkg/errors"
	pv "github.com/protoconf/protoconf/datatypes/proto/v1/protoconfvalue"
	pc "github.com/protoconf/protoconf/server/api/proto/v1/protoconfmutation"
	"github.com/protoconf/protoconf/utils"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var conn *grpc.ClientConn

type cliCommand struct {
	ui cli.Ui
}

var ui = &cli.BasicUi{
	Reader:      os.Stdin,
	Writer:      os.Stdout,
	ErrorWriter: os.Stderr,
}

type fieldsArray []string

func (i *fieldsArray) String() string {
	arr := []string{}
	for _, s := range *i {
		arr = append(arr, s)
	}
	return fmt.Sprintf("%v", arr)
}

func (i *fieldsArray) Set(value string) error {
	*i = append(*i, value)
	return nil
}

type cliConfig struct {
	protoconfRoot string
	protoFile     string
	protoMsg      string
	serverAddress string
	configPath    string
	metadataStr   string
	fieldsArray   fieldsArray
}

func newFlagSet() (*flag.FlagSet, *cliConfig) {
	flags := flag.NewFlagSet("mutate", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintln(flags.Output(), "Usage: [OPTION]...")
		helpText := `
A CLI util to communicate with the mutation server easily. 
	`
		fmt.Fprintln(flags.Output(), helpText)
		flags.PrintDefaults()
	}

	config := &cliConfig{}
	flags.StringVar(&config.protoconfRoot, "root", "./src", "The root of protoconf src.")
	flags.StringVar(&config.protoFile, "proto", "", "Path to the proto file")
	flags.StringVar(&config.protoMsg, "msg", "", "Name of the message inside the -proto file")
	flags.StringVar(&config.serverAddress, "addr", "localhost:4301", "Server address")
	flags.StringVar(&config.configPath, "path", "", "Path to put the config in")
	flags.StringVar(&config.metadataStr, "metadata", "", "Metadata string to pass to the pre/post install script")
	flags.Var(&config.fieldsArray, "field", "fields to set inside -msg")

	return flags, config
}

func (c *cliCommand) Run(args []string) int {
	flags, config := newFlagSet()
	flags.Parse(args)

	if config.protoFile == "" || config.configPath == "" || config.protoMsg == "" || len(config.fieldsArray) < 1 {
		c.ui.Output(c.Help())
		return 0
	}
	path := config.configPath

	root, err := filepath.Abs(config.protoconfRoot)
	if err != nil {
		log.Fatal("failed to get root path:", err)
	}
	anyResolver, err := utils.LoadAnyResolver(root, config.protoFile)
	if err != nil {
		log.Fatal("failed to get AnyResolver:", err)
	}

	name, err := anyResolver.Resolve(config.protoMsg)
	if err != nil {
		log.Fatal(errors.Wrapf(err, "could not find typeUrl for %s", config.protoMsg))
	}

	msg, err := dynamic.AsDynamicMessage(name)
	if err != nil {
		log.Fatal(err)
	}

	for _, fName := range config.fieldsArray {
		ret := strings.SplitN(fName, "=", 2)
		field := msg.GetMessageDescriptor().FindFieldByName(ret[0])
		if field == nil {
			log.Fatalf("%s is not a field in %s", ret[0], msg.XXX_MessageName())
		}
		switch field.GetType() {
		case dpb.FieldDescriptorProto_TYPE_DOUBLE:
			setFloat(msg, ret[0], ret[1], func(s interface{}) interface{} { return s })
		case dpb.FieldDescriptorProto_TYPE_FLOAT:
			setFloat(msg, ret[0], ret[1], func(s interface{}) interface{} { return s })
		case dpb.FieldDescriptorProto_TYPE_INT64:
			setNumeric(msg, ret[0], ret[1], func(s interface{}) interface{} { return s })
		case dpb.FieldDescriptorProto_TYPE_UINT64:
			setNumeric(msg, ret[0], ret[1], func(s interface{}) interface{} { return s })
		case dpb.FieldDescriptorProto_TYPE_INT32:
			setNumeric(msg, ret[0], ret[1], func(s interface{}) interface{} { return int32(s.(int64)) })
		case dpb.FieldDescriptorProto_TYPE_FIXED64:
			setNumeric(msg, ret[0], ret[1], func(s interface{}) interface{} { return s })
		case dpb.FieldDescriptorProto_TYPE_FIXED32:
			setNumeric(msg, ret[0], ret[1], func(s interface{}) interface{} { return int32(s.(int64)) })
		case dpb.FieldDescriptorProto_TYPE_BOOL:
			b, e := strconv.ParseBool(ret[1])
			if e != nil {
				log.Fatal(e)
			}
			setField(msg, ret[0], b, func(s interface{}) interface{} {
				return s
			})
		case dpb.FieldDescriptorProto_TYPE_STRING:
			setField(msg, ret[0], ret[1], func(s interface{}) interface{} { return s })
		case dpb.FieldDescriptorProto_TYPE_UINT32:
			setNumeric(msg, ret[0], ret[1], func(s interface{}) interface{} { return uint32(s.(int64)) })
		case dpb.FieldDescriptorProto_TYPE_SFIXED32:
			setNumeric(msg, ret[0], ret[1], func(s interface{}) interface{} { return int32(s.(int64)) })
		case dpb.FieldDescriptorProto_TYPE_SFIXED64:
			setNumeric(msg, ret[0], ret[1], func(s interface{}) interface{} { return s })
		case dpb.FieldDescriptorProto_TYPE_SINT32:
			setNumeric(msg, ret[0], ret[1], func(s interface{}) interface{} { return uint32(s.(int64)) })
		case dpb.FieldDescriptorProto_TYPE_SINT64:
			setNumeric(msg, ret[0], ret[1], func(s interface{}) interface{} { return s })
		}
	}

	log.Println(msg.String())
	address := config.serverAddress
	conn, err = grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatal(fmt.Errorf("error connecting to server address=%s err=%s", address, err))
	}
	defer conn.Close()
	any, err := ptypes.MarshalAny(msg)
	if err != nil {
		log.Fatal(fmt.Errorf("error marshalling message to any message=%s err=%s", msg, err))
	}
	log.Println(msg)
	log.Println(any)
	configValue := &pv.ProtoconfValue{ProtoFile: config.protoFile, Value: any}
	request := &pc.ConfigMutationRequest{Path: config.configPath, Value: configValue, ScriptMetadata: config.metadataStr}

	client := pc.NewProtoconfMutationServiceClient(conn)
	// Wait until the server finishes long git operations
	timeout := 60 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if _, err := client.MutateConfig(ctx, request); err != nil {
		log.Fatal(fmt.Errorf("error mutating path=%s err=%s", path, err))
	}
	log.Printf("Mutated %s successfully", path)
	return 0
}

type typerFunc func(interface{}) interface{}

func setNumeric(msg *dynamic.Message, key, val string, typer typerFunc) {
	i, err := strconv.ParseInt(val, 0, 64)
	if err != nil {
		log.Fatal(err)
	}
	setField(msg, key, i, typer)
}

func setFloat(msg *dynamic.Message, key, val string, typer typerFunc) {
	i, err := strconv.ParseFloat(val, 64)
	if err != nil {
		log.Fatal(err)
	}
	setField(msg, key, i, typer)
}

func setField(msg *dynamic.Message, key string, val interface{}, typer typerFunc) {
	err := msg.TrySetFieldByName(key, typer(val))
	if err != nil {
		log.Fatal(err)
	}
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
	return "Write to mutation server"
}

// Command is a cli.CommandFactory
func Command() (cli.Command, error) {
	return &cliCommand{ui: ui}, nil
}
