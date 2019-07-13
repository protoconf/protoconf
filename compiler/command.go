package compiler

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/cli"
	"go.starlark.net/repl"
	"go.starlark.net/starlark"
	"protoconf.com/consts"
)

type cliCommand struct{}

type cliConfig struct {
	repl           bool
	verboseLogging bool
}

func newFlagSet() (*flag.FlagSet, *cliConfig) {
	flags := flag.NewFlagSet("", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintln(flags.Output(), "Usage: [OPTION]... protoconf_root [config]...")
		flags.PrintDefaults()
	}

	config := &cliConfig{}
	flags.BoolVar(&config.repl, "repl", false, "Interactive REPL mode")
	flags.BoolVar(&config.verboseLogging, "V", false, "Verbose logging")

	return flags, config
}

func (c *cliCommand) Run(args []string) int {
	flags, config := newFlagSet()
	flags.Parse(args)

	if flags.NArg() < 1 {
		flags.Usage()
		return 1
	}

	protoconfRoot := strings.TrimSpace(flags.Args()[0])
	compiler := NewCompiler(protoconfRoot, config.verboseLogging)

	if config.repl {
		compiler.REPL()
		return 0
	}

	var configs []string

	if flags.NArg() == 1 {
		var err error
		configs, err = getAllConfigs(protoconfRoot)
		if err != nil {
			log.Printf("Error getting all configs from %s, err=%s", protoconfRoot, err)
			return 1
		}
	} else {
		configs = flags.Args()[1:]
	}

	for _, config := range configs {
		filename := strings.TrimSpace(config)
		if err := compiler.CompileFile(filename); err != nil {
			log.Printf("Error compiling config %s, err=%s", filename, err)
			return 1
		}
	}

	return 0
}

func (c *cliCommand) Help() string {
	var b bytes.Buffer
	b.WriteString(c.Synopsis())
	b.WriteString("\n")
	flags, _ := newFlagSet()
	flags.SetOutput(&b)
	flags.Usage()
	return b.String()
}

func (c *cliCommand) Synopsis() string {
	return "Compile configs"
}

// Command is a cli.CommandFactory
func Command() (cli.Command, error) {
	return &cliCommand{}, nil
}

func getAllConfigs(protoconfRoot string) ([]string, error) {
	srcDir, err := filepath.Abs(filepath.Join(protoconfRoot, consts.SrcPath))
	if err != nil {
		return nil, err
	}

	var configs []string
	err = filepath.Walk(srcDir, func(path string, f os.FileInfo, err error) error {
		ext := filepath.Ext(path)
		if ext == consts.ConfigExtension || ext == consts.MultiConfigExtension {
			configs = append(configs, strings.TrimPrefix(path, srcDir))
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return configs, nil
}

func (c *compiler) REPL() {
	fmt.Printf("Protoconf %s\n", consts.Version)

	loader := c.getLoader()
	thread := &starlark.Thread{
		Load: loader.load,
	}

	repl.REPL(thread, loader.modules)
}
