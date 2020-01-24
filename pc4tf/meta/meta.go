package meta

import (
	"github.com/jhump/protoreflect/desc/builder"
)

// https://www.terraform.io/docs/configuration/resources.html#meta-arguments
func MetaFile() *builder.FileBuilder {
	file := builder.NewFile("terraform/meta.proto")
	file.SetProto3(true)
	file.SetPackageName("terraform.meta")

	metaMsg := builder.NewMessage("MetaFields")

	// depends_on
	fieldDependsOn := builder.NewMapField("depends_on", builder.FieldTypeString(), builder.FieldTypeString())
	metaMsg.AddField(fieldDependsOn)

	// count
	fieldCount := builder.NewField("count", builder.FieldTypeInt32())
	metaMsg.AddField(fieldCount)

	// for_each
	fieldForEach := builder.NewMapField("for_each", builder.FieldTypeString(), builder.FieldTypeString())
	metaMsg.AddField(fieldForEach)

	// provider
	fieldProvider := builder.NewField("provider", builder.FieldTypeString())
	metaMsg.AddField(fieldProvider)

	// lifecycle
	lifecycleMsg := builder.NewMessage("Lifecycle")
	fieldCreateBeforeDestroy := builder.NewField("create_before_destroy", builder.FieldTypeBool())
	fieldPreventDestroy := builder.NewField("prevent_destroy", builder.FieldTypeBool())
	fieldIgnoreChanges := builder.NewField("ignore_changes", builder.FieldTypeString()).SetRepeated()
	lifecycleMsg.AddField(fieldCreateBeforeDestroy)
	lifecycleMsg.AddField(fieldPreventDestroy)
	lifecycleMsg.AddField(fieldIgnoreChanges)

	// provisioner

	// connection

	file.AddMessage(metaMsg)
	file.AddMessage(lifecycleMsg)
	return file
}
