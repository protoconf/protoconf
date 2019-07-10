package libprotoconf

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
	"protoconf.com/consts"
	"protoconf.com/utils"
)

// NewFileWatcher creates a new file-backed protoconf watcher
func NewFileWatcher(protoconfRoot string) (Watcher, error) {
	fsnotifyWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	absRoot, err := filepath.Abs(protoconfRoot)
	if err != nil {
		return nil, err
	}

	watcher := &fileWatcher{
		fsnotifyWatcher: fsnotifyWatcher,
		protoconfRoot:   absRoot,
		watches:         make(map[string]([]chan struct{})),
	}

	go watcher.readEvents()

	return watcher, nil
}

type fileWatcher struct {
	fsnotifyWatcher *fsnotify.Watcher
	protoconfRoot   string
	watches         map[string]([]chan struct{})
	lock            sync.Mutex
}

// Watch a value given its path
func (w *fileWatcher) Watch(path string, stopCh <-chan struct{}) (<-chan Result, error) {
	if path != filepath.ToSlash(filepath.Clean(path)) || path == "" {
		return nil, fmt.Errorf("invalid path to watch, path=%s", path)
	}

	absPath := filepath.Join(w.protoconfRoot, consts.CompiledConfigPath, path+consts.CompiledConfigExtension)
	fsCh := make(chan struct{})
	if err := w.addWatch(absPath, fsCh); err != nil {
		return nil, err
	}

	watchCh := make(chan Result)
	go func() {
		defer close(watchCh)
		for {
			protoconfValue, err := utils.ReadConfig(w.protoconfRoot, path)
			if err != nil {
				watchCh <- Result{nil, err}
				return
			}
			watchCh <- Result{protoconfValue.GetValue(), nil}
			select {
			case _, ok := <-fsCh:
				if !ok {
					return
				}
			case <-stopCh:
				_ = w.removeWatch(absPath, fsCh)
				return
			}
		}
	}()

	return watchCh, nil
}

func (w *fileWatcher) addWatch(path string, ch chan struct{}) error {
	w.lock.Lock()
	defer w.lock.Unlock()

	w.watches[path] = append(w.watches[path], ch)
	if err := w.fsnotifyWatcher.Add(path); err != nil {
		w.watches[path] = removeChannel(ch, w.watches[path])
		return fmt.Errorf("error fs watching path %s, err=%s", path, err)
	}

	return nil
}

func (w *fileWatcher) removeWatch(path string, ch chan struct{}) error {
	w.lock.Lock()
	defer w.lock.Unlock()

	w.watches[path] = removeChannel(ch, w.watches[path])
	if len(w.watches[path]) == 0 {
		return w.fsnotifyWatcher.Remove(path)
	}

	return nil
}

func removeChannel(ch chan struct{}, chs []chan struct{}) []chan struct{} {
	for idx, val := range chs {
		if val == ch {
			chs[idx] = chs[len(chs)-1]
			chs[len(chs)-1] = nil
			return chs[:len(chs)-1]
		}
	}
	return chs
}

func (w *fileWatcher) readEvents() {
	for {
		select {
		case event, ok := <-w.fsnotifyWatcher.Events:
			if !ok {
				w.closeWatchers()
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				for _, channel := range w.watches[event.Name] {
					channel <- struct{}{}
				}
			}
		case <-w.fsnotifyWatcher.Errors:
			w.closeWatchers()
		}
	}
}

func (w *fileWatcher) closeWatchers() {
	for _, pathWatches := range w.watches {
		for _, watch := range pathWatches {
			close(watch)
		}
	}
	w.watches = nil
}

func (w *fileWatcher) Close() {
	w.closeWatchers()
	w.fsnotifyWatcher.Close()
}
