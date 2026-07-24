package etcd

import (
	"context"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/Scalingo/go-utils/errors/v3"
)

type Watcher struct {
	client    *clientv3.Client
	watcher   clientv3.Watcher
	watchChan clientv3.WatchChan
}

func NewWatcher(ctx context.Context, prefix string) (Watcher, error) {
	client, err := NewClient(ctx)
	if err != nil {
		return Watcher{}, errors.Wrapf(ctx, err, "create etcd client")
	}
	wc := clientv3.NewWatcher(client)

	// Use context.Background() to avoid the resulting chan to be closed at the end of a HTTP request
	// https://godoc.org/go.etcd.io/etcd/v3/clientv3#Watcher
	wchan := wc.Watch(context.Background(), prefix, clientv3.WithPrefix())

	return Watcher{
		client:    client,
		watcher:   wc,
		watchChan: wchan,
	}, nil
}

func (w Watcher) WatchChan(ctx context.Context) clientv3.WatchChan {
	return w.watchChan
}

func (w Watcher) Close(ctx context.Context) error {
	err := w.watcher.Close()
	if err != nil {
		return errors.Wrapf(ctx, err, "close etcd watcher")
	}

	err = w.client.Close()
	if err != nil {
		return errors.Wrapf(ctx, err, "close etcd client")
	}

	return nil
}
