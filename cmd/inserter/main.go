package main

import (
	"github.com/protoconf/protoconf/command"
	"github.com/protoconf/protoconf/inserter"
)

func main() {
	command.RunCommand("inserter", inserter.Command)
}
