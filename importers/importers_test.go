package importers

import (
	"os"
	"testing"

	"github.com/jhump/protoreflect/desc/builder"
	assert "github.com/stretchr/testify/require"
)

func TestImporters(t *testing.T) {
	dir, err := os.MkdirTemp("", "generate_test")
	assert.NoError(t, err, "failed to create tmp folder")
	i := NewImporter("master/package/v1/master.proto", dir)

	subFile := builder.NewFile("subfile.proto")
	subFile.SetPackageName("subfile")
	subMsg := builder.NewMessage("SubMessage")
	err = subFile.TryAddMessage(subMsg)
	assert.NoError(t, err, "failed to add subMsg to subFile")
	i.RegisterFile(subFile)

	masterMsg := builder.NewMessage("MasterMsg")
	field := builder.NewField("sub_message", builder.FieldTypeMessage(subMsg))
	err = masterMsg.TryAddField(field)
	assert.NoError(t, err, "failed to add field to masterMsg")
	err = i.MasterFile.TryAddMessage(masterMsg)
	assert.NoError(t, err, "failed to add masterMsg to MasterFile")
	err = i.SaveAll()
	assert.NoError(t, err, "failed to save all files")
}
