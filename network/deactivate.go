package network

import (
	"context"
	"fmt"

	"gopkg.in/errgo.v1"

	"github.com/Scalingo/go-utils/logger"
	"github.com/Scalingo/sand/api/types"
	"github.com/pkg/errors"
)

func (c *repository) Deactivate(ctx context.Context, network types.Network) error {
	m := c.managers.Get(network.Type)

	switch network.Type {
	case types.OverlayNetworkType:
		err := m.Deactivate(ctx, network)
		if err != nil {
			return errgo.Notef(err, "fail to deactive overlay network")
		}

		err = m.StopListenNetworkChange(ctx, network)
		if err != nil {
			return errors.Wrapf(err, "fail to stop listening for endpoints change on network '%s'", network)
		}
	default:
		return errors.New("unknown network type")
	}

	err := c.deleteNodeFromStore(ctx, c.config.PublicHostname, network)
	if err != nil {
		return errors.Wrapf(err, "fail to delete network from store")
	}
	return nil
}

func (c *repository) deleteNodeFromStore(ctx context.Context, hostname string, network types.Network) error {
	log := logger.Get(ctx)
	log.WithField("host", hostname).Info("unlinking host")

	err := c.store.Delete(
		ctx,
		fmt.Sprintf("/nodes/%s/networks/%s", hostname, network.ID),
	)
	if err != nil {
		return errors.Wrapf(err, "fail to delete network-host link %s from store", network)
	}
	err = c.store.Delete(
		ctx,
		fmt.Sprintf("/nodes-networks/%s/%s", network.ID, hostname),
	)
	if err != nil {
		return errors.Wrapf(err, "fail to delete network-host link %s from store", network)
	}
	return nil
}
