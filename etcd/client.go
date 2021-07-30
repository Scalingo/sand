package etcd

import (
	"os"

	etcdutils "github.com/Scalingo/go-utils/etcd"
	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/pkg/errors"
)

func NewClient() (*clientv3.Client, error) {
	// Error has already been checked in the config initialization. We can safely ignore it here
	etcdConfig, _ := etcdutils.ConfigFromEnv()
	if os.Getenv("GO_ENV") == "development" && etcdConfig.TLS != nil {
		etcdConfig.TLS.InsecureSkipVerify = true
	}

	client, err := clientv3.New(etcdConfig)
	if err != nil {
		return nil, errors.Wrap(err, "fail to create etcd client")
	}
	return client, nil
}
