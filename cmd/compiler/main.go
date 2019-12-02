package main

import (
	"github.com/protoconf/protoconf/command"
	"github.com/protoconf/protoconf/compiler"
)

func main() {
	command.RunCommand("compiler", compiler.Command)
}
