package lib

import (
	"fmt"

	pbproto "github.com/golang/protobuf/proto"
	dpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/protoconf/protoconf/compiler/proto"
	"github.com/protoconf/protoconf/protostdlib/secret"
	"github.com/protoconf/protoconf/utils"
	"go.starlark.net/starlark"
)

type config struct {
	filename   string
	locals     starlark.StringDict
	validators map[string]*starlark.Function
}

func (c *config) main() (starlark.Value, error) {
	mainVal, ok := c.locals["main"]
	if !ok {
		return nil, fmt.Errorf("no `main' function found in %s", c.filename)
	}
	main, ok := mainVal.(starlark.Callable)
	if !ok {
		return nil, fmt.Errorf("`main' must be a function (got a %s)", mainVal.Type())
	}

	thread := &starlark.Thread{
		Print: starPrint,
	}

	mainVal, err := starlark.Call(thread, main, nil, nil)
	if err != nil {
		return nil, err
	}
	return mainVal, nil
}

func (c *config) validate(value interface{}) error {
	message, ok := value.(*dynamic.Message)
	if !ok {
		if _, ok := value.(pbproto.Message); ok {
			return nil
		}
		return fmt.Errorf("expecting a proto message to validate, got=%v", value)
	}

	if message == nil {
		return nil
	}

	if err := stdLibValidate(message); err != nil {
		return err
	}

	if validator, ok := c.validators[message.GetMessageDescriptor().GetFullyQualifiedName()]; ok {
		thread := &starlark.Thread{
			Print: starPrint,
		}
		args := starlark.Tuple([]starlark.Value{
			proto.NewStarProtoMessage(message),
		})
		if _, err := starlark.Call(thread, validator, args, nil); err != nil {
			return err
		}
	}

	for _, field := range message.GetMessageDescriptor().GetFields() {
		if field.GetType() != dpb.FieldDescriptorProto_TYPE_MESSAGE {
			continue
		}
		if field.IsMap() {
			mp := message.GetField(field).(map[interface{}]interface{})
			if field.GetMapKeyType().GetType() == dpb.FieldDescriptorProto_TYPE_MESSAGE {
				for key := range mp {
					if err := c.validate(key); err != nil {
						return err
					}
				}
			}
			if field.GetMapValueType().GetType() == dpb.FieldDescriptorProto_TYPE_MESSAGE {
				for _, value := range mp {
					if err := c.validate(value); err != nil {
						return err
					}
				}
			}
		} else if field.IsRepeated() {
			length := len(message.GetField(field).([]interface{}))
			for i := 0; i < length; i++ {
				if err := c.validate(message.GetRepeatedField(field, i)); err != nil {
					return err
				}
			}
		} else {
			if err := c.validate(message.GetField(field)); err != nil {
				return err
			}
		}
	}
	return nil
}

func stdLibValidate(message *dynamic.Message) error {
	fqn := message.GetMessageDescriptor().GetFullyQualifiedName()
	charSecret := &secret.CharSecret{}
	if fqn == utils.MessageFQN(charSecret) {
		if err := message.ConvertTo(charSecret); err != nil {
			return fmt.Errorf("error converting message to CharSecret, message=%s err=%s", message, err)
		}
		if len(charSecret.GetChar()) != 1 {
			return fmt.Errorf("CharSecret.char must have exactly one character, got char=%s", charSecret.GetChar())
		}
		if charSecret.GetLength() < 1 {
			return fmt.Errorf("CharSecret.length must be greater than 0, got length=%d", charSecret.GetLength())
		}
	}

	return nil
}
