module github.com/Scalingo/sand

go 1.16

require (
	github.com/Microsoft/go-winio v0.4.16 // indirect
	github.com/Scalingo/go-etcd-lock/v5 v5.0.5
	github.com/Scalingo/go-handlers v1.4.2
	github.com/Scalingo/go-plugins-helpers v1.3.0
	github.com/Scalingo/go-utils/etcd v1.1.0
	github.com/Scalingo/go-utils/graceful v1.1.0
	github.com/Scalingo/go-utils/logger v1.1.0
	github.com/bits-and-blooms/bitset v1.2.1
	github.com/coreos/go-systemd v0.0.0-20190321100706-95778dfbb74e // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.0 // indirect
	github.com/docker/docker v20.10.12+incompatible
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-plugins-helpers v0.0.0-20200102110956-c9a8a2d92ccc // indirect
	github.com/docker/libnetwork v0.8.0-dev.2.0.20171213192018-26531e56a76d
	github.com/gofrs/uuid v4.2.0+incompatible
	github.com/golang/mock v1.6.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/kr/text v0.2.0 // indirect
	github.com/magefile/mage v1.12.1
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.8.1
	github.com/stretchr/testify v1.7.0
	github.com/urfave/cli v1.22.5
	github.com/vishvananda/netlink v1.0.0
	github.com/vishvananda/netns v0.0.0-20210104183010-2eb08e3e575f
	go.etcd.io/etcd/api/v3 v3.5.1
	go.etcd.io/etcd/client/v3 v3.5.1
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.7.0 // indirect
	go.uber.org/zap v1.18.1 // indirect
	golang.org/x/net v0.0.0-20210726213435-c6fcb2dbf985 // indirect
	google.golang.org/genproto v0.0.0-20210729151513-df9385d47c1b // indirect
	// This shouldn't be upgraded as long as go.etcd.io/etcd/v3 has not been updated.
	// Waiting for etcd 3.5 release: https://github.com/etcd-io/etcd/issues/12124
	google.golang.org/grpc v1.44.0
	gopkg.in/errgo.v1 v1.0.1
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gotest.tools/v3 v3.0.3 // indirect
)
