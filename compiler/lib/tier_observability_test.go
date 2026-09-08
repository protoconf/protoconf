package lib

import (
	"context"
	"log/slog"
	"sync"
	"testing"

	"github.com/protoconf/protoconf/utils/testdata"
	"github.com/stretchr/testify/require"
)

// recordingHandler captures slog records so a test can assert on the
// structured fields an operator would actually see, rather than scraping
// formatted log text.
type recordingHandler struct {
	mu      sync.Mutex
	records []slog.Record
}

func (h *recordingHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *recordingHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = append(h.records, r.Clone())
	return nil
}

func (h *recordingHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *recordingHandler) WithGroup(string) slog.Handler      { return h }

// find returns the LAST record with the given message, and whether one
// existed. Last, not first: a test that drives more than one compile past
// the same handler needs the most recent line, not the baseline one.
func (h *recordingHandler) find(msg string) (slog.Record, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for i := len(h.records) - 1; i >= 0; i-- {
		if h.records[i].Message == msg {
			return h.records[i], true
		}
	}
	return slog.Record{}, false
}

// attr returns the named attribute's value as a string-ish any, and whether the
// record carried it at all.
func attrOf(r slog.Record, key string) (slog.Value, bool) {
	var (
		out   slog.Value
		found bool
	)
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == key {
			out, found = a.Value, true
			return false
		}
		return true
	})
	return out, found
}

// TestCompileFinishedReportsResolutionTiers covers plan prohibition 3
// (T-13-13): an escalation to an expensive resolution tier must never fire
// invisibly. D-02 deleted the whole-tree eager fallback and its
// FellBackToEager boolean; this test pins the replacement observability
// contract in kind rather than dropping it (RESEARCH.md Pitfall 1) — three
// tier counters, reported on every "compile finished" line, present and zero
// when nothing escalated, so absent and zero stay distinguishable.
func TestCompileFinishedReportsResolutionTiers(t *testing.T) {
	// Baseline: a corpus compile that never needs to escalate past the
	// growable MessageRegistry must still REPORT all three counters, as zero.
	// A line that only appears when a tier escalates would leave "absent" and
	// "did not escalate" indistinguishable.
	t.Run("reports zero for all three tiers when nothing escalates", func(t *testing.T) {
		h := &recordingHandler{}
		prev := slog.Default()
		slog.SetDefault(slog.New(h))
		t.Cleanup(func() { slog.SetDefault(prev) })

		dir := t.TempDir()
		require.NoError(t, testdata.GenerateCorpus(dir, 20))

		c, err := NewCompiler(dir, false)
		require.NoError(t, err)
		require.NoError(t, c.CompileFile("main.mpconf"))

		rec, ok := h.find("compile finished")
		require.True(t, ok, `compiler must emit a "compile finished" line an operator can read`)

		for _, key := range []string{"symbolIndexBuilds", "symbolIndexCacheHits", "scanResolutions"} {
			v, present := attrOf(rec, key)
			require.True(t, present, "the compile finished line must always carry %s, so absent and zero are distinguishable", key)
			require.EqualValues(t, 0, v.Int64(), "%s must be zero: main.mpconf's load()'d types resolve without escalating", key)
		}
	})

	// And the reported values must track reality: once a tier has actually
	// fired on this registry, the next compile's line agrees with the
	// registry's own counters. This is the D-02 replacement for the old
	// test's direct reg.ParseAll() call — there is no whole-tree fallback
	// left to trigger, so Tier 3 (the symbol index) is driven directly on
	// the same registry the compiler is using.
	t.Run("reports agree with the registry's own counters once a tier has fired", func(t *testing.T) {
		h := &recordingHandler{}
		prev := slog.Default()
		slog.SetDefault(slog.New(h))
		t.Cleanup(func() { slog.SetDefault(prev) })

		dir := t.TempDir()
		require.NoError(t, testdata.GenerateCorpus(dir, 20))

		c, err := NewCompiler(dir, false)
		require.NoError(t, err)
		require.NoError(t, c.CompileFile("main.mpconf"))

		registry := c.ModuleService.GetProtoRegistry()
		require.Equal(t, 0, registry.IndexBuildCount(), "baseline compile must not have built the index")

		// corpus.pkg7.Msg7 is never load()'d by main.mpconf (which only
		// demands pkg0..pkg4), so this is a genuine escalation on this
		// registry, not a cache hit on work the compile already did.
		_, ok, err := registry.SymbolFile("corpus.pkg7.Msg7")
		require.NoError(t, err)
		require.True(t, ok)
		require.Equal(t, 1, registry.IndexBuildCount())

		require.NoError(t, c.CompileFile("main.mpconf"))

		rec, ok := h.find("compile finished")
		require.True(t, ok)

		v, present := attrOf(rec, "symbolIndexBuilds")
		require.True(t, present)
		require.EqualValues(t, registry.IndexBuildCount(), v.Int64(),
			"the reported value must agree with the registry's actual state, otherwise the escalation fired invisibly")
	})
}
