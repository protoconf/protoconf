package meta

import (
	"github.com/jhump/protoreflect/desc/builder"
)

// MetaFile creates a `meta.proto` file which includes meta arguments for
// terraform resources.
// https://www.terraform.io/docs/configuration/resources.html#meta-arguments
func MetaFile() *builder.FileBuilder {
	file := builder.NewFile("terraform/v1/meta.proto")
	file.SetProto3(true)
	file.SetPackageName("terraform.v1")

	metaMsg := builder.NewMessage("MetaFields")

	// depends_on
	fieldDependsOn := builder.NewField("depends_on", builder.FieldTypeString()).SetRepeated().SetJsonName("depends_on")
	metaMsg.AddField(fieldDependsOn)

	// count
	fieldCount := builder.NewField("count", builder.FieldTypeInt32())
	metaMsg.AddField(fieldCount)

	// provider
	fieldProvider := builder.NewField("provider", builder.FieldTypeString())
	metaMsg.AddField(fieldProvider)

	// lifecycle
	lifecycleMsg := builder.NewMessage("Lifecycle")
	fieldCreateBeforeDestroy := builder.NewField("create_before_destroy", builder.FieldTypeBool()).SetJsonName("create_before_destroy")
	fieldPreventDestroy := builder.NewField("prevent_destroy", builder.FieldTypeBool()).SetJsonName("prevent_destroy")
	fieldIgnoreChanges := builder.NewField("ignore_changes", builder.FieldTypeString()).SetRepeated().SetJsonName("ignore_changes")
	lifecycleMsg.AddField(fieldCreateBeforeDestroy)
	lifecycleMsg.AddField(fieldPreventDestroy)
	lifecycleMsg.AddField(fieldIgnoreChanges)

	// provisioner

	// connection

	file.AddMessage(metaMsg)
	file.AddMessage(lifecycleMsg)
	return file
}
