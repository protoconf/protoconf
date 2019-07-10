package main

import (
	"github.com/mitchellh/cli"
	"protoconf.com/command"
	"protoconf.com/agent"
	"protoconf.com/compiler"
	"protoconf.com/inserter"
	"protoconf.com/server"
)

func main() {
	command.RunSubcommands("protoconf",
		map[string]cli.CommandFactory{
			"agent": agent.Command,
			"compile": compiler.Command,
			"insert": inserter.Command,
			"serve": server.Command,
		},
	)
}
