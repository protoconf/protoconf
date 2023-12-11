package inserter

import (
	"bytes"
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/kvtools/consul"
	"github.com/kvtools/etcdv3"
	"github.com/kvtools/valkeyrie"
	"github.com/kvtools/valkeyrie/store"
	"github.com/kvtools/zookeeper"
	"github.com/mitchellh/cli"
	"github.com/protoconf/protoconf/command"
	"github.com/protoconf/protoconf/compiler/lib"
	"github.com/protoconf/protoconf/compiler/lib/parser"
	"github.com/protoconf/protoconf/consts"
	v1 "github.com/protoconf/protoconf/datatypes/proto/v1"
	"google.golang.org/protobuf/proto"
)

type cliCommand struct{}

type cliConfig struct {
	delete bool
}

func init() {
	// set the number of CPUs to use.
	// By default, the number of CPUs is the number of CPUs on the machine.
	// Just to show we can change the number of CPUs
	// I have added this below command.
	runtime.GOMAXPROCS(runtime.NumCPU())
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
	ctx := context.Background()
	if kVConfig.Store == command.KVStoreConsul {
		kvStore, err = valkeyrie.NewStore(ctx, consul.StoreName, []string{kVConfig.Address}, nil)
	} else if kVConfig.Store == command.KVStoreEtcd {
		var address string
		if kVConfig.Address != "" {
			address = kVConfig.Address
		} else {
			address = consts.EtcdDefaultAddress
		}
		kvStore, err = valkeyrie.NewStore(ctx, etcdv3.StoreName, []string{address}, nil)
	} else if kVConfig.Store == command.KVStoreZookeeper {
		var address string
		if kVConfig.Address != "" {
			address = kVConfig.Address
		} else {
			address = consts.ZookeeperDefaultAddress
		}
		kvStore, err = valkeyrie.NewStore(ctx, zookeeper.StoreName, []string{address}, nil)
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
			if err := kvStore.Delete(ctx, kVConfig.Prefix+configName); err != nil {
				log.Printf("Error deleting config %s, err=%s", configName, err)
				return 1
			}
		}
	} else {
		protoconfRoot := strings.TrimSpace(flags.Args()[0])
		inserter := NewProtoconfInserter(protoconfRoot, kvStore)
		inserter.Prefix = kVConfig.Prefix
		wg := &sync.WaitGroup{}
		path := flags.Args()[1]
		configName := filepath.ToSlash(strings.TrimSpace(path))
		if err := inserter.InsertConfig(configName); err != nil {
			log.Printf("Error inserting config %s, err=%s", configName, err)
		}
		for i := 1; i < flags.NArg(); i++ {
			wg.Add(1)
			go func(path string) {
				defer wg.Done()
				log.Print(path)

				configName := filepath.ToSlash(strings.TrimSpace(path))
				if err := inserter.InsertConfig(configName); err != nil {
					log.Printf("Error inserting config %s, err=%s", configName, err)
				}
			}(flags.Args()[i])
		}
		wg.Wait()
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

type ProtoconfInserter struct {
	protoconfRoot string
	kvStore       store.Store
	parser        *parser.Parser
	Prefix        string
}

func NewProtoconfInserter(protoconfRoot string, kvStore store.Store) *ProtoconfInserter {
	ms := lib.NewModuleService(protoconfRoot)
	ms.LoadFromLockFile()
	return &ProtoconfInserter{
		protoconfRoot: protoconfRoot,
		kvStore:       kvStore,
		parser:        parser.NewParser(ms.GetProtoFilesRegistry()),
	}

}

func (i *ProtoconfInserter) InsertConfig(configFile string) error {
	now := time.Now()
	if !strings.HasSuffix(configFile, consts.CompiledConfigExtension) {
		return fmt.Errorf("config must be a %s file, file=%s", consts.CompiledConfigExtension, configFile)
	}
	configName := strings.TrimSuffix(configFile, consts.CompiledConfigExtension)

	protoconfValue := &v1.ProtoconfValue{}
	err := i.parser.ReadConfig(filepath.Join(i.protoconfRoot, consts.CompiledConfigPath, configFile), protoconfValue)
	if err != nil {
		return err
	}

	data, err := proto.Marshal(protoconfValue)
	if err != nil {
		return fmt.Errorf("error marshaling ProtoconfValue to bytes, value=%v", protoconfValue)
	}

	kvPath := i.Prefix + configName
	write := base64.StdEncoding.EncodeToString(data)
	ctx := context.Background()
	if err := i.kvStore.Put(ctx, kvPath, []byte(write), nil); err != nil {
		return fmt.Errorf("error writing to key-value store, path=%s", kvPath)
	}

	fmt.Printf("Path %s inserted successfully (took: %v)\n", kvPath, time.Since(now))
	return nil
}
