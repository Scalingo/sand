module github.com/Scalingo/sand

go 1.20

require (
	github.com/Scalingo/go-etcd-lock/v5 v5.0.6
	github.com/Scalingo/go-handlers v1.8.0
	github.com/Scalingo/go-plugins-helpers v1.3.0
	github.com/Scalingo/go-utils/etcd v1.1.1
	github.com/Scalingo/go-utils/graceful v1.1.1
	github.com/Scalingo/go-utils/logger v1.2.0
	github.com/bits-and-blooms/bitset v1.7.0
	github.com/gofrs/uuid v4.4.0+incompatible
	github.com/golang/mock v1.6.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/magefile/mage v1.14.0
	github.com/moby/moby v23.0.2+incompatible
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.9.2
	github.com/stretchr/testify v1.8.2
	github.com/urfave/cli v1.22.13
	github.com/vishvananda/netlink v1.1.0
	github.com/vishvananda/netns v0.0.4
	go.etcd.io/etcd/api/v3 v3.5.8
	go.etcd.io/etcd/client/v3 v3.5.7
	golang.org/x/sys v0.8.0
	// This shouldn't be upgraded as long as go.etcd.io/etcd/v3 has not been updated.
	// Waiting for etcd 3.5 release: https://github.com/etcd-io/etcd/issues/12124
	google.golang.org/grpc v1.54.0
	gopkg.in/errgo.v1 v1.0.1
)

require (
	github.com/Microsoft/go-winio v0.6.0 // indirect
	github.com/Scalingo/errgo-rollbar v0.2.1 // indirect
	github.com/Scalingo/go-utils/crypto v1.0.0 // indirect
	github.com/Scalingo/go-utils/errors/v2 v2.2.0 // indirect
	github.com/Scalingo/go-utils/security v1.0.0 // indirect
	github.com/Scalingo/logrus-rollbar v1.4.1 // indirect
	github.com/coreos/go-semver v0.3.1 // indirect
	github.com/coreos/go-systemd v0.0.0-20190321100706-95778dfbb74e // indirect
	github.com/coreos/go-systemd/v22 v22.5.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-plugins-helpers v0.0.0-20200102110956-c9a8a2d92ccc // indirect
	github.com/facebookgo/grace v0.0.0-20180706040059-75cf19382434 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rollbar/rollbar-go v1.4.4 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/urfave/negroni v1.0.0 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.5.9 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.24.0 // indirect
	golang.org/x/mod v0.10.0 // indirect
	golang.org/x/net v0.8.0 // indirect
	golang.org/x/text v0.8.0 // indirect
	golang.org/x/tools v0.7.0 // indirect
	google.golang.org/genproto v0.0.0-20230110181048-76db0878b65f // indirect
	google.golang.org/protobuf v1.30.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	gotest.tools/v3 v3.4.0 // indirect
)
