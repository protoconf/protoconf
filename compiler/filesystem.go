// +build !js

package compiler

import (
	"context"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

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
