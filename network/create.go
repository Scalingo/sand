package network

import (
	"context"
	"fmt"
	"net"
	"path/filepath"
	"regexp"
	"time"

	"github.com/Scalingo/go-internal-tools/logger"
	"github.com/Scalingo/sand/api/params"
	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/ipallocator"
	"github.com/Scalingo/sand/network/overlay"
	"github.com/pborman/uuid"
	"github.com/pkg/errors"
)

var (
	uuidRegexp = regexp.MustCompile("(?i)^[0-9A-F]{8}-[0-9A-F]{4}-4[0-9A-F]{3}-[89AB][0-9A-F]{3}-[0-9A-F]{12}$")
)

func (r *repository) Create(ctx context.Context, params params.NetworkCreate) (types.Network, error) {
	log := logger.Get(ctx).WithField("network_name", params.Name)

	if params.Type == "" {
		params.Type = types.OverlayNetworkType
	}

	uuid := uuid.NewRandom().String()
	if params.ID != "" {
		if !uuidRegexp.MatchString(params.ID) {
			return types.Network{}, errors.Errorf("invalid UUID %v", params.ID)
		}
		uuid = params.ID
	}

	iprange := DefaultIPRange
	if params.IPRange != "" {
		_, _, err := net.ParseCIDR(params.IPRange)
		if err != nil {
			return types.Network{}, errors.Wrapf(err, "invalid IP CIDR")
		}
		iprange = params.IPRange
	}

	ip, err := r.allocator.AllocateIP(ctx, uuid, ipallocator.AllocateIPOpts{
		AddressRange: iprange,
	})
	if err != nil {
		return types.Network{}, errors.Wrapf(err, "fail to allocate gateway IP")
	}
	log.Infof("gateway IP allocated: %s", ip)

	if params.Name == "" {
		params.Name = fmt.Sprintf("net-sc-%s", uuid)
	}

	network := types.Network{
		ID:        uuid,
		IPRange:   iprange,
		Gateway:   ip,
		CreatedAt: time.Now(),
		Name:      params.Name,
		Type:      params.Type,
		NSHandlePath: filepath.Join(
			r.config.NetnsPath, fmt.Sprintf("%s%s", r.config.NetnsPrefix, uuid),
		),
	}

	vniGen := overlay.NewVNIGenerator(ctx, r.config, r.store)

	switch network.Type {
	case types.OverlayNetworkType:
		err := vniGen.Lock(ctx)
		if err != nil {
			return network, errors.Wrapf(err, "fail to lock VNI generator for %s", network)
		}
		vni, err := vniGen.Generate(ctx)
		if err != nil {
			return network, errors.Wrapf(err, "fail to generate VNI for %s", network)
		}

		log.Debugf("vni is %v", vni)
		network.VxLANVNI = vni
	default:
		return network, errors.New("invalid network type for init")
	}

	err = r.store.Set(ctx, network.StorageKey(), &network)
	if err != nil {
		return network, errors.Wrapf(err, "fail to get network %s from store", network)
	}

	if vniGen != nil {
		err := vniGen.Unlock(ctx)
		if err != nil {
			log.WithError(err).Errorf("fail to unlock VNI generator for %s", network)
		}
	}
	return network, nil
}
