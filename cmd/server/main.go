package main

import (
	"github.com/protoconf/protoconf/command"
	"github.com/protoconf/protoconf/server"
)

func main() {
	command.RunCommand("server", server.Command)
}
