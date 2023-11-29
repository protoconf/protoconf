package lib

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/hashicorp/go-getter"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/protoconf/protoconf/compiler/module/v1"
	"github.com/protoconf/protoconf/compiler/starproto"
	"github.com/protoconf/protoconf/consts"
	"go.starlark.net/starlark"
	"golang.org/x/mod/sumdb/dirhash"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/encoding/prototext"
)

const lockFile = `protoconf.lock`
const cacheDir = `.protoconf_cache`

type ModuleService struct {
	protoconfRoot string
	head          *module.RemoteRepo
	groupLock     *sync.Map
	mutex         sync.RWMutex
}

func NewModuleService(protoconfRoot string) *ModuleService {
	return &ModuleService{
		protoconfRoot: protoconfRoot,
		mutex:         sync.RWMutex{},
		groupLock:     &sync.Map{},
		head: &module.RemoteRepo{
			Deps: map[string]*module.RemoteRepo{},
		}}
}

func (m *ModuleService) GetProtoPaths() []string {
	arr := []string{}
	return m.protoPaths(m.head, arr)
}

func (m *ModuleService) protoPaths(r *module.RemoteRepo, input []string) []string {
	for _, path := range append(r.AdditionalProtoDirs, r.SourcePath) {
		if path != "" {
			input = append(input, filepath.Join(m.protoconfRoot, cacheDir, r.Name, path))
		}
	}
	for _, dep := range r.Deps {
		input = m.protoPaths(dep, input)
	}
	return input

}

func (m *ModuleService) Init(ctx context.Context, initFiles ...string) error {
	os.MkdirAll(filepath.Join(m.protoconfRoot, cacheDir), 0755)
	grp, _ := errgroup.WithContext(ctx)
	thread := &starlark.Thread{}
	for _, file := range initFiles {
		filePath := filepath.Join(m.protoconfRoot, consts.SrcPath, file)
		grp.Go(func() error {
			b, err := os.ReadFile(filePath)
			if err != nil {
				return errors.Join(fmt.Errorf("failed to read file: %s", filePath), err)
			}
			locals, err := starlark.ExecFile(thread, filePath, b, starlark.StringDict{"remote_repo": starlark.NewBuiltin("remote_repo", m.Add)})
			for name, v := range locals {
				dyn, _ := starproto.ToProtoMessage(v)
				msg := &module.RemoteRepo{}
				dyn.ConvertTo(msg)
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

	detectedUrl, err := getter.Detect(remoteRepo.Url, ".", getter.Detectors)
	if err != nil {
		return nil, errors.Join(errors.New("not a valid repo url"), err)
	}
	u, err := url.Parse(detectedUrl)
	if err != nil {
		return nil, errors.Join(errors.New("failed to parse repo url"), err)
	}
	query := u.Query()
	switch x := remoteRepo.Pin.(type) {
	case *module.RemoteRepo_Branch:
		query.Set("ref", x.Branch)
	case *module.RemoteRepo_Commit:
		query.Set("ref", x.Commit)
	case *module.RemoteRepo_Tag:
		query.Set("ref", x.Tag)
	case *module.RemoteRepo_Checksum:
		query.Set("checksum", x.Checksum)
	}
	u.RawQuery = query.Encode()

	remoteRepo.GetterUrl = u.String()
	if remoteRepo.Name == "" {
		sanitized := strings.TrimPrefix(remoteRepo.GetterUrl, "git::")
		parsedSanitized, _ := url.Parse(sanitized)
		remoteRepo.Name = label.ImportPathToBazelRepoName(filepath.Join(parsedSanitized.Hostname(), strings.TrimSuffix(filepath.Join(strings.Split(parsedSanitized.Path, "/")[:3]...), ".git")))
	}
	m.head.Deps[remoteRepo.Name] = remoteRepo

	dynamicMsg, _ := dynamic.AsDynamicMessage(remoteRepo)
	return starproto.NewStarProtoMessage(dynamicMsg), nil
}

func (m *ModuleService) Load(thread *starlark.Thread, moduleName string) (starlark.StringDict, error) {
	metadata := ParseModulePath(moduleName)

	moduleRoot := filepath.Join(m.protoconfRoot, cacheDir, m.head.Deps[metadata.Repo].Name)
	c := NewCompiler(moduleRoot, false)
	return c.GetLoader().Load(thread, metadata.Filepath)
}

func (m *ModuleService) Lock() error {
	b, err := prototext.MarshalOptions{Multiline: true}.Marshal(m.head)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(m.protoconfRoot, lockFile), b, 0644)
}

func (m *ModuleService) LoadFromLockFile() error {
	b, err := os.ReadFile(filepath.Join(m.protoconfRoot, lockFile))
	if err != nil {
		return nil
	}
	return prototext.Unmarshal(b, m.head)
}

func (m *ModuleService) getLocker(label string) *sync.Mutex {
	item, _ := m.groupLock.LoadOrStore(label, &sync.Mutex{})
	locker, _ := item.(*sync.Mutex)
	return locker
}
func (m *ModuleService) Download(ctx context.Context, r *module.RemoteRepo) error {
	repoCacheDir := filepath.Join(m.protoconfRoot, cacheDir, r.Name)
	label := r.Name

	if r, ok := m.head.Deps[label]; ok {
		h, err := dirhash.HashDir(repoCacheDir, "", dirhash.DefaultHash)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			log.Println(err)
		}
		if r.Integrity != "" && h == r.Integrity {
			return nil
		}
	}

	locker := m.getLocker(label)
	locker.Lock()
	err := getter.GetAny(repoCacheDir, r.GetterUrl, getter.WithContext(ctx), getter.WithProgress(m))
	locker.Unlock()
	if err != nil {
		return err
	}
	h, err := dirhash.HashDir(repoCacheDir, "", dirhash.DefaultHash)
	if err != nil {
		return err
	}
	if r.Integrity == "" {
		r.Integrity = h
	}
	if r.Integrity == h {
		return nil
	}
	return fmt.Errorf("bad checksum for module %s, got: %s, expected: %s", r.Name, h, r.Integrity)

}

func (m *ModuleService) Sync(ctx context.Context) error {
	grp, _ := errgroup.WithContext(ctx)
	for _, r := range m.head.Deps {
		remoteRepo := r
		grp.Go(func() error {
			return m.Download(ctx, remoteRepo)
		})
	}
	err := grp.Wait()
	if err != nil {
		return err
	}
	return m.Lock()
}

func (ModuleService) TrackProgress(src string, currentSize, totalSize int64, stream io.ReadCloser) io.ReadCloser {
	log.Println(src, currentSize, totalSize)
	return stream
}

var loadMatcherRegex = regexp.MustCompile(`(?P<repo>@([^//]+|\.)|)(?P<filepath>(//|).*(?P<ext>(.proto|.pinc|.pconf|.mpconf|.star)))`)

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
