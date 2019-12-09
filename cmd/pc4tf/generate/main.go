package main

import (
	"github.com/protoconf/protoconf/command"
	"github.com/protoconf/protoconf/pc4tf/generate"
)

func main() {
	command.RunCommand("pc4tf", generate.Command)
}
