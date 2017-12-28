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
	Add(context.Context, netmanager.NetManager, types.Network) error
	Remove(context.Context, types.Network) error
}

type listener struct {
	sync.Mutex
	config          *config.Config
	store           store.Store
	networkWatchers map[string]io.Closer
}

func NewNetworkEndpointListener(config *config.Config, store store.Store) NetworkEndpointListener {
	return &listener{config: config, store: store, networkWatchers: map[string]io.Closer{}}
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

func (l *listener) Add(ctx context.Context, nm netmanager.NetManager, network types.Network) error {
	log := logger.Get(ctx)
	l.Lock()
	defer l.Unlock()

	if _, ok := l.networkWatchers[network.ID]; ok {
		return nil
	}

	w, closer, err := l.store.Watch(ctx, network.EndpointsStorageKey(""))
	if err != nil {
		return errors.Wrapf(err, "fail to create watcher for network %s", network)
	}

	l.networkWatchers[network.ID] = closer

	go func(wchan clientv3.WatchChan) {
		defer func() {
			err := closer.Close()
			if err != nil {
				log.WithError(err).Error("fail to stop etcd watcher")
			}
		}()
		log.Debug("start listening to etcd watch chan")
		for resp := range wchan {
			if err := resp.Err(); err != nil {
				log.WithError(err).Error("error when watching events")
			}
			for _, event := range resp.Events {
				switch event.Type {
				case mvccpb.PUT:
					var endpoint types.Endpoint
					err := json.NewDecoder(bytes.NewReader(event.Kv.Value)).Decode(&endpoint)
					if err != nil {
						log.WithError(err).Error("fail to decode JSON")
						continue
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
						log.WithError(err).Errorf("fail to add endpoint '%s' ARP/FDB neigh rules", endpoint)
					}

				case mvccpb.DELETE:
				}
			}
		}
		log.Debug("etcd watch chan is closed")
	}(w)
	return nil
}
