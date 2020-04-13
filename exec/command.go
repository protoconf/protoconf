package exec

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/mitchellh/cli"
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
	protoconfPath string
	protosDir     string
}

func newFlagSet() (*flag.FlagSet, *cliConfig) {
	flags := flag.NewFlagSet("", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintln(flags.Output(), "Usage: [OPTION]...")
		flags.PrintDefaults()
	}

	config := &cliConfig{}
	flags.StringVar(&config.protoconfPath, "config", "", "The path of the protoconf config.")
	flags.StringVar(&config.protosDir, "proto_dir", "", "The path on disk where the .proto files could be found.")

	return flags, config
}

func (c *cliCommand) Run(args []string) int {
	flags, config := newFlagSet()
	flags.Parse(args)
	e, err := NewExecutor(config.protoconfPath, config.protosDir)
	if err != nil {
		return 1
	}
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	defer func() {
		signal.Stop(ch)
		cancel()
	}()
	go func() {
		select {
		case <-ch:
			cancel()
		case <-ctx.Done():
		}
	}()
	err = e.Start(ctx)
	if err != nil {
		log.Fatal(err)
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
	return "Watches keys and execute on changes"
}

// Command is a cli.CommandFactory
func Command() (cli.Command, error) {
	return &cliCommand{ui: ui}, nil
}
