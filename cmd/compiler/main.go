package main

import (
	"github.com/protoconf/protoconf/command"
	"github.com/protoconf/protoconf/compiler"
)

func main() {
	// compile does not need a GC
	// debug.SetGCPercent(-1)
	command.RunCommand("compiler", compiler.Command)
}
