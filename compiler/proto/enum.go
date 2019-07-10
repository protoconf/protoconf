package proto

import (
	"fmt"

	"github.com/jhump/protoreflect/desc"
	"go.starlark.net/starlark"
)

type starProtoEnumValue struct {
	desc *desc.EnumValueDescriptor
}

func (v *starProtoEnumValue) String() string {
	return fmt.Sprintf("proto.Enum <%s %s=%d>", v.desc.GetEnum().GetName(), v.desc.GetName(), v.desc.GetNumber())
}
func (v *starProtoEnumValue) Type() string         { return v.desc.GetEnum().GetName() }
func (v *starProtoEnumValue) Freeze()              {}
func (v *starProtoEnumValue) Truth() starlark.Bool { return starlark.True }
func (v *starProtoEnumValue) Hash() (uint32, error) {
	return starlark.MakeInt64(int64(v.desc.GetNumber())).Hash()
}
