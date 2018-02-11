package overlay

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"sync"

	"github.com/Scalingo/go-internal-tools/logger"
	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/config"
	"github.com/Scalingo/sand/network/netmanager"
	"github.com/Scalingo/sand/store"
	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type NetworkEndpointListener interface {
	Add(context.Context, netmanager.NetManager, types.Network) (chan struct{}, error)
	Remove(context.Context, types.Network) error
}

type listener struct {
	sync.Mutex
	config          *config.Config
	store           store.Store
	networkWatchers map[string]io.Closer

	// globalContext is the context used to start etcd watcher when it is
	// canceled all resources are released. We can't use the one of Add, as it is
	// often bount to a temporary http request, the context gets canceled
	// directly at the end of the request and the watch breaks
	globalContext context.Context
}

func NewNetworkEndpointListener(ctx context.Context, config *config.Config, store store.Store) NetworkEndpointListener {
	return &listener{config: config, store: store, networkWatchers: map[string]io.Closer{}, globalContext: ctx}
}

func (l *listener) Remove(ctx context.Context, network types.Network) error {
	l.Lock()
	defer l.Unlock()

	if w, ok := l.networkWatchers[network.ID]; !ok {
		return nil
	} else {
		err := w.Close()
		if err != nil {
			return errors.Wrapf(err, "fail to stop watcher")
		}
		delete(l.networkWatchers, network.ID)
	}
	return nil
}

func (l *listener) Add(ctx context.Context, nm netmanager.NetManager, network types.Network) (chan struct{}, error) {
	log := logger.Get(ctx)
	l.Lock()
	defer l.Unlock()

	if _, ok := l.networkWatchers[network.ID]; ok {
		return nil, nil
	}

	w, err := l.store.Watch(l.globalContext, network.EndpointsStorageKey(""))
	if err != nil {
		return nil, errors.Wrapf(err, "fail to create watcher for network %s", network)
	}

	l.networkWatchers[network.ID] = w

	done := make(chan struct{})
	go func(w store.Watcher) {
		defer close(done)
		log.Debug("start listening to etcd watch chan")
		for {
			resp, ok := w.NextResponse()
			if !ok {
				break
			}
			err := l.handleMessage(l.globalContext, resp, nm, network)
			if err != nil {
				log.WithError(err).Error("fail to handle watch response")
			}
		}
		log.Debug("watcher closed")
	}(w)

	return done, nil
}

func (l *listener) handleMessage(ctx context.Context, resp clientv3.WatchResponse, nm netmanager.NetManager, network types.Network) error {
	log := logger.Get(ctx)

	if err := resp.Err(); err != nil {
		return errors.Wrapf(err, "error when watching events")
	}
	for _, event := range resp.Events {
		switch event.Type {
		case mvccpb.PUT:
			var endpoint types.Endpoint
			err := json.NewDecoder(bytes.NewReader(event.Kv.Value)).Decode(&endpoint)
			if err != nil {
				return errors.Wrapf(err, "fail to decode JSON")
			}

			log = log.WithFields(logrus.Fields{
				"endpoint_id":        endpoint.ID,
				"endpoint_target_ip": endpoint.TargetVethIP,
				"endpoint_hostname":  endpoint.Hostname,
			})
			log.Info("etcd watch got new endpoint")
			ctx = logger.ToCtx(ctx, log)

			err = nm.AddEndpointNeigh(ctx, network, endpoint)
			if err != nil {
				log.WithError(err).Error("fail to add endpoint ARP/FDB neigh rules")
			}

		case mvccpb.DELETE:
			var endpoint types.Endpoint
			err := l.store.GetWithRevision(ctx, string(event.Kv.Key), event.Kv.ModRevision-1, false, &endpoint)
			if err != nil {
				return errors.Wrapf(err, "fail to get endpoint %v", string(event.Kv.Key))
			}

			log = log.WithFields(logrus.Fields{
				"endpoint_id":        endpoint.ID,
				"endpoint_target_ip": endpoint.TargetVethIP,
				"endpoint_hostname":  endpoint.Hostname,
			})
			log.Info("etcd watch got deleted endpoint")
			ctx = logger.ToCtx(ctx, log)

			err = nm.RemoveEndpointNeigh(ctx, network, endpoint)
			if err != nil {
				log.WithError(err).Error("fail to add endpoint ARP/FDB neigh rules")
			}
		}
	}
	return nil
}
