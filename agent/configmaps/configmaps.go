// Package template contains the Example store implementation.
// This package is a template only.
package configmaps

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"github.com/kvtools/valkeyrie"
	"github.com/kvtools/valkeyrie/store"
	cv1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/retry"
)

// StoreName the name of the store.
// TODO implement me.
const StoreName = "configmaps"

// registers Example to Valkeyrie.
func init() {
	valkeyrie.Register(StoreName, newStore)
}

// Config the Example configuration.
// TODO implement me.
type Config struct {
	Namespace string
	Create    bool
}

func newStore(ctx context.Context, endpoints []string, options valkeyrie.Config) (store.Store, error) {
	cfg, ok := options.(*Config)
	if !ok && options != nil {
		return nil, &store.InvalidConfigurationError{Store: StoreName, Config: options}
	}

	return New(ctx, endpoints, cfg)
}

// Store implements the store.Store interface.
// TODO implement me.
type Store struct {
	clientset *kubernetes.Clientset
	config    *Config
	mutex     *sync.Mutex
	logger    *slog.Logger
}

func getClientset() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		slog.Default().Debug("could not get in cluster config", slog.String("error", err.Error()))
		kubeconfig :=
			clientcmd.NewDefaultClientConfigLoadingRules().GetDefaultFilename()
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {

			return nil, err
		}
	}
	return kubernetes.NewForConfig(config)
}

// New creates a new Example client.
// TODO implement me.
func New(ctx context.Context, endpoints []string, options *Config) (*Store, error) {
	clientset, err := getClientset()
	if err != nil {
		return nil, err
	}
	if options.Namespace == "" {
		options.Namespace = defaultNamespace()
	}
	logger := slog.Default()
	logger = logger.With(slog.String("namespace", options.Namespace))
	return &Store{
		clientset: clientset,
		config:    options,
		mutex:     &sync.Mutex{},
		logger:    logger,
	}, nil
}

func defaultNamespace() string {
	// This way assumes you've set the POD_NAMESPACE environment variable using the downward API.
	// This check has to be done first for backwards compatibility with the way InClusterConfig was originally set up
	if ns, ok := os.LookupEnv("POD_NAMESPACE"); ok {
		return ns
	}

	// Fall back to the namespace associated with the service account token, if available
	if data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		if ns := strings.TrimSpace(string(data)); len(ns) > 0 {
			return ns
		}
	}

	return "default"
}

// Put a value at the specified key.
func (s *Store) Put(ctx context.Context, key string, value []byte, opts *store.WriteOptions) error {
	filename := filepath.Base(key)
	configMapName := strings.ToLower(
		strings.ReplaceAll(
			strings.ReplaceAll(filepath.Dir(key), "/", "---"), "_", "--"),
	)
	cmClient := s.clientset.CoreV1().ConfigMaps(s.config.Namespace)
	cm, err := cmClient.Get(ctx, configMapName, v1.GetOptions{})
	if err != nil {
		cm.ObjectMeta.Name = configMapName
		retryableFn := func(error) bool {
			return true
		}
		retryErr := retry.OnError(retry.DefaultBackoff, retryableFn, func() error {
			cm, err = cmClient.Create(ctx, cm, v1.CreateOptions{})
			return err
		})
		if retryErr != nil {
			return retryErr
		}
	}
	if cm.Data == nil {
		cm.Data = map[string]string{}
	}
	cm.Data[filename] = string(value)

	_, err = cmClient.Update(
		ctx, cm, v1.UpdateOptions{},
	)
	return err
}

// Get a value given its key.
func (s *Store) Get(ctx context.Context, key string, opts *store.ReadOptions) (*store.KVPair, error) {
	logger := s.logger.With(slog.String("key", key))
	logger.Debug("getting key")
	filename := filepath.Base(key)
	configMapName := strings.ToLower(
		strings.ReplaceAll(
			strings.ReplaceAll(filepath.Dir(key), "/", "---"), "_", "--"),
	)
	cmClient := s.clientset.CoreV1().ConfigMaps(s.config.Namespace)

	retryableFn := func(error) bool {
		return true
	}
	var c *cv1.ConfigMap
	var err error
	retryErr := retry.OnError(retry.DefaultBackoff, retryableFn, func() error {
		c, err = cmClient.Get(ctx, configMapName, v1.GetOptions{})
		return err
	})
	logger.Debug("got configmap", slog.Any("error", retryErr))
	if retryErr != nil {
		return nil, retryErr
	}

	if value, ok := c.Data[filename]; ok {
		return &store.KVPair{Key: key, Value: []byte(value)}, nil
	}
	if value, ok := c.Data[filename]; ok {
		return &store.KVPair{Key: key, Value: []byte(value)}, nil
	}
	return nil, fmt.Errorf("could not find key %s", key)
}

// Delete the value at the specified key.
func (s Store) Delete(ctx context.Context, key string) error {
	filename := filepath.Base(key)
	configMapName := strings.ToLower(
		strings.ReplaceAll(
			strings.ReplaceAll(filepath.Dir(key), "/", "---"), "_", "--"),
	)
	cmClient := s.clientset.CoreV1().ConfigMaps(s.config.Namespace)
	retryableFn := func(error) bool {
		return true
	}
	var cm *cv1.ConfigMap
	var err error
	retryErr := retry.OnError(retry.DefaultBackoff, retryableFn, func() error {
		cm, err = cmClient.Get(ctx, configMapName, v1.GetOptions{})
		return err
	})
	if retryErr != nil {
		return retryErr
	}
	delete(cm.Data, filename)
	retryErr = retry.OnError(retry.DefaultBackoff, retryableFn, func() error {
		_, err = cmClient.Update(
			ctx, cm, v1.UpdateOptions{},
		)
		return err
	})
	if retryErr != nil {
		return retryErr
	}
	return err
}

// Exists Verify if a Key exists in the store.
func (s Store) Exists(ctx context.Context, key string, opts *store.ReadOptions) (bool, error) {
	// TODO implement me
	return true, nil
}

// Watch for changes on a key.
func (s *Store) Watch(ctx context.Context, key string, opts *store.ReadOptions) (<-chan *store.KVPair, error) {
	logger := s.logger.With(slog.String("key", key))
	logger.Debug("watching key")
	cmClient := s.clientset.CoreV1().ConfigMaps(s.config.Namespace)
	retryableFn := func(error) bool {
		return true
	}
	var c watch.Interface
	var err error
	retryErr := retry.OnError(retry.DefaultBackoff, retryableFn, func() error {
		c, err = cmClient.Watch(ctx, v1.ListOptions{FieldSelector: ""})
		return err
	})
	if retryErr != nil {
		return nil, retryErr
	}
	if err != nil {
		return nil, err
	}
	ch := make(chan *store.KVPair)
	kv, err := s.Get(ctx, key, &store.ReadOptions{})
	if err != nil {
		logger.With(slog.Any("error", err)).Error("error getting initial key")
	}

	if kv != nil {
		go func() {
			ch <- kv
		}()
	}
	go func() {
		var previousKv *store.KVPair
		for {
			select {
			case <-ctx.Done():
				err := context.Cause(ctx)
				logger.Info("context done", "cause", err)
				return
			case event := <-c.ResultChan():
				logger.Debug("got modified event", "event_type", event.Type)
				kv, err := s.Get(ctx, key, &store.ReadOptions{})
				if err != nil {
					logger.With(slog.Any("error", err)).Error("error getting updated key")
				}
				if reflect.DeepEqual(previousKv, kv) {
					continue
				}
				previousKv = kv

				if kv != nil {
					go func() {
						ch <- kv
					}()
				}
			}
		}
	}()

	return ch, nil
}

// WatchTree watches for changes on child nodes under a given directory.
func (s Store) WatchTree(ctx context.Context, directory string, opts *store.ReadOptions) (<-chan []*store.KVPair, error) {
	// TODO implement me
	panic("implement me")
}

// NewLock creates a lock for a given key.
// The returned Locker is not held and must be acquired with `.Lock`.
// The Value is optional.
func (s Store) NewLock(ctx context.Context, key string, opts *store.LockOptions) (store.Locker, error) {
	// TODO implement me
	panic("implement me")
}

// List the content of a given prefix.
func (s Store) List(ctx context.Context, directory string, opts *store.ReadOptions) ([]*store.KVPair, error) {
	// TODO implement me
	panic("implement me")
}

// DeleteTree deletes a range of keys under a given directory.
func (s Store) DeleteTree(ctx context.Context, directory string) error {
	// TODO implement me
	panic("implement me")
}

// AtomicPut Atomic CAS operation on a single value.
// Pass previous = nil to create a new key.
func (s Store) AtomicPut(ctx context.Context, key string, value []byte, previous *store.KVPair, opts *store.WriteOptions) (bool, *store.KVPair, error) {
	// TODO implement me
	panic("implement me")
}

// AtomicDelete Atomic delete of a single value.
func (s Store) AtomicDelete(ctx context.Context, key string, previous *store.KVPair) (bool, error) {
	// TODO implement me
	panic("implement me")
}

// Close the store connection.
func (s Store) Close() error {
	// TODO implement me
	panic("implement me")
}
