package store

import (
	"io"

	"go.etcd.io/etcd/v3/clientv3"

	"github.com/Scalingo/sand/etcd"
	"github.com/pkg/errors"
)

func (s *store) newEtcdClient() (clientv3.KV, io.Closer, error) {
	c, err := etcd.NewClient()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "fail to get etcd client from config")
	}
	return clientv3.KV(c), c, nil
}
