package docker

import (
	"github.com/Scalingo/sand/config"
	"github.com/Scalingo/sand/endpoint"
	"github.com/Scalingo/sand/ipallocator"
	sandnetwork "github.com/Scalingo/sand/network"
)

type DockerPlugin struct {
	DockerNetworkPlugin *dockerNetworkPlugin
	DockerIPAMPlugin    *dockerIPAMPlugin
}

func NewDockerPlugin(c *config.Config, nr sandnetwork.Repository, er endpoint.Repository, r Repository, a ipallocator.IPAllocator) *DockerPlugin {
	return &DockerPlugin{
		DockerNetworkPlugin: &dockerNetworkPlugin{
			networkRepository:      nr,
			endpointRepository:     er,
			dockerPluginRepository: r,
		},
		DockerIPAMPlugin: &dockerIPAMPlugin{
			allocator:         a,
			networkRepository: nr,
		},
	}
}
