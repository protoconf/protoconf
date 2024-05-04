package main

import (
	_ "github.com/bufbuild/protovalidate-go"
	_ "github.com/bufbuild/protovalidate-go/legacy"
	protoconf_pb "github.com/protoconf/protoconf/pb/protoconf/v1"

	"github.com/mitchellh/cli"
	"github.com/protoconf/protoconf/agent"
	"github.com/protoconf/protoconf/command"
	"github.com/protoconf/protoconf/compiler"
	"github.com/protoconf/protoconf/devserver"
	"github.com/protoconf/protoconf/inserter"
	"github.com/protoconf/protoconf/mod"
	"github.com/protoconf/protoconf/mutate"
	"github.com/protoconf/protoconf/server"
)

func init() {
	newCompileRequest := &protoconf_pb.ConfigMutationResponse{}
	_ = newCompileRequest
}

func main() {
	command.RunSubcommands("protoconf",
		map[string]cli.CommandFactory{
			"agent":     agent.Command,
			"compile":   compiler.Command,
			"devserver": devserver.Command,
			"insert":    inserter.Command,
			"mutate":    mutate.Command,
			"serve":     server.Command,
			"mod init":  mod.NewInitCommand,
			"mod sync":  mod.NewSyncCommand,
			"mod tidy":  mod.NewTidyCommand,
		},
	)
}
