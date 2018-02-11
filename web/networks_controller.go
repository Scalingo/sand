package web

import (
	"github.com/Scalingo/sand/config"
	"github.com/Scalingo/sand/endpoint"
	"github.com/Scalingo/sand/network"
	"github.com/Scalingo/sand/network/netmanager"
	"github.com/Scalingo/sand/store"
)

type NetworksController struct {
	Config             *config.Config
	Store              store.Store
	EndpointRepository endpoint.Repository
	NetworkRepository  network.Repository
}

func NewNetworksController(c *config.Config, managers netmanager.ManagerMap) NetworksController {
	store := store.New(c)
	return NetworksController{
		Config:             c,
		Store:              store,
		EndpointRepository: endpoint.NewRepository(c, store, managers),
		NetworkRepository:  network.NewRepository(c, store, managers),
	}
}
