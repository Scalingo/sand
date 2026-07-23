package metrics

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/Scalingo/go-utils/errors/v3"
)

type NodeEnsureNetworkEndpointsStatus string

const (
	NodeEnsureNetworkEndpointsStatusSuccess NodeEnsureNetworkEndpointsStatus = "success"
	NodeEnsureNetworkEndpointsStatusError   NodeEnsureNetworkEndpointsStatus = "error"
)

type NodeEnsureNetworkEndpointsExecution struct {
	Duration                 time.Duration
	Status                   NodeEnsureNetworkEndpointsStatus
	EndpointsListed          int
	EndpointsActive          int
	EndpointsInactive        int
	EndpointsEnsured         int
	EndpointsDeactivated     int
	NetworksEnsured          int
	NetworkARPEntriesAdded   int
	NetworkFDBEntriesAdded   int
	NetworkARPEntriesRemoved int
	NetworkFDBEntriesRemoved int
	NetworksMissing          int
	EndpointListErrors       int
	NetworkLookupErrors      int
	NetworkEnsureErrors      int
	EndpointEnsureErrors     int
	EndpointDeactivateErrors int
}

type nodeEnsureNetworkEndpointsMetrics struct {
	meter     metric.Meter
	duration  metric.Float64Histogram
	endpoints metric.Int64Gauge
	networks  metric.Int64Gauge
	errors    metric.Int64Gauge
}

const (
	nodeEnsureNetworkEndpointsMeter    = "scalingo.sand.node.ensure_network_endpoints"
	nodeEnsureNetworkEndpointsPrefix   = "scalingo.sand.node.ensure_network_endpoints"
	nodeEnsureNetworkEndpointsDuration = nodeEnsureNetworkEndpointsPrefix + ".duration"
	nodeEnsureNetworkEndpointsEndpoint = nodeEnsureNetworkEndpointsPrefix + ".endpoints"
	nodeEnsureNetworkEndpointsNetwork  = nodeEnsureNetworkEndpointsPrefix + ".networks"
	nodeEnsureNetworkEndpointsError    = nodeEnsureNetworkEndpointsPrefix + ".errors"

	attrNodeEnsureNetworkEndpointsStatus = "scalingo.node.ensure_network_endpoints.status"
	attrNodeEnsureNetworkEndpointsState  = "scalingo.node.ensure_network_endpoints.state"
	attrNodeEnsureNetworkEndpointsStep   = "scalingo.node.ensure_network_endpoints.step"
)

func (m *GlobalExporter) registerNodeEnsureNetworkEndpointsMetrics(ctx context.Context) error {
	nodeMetrics := &nodeEnsureNetworkEndpointsMetrics{}
	nodeMetrics.meter = otel.Meter(nodeEnsureNetworkEndpointsMeter)

	var err error
	nodeMetrics.duration, err = nodeMetrics.meter.Float64Histogram(
		nodeEnsureNetworkEndpointsDuration,
		metric.WithDescription("Duration of node network endpoints ensure execution"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return errors.Wrap(ctx, err, "register node network endpoints ensure duration metric")
	}

	nodeMetrics.endpoints, err = nodeMetrics.meter.Int64Gauge(
		nodeEnsureNetworkEndpointsEndpoint,
		metric.WithDescription("Number of endpoints handled during node network endpoints ensure execution"),
	)
	if err != nil {
		return errors.Wrap(ctx, err, "register node network endpoints ensure endpoints metric")
	}

	nodeMetrics.networks, err = nodeMetrics.meter.Int64Gauge(
		nodeEnsureNetworkEndpointsNetwork,
		metric.WithDescription("Number of networks handled during node network endpoints ensure execution"),
	)
	if err != nil {
		return errors.Wrap(ctx, err, "register node network endpoints ensure networks metric")
	}

	nodeMetrics.errors, err = nodeMetrics.meter.Int64Gauge(
		nodeEnsureNetworkEndpointsError,
		metric.WithDescription("Number of errors during node network endpoints ensure execution"),
	)
	if err != nil {
		return errors.Wrap(ctx, err, "register node network endpoints ensure errors metric")
	}

	m.nodeEnsureNetworkEndpointsMetrics = nodeMetrics
	return nil
}

func (m *GlobalExporter) NodeEnsureNetworkEndpointsWriteExecution(ctx context.Context, execution NodeEnsureNetworkEndpointsExecution) {
	if m.nodeEnsureNetworkEndpointsMetrics == nil {
		return
	}

	statusAttr := attribute.String(attrNodeEnsureNetworkEndpointsStatus, string(execution.Status))
	m.nodeEnsureNetworkEndpointsMetrics.duration.Record(ctx, execution.Duration.Seconds(), metric.WithAttributes(statusAttr))
	m.recordNodeEnsureEndpoint(ctx, execution.EndpointsListed, "listed", statusAttr)
	m.recordNodeEnsureEndpoint(ctx, execution.EndpointsActive, "active", statusAttr)
	m.recordNodeEnsureEndpoint(ctx, execution.EndpointsInactive, "inactive", statusAttr)
	m.recordNodeEnsureEndpoint(ctx, execution.EndpointsEnsured, "ensured", statusAttr)
	m.recordNodeEnsureEndpoint(ctx, execution.EndpointsDeactivated, "deactivated", statusAttr)
	m.recordNodeEnsureEndpoint(ctx, execution.NetworkARPEntriesAdded, "arp_added_by_network_ensure", statusAttr)
	m.recordNodeEnsureEndpoint(ctx, execution.NetworkFDBEntriesAdded, "fdb_added_by_network_ensure", statusAttr)
	m.recordNodeEnsureEndpoint(ctx, execution.NetworkARPEntriesRemoved, "arp_removed_by_network_ensure", statusAttr)
	m.recordNodeEnsureEndpoint(ctx, execution.NetworkFDBEntriesRemoved, "fdb_removed_by_network_ensure", statusAttr)
	m.recordNodeEnsureNetwork(ctx, execution.NetworksEnsured, "ensured", statusAttr)
	m.recordNodeEnsureNetwork(ctx, execution.NetworksMissing, "missing", statusAttr)
	m.recordNodeEnsureError(ctx, execution.EndpointListErrors, "endpoint_list", statusAttr)
	m.recordNodeEnsureError(ctx, execution.NetworkLookupErrors, "network_lookup", statusAttr)
	m.recordNodeEnsureError(ctx, execution.NetworkEnsureErrors, "network_ensure", statusAttr)
	m.recordNodeEnsureError(ctx, execution.EndpointEnsureErrors, "endpoint_ensure", statusAttr)
	m.recordNodeEnsureError(ctx, execution.EndpointDeactivateErrors, "endpoint_deactivate", statusAttr)
}

func (m *GlobalExporter) recordNodeEnsureEndpoint(ctx context.Context, value int, state string, attrs ...attribute.KeyValue) {
	attrs = append(attrs, attribute.String(attrNodeEnsureNetworkEndpointsState, state))
	m.nodeEnsureNetworkEndpointsMetrics.endpoints.Record(ctx, int64(value), metric.WithAttributes(attrs...))
}

func (m *GlobalExporter) recordNodeEnsureNetwork(ctx context.Context, value int, state string, attrs ...attribute.KeyValue) {
	attrs = append(attrs, attribute.String(attrNodeEnsureNetworkEndpointsState, state))
	m.nodeEnsureNetworkEndpointsMetrics.networks.Record(ctx, int64(value), metric.WithAttributes(attrs...))
}

func (m *GlobalExporter) recordNodeEnsureError(ctx context.Context, value int, step string, attrs ...attribute.KeyValue) {
	attrs = append(attrs, attribute.String(attrNodeEnsureNetworkEndpointsStep, step))
	m.nodeEnsureNetworkEndpointsMetrics.errors.Record(ctx, int64(value), metric.WithAttributes(attrs...))
}
