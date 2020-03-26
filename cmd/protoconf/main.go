package main

import (
	"github.com/mitchellh/cli"
	"github.com/protoconf/protoconf/agent"
	"github.com/protoconf/protoconf/command"
	"github.com/protoconf/protoconf/compiler"
	golangimporter "github.com/protoconf/protoconf/importers/golang_importer"
	"github.com/protoconf/protoconf/inserter"
	"github.com/protoconf/protoconf/server"
)

func main() {
	command.RunSubcommands("protoconf",
		map[string]cli.CommandFactory{
			"agent":   agent.Command,
			"compile": compiler.Command,
			"import golang":  golangimporter.Command,
			"insert":  inserter.Command,
			"serve":   server.Command,
		},
	)
}
