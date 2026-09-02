package agent

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/mitchellh/cli"
	"google.golang.org/protobuf/proto"

	configtool "github.com/protoconf/libprotoconf"
	protoconf_agent_config "github.com/protoconf/protoconf/agent/config/v1"
	"github.com/protoconf/protoconf/command"
)

type cliCommand struct {
	config *protoconf_agent_config.AgentConfig
	flag   *flag.FlagSet
}

func (c *cliCommand) Run(args []string) int {
	err := c.flag.Parse(args)
	if err != nil {
		fmt.Fprint(os.Stderr, "failed to parse flags", err)
		return 2
	}
	if err := RunAgent(context.Background(), c.config); err != nil {
		slog.Default().Error("error running agent", "error", err)
		return 1
	}
	return 0
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
	return "Start a gRPC agent that serves configs from a key-value store to subscribers"
}

// Command is a cli.CommandFactory
func Command() (cli.Command, error) {
	c := &cliCommand{
		config: &protoconf_agent_config.AgentConfig{
			GrpcAddress: ":4300",
			HttpAddress: ":4380",
		}}
	lpc := configtool.NewConfig(c.config)
	lpc.SetEnvKeyPrefix("PROTOCONF_AGENT")
	// base is the pristine factory-default snapshot handed to command.NewConfigLayerer below.
	// It is no longer mutated as an accumulator (that role now belongs to layerer.fileLayer);
	// it captures GrpcAddress: ":4300" and HttpAddress: ":4380".
	base := proto.Clone(c.config)
	lpc.Environment()
	c.flag = flag.NewFlagSet(string(c.config.ProtoReflect().Descriptor().FullName()), flag.ContinueOnError)
	lpc.PopulateFlagSet(c.flag)

	c.flag.VisitAll(func(f *flag.Flag) {
		switch f.Name {
		case "dev":
			f.Usage = "Run the agent in development mode watching local protoconf directory for file changes\n[env: PROTOCONF_AGENT_DEV]"
		case "enable-otel":
			f.Usage = "Export OpenTelemetry traces and metrics to an OTLP/gRPC collector. Off by default -- no collector is contacted when unset\n[env: PROTOCONF_AGENT_ENABLE_OTEL]"
		case "grpc-address":
			f.Usage = "Address to bind the gRPC listener\n[env: PROTOCONF_AGENT_GRPC_ADDRESS]"
		case "http-address":
			f.Usage = "Address to bind the admin HTTP listener\n[env: PROTOCONF_AGENT_HTTP_ADDRESS]"
		case "insecure":
			f.Usage = "Skip TLS gRPC TLS configuration\n[env: PROTOCONF_AGENT_INSECURE]"
		case "prefix":
			f.Usage = "Key-value store key prefix\n[env: PROTOCONF_AGENT_PREFIX]"
		case "store-address":
			f.Usage = "Key-value store addresses\n" + f.Usage
		case "store":
			f.Usage = "Key-value store type\n" + f.Usage
		}
	})
	// layerer owns the accumulated config-file layer and the env/flag provenance set for
	// this Command() instance. Constructed after PopulateFlagSet (so c.flag is fully
	// populated) and before flag.Parse runs (so markExplicitFlags can observe flags parsed
	// before each -config-file occurrence via c.flag.Visit).
	layerer := command.NewConfigLayerer(base, c.flag)
	c.flag.Func("config-file", "Agent configuration file (available formats: json, jsonnet, yaml, pb). Values are overridden by PROTOCONF_AGENT_* environment variables and by command-line flags.", func(filename string) error {
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
		// comparison (08-REVIEW.md CR-01). This is a change from the agent's historical
		// behavior, where a config file previously overrode PROTOCONF_AGENT_* environment
		// variables. The ordering now holds across repeated -config-file flags and for the
		// message-typed tls_config / store_tls fields, not only for a single file and
		// scalar fields.
		layerer.LayerConfigFile(c.config, preFile)
		return nil
	})

	return c, nil
}
