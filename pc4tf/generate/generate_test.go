package generate

import (
	"fmt"
	"log"
	"os/user"
	"path/filepath"
	"runtime"
	"testing"

	// "github.com/jhump/protoreflect/desc/builder"
	assert "github.com/stretchr/testify/require"
)

func TestGenerate(t *testing.T) {
	u, err := user.Current()
	assert.NoError(t, err)
	dir_postfix := filepath.Join(u.HomeDir, ".terraform.d", "plugins", fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH))
	dir, err := filepath.Abs(dir_postfix)
	g := NewGenerator(dir, "/tmp/test")
	err = g.PopulateProviders()
	assert.NoError(t, err)
	for name, _ := range g.Providers {
		log.Println("found", name)
	}
	assert.Equal(t, "", "")
	err = g.Save()
	assert.NoError(t, err)
}
