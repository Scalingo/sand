package web

import (
	"github.com/Scalingo/networking-agent/config"
	"github.com/Scalingo/networking-agent/network/overlay"
	"github.com/Scalingo/networking-agent/store"
)

type EndpointsController struct {
	Config   *config.Config
	Store    store.Store
	Listener overlay.NetworkEndpointListener
}

func NewEndpointsController(c *config.Config, listener overlay.NetworkEndpointListener) EndpointsController {
	return EndpointsController{Config: c, Store: store.New(c), Listener: listener}
}
