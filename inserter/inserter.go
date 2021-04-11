package inserter

import (
	"bytes"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/abronan/valkeyrie"
	"github.com/abronan/valkeyrie/store"
	"github.com/abronan/valkeyrie/store/consul"
	"github.com/abronan/valkeyrie/store/etcd/v2"
	"github.com/abronan/valkeyrie/store/zookeeper"
	"github.com/golang/protobuf/proto"
	"github.com/mitchellh/cli"
	"github.com/protoconf/protoconf/command"
	"github.com/protoconf/protoconf/consts"
	"github.com/protoconf/protoconf/utils"
)

type cliCommand struct{}

type cliConfig struct {
	delete bool
}

func newFlagSet() (*flag.FlagSet, *cliConfig, *command.KVStoreConfig) {
	flags := flag.NewFlagSet("", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintln(flags.Output(), "Usage: [OPTION]... [protoconf_root] config...")
		flags.PrintDefaults()
	}

	kVConfig := &command.KVStoreConfig{}
	command.AddKVStoreFlags(flags, kVConfig)

	config := &cliConfig{}
	flags.BoolVar(&config.delete, "d", false, "Delete a config from the key-value store")

	return flags, config, kVConfig
}

func (c *cliCommand) Run(args []string) int {
	flags, config, kVConfig := newFlagSet()
	flags.Parse(args)

	if flags.NArg() < 1 || (!config.delete && flags.NArg() < 2) {
		flags.Usage()
		return 1
	}

	var kvStore store.Store
	var err error
	if kVConfig.Store == command.KVStoreConsul {
		consul.Register()
		kvStore, err = valkeyrie.NewStore(store.CONSUL, []string{kVConfig.Address}, nil)
	} else if kVConfig.Store == command.KVStoreEtcd {
		etcd.Register()
		var address string
		if kVConfig.Address != "" {
			address = kVConfig.Address
		} else {
			address = consts.EtcdDefaultAddress
		}
		kvStore, err = valkeyrie.NewStore(store.ETCD, []string{address}, nil)
	} else if kVConfig.Store == command.KVStoreZookeeper {
		zookeeper.Register()
		var address string
		if kVConfig.Address != "" {
			address = kVConfig.Address
		} else {
			address = consts.ZookeeperDefaultAddress
		}
		kvStore, err = valkeyrie.NewStore(store.ZK, []string{address}, nil)
	} else {
		log.Fatalf("Unknown key-value store %s", kVConfig.Store)
	}

	if err != nil {
		log.Printf("Error connecting to key-value store, err=%s", err)
		return 1
	}

	if config.delete {
		for i := 0; i < flags.NArg(); i++ {
			configName := filepath.ToSlash(strings.TrimSpace(flags.Args()[i]))
			if err := kvStore.Delete(kVConfig.Prefix + configName); err != nil {
				log.Printf("Error deleting config %s, err=%s", configName, err)
				return 1
			}
		}
	} else {
		protoconfRoot := strings.TrimSpace(flags.Args()[0])
		for i := 1; i < flags.NArg(); i++ {
			configName := filepath.ToSlash(strings.TrimSpace(flags.Args()[i]))
			if err := insertConfig(configName, protoconfRoot, kvStore, kVConfig.Prefix); err != nil {
				log.Printf("Error inserting config %s, err=%s", configName, err)
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
	flags, _, _ := newFlagSet()
	flags.SetOutput(&b)
	flags.Usage()
	return b.String()
}

func (c *cliCommand) Synopsis() string {
	return "Insert a materialized config to the key-value store"
}

// Command is a cli.CommandFactory
func Command() (cli.Command, error) {
	return &cliCommand{}, nil
}

func insertConfig(configFile string, protoconfRoot string, kvStore store.Store, prefix string) error {
	if !strings.HasSuffix(configFile, consts.CompiledConfigExtension) {
		return fmt.Errorf("config must be a %s file, file=%s", consts.CompiledConfigExtension, configFile)
	}
	configName := strings.TrimSuffix(configFile, consts.CompiledConfigExtension)

	protoconfValue, err := utils.ReadConfig(protoconfRoot, configName)
	if err != nil {
		return err
	}

	data, err := proto.Marshal(protoconfValue)
	if err != nil {
		return fmt.Errorf("error marshaling ProtoconfValue to bytes, value=%v", protoconfValue)
	}

	kvPath := prefix + configName
	write := base64.StdEncoding.EncodeToString(data)
	if err := kvStore.Put(kvPath, []byte(write), nil); err != nil {
		return fmt.Errorf("error writing to key-value store, path=%s", kvPath)
	}

	fmt.Printf("Path %s inserted successfully\n", kvPath)
	return nil
}
