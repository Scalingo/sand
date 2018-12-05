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
