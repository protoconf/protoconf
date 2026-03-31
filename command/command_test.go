package command

import (
	"strings"
	"testing"

	"github.com/protoconf/protoconf/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// _ ensures testutil is referenced (satisfies D-03 import requirement).
var _ = testutil.NewAny

// mockUi is a simple in-memory cli.Ui for asserting method calls in tests.
type mockUi struct {
	outputs []string
	infos   []string
	errors  []string
	warns   []string
}

func (m *mockUi) Ask(query string) (string, error)       { return query, nil }
func (m *mockUi) AskSecret(query string) (string, error) { return query, nil }
func (m *mockUi) Output(s string)                        { m.outputs = append(m.outputs, s) }
func (m *mockUi) Info(s string)                          { m.infos = append(m.infos, s) }
func (m *mockUi) Error(s string)                         { m.errors = append(m.errors, s) }
func (m *mockUi) Warn(s string)                          { m.warns = append(m.warns, s) }

func TestDefaultUI(t *testing.T) {
	assert.NotNil(t, DefaultUI, "DefaultUI must not be nil")
	// DefaultUI is statically typed as *cli.ConcurrentUi — just verify it is non-nil.
}

func TestNewPrefixedUi(t *testing.T) {
	prefix := "prefix> "
	p := NewPrefixedUi(prefix)
	require.NotNil(t, p)
	assert.Equal(t, prefix, p.Prefix)
	assert.Equal(t, DefaultUI, p.Ui, "Ui field must default to DefaultUI")
}

func TestPrefixedUi_Output(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single line",
			input:    "hello",
			expected: "prefix> hello",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "prefix> ",
		},
		{
			name:     "multi-line",
			input:    "line1\nline2",
			expected: "prefix> line1\nprefix> line2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &mockUi{}
			p := &PrefixedUi{Ui: m, Prefix: "prefix> "}
			p.Output(tt.input)
			require.Len(t, m.outputs, 1)
			assert.Equal(t, tt.expected, m.outputs[0])
		})
	}
}

func TestPrefixedUi_Info(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "single line", input: "info msg", expected: "> info msg"},
		{name: "empty string", input: "", expected: "> "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &mockUi{}
			p := &PrefixedUi{Ui: m, Prefix: "> "}
			p.Info(tt.input)
			require.Len(t, m.infos, 1)
			assert.Equal(t, tt.expected, m.infos[0])
		})
	}
}

func TestPrefixedUi_Error(t *testing.T) {
	m := &mockUi{}
	p := &PrefixedUi{Ui: m, Prefix: "ERR: "}
	p.Error("something went wrong")
	require.Len(t, m.errors, 1)
	assert.True(t, strings.HasPrefix(m.errors[0], "ERR: "), "error output must be prefixed")
}

func TestPrefixedUi_Warn(t *testing.T) {
	m := &mockUi{}
	p := &PrefixedUi{Ui: m, Prefix: "WARN: "}
	p.Warn("be careful")
	require.Len(t, m.warns, 1)
	assert.Equal(t, "WARN: be careful", m.warns[0])
}

func TestPrefixedUi_Ask(t *testing.T) {
	m := &mockUi{}
	p := &PrefixedUi{Ui: m, Prefix: "? "}
	answer, err := p.Ask("your name")
	require.NoError(t, err)
	// mockUi.Ask returns the query it received, so verify prefix was prepended.
	assert.Equal(t, "? your name", answer)
}

func TestPrefixedUi_AskSecret(t *testing.T) {
	m := &mockUi{}
	p := &PrefixedUi{Ui: m, Prefix: "? "}
	secret, err := p.AskSecret("password")
	require.NoError(t, err)
	assert.Equal(t, "? password", secret)
}

// TestPrefixedUi_NilUiPanics verifies that constructing a PrefixedUi with a nil
// Ui and calling a method results in a panic (error path / TEST-16).
func TestPrefixedUi_NilUiPanics(t *testing.T) {
	p := &PrefixedUi{Ui: nil, Prefix: "x> "}
	assert.Panics(t, func() {
		p.Output("hello")
	}, "calling Output on a PrefixedUi with nil Ui must panic")
}
