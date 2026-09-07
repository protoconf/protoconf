package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestParseOneFailsLoudlyOnBrokenProto covers plan prohibition 1
// (verification: manual): a proto that fails to parse on the lazy path must
// not be silently swallowed into a successful compile.
//
// The eager path failed loudly by construction — Import parsed everything up
// front, so a broken file surfaced at construction. On-demand parsing moves
// that failure to first use, which is exactly where a swallowed error would
// become invisible: the compile would carry on with a type it never actually
// loaded.
func TestParseOneFailsLoudlyOnBrokenProto(t *testing.T) {
	src := t.TempDir()

	// Syntactically invalid: unterminated message body, bogus field.
	broken := "syntax = \"proto3\";\npackage broken.v1;\n\nmessage Broken {\n    this is not a field\n"
	require.NoError(t, os.WriteFile(filepath.Join(src, "broken.proto"), []byte(broken), 0o644))

	d := NewDescriptorRegistry()
	d.ImportPaths = []string{src}

	fd, err := d.ParseOne("broken.proto")

	require.Error(t, err, "a malformed proto must surface an error, never a silent nil-error miss")
	require.Nil(t, fd, "a failed parse must not hand back a descriptor")

	// And it must not be recorded as if it had loaded — a poisoned cache entry
	// would make the SECOND request succeed, turning a hard failure into an
	// intermittent one.
	d.mu.RLock()
	_, present := d.FileRegistry["broken.proto"]
	d.mu.RUnlock()
	require.False(t, present, "a file that failed to parse must not be memoised as loaded")

	// A second attempt must fail the same way rather than succeeding off a
	// half-populated cache.
	fd2, err2 := d.ParseOne("broken.proto")
	require.Error(t, err2, "the failure must be reproducible, not swallowed on retry")
	require.Nil(t, fd2)
}
