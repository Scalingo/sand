package endpoint

import (
	"context"

	"github.com/Scalingo/sand/api/params"
	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/config"
	"github.com/Scalingo/sand/network/netmanager"
	"github.com/Scalingo/sand/store"
)

const (
	NetworkEndpointPrefix = "/network-endpoints"
)

type Repository interface {
	List(context.Context, map[string]string) ([]types.Endpoint, error)
	Create(context.Context, types.Network, params.EndpointCreate) (types.Endpoint, error)
	Save(context.Context, types.Endpoint) error
	Activate(context.Context, types.Network, types.Endpoint, params.EndpointActivate) (types.Endpoint, error)
	Delete(context.Context, types.Network, types.Endpoint, DeleteOpts) error
	Deactivate(context.Context, types.Network, types.Endpoint) (types.Endpoint, error)

	// If the endpoint has already been attach to the network in the kv store
	Exists(context.Context, string) (types.Endpoint, bool, error)
}

type repository struct {
	config   *config.Config
	store    store.Store
	managers netmanager.ManagerMap
}

func NewRepository(config *config.Config, store store.Store, managers netmanager.ManagerMap) Repository {
	return &repository{config: config, store: store, managers: managers}
}
