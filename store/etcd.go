package store

import (
	"context"
	"io"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/Scalingo/go-utils/errors/v3"
	"github.com/Scalingo/sand/etcd"
)

func (s *store) newEtcdClient(ctx context.Context) (clientv3.KV, io.Closer, error) {
	c, err := etcd.NewClient(ctx)
	if err != nil {
		return nil, nil, errors.Wrapf(ctx, err, "get etcd client from config")
	}
	return clientv3.KV(c), c, nil
}
