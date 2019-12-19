package main

import (
	"github.com/mitchellh/cli"
	"github.com/protoconf/protoconf/command"
	"github.com/protoconf/protoconf/pc4tf/generate"
)

func main() {
	command.RunSubcommands("pc4tf",
		map[string]cli.CommandFactory{
			"generate": generate.Command,
		},
	)
}
