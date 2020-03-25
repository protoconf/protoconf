package golangimporter

import (
	"testing"

	assert "github.com/stretchr/testify/require"
)

func TestGolangImporter(t *testing.T) {

	i, err := NewGolangImporter("github.com/prometheus/prometheus/config", "/tmp/importers/golang/prometheus", "/Users/smintz/go/src")
	// i, err := NewGolangImporter("github.com/hashicorp/nomad/nomad/structs", "/tmp/importers/golang", "/Users/smintz/go/src")
	assert.NoError(t, err, "failed to create GolangImporter")
	importer := i.GetImporter()
	err = importer.SaveAll()
	assert.NoError(t, err, "failed to save files")
}
