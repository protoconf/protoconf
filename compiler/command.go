package compiler

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/mitchellh/cli"
)

type cliCommand struct{}

type cliConfig struct {
	verboseLogging bool
}

func newFlagSet() (*flag.FlagSet, *cliConfig) {
	flags := flag.NewFlagSet("", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintln(flags.Output(), "Usage: [OPTION]... protoconf_root [config]...")
		flags.PrintDefaults()
	}

	config := &cliConfig{}
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

	compiler := NewCompiler(config.verboseLogging)
	protoconfRoot := strings.TrimSpace(flags.Args()[0])

	if flags.NArg() == 1 {
		// FIXME
		log.Printf("Error: compiling all the configs in the directory isn't supported yet")
		return 1
	} else {
		for i := 1; i < flags.NArg(); i++ {
			filename := strings.TrimSpace(flags.Args()[i])
			if err := compiler.CompileFile(filename, protoconfRoot); err != nil {
				log.Printf("Error compiling config %s, err: %s", filename, err)
				return 1
			}
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
