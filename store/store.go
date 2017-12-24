package store

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"reflect"

	"github.com/Scalingo/go-internal-tools/logger"
	"github.com/Scalingo/networking-agent/config"
	"github.com/Scalingo/networking-agent/etcd"
	"github.com/coreos/etcd/clientv3"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Closer func() error

func (c Closer) Close() error {
	return c()
}

var ErrNotFound = errors.New("not found")

type Store interface {
	Get(ctx context.Context, key string, recursive bool, data interface{}) error
	Set(ctx context.Context, key string, data interface{}) error
	Delete(ctx context.Context, key string) error
	Watch(ctx context.Context, key string) (clientv3.WatchChan, io.Closer, error)
}

type store struct {
	config *config.Config
}

func New(c *config.Config) Store {
	return &store{config: c}
}

func (s *store) Get(ctx context.Context, key string, recursive bool, data interface{}) error {
	log := logger.Get(ctx).WithField("scope", "store")
	key = s.Key(key)
	c, closer, err := s.newEtcdClient()
	if err != nil {
		return errors.Wrap(err, "fail to build etcd client")
	}
	defer closer.Close()
	opts := []clientv3.OpOption{}
	if recursive {
		opts = append(opts, clientv3.WithPrefix())
	}
	res, err := c.Get(ctx, key, opts...)
	if err != nil {
		return errors.Wrapf(err, "fail to read key %v", key)
	}
	log.WithFields(logrus.Fields{"key": key, "nodes": len(res.Kvs)}).Debug("get key")

	if len(res.Kvs) == 0 {
		return ErrNotFound
	}

	// If call is recursive, we get an array of data and build the corresponding JSON
	content := res.Kvs[0].Value
	if reflect.TypeOf(data).Elem().Kind() == reflect.Slice {
		content = []byte{'['}
		for i, kv := range res.Kvs {
			content = append(content, kv.Value...)
			if i < len(res.Kvs)-1 {
				content = append(content, ',')
			}
		}

		content = append(content, ']')
	}
	return json.NewDecoder(bytes.NewReader(content)).Decode(&data)
}

func (s *store) Set(ctx context.Context, key string, data interface{}) error {
	log := logger.Get(ctx).WithField("scope", "store")
	key = s.Key(key)
	c, closer, err := s.newEtcdClient()
	if err != nil {
		return errors.Wrap(err, "fail to build etcd client")
	}
	defer closer.Close()

	out, err := json.Marshal(&data)
	if err != nil {
		return errors.Wrapf(err, "fail to encode to JSON")
	}

	_, err = c.Put(ctx, key, string(out))
	if err != nil {
		return errors.Wrapf(err, "fail to read key %v", key)
	}

	log.WithFields(logrus.Fields{"key": key}).Debug("put key")
	return nil
}

func (s *store) Delete(ctx context.Context, key string) error {
	log := logger.Get(ctx).WithField("scope", "store")
	key = s.Key(key)
	c, closer, err := s.newEtcdClient()
	if err != nil {
		return errors.Wrap(err, "fail to build etcd client")
	}
	defer closer.Close()

	_, err = c.Delete(ctx, key)
	if err != nil {
		return errors.Wrapf(err, "fail to delete key %v", key)
	}
	log.WithFields(logrus.Fields{"key": key}).Debug("delete key")
	return nil
}

func (s *store) Watch(ctx context.Context, prefix string) (clientv3.WatchChan, io.Closer, error) {
	log := logger.Get(ctx)
	prefix = s.Key(prefix)

	client, err := etcd.NewClient(s.config)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "fail to create etcd client")
	}

	wc := clientv3.NewWatcher(client)
	log.Infof("create endpoints watcher on prefix %v", prefix)

	// Use context.Background() to avoid the resulting chan to be closed at the end of a HTTP request
	// https://godoc.org/github.com/coreos/etcd/clientv3#Watcher
	wchan := wc.Watch(context.Background(), prefix, clientv3.WithPrefix())

	return wchan, Closer(func() error {
		err := client.Close()
		if err != nil {
			return err
		}
		err = wc.Close()
		if err != nil {
			return err
		}
		return nil
	}), nil
}

func (s *store) Key(key string) string {
	return s.config.EtcdPrefix + key
}
