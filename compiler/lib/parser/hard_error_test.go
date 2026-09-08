package parser

import (
	"testing"

	"github.com/protoconf/protoconf/utils"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// TestUnresolvableTypeURLErrorIsDiagnostic pins D-02: a type URL that
// survives the whole resolveTiers chain returns an error that (a) still
// satisfies errors.Is(err, protoregistry.NotFound), so existing callers that
// check for that sentinel keep working unchanged, and carries three pieces
// of diagnostic content an operator needs to act on the failure: (b) the
// unresolved type URL itself, (c) the import root(s) that were searched, and
// (d) the symbol index's build/cache state.
func TestUnresolvableTypeURLErrorIsDiagnostic(t *testing.T) {
	src := writeNestedAnySrc(t)
	_, p := newLazyParser(src)

	const url = "type.googleapis.com/ghost.v1.TrulyMissing"
	_, err := p.TypeResolver.FindMessageByURL(url)
	require.Error(t, err)

	require.ErrorIs(t, err, protoregistry.NotFound)
	require.ErrorContains(t, err, url)
	require.ErrorContains(t, err, src)
	require.ErrorContains(t, err, "rebuilt",
		"a first lookup on a fresh registry must report the index as freshly built")
}

// TestUnresolvableTypeURLReportsIndexState pins the same lookup run against
// two registries sharing one CacheDir over an unchanged tree: the first
// (cold) build reports "rebuilt", the second reports "cache hit" -- the
// error's index-state content must distinguish the two rather than reporting
// the same word regardless of how the index was obtained.
func TestUnresolvableTypeURLReportsIndexState(t *testing.T) {
	src := writeNestedAnySrc(t)
	cacheDir := t.TempDir()
	const url = "type.googleapis.com/ghost.v1.TrulyMissing"

	registry1 := utils.NewDescriptorRegistry()
	registry1.ImportPaths = []string{src}
	registry1.CacheDir = cacheDir
	p1 := NewParserWithDescriptorRegistry(registry1)

	_, err1 := p1.TypeResolver.FindMessageByURL(url)
	require.Error(t, err1)
	require.ErrorContains(t, err1, "rebuilt")

	registry2 := utils.NewDescriptorRegistry()
	registry2.ImportPaths = []string{src}
	registry2.CacheDir = cacheDir
	p2 := NewParserWithDescriptorRegistry(registry2)

	_, err2 := p2.TypeResolver.FindMessageByURL(url)
	require.Error(t, err2)
	require.ErrorContains(t, err2, "cache hit")
}

// TestEmptyTypeURLFailsWithoutScanOrIndex is the empty/malformed-input edge
// TYPE-08's adjacency case exists for: an implementation that scanned or
// indexed the whole tree before giving up on a URL with nothing resolvable
// in it would still return the right error, so both counters -- not just the
// error -- are asserted.
func TestEmptyTypeURLFailsWithoutScanOrIndex(t *testing.T) {
	src := writeNestedAnySrc(t)
	registry, p := newLazyParser(src)

	// "" is the empty type URL. "nohostprefixhere" carries no
	// "type.googleapis.com/" host prefix and, once that (absent) prefix is
	// stripped, is a single all-lowercase segment with no PascalCase
	// component -- structurally not a resolvable package.Message shape.
	for _, url := range []string{"", "nohostprefixhere"} {
		_, err := p.TypeResolver.FindMessageByURL(url)
		require.Error(t, err, "url=%q", url)
		require.ErrorIs(t, err, protoregistry.NotFound, "url=%q", url)
	}

	require.Equal(t, 0, registry.ScanResolutionCount())
	require.Equal(t, 0, registry.IndexBuildCount())
}

// TestTier0HitConsultsNothingElse pins the D-01 tier order's other edge: a
// symbol resolvable at Tier 0 (the construction-time snapshot) must never
// consult the scan or index tiers at all.
func TestTier0HitConsultsNothingElse(t *testing.T) {
	src := writeNestedAnySrc(t)
	registry, p := newLazyParser(src)

	mt, err := p.TypeResolver.FindMessageByURL("type.googleapis.com/google.protobuf.Value")
	require.NoError(t, err)
	require.NotNil(t, mt)

	require.Equal(t, 0, registry.ScanResolutionCount())
	require.Equal(t, 0, registry.IndexBuildCount())
}

// TestBothEntryPointsShareTheTierChain pins TYPE-08 at the error boundary:
// FindMessageByURL and FindMessageByName, asked about the same unresolvable
// symbol, must produce errors carrying the same searched-roots and
// index-state content -- proving one resolveTiers implementation backs both
// entry points, not two independently-written chains that could drift.
func TestBothEntryPointsShareTheTierChain(t *testing.T) {
	src := writeNestedAnySrc(t)
	_, p := newLazyParser(src)

	_, byURLErr := p.TypeResolver.FindMessageByURL("type.googleapis.com/ghost.v1.TrulyMissing")
	require.Error(t, byURLErr)

	_, byNameErr := p.TypeResolver.FindMessageByName("ghost.v1.TrulyMissing")
	require.Error(t, byNameErr)

	require.ErrorContains(t, byURLErr, src)
	require.ErrorContains(t, byNameErr, src)
	require.ErrorContains(t, byURLErr, "rebuilt")
	require.ErrorContains(t, byNameErr, "rebuilt")
}
