package etcd

import (
	"context"
	"os"

	clientv3 "go.etcd.io/etcd/client/v3"

	etcdutils "github.com/Scalingo/go-utils/etcd"

	"github.com/Scalingo/go-utils/errors/v3"
)

func NewClient(ctx context.Context) (*clientv3.Client, error) {
	// Error has already been checked in the config initialization. We can safely ignore it here
	etcdConfig, _ := etcdutils.ConfigFromEnv()
	if os.Getenv("GO_ENV") == "development" && etcdConfig.TLS != nil {
		etcdConfig.TLS.InsecureSkipVerify = true
	}

	client, err := clientv3.New(etcdConfig)
	if err != nil {
		return nil, errors.Wrap(ctx, err, "create etcd client")
	}
	return client, nil
}
