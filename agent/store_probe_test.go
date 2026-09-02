package agent

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/kvtools/valkeyrie/store"
	protoconf_agent_config "github.com/protoconf/protoconf/agent/config/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// probeStore is a local store.Store fake used only to observe the key passed
// to Exists by the agent constructors, and to control what Exists returns.
// It embeds store.Store so only Exists needs a body; nothing else is called
// during construction.
type probeStore struct {
	store.Store
	gotKey string
	exists bool
	err    error
}

func (p *probeStore) Exists(_ context.Context, key string, _ *store.ReadOptions) (bool, error) {
	p.gotKey = key
	return p.exists, p.err
}

func TestStoreAvailabilityProbe(t *testing.T) {
	newAgentConstructors := []struct {
		name     string
		newAgent func(store.Store, *protoconf_agent_config.AgentConfig) error
	}{
		{
			name: "NewProtoconfKVAgent",
			newAgent: func(s store.Store, cfg *protoconf_agent_config.AgentConfig) error {
				_, err := NewProtoconfKVAgent(s, cfg)
				return err
			},
		},
		{
			name: "NewProtoconfKVAgentRollout",
			newAgent: func(s store.Store, cfg *protoconf_agent_config.AgentConfig) error {
				_, err := NewProtoconfKVAgentRollout(s, cfg)
				return err
			},
		},
	}

	for _, tc := range newAgentConstructors {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("probe key is non-empty after leading-slash normalization", func(t *testing.T) {
				for _, prefix := range []string{"", "some/prefix"} {
					ps := &probeStore{exists: false, err: nil}
					err := tc.newAgent(ps, &protoconf_agent_config.AgentConfig{Prefix: prefix})
					require.NoError(t, err)
					normalized := strings.TrimPrefix(ps.gotKey, "/")
					assert.NotEmpty(t, normalized, "probe key %q normalizes to empty for prefix %q", ps.gotKey, prefix)
				}
			})

			t.Run("absent key succeeds", func(t *testing.T) {
				ps := &probeStore{exists: false, err: nil}
				err := tc.newAgent(ps, &protoconf_agent_config.AgentConfig{})
				assert.NoError(t, err)
			})

			t.Run("transport error fails", func(t *testing.T) {
				ps := &probeStore{err: errors.New("etcdserver: key is not provided")}
				err := tc.newAgent(ps, &protoconf_agent_config.AgentConfig{})
				require.Error(t, err)
				assert.Contains(t, err.Error(), "store is not available")
			})
		})
	}
}
