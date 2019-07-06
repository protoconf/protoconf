package main

import (
	"protoconf.com/command"
	"protoconf.com/inserter"
)

func main() {
	command.RunCommand("inserter", inserter.Command)
}
