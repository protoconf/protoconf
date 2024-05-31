package command

import (
	"flag"
	"fmt"
	"os"
	"regexp"

	"github.com/mitchellh/cli"
	"github.com/protoconf/protoconf/consts"
)

// RunCommand runs a single command
func RunCommand(commandName string, command cli.CommandFactory) {
	const defaultCommand = "default"
	run(commandName,
		append([]string{defaultCommand}, os.Args[1:]...),
		map[string]cli.CommandFactory{
			defaultCommand: command,
		})
}

// RunSubcommands a command given its subcommands
func RunSubcommands(commandName string, subcommands map[string]cli.CommandFactory) {
	run(commandName, os.Args[1:], subcommands)
}

func run(commandName string, args []string, subcommands map[string]cli.CommandFactory) {
	for _, arg := range args {
		if arg == "--" {
			break
		}

		if arg == "-v" || arg == "--version" {
			args = []string{"-v"}
			break
		}
	}

	c := cli.NewCLI(commandName, commandName+" "+consts.Version)
	c.Args = args
	c.Commands = subcommands

	exitStatus, err := c.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}

	os.Exit(exitStatus)
}

const (
	KVStoreConsul     = "consul"
	KVStoreZookeeper  = "zookeeper"
	KVStoreEtcd       = "etcd"
	KVStoreConfigMaps = "configmaps"
)

// KVStoreConfig holds the key-value store configuration set from the command line
type KVStoreConfig struct {
	Address   string
	Store     string
	Prefix    string
	Namespace string
}

// AddKVStoreFlags adds to an existing flagset the command lines flags to configure the key-value store connection
func AddKVStoreFlags(fs *flag.FlagSet, kv *KVStoreConfig) {
	fs.StringVar(&kv.Address, "store-address", "", "Key-value store address")
	fs.StringVar(&kv.Store, "store", KVStoreConsul, "Key-value store type (consul/zookeeper/etcd)")
	fs.StringVar(&kv.Prefix, "prefix", "", "Key-value store key prefix")
	fs.StringVar(&kv.Namespace, "namespace", "", "Kubernetes namespace for config maps insertion")
}

var DefaultUI = &cli.ConcurrentUi{
	Ui: &cli.ColoredUi{
		Ui: &cli.BasicUi{
			Writer:      os.Stdout,
			Reader:      os.Stdin,
			ErrorWriter: os.Stderr,
		},
		WarnColor:  cli.UiColorYellow,
		InfoColor:  cli.UiColorGreen,
		ErrorColor: cli.UiColorRed,
	},
}

type PrefixedUi struct {
	Ui     cli.Ui
	Prefix string
}

var _ = cli.Ui(&PrefixedUi{})

func NewPrefixedUi(prefix string) *PrefixedUi {
	return &PrefixedUi{
		Ui:     DefaultUI,
		Prefix: prefix,
	}
}

var re = regexp.MustCompile(`(?m)^`)

func (p *PrefixedUi) write(s string, f func(string)) {
	s = re.ReplaceAllString(s, p.Prefix)
	f(s)
}

func (p *PrefixedUi) Output(s string) {
	p.write(s, p.Ui.Output)
}

func (p *PrefixedUi) Info(s string) {
	p.write(s, p.Ui.Info)
}

func (p *PrefixedUi) Error(s string) {
	p.write(s, p.Ui.Error)
}

func (p *PrefixedUi) Ask(query string) (string, error) {
	return p.Ui.Ask(p.Prefix + query)
}

func (p *PrefixedUi) AskSecret(query string) (string, error) {
	return p.Ui.AskSecret(p.Prefix + query)
}

func (p *PrefixedUi) Warn(s string) {
	p.write(s, p.Ui.Warn)
}
