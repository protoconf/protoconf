package testdata

import (
	"bytes"
	"compress/gzip"
	"embed"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

//go:embed bad_proto large small
var TestData embed.FS

func TestDir(name string) string {
	dirPath, err := os.MkdirTemp("", "protoconf-"+name)
	if err != nil {
		slog.Error("error", err)
		os.Exit(1)
	}

	// Collect .fds.gz files to decompress into cache dir after walk
	var fdsFiles []struct {
		label string
		data  []byte
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

		// Collect .fds.gz files for later decompression into .protoconf_cache/
		if strings.HasSuffix(path, ".fds.gz") {
			label := strings.TrimSuffix(filepath.Base(path), ".fds.gz")
			fdsFiles = append(fdsFiles, struct {
				label string
				data  []byte
			}{label: label, data: b})
			// Still write the .fds.gz file to its original location
		}

		return os.WriteFile(fullPath, b, 0644)
	})
	if err != nil {
		slog.Error("init error", "error", err)
		os.Exit(1)
	}

	testDir := filepath.Join(dirPath, name)

	// Decompress .fds.gz files into .protoconf_cache/ directory
	if len(fdsFiles) > 0 {
		cacheDir := filepath.Join(testDir, ".protoconf_cache")
		if err := os.MkdirAll(cacheDir, 0755); err != nil {
			slog.Error("failed to create cache dir", "error", err)
			os.Exit(1)
		}
		for _, f := range fdsFiles {
			gr, err := gzip.NewReader(bytes.NewReader(f.data))
			if err != nil {
				slog.Error("failed to create gzip reader", "label", f.label, "error", err)
				continue
			}
			decompressed, err := io.ReadAll(gr)
			gr.Close()
			if err != nil {
				slog.Error("failed to decompress fds", "label", f.label, "error", err)
				continue
			}
			fdsPath := filepath.Join(cacheDir, f.label+".fds")
			if err := os.WriteFile(fdsPath, decompressed, 0644); err != nil {
				slog.Error("failed to write fds cache", "label", f.label, "error", err)
			}
		}
	}

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
