## To be released

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
