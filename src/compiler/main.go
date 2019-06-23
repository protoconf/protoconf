// +build !js

package main

import (
	"context"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) < 3 {
		log.Fatalf("Usage: %s protoconf_root config_to_compile...", os.Args[0])
	}

	protoconfRoot := strings.TrimSpace(os.Args[1])

	for i := 2; i < len(os.Args); i++ {
		filename := strings.TrimSpace(os.Args[i])
		if err := compileFile(filename, protoconfRoot); err != nil {
			log.Fatalf("Error compiling config %s, err: %s", filename, err)
		}
	}
}

func (r *localFileReader) ReadFile(ctx context.Context, path string) ([]byte, error) {
	return ioutil.ReadFile(filepath.Join(r.root, path))
}

func mkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func stat(path string) (bool, bool, error) {
	pathStat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, false, nil
		}
		return false, false, err
	}
	return true, pathStat.IsDir(), nil
}

func openFile(path string) (io.ReadCloser, error) {
	return os.Open(path)
}

func writeFile(filename string, bytes []byte) error {
	return ioutil.WriteFile(filename, bytes, 0644)
}
