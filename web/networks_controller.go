package web

import (
	"github.com/Scalingo/sand/config"
	"github.com/Scalingo/sand/endpoint"
	"github.com/Scalingo/sand/network"
	"github.com/Scalingo/sand/network/overlay"
	"github.com/Scalingo/sand/store"
)

type NetworksController struct {
	Config             *config.Config
	Store              store.Store
	Listener           overlay.NetworkEndpointListener
	EndpointRepository endpoint.Repository
	NetworkRepository  network.Repository
}

func NewNetworksController(c *config.Config, listener overlay.NetworkEndpointListener) NetworksController {
	store := store.New(c)
	return NetworksController{
		Config:             c,
		Store:              store,
		Listener:           listener,
		EndpointRepository: endpoint.NewRepository(c, store),
		NetworkRepository:  network.NewRepository(c, store, listener),
	}
}
