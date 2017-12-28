package web

import (
	"github.com/Scalingo/sand/config"
	"github.com/Scalingo/sand/endpoint"
	"github.com/Scalingo/sand/network"
	"github.com/Scalingo/sand/network/overlay"
	"github.com/Scalingo/sand/store"
)

type EndpointsController struct {
	Config             *config.Config
	Store              store.Store
	EndpointRepository endpoint.Repository
	NetworkRepository  network.Repository
}

func NewEndpointsController(c *config.Config, listener overlay.NetworkEndpointListener) EndpointsController {
	store := store.New(c)
	return EndpointsController{
		Config: c, Store: store,
		EndpointRepository: endpoint.NewRepository(c, store),
		NetworkRepository:  network.NewRepository(c, store, listener),
	}
}
