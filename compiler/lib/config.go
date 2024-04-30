package lib

import (
	"errors"
	"fmt"

	"github.com/jhump/protoreflect/dynamic"
	"github.com/jhump/protoreflect/dynamic/msgregistry"
	"github.com/protoconf/protoconf/compiler/starproto"
	"go.starlark.net/starlark"
	"google.golang.org/protobuf/proto"
	pbproto "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/protoadapt"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/bufbuild/protovalidate-go"
	"github.com/bufbuild/protovalidate-go/legacy"
)

type config struct {
	filename        string
	locals          starlark.StringDict
	validators      map[string]*starlark.Function
	messageRegistry msgregistry.MessageRegistry
	protoResolver   protoregistry.MessageTypeResolver
}

var (
	ErrMainNotCallable = errors.New("`main` must be a function")
	ErrMainNotFound    = errors.New("`main` function not found")
)

func (c *config) main() (starlark.Value, error) {
	mainVal, ok := c.locals["main"]
	if !ok {
		return nil, errors.Join(ErrMainNotFound, fmt.Errorf("in file: %s", c.filename))
	}
	main, ok := mainVal.(starlark.Callable)
	if !ok {
		return nil, errors.Join(ErrMainNotCallable, fmt.Errorf("(got a %s)", mainVal.Type()))
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

var (
	ErrInvalidConfig = errors.New("config is not valid")
)

func (c *config) validate(value interface{}) error {
	var message *dynamic.Message
	switch result := value.(type) {
	case *anypb.Any:
		if result == nil {
			return nil
		}
		md, err := c.messageRegistry.FindMessageTypeByUrl(result.GetTypeUrl())

		if errors.Is(err, &msgregistry.ErrUnexpectedType{URL: result.GetTypeUrl()}) {
			return nil
		}
		if err != nil {
			return err
		}
		message = dynamic.NewMessage(md)
		err = message.Unmarshal(result.GetValue())
		if err != nil {
			return err
		}
	case pbproto.Message:
		m, err := dynamic.AsDynamicMessage(protoadapt.MessageV1Of(result))
		if err != nil {
			return err
		}
		message = m
	case *dynamic.Message:
		message = result
	default:
		return fmt.Errorf("expecting a proto message to validate, got=%v", value)
	}

	if message == nil {
		return nil
	}

	pbmsg := dynamicpb.NewMessage(message.GetMessageDescriptor().UnwrapMessage())
	b, _ := message.Marshal()
	proto.Unmarshal(b, pbmsg)

	validator, err := protovalidate.New(
		legacy.WithLegacySupport(legacy.ModeMerge),
		protovalidate.WithDescriptors(message.GetMessageDescriptor().UnwrapMessage()),
	)
	if err != nil {
		return err
	}
	err = validator.Validate(pbmsg)
	if err != nil {
		return errors.Join(ErrInvalidConfig, err)
	}

	if validator, ok := c.validators[message.GetMessageDescriptor().GetFullyQualifiedName()]; ok {
		thread := &starlark.Thread{
			Print: starPrint,
		}
		args := starlark.Tuple([]starlark.Value{
			starproto.NewStarProtoMessage(message),
		})
		if _, err := starlark.Call(thread, validator, args, nil); err != nil {
			return errors.Join(ErrInvalidConfig, err)
		}
	}

	for _, field := range message.GetMessageDescriptor().GetFields() {
		if field.GetType() != descriptorpb.FieldDescriptorProto_TYPE_MESSAGE {
			continue
		}
		if field.IsMap() {
			mp := message.GetField(field).(map[interface{}]interface{})
			if field.GetMapKeyType().GetType() == descriptorpb.FieldDescriptorProto_TYPE_MESSAGE {
				for key := range mp {
					if err := c.validate(key); err != nil {
						return err
					}
				}
			}
			if field.GetMapValueType().GetType() == descriptorpb.FieldDescriptorProto_TYPE_MESSAGE {
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
