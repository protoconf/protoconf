package starproto

import (
	"fmt"

	"github.com/jhump/protoreflect/desc"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
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
func (v *starProtoEnumValue) CompareSameType(op syntax.Token, y starlark.Value, depth int) (bool, error) {
	// false means no diff
	n := y.(*starProtoEnumValue)
	if v.desc.GetEnum() != n.desc.GetEnum() {
		return true, fmt.Errorf("enums %v and %v are not from the same type", v.desc.GetEnum().GetName(), n.desc.GetEnum().GetName())
	}
	switch op {
	case syntax.EQL:
		return (v.desc.GetNumber() == n.desc.GetNumber()), nil
	case syntax.NEQ:
		return (v.desc.GetNumber() != n.desc.GetNumber()), nil
	case syntax.GT:
		return (v.desc.GetNumber() > n.desc.GetNumber()), nil
	case syntax.GE:
		return (v.desc.GetNumber() >= n.desc.GetNumber()), nil
	case syntax.LT:
		return (v.desc.GetNumber() < n.desc.GetNumber()), nil
	case syntax.LE:
		return (v.desc.GetNumber() <= n.desc.GetNumber()), nil
	}
	return false, nil
}
