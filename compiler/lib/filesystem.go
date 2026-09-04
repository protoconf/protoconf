//go:build !js
// +build !js

package lib

import (
	"io"
	"os"
)

func mkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func openFile(path string) (io.ReadCloser, error) {
	return os.Open(path)
}

func writeFile(filename string, bytes []byte) error {
	return os.WriteFile(filename, bytes, 0644)
}
