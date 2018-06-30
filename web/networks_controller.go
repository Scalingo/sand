package web

import (
	"github.com/Scalingo/sand/config"
	"github.com/Scalingo/sand/endpoint"
	"github.com/Scalingo/sand/ipallocator"
	"github.com/Scalingo/sand/network"
)

type NetworksController struct {
	Config             *config.Config
	EndpointRepository endpoint.Repository
	NetworkRepository  network.Repository
	IPAllocator        ipallocator.IPAllocator
}

func NewNetworksController(c *config.Config, n network.Repository, e endpoint.Repository, a ipallocator.IPAllocator) NetworksController {
	return NetworksController{
		Config:             c,
		NetworkRepository:  n,
		EndpointRepository: e,
		IPAllocator:        a,
	}
}
