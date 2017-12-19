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
	"github.com/pborman/uuid"
	"github.com/pkg/errors"
)

type Repository interface {
	Create(ctx context.Context, params params.CreateNetworkParams) (types.Network, error)
	Ensure(ctx context.Context, network types.Network) error
	Delete(ctx context.Context, network types.Network) error

	// If the network exists, return it, nil otherwise
	Exists(ctx context.Context, name string) (types.Network, bool, error)
}

type repository struct {
	config *config.Config
	store  store.Store
}

func NewRepository(config *config.Config, store store.Store) Repository {
	return &repository{config: config, store: store}
}

func (c *repository) Ensure(ctx context.Context, network types.Network) error {
	switch network.Type {
	case types.OverlayNetworkType:
		err := overlay.Ensure(ctx, c.config, network)
		if err != nil {
			return errors.Wrapf(err, "fail to ensure overlay network %s", network)
		}
	default:
		return errors.New("invalid network type")
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
	uuid := uuid.NewUUID().String()
	network := types.Network{
		ID:        uuid,
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

	err := c.store.Set(ctx, network.StorageKey(), &network)
	if err != nil {
		return network, errors.Wrapf(err, "fail to get network %s from store", network)
	}

	err = c.store.Set(
		ctx,
		fmt.Sprintf("/nodes/%s/networks/%s", c.config.PublicHostname, network.ID),
		map[string]interface{}{"id": network.ID, "created_at": time.Now()},
	)
	if err != nil {
		return network, errors.Wrapf(err, "err to store nodes link to network %s", network)
	}
	err = c.store.Set(
		ctx,
		fmt.Sprintf("/nodes-networks/%s/%s", network.ID, c.config.PublicHostname),
		map[string]interface{}{"id": network.ID, "created_at": time.Now()},
	)
	if err != nil {
		return network, errors.Wrapf(err, "err to store network %s link to hostname", network)
	}

	if vniGen != nil {
		err := vniGen.Unlock(ctx)
		if err != nil {
			log.WithError(err).Errorf("fail to unlock VNI generator for %s", network)
		}
	}
	return network, nil
}
