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

	EnsureEndpointsNeigh(context.Context, types.Network, []types.Endpoint) (EnsureEndpointsNeighResults, error)
	AddEndpointNeigh(context.Context, types.Network, types.Endpoint) (EndpointNeighResults, error)
	RemoveEndpointNeigh(context.Context, types.Network, types.Endpoint) (EndpointNeighResults, error)

	ListenNetworkChange(context.Context, types.Network) error
	StopListenNetworkChange(context.Context, types.Network) error
}

type EnsureEndpointsNeighResults struct {
	AddedARPEntries   int
	AddedFDBEntries   int
	RemovedARPEntries int
	RemovedFDBEntries int
}

type EndpointNeighResults struct {
	AddedARPEntry   bool
	AddedFDBEntry   bool
	RemovedARPEntry bool
	RemovedFDBEntry bool
}

var EndpointAlreadyDisabledErr = errors.New("endpoint already disabled")
