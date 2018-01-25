package network

import (
	"context"
	"fmt"
	"net"
	"path/filepath"
	"time"

	"github.com/Scalingo/go-internal-tools/logger"
	"github.com/Scalingo/sand/api/params"
	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/config"
	"github.com/Scalingo/sand/ipallocator"
	"github.com/Scalingo/sand/network/netmanager"
	"github.com/Scalingo/sand/network/overlay"
	"github.com/Scalingo/sand/store"
	"github.com/pborman/uuid"
	"github.com/pkg/errors"
)

const (
	DefaultIPRange = "10.0.0.0/24"
)

type Repository interface {
	Create(ctx context.Context, params params.CreateNetworkParams) (types.Network, error)
	Ensure(ctx context.Context, network types.Network) error
	Delete(ctx context.Context, network types.Network) error

	// If the network exists, return it, nil otherwise
	Exists(ctx context.Context, name string) (types.Network, bool, error)
}

type repository struct {
	config   *config.Config
	store    store.Store
	listener overlay.NetworkEndpointListener
	managers map[types.NetworkType]netmanager.NetManager
}

func NewRepository(config *config.Config, store store.Store, listener overlay.NetworkEndpointListener) Repository {
	return &repository{
		config: config, store: store, listener: listener,
		managers: map[types.NetworkType]netmanager.NetManager{
			types.OverlayNetworkType: overlay.NewManager(config, listener),
		},
	}
}

func (c *repository) Ensure(ctx context.Context, network types.Network) error {
	switch network.Type {
	case types.OverlayNetworkType:
		m := c.managers[network.Type]
		err := m.Ensure(ctx, network)
		if err != nil {
			return errors.Wrapf(err, "fail to ensure overlay network %s", network)
		}
		var endpoints []types.Endpoint
		err = c.store.Get(ctx, network.EndpointsStorageKey(""), true, &endpoints)
		if err != nil && err != store.ErrNotFound {
			return errors.Wrapf(err, "fail to get network endpoints")
		}

		if len(endpoints) > 0 {
			err = m.EnsureEndpointsNeigh(ctx, network, endpoints)
			if err != nil {
				return errors.Wrapf(err, "fail to ensure neighbors (ARP/FDB)")
			}
		}

		_, err = c.listener.Add(ctx, m, network)
		if err != nil {
			return errors.Wrapf(err, "fail to listen for new endpoints on network '%s'", network)
		}
	default:
		return errors.New("invalid network type")
	}

	// Ability to list all networks with node hostname as prefix
	err := c.store.Set(
		ctx,
		fmt.Sprintf("/nodes/%s/networks/%s", c.config.PublicHostname, network.ID),
		map[string]interface{}{"id": network.ID, "created_at": time.Now()},
	)
	if err != nil {
		return errors.Wrapf(err, "err to store nodes link to network %s", network)
	}

	// Ability to list nodes present in a network
	err = c.store.Set(
		ctx,
		fmt.Sprintf("/nodes-networks/%s/%s", network.ID, c.config.PublicHostname),
		map[string]interface{}{"id": network.ID, "created_at": time.Now()},
	)
	if err != nil {
		return errors.Wrapf(err, "err to store network %s link to hostname", network)
	}

	return nil
}

func (c *repository) Create(ctx context.Context, params params.CreateNetworkParams) (types.Network, error) {
	log := logger.Get(ctx).WithField("network_name", params.Name)

	if params.Type == "" {
		params.Type = types.OverlayNetworkType
	}

	network, err := c.new(ctx, params)
	if err != nil {
		return network, errors.Wrapf(err, "fail to initialize network %s", params.Name)
	}

	log = log.WithField("network_id", network.ID)
	ctx = logger.ToCtx(ctx, log)

	err = c.Ensure(ctx, network)
	if err != nil {
		return network, errors.Wrapf(err, "fail to ensure network %s", network)
	}

	return network, nil
}

func (c *repository) Exists(ctx context.Context, id string) (types.Network, bool, error) {
	network := types.Network{
		ID: id,
	}
	if id == "" {
		return network, false, nil
	}

	err := c.store.Get(ctx, network.StorageKey(), false, &network)
	if err != nil && err != store.ErrNotFound {
		return network, false, errors.Wrapf(err, "fail to get network %s from store", network.Name)
	}
	if err == store.ErrNotFound {
		return network, false, nil
	}
	return network, true, err
}

func (c *repository) new(ctx context.Context, params params.CreateNetworkParams) (types.Network, error) {
	log := logger.Get(ctx)
	uuid := uuid.NewRandom().String()

	iprange := DefaultIPRange
	if params.IPRange != "" {
		_, _, err := net.ParseCIDR(params.IPRange)
		if err != nil {
			return types.Network{}, errors.Wrapf(err, "invalid IP CIDR")
		}
		iprange = params.IPRange
	}

	allocator := ipallocator.New(c.config, c.store, uuid, ipallocator.WithIPRange(iprange))
	ip, mask, err := allocator.AllocateIP(ctx)
	if err != nil {
		return types.Network{}, errors.Wrapf(err, "fail to allocate gateway IP")
	}

	log.Infof("gateway IP allocated: %s/%d", ip, mask)

	network := types.Network{
		ID:        uuid,
		IPRange:   iprange,
		Gateway:   fmt.Sprintf("%s/%d", ip.String(), mask),
		CreatedAt: time.Now(),
		Name:      params.Name,
		Type:      params.Type,
		NSHandlePath: filepath.Join(
			c.config.NetnsPath, fmt.Sprintf("%s%s", c.config.NetnsPrefix, uuid),
		),
	}

	vniGen := overlay.NewVNIGenerator(ctx, c.config, c.store)

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

	err = c.store.Set(ctx, network.StorageKey(), &network)
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
