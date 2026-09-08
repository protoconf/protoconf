package utils

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/stretchr/testify/require"
)

const canonicalRaceProto = `syntax = "proto3";
package canon.v1;

message CanonMessage {
    string value = 1;
}
`

// canonicalRaceRoot writes a one-proto import root. One file is enough: the
// interleaving under test is per-path, not per-tree.
func canonicalRaceRoot(t *testing.T) string {
	t.Helper()

	src := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(src, "canon.proto"), []byte(canonicalRaceProto), 0o644))

	return src
}

// TestParseOneReturnsCanonicalPointerWhenRacingConcurrentInsert covers code
// review WR-02.
//
// ParseOne must parse outside d.mu — protoparse calls LookupImport from inside
// ParseFiles, and that closure takes RLock, so holding the write lock across
// the parse would self-deadlock. That correct choice opens a window: some
// other writer could insert the same path into FileRegistry while ParseOne's
// own parse is still in flight outside the lock. recordFileLocked is a no-op
// when the path is already present, so a ParseOne that returned its own
// freshly-parsed fds[0] would hand its caller a different pointer than every
// subsequent map lookup observes — breaking the pointer-identity contract
// TestParseMemoization asserts for LAZY-02.
//
// D-02 deleted the whole-tree eager fallback (ParseAll) that used to stage
// this interleaving for free. The afterParseHook seam still makes it
// deterministic instead of probabilistic: it now drives a direct, independently
// parsed, locked insert for the same path — the same recordFileLocked call any
// other writer would make — rather than a goroutine-racing version of this
// test, which would pass whenever it simply missed the window (the "green
// without exercising the claim" failure mode code review flagged as WR-05
// elsewhere in this phase).
func TestParseOneReturnsCanonicalPointerWhenRacingConcurrentInsert(t *testing.T) {
	src := canonicalRaceRoot(t)

	d := NewDescriptorRegistry()
	d.ImportPaths = []string{src}

	// Force the exact interleaving: after ParseOne has parsed but before it
	// takes the write lock to insert, an independent parse of the same file
	// is inserted directly under the lock. That insert's recordFileLocked
	// call becomes the canonical entry, so ParseOne's own subsequent
	// recordFileLocked call becomes a no-op.
	var hookRuns int
	d.afterParseHook = func() {
		hookRuns++

		parser := &protoparse.Parser{
			ImportPaths: []string{src},
			Accessor: func(filename string) (io.ReadCloser, error) {
				return os.Open(filename)
			},
		}
		fds, err := parser.ParseFiles("canon.proto")
		require.NoError(t, err)
		require.Len(t, fds, 1)

		d.mu.Lock()
		d.recordFileLocked(fds[0])
		d.mu.Unlock()
	}

	got, err := d.ParseOne("canon.proto")
	require.NoError(t, err)
	require.NotNil(t, got)

	// Guard against a vacuous pass: if the hook never ran, ParseOne took the
	// cache fast path and the race was never staged at all.
	require.Equal(t, 1, hookRuns, "afterParseHook must have run, or the concurrent-insert interleaving was never staged")

	d.mu.RLock()
	canonical, ok := d.FileRegistry["canon.proto"]
	d.mu.RUnlock()
	require.True(t, ok, "the path must be present in FileRegistry after ParseOne")

	// The actual WR-02 assertion: what ParseOne handed back is the same
	// pointer the registry considers canonical.
	require.Same(t, canonical, got,
		"ParseOne returned a non-canonical descriptor pointer: callers would disagree with FileRegistry about the identity of canon.proto")

	// And the contract holds for the next caller too, who takes the fast path.
	again, err := d.ParseOne("canon.proto")
	require.NoError(t, err)
	require.Same(t, canonical, again, "a subsequent ParseOne must observe the same canonical pointer")
}
