# SAND Network Daemon

SAND is simple API designed to create overlay networks based on **VxLAN** in an
infrastructure, basing its configuration on
[**etcd**](https://coreos.com/etcd/)

The goal is to create private overlay network to link containers together,
while being agnostic of the container technology. *Libnetwork* `overlay`
network type is working but is so much bound to *Docker*, that you don't really
have the choice of the container engine in your infrastructure and you get locked
to *Docker*.

## Design

The SAND network daemon should be installed on all the hosts which will have
containers running in one of the overlay networks. All the created overlay
networks can use the same IP range as they are completely isolated from each
other. By default each overlay network will get IP address in `10.0.0.0/24`

Creating a network is a no-op operation where a unique VxLAN ID is allocated
and where the network configuration is stored on *etcd*.

When a first endpoint is added to a network, the service will create a
dedicated network namespace containing the network **VxLAN** on the server
adding the endpoint. A pair of **veth** interfaces will link the targeted
namespace and the overlay namespace. All the **veth** interfaces are linked to
the **VxLAN** with a bridge interface.

At the moment an endpoint is added or removed, all the other hosts having at
least one endpoint in the same network are adding routes to the newly created
endpoint modifying `ARP` and `FDB` tables of the **VxLAN** interface.

## Installing

To install the server:

```
go get github.com/Scalingo/sand/cmd/sand-agent
```

To install the CLI:

```
go get github.com/Scalingo/sand/cmd/sand-agent-cli
```

## Configuration (from environment)

* `NETNS_PATH` default: `/var/run/netns`, location where SAND will create network namespace handlers
* `NETNS_PREFIX` default: `sc-ns-`, name prefix for the network namespace handler files
* `HTTP_PORT` default: `9999`, port bind by the SAND HTTP API
* `PUBLIC_HOSTNAME` default: `$(hostname)`, endpoints are attached to a
  hostname, an agent won't accept to delete a endpoint if its not owned by its hostname
* `PUBLIC_IP` IP of the host which will be used in the configuration of VxLAN routing rules
* `ROLLBAR_TOKEN` If token is defined, all errors will be send to [Rollbar](https://rollbar.com/)
* `GO_ENV` default: `development`, name of the environment, will be forwarded to Rollbar if configured

### ETCD TLS configuration

> Note that at least ETCD v3 is required to run SAND

* `ETCD_PREFIX` default: `/sc-net`, configuration of SAND is stored in ETCD, this is the prefix used by the keys
* `ETCD_HOSTS` default: `http://127.0.0.1:2379`, URL of the etcd instance/instances, ie. `https://10.0.0.1:2379,10.0.0.2:2379,10.0.0.3:2379`

* `ETCD_TLS_CERT` Path to the client certificate to reach ETCD
* `ETCD_TLS_KEY` Path to the private key authenticating the client certificate
* `ETCD_CACERT` Path to the CA used by ETCD server certificate

### HTTP TLS authentication

If all three are defined, server will serve HTTPS instead of HTTP with client
certificate authentication will be enabled, refusing requests from unauthorized
clients.

* `HTTP_TLS_CERT` Path to the server certificate sent by the server
* `HTTP_TLS_KEY` Path to the private key authenticating the server certificate
* `HTTP_TLS_CA` Path to the CA used by SAND client certificates

## References

> `GET` requests accept parameters through URL query parameters
> `POST` requests accept a JSON body
> `POST` and `GET` requests retun a JSON body

* `GET /networks`
* `POST /networks`
  Parameters:
  * `name` - string - Name of the network, generated automatically if not set
  * `ip_range` - string - IP Range from which endpoint IP will be allocated from
* `DELETE /networks/{id}`
* `GET /endpoints`
  Parameters:
  * `network_id` - string - Filter the returned networks by network
  * `hostname` - string - Filter the returned endpoints by hostname
* `POST /endpoints`
  Parameters:
  * `network_id` - string - ID to the network to use
  * `ns_handle_path` - string - path to the target namespace handler to inject the network
* `DELETE /endpoints/{id}`

## Go client package

Documentation: [godoc](https://godoc.org/github.com/Scalingo/sand/client/sand)

```
import "github.com/Scalingo/sand/client/sand

func main() {
	opts := []sand.Opt{
		sand.WithURL(a.config.ApiURL),
	}
	config, _ := sand.TlsConfig(
		caPath, certPath, keyPath,
	)
	opts = append(opts, sand.WithTlsConfig(config))
	client := sand.NewClient(opts)

	// Use the client
}
```

## CLI

```
sand-agent-cli network-list
sand-agent-cli network-create [--name name]
sand-agent-cli network-delete --network id
sand-agent-cli endpoint-list [--network id] [--hostname hostname]
sand-agent-cli endpoint-create --network id --ns path_target_namespace_handler
sand-agent-cli endpoint-delete --endpoint id
```

### Global flags

```
   --api-url value    when requests will be sent (default: "http://localhost:9999") [$SAND_API_URL]
   --cert-file value  identify HTTPS client using this SSL certificate file [$SAND_CERT_FILE]
   --key-file value   identify HTTPS client using this SSL key file [$SAND_KEY_FILE]
   --ca-file value    verify certificates of HTTPS-enabled servers using this CA bundle [$SAND_CA_FILE]
   --help, -h         show help
   --version, -v      print the version
```

## Docker Integration

Start with the environment variable `ENABLE_DOCKER_PLUGIN=true`

It will use the port **9998** by default to communicate with Docker. Change
`DOCKER_PLUGIN_HTTP_PORT` to customize it.

```
# On each server which should be part of a network
# the sand-id SHOULD be defined, as it should be common to all nodes running the network
# and docker is not returning the internal ID, so the knowledge has to be external from docker
$ docker network create --driver sand --ipam-opt sand-id=<uuid> --opt sand-id=<uuid> [--opt sans-name=<name>] <name>

# Start a container in the sand network defined in the docker network
$ docker run --network <name> ubuntu:latest bash

```

## Testing

* Single node with `docker-compose`, just run `docker-compose up` and you can
start using SAND.

* Multinodes using `vagrant`, just run `vagrant up` to start two nodes with
Docker installed and SAND code mounted in them.

* More tests and mocking of `netlink` commands

