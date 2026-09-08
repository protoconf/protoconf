package parser

import (
	"errors"
	"fmt"
	"os"

	_ "github.com/bufbuild/protovalidate-go"
	_ "github.com/bufbuild/protovalidate-go/legacy"
	_ "github.com/protoconf/protoconf/pb/protoconf/v1"

	"github.com/jhump/protoreflect/desc"
	"github.com/protoconf/protoconf/utils"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
)

// Parser provides a wrapper around jhump/protoreflect/protoparse that will keep a cache of dpd.FileDescriptor
type Parser struct {
	LocalResolver   *protoregistry.Types
	FilesResolver   *protoregistry.Files
	FileDescriptors map[string]*desc.FileDescriptor
	// TypeResolver is LocalResolver's growable superset: it falls through to
	// the registry's MessageRegistry (and, on a second miss, the D-03 eager
	// fallback) for a type parsed after construction. Use this, not
	// LocalResolver, for any resolution that must see lazily-loaded types.
	TypeResolver *RegistryTypeResolver

	registry *utils.DescriptorRegistry
}

func NewParserWithDescriptorRegistry(registry *utils.DescriptorRegistry) *Parser {
	files := registry.GetFilesResolver()
	localResolver := registry.GetTypesResolver(files)
	return &Parser{
		FilesResolver:   files,
		LocalResolver:   localResolver,
		FileDescriptors: registry.FileRegistry,
		TypeResolver:    NewRegistryTypeResolver(registry, localResolver),
		registry:        registry,
	}
}

// RegistryTypeResolver resolves message types first from a fixed,
// construction-time *protoregistry.Types snapshot — byte-identical to every
// eager consumer's existing behavior, since a *protoregistry.Types is never
// written after construction and is documented concurrent-safe to read —
// then falls through to the registry's growable MessageRegistry, populated
// as ParseOne lazily parses files, and finally triggers the D-03 whole-tree
// eager fallback on a second miss. Only top-level messages resolve through
// the MessageRegistry branch, matching GetTypesResolver's existing
// top-level-only registration; nested-type resolution is TYPE-01, Phase 13.
type RegistryTypeResolver struct {
	registry *utils.DescriptorRegistry
	snapshot *protoregistry.Types
}

func NewRegistryTypeResolver(registry *utils.DescriptorRegistry, snapshot *protoregistry.Types) *RegistryTypeResolver {
	return &RegistryTypeResolver{registry: registry, snapshot: snapshot}
}

func (r *RegistryTypeResolver) FindMessageByURL(url string) (protoreflect.MessageType, error) {
	mt, err := r.snapshot.FindMessageByURL(url)
	if err == nil {
		return mt, nil
	}
	if !errors.Is(err, protoregistry.NotFound) {
		return nil, err
	}
	if md, mErr := r.registry.MessageRegistry.FindMessageTypeByUrl(url); mErr == nil && md != nil {
		return dynamicpb.NewMessageType(md.UnwrapMessage()), nil
	}
	// Trigger the D-03 fallback and retry once regardless of ParseAll's own
	// error: a partial whole-tree parse may still have registered the
	// requested type before hitting an unrelated broken file elsewhere.
	_ = r.registry.ParseAll()
	if md, mErr := r.registry.MessageRegistry.FindMessageTypeByUrl(url); mErr == nil && md != nil {
		return dynamicpb.NewMessageType(md.UnwrapMessage()), nil
	}
	return nil, fmt.Errorf("%w: %s", protoregistry.NotFound, url)
}

func (r *RegistryTypeResolver) FindMessageByName(name protoreflect.FullName) (protoreflect.MessageType, error) {
	mt, err := r.snapshot.FindMessageByName(name)
	if err == nil {
		return mt, nil
	}
	if !errors.Is(err, protoregistry.NotFound) {
		return nil, err
	}
	url := "type.googleapis.com/" + string(name)
	if md, mErr := r.registry.MessageRegistry.FindMessageTypeByUrl(url); mErr == nil && md != nil {
		return dynamicpb.NewMessageType(md.UnwrapMessage()), nil
	}
	_ = r.registry.ParseAll()
	if md, mErr := r.registry.MessageRegistry.FindMessageTypeByUrl(url); mErr == nil && md != nil {
		return dynamicpb.NewMessageType(md.UnwrapMessage()), nil
	}
	return nil, fmt.Errorf("%w: %s", protoregistry.NotFound, name)
}

// FindExtensionByName delegates to the snapshot only: GetTypesResolver
// registers messages and enums and never extensions, so there is nothing to
// grow on the MessageRegistry side.
func (r *RegistryTypeResolver) FindExtensionByName(field protoreflect.FullName) (protoreflect.ExtensionType, error) {
	return r.snapshot.FindExtensionByName(field)
}

func (r *RegistryTypeResolver) FindExtensionByNumber(message protoreflect.FullName, field protoreflect.FieldNumber) (protoreflect.ExtensionType, error) {
	return r.snapshot.FindExtensionByNumber(message, field)
}

func (p *Parser) ParseFilesX(filenames ...string) (results []*desc.FileDescriptor, err error) {
	for _, filename := range filenames {
		if fd, ok := p.registry.FileDescriptor(filename); ok {
			results = append(results, fd)
			continue
		}
		// Locked read: once filesResolver grows, this is the only reader in
		// the tree that can observe it mid-write (T-12-01).
		resolvedFd, resolverErr := p.registry.FindFileByPath(filename)
		// ErrNoGrowableResolver means "this registry is eager and has no
		// growable view" (D-03, ImportPaths empty, d.filesResolver
		// permanently nil) — NOT "this file does not exist". On that path
		// p.FilesResolver is the fixed, non-growing snapshot taken at
		// construction time, which a consumer may have hand-extended
		// (server/server.go's six well-known files). Reading it directly
		// here is correct and race-free: nothing can grow an object the
		// registry never armed.
		if errors.Is(resolverErr, utils.ErrNoGrowableResolver) {
			resolvedFd, resolverErr = p.FilesResolver.FindFileByPath(filename)
		}
		if resolverErr != nil {
			parsed, parseErr := p.registry.ParseOne(filename)
			if parseErr != nil {
				return nil, errors.Join(resolverErr, parseErr)
			}
			results = append(results, parsed)
			continue
		}
		// A resolver hit means something registered this path, and on the
		// compiler's lazy path that only ever happens at the same locked
		// insert point (recordFileLocked/registerFileLocked) that also
		// writes FileRegistry — so the canonical entry is there. Return it
		// instead of minting a fresh desc.WrapFile/desc.CreateFileDescriptor
		// wrapper, which would hand back a second Go pointer for one logical
		// file (RSLV-03): every map keyed by descriptor and every
		// require.Same assertion in the tree depends on one pointer per file.
		if canonical, ok := p.registry.FileDescriptor(resolvedFd.Path()); ok {
			results = append(results, canonical)
			continue
		}
		// The file is in the resolver but genuinely absent from
		// FileRegistry — the mutation server's six hand-registered
		// well-known files (server/server.go). Fall back to wrapping the
		// resolver's own descriptor, unchanged from before this phase.
		fd := resolvedFd
		d, err := desc.WrapFile(fd)
		if err != nil {
			f := protodesc.ToFileDescriptorProto(fd)
			deps := []*desc.FileDescriptor{}
			for _, dep := range f.Dependency {
				dd, err := desc.LoadFileDescriptor(dep)
				if err != nil {
					return nil, err
				}
				deps = append(deps, dd)
			}
			d, err = desc.CreateFileDescriptor(f, deps...)
			if err != nil {
				return nil, err
			}

			results = append(results, d)
		} else {
			results = append(results, d)
		}

	}
	return results, nil
}

func (p *Parser) ReadConfig(filename string, msg proto.Message) error {
	configReader, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	return protojson.UnmarshalOptions{Resolver: p.TypeResolver}.Unmarshal(configReader, msg)
}
