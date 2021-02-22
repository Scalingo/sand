module github.com/Scalingo/sand

go 1.15

require (
	github.com/Microsoft/go-winio v0.4.16 // indirect
	github.com/Scalingo/go-etcd-lock/v5 v5.0.4
	github.com/Scalingo/go-handlers v1.4.0
	github.com/Scalingo/go-plugins-helpers v1.3.0
	github.com/Scalingo/go-utils/etcd v1.0.1
	github.com/Scalingo/go-utils/graceful v1.0.0
	github.com/Scalingo/go-utils/logger v1.0.0
	github.com/docker/docker v1.13.1
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-plugins-helpers v0.0.0-20200102110956-c9a8a2d92ccc // indirect
	github.com/docker/libnetwork v0.8.0-dev.2.0.20171213192018-26531e56a76d
	github.com/gofrs/uuid v3.4.0+incompatible
	github.com/golang/mock v1.5.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/magefile/mage v1.11.0
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.8.0
	github.com/stretchr/testify v1.7.0
	github.com/urfave/cli v1.22.5
	github.com/vishvananda/netlink v1.0.0
	github.com/vishvananda/netns v0.0.0-20210104183010-2eb08e3e575f
	github.com/willf/bitset v1.1.11
	go.etcd.io/etcd/v3 v3.3.0-rc.0.0.20200826232710-c20cc05fc548
	// This shouldn't be upgraded as long as go.etcd.io/etcd/v3 has not been updated.
	// Waiting for etcd 3.5 release: https://github.com/etcd-io/etcd/issues/12124
	google.golang.org/grpc v1.29.1
	gopkg.in/errgo.v1 v1.0.1
)
