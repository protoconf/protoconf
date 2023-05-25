package exec

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ghodss/yaml"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/pkg/errors"
	pc "github.com/protoconf/protoconf/agent/api/proto/v1"
	exec_config "github.com/protoconf/protoconf/exec/config"
	"github.com/protoconf/protoconf/utils"
	"go.uber.org/zap"
)

type watcher struct {
	config   *exec_config.WatcherConfig
	logger   *zap.Logger
	client   pc.ProtoconfServiceClient
	executor *Executor
	mutex    sync.Mutex
}

func newWatcher(config *exec_config.WatcherConfig, executor *Executor, client pc.ProtoconfServiceClient, logger *zap.Logger) *watcher {
	return &watcher{
		config:   config,
		logger:   logger.With(zap.String("item", config.Path)),
		client:   client,
		executor: executor,
	}
}

// SetConfig allow updating the watcher config after initiated
func (w *watcher) SetConfig(config *exec_config.WatcherConfig) {
	w.config = config
}

// Start watches for changes on a specific config
func (w *watcher) Start(ctx context.Context) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	stream, err := w.client.SubscribeForConfig(ctx, &pc.ConfigSubscriptionRequest{Path: w.config.Path})
	if err != nil {
		return err
	}
	updateCh := make(chan *pc.ConfigUpdate)
	errCh := make(chan error)
	go func() {
		for {
			w.logger.Info("waiting for update")
			update, err := stream.Recv()

			if err == io.EOF {
				errCh <- errors.Errorf("Connection closed while streaming config path=%s", w.config.Path)
			}

			if err != nil {
				errCh <- errors.Errorf("Error while streaming config path=%s err=%v", w.config.Path, err)
			}
			w.logger.Info("got update")
			updateCh <- update
		}
	}()
	for {
		select {
		case <-ctx.Done():
			w.logger.Info("got context done signal")
			return nil
		case err := <-errCh:
			return err
		case update, ok := <-updateCh:
			if !ok {
				break
			}

			value := update.GetValue()
			anyResolver, err := utils.LoadAnyResolver(w.executor.protosDir, w.config.ProtoFile)
			if err != nil {
				return errors.Wrap(err, "failed to get AnyResolver")
			}

			name, err := anyResolver.Resolve(value.GetTypeUrl())
			if err != nil {
				return errors.Wrapf(err, "could not find typeUrl for %s", value.GetTypeUrl())
			}

			msg, err := dynamic.AsDynamicMessage(name)
			if err != nil {
				return err
			}

			err = msg.Unmarshal(value.GetValue())
			if err != nil {
				return err
			}

			for _, action := range w.config.Actions {
				w.runAction(ctx, action, msg)
			}
			w.logger.Info("finished running actions")
		}
	}

}

func (w *watcher) runAction(ctx context.Context, action *exec_config.Action, msg *dynamic.Message) {
	var log *zap.Logger
	var actionErr error = nil
	switch x := action.Action.(type) {
	case *exec_config.Action_Http:
		log = w.logger.With(zap.String("action", "http"), zap.String("uri", x.Http.Uri), zap.String("method", x.Http.Method))
		actionErr = w.runHTTPAction(ctx, x.Http, msg)
	case *exec_config.Action_Restart:
		log = w.logger.With(zap.String("action", "restart"))
	case *exec_config.Action_File:
		log = w.logger.With(zap.String("action", "file"), zap.String("filepath", x.File.Path))
		actionErr = w.runWriteAction(ctx, x.File, msg)
	case *exec_config.Action_Signal:
		log = w.logger.With(zap.String("action", "signal"), zap.String("pid_file", x.Signal.PidFile), zap.String("signal", x.Signal.Signal.String()))
	}
	log.Debug("action finished")
	if actionErr != nil {
		log.Error("error running", zap.Error(actionErr))
		for _, a := range action.OnError {
			w.runAction(ctx, a, msg)
		}
	} else {
		for _, a := range action.Then {
			w.runAction(ctx, a, msg)
		}
	}
}

func (w *watcher) runHTTPAction(ctx context.Context, httpAction *exec_config.ActionTypeHttp, msg *dynamic.Message) error {
	logger := w.logger.With(zap.String("func", "runHTTPAction"))

	jsonBytes, err := msg.MarshalJSON()
	if err != nil {
		return errors.Wrap(err, "failed to marshal value")
	}
	logger.Debug("json content", zap.String("json", string(jsonBytes)))

	req, err := http.NewRequestWithContext(ctx, httpAction.Method, httpAction.Uri, strings.NewReader(string(jsonBytes)))
	if err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	newStr := buf.String()
	logger.Info("got response", zap.Int("code", resp.StatusCode), zap.String("content", newStr))
	return nil
}

func (w *watcher) runWriteAction(ctx context.Context, action *exec_config.ActionTypeWriteToFile, msg *dynamic.Message) error {
	logger := w.logger.With(zap.String("func", "runWriteAction"))
	var marshaledBytes []byte
	var marshalErr error = nil
	marshalerName := "unknown"
	switch action.GetSerializer() {
	case exec_config.Config_JSON:
		logger.Debug("json marshaler")
		marshalerName = "json"
		marshaledBytes, marshalErr = msg.MarshalJSONIndent()
	case exec_config.Config_YAML:
		logger.Debug("yaml marshaler")
		marshalerName = "yaml"
		b, err := msg.MarshalJSONIndent()
		if err != nil {
			return err
		}
		marshaledBytes, marshalErr = yaml.JSONToYAML(b)
	case exec_config.Config_PB:
		logger.Debug("protobuf text marshaler")
		marshalerName = "protobuf_text"
		marshaledBytes, marshalErr = msg.MarshalTextIndent()
	default:
		return errors.Errorf("could not find marshaler for %s", action.GetSerializer())
	}
	if marshalErr != nil {
		return marshalErr
	}
	bytesToWrite := append([]byte(action.GetHeader()), marshaledBytes...)
	bytesToWrite = append(bytesToWrite, []byte(action.GetFooter())...)
	err := os.MkdirAll(filepath.Dir(action.Path), 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}
	err = os.WriteFile(action.Path, bytesToWrite, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}
	logger.Debug("content", zap.String(marshalerName, string(bytesToWrite)))

	return nil
}
