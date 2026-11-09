package inserter

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/protoconf/protoconf/agent/dummykv"
	"github.com/protoconf/protoconf/utils/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

const matSuffix = ".materialized_JSON"

func testSig(email string, step int) *object.Signature {
	return &object.Signature{Name: "Test", Email: email, When: time.Unix(int64(1700000000+step), 0)}
}

func writeTestFile(tb testing.TB, root, rel, content string) {
	tb.Helper()
	p := filepath.Join(root, rel)
	require.NoError(tb, os.MkdirAll(filepath.Dir(p), 0o755))
	require.NoError(tb, os.WriteFile(p, []byte(content), 0o644))
}

// commitPaths adds paths (relative to the worktree root) and commits them with
// fixed signatures so hashes are deterministic.
func commitPaths(tb testing.TB, w *git.Worktree, step int, parents []plumbing.Hash, paths ...string) plumbing.Hash {
	tb.Helper()
	for _, p := range paths {
		_, err := w.Add(p)
		require.NoError(tb, err)
	}
	h, err := w.Commit(fmt.Sprintf("commit %d", step), &git.CommitOptions{
		Author:    testSig("test-author@example.com", step),
		Committer: testSig("test-committer@example.com", step),
		Parents:   parents,
	})
	require.NoError(tb, err)
	return h
}

func testMessageJSON(v string) string {
	return fmt.Sprintf(`{"protoFile":"test.proto","value":{"@type":"type.googleapis.com/test.v1.TestMessage","stringValue":%q}}`, v)
}

// buildManyConfigsRepo makes a protoconf root (git root == protoconf root) with
// `commits` commits of growing noise and `files` configs touched round-robin.
// The last name is test.materialized_JSON, which only the root commit touches.
func buildManyConfigsRepo(tb testing.TB, files, commits int) (string, []string) {
	tb.Helper()
	root := testdata.SmallTestDir()
	repo, err := git.PlainOpen(root)
	require.NoError(tb, err)
	w, err := repo.Worktree()
	require.NoError(tb, err)
	var names []string
	for n := 0; n < files; n++ {
		names = append(names, fmt.Sprintf("many/cfg_%03d%s", n, matSuffix))
	}
	names = append(names, "test"+matSuffix)
	for c := 0; c < commits; c++ {
		noise := fmt.Sprintf("noise/%d.txt", c)
		writeTestFile(tb, root, noise, fmt.Sprintf("noise %d", c))
		paths := []string{noise}
		if c%3 == 0 {
			cfg := filepath.Join("materialized_config", fmt.Sprintf("many/cfg_%03d%s", (c/3)%files, matSuffix))
			writeTestFile(tb, root, cfg, testMessageJSON(fmt.Sprintf("v%d", c)))
			paths = append(paths, cfg)
		}
		commitPaths(tb, w, c, nil, paths...)
	}
	return root, names
}

func scenarioPath(name string) string {
	return filepath.Join("cfg", "materialized_config", name+matSuffix)
}

// buildScenarioRepo builds a repo with a merge, a rename, dirty, staged,
// untracked and missing files. Files live under cfg/materialized_config/.
func buildScenarioRepo(tb testing.TB) (string, map[string]plumbing.Hash, []string) {
	tb.Helper()
	root := tb.TempDir()
	repo, err := git.PlainInit(root, false)
	require.NoError(tb, err)
	w, err := repo.Worktree()
	require.NoError(tb, err)
	put := func(name, content string) {
		writeTestFile(tb, root, scenarioPath(name), content)
	}
	commits := map[string]plumbing.Hash{}

	for _, n := range []string{"a", "b", "both", "old"} {
		put(n, testMessageJSON(n+"0"))
	}
	commits["c0"] = commitPaths(tb, w, 0, nil, scenarioPath("a"), scenarioPath("b"), scenarioPath("both"), scenarioPath("old"))
	head, err := repo.Head()
	require.NoError(tb, err)
	mainBranch := head.Name()

	require.NoError(tb, w.Checkout(&git.CheckoutOptions{Branch: plumbing.NewBranchReferenceName("side"), Create: true}))
	put("s", testMessageJSON("s1"))
	commits["s1"] = commitPaths(tb, w, 1, nil, scenarioPath("s"))
	put("both", testMessageJSON("both-side"))
	commits["s2"] = commitPaths(tb, w, 2, nil, scenarioPath("both"))

	require.NoError(tb, w.Checkout(&git.CheckoutOptions{Branch: mainBranch}))
	put("b", testMessageJSON("b-main"))
	commits["m1"] = commitPaths(tb, w, 3, nil, scenarioPath("b"))

	put("s", testMessageJSON("s1"))
	put("both", testMessageJSON("both-side"))
	commits["M"] = commitPaths(tb, w, 4, []plumbing.Hash{commits["m1"], commits["s2"]}, scenarioPath("s"), scenarioPath("both"))

	_, err = w.Move(scenarioPath("old"), scenarioPath("new"))
	require.NoError(tb, err)
	commits["r"] = commitPaths(tb, w, 5, nil)

	put("dirty", testMessageJSON("dirty0"))
	commits["d"] = commitPaths(tb, w, 6, nil, scenarioPath("dirty"))

	put("dirty", testMessageJSON("dirty-uncommitted"))
	put("staged", testMessageJSON("staged"))
	_, err = w.Add(scenarioPath("staged"))
	require.NoError(tb, err)
	put("untracked", testMessageJSON("untracked"))

	names := []string{"a", "b", "both", "s", "old", "new", "dirty", "staged", "untracked", "missing"}
	for i := range names {
		names[i] += matSuffix
	}
	return root, commits, names
}

// buildShallowRepo simulates a shallow clone: commit 5 is the shallow cut and
// its parent object is gone, so the walker fails with ErrObjectNotFound.
func buildShallowRepo(tb testing.TB) (string, []plumbing.Hash, []string) {
	tb.Helper()
	root := tb.TempDir()
	repo, err := git.PlainInit(root, false)
	require.NoError(tb, err)
	w, err := repo.Worktree()
	require.NoError(tb, err)
	var commits []plumbing.Hash
	var names []string
	for k := 0; k < 8; k++ {
		name := fmt.Sprintf("cfg_%d", k)
		writeTestFile(tb, root, scenarioPath(name), testMessageJSON(name))
		commits = append(commits, commitPaths(tb, w, k, nil, scenarioPath(name)))
		names = append(names, name+matSuffix)
	}
	names = append(names, "missing"+matSuffix)
	require.NoError(tb, repo.Storer.SetShallow([]plumbing.Hash{commits[5]}))
	require.NoError(tb, repo.Storer.(storer.LooseObjectStorer).DeleteLooseObject(commits[4]))
	return root, commits, names
}

// newGitOnlyInserter opens a fresh repo instance so object caches are never shared.
func newGitOnlyInserter(tb testing.TB, gitRoot, rel string) *ProtoconfInserter {
	tb.Helper()
	repo, err := git.PlainOpen(gitRoot)
	require.NoError(tb, err)
	return &ProtoconfInserter{repo: repo, isGit: true, rel: rel, logger: slog.Default()}
}

// TestGatherMetadataScenarios pins what the per-file go-git walk returns today.
// The cases include go-git quirks (iteration-order diff, not real parents).
func TestGatherMetadataScenarios(t *testing.T) {
	root, commits, _ := buildScenarioRepo(t)
	i := newGitOnlyInserter(t, root, "cfg")
	get := func(name string) (string, error) {
		m, err := i.GatherMetadata(filepath.Join("cfg", "materialized_config", name+matSuffix))
		if err != nil {
			return "", err
		}
		return m.Commit, nil
	}

	want := map[string]string{
		"dirty": "d", "new": "r", "old": "r",
		"s": "M", "both": "M", "b": "m1", "a": "s1",
	}
	for name, c := range want {
		got, err := get(name)
		require.NoError(t, err, name)
		assert.Equal(t, commits[c].String(), got, "file %s want commit %s", name, c)
	}
	for _, name := range []string{"staged", "untracked", "missing"} {
		_, err := get(name)
		require.Error(t, err, name)
		assert.True(t, errors.Is(err, io.EOF), name)
		assert.Contains(t, err.Error(), "error getting next git log item for file")
	}

	t.Run("shallow", func(t *testing.T) {
		root, commits, _ := buildShallowRepo(t)
		i := newGitOnlyInserter(t, root, "cfg")
		for k := 6; k <= 7; k++ {
			m, err := i.GatherMetadata(scenarioPath(fmt.Sprintf("cfg_%d", k)))
			require.NoError(t, err, k)
			assert.Equal(t, commits[k].String(), m.Commit)
		}
		for _, name := range []string{"cfg_0", "cfg_1", "cfg_2", "cfg_3", "cfg_4", "cfg_5", "missing"} {
			_, err := i.GatherMetadata(scenarioPath(name))
			require.Error(t, err, name)
			assert.True(t, errors.Is(err, plumbing.ErrObjectNotFound), "%s: %v", name, err)
		}
	})

	t.Run("empty repo", func(t *testing.T) {
		dir := t.TempDir()
		_, err := git.PlainInit(dir, false)
		require.NoError(t, err)
		_, err = newGitOnlyInserter(t, dir, "").GatherMetadata("x")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error getting git log for file")
	})

	t.Run("non git", func(t *testing.T) {
		m, err := (&ProtoconfInserter{isGit: false}).GatherMetadata("x")
		require.NoError(t, err)
		assert.Equal(t, "not_a_git_repo", m.Commit)
	})
}

func quietLogger(tb testing.TB) {
	tb.Helper()
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	tb.Cleanup(func() { slog.SetDefault(old) })
}

func newBenchInserter(b *testing.B, files, commits int) (*ProtoconfInserter, []string) {
	b.Helper()
	quietLogger(b)
	root, names := buildManyConfigsRepo(b, files, commits)
	kv, err := dummykv.New(context.Background(), []string{}, &dummykv.Config{})
	require.NoError(b, err)
	i := NewProtoconfInserter(root, kv)
	require.NotNil(b, i)
	for _, n := range names {
		require.NoError(b, i.InsertConfigFile(n)) // keep error paths out of the timed loop
	}
	return i, names
}

func BenchmarkInsertManyConfigs(b *testing.B) {
	i, names := newBenchInserter(b, 100, 300)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		i.insertConfigFiles(names)
	}
}

// assertSameMetadata checks that the prefetched inserter returns what the
// per-file walk returns for every name: same error text and sentinels, or the
// same metadata apart from InsertedAt.
func assertSameMetadata(t *testing.T, prefetched, perFile *ProtoconfInserter, names []string) {
	t.Helper()
	prefetched.prefetchLastCommits(names)
	for _, name := range names {
		path := filepath.Join(prefetched.rel, ensureConfigMaterialized(name))
		got, gotErr := prefetched.GatherMetadata(path)
		want, wantErr := perFile.GatherMetadata(path)
		if wantErr != nil {
			require.Error(t, gotErr, name)
			assert.Equal(t, wantErr.Error(), gotErr.Error(), name)
			for _, sentinel := range []error{io.EOF, plumbing.ErrObjectNotFound, plumbing.ErrReferenceNotFound} {
				assert.Equal(t, errors.Is(wantErr, sentinel), errors.Is(gotErr, sentinel), "%s: %v", name, sentinel)
			}
			continue
		}
		require.NoError(t, gotErr, name)
		// The non-git result has no InsertedAt; both sides must agree.
		require.Equal(t, want.InsertedAt == nil, got.InsertedAt == nil, name)
		if want.Commit != "not_a_git_repo" {
			require.NotNil(t, got.InsertedAt, name)
		}
		got.InsertedAt, want.InsertedAt = nil, nil
		assert.True(t, proto.Equal(want, got), "%s: want %v got %v", name, want, got)
	}
}

func TestPrefetchMatchesPerFileWalk(t *testing.T) {
	t.Run("scenario", func(t *testing.T) {
		root, _, names := buildScenarioRepo(t)
		pre := newGitOnlyInserter(t, root, "cfg")
		assertSameMetadata(t, pre, newGitOnlyInserter(t, root, "cfg"), names)
		assert.Len(t, pre.lastCommits, len(names))
		assert.Contains(t, pre.lastCommits, "cfg/materialized_config/a.materialized_JSON")
	})
	t.Run("shallow", func(t *testing.T) {
		root, _, names := buildShallowRepo(t)
		assertSameMetadata(t, newGitOnlyInserter(t, root, "cfg"), newGitOnlyInserter(t, root, "cfg"), names)
	})
	t.Run("empty repo", func(t *testing.T) {
		dir := t.TempDir()
		_, err := git.PlainInit(dir, false)
		require.NoError(t, err)
		pre := newGitOnlyInserter(t, dir, "")
		assertSameMetadata(t, pre, newGitOnlyInserter(t, dir, ""), []string{"x" + matSuffix})
		assert.Empty(t, pre.lastCommits)
	})
	t.Run("non git", func(t *testing.T) {
		assertSameMetadata(t, &ProtoconfInserter{}, &ProtoconfInserter{}, []string{"x" + matSuffix})
	})
	t.Run("many configs", func(t *testing.T) {
		root, names := buildManyConfigsRepo(t, 20, 60)
		assertSameMetadata(t, newGitOnlyInserter(t, root, ""), newGitOnlyInserter(t, root, ""), names)
	})
}

// The prefetched map must not outlive one insert call: library callers commit
// between inserts on the same inserter.
func TestInsertConfigFilesClearsPrefetch(t *testing.T) {
	quietLogger(t)
	kv, err := dummykv.New(context.Background(), []string{}, &dummykv.Config{})
	require.NoError(t, err)
	i := NewProtoconfInserter(testdata.SmallTestDir(), kv)
	require.NotNil(t, i)
	i.insertConfigFiles([]string{"test" + matSuffix})
	assert.Nil(t, i.lastCommits)
}
