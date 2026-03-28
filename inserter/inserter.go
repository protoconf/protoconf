package inserter

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
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
	configtool "github.com/protoconf/libprotoconf"
	"github.com/protoconf/protoconf/agent/configmaps"
	"github.com/protoconf/protoconf/compiler/lib"
	"github.com/protoconf/protoconf/compiler/lib/parser"
	"github.com/protoconf/protoconf/consts"
	protoconf_inserter_config "github.com/protoconf/protoconf/inserter/config/v1"
	protoconf_pb "github.com/protoconf/protoconf/pb/protoconf/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const Stable = "STABLE"

type cliCommand struct {
	config *protoconf_inserter_config.InserterConfig
	flag   *flag.FlagSet
}

func (c *cliCommand) Run(args []string) int {
	logger := slog.Default()
	logger.With("args", args).Info("starting inserter")
	err := c.flag.Parse(args)
	if err != nil {
		fmt.Fprint(os.Stderr, "failed to parse flags", err)
		return 2
	}

	if c.flag.NArg() < 1 || (!c.config.Delete && c.flag.NArg() < 2) {
		c.flag.Usage()
		return 1
	}

	var kvStore store.Store
	ctx := context.Background()
	switch c.config.Store {
	case protoconf_inserter_config.InserterConfig_consul:
		addresses := c.config.StoreAddress
		if len(addresses) == 0 {
			addresses = []string{"127.0.0.1:8500"}
		}
		kvStore, err = valkeyrie.NewStore(ctx, consul.StoreName, addresses, nil)
	case protoconf_inserter_config.InserterConfig_etcd:
		addresses := c.config.StoreAddress
		if len(addresses) == 0 {
			addresses = []string{consts.EtcdDefaultAddress}
		}
		kvStore, err = valkeyrie.NewStore(ctx, etcdv3.StoreName, addresses, nil)
	case protoconf_inserter_config.InserterConfig_zookeeper:
		addresses := c.config.StoreAddress
		if len(addresses) == 0 {
			addresses = []string{consts.ZookeeperDefaultAddress}
		}
		kvStore, err = valkeyrie.NewStore(ctx, zookeeper.StoreName, addresses, nil)
	case protoconf_inserter_config.InserterConfig_configmaps:
		kvStore, err = configmaps.New(ctx, []string{}, &configmaps.Config{Namespace: c.config.Namespace})
	default:
		slog.Error("Unknown key-value store", "store", c.config.Store)
		return 1
	}

	if err != nil {
		logger.With("error", err).Error("Error connecting to key-value store")
		return 1
	}

	if c.config.Delete {
		for i := 0; i < c.flag.NArg(); i++ {
			configName := filepath.ToSlash(strings.TrimSpace(c.flag.Args()[i]))
			if err := kvStore.Delete(ctx, c.config.Prefix+configName); err != nil {
				logger.With("error", err, "key", configName).Error("Error deleting config")
				return 1
			}
		}
	} else {
		protoconfRoot := strings.TrimSpace(c.flag.Args()[0])
		inserter := NewProtoconfInserter(protoconfRoot, kvStore)
		inserter.Prefix = c.config.Prefix
		wg := &sync.WaitGroup{}
		for i := 1; i < c.flag.NArg(); i++ {
			wg.Add(1)
			go func(path string) {
				defer wg.Done()

				configName := filepath.ToSlash(strings.TrimSpace(path))
				if err := inserter.InsertConfigFile(configName); err != nil {
					logger.With("key", configName, "error", err).Error("Error inserting config")
				}
			}(c.flag.Args()[i])
		}
		wg.Wait()
	}

	return 0
}

func (c *cliCommand) Help() string {
	var b bytes.Buffer
	b.WriteString(c.Synopsis())
	b.WriteString("\n")
	c.flag.SetOutput(&b)
	c.flag.Usage()
	return b.String()
}

func (c *cliCommand) Synopsis() string {
	return "Insert a materialized config to the key-value store"
}

// Command is a cli.CommandFactory
func Command() (cli.Command, error) {
	c := &cliCommand{
		config: &protoconf_inserter_config.InserterConfig{},
	}
	lpc := configtool.NewConfig(c.config)
	lpc.SetEnvKeyPrefix("PROTOCONF_INSERTER")
	lpc.Environment()
	c.flag = flag.NewFlagSet(string(c.config.ProtoReflect().Descriptor().FullName()), flag.ContinueOnError)
	lpc.PopulateFlagSet(c.flag)
	c.flag.Func("config-file", "Inserter configuration file (available formats: json, yaml, pb)", func(filename string) error {
		b, err := os.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("failed to read config file: %v", err)
		}
		orig := proto.Clone(c.config)
		err = lpc.Unmarshal(filename, b)
		if err != nil {
			return fmt.Errorf("failed to parse config file: %v", err)
		}
		// NOTE: proto.Merge(orig, c.config) merges file values ON TOP of env var values,
		// meaning file values override env vars. This matches the agent pattern per D-13.
		// The stated precedence in PCLI-09 (env > file) would require the reverse merge
		// direction, but we intentionally match the existing agent behavior here. Fixing
		// the agent's merge direction is a separate concern outside this phase.
		proto.Merge(orig, c.config)
		c.config, _ = orig.(*protoconf_inserter_config.InserterConfig)
		return nil
	})
	return c, nil
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
	ms, err := lib.NewModuleService(protoconfRoot)
	if err != nil {
		logger.Error("error creating module service", "error", err)
		return nil
	}
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
		parser:        parser.NewParserWithDescriptorRegistry(ms.GetProtoRegistry()),
		repo:          repo,
		rel:           rel,
		logger:        logger,
		isGit:         isGit,
	}

}

func ensureConfigMaterialized(configFile string) string {
	if !strings.HasSuffix(configFile, consts.CompiledConfigExtension) {
		configFile = configFile + consts.CompiledConfigPath
	}
	if !strings.HasPrefix(configFile, consts.CompiledConfigPath) {
		configFile = filepath.Join(consts.CompiledConfigPath, configFile)
	}
	return configFile
}

var ErrInsertionCompleted = errors.New("insertion completed")

func (i *ProtoconfInserter) InsertConfigFile(configFile string) error {
	configFile = ensureConfigMaterialized(configFile)
	metadata, err := i.GatherMetadata(filepath.Join(i.rel, configFile))
	if err != nil {
		return err
	}
	configName := strings.TrimPrefix(strings.TrimSuffix(configFile, consts.CompiledConfigExtension), consts.CompiledConfigPath)

	protoconfValue := &protoconf_pb.ProtoconfValue{}
	err = i.parser.ReadConfig(filepath.Join(i.protoconfRoot, configFile), protoconfValue)
	if err != nil {
		return err
	}
	protoconfValue.Metadata = metadata
	return i.InsertConfig(configName, protoconfValue, metadata)
}

func (i *ProtoconfInserter) getKvStoreForConfig(protoconfValue *protoconf_pb.ProtoconfValue) store.Store {
	if ns := protoconfValue.GetRolloutConfig().GetNamespace(); ns != "" {
		kvStore, err := configmaps.New(context.Background(), []string{}, &configmaps.Config{Namespace: ns})
		if err == nil {
			return kvStore
		}
	}
	return i.kvStore

}

func (i *ProtoconfInserter) InsertConfig(configName string, protoconfValue *protoconf_pb.ProtoconfValue, metadata *protoconf_pb.Metadata) error {
	now := time.Now()
	logger := i.logger.With("key", configName, "commit", metadata.Commit[0:8], "namespace", protoconfValue.GetRolloutConfig().GetNamespace())
	err := i.XXXinsertVersion(configName, fmt.Sprintf("%d.%s", metadata.CommittedAt.Seconds, metadata.Commit), protoconfValue, metadata)
	if err != nil {
		return err
	}
	kvStore := i.getKvStoreForConfig(protoconfValue)

	kvRolloutConfig := filepath.Join(i.Prefix, configName, "rollout.json")
	if protoconfValue.RolloutConfig != nil {
		rolloutConfig := &protoconf_pb.ProtoconfValue_ConfigRollout{
			DefaultCooldownTime:   durationpb.New(time.Second * 60),
			DefaultExpirationTime: durationpb.New(time.Second * 300),
		}
		proto.Merge(rolloutConfig, protoconfValue.RolloutConfig)
		rolloutConfig.Stages = []*protoconf_pb.ProtoconfValue_ConfigRollout_Stage{}
		ctx, cancel := context.WithCancelCause(context.Background())
		ctx, _ = signal.NotifyContext(ctx, os.Interrupt)
		context.AfterFunc(ctx, func() {
			err := context.Cause(ctx)
			logger.With("error", err).Error("stopped. deleting rollout config")
			logger.With("result", kvStore.Put(context.Background(), kvRolloutConfig, []byte("{}"), &store.WriteOptions{})).Info("deleted rollout config")
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
				if err := kvStore.Put(ctx, kvRolloutConfig, data, nil); err != nil {
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
		logger.With("result", kvStore.Put(context.Background(), kvRolloutConfig, []byte("{}"), &store.WriteOptions{})).Info("Setting empty rollout.json")
	}

	err = i.XXXinsertVersion(configName, Stable, protoconfValue, metadata)
	if err != nil {
		return err
	}

	logger.With("completed_in", time.Since(now)).Info("inserted successfully")
	return nil
}

func (i *ProtoconfInserter) XXXinsertVersion(configName string, version string, protoconfValue *protoconf_pb.ProtoconfValue, metadata *protoconf_pb.Metadata) error {
	logger := i.logger.With("key", configName, "version", version, "commit", metadata.Commit[0:8], "namespace", protoconfValue.GetRolloutConfig().GetNamespace())
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
	kvStore := i.getKvStoreForConfig(protoconfValue)

	// Writing config data
	logger.Debug("writing config binary data")
	data, err := proto.Marshal(protoconfValue)
	if err != nil {
		return fmt.Errorf("error marshaling ProtoconfValue to bytes, value=%v", protoconfValue)
	}
	write := base64.StdEncoding.EncodeToString(data)
	if err := kvStore.Put(ctx, kvConfigPbPath, []byte(write), nil); err != nil {
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
	if err := kvStore.Put(ctx, kvConfigJsonPath, data, nil); err != nil {
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

func (i *ProtoconfInserter) GatherMetadata(configFile string) (*protoconf_pb.Metadata, error) {
	metaMutex.Lock()
	defer metaMutex.Unlock()
	if !i.isGit {
		return &protoconf_pb.Metadata{Commit: "not_a_git_repo"}, nil
	}
	gitLog, err := i.repo.Log(&git.LogOptions{FileName: &configFile})
	if err != nil {
		return nil, errors.Join(err, fmt.Errorf("error getting git log for file %s", configFile))
	}
	commit, err := gitLog.Next()
	if err != nil {
		return nil, errors.Join(err, fmt.Errorf("error getting next git log item for file %s", configFile))
	}
	return &protoconf_pb.Metadata{
		Commit:         commit.Hash.String(),
		CommitterEmail: commit.Committer.Email,
		AuthorEmail:    commit.Author.Email,
		CommittedAt:    timestamppb.New(commit.Committer.When),
		AuthoredAt:     timestamppb.New(commit.Committer.When),
		InsertedAt:     timestamppb.Now(),
	}, nil

}
