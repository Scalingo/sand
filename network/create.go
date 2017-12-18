package network

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/Scalingo/go-internal-tools/logger"
	"github.com/Scalingo/networking-agent/api/params"
	"github.com/Scalingo/networking-agent/api/types"
	"github.com/Scalingo/networking-agent/config"
	"github.com/Scalingo/networking-agent/network/overlay"
	"github.com/Scalingo/networking-agent/store"
	"github.com/pkg/errors"
)

type Repository interface {
	Create(ctx context.Context, params params.CreateNetworkParams) (types.Network, error)

	// If the network exists, return it, nil otherwise
	Exists(ctx context.Context, name string) (types.Network, error)
}

type repository struct {
	config *config.Config
	store  store.Store
}

func NewRepository(config *config.Config, store store.Store) Repository {
	return &repository{config: config, store: store}
}

func (c *repository) Create(ctx context.Context, params params.CreateNetworkParams) (types.Network, error) {
	log := logger.Get(ctx).WithField("network_name", params.Name)

	if params.Type == "" {
		params.Type = types.OverlayNetworkType
	}

	network, err := c.Exists(ctx, params.Name)
	if err != nil {
		return network, errors.Wrapf(err, "fail to check existance of network '%s'", params.Name)
	}

	if !network.CreatedAt.IsZero() {
		log.Info("existing network")
	} else {
		log.Info("creating new network")
		network, err = c.new(ctx, params)
		if err != nil {
			return network, errors.Wrapf(err, "fail to initialize network %s", params.Name)
		}
	}

	switch network.Type {
	case types.OverlayNetworkType:
		err = overlay.Ensure(ctx, c.config, network)
		if err != nil {
			return network, errors.Wrapf(err, "fail to ensure overlay network %s", network.Name)
		}
	default:
		return network, errors.Wrapf(err, "invalid network type")
	}

	return network, nil
}

func (c *repository) Exists(ctx context.Context, name string) (types.Network, error) {
	network := types.Network{
		Name: name,
	}

	err := c.store.Get(ctx, network.StorageKey(), false, &network)
	if err != nil && err != store.ErrNotFound {
		return network, errors.Wrapf(err, "fail to get network %s from store", network.Name)
	}
	if err == store.ErrNotFound {
		return network, nil
	}
	return network, err
}

func (c *repository) new(ctx context.Context, params params.CreateNetworkParams) (types.Network, error) {
	log := logger.Get(ctx)
	network := types.Network{
		CreatedAt: time.Now(),
		Name:      params.Name,
		Type:      params.Type,
		NSHandlePath: filepath.Join(
			c.config.NetnsPath, fmt.Sprintf("%s%s", c.config.NetnsPrefix, params.Name),
		),
	}

	vniGen := overlay.NewVNIGenerator(ctx, c.config, c.store)

	switch network.Type {
	case types.OverlayNetworkType:
		err := vniGen.Lock(ctx)
		if err != nil {
			return network, errors.Wrapf(err, "fail to lock VNI generator for %s", network.Name)
		}
		vni, err := vniGen.Generate(ctx)
		if err != nil {
			return network, errors.Wrapf(err, "fail to generate VNI for %s", network.Name)
		}

		log.Debugf("vni is %v", vni)
		network.VxLANVNI = vni
	default:
		return network, errors.New("invalid network type for init")
	}

	err := c.store.Set(ctx, network.StorageKey(), &network)
	if err != nil {
		return network, errors.Wrapf(err, "fail to get network %s from store", network.Name)
	}

	if vniGen != nil {
		err := vniGen.Unlock(ctx)
		if err != nil {
			log.WithError(err).Errorf("fail to unlock VNI generator for %s", network.Name)
		}
	}
	return network, nil
}
