package docker

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/Scalingo/sand/config"
	"github.com/Scalingo/sand/store"
)

type Repository interface {
	SaveNetwork(context.Context, DockerPluginNetwork) error
	SaveEndpoint(context.Context, DockerPluginEndpoint) error
	ListEndpoints(context.Context) ([]DockerPluginEndpoint, error)
	GetNetworkByDockerID(context.Context, string) (DockerPluginNetwork, error)
	GetEndpointByDockerID(context.Context, string) (DockerPluginEndpoint, error)
	DeleteNetwork(context.Context, DockerPluginNetwork) error
	DeleteEndpoint(context.Context, DockerPluginEndpoint) error
}

type DockerPluginNetwork struct {
	DockerNetworkID string
	SandNetworkID   string
}

func (n DockerPluginNetwork) StorageKey() string {
	return fmt.Sprintf("/docker-networks/%s", n.DockerNetworkID)
}

type DockerPluginEndpoint struct {
	DockerPluginNetwork
	DockerEndpointID string
	SandEndpointID   string
}

func (n DockerPluginEndpoint) StorageKey() string {
	return fmt.Sprintf("/docker-endpoints/%s", n.DockerEndpointID)
}

type repository struct {
	config *config.Config
	store  store.Store
}

func NewRepository(c *config.Config, s store.Store) repository {
	return repository{config: c, store: s}
}

func (r repository) SaveNetwork(ctx context.Context, n DockerPluginNetwork) error {
	return r.store.Set(ctx, n.StorageKey(), &n)
}

func (r repository) SaveEndpoint(ctx context.Context, e DockerPluginEndpoint) error {
	return r.store.Set(ctx, e.StorageKey(), &e)
}

func (r repository) GetNetworkByDockerID(ctx context.Context, id string) (DockerPluginNetwork, error) {
	n := DockerPluginNetwork{DockerNetworkID: id}
	err := r.store.Get(ctx, n.StorageKey(), false, &n)
	return n, err
}

func (r repository) GetEndpointByDockerID(ctx context.Context, id string) (DockerPluginEndpoint, error) {
	e := DockerPluginEndpoint{DockerEndpointID: id}
	err := r.store.Get(ctx, e.StorageKey(), false, &e)
	return e, err
}

func (r repository) ListEndpoints(ctx context.Context) ([]DockerPluginEndpoint, error) {
	res := make([]DockerPluginEndpoint, 0)
	err := r.store.Get(ctx, "/docker-endpoints/", true, &res)
	if err != nil {
		return nil, errors.Wrap(err, "query etcd for docker endpoints")
	}
	return res, nil
}

func (r repository) DeleteEndpoint(ctx context.Context, e DockerPluginEndpoint) error {
	return r.store.Delete(ctx, e.StorageKey())
}

func (r repository) DeleteNetwork(ctx context.Context, n DockerPluginNetwork) error {
	return r.store.Delete(ctx, n.StorageKey())
}
