package network

import (
	"context"
	"net"

	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/ipallocator"
	"github.com/pkg/errors"
)

func (r *repository) DeleteEndpoint(ctx context.Context, n types.Network, e types.Endpoint) error {
	err := r.managers[types.OverlayNetworkType].DeleteEndpoint(ctx, n, e)
	if err != nil {
		return errors.Wrapf(err, "fail to delete endpoint from overlay network")
	}

	allocator := ipallocator.New(r.config, r.store, n.ID, ipallocator.WithIPRange(n.IPRange))

	ip, _, err := net.ParseCIDR(e.TargetVethIP)
	if err != nil {
		return errors.Wrapf(err, "fail to parse IP from endpoint")
	}

	err = allocator.ReleaseIP(ctx, ip)
	if err != nil {
		return errors.Wrapf(err, "fail to release IP for endpoint")
	}
	return nil
}
