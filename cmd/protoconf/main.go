package main

import (
	"github.com/mitchellh/cli"
	"github.com/protoconf/protoconf/agent"
	"github.com/protoconf/protoconf/command"
	"github.com/protoconf/protoconf/compiler"
	"github.com/protoconf/protoconf/inserter"
	"github.com/protoconf/protoconf/mod"
	"github.com/protoconf/protoconf/mutate"
	"github.com/protoconf/protoconf/server"
)

func main() {
	command.RunSubcommands("protoconf",
		map[string]cli.CommandFactory{
			"agent":    agent.Command,
			"compile":  compiler.Command,
			"insert":   inserter.Command,
			"mutate":   mutate.Command,
			"serve":    server.Command,
			"mod init": mod.NewInitCommand,
			"mod sync": mod.NewSyncCommand,
			"mod tidy": mod.NewTidyCommand,
		},
	)
}
