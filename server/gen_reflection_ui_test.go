package server

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/protoconf/protoconf/consts"
	"github.com/protoconf/protoconf/utils/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

// brokenNoValueConfig is a valid ProtoconfValue JSON document with a
// protoFile but no "value" key at all, so parser.ReadConfig succeeds and
// GetValue().GetTypeUrl() is the empty string. That empty type URL is what
// drives FindMessageByURL to fail on the post-ReadConfig branch inside
// collectExamples -- the branch that used to abort the whole walk (D-05).
// The two committed bad_json/bad_proto_file fixtures fail earlier, inside
// ReadConfig itself, and would not exercise this branch.
const brokenNoValueConfig = `{
  "protoFile": "test.proto"
}`

// writeMutableConfig writes name (with content) into root's mutable_config/
// directory, creating the directory if the test already removed it.
func writeMutableConfig(t *testing.T, root, name, content string) {
	t.Helper()
	dir := filepath.Join(root, consts.MutableConfigPath)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
}

// emptyMutableConfig removes every file currently in root's mutable_config/
// directory (but keeps the directory itself), so a fixture starts from a
// clean slate before a test writes its own files into it.
func emptyMutableConfig(t *testing.T, root string) {
	t.Helper()
	dir := filepath.Join(root, consts.MutableConfigPath)
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	for _, e := range entries {
		require.NoError(t, os.Remove(filepath.Join(dir, e.Name())))
	}
}

// newInitializedServer builds a ProtoconfMutationServer against root and
// runs Init, which populates exampleMaker -- collectExamples only appends an
// example when the resolved message name has an entry there.
func newInitializedServer(t *testing.T, root string) *ProtoconfMutationServer {
	t.Helper()
	s, err := NewProtoconfMutationServer(root)
	require.NoError(t, err)
	s.Init(grpc.NewServer())
	return s
}

func TestCollectExamplesContinuesPastUnresolvableConfig(t *testing.T) {
	root := testdata.SmallTestDir()
	// "aaa_broken" sorts lexically before every committed fixture, so before
	// the D-05 fix the walk would abort here and drop mutation_test's example.
	writeMutableConfig(t, root, "aaa_broken.materialized_JSON", brokenNoValueConfig)
	s := newInitializedServer(t, root)

	examples, err := s.collectExamples()
	require.Error(t, err)

	var names []string
	for _, ex := range examples {
		names = append(names, ex.Name)
	}
	assert.Contains(t, names, "mutation_test")
}

func TestCollectExamplesAggregateNamesEveryFailure(t *testing.T) {
	root := testdata.SmallTestDir()
	writeMutableConfig(t, root, "aaa_broken.materialized_JSON", brokenNoValueConfig)
	s := newInitializedServer(t, root)

	_, err := s.collectExamples()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "aaa_broken")
	assert.Contains(t, err.Error(), "bad_json")
	assert.Contains(t, err.Error(), "bad_proto_file")
}

func TestGenReflectionUIEmptyMutableConfig(t *testing.T) {
	t.Run("empty directory", func(t *testing.T) {
		root := testdata.SmallTestDir()
		emptyMutableConfig(t, root)

		s := newInitializedServer(t, root)
		examples, err := s.collectExamples()
		require.NoError(t, err)
		require.Empty(t, examples)
	})

	t.Run("absent directory", func(t *testing.T) {
		root := testdata.SmallTestDir()
		require.NoError(t, os.RemoveAll(filepath.Join(root, consts.MutableConfigPath)))

		s := newInitializedServer(t, root)
		examples, err := s.collectExamples()
		require.NoError(t, err)
		require.Empty(t, examples)
	})
}

func TestGenReflectionUISingleUnresolvableConfig(t *testing.T) {
	root := testdata.SmallTestDir()
	emptyMutableConfig(t, root)
	writeMutableConfig(t, root, "only_broken.materialized_JSON", brokenNoValueConfig)

	s := newInitializedServer(t, root)
	examples, err := s.collectExamples()
	require.Error(t, err)
	require.Empty(t, examples)
	assert.Contains(t, err.Error(), "only_broken")
}

func TestReflectionFailureSetIsOrderIndependent(t *testing.T) {
	first := []reflectionFailure{
		{path: "one", typeURL: "type.googleapis.com/A", err: errors.New("boom")},
		{path: "two", typeURL: "type.googleapis.com/B", err: errors.New("bang")},
	}
	reordered := []reflectionFailure{first[1], first[0]}

	assert.Equal(t, reflectionFailureFingerprint(first), reflectionFailureFingerprint(reordered))
}

func TestReflectionFailureReasonChangeRelogs(t *testing.T) {
	before := []reflectionFailure{
		{path: "one", typeURL: "type.googleapis.com/A", err: errors.New("boom")},
	}
	after := []reflectionFailure{
		{path: "one", typeURL: "type.googleapis.com/A", err: errors.New("a different reason")},
	}

	assert.NotEqual(t, reflectionFailureFingerprint(before), reflectionFailureFingerprint(after))
}

func TestGenReflectionUIConcurrentCallsAreRaceFree(t *testing.T) {
	root := testdata.SmallTestDir()
	s := newInitializedServer(t, root)
	ctx := context.Background()

	var wg sync.WaitGroup
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			// A fixture with broken configs returns a non-nil error by
			// design (SmallTestDir's bad_json/bad_proto_file) -- the point
			// of this test is that both calls complete and the shared
			// change-detection state (reflectionMu, lastReflectionFingerprint)
			// is race-free, not that either call succeeds.
			_ = s.GenReflectionUI(ctx, grpc.NewServer(), &http.Server{})
		}()
	}
	wg.Wait()
}
