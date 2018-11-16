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
