package inserter

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/kvtools/consul"
	"github.com/kvtools/etcdv3"
	"github.com/kvtools/valkeyrie"
	"github.com/kvtools/valkeyrie/store"
	"github.com/kvtools/zookeeper"
	"github.com/mitchellh/cli"
	"github.com/protoconf/protoconf/agent/configmaps"
	"github.com/protoconf/protoconf/command"
	"github.com/protoconf/protoconf/compiler/lib"
	"github.com/protoconf/protoconf/compiler/lib/parser"
	"github.com/protoconf/protoconf/consts"
	datatypes "github.com/protoconf/protoconf/datatypes/proto/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const Stable = "STABLE"

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
	logger := slog.Default()
	logger.With("args", args).Info("starting inserter")
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
	} else if kVConfig.Store == command.KVStoreConfigMaps {
		kvStore, err = configmaps.New(ctx, []string{}, &configmaps.Config{Namespace: kVConfig.Namespace})
	} else {
		log.Fatalf("Unknown key-value store %s", kVConfig.Store)
	}

	if err != nil {
		logger.With("error", err).Error("Error connecting to key-value store")
		return 1
	}

	if config.delete {
		for i := 0; i < flags.NArg(); i++ {
			configName := filepath.ToSlash(strings.TrimSpace(flags.Args()[i]))
			if err := kvStore.Delete(ctx, kVConfig.Prefix+configName); err != nil {
				logger.With("error", err, "key", configName).Error("Error deleting config")
				return 1
			}
		}
	} else {
		protoconfRoot := strings.TrimSpace(flags.Args()[0])
		inserter := NewProtoconfInserter(protoconfRoot, kvStore)
		inserter.Prefix = kVConfig.Prefix
		wg := &sync.WaitGroup{}
		for i := 1; i < flags.NArg(); i++ {
			wg.Add(1)
			go func(path string) {
				defer wg.Done()

				configName := filepath.ToSlash(strings.TrimSpace(path))
				if err := inserter.InsertConfigFile(configName); err != nil {
					logger.With("key", configName, "error", err).Error("Error inserting config")
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
	repo          *git.Repository
	rel           string
	logger        *slog.Logger
	isGit         bool
}

func NewProtoconfInserter(protoconfRoot string, kvStore store.Store) *ProtoconfInserter {
	logger := slog.Default()
	logger.Debug("loading module service")
	ms := lib.NewModuleService(protoconfRoot)
	ms.LoadFromLockFile()
	protoconfRootAbs, _ := filepath.Abs(protoconfRoot)
	gitRoot := protoconfRootAbs
	isGit := true

	logger.Debug("loading git info")
	repo, err := git.PlainOpen(gitRoot)
	var rel string
	for errors.Is(err, git.ErrRepositoryNotExists) {
		gitRoot = filepath.Dir(gitRoot)
		if gitRoot == "/" {
			break
		}
		rel, _ = filepath.Rel(gitRoot, protoconfRootAbs)
		repo, err = git.PlainOpen(gitRoot)
	}
	if err != nil {
		logger.Info("not a git repo")
		isGit = false
	}
	return &ProtoconfInserter{
		protoconfRoot: protoconfRoot,
		kvStore:       kvStore,
		parser:        parser.NewParser(ms.GetProtoFilesRegistry()),
		repo:          repo,
		rel:           rel,
		logger:        logger,
		isGit:         isGit,
	}

}

var ErrInsertionCompleted = errors.New("insertion completed")

func (i *ProtoconfInserter) InsertConfigFile(configFile string) error {
	if !strings.HasSuffix(configFile, consts.CompiledConfigExtension) {
		configFile = configFile + consts.CompiledConfigPath
	}
	metadata, err := i.GatherMetadata(filepath.Join(i.rel, consts.CompiledConfigPath, configFile))
	if err != nil {
		return err
	}
	configName := strings.TrimSuffix(configFile, consts.CompiledConfigExtension)

	protoconfValue := &datatypes.ProtoconfValue{}
	err = i.parser.ReadConfig(filepath.Join(i.protoconfRoot, consts.CompiledConfigPath, configFile), protoconfValue)
	if err != nil {
		return err
	}
	protoconfValue.Metadata = metadata
	return i.InsertConfig(configName, protoconfValue, metadata)

}

func (i *ProtoconfInserter) InsertConfig(configName string, protoconfValue *datatypes.ProtoconfValue, metadata *datatypes.Metadata) error {
	now := time.Now()
	logger := i.logger.With("key", configName, "commit", metadata.Commit[0:8])
	err := i.XXXinsertVersion(configName, fmt.Sprintf("%d.%s", metadata.CommittedAt.Seconds, metadata.Commit), protoconfValue, metadata)
	if err != nil {
		return err
	}

	kvRolloutConfig := filepath.Join(i.Prefix, configName, "rollout.json")
	if protoconfValue.RolloutConfig != nil {
		rolloutConfig := &datatypes.ProtoconfValue_ConfigRollout{
			DefaultCooldownTime:   durationpb.New(time.Second * 60),
			DefaultExpirationTime: durationpb.New(time.Second * 300),
		}
		proto.Merge(rolloutConfig, protoconfValue.RolloutConfig)
		rolloutConfig.Stages = []*datatypes.ProtoconfValue_ConfigRollout_Stage{}
		ctx, cancel := context.WithCancelCause(context.Background())
		ctx, _ = signal.NotifyContext(ctx, os.Interrupt)
		context.AfterFunc(ctx, func() {
			err := context.Cause(ctx)
			logger.With("error", err).Error("stopped. deleting rollout config")
			logger.With("result", i.kvStore.Put(context.Background(), kvRolloutConfig, []byte("{}"), &store.WriteOptions{})).Info("deleted rollout config")
		})
		defer cancel(ErrInsertionCompleted)
		for _, stage := range protoconfValue.RolloutConfig.Stages {
			stageLogger := logger.With("stage", stage.Channel)
			select {
			case <-ctx.Done():
				stageLogger.Info("context canceled, skipping stage")
				return nil
			default:
				expiration := stage.Expiration
				if expiration == nil {
					expiration = rolloutConfig.DefaultExpirationTime
				}
				stage.ExpiresAt = timestamppb.New(time.Now().Add(expiration.AsDuration()))
				stage.Version = fmt.Sprintf("%d.%s", metadata.CommittedAt.Seconds, metadata.Commit)
				rolloutConfig.Stages = append(rolloutConfig.Stages, stage)
				data, err := protojson.MarshalOptions{Indent: "  "}.Marshal(rolloutConfig)
				if err != nil {
					return fmt.Errorf("error marshaling rollout to json, value=%v", rolloutConfig)
				}
				if err := i.kvStore.Put(ctx, kvRolloutConfig, data, nil); err != nil {
					return fmt.Errorf("error writing to key-value store, path=%s", kvRolloutConfig)
				}
				cooldown := stage.Cooldown
				if cooldown == nil {
					cooldown = rolloutConfig.DefaultCooldownTime
				}
				stageLogger.With("cooldown", cooldown.AsDuration()).Info("rollout stage updated")
				sleep, cancel := context.WithTimeout(ctx, cooldown.AsDuration())
				defer cancel()
				<-sleep.Done()
				if context.Cause(ctx) != nil {
					return context.Cause(ctx)
				}
			}
		}
	} else {
		logger.With("result", i.kvStore.Put(context.Background(), kvRolloutConfig, []byte("{}"), &store.WriteOptions{})).Info("Setting empty rollout.json")
	}

	err = i.XXXinsertVersion(configName, Stable, protoconfValue, metadata)
	if err != nil {
		return err
	}

	logger.With("completed_in", time.Since(now)).Info("inserted successfully")
	return nil
}

func (i *ProtoconfInserter) XXXinsertVersion(configName string, version string, protoconfValue *datatypes.ProtoconfValue, metadata *datatypes.Metadata) error {
	logger := i.logger.With("key", configName, "version", version, "commit", metadata.Commit[0:8])
	logger.Debug("starting version insertion")
	kvConfigJsonPath := filepath.Join(i.Prefix, configName, version, "config.json")
	kvConfigPbPath := filepath.Join(i.Prefix, configName, version, "config.data")
	kvMetadataPath := filepath.Join(i.Prefix, configName, version, "metadata.json")
	if version == Stable {
		kvConfigJsonPath = filepath.Join(i.Prefix, configName, "config.json")
		kvConfigPbPath = filepath.Join(i.Prefix, configName, "config.data")
		kvMetadataPath = filepath.Join(i.Prefix, configName, "metadata.json")
	}
	ctx := context.Background()

	// Writing config data
	logger.Debug("writing config binary data")
	data, err := proto.Marshal(protoconfValue)
	if err != nil {
		return fmt.Errorf("error marshaling ProtoconfValue to bytes, value=%v", protoconfValue)
	}
	write := base64.StdEncoding.EncodeToString(data)
	if err := i.kvStore.Put(ctx, kvConfigPbPath, []byte(write), nil); err != nil {
		return errors.Join(err, fmt.Errorf("error writing to key-value store, path=%s", kvConfigPbPath))
	}
	logger.Debug("finished writing config binary data")

	// Writing config json
	logger.Debug("writing config json data")
	mt, err := i.parser.LocalResolver.FindMessageByURL(protoconfValue.Value.TypeUrl)
	if err != nil {
		return err
	}
	new := dynamicpb.NewMessage(mt.Descriptor())
	err = protoconfValue.Value.UnmarshalTo(new)
	if err != nil {
		return err
	}

	data, err = protojson.MarshalOptions{Multiline: true}.Marshal(new)
	if err != nil {
		return errors.Join(err, fmt.Errorf("error marshaling ProtoconfValue to json, value=%v", protoconfValue))
	}
	if err := i.kvStore.Put(ctx, kvConfigJsonPath, data, nil); err != nil {
		return errors.Join(err, fmt.Errorf("error writing to key-value store, path=%s", kvConfigJsonPath))
	}
	logger.Debug("finished writing config json data")

	// Writing metadata json
	logger.Debug("writing metadata json")
	data, err = protojson.MarshalOptions{Multiline: true}.Marshal(metadata)
	if err != nil {
		return errors.Join(err, fmt.Errorf("error marshaling metadata to json, value=%v", protoconfValue))
	}
	if err := i.kvStore.Put(ctx, kvMetadataPath, data, nil); err != nil {
		return fmt.Errorf("error writing to key-value store, path=%s", kvMetadataPath)
	}
	logger.Debug("finished writing metadata json")
	return nil

}

var metaMutex *sync.Mutex = &sync.Mutex{}

func (i *ProtoconfInserter) GatherMetadata(configFile string) (*datatypes.Metadata, error) {
	metaMutex.Lock()
	defer metaMutex.Unlock()
	if !i.isGit {
		return &datatypes.Metadata{Commit: "not_a_git_repo"}, nil
	}
	gitLog, err := i.repo.Log(&git.LogOptions{FileName: &configFile})
	if err != nil {
		return nil, errors.Join(err, fmt.Errorf("error getting git log for file %s", configFile))
	}
	commit, err := gitLog.Next()
	if err != nil {
		return nil, errors.Join(err, fmt.Errorf("error getting next git log item for file %s", configFile))
	}
	return &datatypes.Metadata{
		Commit:         commit.Hash.String(),
		CommitterEmail: commit.Committer.Email,
		AuthorEmail:    commit.Author.Email,
		CommittedAt:    timestamppb.New(commit.Committer.When),
		AuthoredAt:     timestamppb.New(commit.Committer.When),
		InsertedAt:     timestamppb.Now(),
	}, nil

}
