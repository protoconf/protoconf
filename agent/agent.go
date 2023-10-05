package agent

import (
	"errors"
	"log"

	protoconfservice "github.com/protoconf/protoconf/agent/api/proto/v1"
	"github.com/protoconf/protoconf/libprotoconf"
)

type server struct {
	watcher libprotoconf.Watcher
	protoconfservice.ProtoconfServiceServer
}

func (s server) SubscribeForConfig(request *protoconfservice.ConfigSubscriptionRequest, srv protoconfservice.ProtoconfService_SubscribeForConfigServer) error {
	path := request.GetPath()
	log.Printf("Watching path=%s", path)

	stopCh := make(chan struct{})
	watchCh, err := s.watcher.Watch(path, stopCh)
	if err != nil {
		return err
	}

	defer func() {
		select {
		case stopCh <- struct{}{}:
		default:
		}
		close(stopCh)
	}()

	ctx := srv.Context()
	for {
		select {
		case <-ctx.Done():
			log.Printf("Client stopped watching path=%s", path)
			return ctx.Err()
		case config, ok := <-watchCh:
			if !ok {
				log.Printf("Watch channel closed for path=%s", path)
				return errors.New("watch channel closed")
			}

			if config.Error != nil {
				log.Printf("Error watching config, path=%s err=%s", path, config.Error)
				return config.Error
			}

			log.Printf("Sending update on path=%s", path)
			resp := protoconfservice.ConfigUpdate{Value: config.Value}
			go func() {
				if err := srv.Send(&resp); err != nil {
					log.Printf("Error sending config update, path=%s srv=%s err=%s", path, srv, err)
				} else {
					log.Printf("Update sent successfully path=%s", path)
				}
			}()
		}
	}
}
