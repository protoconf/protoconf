package utils

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	_ "github.com/bufbuild/protovalidate-go"
	_ "github.com/bufbuild/protovalidate-go/legacy"
	"github.com/jhump/protoreflect/dynamic"
	_ "github.com/protoconf/protoconf/pb/protoconf/v1"

	"github.com/protoconf/protoconf/consts"
	"github.com/protoconf/protoconf/utils/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDescriptorRegistry(t *testing.T) {
	registry := NewDescriptorRegistry()
	largePath := filepath.Join(testdata.SmallTestDir(), consts.SrcPath)
	err := registry.Import(registry.Parse, []*regexp.Regexp{}, largePath)
	require.NoError(t, err)

	files := registry.GetFilesResolver()
	_, err = files.FindFileByPath("test.proto")
	require.NoError(t, err)

	types := registry.GetTypesResolver(files)
	_, err = types.FindMessageByName("test.v1.TestMessage")
	require.NoError(t, err)

	desc, err := registry.MessageRegistry.FindMessageTypeByUrl("type.googleapis.com/test.v1.TestMessage")
	assert.NoError(t, err)
	assert.NotNil(t, desc)
	assert.NotNil(t, dynamic.NewMessage(desc))

	storeFile := filepath.Join(os.TempDir(), "data.fds")
	h, err := registry.Store(storeFile)
	require.NoError(t, err)

	newReg := NewDescriptorRegistry()
	err = newReg.Load(storeFile, "baddd")
	require.Error(t, err)
	err = newReg.Load(storeFile, h)
	require.NoError(t, err)

}
