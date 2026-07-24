package node

import (
	"context"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/Scalingo/go-utils/errors/v3"
	"github.com/Scalingo/go-utils/logger"
	"github.com/Scalingo/sand/api/params"
	"github.com/Scalingo/sand/config"
	"github.com/Scalingo/sand/endpoint"
	"github.com/Scalingo/sand/network"
	"github.com/Scalingo/sand/store"
)

func EnsureNetworkEndpoints(ctx context.Context, c *config.Config, repo network.Repository, erepo endpoint.Repository) error {
	log := logger.Get(ctx)
	ctx = logger.ToCtx(ctx, log)

	log.Info("Ensure networks on node")

	endpoints, err := erepo.List(ctx, map[string]string{"hostname": c.GetPeerHostname()})
	if errors.Is(err, store.ErrNotFound) {
		return nil
	}
	if err != nil {
		return errors.Wrapf(ctx, err, "list endpoints of %v", c.GetPeerHostname())
	}

	for _, endpoint := range endpoints {
		ctx, log := logger.WithFieldsToCtx(ctx, logrus.Fields{
			"endpoint_id": endpoint.ID,
		})

		if !endpoint.Active {
			log.Debug("skip inactive endpoint")
			continue
		}

		ctx, log = logger.WithFieldsToCtx(ctx, logrus.Fields{
			"network_id":          endpoint.NetworkID,
			"endpoint_netns_path": endpoint.TargetNetnsPath,
		})
		log.Info("restoring endpoint")

		network, ok, err := repo.Exists(ctx, endpoint.NetworkID)
		if err != nil {
			return errors.Wrapf(ctx, err, "get network")
		}
		if !ok {
			log.WithError(errors.Errorf(ctx, "network not found for %v", endpoint)).Error("Failed to find network, skip endpoint")
			continue
		}

		log.Info("ensuring network")
		err = repo.Ensure(ctx, network)
		if err != nil {
			log.WithError(err).Error("Failed to ensure network")
			continue
		}

		_, err = erepo.Activate(ctx, network, endpoint, params.EndpointActivate{
			NSHandlePath: endpoint.TargetNetnsPath,
			SetAddr:      true,
			MoveVeth:     true,
		})
		if err != nil {
			// If the netns path no longer exists, deactivate the endpoint. Other
			// activation errors should not prevent the remaining endpoints from being ensured.
			if errors.Is(err, os.ErrNotExist) {
				_, err = erepo.Deactivate(ctx, network, endpoint)
				if err != nil {
					log.WithError(err).Error("Failed to deactivate endpoint")
					continue
				}
			} else {
				log.WithError(err).Error("Failed to ensure endpoint")
				continue
			}
		}
	}

	return nil
}
