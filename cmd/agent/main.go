package main

import (
	"protoconf.com/command"
	"protoconf.com/agent"
)

func main() {
	command.RunCommand("agent", agent.Command)
}
