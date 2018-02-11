package network

import (
	"context"

	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/store"
	"github.com/pkg/errors"
)

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
