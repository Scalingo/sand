package idmanager

import (
	"context"
	"fmt"
	"io"

	"github.com/pkg/errors"

	etcdlock "github.com/Scalingo/go-etcd-lock/v5/lock"
	"github.com/Scalingo/sand/config"
	"github.com/Scalingo/sand/etcd"
	"github.com/Scalingo/sand/store"
)

var ErrNoIDAvailable = errors.New("no new ID available")

type Manager interface {
	Lock(context.Context) (Unlocker, error)
	Generate(context.Context) (int, error)
}

type Unlocker interface {
	Unlock(ctx context.Context) error
}

type manager struct {
	store  store.Store
	config *config.Config
	field  string
	name   string
	prefix string
}

type lock struct {
	resourceLock   etcdlock.Lock
	lockingBackend io.Closer
}

func New(c *config.Config, s store.Store, name string, field string, prefix string) Manager {
	return &manager{config: c, store: s, field: field, name: name, prefix: prefix}
}

func (m *manager) Lock(ctx context.Context) (Unlocker, error) {
	client, err := etcd.NewClient()
	if err != nil {
		return nil, errors.Wrapf(err, "fail to get etcd client")
	}
	resourceLock, err := etcdlock.NewEtcdLocker(client).WaitAcquire(fmt.Sprintf("/%s-idgen", m.name), 300)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to get etcd lock")
	}
	return lock{
		resourceLock:   resourceLock,
		lockingBackend: client,
	}, nil
}

func (l lock) Unlock(ctx context.Context) error {
	if l.resourceLock == nil {
		return errors.New("not locked")
	}
	lockErr := l.resourceLock.Release()
	backendErr := l.lockingBackend.Close()
	if lockErr != nil {
		return errors.Wrapf(lockErr, "fail to release etcd lock, backendErr: %v", backendErr)
	}
	if backendErr != nil {
		return errors.Wrapf(backendErr, "fail to close etcd client")
	}
	return nil
}

func (m *manager) Generate(ctx context.Context) (int, error) {
	var items []map[string]interface{}

	// Retrieving the list of networks as a map of etcd keys to network objects
	err := m.store.Get(ctx, m.prefix, true, &items)
	if err == store.ErrNotFound {
		return 1, nil
	}
	if err != nil {
		return -1, errors.Wrapf(err, "fail to get list of items with prefix %s from store", m.prefix)
	}

	// Generating a "set" of existing IDs
	ids := map[int]bool{}
	for _, item := range items {
		ids[int(item[m.field].(float64))] = true
	}

	// Searching for the first available ID until the maximum
	for i := 1; i <= m.config.MaxVNI; i++ {
		if !ids[i] {
			return i, nil
		}
	}
	return -1, ErrNoIDAvailable
}
