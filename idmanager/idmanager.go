package idmanager

import (
	"context"
	stderrors "errors"
	"fmt"
	"io"

	etcdlock "github.com/Scalingo/go-etcd-lock/v5/lock"
	"github.com/Scalingo/go-utils/errors/v3"
	"github.com/Scalingo/sand/etcd"
	"github.com/Scalingo/sand/store"
)

var ErrNoIDAvailable = stderrors.New("no new ID available")

type Manager interface {
	Lock(context.Context) (Unlocker, error)
	Generate(context.Context) (int, error)
}

type Unlocker interface {
	Unlock(ctx context.Context) error
}

type manager struct {
	store      store.Store
	maxIDValue int
	field      string
	name       string
	prefix     string
}

type lock struct {
	resourceLock   etcdlock.Lock
	lockingBackend io.Closer
}

func New(maxIDValue int, s store.Store, name string, field string, prefix string) Manager {
	return &manager{maxIDValue: maxIDValue, store: s, field: field, name: name, prefix: prefix}
}

func (m *manager) Lock(ctx context.Context) (Unlocker, error) {
	client, err := etcd.NewClient(ctx)
	if err != nil {
		return nil, errors.Wrapf(ctx, err, "get etcd client")
	}
	resourceLock, err := etcdlock.NewEtcdLocker(client).WaitAcquire(fmt.Sprintf("/%s-idgen", m.name), 300)
	if err != nil {
		return nil, errors.Wrapf(ctx, err, "get etcd lock")
	}
	return lock{
		resourceLock:   resourceLock,
		lockingBackend: client,
	}, nil
}

func (l lock) Unlock(ctx context.Context) error {
	if l.resourceLock == nil {
		return errors.New(ctx, "not locked")
	}
	lockErr := l.resourceLock.Release()
	backendErr := l.lockingBackend.Close()
	if lockErr != nil {
		return errors.Wrapf(ctx, lockErr, "release etcd lock, backendErr: %v", backendErr)
	}
	if backendErr != nil {
		return errors.Wrapf(ctx, backendErr, "close etcd client")
	}
	return nil
}

func (m *manager) Generate(ctx context.Context) (int, error) {
	var items []map[string]any

	// Retrieving the list of networks as a map of etcd keys to network objects
	err := m.store.Get(ctx, m.prefix, true, &items)
	if err == store.ErrNotFound {
		return 1, nil
	}
	if err != nil {
		return -1, errors.Wrapf(ctx, err, "get list of items with prefix %s from store", m.prefix)
	}

	// Generating a "set" of existing IDs
	ids := map[int]bool{}
	for _, item := range items {
		ids[int(item[m.field].(float64))] = true
	}

	// Searching for the first available ID until the maximum
	for i := 1; i <= m.maxIDValue; i++ {
		if !ids[i] {
			return i, nil
		}
	}
	return -1, ErrNoIDAvailable
}
