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
	err = DownloadPlugin(dir, "random", "2.2.1")
	assert.NoError(t, err)
	dst, err := ioutil.TempDir("", "generate_dest")
	assert.NoError(t, err)
	dst = "/tmp/importers/terraform"
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
