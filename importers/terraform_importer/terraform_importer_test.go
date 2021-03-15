package terraformimporter

import (
	"io/ioutil"
	"log"
	"testing"

	assert "github.com/stretchr/testify/require"
)

func TestGenerate(t *testing.T) {
	dir, err := ioutil.TempDir("", "generate_test")
	assert.NoError(t, err)
	provider := "google"
	version := "3.59.0"
	err = DownloadPlugin(dir, provider, version)
	assert.NoError(t, err)
	dst, err := ioutil.TempDir("", "generate_dest")
	assert.NoError(t, err)
	g := NewGenerator(dir, dst)
	err = g.PopulateProviders()
	assert.NoError(t, err)
	for name := range g.Providers {
		log.Println("found", name)
	}
	assert.Equal(t, "", "")
	err = g.Save()
	assert.NoError(t, err)
}
