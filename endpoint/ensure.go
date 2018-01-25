package endpoint

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/Scalingo/sand/api/params"
	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/ipallocator"
	"github.com/Scalingo/sand/network/overlay"
	"github.com/pborman/uuid"
	"github.com/pkg/errors"
)

func (r *repository) Create(ctx context.Context, n types.Network, params params.CreateEndpointParams) (types.Endpoint, error) {
	endpoint, ok, err := r.Exists(ctx, n, params.NSHandlePath)
	if err != nil {
		return endpoint, errors.Wrapf(err, "fail to get existance of endpoint in %s (store)", params.NSHandlePath)
	}

	if !ok {
		allocator := ipallocator.New(r.config, r.store, n.ID, ipallocator.WithIPRange(n.IPRange))
		ip, mask, err := allocator.AllocateIP(ctx)
		if err != nil {
			return endpoint, errors.Wrapf(err, "fail to allocate IP for endpoint")
		}

		endpoint = types.Endpoint{
			ID:              uuid.NewRandom().String(),
			Hostname:        r.config.PublicHostname,
			HostIP:          r.config.PublicIP,
			NetworkID:       n.ID,
			CreatedAt:       time.Now(),
			TargetVethIP:    fmt.Sprintf("%s/%d", ip.String(), mask),
			TargetVethMAC:   ipv4ToMac(ip),
			TargetNetnsPath: params.NSHandlePath,
		}
	}

	return r.Ensure(ctx, n, endpoint)
}

func (r *repository) Ensure(ctx context.Context, n types.Network, endpoint types.Endpoint) (types.Endpoint, error) {
	var err error

	switch n.Type {
	case types.OverlayNetworkType:
		endpoint, err = overlay.EnsureEndpoint(ctx, n, endpoint)
		if err != nil {
			return endpoint, errors.Wrapf(err, "fail to ensure endpoint")
		}
	default:
		return endpoint, errors.New("unknown network type")
	}

	err = r.store.Set(ctx, endpoint.StorageKey(), &endpoint)
	if err != nil {
		return endpoint, errors.Wrapf(err, "fail to save endpoint %s in store", endpoint)
	}

	err = r.store.Set(ctx, endpoint.NetworkStorageKey(), &endpoint)
	if err != nil {
		return endpoint, errors.Wrapf(err, "fail to save endpoint %s in store network", endpoint)
	}

	return endpoint, nil
}

func ipv4ToMac(ip net.IP) string {
	ip = ip.To4()
	return fmt.Sprintf("02:42:%02x:%02x:%02x:%02x", ip[0], ip[1], ip[2], ip[3])
}
