package command

import (
	"flag"
	"fmt"
	"os"

	"github.com/mitchellh/cli"
	"protoconf.com/consts"
)

// RunCommand runs a single command
func RunCommand(commandName string, command cli.CommandFactory) {
	const defaultCommand = "default"
	run(commandName,
		append([]string{defaultCommand}, os.Args[1:]...),
		map[string]cli.CommandFactory{
			defaultCommand: command,
		})
}

// RunSubcommands a command given its subcommands
func RunSubcommands(commandName string, subcommands map[string]cli.CommandFactory) {
	run(commandName, os.Args[1:], subcommands)
}

func run(commandName string, args []string, subcommands map[string]cli.CommandFactory) {
	for _, arg := range args {
		if arg == "--" {
			break
		}

		if arg == "-v" || arg == "--version" {
			args = []string{"-v"}
			break
		}
	}

	c := cli.NewCLI(commandName, consts.Version)
	c.Args = args
	c.Commands = subcommands

	exitStatus, err := c.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}

	os.Exit(exitStatus)
}

// KVStoreConfig holds the key-value store configuration set from the command line
type KVStoreConfig struct {
	Address string
	Prefix string
}

// AddKVStoreFlags adds to an existing flagset the command lines flags to configure the key-value store connection
func AddKVStoreFlags(fs *flag.FlagSet, kv *KVStoreConfig) {
	fs.StringVar(&kv.Address, "kv", "", "Key-value store address")
	fs.StringVar(&kv.Prefix, "prefix", "", "Key-value store key prefix")
}
