package etcd

import (
	"context"
	"fmt"
	"service-discovery/discovery"

	clientv3 "go.etcd.io/etcd/client/v3"
)

var (
	_ discovery.Watcher = (*watcher)(nil)
)

type watcher struct {
	client    *clientv3.Client
	watcher   clientv3.Watcher
	namespace string
	name      string
}

func newWatcher(ctx context.Context, namespace string, name string, client *clientv3.Client) (*watcher, error) {
	w := &watcher{
		client:    client,
		namespace: namespace,
		name:      name,
		watcher:   clientv3.NewWatcher(client),
	}

	return w, nil
}

func (r *watcher) Watch(ctx context.Context) <-chan struct{} {
	key := fmt.Sprintf("%s/%s", r.namespace, r.name)

	etcdWatcherCh := r.watcher.Watch(ctx, key, clientv3.WithPrefix())
	ch := make(chan struct{})
	go func(ctx context.Context, etcdCh clientv3.WatchChan, retCh chan struct{}) {
		for {
			select {
			case watchResp, ok := <-etcdWatcherCh:
				if !ok || watchResp.Canceled {
					close(retCh)
					return
				}
				retCh <- struct{}{}
			case <-ctx.Done():
				close(retCh)
				return
			}
		}
	}(ctx, etcdWatcherCh, ch)

	return ch
}

func (r *watcher) Stop() error {
	return r.watcher.Close()
}
