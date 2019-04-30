package netmanager

import (
	"context"
	"errors"

	"github.com/Scalingo/sand/api/params"
	"github.com/Scalingo/sand/api/types"
)

type NetManager interface {
	Ensure(context.Context, types.Network) error
	Deactivate(context.Context, types.Network) error

	EnsureEndpoint(context.Context, types.Network, types.Endpoint, params.EndpointActivate) (types.Endpoint, error)
	DeleteEndpoint(context.Context, types.Network, types.Endpoint) error

	EnsureEndpointsNeigh(context.Context, types.Network, []types.Endpoint) error
	AddEndpointNeigh(context.Context, types.Network, types.Endpoint) error
	RemoveEndpointNeigh(context.Context, types.Network, types.Endpoint) error

	ListenNetworkChange(context.Context, types.Network) error
}

var EndpointAlreadyDisabledErr = errors.New("endpoint already disabled")
