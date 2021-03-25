package main

import (
	"github.com/protoconf/protoconf/command"
	"github.com/protoconf/protoconf/exec"
)

func main() {
	command.RunCommand("exec", exec.Command)
}
