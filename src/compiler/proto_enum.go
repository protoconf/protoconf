package main

import (
	"fmt"
	"sort"

	"github.com/jhump/protoreflect/desc"
	"go.starlark.net/starlark"
)

type starProtoEnumType struct {
	desc *desc.EnumDescriptor
}

var _ starlark.HasAttrs = (*starProtoEnumType)(nil)

func (t *starProtoEnumType) String() string {
	return fmt.Sprintf("<proto.EnumType %q>", t.desc.GetName())
}
func (t *starProtoEnumType) Type() string         { return "proto.EnumType" }
func (t *starProtoEnumType) Freeze()              {}
func (t *starProtoEnumType) Truth() starlark.Bool { return starlark.True }
func (t *starProtoEnumType) Hash() (uint32, error) {
	return 0, fmt.Errorf("unhashable type: %s", t.Type())
}

func (t *starProtoEnumType) Attr(attrName string) (starlark.Value, error) {
	if value := t.desc.FindValueByName(attrName); value != nil {
		return &starProtoEnumValue{desc: value}, nil
	}
	return nil, nil
}

func (t *starProtoEnumType) AttrNames() []string {
	names := make([]string, 0, len(t.desc.GetValues()))
	for _, value := range t.desc.GetValues() {
		names = append(names, value.GetName())
	}
	sort.Strings(names)
	return names
}

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

func enumFromStarlark(t *desc.EnumDescriptor, star *starProtoEnumValue) (int32, error) {
	// FIXME: is pointer equality appropriate here? Consider GetFullyQualifiedName() string
	if t != star.desc.GetEnum() {
		return 0, fmt.Errorf("type error: value %s can't be assigned to type `%s'", star, t)
	}
	return star.desc.GetNumber(), nil
}
