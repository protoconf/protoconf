package server

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"

	protoconfparser "github.com/protoconf/protoconf/compiler/lib/parser"
	"github.com/protoconf/protoconf/consts"
	"github.com/protoconf/protoconf/utils"
	"github.com/protoconf/protoconf/utils/testdata"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// capturingHandler records slog output so a test can assert on the structured
// fields an operator would see, rather than scraping formatted text.
type capturingHandler struct {
	mu      sync.Mutex
	records []slog.Record
}

func (h *capturingHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *capturingHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = append(h.records, r.Clone())
	return nil
}

func (h *capturingHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *capturingHandler) WithGroup(string) slog.Handler      { return h }

// mentioning reports whether any record carries an attribute whose value
// renders as want.
func (h *capturingHandler) mentioning(key, want string) (slog.Record, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, r := range h.records {
		var hit bool
		r.Attrs(func(a slog.Attr) bool {
			if a.Key == key && a.Value.String() == want {
				hit = true
				return false
			}
			return true
		})
		if hit {
			return r, true
		}
	}
	return slog.Record{}, false
}

// ineligibleServiceProto declares a service whose rpc does NOT return
// protoconf.v1.ConfigMutationResponse, so Init considers it and finds no
// eligible method.
const ineligibleServiceProto = `syntax = "proto3";
package inelig.v1;

message Thing {
    string value = 1;
}

service IneligibleService {
    rpc DoThing(Thing) returns (Thing);
}
`

// TestInitLogsServiceItCannotRegister covers plan prohibition 4
// (verification: manual): a service Init cannot register must not vanish
// without a trace.
//
// CONS-01 stopped the catalog going dark at DISCOVERY. This is the same
// failure one level down, at ELIGIBILITY: a service declared under src/ whose
// rpcs return the wrong type used to be dropped by a bare `continue`, with
// registration logged but non-registration silent. An operator saw no service
// and no explanation.
func TestInitLogsServiceItCannotRegister(t *testing.T) {
	root := t.TempDir()
	srcDir := filepath.Join(root, consts.SrcPath)
	require.NoError(t, os.MkdirAll(srcDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "ineligible.proto"), []byte(ineligibleServiceProto), 0o644))

	h := &capturingHandler{}
	prev := logger
	logger = slog.New(h)
	t.Cleanup(func() { logger = prev })

	s, err := NewProtoconfMutationServer(root)
	require.NoError(t, err)

	rpcServer := grpc.NewServer()
	require.NotPanics(t, func() { s.Init(rpcServer) })

	// It must genuinely not be registered — otherwise this test is asserting
	// a log line for a case that never occurs.
	require.NotContains(t, rpcServer.GetServiceInfo(), "inelig.v1.IneligibleService",
		"a service with no ConfigMutationResponse-returning rpc should not be registered")

	rec, found := h.mentioning("service", "inelig.v1.IneligibleService")
	require.True(t, found,
		"a service Init considered but could not register must be reported, not dropped silently (plan prohibition 4)")
	require.Equal(t, slog.LevelWarn, rec.Level,
		"an unregistrable service is an operator-actionable condition, so it should be at least a warning")
}

// TestDiscoveryScanDoesNotBackReflection covers plan prohibition 5
// (verification: manual) and locks in decision D-02: Init's new eager
// discovery registry is scoped to service REGISTRATION only and must not be
// wired into the gRPC reflection resolvers.
//
// The assertion is behavioural rather than a grep: with a deliberately bare
// parser standing in for the lazy registry at startup, registration must see
// the custom service while reflection's resolver must still not know its file.
// That asymmetry is exactly what D-02 chose, so widening reflection to the
// discovery registry — the tempting "fix" — turns this test red on purpose.
func TestDiscoveryScanDoesNotBackReflection(t *testing.T) {
	protoconfRoot := testdata.SmallTestDir()

	s, err := NewProtoconfMutationServer(protoconfRoot)
	require.NoError(t, err)

	// Stand in for the lazy registry at startup: a bare registry holding only
	// the well-known-type seed, before any config has been compiled.
	s.parser = protoconfparser.NewParserWithDescriptorRegistry(utils.NewDescriptorRegistry())

	rpcServer := grpc.NewServer()
	require.NotPanics(t, func() { s.Init(rpcServer) })

	// Registration sees it — that is CONS-01, already fixed.
	require.Contains(t, rpcServer.GetServiceInfo(), "test.v1.TestService",
		"the discovery scan must register the custom service (CONS-01)")

	// Reflection does not — that is D-02, deliberately left narrow.
	_, err = s.parser.FilesResolver.FindFileByPath("test.proto")
	require.ErrorIs(t, err, protoregistry.NotFound,
		"reflection still resolves through s.parser, so the discovery registry must not have been merged into it (D-02); widening reflection is a later phase's decision")
}
