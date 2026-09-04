package parser

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/protoconf/protoconf/utils"
	"github.com/protoconf/protoconf/utils/testdata"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

// TestParseMemoization asserts LAZY-02: a second ParseOne request for the
// same path is a map lookup, not a re-parse (pointer identity), the loaded
// count does not double-count a path requested twice, the whole transitive
// dependency closure is memoised too, and concurrent requests for the same
// not-yet-parsed path are collapsed by singleflight into exactly one parse.
func TestParseMemoization(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, testdata.GenerateCorpus(dir, 10))
	src := filepath.Join(dir, "src")

	dr := utils.NewDescriptorRegistry()
	dr.ImportPaths = []string{src}

	fd1, err := dr.ParseOne("pkg7/msg7.proto")
	require.NoError(t, err)
	require.NotNil(t, fd1)

	countAfterFirst := dr.LoadedFileCount()

	fd2, err := dr.ParseOne("pkg7/msg7.proto")
	require.NoError(t, err)
	require.Same(t, fd1, fd2, "second request for the same path must return the identical pointer, not a re-parse")

	countAfterSecond := dr.LoadedFileCount()
	require.Equal(t, countAfterFirst, countAfterSecond, "requesting the same path twice must not double-count LoadedFileCount")
	require.Greater(t, countAfterFirst, 0)

	for _, dep := range fd1.GetDependencies() {
		_, ok := dr.FileDescriptor(dep.GetName())
		require.True(t, ok, "transitive dependency %s should already be memoised", dep.GetName())
	}

	// Concurrency: N goroutines requesting the same not-yet-parsed path on a
	// cold registry must collapse to one parse via singleflight, and every
	// caller must observe the same descriptor pointer.
	dr2 := utils.NewDescriptorRegistry()
	dr2.ImportPaths = []string{src}

	const n = 8
	var mu sync.Mutex
	pointers := make([]interface{}, 0, n)

	g := new(errgroup.Group)
	for i := 0; i < n; i++ {
		g.Go(func() error {
			fd, err := dr2.ParseOne("pkg7/msg7.proto")
			if err != nil {
				return err
			}
			mu.Lock()
			pointers = append(pointers, fd)
			mu.Unlock()
			return nil
		})
	}
	require.NoError(t, g.Wait())
	require.Len(t, pointers, n)
	for _, p := range pointers {
		require.Same(t, pointers[0], p, "concurrent ParseOne calls for the same path must all observe the same parse")
	}
	require.Equal(t, countAfterFirst, dr2.LoadedFileCount(), "concurrent cold load should load the same set of files as the sequential run")
}

// TestLazyParseDoesNotMutateLocalFiles asserts LAZY-03: an on-demand parse
// never changes what Store() serializes. Golden regA is populated with a
// narrow eager Import (pkg0 only); subject regB is populated identically,
// then services a ParseOne request the narrow import never covered. Store()
// output must be byte-identical between the two, proving ParseOne wrote
// only to FileRegistry/lazyLoaded and never touched localFiles.
func TestLazyParseDoesNotMutateLocalFiles(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, testdata.GenerateCorpus(dir, 20))
	src := filepath.Join(dir, "src")

	regA := utils.NewDescriptorRegistry()
	require.NoError(t, regA.Import(regA.Parse, nil, filepath.Join(src, "pkg0")))
	aPath := filepath.Join(t.TempDir(), "a.fds")
	hA, err := regA.Store(aPath)
	require.NoError(t, err)
	bytesA, err := os.ReadFile(aPath)
	require.NoError(t, err)

	regB := utils.NewDescriptorRegistry()
	require.NoError(t, regB.Import(regB.Parse, nil, filepath.Join(src, "pkg0")))
	regB.ImportPaths = []string{src}
	_, err = regB.ParseOne("pkg9/msg9.proto")
	require.NoError(t, err)

	bPath := filepath.Join(t.TempDir(), "b.fds")
	hB, err := regB.Store(bPath)
	require.NoError(t, err)
	bytesB, err := os.ReadFile(bPath)
	require.NoError(t, err)

	require.Equal(t, hA, hB)
	require.Equal(t, bytesA, bytesB)

	// Idempotency: requesting the same path a second time must not change
	// what Store serializes either.
	_, err = regB.ParseOne("pkg9/msg9.proto")
	require.NoError(t, err)
	hB2, err := regB.Store(bPath)
	require.NoError(t, err)
	bytesB2, err := os.ReadFile(bPath)
	require.NoError(t, err)
	require.Equal(t, hB, hB2)
	require.Equal(t, bytesB, bytesB2)
}
