package etcd

import (
	"context"

	"github.com/pkg/errors"
	"go.etcd.io/etcd/v3/clientv3"
)

type Watcher struct {
	client    *clientv3.Client
	watcher   clientv3.Watcher
	watchChan clientv3.WatchChan
}

func NewWatcher(prefix string) (Watcher, error) {
	client, err := NewClient()
	if err != nil {
		return Watcher{}, errors.Wrapf(err, "fail to create etcd client")
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

func (w Watcher) WatchChan() clientv3.WatchChan {
	return w.watchChan
}

func (w Watcher) Close() error {
	err := w.watcher.Close()
	if err != nil {
		return errors.Wrapf(err, "fail to close etcd watcher")
	}

	err = w.client.Close()
	if err != nil {
		return errors.Wrapf(err, "fail to close etcd client")
	}

	return nil
}
