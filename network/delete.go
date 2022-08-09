package network

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/Scalingo/go-utils/logger"
	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/ipallocator"
	"github.com/Scalingo/sand/store"
)

func (c *repository) Delete(ctx context.Context, network types.Network, a ipallocator.IPAllocator) error {
	log := logger.Get(ctx)
	log.Info("Delete network")

	var nets []types.Network
	err := c.store.Get(
		ctx,
		fmt.Sprintf("/nodes-networks/%s/", network.ID),
		true,
		&nets,
	)
	if err != nil && err != store.ErrNotFound {
		return errors.Wrapf(err, "fail to get list of link nodes-networks of %s", network)
	}

	if len(nets) == 0 {
		log.Infof("Deleting network %v definition", network)
		err = c.store.Delete(ctx, network.StorageKey())
		if err != nil {
			return errors.Wrapf(err, "fail to delete network %s from store", network)
		}

		err = a.ReleasePool(ctx, network.ID)
		if err != nil {
			return errors.Wrapf(err, "fail to release pool of network %s", network)
		}
	} else {
		log.Infof("Network still on %d hosts, keeping %v definition", len(nets), network)
	}
	return nil
}
