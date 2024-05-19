package mod

import (
	"bytes"
	"context"
	"flag"
	"os"

	"github.com/mitchellh/cli"
	"github.com/protoconf/libprotoconf"
	"github.com/protoconf/protoconf/compiler/lib"
)

type modInitCommand struct {
	ui   cli.Ui
	ms   *lib.ModuleService
	flag *flag.FlagSet
}

type modSyncCommand struct {
	ui   cli.Ui
	ms   *lib.ModuleService
	flag *flag.FlagSet
}

type modTidyCommand struct {
	ui   cli.Ui
	ms   *lib.ModuleService
	flag *flag.FlagSet
}

func help(synopsis string, fs *flag.FlagSet) string {
	var b bytes.Buffer
	b.WriteString(synopsis)
	b.WriteString("\n")
	fs.SetOutput(&b)
	fs.Usage()
	return b.String()
}

var _ cli.Command = (*modInitCommand)(nil)
var _ cli.Command = (*modSyncCommand)(nil)

func (c *modInitCommand) Synopsis() string {
	return "Init"
}

func (c *modInitCommand) Help() string {
	return help(c.Synopsis(), c.flag)
}

func (c *modInitCommand) Run(args []string) int {
	c.flag.Parse(args)
	c.ui.Info(c.ms.Config.String())
	c.ms.LoadFromLockFile()
	err := c.ms.Init(context.Background(), "CONFIGSPACE")
	if err != nil {
		c.ui.Error(err.Error())
		return 1
	}
	return 0
}

func defaultModuleService(fs *flag.FlagSet) *lib.ModuleService {
	ms := lib.NewModuleService(".")
	lpc := libprotoconf.NewConfig(ms.Config)
	lpc.SetEnvKeyPrefix("PROTOCONF_MOD")
	lpc.Environment()
	lpc.PopulateFlagSet(fs)
	return ms
}

func NewInitCommand() (cli.Command, error) {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)

	return &modInitCommand{
		ui: &cli.BasicUi{
			Writer:      os.Stdout,
			ErrorWriter: os.Stderr,
		},
		ms:   defaultModuleService(fs),
		flag: fs,
	}, nil
}

func (c *modSyncCommand) Synopsis() string {
	return "Sync"
}

func (c *modSyncCommand) Help() string {
	return help(c.Synopsis(), c.flag)
}

func (c *modSyncCommand) Run(args []string) int {
	c.flag.Parse(args)
	err := c.ms.LoadFromLockFile()
	if err != nil {
		c.ui.Error(err.Error())
		return 1
	}
	err = c.ms.Sync(context.Background())
	if err != nil {
		c.ui.Error(err.Error())
		return 1
	}
	return 0
}

func NewSyncCommand() (cli.Command, error) {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	return &modSyncCommand{
		ui: &cli.BasicUi{
			Writer:      os.Stdout,
			ErrorWriter: os.Stderr,
		},
		ms:   defaultModuleService(fs),
		flag: fs,
	}, nil
}

func (c *modTidyCommand) Synopsis() string {
	return "Same as init + sync"
}

func (c *modTidyCommand) Help() string {
	return help(c.Synopsis(), c.flag)
}

func (c *modTidyCommand) Run(args []string) int {
	c.flag.Parse(args)
	err := c.ms.LoadFromLockFile()
	if err != nil {
		c.ui.Error(err.Error())
		return 1
	}
	err = c.ms.Init(context.Background(), "CONFIGSPACE")
	if err != nil {
		c.ui.Error("Failed to initialize CONFIGSPACE")
		c.ui.Error(err.Error())
		return 1
	}
	err = c.ms.Sync(context.Background())
	if err != nil {
		c.ui.Error("Failed to sync protoconf.lock")
		c.ui.Error(err.Error())
		return 1
	}
	return 0
}

func NewTidyCommand() (cli.Command, error) {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	return &modTidyCommand{
		ui: &cli.BasicUi{
			Writer:      os.Stdout,
			ErrorWriter: os.Stderr,
		},
		ms:   defaultModuleService(fs),
		flag: fs,
	}, nil
}
