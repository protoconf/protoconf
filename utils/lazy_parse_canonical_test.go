package utils

import (
	"os"
	"path/filepath"
	"testing"

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

// TestParseOneReturnsCanonicalPointerWhenRacingParseAll covers code review
// WR-02.
//
// ParseOne must parse outside d.mu — protoparse calls LookupImport from inside
// ParseFiles, and that closure takes RLock, so holding the write lock across
// the parse would self-deadlock. That correct choice opens a window: ParseAll
// holds d.mu across its entire whole-tree parse and can complete inside it.
// recordFileLocked is a no-op when the path is already present, so a ParseOne
// that returned its own freshly-parsed fds[0] would hand its caller a
// different pointer than every subsequent map lookup observes — breaking the
// pointer-identity contract TestParseMemoization asserts for LAZY-02.
//
// The afterParseHook seam makes that interleaving deterministic instead of
// probabilistic. A goroutine-racing version of this test would pass whenever it
// simply missed the window, which is the "green without exercising the claim"
// failure mode code review flagged as WR-05 elsewhere in this phase.
func TestParseOneReturnsCanonicalPointerWhenRacingParseAll(t *testing.T) {
	src := canonicalRaceRoot(t)

	d := NewDescriptorRegistry()
	d.ImportPaths = []string{src}

	// Force the exact interleaving: after ParseOne has parsed but before it
	// takes the write lock to insert, run a full ParseAll. ParseAll inserts
	// its own descriptor for this same path, so ParseOne's subsequent
	// recordFileLocked call becomes a no-op.
	var hookRuns int
	d.afterParseHook = func() {
		hookRuns++
		require.NoError(t, d.ParseAll())
	}

	got, err := d.ParseOne("canon.proto")
	require.NoError(t, err)
	require.NotNil(t, got)

	// Guard against a vacuous pass: if the hook never ran, ParseOne took the
	// cache fast path and the race was never staged at all.
	require.Equal(t, 1, hookRuns, "afterParseHook must have run, or the ParseAll interleaving was never staged")

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
