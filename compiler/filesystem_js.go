package compiler

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall/js"
)

func (r *localFileReader) ReadFile(ctx context.Context, path string) ([]byte, error) {
	absPath := filepath.Join(r.root, path)
	contents := js.Global().Call("readFile", absPath)
	if contents == js.Null() {
		return nil, fmt.Errorf("Error reading %s", absPath)
	}
	return []byte(contents.String()), nil
}

func mkdirAll(path string, perm os.FileMode) error {
	return nil
}

func stat(path string) (bool, bool, error) {
	contents := js.Global().Call("readFile", path)
	return contents != js.Null(), false, nil
}

func openFile(path string) (io.ReadCloser, error) {
	contents := js.Global().Call("readFile", path)
	if contents == js.Null() {
		return nil, fmt.Errorf("Error reading %s", path)
	}
	readCloser := ioutil.NopCloser(bytes.NewReader([]byte(contents.String())))
	return readCloser, nil
}

func writeFile(filename string, contents []byte) error {
	written := js.Global().Call("writeFile", filename, string(contents)).Bool()
	if !written {
		return fmt.Errorf("error writing to %s", filename)
	}
	return nil
}
