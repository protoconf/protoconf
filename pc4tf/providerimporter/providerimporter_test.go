package providerimporter

import (
	"log"
	"testing"

	assert "github.com/stretchr/testify/require"
)

func TestBuilder(t *testing.T) {
	log.Println("starting")
	meta, err := findPlugin("provider", "random", "2.2.1")
	assert.NoError(t, err)
	config := newGRPCClientConfig(meta)
	client, err := NewGRPCClient(config)
	assert.NoError(t, err)
	defer client.Close()
	p, err := NewProviderImporter("random", client)
	assert.NoError(t, err)
	Print(p.Provider)
	Print(p.Resources)
	Print(p.Datasources)
}
