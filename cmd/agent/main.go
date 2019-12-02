package main

import (
	"github.com/protoconf/protoconf/agent"
	"github.com/protoconf/protoconf/command"
)

func main() {
	command.RunCommand("agent", agent.Command)
}
