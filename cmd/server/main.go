package main

import (
	"protoconf.com/command"
	"protoconf.com/server"
)

func main() {
	command.RunCommand("server", server.Command)
}
