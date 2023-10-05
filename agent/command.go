package agent

import (
	"bytes"
	"flag"
	"fmt"
	"os"

	"github.com/mitchellh/cli"
	"google.golang.org/protobuf/proto"

	configtool "github.com/protoconf/libprotoconf"
	protoconf_agent_config "github.com/protoconf/protoconf/agent/config/v1"
)

type cliCommand struct {
	config *protoconf_agent_config.AgentConfig
	flag   *flag.FlagSet
}

func (c *cliCommand) Run(args []string) int {
	c.flag.Parse(args)
	return RunAgent(c.config)
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
	return "Runs a Protoconf agent"
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
	lpc.Environment()
	c.flag = lpc.DefaultFlagSet()
	c.flag.VisitAll(func(f *flag.Flag) {
		switch f.Name {
		case "dev":
			f.Usage = "Run the agent in development mode watching local protoconf directory for file changes\n[env: PROTOCONF_AGENT_DEV]"
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
	c.flag.Func("config-file", "Agent configuration file (available formats: json, jsonnet, yaml, pb)", func(filename string) error {
		b, err := os.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("failed to read config file: %v", err)
		}
		orig := proto.Clone(c.config)
		err = lpc.Unmarshal(filename, b)
		if err != nil {
			return fmt.Errorf("failed to parse config file: %v", err)
		}
		proto.Merge(orig, c.config)
		c.config, _ = orig.(*protoconf_agent_config.AgentConfig)
		return nil
	})

	return c, nil
}
