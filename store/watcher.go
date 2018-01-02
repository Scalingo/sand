package store

import (
	"context"
	"io"

	"github.com/Scalingo/go-internal-tools/logger"
	"github.com/Scalingo/sand/etcd"
	"github.com/coreos/etcd/clientv3"
	"github.com/pkg/errors"
)

type Watcher interface {
	NextResponse() (clientv3.WatchResponse, bool)
	io.Closer
}

func (s *store) Watch(ctx context.Context, prefix string) (Watcher, error) {
	log := logger.Get(ctx)
	prefix = s.Key(prefix)

	client, err := etcd.NewClient(s.config)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to create etcd client")
	}

	wc := clientv3.NewWatcher(client)
	log.Infof("create endpoints watcher on prefix %v", prefix)

	// Use context.Background() to avoid the resulting chan to be closed at the end of a HTTP request
	// https://godoc.org/github.com/coreos/etcd/clientv3#Watcher
	wchan := wc.Watch(context.Background(), prefix, clientv3.WithPrefix())

	return watcher{etcdClient: client, etcdWatcher: wc, etcdWatchChan: wchan, ready: make(chan struct{})}, nil
}

type watcher struct {
	etcdClient    *clientv3.Client
	etcdWatcher   clientv3.Watcher
	etcdWatchChan clientv3.WatchChan
	ready         chan struct{}
}

func (w watcher) NextResponse() (clientv3.WatchResponse, bool) {
	res, ok := <-w.etcdWatchChan
	return res, ok
}

func (w watcher) Close() error {
	err := w.etcdWatcher.Close()
	if err != nil {
		return err
	}
	err = w.etcdClient.Close()
	if err != nil {
		return err
	}
	return nil
}
