package terraformimporter

import (
	"bytes"
	"flag"
	"fmt"
	"log"

	// "path/filepath"
	"runtime"
	// "strings"

	"github.com/mitchellh/cli"
)

type cliCommand struct{}

type cliConfig struct {
	importPath string
	outputPath string
}

func newFlagSet() (*flag.FlagSet, *cliConfig) {
	flags := flag.NewFlagSet("", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintln(flags.Output(), "Usage: [OPTION]...")
		flags.PrintDefaults()
	}

	config := &cliConfig{}
	flags.StringVar(&config.importPath, "import_path", fmt.Sprintf(".terraform/providers/*/*/*/*/%s_%s", runtime.GOOS, runtime.GOARCH), "Path of terraform plugins")
	flags.StringVar(&config.outputPath, "output", "src", "Path to write proto files to.")

	return flags, config
}

func (c *cliCommand) Run(args []string) int {
	flags, config := newFlagSet()
	flags.Parse(args)

	g := NewGenerator(config.importPath, config.outputPath)
	err := g.PopulateProviders()
	if err != nil {
		log.Println("Failed to generate providers", err)
		return 1
	}
	err = g.Save()
	if err != nil {
		log.Println("failed to write proto files", err)
		return 1
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
	return "Creates proto files from terraform providers schema"
}

// Command is a cli.CommandFactory
func Command() (cli.Command, error) {
	return &cliCommand{}, nil
}
