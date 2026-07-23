package node

import (
	"context"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/Scalingo/go-utils/logger"
	"github.com/Scalingo/sand/api/params"
	"github.com/Scalingo/sand/config"
	"github.com/Scalingo/sand/endpoint"
	"github.com/Scalingo/sand/metrics"
	"github.com/Scalingo/sand/network"
	"github.com/Scalingo/sand/store"
)

func EnsureNetworkEndpoints(ctx context.Context, c *config.Config, repo network.Repository, erepo endpoint.Repository, metricsExporter metrics.Exporter) (err error) {
	log := logger.Get(ctx)
	ctx = logger.ToCtx(ctx, log)
	startedAt := time.Now()
	execution := metrics.NodeEnsureNetworkEndpointsExecution{}
	defer func() {
		execution.Duration = time.Since(startedAt)
		execution.Status = metrics.NodeEnsureNetworkEndpointsStatusSuccess
		if err != nil {
			execution.Status = metrics.NodeEnsureNetworkEndpointsStatusError
		}
		if metricsExporter != nil {
			metricsExporter.NodeEnsureNetworkEndpointsWriteExecution(ctx, execution)
		}
	}()

	log.Info("Ensure networks on node")

	endpoints, err := erepo.List(ctx, map[string]string{"hostname": c.GetPeerHostname()})
	if errors.Cause(err) == store.ErrNotFound {
		return nil
	}
	if err != nil {
		execution.EndpointListErrors++
		return errors.Wrapf(err, "failed to list endpoints of %v", c.GetPeerHostname())
	}
	execution.EndpointsListed = len(endpoints)

	for _, endpoint := range endpoints {
		endpointLog := log.WithFields(logrus.Fields{
			"endpoint_id": endpoint.ID,
		})
		endpointCtx := logger.ToCtx(ctx, endpointLog)

		if !endpoint.Active {
			execution.EndpointsInactive++
			endpointLog.Debug("skip inactive endpoint")
			continue
		}
		execution.EndpointsActive++

		endpointLog = endpointLog.WithFields(logrus.Fields{
			"network_id":          endpoint.NetworkID,
			"endpoint_netns_path": endpoint.TargetNetnsPath,
		})
		endpointCtx = logger.ToCtx(ctx, endpointLog)
		endpointLog.Info("restoring endpoint")

		network, ok, err := repo.Exists(endpointCtx, endpoint.NetworkID)
		if err != nil {
			execution.NetworkLookupErrors++
			return errors.Wrapf(err, "failed to get network")
		}
		if !ok {
			execution.NetworksMissing++
			endpointLog.WithError(errors.Errorf("network not found for %v", endpoint)).Error("skip endpoint")
			continue
		}

		endpointLog.Info("ensuring network")
		results, err := repo.Ensure(endpointCtx, network)
		if err != nil {
			execution.NetworkEnsureErrors++
			endpointLog.WithError(err).Error("failed to ensure network")
			continue
		}
		execution.NetworksEnsured++
		execution.NetworkARPEntriesAdded += results.AddedARPEntries
		execution.NetworkFDBEntriesAdded += results.AddedFDBEntries
		execution.NetworkARPEntriesRemoved += results.RemovedARPEntries
		execution.NetworkFDBEntriesRemoved += results.RemovedFDBEntries

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
					execution.EndpointDeactivateErrors++
					endpointLog.WithError(err).Error("failed to deactivate endpoint")
					continue
				}
				execution.EndpointsDeactivated++
			} else {
				execution.EndpointEnsureErrors++
				endpointLog.WithError(err).Error("failed to ensure endpoint")
				continue
			}
		} else {
			execution.EndpointsEnsured++
		}
	}

	return nil
}
