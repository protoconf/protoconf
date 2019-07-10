package main

import (
	"fmt"
	"syscall/js"

	compiler "protoconf.com/compiler_library"
)

func compile(this js.Value, args []js.Value) interface{} {
	if err := compiler.NewCompiler("/", false).CompileFile(args[0].String()); err != nil {
		js.Global().Call("toast", fmt.Sprintf("Error compiling config, err: %s", err))
	}
	return nil
}

func main() {
	js.Global().Set("protoconfCompile", js.FuncOf(compile))
	c := make(chan struct{}, 0)
	<-c
}
