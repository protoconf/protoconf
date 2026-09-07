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

// find returns the first record with the given message, and whether one existed.
func (h *recordingHandler) find(msg string) (slog.Record, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, r := range h.records {
		if r.Message == msg {
			return r, true
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

// TestCompileFinishedReportsEagerFallback covers plan prohibition 2
// (verification: manual): the D-03 eager fallback must never fire invisibly.
//
// D-03 keeps a latched one-shot whole-tree ParseAll for the case a materialized
// config names a bare @type with no protoFile hint. That fallback undoes this
// phase's entire performance claim for the compile it fires on, so an operator
// has to be able to see that it happened. If it could fire silently, LAZY-01's
// claim would be unfalsifiable in the field — which is precisely what this
// prohibition forbids.
//
// Asserted on the structured attribute rather than formatted text, so the test
// pins the operator-visible contract without being brittle about phrasing.
func TestCompileFinishedReportsEagerFallback(t *testing.T) {
	// Baseline: a corpus compile that never needs the fallback must still
	// REPORT the field, as false. A line that only appears when the fallback
	// fires would leave "absent" and "did not fire" indistinguishable.
	t.Run("reports false when the fallback never fires", func(t *testing.T) {
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

		v, present := attrOf(rec, "eagerFallback")
		require.True(t, present, "the compile finished line must always carry eagerFallback, so absent and false are distinguishable")
		require.False(t, v.Bool(), "the generated corpus resolves every type by path; the fallback should not have fired")

		require.False(t, c.ModuleService.GetProtoRegistry().FellBackToEager(),
			"the reported value must agree with the registry's actual state")
	})

	// And the reported value must track reality: after the fallback has
	// actually fired, the accessor backing the log field says so.
	t.Run("reports true once the fallback has fired", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, testdata.GenerateCorpus(dir, 20))

		c, err := NewCompiler(dir, false)
		require.NoError(t, err)
		require.NoError(t, c.CompileFile("main.mpconf"))

		reg := c.ModuleService.GetProtoRegistry()
		require.False(t, reg.FellBackToEager())

		require.NoError(t, reg.ParseAll())

		require.True(t, reg.FellBackToEager(),
			"once ParseAll has run, the flag the compile finished line reports must be true — otherwise the fallback fired invisibly")
	})
}
