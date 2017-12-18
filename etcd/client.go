package etcd

import (
	"time"

	"github.com/Scalingo/networking-agent/config"
	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/pkg/transport"
	"github.com/pkg/errors"
)

func NewClient(c *config.Config) (*clientv3.Client, error) {
	etcdConfig := clientv3.Config{
		Endpoints:   c.EtcdEndpoints,
		DialTimeout: 5 * time.Second,
	}

	if c.EtcdWithTLS {
		tlsInfo := transport.TLSInfo{
			CertFile:      c.EtcdTLSCert,
			KeyFile:       c.EtcdTLSKey,
			TrustedCAFile: c.EtcdTLSCA,
		}
		tlsConfig, err := tlsInfo.ClientConfig()
		if err != nil {
			return nil, errors.Wrap(err, "fail to create tls info config")
		}
		etcdConfig.TLS = tlsConfig
	}

	client, err := clientv3.New(etcdConfig)
	if err != nil {
		return nil, errors.Wrap(err, "fail to create etcd client")
	}
	return client, nil
}
