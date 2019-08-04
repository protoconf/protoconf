package libprotoconf

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/any"
	protoconfvalue "protoconf.com/datatypes/proto/v1/protoconfvalue"
	"protoconf.com/protostdlib/secret"
	"protoconf.com/utils"
)

// Watcher enables getting updates on protoconf paths
type Watcher interface {
	Watch(path string, stopCh <-chan struct{}) (<-chan Result, error)
	Close()
}

// Result of the Watch operation or error
type Result struct {
	Value *any.Any
	Error error
}

func injectSecrets(value *protoconfvalue.ProtoconfValue) (*any.Any, error) {
	any := &any.Any{TypeUrl: value.GetValue().GetTypeUrl()}
	bytes := value.GetValue().GetValue()

	secretsSorted := make([]*protoconfvalue.SecretMetadata, len(value.GetSecrets()))
	copy(secretsSorted, value.GetSecrets())
	sort.Slice(secretsSorted, func(i int, j int) bool {
		return secretsSorted[i].GetPos() < secretsSorted[j].GetPos()
	})

	offset := 0
	for _, secretMetadata := range secretsSorted {
		pos := int(secretMetadata.GetPos()) + offset
		length := int(secretMetadata.GetLen())
		unarmedSecret := &secret.Secret{}
		if err := proto.Unmarshal(bytes[pos:pos+length], unarmedSecret); err != nil {
			return nil, errors.New("error unmarshaling secret")
		}

		armedSecret, err := armSecret(unarmedSecret)
		if err != nil {
			if unarmedSecret.GetIgnoreIfMissing() {
				continue
			}
			return nil, fmt.Errorf("error arming secret err=%s", err)
		}

		data, err := proto.Marshal(armedSecret)
		if err != nil {
			return nil, errors.New("error marshaling secret")
		}

		oldLength := len(bytes)
		if bytes, err = utils.ReplaceProtoBytes(bytes, pos, length, data); err != nil {
			return nil, fmt.Errorf("error replacing proto bytes err=%s", err)
		}
		offset += len(bytes) - oldLength
	}
	any.Value = bytes
	return any, nil
}

func armSecret(unarmedSecret *secret.Secret) (*secret.Secret, error) {
	switch t := unarmedSecret.GetSecret().(type) {
	case *secret.Secret_Value:
		return unarmedSecret, nil
	case *secret.Secret_CharSecret:
		return &secret.Secret{
			Secret: &secret.Secret_Value{Value: strings.Repeat(t.CharSecret.GetChar(), int(t.CharSecret.GetLength()))},
		}, nil
	case nil:
		return nil, errors.New("Secret.secret was not set")
	default:
		return nil, fmt.Errorf("Secret.secret has unknown type=%T", t)
	}
}
