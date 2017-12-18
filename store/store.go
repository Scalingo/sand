package store

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"

	"github.com/Scalingo/go-internal-tools/logger"
	"github.com/Scalingo/networking-agent/config"
	"github.com/coreos/etcd/clientv3"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var ErrNotFound = errors.New("not found")

type Store interface {
	Get(ctx context.Context, key string, recursive bool, data interface{}) error
	Set(ctx context.Context, key string, data interface{}) error
	Delete(ctx context.Context, key string) error
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
	c, err := s.newEtcdClient()
	if err != nil {
		return errors.Wrap(err, "fail to build etcd client")
	}
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
	c, err := s.newEtcdClient()
	if err != nil {
		return errors.Wrap(err, "fail to build etcd client")
	}

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
	c, err := s.newEtcdClient()
	if err != nil {
		return errors.Wrap(err, "fail to build etcd client")
	}

	_, err = c.Delete(ctx, key)
	if err != nil {
		return errors.Wrapf(err, "fail to delete key %v", key)
	}
	log.WithFields(logrus.Fields{"key": key}).Debug("delete key")
	return nil
}

func (s *store) Key(key string) string {
	return s.config.EtcdPrefix + key
}
