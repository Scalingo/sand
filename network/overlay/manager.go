package overlay

import (
	"github.com/Scalingo/sand/config"
)

type manager struct {
	config   *config.Config
	listener NetworkEndpointListener
}

func NewManager(c *config.Config, listener NetworkEndpointListener) manager {
	return manager{config: c, listener: listener}
}
