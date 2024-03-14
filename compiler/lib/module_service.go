package lib

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
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
	"github.com/protoconf/protoconf/compiler/lib/parser"
	"github.com/protoconf/protoconf/compiler/module/v1"
	"github.com/protoconf/protoconf/compiler/starproto"
	"github.com/protoconf/protoconf/consts"
	"github.com/protoconf/protoconf/utils"
	"go.starlark.net/starlark"
	"golang.org/x/mod/sumdb/dirhash"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/reflect/protoregistry"
)

type ModuleService struct {
	Config      *module.ModuleServiceConfig
	head        *module.RemoteRepo
	mutex       sync.RWMutex
	downloadMux *sync.Mutex
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

func (m *ModuleService) GetProtoPaths() []string {
	arr := []string{}
	return m.protoPaths(m.head, arr)
}

func (m *ModuleService) protoPaths(r *module.RemoteRepo, input []string) []string {
	for _, path := range append(r.AdditionalProtoDirs, r.SourcePath) {
		if path != "" {
			input = append(input, filepath.Join(m.getCacheDir(), r.Label, path))
		}
	}
	for _, dep := range r.Deps {
		input = m.protoPaths(dep, input)
	}
	return input

}

func (m *ModuleService) Init(ctx context.Context, initFiles ...string) error {
	os.MkdirAll(m.getCacheDir(), 0755)
	grp, _ := errgroup.WithContext(ctx)
	thread := &starlark.Thread{}
	for _, file := range initFiles {
		filePath := filepath.Join(m.getProtoconfPath(), file)
		grp.Go(func() error {
			b, err := os.ReadFile(filePath)
			if err != nil {
				return errors.Join(fmt.Errorf("failed to read file: %s", filePath), err)
			}
			locals, err := starlark.ExecFile(thread, filePath, b, starlark.StringDict{"remote_repo": starlark.NewBuiltin("remote_repo", m.Add)})
			for name, v := range locals {
				dyn, _ := starproto.ToProtoMessage(v)
				msg := &module.RemoteRepo{}
				originalGetterUrl := ""
				if msgTmp, ok := m.head.Deps[name]; ok {
					originalGetterUrl = msgTmp.GetterUrl
					msg = msgTmp
				}

				// Arrays must be reseted before merge
				msg.AdditionalProtoDirs = []string{}
				msg.ExcludeFileRegexps = []string{}
				dyn.MergeInto(msg)
				if !strings.EqualFold(originalGetterUrl, msg.GetterUrl) {
					msg.Integrity = ""
				}
				m.mutex.Lock()
				m.head.Deps[name] = msg
				m.mutex.Unlock()
			}
			return err
		})
	}
	err := grp.Wait()
	if err != nil {
		return err
	}
	return m.Lock()
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
	if remoteRepo.Label == "" {
		sanitized := strings.TrimPrefix(remoteRepo.GetterUrl, "git::")
		parsedSanitized, _ := url.Parse(sanitized)
		remoteRepo.Label = label.ImportPathToBazelRepoName(
			filepath.Join(
				parsedSanitized.Hostname(),
				strings.TrimSuffix(filepath.Join(strings.Split(parsedSanitized.Path, "/")[:3]...), ".git"),
				parsedSanitized.Query().Get("ref"),
			),
		)
	}

	dynamicMsg, _ := dynamic.AsDynamicMessage(remoteRepo)
	return starproto.NewStarProtoMessage(dynamicMsg), nil
}

func (m *ModuleService) Load(thread *starlark.Thread, moduleName string) (starlark.StringDict, error) {
	metadata := ParseModulePath(moduleName)
	if metadata == nil {
		return nil, errors.New("could not parse module path")
	}
	if metadata.Ext == ".proto" {
		return thread.Load(thread, metadata.Filepath)
	}
	repo, ok := m.head.Deps[metadata.Repo]
	if !ok {
		return nil, errors.New("repo does not exist in workspace")
	}

	moduleRoot := filepath.Join(m.getCacheDir(), repo.Label)
	c := NewCompiler(moduleRoot, false)
	c.parser = parser.NewParser(m.GetProtoFilesRegistry())
	newThread := &starlark.Thread{}
	newThread.Load = c.GetLoader().Load
	return c.GetLoader().Load(newThread, metadata.Filepath)
}

func (m *ModuleService) Lock() error {
	b, err := prototext.MarshalOptions{Multiline: true}.Marshal(m.head)
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
	return prototext.Unmarshal(b, m.head)
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
	err := getter.GetAny(repoCacheDir, r.GetterUrl, getter.WithContext(ctx))
	m.downloadMux.Unlock()
	if err != nil {
		return err
	}
	r.Integrity, err = m.Validate(r)
	return err

}

func (m *ModuleService) GenFileDescriptorSet(r *module.RemoteRepo) error {
	registry := utils.NewDescriptorRegistry()
	files := m.protoPaths(r, []string{})
	excludes := []*regexp.Regexp{}
	for _, str := range r.ExcludeFileRegexps {
		newRe := regexp.MustCompile(str)
		excludes = append(excludes, newRe)
	}

	err := registry.Import(utils.ParseFilesButDoNotLink, excludes, files...)
	if err != nil {
		return err
	}

	h, err := registry.Store(filepath.Join(m.getCacheDir(), r.Label+".fds"))
	r.FileDescriptorSetSum = h
	return err
}

func (m *ModuleService) GetProtoFilesRegistry() *protoregistry.Files {
	return m.GetProtoRegistry().GetFilesResolver()

}

var cachedRegistry *utils.DescriptorRegistry

func (m *ModuleService) GetProtoRegistry() *utils.DescriptorRegistry {
	if cachedRegistry != nil {
		return cachedRegistry
	}
	registry := utils.NewDescriptorRegistry()
	for _, dep := range m.head.Deps {
		registry.Load(filepath.Join(m.getCacheDir(), dep.Label+".fds"), dep.FileDescriptorSetSum)
	}
	registry.Import(utils.Parse, []*regexp.Regexp{}, filepath.Join(m.getProtoconfPath(), consts.SrcPath))
	cachedRegistry = registry
	return registry

}
func (m *ModuleService) Sync(ctx context.Context) error {
	grp, _ := errgroup.WithContext(ctx)
	for _, r := range m.head.Deps {
		remoteRepo := r
		grp.Go(func() error {
			err := m.Download(ctx, remoteRepo)
			if err != nil {
				return err
			}
			return m.GenFileDescriptorSet(remoteRepo)
		})
	}
	err := grp.Wait()
	if err != nil {
		return err
	}
	return m.Lock()
}

var loadMatcherRegex = regexp.MustCompile(`((?P<repo>(@[^//]+|\.)|))(?P<filepath>(//|).*(?P<ext>(.proto|.pinc|.pconf|.mpconf|.star)))`)

type ModulePath struct {
	Repo     string
	Filepath string
	Ext      string
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
