package parser

import (
	"path/filepath"
	"testing"

	"github.com/protoconf/protoconf/utils"
	"github.com/protoconf/protoconf/utils/testdata"
	"github.com/stretchr/testify/require"
)

// TestReReferencedProtoKeepsPointerIdentity asserts RSLV-03 / ROADMAP
// criterion 1: compiling a path that loads proto A, then proto B, then
// re-references A resolves all three correctly in one pass, and A's
// descriptor is the identical Go pointer both times ParseFilesX returns
// it — regardless of which of ParseFilesX's lookup branches served the
// request. Corpus of 20 so pkg2 (chosen as a low-index A) can legitimately
// sit in pkg9's (B) own transitive dependency closure — the interleaving
// where a second descriptor for A would otherwise be minted.
func TestReReferencedProtoKeepsPointerIdentity(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, testdata.GenerateCorpus(dir, 20))
	src := filepath.Join(dir, "src")

	dr := utils.NewDescriptorRegistry()
	dr.ImportPaths = []string{src}
	p := NewParserWithDescriptorRegistry(dr)

	// A then B then A.
	aFirstResults, err := p.ParseFilesX("pkg2/msg2.proto")
	require.NoError(t, err)
	require.Len(t, aFirstResults, 1)
	aFirst := aFirstResults[0]
	require.Equal(t, "pkg2/msg2.proto", aFirst.GetName())

	bResults, err := p.ParseFilesX("pkg9/msg9.proto")
	require.NoError(t, err)
	require.Len(t, bResults, 1)
	b := bResults[0]
	require.Equal(t, "pkg9/msg9.proto", b.GetName())

	aSecondResults, err := p.ParseFilesX("pkg2/msg2.proto")
	require.NoError(t, err)
	require.Len(t, aSecondResults, 1)
	aSecond := aSecondResults[0]
	require.Equal(t, "pkg2/msg2.proto", aSecond.GetName())

	require.Same(t, aFirst, aSecond, "A's descriptor must be one Go object across both references")

	// Canonical for every branch: the descriptor ParseFilesX returned must be
	// require.Same as the registry's own canonical entry, no matter which
	// lookup branch served the request. This is the assertion that actually
	// guards the contract — branch 2 cannot be forced from a black-box test
	// once FileRegistry and the growable resolver are written at the same
	// locked insert point, so the contract is asserted over every returned
	// descriptor rather than over that one branch.
	canonicalA, ok := dr.FileDescriptor("pkg2/msg2.proto")
	require.True(t, ok, "registry must hold a canonical entry for pkg2/msg2.proto")
	require.Same(t, canonicalA, aFirst, "ParseFilesX must return the registry's canonical descriptor for A (first reference)")
	require.Same(t, canonicalA, aSecond, "ParseFilesX must return the registry's canonical descriptor for A (re-reference)")

	canonicalB, ok := dr.FileDescriptor("pkg9/msg9.proto")
	require.True(t, ok, "registry must hold a canonical entry for pkg9/msg9.proto")
	require.Same(t, canonicalB, b, "ParseFilesX must return the registry's canonical descriptor for B")

	// Adjacency and ordering: a single call listing three filenames with the
	// first repeated as the third returns exactly 3 results in argument
	// order — equal filenames neither merge into one result nor collide into
	// different descriptors.
	results, err := p.ParseFilesX("pkg2/msg2.proto", "pkg9/msg9.proto", "pkg2/msg2.proto")
	require.NoError(t, err)
	require.Len(t, results, 3)
	require.Equal(t, "pkg2/msg2.proto", results[0].GetName())
	require.Equal(t, "pkg9/msg9.proto", results[1].GetName())
	require.Equal(t, "pkg2/msg2.proto", results[2].GetName())
	require.Same(t, results[0], results[2])

	// Empty input: no arguments, no results, no error, no panic.
	empty, err := p.ParseFilesX()
	require.NoError(t, err)
	require.Empty(t, empty)

	// Unknown path: a non-nil error and no partial results silently
	// substituted.
	unknown, err := p.ParseFilesX("pkg99/msg99.proto")
	require.Error(t, err)
	require.Empty(t, unknown)
}
