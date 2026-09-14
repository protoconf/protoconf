package lib

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/hashicorp/go-getter"
	"github.com/protoconf/protoconf/compiler/module/v1"
	"github.com/protoconf/protoconf/utils/testdata"
	"github.com/stretchr/testify/require"
	"golang.org/x/mod/sumdb/dirhash"
)

func TestParseModulePath(t *testing.T) {
	type args struct {
		moduleName string
	}
	tests := []struct {
		name string
		args args
		want *ModulePath
	}{
		{
			name: "starlib",
			args: args{
				moduleName: "re.star",
			},
			want: &ModulePath{
				Filepath: "re.star",
				Ext:      ".star",
			},
		},
		{
			name: "@terraform util.pinc",
			args: args{
				moduleName: "@terraform_repo//terraform/v1/util.pinc",
			},
			want: &ModulePath{
				Repo:     "terraform_repo",
				Filepath: "//terraform/v1/util.pinc",
				Ext:      ".pinc",
			},
		},
		{
			name: "local util.pinc",
			args: args{
				moduleName: "//terraform/v1/util.pinc",
			},
			want: &ModulePath{
				Filepath: "//terraform/v1/util.pinc",
				Ext:      ".pinc",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseModulePath(tt.args.moduleName); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseModulePath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestModuleService_Sync(t *testing.T) {
	testDir := testdata.SmallTestDir()
	tests := []struct {
		name    string
		head    *module.RemoteRepo
		wantErr error
	}{
		{
			name: "empty",
			head: &module.RemoteRepo{
				Url: ".",
			},
		},
		{
			name: "no integrity",
			head: &module.RemoteRepo{
				Url: ".",
				Deps: map[string]*module.RemoteRepo{
					"vizceral_repo": {
						Url:        filepath.Join(testDir, "internal/vizceral.tgz"),
						Pin:        &module.RemoteRepo_Checksum{Checksum: "896b13d56bd1787089ca5767656c7ef1"},
						SourcePath: "src",
					},
				},
			},
		},
		{
			name:    "bad integrity",
			wantErr: ErrorRemoteRepoValidationFailed,
			head: &module.RemoteRepo{
				Url: ".",
				Deps: map[string]*module.RemoteRepo{
					"vizceral_repo": {
						Url:                  filepath.Join(testDir, "internal/vizceral.tgz"),
						Pin:                  &module.RemoteRepo_Checksum{Checksum: "896b13d56bd1787089ca5767656c7ef1"},
						FileDescriptorSetSum: "98173254590e7bfb78555e03033e756e",
						Integrity:            "hello world",
					},
				},
			},
		},
		{
			name: "good integrity",
			head: &module.RemoteRepo{
				Url: ".",
				Deps: map[string]*module.RemoteRepo{
					"vizceral_repo": {
						Url:        filepath.Join(testDir, "internal/vizceral.tgz"),
						Pin:        &module.RemoteRepo_Checksum{Checksum: "896b13d56bd1787089ca5767656c7ef1"},
						Integrity:  "h1:mKU/VAicQpQB3uVzxxAlTZsKPewktnNXQvwmBXI5W9o=",
						SourcePath: "src",
					},
				},
			},
		},
		{
			name: "download deps",
			head: &module.RemoteRepo{
				Url: ".",
				Deps: map[string]*module.RemoteRepo{
					"vizceral_repo": {
						Url:        "github.com/protoconf/protoconf-xds",
						Pin:        &module.RemoteRepo_Commit{Commit: "27d699b"},
						Integrity:  "h1:MHam+LpdxRMBNGdU+WzfqO6HWpMNfS/6ZHEU+nI1KpI=",
						SourcePath: "src",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{})).With("test", t.Name())
			slog.SetDefault(logger)
			m, err := NewModuleService(testdata.SmallTestDir())
			require.NoError(t, err)
			m.head = tt.head
			require.NoError(t, m.Walk(func(r *module.RemoteRepo) error {
				if r.Url == "." {
					return nil
				}
				r.GetterUrl, _ = getter.Detect(r.Url, testDir, getter.Detectors)
				r.Label = repoLabel(r)
				return nil
			}))
			if err := m.Sync(context.Background()); !errors.Is(err, tt.wantErr) {
				t.Errorf("ModuleService.Sync() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestWalkVisitsInSortedKeyOrder is a regression test for walk() sorting the
// `keys` slice without reindexing the parallel `deps` slice, which made
// traversal order actually Go's randomized map-iteration order rather than
// the sorted order the sort.Strings(keys) call implied.
func TestWalkVisitsInSortedKeyOrder(t *testing.T) {
	head := &module.RemoteRepo{
		Url: ".",
		Deps: map[string]*module.RemoteRepo{
			"zebra":   {Url: "zebra"},
			"alpha":   {Url: "alpha"},
			"mike":    {Url: "mike"},
			"bravo":   {Url: "bravo"},
			"charlie": {Url: "charlie"},
			"delta":   {Url: "delta"},
			"foxtrot": {Url: "foxtrot"},
			"golf":    {Url: "golf"},
			"hotel":   {Url: "hotel"},
			"india":   {Url: "india"},
			"juliet":  {Url: "juliet"},
			"kilo":    {Url: "kilo"},
			"lima":    {Url: "lima"},
		},
	}
	// Run several times: map iteration order is randomized per range, so a
	// misaligned keys/deps pairing would eventually surface a wrong order.
	for i := 0; i < 20; i++ {
		var visited []string
		require.NoError(t, walk(head, func(r *module.RemoteRepo) error {
			if r.Url != "." {
				visited = append(visited, r.Url)
			}
			return nil
		}))
		want := make([]string, len(visited))
		copy(want, visited)
		sort.Strings(want)
		require.Equal(t, want, visited, "walk() must visit dependencies in sorted key order")
	}
}

// TestValidatePropagatesHashDirErrors is a regression test for Validate()
// silently discarding any dirhash.HashDir error other than os.ErrNotExist
// whenever r.Integrity=="dummy" -- which stamped Integrity="" as if hashing
// had succeeded instead of surfacing the real failure.
func TestValidatePropagatesHashDirErrors(t *testing.T) {
	root := t.TempDir()
	m, err := NewModuleService(root)
	require.NoError(t, err)

	t.Run("not downloaded yet -> ErrorRemoteRepoNotDownloaded (unchanged)", func(t *testing.T) {
		r := &module.RemoteRepo{Label: "missing", Integrity: "dummy"}
		_, err := m.Validate(r)
		require.ErrorIs(t, err, ErrorRemoteRepoNotDownloaded)
	})

	t.Run("HashDir error other than ErrNotExist propagates instead of being swallowed", func(t *testing.T) {
		// dirhash.HashDir's DirFiles returns a plain "%s is not a directory"
		// error (not wrapping os.ErrNotExist) when the target path exists
		// but is not a directory -- e.g. a file where a directory is
		// expected. This mirrors the class of error hashicorp/go-getter's
		// //subdir extraction path can produce (see debug session evidence).
		notADir := filepath.Join(root, ".protoconf_cache", "broken")
		require.NoError(t, os.MkdirAll(filepath.Dir(notADir), 0755))
		require.NoError(t, os.WriteFile(notADir, []byte("not a directory"), 0644))

		r := &module.RemoteRepo{Label: "broken", Integrity: "dummy"}
		_, err := m.Validate(r)
		require.Error(t, err, "a real HashDir error must not be silently swallowed into a fake success")
		require.ErrorIs(t, err, ErrorRemoteRepoValidationFailed)
		require.NotErrorIs(t, err, ErrorRemoteRepoNotDownloaded)
	})

	t.Run("boundary: real hash mismatch still reported as before", func(t *testing.T) {
		okDir := filepath.Join(root, ".protoconf_cache", "ok")
		require.NoError(t, os.MkdirAll(okDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(okDir, "f.txt"), []byte("content"), 0644))

		r := &module.RemoteRepo{Label: "ok", Integrity: "not-the-real-hash"}
		_, err := m.Validate(r)
		require.ErrorIs(t, err, ErrorRemoteRepoValidationFailed)
	})

	t.Run("boundary: successful hash match still returns nil error", func(t *testing.T) {
		okDir := filepath.Join(root, ".protoconf_cache", "match")
		require.NoError(t, os.MkdirAll(okDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(okDir, "f.txt"), []byte("content"), 0644))
		r := &module.RemoteRepo{Label: "match"}
		h, err := dirhash.HashDir(okDir, "", hash1)
		require.NoError(t, err)
		r.Integrity = h
		gotHash, err := m.Validate(r)
		require.NoError(t, err)
		require.Equal(t, h, gotHash)
	})
}

// TestDownloadDepsSkipsSentinelHead is a regression test for Sync() calling
// DownloadDeps(ctx, r) for every node Walk() visits, including the
// CONFIGSPACE sentinel/root -- which redundantly re-Downloaded/re-Validated
// every top-level dependency a second time within the same Sync() call.
func TestDownloadDepsSkipsSentinelHead(t *testing.T) {
	m, err := NewModuleService(t.TempDir())
	require.NoError(t, err)
	head := &module.RemoteRepo{
		Url: ".",
		Deps: map[string]*module.RemoteRepo{
			// A GetterUrl pointing at a nonexistent scheme: if DownloadDeps
			// actually tried to process this dependency, Download() would
			// attempt a real (failing) getter.GetAny call and return an
			// error. The sentinel guard must return nil before ever
			// reaching this dependency.
			"bogus": {Url: "bogus", GetterUrl: "not-a-real-scheme://nope", Label: "bogus"},
		},
	}
	require.NoError(t, m.DownloadDeps(context.Background(), head))
}
