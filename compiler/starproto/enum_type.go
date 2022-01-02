package starproto

import (
	"fmt"
	"sort"

	"github.com/jhump/protoreflect/desc"
	"go.starlark.net/starlark"
)

type starProtoEnumType struct {
	desc *desc.EnumDescriptor
}

func (t *starProtoEnumType) String() string {
	return fmt.Sprintf("<proto.EnumType %s>", t.desc.GetName())
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

var _ starlark.HasAttrs = (*starProtoEnumType)(nil)
