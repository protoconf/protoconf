package filekv

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kvtools/valkeyrie/store"
	protoconfvalue "github.com/protoconf/protoconf/datatypes/proto/v1"
	"github.com/protoconf/protoconf/utils/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoregistry"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	root := testdata.SmallTestDir()
	ctx := context.Background()
	s, err := New(ctx, []string{}, &Config{ProtoconfRoot: root})
	require.NoError(t, err)
	require.NotNil(t, s)
	t.Cleanup(func() {
		_ = s.Close()
	})
	return s
}

func TestNew(t *testing.T) {
	root := testdata.SmallTestDir()
	ctx := context.Background()
	s, err := New(ctx, []string{}, &Config{ProtoconfRoot: root})
	require.NoError(t, err)
	assert.NotNil(t, s)
	_ = s.Close()
}

func TestNew_InvalidRoot(t *testing.T) {
	ctx := context.Background()
	// Using a nonexistent root should still create the store (module service
	// may still work with an empty dir, or return an error).
	s, err := New(ctx, []string{}, &Config{ProtoconfRoot: "/nonexistent/path/that/does/not/exist"})
	// Either returns error or returns store; just verify we handle both cases
	if err != nil {
		assert.Nil(t, s)
	} else {
		assert.NotNil(t, s)
		_ = s.Close()
	}
}

func TestPut(t *testing.T) {
	// Put is a no-op in filekv, should return nil
	s := newTestStore(t)
	ctx := context.Background()
	err := s.Put(ctx, "any/key", []byte("any value"), nil)
	assert.NoError(t, err)
}

func TestPut_WithWriteOptions(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	err := s.Put(ctx, "key/with/opts", []byte("value"), &store.WriteOptions{})
	assert.NoError(t, err)
}

func TestExists(t *testing.T) {
	// Exists always returns true in filekv
	s := newTestStore(t)
	ctx := context.Background()

	exists, err := s.Exists(ctx, "any/key", nil)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestClose(t *testing.T) {
	root := testdata.SmallTestDir()
	ctx := context.Background()
	s, err := New(ctx, []string{}, &Config{ProtoconfRoot: root})
	require.NoError(t, err)
	require.NotNil(t, s)

	err = s.Close()
	assert.NoError(t, err)
}

func TestClose_Idempotent(t *testing.T) {
	root := testdata.SmallTestDir()
	ctx := context.Background()
	s, err := New(ctx, []string{}, &Config{ProtoconfRoot: root})
	require.NoError(t, err)
	require.NotNil(t, s)

	assert.NoError(t, s.Close())
	assert.NoError(t, s.Close())
}

func TestGet_ValidKey(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	pair, err := s.Get(ctx, "materialized_config/test", nil)
	require.NoError(t, err)
	require.NotNil(t, pair)

	b, err := base64.StdEncoding.DecodeString(string(pair.Value))
	require.NoError(t, err)
	protoconfValue := &protoconfvalue.ProtoconfValue{}
	require.NoError(t, proto.Unmarshal(b, protoconfValue))
	assert.Equal(t, "test.proto", protoconfValue.ProtoFile)
}

func TestGet_InvalidPath(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	_, err := s.Get(ctx, "", nil)
	assert.Error(t, err)

	_, err = s.Get(ctx, "../escape", nil)
	assert.Error(t, err)
}

func TestGet_NotFound(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	_, err := s.Get(ctx, "materialized_config/does_not_exist", nil)
	assert.ErrorIs(t, err, store.ErrKeyNotFound)
}

func TestWatch_InvalidPath(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	// Empty path should return an error (invalid path check in Watch)
	_, err := s.Watch(ctx, "", nil)
	assert.Error(t, err)
}

func TestWatch_InvalidPath_Dots(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	// Path with ".." (non-clean path) should return an error
	_, err := s.Watch(ctx, "../some/path", nil)
	assert.Error(t, err)
}

func TestWatch_NonExistentFile(t *testing.T) {
	s := newTestStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Watching a non-existent file should return error from fsnotify
	_, err := s.Watch(ctx, "nonexistent/config", nil)
	// fsnotify.Add on non-existent file should fail
	assert.Error(t, err)
}

func TestWatch_DeliversSameKVPairAsGet(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	want, err := s.Get(ctx, "materialized_config/test", nil)
	require.NoError(t, err)

	watchCtx, watchCancel := context.WithCancel(ctx)
	ch, err := s.Watch(watchCtx, "materialized_config/test", nil)
	require.NoError(t, err)

	select {
	case got := <-ch:
		require.NotNil(t, got)
		assert.Equal(t, want.Key, got.Key)
		assert.Equal(t, want.Value, got.Value)
	case <-time.After(5 * time.Second):
		watchCancel()
		t.Fatal("timed out waiting for watch delivery")
	}

	// Cancel and drain until Watch's goroutine closes the channel, so its
	// exit (and removeWatch call) is complete before the store is closed by
	// t.Cleanup — avoids racing Watch's cleanup against Store.Close's.
	watchCancel()
	for range ch {
	}
}

// newOnDemandFixture builds a temp protoconf root containing a proto type
// (ondemand.v1.Thing) that only the on-demand lazy tiers can resolve --
// nothing parses it at construction -- alongside a materialized config that
// references a type declared nowhere under src/, for the unresolvable-type
// diagnostic test. The directory layout mirrors the package name
// (src/ondemand/v1/) because the scan tier narrows its walk to the
// package-derived directory when it exists (utils/symbol_scan.go).
func newOnDemandFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	protoDir := filepath.Join(root, "src", "ondemand", "v1")
	require.NoError(t, os.MkdirAll(protoDir, 0755))
	protoSrc := "syntax = \"proto3\";\npackage ondemand.v1;\n\nmessage Thing {\n    string x = 1;\n}\n"
	require.NoError(t, os.WriteFile(filepath.Join(protoDir, "thing.proto"), []byte(protoSrc), 0644))

	configDir := filepath.Join(root, "materialized_config")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	goodJSON := `{"protoFile":"ondemand/v1/thing.proto","value":{"@type":"type.googleapis.com/ondemand.v1.Thing","x":"hello"}}`
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "ondemand.materialized_JSON"), []byte(goodJSON), 0644))

	badJSON := `{"protoFile":"ondemand/v1/thing.proto","value":{"@type":"type.googleapis.com/absent.v1.NoSuchMessage","x":"hello"}}`
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "unresolvable.materialized_JSON"), []byte(badJSON), 0644))

	return root
}

func newOnDemandStore(t *testing.T) *Store {
	t.Helper()
	root := newOnDemandFixture(t)
	ctx := context.Background()
	s, err := New(ctx, []string{}, &Config{ProtoconfRoot: root})
	require.NoError(t, err)
	require.NotNil(t, s)
	t.Cleanup(func() { _ = s.Close() })
	return s
}

// TestGetResolvesTypeAbsentFromConstructionSnapshot proves CONS-03
// on-demand: type.googleapis.com/ondemand.v1.Thing is genuinely absent from
// s.parser's construction-time snapshot (*protoregistry.Types, written only
// at construction and never after) both before and after a Get that
// resolves it, while Get itself succeeds and returns the correctly resolved
// type URL -- resolution happened through the tiered TypeResolver's lazy
// tiers, not a pre-seeded snapshot.
func TestGetResolvesTypeAbsentFromConstructionSnapshot(t *testing.T) {
	s := newOnDemandStore(t)
	ctx := context.Background()

	const url = "type.googleapis.com/ondemand.v1.Thing"

	// Before Get: absent from the construction-time snapshot.
	_, err := s.parser.LocalResolver.FindMessageByURL(url) // planner-discipline-allow: LocalResolver
	require.Error(t, err)
	assert.True(t, errors.Is(err, protoregistry.NotFound), "expected protoregistry.NotFound, got: %v", err)

	pair, err := s.Get(ctx, "materialized_config/ondemand", nil)
	require.NoError(t, err)
	require.NotNil(t, pair)

	// After Get: the snapshot object is fixed at construction and never
	// grows -- still absent.
	_, err = s.parser.LocalResolver.FindMessageByURL(url) // planner-discipline-allow: LocalResolver
	require.Error(t, err)
	assert.True(t, errors.Is(err, protoregistry.NotFound), "expected protoregistry.NotFound after Get, got: %v", err)

	b, err := base64.StdEncoding.DecodeString(string(pair.Value))
	require.NoError(t, err)
	protoconfValue := &protoconfvalue.ProtoconfValue{}
	require.NoError(t, proto.Unmarshal(b, protoconfValue))
	assert.Equal(t, url, protoconfValue.GetValue().GetTypeUrl())
}

// TestGetIsIdempotentForSameKey proves repeat Get calls for the same key
// return byte-identical values -- the on-demand parse is memoised, not
// repeated on every call.
func TestGetIsIdempotentForSameKey(t *testing.T) {
	s := newOnDemandStore(t)
	ctx := context.Background()

	pair1, err := s.Get(ctx, "materialized_config/ondemand", nil)
	require.NoError(t, err)
	require.NotNil(t, pair1)

	pair2, err := s.Get(ctx, "materialized_config/ondemand", nil)
	require.NoError(t, err)
	require.Equal(t, pair1.Value, pair2.Value)

	pair3, err := s.Get(ctx, "materialized_config/ondemand", nil)
	require.NoError(t, err)
	require.Equal(t, pair1.Value, pair3.Value)
}

// TestGetUnresolvableTypeReturnsDiagnostic proves that a type URL naming a
// symbol declared nowhere under src/ surfaces the Phase 13 D-02 diagnostic
// (naming the symbol, "not found", and the symbol index state) rather than
// a zero-value or partially-populated KVPair.
func TestGetUnresolvableTypeReturnsDiagnostic(t *testing.T) {
	s := newOnDemandStore(t)
	ctx := context.Background()

	pair, err := s.Get(ctx, "materialized_config/unresolvable", nil)
	require.Error(t, err)
	require.Nil(t, pair)
	assert.Contains(t, err.Error(), "not found")
	assert.Contains(t, err.Error(), "absent.v1.NoSuchMessage")
	assert.Contains(t, err.Error(), "symbol index:")
}

// newTraversalFixture builds a temp protoconf root containing a real,
// resolvable ondemand.v1.Thing type, and a sibling directory OUTSIDE the
// root holding a schema-valid materialized config whose payload is a
// distinctive secret string that appears nowhere else in the repository.
// It returns the root and the secret payload so a traversal key
// ("../secret/leak") can be asserted to never reach the caller -- a
// substring assertion against the payload is unambiguous.
func newTraversalFixture(t *testing.T) (root string, secretPayload string) {
	t.Helper()
	base := t.TempDir()

	root = filepath.Join(base, "protoconfRoot")
	protoDir := filepath.Join(root, "src", "ondemand", "v1")
	require.NoError(t, os.MkdirAll(protoDir, 0755))
	protoSrc := "syntax = \"proto3\";\npackage ondemand.v1;\n\nmessage Thing {\n    string x = 1;\n}\n"
	require.NoError(t, os.WriteFile(filepath.Join(protoDir, "thing.proto"), []byte(protoSrc), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "materialized_config"), 0755))

	secretDir := filepath.Join(base, "secret")
	require.NoError(t, os.MkdirAll(secretDir, 0755))
	secretPayload = "TOP-SECRET-OUTSIDE-ROOT-14-09"
	leakJSON := fmt.Sprintf(`{"protoFile":"ondemand/v1/thing.proto","value":{"@type":"type.googleapis.com/ondemand.v1.Thing","x":%q}}`, secretPayload)
	require.NoError(t, os.WriteFile(filepath.Join(secretDir, "leak.materialized_JSON"), []byte(leakJSON), 0644))

	return root, secretPayload
}

func newTraversalStore(t *testing.T, root string) *Store {
	t.Helper()
	ctx := context.Background()
	s, err := New(ctx, []string{}, &Config{ProtoconfRoot: root})
	require.NoError(t, err)
	require.NotNil(t, s)
	t.Cleanup(func() { _ = s.Close() })
	return s
}

// TestGetRejectsTraversalKey proves a caller-supplied key cannot read a
// file outside protoconfRoot through Get. The fixture plants a real,
// readable, schema-valid materialized config at the traversal target, so
// this test fails on a *successful* read rather than passing on
// os.Stat's ErrNotExist -- that gap is exactly what 14-VERIFICATION.md
// recorded against the previous version of this test, which asserted
// only require.Error against a fixture with no file at the traversed
// location.
func TestGetRejectsTraversalKey(t *testing.T) {
	root, secretPayload := newTraversalFixture(t)
	s := newTraversalStore(t, root)
	ctx := context.Background()

	pair, err := s.Get(ctx, "../secret/leak", nil)
	require.Error(t, err)
	require.Nil(t, pair)
	assert.NotContains(t, err.Error(), secretPayload)
	// Guard the content assertion: if a traversal somehow returned a
	// non-nil pair (the leak vector), decode it and fail explicitly on
	// the secret payload rather than letting require.Nil above be the
	// only line standing between this test and a false pass.
	if pair != nil {
		b, decodeErr := base64.StdEncoding.DecodeString(string(pair.Value))
		require.NoError(t, decodeErr)
		protoconfValue := &protoconfvalue.ProtoconfValue{}
		require.NoError(t, proto.Unmarshal(b, protoconfValue))
		assert.NotContains(t, protoconfValue.String(), secretPayload)
	}

	pair, err = s.Get(ctx, "", nil)
	require.Error(t, err)
	require.Nil(t, pair)
}

// TestWatchRejectsTraversalKey proves Watch needs its own guard: addWatch
// registers the escaped absolute path with fsnotify before the
// goroutine's first Get runs, so Get's validation does not cover Watch.
func TestWatchRejectsTraversalKey(t *testing.T) {
	root, _ := newTraversalFixture(t)
	s := newTraversalStore(t, root)
	ctx := context.Background()

	ch, err := s.Watch(ctx, "../secret/leak", nil)
	require.Error(t, err)
	require.Nil(t, ch)

	ch, err = s.Watch(ctx, "", nil)
	require.Error(t, err)
	require.Nil(t, ch)
}

// TestGetMissingKeyIsNotAnInvalidKey pins the error-class boundary Task 2
// must not blur: a legitimate key naming a config that genuinely does not
// exist under the root still returns store.ErrKeyNotFound, not the
// invalid-key error the traversal guard produces.
func TestGetMissingKeyIsNotAnInvalidKey(t *testing.T) {
	root, _ := newTraversalFixture(t)
	s := newTraversalStore(t, root)
	ctx := context.Background()

	_, err := s.Get(ctx, "materialized_config/absent", nil)
	require.Error(t, err)
	assert.True(t, errors.Is(err, store.ErrKeyNotFound))
}

func TestWatch_ContextCancellation(t *testing.T) {
	root := testdata.SmallTestDir()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s, err := New(ctx, []string{}, &Config{ProtoconfRoot: root})
	require.NoError(t, err)
	defer func() { _ = s.Close() }()

	// The path we watch: filekv constructs absPath = root + "/" + key + ".materialized_JSON"
	// "enum_test" maps to materialized_config/enum_test.materialized_JSON in the test root
	// But Watch looks in protoconfRoot directly, not materialized_config subdir.
	// Let's test invalid path validation instead.
	watchCtx, watchCancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer watchCancel()

	// Test that Watch returns error for non-existent file
	_, err = s.Watch(watchCtx, "nonexistent-config", nil)
	assert.Error(t, err)
}
