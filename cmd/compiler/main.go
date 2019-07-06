package main

import (
	"protoconf.com/command"
	"protoconf.com/compiler"
)

func main() {
	command.RunCommand("compiler", compiler.Command)
}
