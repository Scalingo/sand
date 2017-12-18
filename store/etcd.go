package store

import (
	"github.com/Scalingo/networking-agent/etcd"
	"github.com/coreos/etcd/clientv3"
	"github.com/pkg/errors"
)

func (s *store) newEtcdClient() (clientv3.KV, error) {
	c, err := etcd.NewClient(s.config)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to get etcd client from config")
	}
	return clientv3.KV(c), nil
}
