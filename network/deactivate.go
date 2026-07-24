package network

import (
	"context"
	"fmt"

	"gopkg.in/errgo.v1"

	"github.com/Scalingo/go-utils/errors/v3"

	"github.com/Scalingo/go-utils/logger"
	"github.com/Scalingo/sand/api/types"
)

func (c *repository) Deactivate(ctx context.Context, network types.Network) error {
	m := c.managers.Get(network.Type)

	switch network.Type {
	case types.OverlayNetworkType:
		err := m.Deactivate(ctx, network)
		if err != nil {
			return errgo.Notef(err, "deactive overlay network")
		}

		err = m.StopListenNetworkChange(ctx, network)
		if err != nil {
			return errors.Wrapf(ctx, err, "stop listening for endpoints change on network '%s'", network)
		}
	default:
		return errors.New(ctx, "unknown network type")
	}

	err := c.deleteNodeFromStore(ctx, c.config.GetPeerHostname(), network)
	if err != nil {
		return errors.Wrapf(ctx, err, "delete network from store")
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
		return errors.Wrapf(ctx, err, "delete network-host link %s from store", network)
	}
	err = c.store.Delete(
		ctx,
		fmt.Sprintf("/nodes-networks/%s/%s", network.ID, hostname),
	)
	if err != nil {
		return errors.Wrapf(ctx, err, "delete network-host link %s from store", network)
	}
	return nil
}
