package node

import (
	"context"
	"os"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

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
	if errors.Cause(err) == store.ErrNotFound {
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "failed to list endpoints of %v", c.GetPeerHostname())
	}

	for _, endpoint := range endpoints {
		endpointLog := log.WithFields(logrus.Fields{
			"endpoint_id": endpoint.ID,
		})
		endpointCtx := logger.ToCtx(ctx, endpointLog)

		if !endpoint.Active {
			endpointLog.Debug("skip inactive endpoint")
			continue
		}

		endpointLog = endpointLog.WithFields(logrus.Fields{
			"network_id":          endpoint.NetworkID,
			"endpoint_netns_path": endpoint.TargetNetnsPath,
		})
		endpointCtx = logger.ToCtx(ctx, endpointLog)
		endpointLog.Info("restoring endpoint")

		network, ok, err := repo.Exists(endpointCtx, endpoint.NetworkID)
		if err != nil {
			return errors.Wrapf(err, "failed to get network")
		}
		if !ok {
			endpointLog.WithError(errors.Errorf("network not found for %v", endpoint)).Error("skip endpoint")
			continue
		}

		endpointLog.Info("ensuring network")
		err = repo.Ensure(endpointCtx, network)
		if err != nil {
			endpointLog.WithError(err).Error("failed to ensure network")
			continue
		}

		_, err = erepo.Activate(endpointCtx, network, endpoint, params.EndpointActivate{
			NSHandlePath: endpoint.TargetNetnsPath,
			SetAddr:      true,
			MoveVeth:     true,
		})
		if err != nil {
			// If the netns path no longer exists, deactivate the endpoint. Other
			// activation errors should not prevent the remaining endpoints from being ensured.
			if os.IsNotExist(errors.Cause(err)) {
				_, err = erepo.Deactivate(endpointCtx, network, endpoint)
				if err != nil {
					endpointLog.WithError(err).Error("failed to deactivate endpoint")
					continue
				}
			} else {
				endpointLog.WithError(err).Error("failed to ensure endpoint")
				continue
			}
		}
	}

	return nil
}
