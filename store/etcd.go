package store

import (
	"io"

	"github.com/Scalingo/networking-agent/etcd"
	"github.com/coreos/etcd/clientv3"
	"github.com/pkg/errors"
)

func (s *store) newEtcdClient() (clientv3.KV, io.Closer, error) {
	c, err := etcd.NewClient(s.config)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "fail to get etcd client from config")
	}
	return clientv3.KV(c), c, nil
}
