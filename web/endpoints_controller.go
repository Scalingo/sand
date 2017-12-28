package web

import (
	"github.com/Scalingo/sand/config"
	"github.com/Scalingo/sand/network/overlay"
	"github.com/Scalingo/sand/store"
)

type EndpointsController struct {
	Config   *config.Config
	Store    store.Store
	Listener overlay.NetworkEndpointListener
}

func NewEndpointsController(c *config.Config, listener overlay.NetworkEndpointListener) EndpointsController {
	return EndpointsController{Config: c, Store: store.New(c), Listener: listener}
}
