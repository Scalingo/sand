package web

import (
	"github.com/Scalingo/sand/config"
	"github.com/Scalingo/sand/endpoint"
	"github.com/Scalingo/sand/network"
)

type EndpointsController struct {
	Config             *config.Config
	EndpointRepository endpoint.Repository
	NetworkRepository  network.Repository
}

func NewEndpointsController(c *config.Config, n network.Repository, e endpoint.Repository) EndpointsController {
	return EndpointsController{
		Config:             c,
		EndpointRepository: e,
		NetworkRepository:  n,
	}
}
