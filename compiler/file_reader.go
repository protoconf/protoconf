package compiler

import (
	"context"
	"fmt"
	"path"
	"strings"

	"path/filepath"
)

// A FileReader controls how load() calls resolve and read other modules.
type FileReader interface {
	// Resolve parses the "name" part of load("name", "symbol") to a path. This
	// is not required to correspond to a true path on the filesystem, but should
	// be "absolute" within the semantics of this FileReader.
	//
	// fromPath will be empty when loading the root module passed to Load().
	Resolve(ctx context.Context, name, fromPath string) (path string, err error)

	// ReadFile reads the content of the file at the given path, which was
	// returned from Resolve().
	ReadFile(ctx context.Context, path string) ([]byte, error)
}

type localFileReader struct {
	root string
}

// LocalFileReader returns a FileReader that resolves and loads files from
// within a given filesystem directory.
func LocalFileReader(root string) FileReader {
	if root == "" {
		panic("LocalFileReader: empty root path")
	}
	return &localFileReader{root}
}

func (r *localFileReader) Resolve(ctx context.Context, name, fromPath string) (string, error) {
	if filepath.Separator != '/' && strings.ContainsRune(name, filepath.Separator) {
		return "", fmt.Errorf("load(%q): invalid character in module name", name)
	}
	canonicalPath := filepath.FromSlash(path.Clean(name))
	if strings.HasPrefix(canonicalPath, string(filepath.Separator)) {
		return strings.TrimPrefix(canonicalPath, string(filepath.Separator)), nil
	}
	return filepath.Join(filepath.Dir(fromPath), canonicalPath), nil
}
