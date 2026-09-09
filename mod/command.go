package mod

import (
	"bytes"
	"context"
	"flag"
	"log/slog"
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

type modCommand struct{}

func (c *modCommand) Run(args []string) int { return cli.RunResultHelp }
func (c *modCommand) Help() string          { return "" }
func (c *modCommand) Synopsis() string {
	return "Manage protoconf module dependencies (init, sync, tidy)"
}

func NewModCommand() (cli.Command, error) {
	return &modCommand{}, nil
}

var _ cli.Command = (*modCommand)(nil)
var _ cli.Command = (*modInitCommand)(nil)
var _ cli.Command = (*modSyncCommand)(nil)

func (c *modInitCommand) Synopsis() string {
	return "Initialize a protoconf.lock file from proto module dependencies"
}

func (c *modInitCommand) Help() string {
	return help(c.Synopsis(), c.flag)
}

func (c *modInitCommand) Run(args []string) int {
	if err := c.flag.Parse(args); err != nil {
		c.ui.Error(err.Error())
		return 2
	}
	c.ui.Info(c.ms.Config.String())
	// Init loads the lock file itself now and returns its parse error
	// before executing CONFIGSPACE (see ModuleService.Init), so this no
	// longer needs its own unchecked LoadFromLockFile call -- which used to
	// swallow a malformed lock file's parse error entirely.
	err := c.ms.Init(context.Background(), "CONFIGSPACE")
	if err != nil {
		c.ui.Error(err.Error())
		return 1
	}
	return 0
}

func defaultModuleService(fs *flag.FlagSet) *lib.ModuleService {
	ms, err := lib.NewModuleService(".")
	if err != nil {
		return nil
	}
	lpc := libprotoconf.NewConfig(ms.Config)
	lpc.SetEnvKeyPrefix("PROTOCONF_MOD")
	if err := lpc.Environment(); err != nil {
		slog.Error("failed to load environment configuration", "error", err)
	}
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
	return "Download and cache proto dependencies declared in protoconf.lock"
}

func (c *modSyncCommand) Help() string {
	return help(c.Synopsis(), c.flag)
}

func (c *modSyncCommand) Run(args []string) int {
	if err := c.flag.Parse(args); err != nil {
		c.ui.Error(err.Error())
		return 2
	}
	err := c.ms.LoadFromLockFile()
	defer func() {
		if err := c.ms.Lock(); err != nil {
			c.ui.Error(err.Error())
		}
	}()
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
	return "Resolve, lock, and download all proto module dependencies (init + sync)"
}

func (c *modTidyCommand) Help() string {
	return help(c.Synopsis(), c.flag)
}

func (c *modTidyCommand) Run(args []string) int {
	if err := c.flag.Parse(args); err != nil {
		c.ui.Error(err.Error())
		return 2
	}
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
