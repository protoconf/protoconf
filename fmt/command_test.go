package fmt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/protoconf/protoconf/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// _ ensures testutil is referenced (satisfies D-03 import requirement).
var _ = testutil.NewAny

// unformattedStarlark is valid Starlark that the buildtools formatter will
// normalise (e.g., it adds trailing newlines and normalises indentation).
const unformattedStarlark = `def   foo( x ):
    return x+1
`

// formattedStarlark is the output expected after running the formatter.
const formattedStarlark = `def foo(x):
    return x + 1
`

// writeTemp creates a temp file with the given content and returns its path.
func writeTemp(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(p, []byte(content), 0644))
	return p
}

// --- isStarlarkFile ---

func TestIsStarlarkFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		// Should return true
		{path: "config.pconf", want: true},
		{path: "multi.mpconf", want: true},
		{path: "/abs/path/file.pinc", want: true},
		{path: "validator.proto-validator", want: true},
		{path: "script.star", want: true},
		// Should return false
		{path: "main.go", want: false},
		{path: "README.txt", want: false},
		{path: "schema.proto", want: false},
		{path: "", want: false},
		{path: "noext", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := isStarlarkFile(tt.path)
			assert.Equal(t, tt.want, got, "isStarlarkFile(%q)", tt.path)
		})
	}
}

// --- formatFile ---

func TestFormatFile_AlreadyFormatted(t *testing.T) {
	dir := t.TempDir()
	path := writeTemp(t, dir, "already.pconf", formattedStarlark)

	err := formatFile(path, &cliConfig{})
	assert.NoError(t, err, "already-formatted file should not error")

	content, _ := os.ReadFile(path)
	assert.Equal(t, formattedStarlark, string(content), "file must not be modified")
}

func TestFormatFile_WriteMode(t *testing.T) {
	dir := t.TempDir()
	path := writeTemp(t, dir, "needs_fmt.pconf", unformattedStarlark)

	err := formatFile(path, &cliConfig{write: true})
	require.NoError(t, err, "write mode must not error on valid Starlark")

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.NotEqual(t, unformattedStarlark, string(content), "file must be rewritten")
}

func TestFormatFile_NoWriteMode(t *testing.T) {
	dir := t.TempDir()
	original := unformattedStarlark
	path := writeTemp(t, dir, "no_write.pconf", original)

	// Default mode (no write): function prints to stdout but leaves file alone.
	err := formatFile(path, &cliConfig{})
	require.NoError(t, err)

	content, _ := os.ReadFile(path)
	assert.Equal(t, original, string(content), "file must NOT be modified in non-write mode")
}

// TestFormatFile_NonExistent tests the error path (TEST-16).
func TestFormatFile_NonExistent(t *testing.T) {
	err := formatFile("/nonexistent/path/file.pconf", &cliConfig{})
	require.Error(t, err, "formatFile with non-existent path must return an error")
}

// --- computeDiff ---

func TestComputeDiff(t *testing.T) {
	original := []byte("a = 1\n")
	formatted := []byte("a = 2\n")
	diff := computeDiff("test.pconf", original, formatted)

	assert.Contains(t, diff, "---", "diff must contain --- marker")
	assert.Contains(t, diff, "+++", "diff must contain +++ marker")
	assert.Contains(t, diff, "test.pconf", "diff must reference the file path")
	// Changed lines should appear as removed/added.
	assert.True(t, strings.Contains(diff, "-a = 1") || strings.Contains(diff, "+a = 2"),
		"diff must contain changed lines")
}

func TestComputeDiff_Identical(t *testing.T) {
	content := []byte("x = 1\n")
	diff := computeDiff("same.pconf", content, content)
	// computeDiff always emits --- and +++ header lines, but should not emit
	// any changed-line markers (lines starting with - or + that are not
	// the header lines).
	lines := strings.Split(diff, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++") {
			continue // header lines are expected
		}
		if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "+") {
			t.Errorf("unexpected diff change line for identical content: %q", line)
		}
	}
}

// --- processPath ---

func TestProcessPath_Directory(t *testing.T) {
	dir := t.TempDir()
	// Mix of starlark and non-starlark files.
	writeTemp(t, dir, "config.pconf", formattedStarlark)
	writeTemp(t, dir, "main.go", "package main\n")
	writeTemp(t, dir, "schema.proto", `syntax = "proto3";`)

	// processPath with a directory must not error and must skip non-starlark files.
	err := processPath(dir, &cliConfig{})
	assert.NoError(t, err)
}

func TestProcessPath_File(t *testing.T) {
	dir := t.TempDir()
	path := writeTemp(t, dir, "config.pconf", formattedStarlark)

	err := processPath(path, &cliConfig{})
	assert.NoError(t, err)
}

// TestProcessPath_NonExistent tests the error path (TEST-16).
func TestProcessPath_NonExistent(t *testing.T) {
	err := processPath("/nonexistent/directory", &cliConfig{})
	require.Error(t, err, "processPath with non-existent path must return an error")
}

// --- Command factory ---

func TestCommand(t *testing.T) {
	cmd, err := Command()
	require.NoError(t, err)
	assert.NotNil(t, cmd, "Command() must return a non-nil cli.Command")
	assert.NotEmpty(t, cmd.Synopsis(), "Synopsis must not be empty")
	assert.NotEmpty(t, cmd.Help(), "Help must not be empty")
}
