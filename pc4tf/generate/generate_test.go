package generate

import (
	"io/ioutil"
	"log"
	"testing"

	// "github.com/jhump/protoreflect/desc/builder"
	"github.com/protoconf/protoconf/pc4tf/provider_importer"
	assert "github.com/stretchr/testify/require"
)

func TestGenerate(t *testing.T) {
	dir, err := ioutil.TempDir("", "generate_test")
	assert.NoError(t, err)
	err = provider_importer.DownloadPlugin(dir, "random", "2.2.1")
	assert.NoError(t, err)
	dst, err := ioutil.TempDir("", "generate_dest")
	assert.NoError(t, err)
	g := NewGenerator(dir, dst)
	err = g.PopulateProviders()
	assert.NoError(t, err)
	for name, _ := range g.Providers {
		log.Println("found", name)
	}
	assert.Equal(t, "", "")
	err = g.Save()
	assert.NoError(t, err)
}
