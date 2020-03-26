package golangimporter

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	// "path/filepath"

	// "strings"

	"github.com/mitchellh/cli"
	"github.com/mitchellh/go-homedir"
)

type cliCommand struct {
	ui cli.Ui
}

var ui = &cli.BasicUi{
	Reader:      os.Stdin,
	Writer:      os.Stdout,
	ErrorWriter: os.Stderr,
}

type cliConfig struct {
	pkg                    string
	top                    string
	outputPath             string
	goSrcHome              string
	filterFiles            bool
	filterFilesAndMessages bool
}

func newFlagSet() (*flag.FlagSet, *cliConfig) {
	flags := flag.NewFlagSet("", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintln(flags.Output(), "Usage: [OPTION]...")
		helpText := `
	Use this command to create proto files from golang structs.
	You should make sure the go package you want to import is in your $GOPATH/src
	`
		fmt.Fprintln(flags.Output(), helpText)
		flags.PrintDefaults()
	}

	config := &cliConfig{}
	flags.StringVar(&config.outputPath, "output", "src", "Path to write proto files to.")
	flags.StringVar(&config.pkg, "package", "", "Fully qualified go package name to import")
	flags.StringVar(&config.top, "top", "", "Name of the top config struct to be used (required by -filter_files and -filter_messages)")
	flags.BoolVar(&config.filterFiles, "filter_files", false, "Remove files which are not required by -top message")
	flags.BoolVar(&config.filterFilesAndMessages, "filter_messages", false, "Same as -filter_files but also clean up the files from unused messages")
	flags.StringVar(&config.goSrcHome, "gopath", filepath.Join(os.Getenv("GOPATH"), "src"), "Override $GOPATH/src path")

	return flags, config
}

func (c *cliCommand) Run(args []string) int {
	flags, config := newFlagSet()
	flags.Parse(args)

	if config.pkg == "" {
		c.ui.Output(c.Help())
		return 0
	}

	env := []string{}
	userHome, err := homedir.Dir()
	if err != nil {
		c.ui.Warn("Failed to detect user's home")
	} else {
		env = append(env, "HOME="+userHome)
	}
	i, err := NewGolangImporter(config.pkg, config.outputPath, config.goSrcHome, env...)
	if err != nil {
		ui.Error(fmt.Sprintf("Failed to initialize the golang importer: %v", err))
		return 1
	}
	importer := i.GetImporter()

	mainFileForFiltering := i.GetFileNameFor(filepath.Base(config.pkg), config.pkg)
	ui.Warn(mainFileForFiltering)
	if config.filterFiles {
		filtered := importer.FilterFiles(mainFileForFiltering, config.top)
		importer.Files = filtered
	}
	if config.filterFilesAndMessages {
		filtered := importer.FilterFilesAndMessages(mainFileForFiltering, config.top)
		importer.Files = filtered
	}
	importer.SaveAll()
	log.Println(mainFileForFiltering, filepath.Base(config.pkg), config.pkg)
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
	return "Creates proto files from go structs"
}

// Command is a cli.CommandFactory
func Command() (cli.Command, error) {
	return &cliCommand{ui: ui}, nil
}
