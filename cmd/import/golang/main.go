package main

import (
	"github.com/protoconf/protoconf/command"
	"github.com/protoconf/protoconf/importers/golang_importer"
)

func main() {
	command.RunCommand("golang", golangimporter.Command)
}
