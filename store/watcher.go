package store

import (
	"context"
	"strings"
	"sync"

	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/etcdserver/api/v3rpc/rpctypes"
	"google.golang.org/grpc/codes"

	"github.com/Scalingo/go-utils/logger"
	"github.com/Scalingo/sand/config"
	"github.com/Scalingo/sand/etcd"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type EtcdWatcher interface {
	WatchChan() clientv3.WatchChan
	Close() error
}

type Registration interface {
	EventChan() <-chan *clientv3.Event
	Unregister()
}

type registration struct {
	eventChan chan *clientv3.Event
	Watcher   Watcher
	key       string
}

func (r registration) EventChan() <-chan *clientv3.Event {
	return r.eventChan
}

func (r registration) Unregister() {
	r.Watcher.unregister(r.key)
}

type Watcher struct {
	config         *config.Config
	prefix         string
	etcdWatcher    EtcdWatcher
	registrations  map[string]registration
	registrationsM *sync.RWMutex
}

type WatcherOpt func(w *Watcher)

func WithPrefix(prefix string) WatcherOpt {
	return func(w *Watcher) {
		prefix = prefixedKey(w.config, prefix)
		w.prefix = prefix
	}
}

func WithEtcdWatcher(etcdWatcher EtcdWatcher) WatcherOpt {
	return func(w *Watcher) {
		w.etcdWatcher = etcdWatcher
	}
}

func NewWatcher(ctx context.Context, config *config.Config, opts ...WatcherOpt) (Watcher, error) {
	log := logger.Get(ctx)
	w := Watcher{
		config:         config,
		registrations:  make(map[string]registration),
		registrationsM: &sync.RWMutex{},
	}

	for _, opt := range opts {
		opt(&w)
	}

	if w.prefix == "" {
		w.prefix = prefixedKey(w.config, "/")
	}

	log.Infof("create endpoints Watcher on prefix %v", w.prefix)

	if w.etcdWatcher == nil {
		etcdWatcher, err := etcd.NewWatcher(w.prefix)
		if err != nil {
			return Watcher{}, errors.Wrapf(err, "fail to create etcd Watcher on %v", w.prefix)
		}
		w.etcdWatcher = etcdWatcher
	}

	go func() {
		w.watchModifications(ctx)
	}()

	return w, nil
}

func (w Watcher) watchModifications(ctx context.Context) {
	log := logger.Get(ctx)
	for res := range w.etcdWatcher.WatchChan() {
		if err := res.Err(); err != nil {
			// If the connection is canceled because grpc (HTTP/2) connection is
			// closed as etcd restart We don't want to throw an error but just keep
			// looping as the client will automatically reconnect.
			if etcderr, ok := err.(rpctypes.EtcdError); ok && etcderr.Code() == codes.Canceled {
				log.WithError(err).Info("watch response canceled, retry")
			} else if err != nil {
				log.WithError(err).Error("fail to handle Watcher response")
			}
		}
		log.WithField("events_count", len(res.Events)).Debug("received events from etcd")
		for _, event := range res.Events {
			log.WithFields(logrus.Fields{
				"event_key": string(event.Kv.Key), "event_type": event.Type,
			}).Info("received event from etcd")

			w.registrationsM.RLock()
			for key, registration := range w.registrations {
				if strings.HasPrefix(string(event.Kv.Key), key) {
					registration.eventChan <- event
				}
			}
			w.registrationsM.RUnlock()
		}
	}
}

func (w Watcher) Register(key string) (Registration, error) {
	w.registrationsM.Lock()
	defer w.registrationsM.Unlock()

	key = prefixedKey(w.config, key)

	if _, ok := w.registrations[key]; ok {
		return registration{}, errors.Errorf("etcd Watcher registration already exists: %v", key)
	}

	c := make(chan *clientv3.Event, 10)
	r := registration{
		key:       key,
		eventChan: c,
		Watcher:   w,
	}
	w.registrations[key] = r

	return r, nil
}

func (w Watcher) unregister(key string) {
	w.registrationsM.Lock()
	defer w.registrationsM.Unlock()

	r, ok := w.registrations[key]
	if !ok {
		return
	}
	close(r.eventChan)
	delete(w.registrations, key)
}

func (w Watcher) Close() error {
	w.registrationsM.Lock()
	for key, r := range w.registrations {
		close(r.eventChan)
		delete(w.registrations, key)
	}
	w.registrationsM.Unlock()

	return w.etcdWatcher.Close()
}
