package metrics

import (
	"context"

	"github.com/Scalingo/go-utils/logger"
)

type Exporter interface {
	NodeEnsureNetworkEndpointsWriteExecution(context.Context, NodeEnsureNetworkEndpointsExecution)
}

type GlobalExporter struct {
	nodeEnsureNetworkEndpointsMetrics *nodeEnsureNetworkEndpointsMetrics
}

func Register(ctx context.Context) (*GlobalExporter, error) {
	log := logger.Get(ctx)

	exporter := &GlobalExporter{}

	err := exporter.registerNodeEnsureNetworkEndpointsMetrics(ctx)
	if err != nil {
		log.WithError(err).Error("Register node ensure network endpoints metrics")
		return nil, err
	}

	return exporter, nil
}
