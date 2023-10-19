package testdata

import (
	"embed"
	"io/fs"
	"log"
	"os"
	"path/filepath"
)

//go:embed bad_proto large small
var TestData embed.FS

func TestDir(name string) string {
	dirPath, err := os.MkdirTemp("", "protoconf-"+name)
	if err != nil {
		log.Fatal(err)
	}

	err = fs.WalkDir(TestData, name, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		fullPath := filepath.Join(dirPath, path)

		if d.IsDir() {
			return os.MkdirAll(fullPath, 0755)
		}
		b, err := TestData.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(fullPath, b, 0644)
	})
	if err != nil {
		log.Fatalf("init error: %v", err)
	}
	return filepath.Join(dirPath, name)
}

func SmallTestDir() string {
	return TestDir("small")
}

func LargeTestDir() string {
	return TestDir("large")
}

func BadProtoTestDir() string {
	return TestDir("bad_proto")
}
