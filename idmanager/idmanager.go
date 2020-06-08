package idmanager

import (
	"context"
	"fmt"
	"io"

	etcdlock "github.com/Scalingo/go-etcd-lock/lock"
	"github.com/Scalingo/sand/config"
	"github.com/Scalingo/sand/etcd"
	"github.com/Scalingo/sand/store"
	"github.com/pkg/errors"
)

type Manager interface {
	Lock(context.Context) (Lock, error)
	Generate(context.Context) (int, error)
}

type Lock interface {
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

func (m *manager) Lock(ctx context.Context) (Lock, error) {
	var l Lock

	client, err := etcd.NewClient()
	if err != nil {
		return l, errors.Wrapf(err, "fail to get etcd client")
	}
	resourceLock, err := etcdlock.NewEtcdLocker(client).WaitAcquire(fmt.Sprintf("/%s-idgen", m.name), 300)
	if err != nil {
		return l, errors.Wrapf(err, "fail to get etcd lock")
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
	err := l.resourceLock.Release()
	if err != nil {
		return errors.Wrapf(err, "fail to release etcd lock")
	}
	err = l.lockingBackend.Close()
	if err != nil {
		return errors.Wrapf(err, "fail to close etcd client")
	}
	return nil
}

func (m *manager) Generate(ctx context.Context) (int, error) {
	var items []map[string]interface{}

	err := m.store.Get(ctx, m.prefix, true, &items)
	if err == store.ErrNotFound {
		return 1, nil
	}
	if err != nil {
		return -1, errors.Wrapf(err, "fail to get list of items with prefix %s from store", m.prefix)
	}

	ids := map[int]bool{}
	for _, item := range items {
		ids[int(item[m.field].(float64))] = true
	}

	for i := 1; ; i++ {
		if !ids[i] {
			return i, nil
		}
	}

	// unreachable
	return -1, errors.New("fail to select new ID")
}
