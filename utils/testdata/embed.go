package testdata

import (
	"embed"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
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
	testDir := filepath.Join(dirPath, name)
	repo, _ := git.PlainInit(testDir, false)
	w, _ := repo.Worktree()
	w.AddGlob(".")
	w.Commit("dummy commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "John Doe",
			Email: "john@doe.org",
			When:  time.Now(),
		},
	})
	return testDir
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
