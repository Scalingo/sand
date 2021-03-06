package network

import (
	"context"

	"github.com/Scalingo/sand/api/params"
	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/config"
	"github.com/Scalingo/sand/ipallocator"
	"github.com/Scalingo/sand/network/netmanager"
	"github.com/Scalingo/sand/store"
)

type Repository interface {
	List(ctx context.Context) ([]types.Network, error)
	Create(ctx context.Context, params params.NetworkCreate) (types.Network, error)
	Ensure(ctx context.Context, network types.Network) error
	Deactivate(ctx context.Context, network types.Network) error
	Delete(ctx context.Context, network types.Network, a ipallocator.IPAllocator) error
	Exists(ctx context.Context, id string) (types.Network, bool, error)
}

type repository struct {
	config   *config.Config
	store    store.Store
	managers netmanager.ManagerMap
}

func NewRepository(config *config.Config, store store.Store, managers netmanager.ManagerMap) Repository {
	return &repository{
		config: config, store: store, managers: managers,
	}
}
