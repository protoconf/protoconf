package provider_importer

import (
	"log"
	"testing"

	// "github.com/jhump/protoreflect/desc/builder"
	assert "github.com/stretchr/testify/require"
)

// func TestGetClient(t *testing.T) {
// 	pluginName := "packet"
// 	log.Println("starting")
//
// 	_, err := NewGRPCClient(pluginName)
// 	assert.NoError(t, err)
// }

func TestBuilder(t *testing.T) {
	log.Println("starting")
	meta, err := findPlugin("provider", "packet")
	assert.NoError(t, err)
	config := newGRPCClientConfig(meta)
	client, err := NewGRPCClient(config)
	assert.NoError(t, err)
	defer client.Close()
	p, err := NewProviderImporter("packet", client)
	assert.NoError(t, err)
	Print(p.Provider)
	Print(p.Resources)
	Print(p.Datasources)
}
