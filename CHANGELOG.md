# Changelog

## To be Released

* chore(go): use go 1.24
* feat: Separate Peer Hostname from API Hostname, the first is used to define
  endpoint and configure vxlan network, other is how to call the SAND API.

## v1.0.3 - 14 Oct 2024

* fix: bump graceful version to v1.2.0 to allow multiple http servers to be started with a single graceful service
* fix(server version): update server version
* chore: bump various dependencies

## v1.0.2 - 28 Sep 2023

* chore: bump various dependencies

## v1.0.1 - 29 Nov 2022

* fix(sand agent cli): add a timeout to the curl http client
* fix(sand client): disable HTTP keepalives to prevent leaking file descriptors
* chore(go): use go 1.17
* chore(deps): bump github.com/Scalingo/go-handlers from 1.4.0 to 1.4.4
* chore(deps): bump github.com/Scalingo/go-utils/etcd from 1.1.0 to 1.1.1
* chore(deps): bump github.com/Scalingo/go-utils/logger from 1.1.0 to 1.2.0
* chore(deps): bump github.com/Scalingo/go-utils/graceful
* chore(deps): bump github.com/Scalingo/go-etcd-lock/v5
* chore(deps): bump github.com/gofrs/uuid from 4.1.0+incompatible to 4.2.0+incompatible
* chore(deps): bump github.com/sirupsen/logrus from 1.8.1 to 1.9.0
* chore(deps): bump github.com/vishvananda/netlink from 1.0.0 to 1.1.0
* chore(deps): bump go.etcd.io/etcd/api/v3 from 3.5.0 to 3.5.6
* chore(deps): bump go.etcd.io/etcd/client/v3 from 3.5.0 to 3.5.6
* chore(deps): bump google.golang.org/grpc from 1.42.0 to 1.51.0
* chore(deps): bump github.com/magefile/mage from 1.11.0 to 1.12.1
* chore(deps): bump github.com/docker/docker from 20.10.11+incompatible to 20.10.17+incompatible
* chore(deps): bump github.com/urfave/cli from 1.22.5 to 1.22.9
* chore(deps): bump github.com/bits-and-blooms/bitset from 1.2.2 to 1.3.0
* chore(deps): bump github.com/stretchr/testify from 1.7.1 to 1.8.0

## v1.0.0 - 20 Oct 2021

* client: don't create a new pool of connection each time a sand.Client is created
* fix: 2 descriptor leaks when the request fails
* chore: Replace github.com/pborman/uuid with github.com/gofrs/uuid
* chore(Dependabot): Update various dependencies
* chore: Migration to Go Module
* Bump github.com/Scalingo/go-handlers from 1.3.1 to 1.4.0
* Bump github.com/golang/mock from 1.4.4 to 1.5.0
* Bump github.com/sirupsen/logrus from 1.7.0 to 1.8.1
* Bump go.etcd.io/etcd from v3-pregomod to v3.5.0
* Bump google.golang.org/grpc from 1.39.0 to 1.41.0
* Bump go version to 1.16 and remove deprecated lib `ioutil`
* chore(bitset): organization changed to bits-and-blooms and update to 1.2.1

## v0.8.1 - 7 Jul 2020

* Fix file descriptor leakage: correctly close etcd client when allocating a unique VxLAN VNI ID.

## v0.8.0 - 10 Mar 2020

* Use only one ETCD connection watching key changes in order to avoid starting
  thousands of listeners in SAND infrastructure deployments

## v0.7.0 - 10 Jan 2020

* New algorithm for VxLAN ID generation
* Update go-etcd-lock lib to remove plenty of bugs
* Add ability to use --timeout on the CLI

## v0.6.0 - 15 Nov 2019

* Update etcd client to 3.4.3
* Fix error management to throw errors only when required
* Bugfix when a node which hostname is a prefix of another, trying to add the other network on start

  ie. `ip-10-0-0-20` considering networks of `ip-10-0-0-207` are theirs

* Correctly stop listening network change when a network is deactive on a node

## v0.5.11 - 8 Nov 2019

* Refactor connection forwarding to sand network, no more fork/exec, change namespace in current thread
* Make logging more quiet when forwarding connections

## v0.5.10 - 5 Nov 2019

* Deactivating inactive endpoint in a network is a no-op

## v0.5.9 - 25 Oct 2019

* Better logging in endpoint listeners watching

## v0.5.8 - 22 Oct 2019

* Integration project with Rollbar:
  Errors will be sent to Rollbar if `ROLLBAR_ENV` and `ROLLBAR_TOKEN` environment variables are present.

## v0.5.7 - 10 June 2019

* Fix endpoint deletion in overlay network in case of target namespace does not exist anymore

## v0.5.6 - 2 May 2019

* Fix socket leak, netlink.Handler should be Delete()
* Fix initialization error when namespace of endpoint does not exist anymore

## v0.5.5 - 7 December 2018

* Update of travis configuration

## v0.5.4 - 7 December 2018

* Add `version` command to CLI
* Add `GET /version` command to Agent

## v0.5.3 - 6 December 2018

* Fix: network#connect on cross node communication

## v0.5.2 - 5 December 2018

* Update Client#NewHTTPRoundTripper, possibility to inject *tls.Config

## v0.5.1 - 21 November 2018

* Do not let docker de-allocate sand network gateway IP
* Fix network#connect when SAND is serving HTTPS traffic

## v0.5.0 - 20 November 2018

* Networks have to be created and deleted through SAND API, not possible from docker integration
* Fix "next IP" allocation algorithm
* Fix docker integration IP allocation pool deletion bug
* Fix docker integration double gateway issue

## v0.4.5 - 20 November 2018

* Fix graceful restart of sand API and sand Docker integration API
* Correctly handle resource deprovisioning

## v0.4.4 - 16 November 2018

* Fix custom ip gateway definition, add CIDR

## v0.4.3 - 16 November 2018

* Fix custom ip range on POST /networks

## v0.2 - 12 February 2018

* Docker Libnetwork remote plugin v1
  Enable it with environment variable `ENABLE_DOCKER_PLUGIN=true`
  Listening port is 9998, configurable with `DOCKER_PLUGIN_HTTP_PORT`
* Lock IP allocator cluster-wide to ensure unicity of allocation

## v0.1 - 07 February 2018

* Initial release
