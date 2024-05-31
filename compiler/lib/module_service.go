package lib

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/hashicorp/go-getter"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/mitchellh/cli"
	"github.com/protoconf/protoconf/command"
	"github.com/protoconf/protoconf/compiler/module/v1"
	"github.com/protoconf/protoconf/compiler/starproto"
	"github.com/protoconf/protoconf/consts"
	"github.com/protoconf/protoconf/utils"
	"go.starlark.net/starlark"
	"golang.org/x/mod/sumdb/dirhash"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoregistry"
)

type ModuleService struct {
	Config         *module.ModuleServiceConfig
	head           *module.RemoteRepo
	mutex          sync.RWMutex
	downloadMux    *sync.Mutex
	cachedRegistry *utils.DescriptorRegistry
}

func NewModuleService(protoconfRoot string) *ModuleService {
	const lockFile = `protoconf.lock`
	const cacheDir = `.protoconf_cache`
	config := &module.ModuleServiceConfig{
		ProtoconfPath: protoconfRoot,
		CacheDir:      cacheDir,
		LockFile:      lockFile,
	}
	return &ModuleService{
		Config:      config,
		mutex:       sync.RWMutex{},
		downloadMux: &sync.Mutex{},
		head: &module.RemoteRepo{
			Url:  ".",
			Deps: map[string]*module.RemoteRepo{},
		},
	}
}

func (m *ModuleService) getProtoconfPath() string {
	path, err := filepath.Abs(m.Config.ProtoconfPath)
	if err != nil {
		log.Fatal(err)
	}
	return path
}

func (m *ModuleService) getCacheDir() string {
	if filepath.IsAbs(m.Config.CacheDir) {
		return m.Config.CacheDir
	}
	return filepath.Join(m.getProtoconfPath(), m.Config.CacheDir)
}

func (m *ModuleService) getLockFile() string {
	if filepath.IsAbs(m.Config.LockFile) {
		return m.Config.LockFile
	}
	return filepath.Join(m.getProtoconfPath(), m.Config.LockFile)
}

func (m *ModuleService) protoPaths(r *module.RemoteRepo, input []string) []string {
	for _, path := range append(r.AdditionalProtoDirs, r.SourcePath) {
		if path != "" {
			input = append(input, filepath.Join(m.getCacheDir(), r.Label, path))
		}
	}
	return input
}

func (m *ModuleService) Init(ctx context.Context, initFiles ...string) error {
	os.MkdirAll(m.getCacheDir(), 0755)
	thread := &starlark.Thread{}
	for _, file := range initFiles {
		filePath := filepath.Join(m.getProtoconfPath(), file)
		b, err := os.ReadFile(filePath)
		if err != nil {
			return errors.Join(fmt.Errorf("failed to read file: %s", filePath), err)
		}
		locals, err := starlark.ExecFile(thread, filePath, b, starlark.StringDict{"remote_repo": starlark.NewBuiltin("remote_repo", m.Add)})
		if err != nil {
			return errors.Join(fmt.Errorf("failed to execute file: %s", filePath), err)
		}
		for name, v := range locals {
			dyn, ok := starproto.ToProtoMessage(v)
			if !ok {
				continue
			}
			msg := &module.RemoteRepo{}
			var originalGetterUrl string
			if msgTmp, ok := m.head.Deps[name]; ok {
				originalGetterUrl = msgTmp.GetterUrl
				msg = msgTmp
			}

			// Arrays must be reseted before merge
			msg.AdditionalProtoDirs = []string{}
			msg.ExcludeFileRegexps = []string{}
			dyn.MergeInto(msg)
			if originalGetterUrl != msg.GetterUrl {
				msg.Integrity = ""
			}
			m.mutex.Lock()
			m.head.Deps[name] = msg
			m.mutex.Unlock()
		}
	}
	return m.MergeLock()
}

func (m *ModuleService) Add(t *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	ui := &cli.BasicUi{
		ErrorWriter: os.Stderr,
		Writer:      os.Stderr,
	}
	d, _ := desc.LoadMessageDescriptorForMessage(&module.RemoteRepo{})
	starMsg, _ := starproto.NewMessageType(d).(starlark.Callable)

	v, err := starMsg.CallInternal(t, args, kwargs)
	if err != nil {
		return nil, err
	}
	msg, _ := starproto.ToProtoMessage(v)
	remoteRepo := &module.RemoteRepo{SourcePath: "src"}
	err = msg.MergeInto(remoteRepo)
	if err != nil {
		return nil, err
	}

	detectedUrl, err := getter.Detect(remoteRepo.Url, m.getProtoconfPath(), getter.Detectors)
	if err != nil {
		return nil, errors.Join(errors.New("not a valid repo url"), err)
	}
	u, err := url.Parse(detectedUrl)
	if err != nil {
		return nil, errors.Join(errors.New("failed to parse repo url"), err)
	}
	query := u.Query()
	if strings.HasPrefix(detectedUrl, "git::") && !query.Has("depth") {
		query.Set("depth", "0")
	}
	switch x := remoteRepo.Pin.(type) {
	case *module.RemoteRepo_Branch:
		query.Set("ref", x.Branch)
	case *module.RemoteRepo_Commit:
		query.Set("ref", x.Commit)
	case *module.RemoteRepo_Tag:
		query.Set("ref", x.Tag)
	case *module.RemoteRepo_Checksum:
		query.Set("checksum", x.Checksum)
	default:
		ui.Error(fmt.Sprintf("Warning! Please provide on of: tag, branch, commit or checksum for remote_repo url: %s", remoteRepo.Url))
	}
	u.RawQuery = query.Encode()

	remoteRepo.GetterUrl = u.String()
	remoteRepo.Label = repoLabel(remoteRepo)
	dynamicMsg, _ := dynamic.AsDynamicMessage(remoteRepo)
	return starproto.NewStarProtoMessage(dynamicMsg), nil
}

func repoLabel(remoteRepo *module.RemoteRepo) string {
	if remoteRepo.Label != "" {
		return remoteRepo.Label
	}
	sanitized := strings.TrimPrefix(remoteRepo.GetterUrl, "git::")
	parsedSanitized, _ := url.Parse(sanitized)
	return label.ImportPathToBazelRepoName(
		filepath.Join(
			parsedSanitized.Hostname(),
			strings.TrimSuffix(filepath.Join(strings.Split(parsedSanitized.Path, "/")[:3]...), ".git"),
			parsedSanitized.Query().Get("ref"),
		),
	)

}

func (m *ModuleService) Lock() error {
	b, err := protojson.MarshalOptions{Multiline: true}.Marshal(m.head)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(m.getLockFile()), b, 0644)
}

func (m *ModuleService) LoadFromLockFile() error {
	b, err := os.ReadFile(filepath.Join(m.getLockFile()))
	if err != nil {
		return nil
	}
	return protojson.Unmarshal(b, m.head)
}

func (m *ModuleService) MergeLock() error {
	err := m.LoadFromLockFile()
	if err != nil {
		return err
	}
	return m.Lock()
}

var ErrorRemoteRepoNoIntegrityInfo = errors.New("missing integrity data")
var ErrorRemoteRepoNotDownloaded = errors.New("remote repo not in local cache")
var ErrorRemoteRepoValidationFailed = errors.New("failed to validate integrity")

func (m *ModuleService) Validate(r *module.RemoteRepo) (string, error) {
	if r.Integrity == "" {
		return "", ErrorRemoteRepoNoIntegrityInfo
	}
	repoCacheDir := filepath.Join(m.getCacheDir(), r.Label)
	h, err := dirhash.HashDir(repoCacheDir, "", hash1)
	if errors.Is(err, os.ErrNotExist) {
		return "", ErrorRemoteRepoNotDownloaded
	}
	if r.Integrity != "dummy" && r.Integrity != h {
		return "", errors.Join(
			ErrorRemoteRepoValidationFailed,
			fmt.Errorf("%s(%s): %s (expected: %s)", r.Label, r.Pin, h, r.Integrity),
		)
	}
	return h, nil
}

func hash1(files []string, open func(string) (io.ReadCloser, error)) (string, error) {
	h := sha256.New()
	files = append([]string(nil), files...)
	sort.Strings(files)
	for _, file := range files {
		if strings.HasPrefix(file, ".git") {
			continue
		}
		if strings.Contains(file, "\n") {
			return "", errors.New("dirhash: filenames with newlines are not supported")
		}
		r, err := open(file)
		if err != nil {
			return "", err
		}
		hf := sha256.New()
		_, err = io.Copy(hf, r)
		r.Close()
		if err != nil {
			return "", err
		}
		fmt.Fprintf(h, "%x  %s\n", hf.Sum(nil), file)
	}
	return "h1:" + base64.StdEncoding.EncodeToString(h.Sum(nil)), nil
}

func (m *ModuleService) Download(ctx context.Context, r *module.RemoteRepo) error {
	if r.GetterUrl == "" {
		return nil
	}
	slog.Debug("downloading repo", "label", r.Label, "integrity", r.Integrity)
	switch _, integrityErr := m.Validate(r); {
	case integrityErr == nil:
		return nil
	case errors.Is(integrityErr, ErrorRemoteRepoValidationFailed):
		return integrityErr
	case errors.Is(integrityErr, ErrorRemoteRepoNoIntegrityInfo):
		r.Integrity = "dummy"
	}
	repoCacheDir := filepath.Join(m.getCacheDir(), r.Label)

	m.downloadMux.Lock()
	command.DefaultUI.Output(fmt.Sprintf("Downloading: %s", r.Url))
	ui := command.NewPrefixedUi(fmt.Sprintf("  => %s: ", r.Label))
	err := getter.GetAny(repoCacheDir, r.GetterUrl, getter.WithContext(ctx), getter.WithProgress(defaultProgressBar))
	m.downloadMux.Unlock()
	if err != nil {
		ui.Error(fmt.Sprintf("Failed: %s", err))
		return err
	}
	ui.Info("Done.")
	r.Integrity, err = m.Validate(r)
	return err
}

func (m *ModuleService) GenFileDescriptorSet(registry *utils.DescriptorRegistry, r *module.RemoteRepo) error {
	cacheFile := filepath.Join(m.getCacheDir(), r.GetLabel()+".fds")
	command.DefaultUI.Output(fmt.Sprintf("Generating file descriptor set for: %s", r.Label))
	ui := command.NewPrefixedUi(fmt.Sprintf("  => %s: ", r.Label))
	err := registry.Load(cacheFile, r.GetFileDescriptorSetSum())
	if err == nil {
		ui.Info("Loaded from cache.")
		return nil
	}
	ui.Warn("Could not load from cache:")
	ui.Warn(err.Error())
	// slog.Info("generating descriptor set", "repo", r.Label, "error", err)
	paths := m.protoPaths(r, []string{})
	excludes := []*regexp.Regexp{}
	for _, str := range r.ExcludeFileRegexps {
		newRe := regexp.MustCompile(str)
		excludes = append(excludes, newRe)
	}

	ui.Info("Parsing protos.")
	err = registry.Import(registry.Parse, excludes, paths...)
	if err != nil {
		ui.Error("Failed to parse proto files:")
		ui.Error(err.Error())
		return errors.Join(fmt.Errorf("failed generate file descriptor set for: %s", r.Label), err)
	}

	ui.Info("Storing in cache.")
	h, err := registry.Store(cacheFile)
	if err != nil {
		ui.Error("Failed to store file descriptor set: " + err.Error())
		return err
	}
	return m.Walk(func(repo *module.RemoteRepo) error {
		if r.Label == repo.Label {
			repo.FileDescriptorSetSum = h
		}
		return m.Lock()
	})
}

func (m *ModuleService) GetProtoFilesRegistry() *protoregistry.Files {
	return m.GetProtoRegistry().GetFilesResolver()

}

func (m *ModuleService) GetProtoRegistry() *utils.DescriptorRegistry {
	if m.cachedRegistry != nil {
		return m.cachedRegistry
	}
	registry := utils.NewDescriptorRegistry()
	for _, dep := range m.head.Deps {
		err := registry.Load(filepath.Join(m.getCacheDir(), dep.Label+".fds"), dep.FileDescriptorSetSum)
		if err != nil {
			slog.Error("failed to load file descriptor set", slog.String("error", err.Error()))
			slog.Error("try run `protoconf mod sync`")
		}
	}
	err := registry.Import(registry.Parse, []*regexp.Regexp{}, filepath.Join(m.getProtoconfPath(), consts.SrcPath))
	if err != nil {
		slog.Error("failed to parse proto files", slog.String("error", err.Error()))

	}
	m.cachedRegistry = registry
	return registry
}

// Sync synchronizes the modules by downloading dependencies and generating file descriptor sets.
// It walks through each remote repository, downloads the repository if necessary, and updates the integrity of matching repositories.
// Then, it downloads the dependencies and generates file descriptor sets for each repository.
// Finally, it returns any error encountered during the process.
func (m *ModuleService) Sync(ctx context.Context) error {
	err := m.Walk(func(r *module.RemoteRepo) error {
		err := m.Download(ctx, r)
		if err != nil {
			return err
		}
		if r.Integrity != "" && r.Integrity != "dummy" {
			m.Walk(func(repo *module.RemoteRepo) error {
				if r.Label == repo.Label {
					repo.Integrity = r.Integrity
				}
				return nil
			})
		}
		defer m.Lock()
		return m.DownloadDeps(ctx, r)
	})
	if err != nil {
		return err
	}
	registry := utils.NewDescriptorRegistry()
	err = m.Walk(func(r *module.RemoteRepo) error {
		if r.Url == "." {
			return nil
		}
		err := m.GenFileDescriptorSet(registry, r)
		return err
	})
	return err
}

// DownloadDeps downloads the dependencies of a remote repository.
// It recursively downloads all the dependencies and their dependencies.
// If a module lock file exists for a dependency, it merges the lock file with the remote repository.
// If the lock file does not exist, it skips the dependency.
// The method returns an error if any error occurs during the download process.
func (m *ModuleService) DownloadDeps(ctx context.Context, r *module.RemoteRepo) error {
	if r.GetterUrl != "" {
		moduleLockFile := filepath.Join(m.getCacheDir(), r.GetLabel(), m.Config.LockFile)
		b, err := os.ReadFile(moduleLockFile)
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, &os.PathError{}) {
		} else {
			if err != nil {
				return err
			}
			new := &module.RemoteRepo{}

			err = protojson.Unmarshal(b, new)
			if err != nil {
				return err
			}
			new.Url = r.Url
			new.GetterUrl = r.GetterUrl
			proto.Merge(r, new)
		}
	}
	for _, dep := range r.GetDeps() {
		err := m.Download(ctx, dep)
		if err != nil {
			return err
		}
		err = m.DownloadDeps(ctx, dep)
		if err != nil {
			return err
		}
	}
	return nil
}

type WalkFunction func(r *module.RemoteRepo) error

func walk(head *module.RemoteRepo, walkFn WalkFunction) error {
	var err error
	keys := []string{}
	deps := []*module.RemoteRepo{}
	for k, dep := range head.GetDeps() {
		keys = append(keys, k)
		deps = append(deps, dep)
	}
	sort.Strings(keys)
	for i := range keys {
		err = errors.Join(walk(deps[i], walkFn))
	}
	return errors.Join(err, walkFn(head))
}

func (m *ModuleService) Walk(walkFn WalkFunction) error {
	return walk(m.head, walkFn)
}

var loadMatcherRegex = regexp.MustCompile(`((?P<repo>(@[^//]+|\.)|))(?P<filepath>(//|).*(?P<ext>(.proto|.pinc|.pconf|.mpconf|.star)))`)

type ModulePath struct {
	Repo     string
	Filepath string
	Ext      string
}

func (m *ModulePath) String() string {
	return fmt.Sprintf("@%s/%s", m.Repo, filepath.Clean(m.Filepath))
}

func ParseModulePath(moduleName string) *ModulePath {
	if loadMatcherRegex.MatchString(moduleName) {
		parts := loadMatcherRegex.FindStringSubmatch(moduleName)
		result := make(map[string]string)
		for i, name := range loadMatcherRegex.SubexpNames() {
			if i != 0 && name != "" {
				result[name] = parts[i]
			}
		}
		return &ModulePath{
			Repo:     strings.TrimPrefix(result["repo"], "@"),
			Filepath: result["filepath"],
			Ext:      result["ext"],
		}
	}
	return nil
}
