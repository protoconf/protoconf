package libprotoconf

import (
	"github.com/golang/protobuf/ptypes/any"
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
