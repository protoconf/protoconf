package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/protoconf/protoconf/agent/filekv"
	protoconf_agent_config "github.com/protoconf/protoconf/agent/config/v1"
	protoconf_pb "github.com/protoconf/protoconf/pb/protoconf/v1"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

// raceTestClientCount is the number of concurrent SubscribeForConfig
// clients driven against one shared agent in
// TestSubscribeForConfigConcurrentClientsAreRaceFree.
const raceTestClientCount = 12

// newRaceTestRoot builds a temp protoconf root holding raceTestClientCount
// distinct message types -- each in its own package-matching directory --
// and a matching materialized config for each. Distinct packages per index
// matter: the scan tier narrows its walk to the package-derived directory,
// so distinct packages give genuinely independent on-demand resolutions
// rather than one memoised parse shared by every goroutine. Layout mirrors
// filekv_test.go's newOnDemandFixture precedent.
func newRaceTestRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	configDir := filepath.Join(root, "materialized_config")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	for i := 0; i < raceTestClientCount; i++ {
		protoDir := filepath.Join(root, "src", "sub", fmt.Sprintf("v%d", i))
		require.NoError(t, os.MkdirAll(protoDir, 0755))
		protoSrc := fmt.Sprintf(
			"syntax = \"proto3\";\npackage sub.v%d;\n\nmessage Msg {\n    string value = 1;\n}\n",
			i,
		)
		require.NoError(t, os.WriteFile(filepath.Join(protoDir, "msg.proto"), []byte(protoSrc), 0644))

		configJSON := fmt.Sprintf(
			`{"protoFile":"sub/v%d/msg.proto","value":{"@type":"type.googleapis.com/sub.v%d.Msg","value":"hello-%d"}}`,
			i, i, i,
		)
		require.NoError(t, os.WriteFile(
			filepath.Join(configDir, fmt.Sprintf("sub%d.materialized_JSON", i)),
			[]byte(configJSON), 0644,
		))
	}

	return root
}

// TestSubscribeForConfigConcurrentClientsAreRaceFree proves SAFE-02 for the
// agent (D-07 e2e half): raceTestClientCount concurrent gRPC clients
// streaming SubscribeForConfig against one long-lived ProtoconfKVAgent
// backed by the lazy filekv store each receive their own config, resolved
// on demand through filekv's Get -> parser.ReadConfig -> shared tiered
// resolver, without a data race. A crossed response (one goroutine
// receiving another's type URL) fails the assertion for that goroutine's
// own expected value, so misrouting cannot pass silently.
func TestSubscribeForConfigConcurrentClientsAreRaceFree(t *testing.T) {
	root := newRaceTestRoot(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fileStore, err := filekv.New(ctx, []string{}, &filekv.Config{ProtoconfRoot: root})
	require.NoError(t, err)
	t.Cleanup(func() { _ = fileStore.Close() })

	server, err := NewProtoconfKVAgent(fileStore, &protoconf_agent_config.AgentConfig{})
	require.NoError(t, err)
	stub, closer := testServer(ctx, server)
	defer closer()

	g, gctx := errgroup.WithContext(ctx)
	for i := 0; i < raceTestClientCount; i++ {
		i := i
		g.Go(func() error {
			path := fmt.Sprintf("materialized_config/sub%d", i)
			wantTypeURL := fmt.Sprintf("type.googleapis.com/sub.v%d.Msg", i)

			watcher, err := stub.SubscribeForConfig(gctx, &protoconf_pb.ConfigSubscriptionRequest{Path: path})
			if err != nil {
				return fmt.Errorf("path=%s: subscribe: %w", path, err)
			}
			update, err := watcher.Recv()
			if err != nil {
				return fmt.Errorf("path=%s: recv: %w", path, err)
			}
			if update.GetError() != "" {
				return fmt.Errorf("path=%s: got update error: %s", path, update.GetError())
			}
			if got := update.GetValue().GetTypeUrl(); got != wantTypeURL {
				return fmt.Errorf("path=%s: got type URL %q, want %q", path, got, wantTypeURL)
			}
			return nil
		})
	}
	require.NoError(t, g.Wait())
}
