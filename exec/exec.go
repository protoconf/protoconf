package exec

/*
This package is used to run and config software which is not built with protoconf support.
You can run `protoconf exec` (or `protoconf-exec`) daemon that could watch for various protoconf paths and act upon change
Possible outputs:
1. Serialize as JSON, YAML, TOML, INI or PB
2. Use go-template
Possible actions:
1. Restart (kill, then start)
2. Signal (send a signal to the running process)
3. Http command (post the content to an http path)
*/

import (
	"context"
	"io"
	"log"

	"github.com/golang/protobuf/ptypes"
	"github.com/pkg/errors"
	pc "github.com/protoconf/protoconf/agent/api/proto/v1"
	exec_config "github.com/protoconf/protoconf/exec/config"
	"go.uber.org/zap"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Executor is an instance that can run sub-process or react to changes in protoconf config path
type Executor struct {
	client    pc.ProtoconfServiceClient
	conn      *grpc.ClientConn
	path      string
	protosDir string
	logger    *zap.Logger
	watchers  map[string]*watcher
}

// NewExecutor returns an Executor instance
func NewExecutor(path string, protosDir string) (*Executor, error) {
	logger, err := zap.NewDevelopment()
	if err != nil {
		return nil, err
	}
	logger = logger.With(zap.String("path", path))
	client, conn := getProtoconfClient()
	executor := &Executor{
		path:      path,
		client:    client,
		conn:      conn,
		logger:    logger,
		protosDir: protosDir,
		watchers:  make(map[string]*watcher),
	}

	return executor, nil
}

// Start the executor loop
func (e *Executor) Start(ctx context.Context) error {
	e.logger.Info("starting executor")
	stream, err := e.client.SubscribeForConfig(ctx, &pc.ConfigSubscriptionRequest{Path: e.path})
	if err != nil {
		return err
	}

	econf := &exec_config.Config{}
	cancelGroup := []context.CancelFunc{}
	for {
		e.logger.Info("waiting for update")
		update, err := stream.Recv()
		e.logger.Info("got update")
		for _, cancel := range cancelGroup {
			cancel()
		}

		if err == io.EOF {
			return errors.Errorf("Connection closed while streaming config path=%s", e.path)
		}

		if st, failed := status.FromError(err); failed {
			if st.Code() == codes.Canceled {
				return nil
			}
		}

		if err != nil {
			return errors.Errorf("Error while streaming config path=%s err=%v", e.path, err)
		}

		err = ptypes.UnmarshalAny(update.GetValue(), econf)
		if err != nil {
			return err
		}

		for _, w := range econf.Items {
			myfunc := func() {
				e.logger.Info("found item", zap.String("item", w.Path))

				watcher := e.watcher(w)
				mctx, cancel := context.WithCancel(ctx)
				cancelGroup = append(cancelGroup, cancel)
				err = watcher.Start(mctx)
				if err != nil {
					e.logger.Info("error", zap.Error(err))
				}
				e.logger.Debug("watcher finished", zap.String("item", w.Path))
			}
			go myfunc()
		}
		e.logger.Debug("finished starting watchers")
	}

}

func (e *Executor) watcher(config *exec_config.WatcherConfig) *watcher {
	if watcher, ok := e.watchers[config.Path]; ok {
		watcher.SetConfig(config)
		return watcher
	}
	e.watchers[config.Path] = newWatcher(config, e, e.client, e.logger)
	return e.watcher(config)
}

// Close will close the grpc connection
func (e *Executor) Close() {
	e.conn.Close()
}

func getProtoconfClient() (pc.ProtoconfServiceClient, *grpc.ClientConn) {
	address := "localhost:4300"
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Error connecting to server address=%s err=%v", address, err)
	}
	c := pc.NewProtoconfServiceClient(conn)
	return c, conn
}
