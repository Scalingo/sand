[![Codeship Status for SAND](https://app.codeship.com/projects/3787f4e2-9515-4e36-aaa4-44d8e5bd1955/status?branch=master)](https://app.codeship.com/projects/425178)

# SAND Network Daemon V1.0.3

SAND is simple API designed to create overlay networks based on
**[VXLAN](https://en.wikipedia.org/wiki/Virtual_Extensible_LAN)** in an infrastructure, basing its
configuration on [**etcd**](https://coreos.com/etcd/).

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

Creating a network is a no-op operation where a unique VXLAN ID is allocated
and where the network configuration is stored on *etcd*.

When a first endpoint is added to a network, the service will create a
dedicated network namespace containing the network **VXLAN** on the server
adding the endpoint. A pair of **veth** interfaces will link the targeted
namespace and the overlay namespace. All the **veth** interfaces are linked to
the **VXLAN** with a bridge interface.

At the moment an endpoint is added or removed, all the other hosts having at
least one endpoint in the same network are adding routes to the newly created
endpoint modifying `ARP` and `FDB` tables of the **VXLAN** interface.

## Installing

To install the server:

```
go install github.com/Scalingo/sand/cmd/sand-agent@latest
```

To install the CLI:

```
go install github.com/Scalingo/sand/cmd/sand-agent-cli@latest
```

## Configuration (from environment)

* `NETNS_PATH` default: `/var/run/netns`, location where SAND will create network namespace handlers
* `NETNS_PREFIX` default: `sc-ns-`, name prefix for the network namespace handler files
* `HTTP_PORT` default: `9999`, port bind by the SAND HTTP API
* `PUBLIC_HOSTNAME` default: `$(hostname)`, endpoints are attached to a
  hostname, an agent won't accept to delete a endpoint if its not owned by its hostname
* `PUBLIC_IP` IP of the host which will be used in the configuration of VXLAN routing rules
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

#### Generating certificates

```
# Generate a CA valid for 5 years
openssl genrsa -out ca.key 4096
openssl req -x509 -new -nodes -key ca.key -sha256 -days 1825 -out ca.pem

# Generate a client certificate valid for 2 years
openssl genrsa -out client.key 4096
openssl req -new -key client.key -out client.csr
openssl x509 -req -in client.csr -CA ca.pem -CAkey ca.key -CAcreateserial -out client.pem -days 730 -sha256
```

## References

> `GET` requests accept parameters through URL query parameters
> `POST` requests accept a JSON body
> `POST` and `GET` requests return a JSON body

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

```go
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

### With Docker

On each server which should be part of a network the sand-id MUST be defined, as
it should be common to all nodes running the network and docker is not returning
the internal ID, so the knowledge has to be external from Docker.

Create the SAND network with:

```
$ sand-agent-cli network-create
New network created:
* id=320a669f-e465-4806-ab46-f2e6620c4311 name=net-sc-320a669f-e465-4806-ab46-f2e6620c4311 type=overlay ip-range=10.0.0.0/24, vni=4
$ SAND_ID="320a669f-e465-4806-ab46-f2e6620c4311"
```

Then create a Docker network:

```
$ docker network create --driver sand --ipam-driver sand --ipam-opt sand-id=$SAND_ID --opt sand-id=$SAND_ID <name>
```

Finally, start as many containers as you want in the SAND network defined in the
docker network:

```
$ docker run -it --rm --network <name> ubuntu:latest bash
```

### Test in Development With Docker Compose

Create the SAND network with:

```
$ sand-agent-cli network-create
New network created:
* id=320a669f-e465-4806-ab46-f2e6620c4311 name=net-sc-320a669f-e465-4806-ab46-f2e6620c4311 type=overlay ip-range=10.0.0.0/24, vni=4
```

The `id=` part is important and is called `SAND_ID` in the remaining of the section.

Your `docker-compose.yml` file MUST use the version 2:

```yml
version: '2'
networks:
  sand-network:
    driver: sand
    driver_opts:
      sand-id: SAND_ID
    ipam:
      driver: sand
      options:
        sand-id: SAND_ID

services:
  service-1:
    ...
    networks:
      - sand-network

  service-2:
    ...
    networks:
      - sand-network
```

## Release a New Version

Bump new version number in:

- `CHANGELOG.md`
- `README.md`

Commit the new version number:

```sh
version="1.0.3"

sed --in-place "s/var Version = \"v\([0-9.]*\)\"/var Version = \"v$version\"/g" config/config.go

git switch --create release/${version}
git add CHANGELOG.md README.md config/config.go
git commit --message="Bump v${version}"
git push --set-upstream origin release/${version}
gh pr create --reviewer=leo-scalingo --title "$(git log -1 --pretty=%B)"
```

Once the pull request merged, you can compile and tag the new release.

```sh
git tag v$version
git push origin master v$version
goreleaser release --skip=publish,announce,sign --clean
gh release create v${version} --generate-notes --prerelease
gh release view v${version} --web
```

On the web interface, unset the pre-release checkbox, check the "Set as the latest release" checkbox, and upload the archives in the `dist` folder.

If you face an issue about a missing library during the compilation, you may be missing the following packages:

```sh
sudo apt-get install build-essentials gcc-multilib g++-multilib
```

## Testing

* Single node with `docker-compose`, just run `docker-compose up` and you can
start using SAND.

* Multinodes using `vagrant`, just run `vagrant up` to start two nodes with
Docker installed and SAND code mounted in them.

* More tests and mocking of `netlink` commands
