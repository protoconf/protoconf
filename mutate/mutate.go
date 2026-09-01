package mutate

import (
	"bytes"
	"crypto/tls"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/mitchellh/cli"
	configtool "github.com/protoconf/libprotoconf"
	"github.com/protoconf/protoconf/command"
	"github.com/protoconf/protoconf/compiler/lib"
	"github.com/protoconf/protoconf/compiler/lib/parser"
	"github.com/protoconf/protoconf/compiler/starproto"
	pv "github.com/protoconf/protoconf/datatypes/proto/v1"
	protoconf_mutate_config "github.com/protoconf/protoconf/mutate/config/v1"
	pc "github.com/protoconf/protoconf/server/api/proto/v1"
	"github.com/protoconf/protoconf/utils"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/anypb"
)

type cliCommand struct {
	ui     cli.Ui
	config *protoconf_mutate_config.MutateConfig
	flag   *flag.FlagSet
}

var ui = &cli.BasicUi{
	Reader:      os.Stdin,
	Writer:      os.Stdout,
	ErrorWriter: os.Stderr,
}

func (c *cliCommand) Run(args []string) int {
	err := c.flag.Parse(args)
	if err != nil {
		fmt.Fprint(os.Stderr, "failed to parse flags", err)
		return 2
	}

	if c.config.ProtoFile == "" || c.config.ConfigPath == "" || c.config.ProtoMsg == "" || len(c.config.Fields) < 1 {
		c.ui.Output(c.Help())
		return 0
	}
	path := c.config.ConfigPath

	root, err := filepath.Abs(c.config.ProtoconfRoot)
	if err != nil {
		slog.Error("failed to get root path", "error", err)
		return 1
	}
	ms, err := lib.NewModuleService(root)
	if err != nil {
		slog.Error("error creating module service", "error", err)
		return 1
	}
	ms.LoadFromLockFile()
	parser := parser.NewParserWithDescriptorRegistry(ms.GetProtoRegistry())
	anyResolver := parser.LocalResolver

	messageType, err := anyResolver.FindMessageByName(protoreflect.FullName(c.config.ProtoMsg))

	if err != nil {
		slog.Error("could not find typeUrl for", "msg", c.config.ProtoMsg, "error", err)
		return 1
	}
	wrap, err := desc.WrapMessage(messageType.Descriptor())
	if err != nil {
		slog.Error("error wrapping message", "error", err)
		return 1
	}

	msg := dynamic.NewMessage(wrap)

	for _, fName := range c.config.Fields {
		ret := strings.SplitN(fName, "=", 2)
		field := msg.GetMessageDescriptor().FindFieldByName(ret[0])
		if field == nil {
			slog.Error("is not a field in ", "field", ret[0], "message", msg.XXX_MessageName())
			return 1
		}
		switch field.GetType() {
		case descriptorpb.FieldDescriptorProto_TYPE_DOUBLE:
			if err := setFloat(msg, ret[0], ret[1], func(s interface{}) interface{} { return s }); err != nil {
				slog.Error("error setting field", "field", ret[0], "error", err)
				return 1
			}
		case descriptorpb.FieldDescriptorProto_TYPE_FLOAT:
			if err := setFloat(msg, ret[0], ret[1], func(s interface{}) interface{} { return s }); err != nil {
				slog.Error("error setting field", "field", ret[0], "error", err)
				return 1
			}
		case descriptorpb.FieldDescriptorProto_TYPE_INT64:
			if err := setNumeric(msg, ret[0], ret[1], func(s interface{}) interface{} { return s }); err != nil {
				slog.Error("error setting field", "field", ret[0], "error", err)
				return 1
			}
		case descriptorpb.FieldDescriptorProto_TYPE_UINT64:
			if err := setNumeric(msg, ret[0], ret[1], func(s interface{}) interface{} { return s }); err != nil {
				slog.Error("error setting field", "field", ret[0], "error", err)
				return 1
			}
		case descriptorpb.FieldDescriptorProto_TYPE_INT32:
			if err := setNumeric(msg, ret[0], ret[1], func(s interface{}) interface{} { return int32(s.(int64)) }); err != nil {
				slog.Error("error setting field", "field", ret[0], "error", err)
				return 1
			}
		case descriptorpb.FieldDescriptorProto_TYPE_FIXED64:
			if err := setNumeric(msg, ret[0], ret[1], func(s interface{}) interface{} { return s }); err != nil {
				slog.Error("error setting field", "field", ret[0], "error", err)
				return 1
			}
		case descriptorpb.FieldDescriptorProto_TYPE_FIXED32:
			if err := setNumeric(msg, ret[0], ret[1], func(s interface{}) interface{} { return int32(s.(int64)) }); err != nil {
				slog.Error("error setting field", "field", ret[0], "error", err)
				return 1
			}
		case descriptorpb.FieldDescriptorProto_TYPE_BOOL:
			b, e := strconv.ParseBool(ret[1])
			if e != nil {
				slog.Error("error parsing bool", "error", e)
				return 1
			}
			setField(msg, ret[0], b, func(s interface{}) interface{} {
				return s
			})
		case descriptorpb.FieldDescriptorProto_TYPE_STRING:
			setField(msg, ret[0], ret[1], func(s interface{}) interface{} { return s })
		case descriptorpb.FieldDescriptorProto_TYPE_UINT32:
			if err := setNumeric(msg, ret[0], ret[1], func(s interface{}) interface{} { return uint32(s.(int64)) }); err != nil {
				slog.Error("error setting field", "field", ret[0], "error", err)
				return 1
			}
		case descriptorpb.FieldDescriptorProto_TYPE_SFIXED32:
			if err := setNumeric(msg, ret[0], ret[1], func(s interface{}) interface{} { return int32(s.(int64)) }); err != nil {
				slog.Error("error setting field", "field", ret[0], "error", err)
				return 1
			}
		case descriptorpb.FieldDescriptorProto_TYPE_SFIXED64:
			if err := setNumeric(msg, ret[0], ret[1], func(s interface{}) interface{} { return s }); err != nil {
				slog.Error("error setting field", "field", ret[0], "error", err)
				return 1
			}
		case descriptorpb.FieldDescriptorProto_TYPE_SINT32:
			if err := setNumeric(msg, ret[0], ret[1], func(s interface{}) interface{} { return uint32(s.(int64)) }); err != nil {
				slog.Error("error setting field", "field", ret[0], "error", err)
				return 1
			}
		case descriptorpb.FieldDescriptorProto_TYPE_SINT64:
			if err := setNumeric(msg, ret[0], ret[1], func(s interface{}) interface{} { return s }); err != nil {
				slog.Error("error setting field", "field", ret[0], "error", err)
				return 1
			}
		}
	}

	slog.Info(msg.String())
	address := c.config.ServerAddress
	var dialOpt grpc.DialOption
	if c.config.TlsCert != "" || c.config.TlsKey != "" || c.config.TlsCa != "" {
		tlsCfg, err := utils.BuildTLSConfig(utils.TLSFiles{
			CertFile: c.config.TlsCert,
			KeyFile:  c.config.TlsKey,
			CAFile:   c.config.TlsCa,
		})
		if err != nil {
			slog.Error("failed to build TLS config", "error", err)
			return 1
		}
		if tlsCfg == nil {
			// CA-only: use system roots + provided CA, no client cert
			tlsCfg = &tls.Config{}
		}
		dialOpt = grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg))
	} else {
		dialOpt = grpc.WithTransportCredentials(insecure.NewCredentials())
	}
	conn, err := grpc.NewClient(address, dialOpt)
	if err != nil {
		slog.Error("error connecting to server", "address", address, "error", err)
		return 1
	}
	defer conn.Close()
	any, err := anypb.New(starproto.ToDynamicPb(msg))
	if err != nil {
		slog.Error("error marshalling message to any", "message", msg, "error", err)
		return 1
	}
	slog.Info(msg.String())
	slog.Info("Info", "any", any)
	configValue := &pv.ProtoconfValue{ProtoFile: c.config.ProtoFile, Value: any}
	request := &pc.ConfigMutationRequest{Path: c.config.ConfigPath, Value: configValue, ScriptMetadata: c.config.MetadataStr}

	client := pc.NewProtoconfMutationServiceClient(conn)
	// Wait until the server finishes long git operations
	timeout := 60 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if _, err := client.MutateConfig(ctx, request); err != nil {
		slog.Error("error mutating", "path", path, "error", err)
		return 1
	}
	slog.Info("Mutated successfully", "path", path)
	return 0
}

type typerFunc func(interface{}) interface{}

func setNumeric(msg *dynamic.Message, key, val string, typer typerFunc) error {
	i, err := strconv.ParseInt(val, 0, 64)
	if err != nil {
		return fmt.Errorf("error parsing int field %q value %q: %w", key, val, err)
	}
	setField(msg, key, i, typer)
	return nil
}

func setFloat(msg *dynamic.Message, key, val string, typer typerFunc) error {
	i, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return fmt.Errorf("error parsing float field %q value %q: %w", key, val, err)
	}
	setField(msg, key, i, typer)
	return nil
}

func setField(msg *dynamic.Message, key string, val interface{}, typer typerFunc) {
	err := msg.TrySetFieldByName(key, typer(val))
	if err != nil {
		slog.Error("error setting field", "error", err)
	}
}

func (c *cliCommand) Help() string {
	var b bytes.Buffer
	b.WriteString(c.Synopsis())
	b.WriteString("\n")
	c.flag.SetOutput(&b)
	c.flag.Usage()
	return b.String()
}

func (c *cliCommand) Synopsis() string {
	return "Send a config mutation to the mutation server via gRPC"
}

// Command is a cli.CommandFactory
func Command() (cli.Command, error) {
	c := &cliCommand{
		ui: ui,
		config: &protoconf_mutate_config.MutateConfig{
			ProtoconfRoot: "./src",
			ServerAddress: "localhost:4301",
		},
	}
	lpc := configtool.NewConfig(c.config)
	lpc.SetEnvKeyPrefix("PROTOCONF_MUTATE")
	// base is the pristine factory-default snapshot handed to command.NewConfigLayerer below.
	// It is no longer mutated as an accumulator (that role now belongs to layerer.fileLayer).
	// Captures ProtoconfRoot: "./src" and ServerAddress: "localhost:4301".
	base := proto.Clone(c.config)
	lpc.Environment()
	c.flag = flag.NewFlagSet(string(c.config.ProtoReflect().Descriptor().FullName()), flag.ContinueOnError)
	lpc.PopulateFlagSet(c.flag)
	// layerer owns the accumulated config-file layer and the env/flag provenance set for
	// this Command() instance. Constructed after PopulateFlagSet (so c.flag is fully
	// populated) and before flag.Parse runs (so markExplicitFlags can observe flags parsed
	// before each -config-file occurrence via c.flag.Visit).
	layerer := command.NewConfigLayerer(base, c.flag)
	c.flag.Func("config-file", "Mutate configuration file (available formats: json, yaml, pb). Values are overridden by PROTOCONF_MUTATE_* environment variables and by command-line flags.", func(filename string) error {
		b, err := os.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("failed to read config file: %v", err)
		}
		preFile := proto.Clone(c.config)
		err = lpc.Unmarshal(filename, b)
		if err != nil {
			return fmt.Errorf("failed to parse config file: %v", err)
		}
		// Precedence implemented here: flags > env vars > config file > proto defaults
		// (PCLI-09). Flags win because flag.Parse runs after lpc.Environment() and writes
		// into this same message; see command.ConfigLayerer for the file/env/default
		// layering, which tracks provenance explicitly rather than inferring it from value
		// comparison (08-REVIEW.md CR-01). The ordering now holds across repeated
		// -config-file flags and for message-typed fields, not only for a single file and
		// scalar fields.
		layerer.LayerConfigFile(c.config, preFile)
		return nil
	})
	return c, nil
}
