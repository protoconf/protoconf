package mod

// TestModInitLockFileShapes pins G-11-3's fix: `mod init` must survive every
// protoconf.lock shape reachable in practice -- no deps key, an empty deps
// object, no lock file at all, and a lock file that cannot be parsed -- and
// it must produce exactly one of two outcomes: persist the CONFIGSPACE
// dependencies and exit 0, or report the parse error and exit 1 leaving the
// file untouched. There is no third outcome (a silent no-op, or a corrupt
// file regenerated over the operator's pinned versions).
//
// This is the mod package's first test file; it drives the real entry point
// (modInitCommand.Run) rather than testing ModuleService in isolation, so a
// regression in mod/command.go's own plumbing (e.g. the deleted
// LoadFromLockFile call in this plan) is caught too.

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/mitchellh/cli"
	"github.com/protoconf/protoconf/compiler/module/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

// testConfigspace declares one local-path repo, label "r", so getter.Detect
// resolves offline (no case touches the network).
const testConfigspace = `r = remote_repo(
    label = "r",
    url = "./internal/x.tgz",
    checksum = "d41d8cd98f00b204e9800998ecf8427e",
)
`

// mergeConfigspace declares the same "r" repo at a new URL, for the
// merges_existing_entry case.
const mergeConfigspace = `r = remote_repo(
    label = "r",
    url = "./internal/NEW.tgz",
    checksum = "d41d8cd98f00b204e9800998ecf8427e",
)
`

// runModInit writes configspace (and lockContent, if writeLock) into a fresh
// t.TempDir(), then drives modInitCommand.Run through a real FlagSet and a
// buffer-backed cli.Ui exactly as the CLI entry point does. It returns the
// exit code, the captured error output, and the lock file's path.
func runModInit(t *testing.T, configspace string, lockContent []byte, writeLock bool) (exitCode int, errOutput string, lockPath string) {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "CONFIGSPACE"), []byte(configspace), 0644))
	lockPath = filepath.Join(dir, "protoconf.lock")
	if writeLock {
		require.NoError(t, os.WriteFile(lockPath, lockContent, 0644))
	}

	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	ms := defaultModuleService(fs)
	require.NotNil(t, ms)

	var errBuf, outBuf bytes.Buffer
	ui := &cli.BasicUi{
		Writer:      &outBuf,
		ErrorWriter: &errBuf,
	}
	cmd := &modInitCommand{
		ui:   ui,
		ms:   ms,
		flag: fs,
	}

	exitCode = cmd.Run([]string{"-protoconfPath", dir})
	return exitCode, errBuf.String(), lockPath
}

// depNames returns the keys of the parsed lock file's deps map.
func readLockDeps(t *testing.T, lockPath string) *module.RemoteRepo {
	t.Helper()
	b, err := os.ReadFile(lockPath)
	require.NoError(t, err)
	rr := &module.RemoteRepo{}
	require.NoError(t, protojson.Unmarshal(b, rr))
	return rr
}

func TestModInitLockFileShapes(t *testing.T) {
	t.Run("no_deps_key", func(t *testing.T) {
		exitCode, errOutput, lockPath := runModInit(t, testConfigspace, []byte(`{"url":"."}`), true)
		require.Equal(t, 0, exitCode, "errOutput: %s", errOutput)
		rr := readLockDeps(t, lockPath)
		require.Contains(t, rr.GetDeps(), "r")
	})

	t.Run("empty_object", func(t *testing.T) {
		exitCode, errOutput, lockPath := runModInit(t, testConfigspace, []byte(`{}`), true)
		require.Equal(t, 0, exitCode, "errOutput: %s", errOutput)
		rr := readLockDeps(t, lockPath)
		require.Contains(t, rr.GetDeps(), "r")
	})

	t.Run("explicit_empty_deps", func(t *testing.T) {
		// This row never panicked even before the fix -- it isolates the
		// silent-no-op defect (MergeLock discarding Init's own merge) from
		// the nil-map defect.
		exitCode, errOutput, lockPath := runModInit(t, testConfigspace, []byte(`{"url":".","deps":{}}`), true)
		require.Equal(t, 0, exitCode, "errOutput: %s", errOutput)
		rr := readLockDeps(t, lockPath)
		require.Contains(t, rr.GetDeps(), "r")
	})

	t.Run("absent", func(t *testing.T) {
		exitCode, errOutput, lockPath := runModInit(t, testConfigspace, nil, false)
		require.Equal(t, 0, exitCode, "errOutput: %s", errOutput)
		rr := readLockDeps(t, lockPath)
		require.Contains(t, rr.GetDeps(), "r")
	})

	t.Run("zero_byte", func(t *testing.T) {
		before := []byte{}
		exitCode, errOutput, lockPath := runModInit(t, testConfigspace, before, true)
		require.Equal(t, 1, exitCode)
		require.Contains(t, errOutput, "proto: syntax error")
		after, err := os.ReadFile(lockPath)
		require.NoError(t, err)
		require.Equal(t, before, after, "lock file must be left untouched on parse failure")
	})

	t.Run("truncated_json", func(t *testing.T) {
		before := []byte(`{"url": `)
		exitCode, errOutput, lockPath := runModInit(t, testConfigspace, before, true)
		require.Equal(t, 1, exitCode)
		require.NotEmpty(t, errOutput)
		after, err := os.ReadFile(lockPath)
		require.NoError(t, err)
		require.Equal(t, before, after, "lock file must be left untouched on parse failure")
	})

	t.Run("unknown_key", func(t *testing.T) {
		before := []byte(`{"url":".","bogus":1}`)
		exitCode, errOutput, lockPath := runModInit(t, testConfigspace, before, true)
		require.Equal(t, 1, exitCode)
		require.NotEmpty(t, errOutput)
		after, err := os.ReadFile(lockPath)
		require.NoError(t, err)
		require.Equal(t, before, after, "lock file must be left untouched on parse failure")
	})

	t.Run("merges_existing_entry", func(t *testing.T) {
		lock := []byte(`{
  "url": ".",
  "deps": {
    "r": {
      "label": "r",
      "url": "./internal/OLD.tgz",
      "integrity": "h1:STALEVALUE=",
      "fileDescriptorSetSum": "deadbeef"
    }
  }
}`)
		exitCode, errOutput, lockPath := runModInit(t, mergeConfigspace, lock, true)
		require.Equal(t, 0, exitCode, "errOutput: %s", errOutput)
		rr := readLockDeps(t, lockPath)
		require.Len(t, rr.GetDeps(), 1, "exactly one entry, not one stale + one new")
		r, ok := rr.GetDeps()["r"]
		require.True(t, ok)
		require.Equal(t, "./internal/NEW.tgz", r.GetUrl())
		require.Empty(t, r.GetIntegrity(), "GetterUrl changed, so integrity must be cleared")
		require.Equal(t, "deadbeef", r.GetFileDescriptorSetSum(), "fileDescriptorSetSum from the lock must survive the merge")
	})
}
