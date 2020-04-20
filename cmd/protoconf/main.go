package main

import (
	"github.com/mitchellh/cli"
	"github.com/protoconf/protoconf/agent"
	"github.com/protoconf/protoconf/command"
	"github.com/protoconf/protoconf/compiler"
	"github.com/protoconf/protoconf/exec"
	golangimporter "github.com/protoconf/protoconf/importers/golang_importer"
	terraformimporter "github.com/protoconf/protoconf/importers/terraform_importer"
	"github.com/protoconf/protoconf/inserter"
	"github.com/protoconf/protoconf/mutate"
	"github.com/protoconf/protoconf/server"
)

func main() {
	command.RunSubcommands("protoconf",
		map[string]cli.CommandFactory{
			"agent":            agent.Command,
			"compile":          compiler.Command,
			"exec":             exec.Command,
			"import golang":    golangimporter.Command,
			"import terraform": terraformimporter.Command,
			"insert":           inserter.Command,
			"mutate":           mutate.Command,
			"serve":            server.Command,
		},
	)
}
