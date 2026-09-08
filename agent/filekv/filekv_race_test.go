package filekv

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	protoconfvalue "github.com/protoconf/protoconf/datatypes/proto/v1"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
)

// tightLoopKeyCount is the number of distinct not-yet-parsed keys the
// writer side of TestFileKVGetTightLoopIsRaceFree resolves concurrently,
// and the number of keys TestFileKVConcurrentGetReturnsCorrectValuePerKey
// asserts individually.
const tightLoopKeyCount = 24

// newTightLoopRoot builds one temp protoconf root holding tightLoopKeyCount
// distinct message types -- each in its own package-matching directory, so
// each Get resolves independently through the scan tier rather than
// sharing one memoised parse -- and a matching materialized config for
// each. Layout mirrors filekv_test.go's newOnDemandFixture precedent.
func newTightLoopRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	configDir := filepath.Join(root, "materialized_config")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	for i := 0; i < tightLoopKeyCount; i++ {
		protoDir := filepath.Join(root, "src", "tight", fmt.Sprintf("v%d", i))
		require.NoError(t, os.MkdirAll(protoDir, 0755))
		protoSrc := fmt.Sprintf(
			"syntax = \"proto3\";\npackage tight.v%d;\n\nmessage Msg {\n    string value = 1;\n}\n",
			i,
		)
		require.NoError(t, os.WriteFile(filepath.Join(protoDir, "msg.proto"), []byte(protoSrc), 0644))

		configJSON := fmt.Sprintf(
			`{"protoFile":"tight/v%d/msg.proto","value":{"@type":"type.googleapis.com/tight.v%d.Msg","value":"hello-%d"}}`,
			i, i, i,
		)
		require.NoError(t, os.WriteFile(
			filepath.Join(configDir, fmt.Sprintf("tight%d.materialized_JSON", i)),
			[]byte(configJSON), 0644,
		))
	}

	return root
}

// TestFileKVGetTightLoopIsRaceFree forces the ParseOne read/record
// interleaving on the shared *utils.DescriptorRegistry continuously,
// rather than trusting TestSubscribeForConfigConcurrentClientsAreRaceFree
// to hit it by luck (SAFE-02, D-07 dedicated half): two unpaced reader
// goroutines hammer s.Get for an already-resolved key with no sleep and no
// pacing for the whole run, while an errgroup of writer goroutines each
// resolve one distinct not-yet-parsed key concurrently, so ParseOne
// records into the shared registry while the readers are mid-flight.
//
// See utils/growable_resolver_race_test.go's TestRegisterFileRacesRangeFiles
// for the unpaced tight-loop precedent this adapts.
func TestFileKVGetTightLoopIsRaceFree(t *testing.T) {
	root := newTightLoopRoot(t)
	ctx := context.Background()
	s, err := New(ctx, []string{}, &Config{ProtoconfRoot: root})
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })

	// Warm key 0 so the readers hammer an already-resolved type while the
	// writers below resolve keys 1..N-1 for the first time.
	_, err = s.Get(ctx, "materialized_config/tight0", nil)
	require.NoError(t, err)

	done := make(chan struct{})
	var wg sync.WaitGroup
	for r := 0; r < 2; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					_, _ = s.Get(ctx, "materialized_config/tight0", nil)
				}
			}
		}()
	}

	g := new(errgroup.Group)
	for i := 1; i < tightLoopKeyCount; i++ {
		i := i
		g.Go(func() error {
			_, err := s.Get(ctx, fmt.Sprintf("materialized_config/tight%d", i), nil)
			return err
		})
	}
	waitErr := g.Wait()

	close(done)
	wg.Wait()

	require.NoError(t, waitErr)
}

// TestFileKVConcurrentGetReturnsCorrectValuePerKey proves the CONS-03
// concurrency edge: all tightLoopKeyCount keys resolved through s.Get in
// parallel each return the config for THEIR OWN key -- a parallel run must
// never cross one key's value onto another key's response.
func TestFileKVConcurrentGetReturnsCorrectValuePerKey(t *testing.T) {
	root := newTightLoopRoot(t)
	ctx := context.Background()
	s, err := New(ctx, []string{}, &Config{ProtoconfRoot: root})
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })

	g := new(errgroup.Group)
	for i := 0; i < tightLoopKeyCount; i++ {
		i := i
		g.Go(func() error {
			key := fmt.Sprintf("materialized_config/tight%d", i)
			wantURL := fmt.Sprintf("type.googleapis.com/tight.v%d.Msg", i)

			pair, err := s.Get(ctx, key, nil)
			if err != nil {
				return fmt.Errorf("key=%s: %w", key, err)
			}
			b, err := base64.StdEncoding.DecodeString(string(pair.Value))
			if err != nil {
				return fmt.Errorf("key=%s: decode: %w", key, err)
			}
			pv := &protoconfvalue.ProtoconfValue{}
			if err := proto.Unmarshal(b, pv); err != nil {
				return fmt.Errorf("key=%s: unmarshal: %w", key, err)
			}
			if got := pv.GetValue().GetTypeUrl(); got != wantURL {
				return fmt.Errorf("key=%s: got type URL %q, want %q", key, got, wantURL)
			}
			return nil
		})
	}
	require.NoError(t, g.Wait())
}
