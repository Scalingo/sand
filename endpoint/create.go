package endpoint

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/Scalingo/sand/api/params"
	"github.com/Scalingo/sand/api/types"
	"github.com/pborman/uuid"
	"github.com/pkg/errors"
)

func (r *repository) Create(ctx context.Context, n types.Network, params params.EndpointCreate) (types.Endpoint, error) {
	var endpoint types.Endpoint

	macAddress, err := ipv4ToMac(params.IPv4Address)
	if err != nil {
		return endpoint, errors.Wrapf(err, "fail to get MAC address from IP")
	}
	if params.MacAddress != "" {
		macAddress = params.MacAddress
	}

	endpoint = types.Endpoint{
		ID:            uuid.NewRandom().String(),
		Hostname:      r.config.PublicHostname,
		HostIP:        r.config.PublicIP,
		NetworkID:     n.ID,
		CreatedAt:     time.Now(),
		TargetVethIP:  params.IPv4Address,
		TargetVethMAC: macAddress,
	}

	err = r.store.Set(ctx, endpoint.StorageKey(), &endpoint)
	if err != nil {
		return endpoint, errors.Wrapf(err, "fail to save endpoint %s in store", endpoint)
	}

	err = r.store.Set(ctx, endpoint.NetworkStorageKey(), &endpoint)
	if err != nil {
		return endpoint, errors.Wrapf(err, "fail to save endpoint %s in store network", endpoint)
	}

	if params.Activate {
		endpoint, err = r.Activate(ctx, n, endpoint, params.ActivateParams)
		if err != nil {
			return endpoint, errors.Wrapf(err, "fail to ensure endpoint")
		}
	}

	return endpoint, nil
}

func ipv4ToMac(ipstr string) (string, error) {
	ip, _, err := net.ParseCIDR(ipstr)
	ip = ip.To4()
	if err != nil {
		return "", errors.Wrapf(err, "invalid CIDR %v", ipstr)
	}
	return fmt.Sprintf("02:84:%02x:%02x:%02x:%02x", ip[0], ip[1], ip[2], ip[3]), nil
}
