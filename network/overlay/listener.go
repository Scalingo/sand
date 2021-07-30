package overlay

import (
	"bytes"
	"context"
	"encoding/json"
	"sync"

	"go.etcd.io/etcd/api/v3/mvccpb"
	"go.etcd.io/etcd/client/v3"

	"github.com/Scalingo/go-utils/logger"
	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/config"
	"github.com/Scalingo/sand/network/netmanager"
	"github.com/Scalingo/sand/store"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type NetworkEndpointListener interface {
	Add(context.Context, netmanager.NetManager, types.Network) (chan struct{}, error)
	Remove(context.Context, types.Network) error
}

type Registrar interface {
	Register(string) (store.Registration, error)
}

type listener struct {
	sync.Mutex
	config               *config.Config
	store                store.Store
	registrar            Registrar
	networkRegistrations map[string]store.Registration

	// globalContext is the context used to start etcd registrar when it is
	// canceled all resources are released. We can't use the one of Add, as it is
	// often bount to a temporary http request, the context gets canceled
	// directly at the end of the request and the watch breaks
	globalContext context.Context
}

func NewNetworkEndpointListener(ctx context.Context, config *config.Config, r Registrar, s store.Store) NetworkEndpointListener {
	return &listener{config: config, registrar: r, store: s, networkRegistrations: map[string]store.Registration{}, globalContext: ctx}
}

func (l *listener) Remove(ctx context.Context, network types.Network) error {
	l.Lock()
	defer l.Unlock()

	if r, ok := l.networkRegistrations[network.ID]; !ok {
		return nil
	} else {
		r.Unregister()
		delete(l.networkRegistrations, network.ID)
	}
	return nil
}

func (l *listener) Add(ctx context.Context, nm netmanager.NetManager, network types.Network) (chan struct{}, error) {
	l.Lock()
	defer l.Unlock()

	if _, ok := l.networkRegistrations[network.ID]; ok {
		return nil, nil
	}

	log := logger.Default().WithFields(logrus.Fields{
		"network_id":   network.ID,
		"network_name": network.Name,
		"network_type": network.Type,
	})
	listenerCtx := logger.ToCtx(l.globalContext, log)

	log.Info("registering to network modifications")
	r, err := l.registrar.Register(network.EndpointsStorageKey(""))
	if err != nil {
		return nil, errors.Wrapf(err, "fail to create registration for network %s", network)
	}
	l.networkRegistrations[network.ID] = r

	done := make(chan struct{})
	go func(r store.Registration) {
		defer close(done)
		log.Info("start listening registration events")
		for event := range r.EventChan() {
			err := l.handleEvent(listenerCtx, event, nm, network)
			if err != nil {
				log.WithError(err).Error("fail to handle registration response")
			}
		}
		log.Info("stop listening registration events")
	}(r)

	return done, nil
}

func (l *listener) handleEvent(ctx context.Context, event *clientv3.Event, nm netmanager.NetManager, network types.Network) error {
	log := logger.Get(ctx)
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
		log.Info("registration got new endpoint")
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
			log.WithError(err).Error("fail to remove endpoint ARP/FDB neigh rules")
		}
	}
	return nil
}
