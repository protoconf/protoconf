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
	"github.com/protoconf/protoconf/utils/testdata"
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
		// protojson's error text uses a non-breaking space (U+00A0) between
		// "proto:" and "syntax" -- match on "syntax error" alone.
		require.Contains(t, errOutput, "syntax error")
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

// runModSync drives modSyncCommand.Run through a real FlagSet and a
// buffer-backed cli.Ui exactly as the CLI entry point does, against an
// existing protoconf root -- a sibling of runModInit for the sync entry
// point. It returns the exit code and the captured error output.
func runModSync(t *testing.T, dir string) (exitCode int, errOutput string) {
	t.Helper()
	fs := flag.NewFlagSet("sync", flag.ContinueOnError)
	ms := defaultModuleService(fs)
	require.NotNil(t, ms)

	var errBuf, outBuf bytes.Buffer
	ui := &cli.BasicUi{
		Writer:      &outBuf,
		ErrorWriter: &errBuf,
	}
	cmd := &modSyncCommand{
		ui:   ui,
		ms:   ms,
		flag: fs,
	}

	exitCode = cmd.Run([]string{"-protoconfPath", dir})
	return exitCode, errBuf.String()
}

// runModInitIn mirrors runModInit but drives modInitCommand.Run against an
// existing protoconf root directory that already carries its own CONFIGSPACE
// (e.g. testdata.SmallTestDir()), instead of creating a fresh t.TempDir()
// and writing a synthetic one.
func runModInitIn(t *testing.T, dir string) (exitCode int, errOutput string) {
	t.Helper()
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
	return exitCode, errBuf.String()
}

// deleteFdsFiles removes every *.fds file under dir/.protoconf_cache,
// keeping the directory itself, so GenFileDescriptorSet's registry.Load
// cache-hit path misses and the parse path actually runs.
func deleteFdsFiles(t *testing.T, dir string) {
	t.Helper()
	cacheDir := filepath.Join(dir, ".protoconf_cache")
	entries, err := os.ReadDir(cacheDir)
	require.NoError(t, err, "cache dir must exist")
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".fds" {
			require.NoError(t, os.Remove(filepath.Join(cacheDir, e.Name())))
		}
	}
}

// writeLockDeps marshals rr back over lockPath, mirroring ModuleService.Lock's
// own protojson options.
func writeLockDeps(t *testing.T, lockPath string, rr *module.RemoteRepo) {
	t.Helper()
	b, err := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(rr)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(lockPath, b, 0644))
}

// TestModSyncNeverPersistsEmptyDescriptorSet closes G-11-7 (UAT test 7's
// prohibition 6): `mod sync` must never persist the checksum of an empty
// descriptor set over a dependency's recorded fileDescriptorSetSum, and must
// never leave a zero-byte .fds in .protoconf_cache. This is tested at the
// mod sync CLI level rather than the registry level, because
// TestModSyncFdsByteIdentical builds its registry via an explicit Import
// over a corpus that always exists -- the missing-path branch this gap
// exploits is structurally unreachable from that test.
func TestModSyncNeverPersistsEmptyDescriptorSet(t *testing.T) {
	t.Run("unsynced_dep_no_getter_url", func(t *testing.T) {
		dir := testdata.SmallTestDir()
		deleteFdsFiles(t, dir)
		_, err := os.Stat(filepath.Join(dir, ".protoconf_cache"))
		require.NoError(t, err, "cache dir must still exist after deleting .fds files")

		lockPath := filepath.Join(dir, "protoconf.lock")
		before := readLockDeps(t, lockPath)
		terraformBefore := before.GetDeps()["terraform_repo"].GetFileDescriptorSetSum()
		vizceralBefore := before.GetDeps()["vizceral_repo"].GetFileDescriptorSetSum()
		require.NotEmpty(t, terraformBefore, "precondition: fixture must start with a real sum")
		require.NotEmpty(t, vizceralBefore, "precondition: fixture must start with a real sum")

		exit, errOut := runModSync(t, dir)
		require.NotZero(t, exit, "errOutput: %s", errOut)
		require.Contains(t, errOut, "terraform_repo", "error output must name a dependency")
		require.Contains(t, errOut, filepath.Join(dir, ".protoconf_cache"), "error output must name a searched directory")

		fdsFiles, err := filepath.Glob(filepath.Join(dir, ".protoconf_cache", "*.fds"))
		require.NoError(t, err)
		require.Empty(t, fdsFiles, "a failed generation must leave no .fds behind")

		after := readLockDeps(t, lockPath)
		terraformAfter := after.GetDeps()["terraform_repo"].GetFileDescriptorSetSum()
		vizceralAfter := after.GetDeps()["vizceral_repo"].GetFileDescriptorSetSum()
		require.Equal(t, terraformBefore, terraformAfter, "recorded sum must survive a failed sync unchanged")
		require.Equal(t, vizceralBefore, vizceralAfter, "recorded sum must survive a failed sync unchanged")
		require.NotEqual(t, "d41d8cd98f00b204e9800998ecf8427e", terraformAfter)
		require.NotEqual(t, "d41d8cd98f00b204e9800998ecf8427e", vizceralAfter)
	})

	// downloaded_dep_bad_source_path proves the guard is keyed on the
	// resolved proto paths yielding nothing, not on GetterUrl being empty:
	// this dependency is genuinely downloaded and extracted, and still
	// corrupts the lock identically if the guard is narrowed to a GetterUrl
	// check. It also exercises walk()'s error accumulation, since the
	// untouched dependency succeeds and is processed after the broken one --
	// exactly the shape that erased the failure before that fix.
	t.Run("downloaded_dep_bad_source_path", func(t *testing.T) {
		dir := testdata.SmallTestDir()

		initExit, initErrOut := runModInitIn(t, dir)
		require.Equal(t, 0, initExit, "errOutput: %s", initErrOut)

		syncExit, syncErrOut := runModSync(t, dir)
		require.Equal(t, 0, syncExit, "errOutput: %s", syncErrOut)

		terraformExtractedDir := filepath.Join(dir, ".protoconf_cache", "terraform_repo")
		_, err := os.Stat(terraformExtractedDir)
		require.NoError(t, err, "dependency must be genuinely downloaded and extracted before this subtest rewrites its sourcePath")

		lockPath := filepath.Join(dir, "protoconf.lock")
		before := readLockDeps(t, lockPath)
		terraformBefore := before.GetDeps()["terraform_repo"].GetFileDescriptorSetSum()
		vizceralBefore := before.GetDeps()["vizceral_repo"].GetFileDescriptorSetSum()
		require.NotEmpty(t, terraformBefore)
		require.NotEmpty(t, vizceralBefore)

		before.GetDeps()["terraform_repo"].SourcePath = "does-not-exist-in-extracted-archive"
		writeLockDeps(t, lockPath, before)
		deleteFdsFiles(t, dir)

		exit, errOut := runModSync(t, dir)
		require.NotZero(t, exit, "errOutput: %s", errOut)

		after := readLockDeps(t, lockPath)
		require.Equal(t, terraformBefore, after.GetDeps()["terraform_repo"].GetFileDescriptorSetSum(), "recorded sum must survive a failed sync unchanged")
		require.Equal(t, vizceralBefore, after.GetDeps()["vizceral_repo"].GetFileDescriptorSetSum(), "the untouched dependency's own sum must also survive unchanged")
		require.NotEqual(t, "d41d8cd98f00b204e9800998ecf8427e", after.GetDeps()["terraform_repo"].GetFileDescriptorSetSum())

		require.NoFileExists(t, filepath.Join(dir, ".protoconf_cache", "terraform_repo.fds"), "the broken dependency's .fds must not exist")
		vizceralFds := filepath.Join(dir, ".protoconf_cache", "vizceral_repo.fds")
		info, err := os.Stat(vizceralFds)
		require.NoError(t, err, "the succeeding dependency's .fds must exist")
		require.Greater(t, info.Size(), int64(0))
	})

	// good_path_control is the non-vacuity control: it proves the guard
	// does not pass by failing everything. Its expected sums come from the
	// committed fixture (utils/testdata/small/protoconf.lock), not from a
	// value this test computed and then compared against itself.
	t.Run("good_path_control", func(t *testing.T) {
		dir := testdata.SmallTestDir()
		deleteFdsFiles(t, dir)

		initExit, initErrOut := runModInitIn(t, dir)
		require.Equal(t, 0, initExit, "errOutput: %s", initErrOut)

		syncExit, syncErrOut := runModSync(t, dir)
		require.Equal(t, 0, syncExit, "errOutput: %s", syncErrOut)

		terraformFds := filepath.Join(dir, ".protoconf_cache", "terraform_repo.fds")
		vizceralFds := filepath.Join(dir, ".protoconf_cache", "vizceral_repo.fds")
		tInfo, err := os.Stat(terraformFds)
		require.NoError(t, err)
		require.Greater(t, tInfo.Size(), int64(0))
		vInfo, err := os.Stat(vizceralFds)
		require.NoError(t, err)
		require.Greater(t, vInfo.Size(), int64(0))

		lockPath := filepath.Join(dir, "protoconf.lock")
		after := readLockDeps(t, lockPath)
		require.Equal(t, "6556e5cfb73f535f9b91738dbd0df926", after.GetDeps()["terraform_repo"].GetFileDescriptorSetSum())
		require.Equal(t, "039f1e1023250b34054894cc58bc8b2b", after.GetDeps()["vizceral_repo"].GetFileDescriptorSetSum())
	})
}
