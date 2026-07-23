package network

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/Scalingo/go-utils/logger"
	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/store"
)

func (c *repository) Ensure(ctx context.Context, network types.Network) (EnsureResults, error) {
	log := logger.Get(ctx)
	log.Info("Ensure network setup")
	results := EnsureResults{}

	m := c.managers.Get(network.Type)

	switch network.Type {
	case types.OverlayNetworkType:
		err := m.Ensure(ctx, network)
		if err != nil {
			return results, errors.Wrapf(err, "fail to ensure overlay network %s", network)
		}
		var endpoints []types.Endpoint
		err = c.store.Get(ctx, network.EndpointsStorageKey(""), true, &endpoints)
		if err != nil && err != store.ErrNotFound {
			return results, errors.Wrapf(err, "fail to get network endpoints")
		}

		neighResults, err := m.EnsureEndpointsNeigh(ctx, network, endpoints)
		if err != nil {
			return results, errors.Wrapf(err, "fail to ensure neighbors (ARP/FDB)")
		}
		results.AddedARPEntries = neighResults.AddedARPEntries
		results.AddedFDBEntries = neighResults.AddedFDBEntries
		results.RemovedARPEntries = neighResults.RemovedARPEntries
		results.RemovedFDBEntries = neighResults.RemovedFDBEntries

		err = m.ListenNetworkChange(ctx, network)
		if err != nil {
			return results, errors.Wrapf(err, "fail to listen for new endpoints on network '%s'", network)
		}
	default:
		return results, errors.New("invalid network type")
	}

	// Ability to list all networks with node hostname as prefix
	err := c.store.Set(
		ctx,
		fmt.Sprintf("/nodes/%s/networks/%s", c.config.GetPeerHostname(), network.ID),
		map[string]interface{}{"id": network.ID, "created_at": time.Now()},
	)
	if err != nil {
		return results, errors.Wrapf(err, "err to store nodes link to network %s", network)
	}

	// Ability to list nodes present in a network
	err = c.store.Set(
		ctx,
		fmt.Sprintf("/nodes-networks/%s/%s", network.ID, c.config.GetPeerHostname()),
		map[string]interface{}{"id": network.ID, "created_at": time.Now()},
	)
	if err != nil {
		return results, errors.Wrapf(err, "err to store network %s link to hostname", network)
	}

	log.Info("Network setup ensured")
	return results, nil
}
