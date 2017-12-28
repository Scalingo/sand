package netmanager

import (
	"context"

	"github.com/Scalingo/sand/api/types"
)

type NetManager interface {
	Ensure(context.Context, types.Network) error
	EnsureEndpointsNeigh(context.Context, types.Network, []types.Endpoint) error
	AddEndpointNeigh(context.Context, types.Network, types.Endpoint) error
}
