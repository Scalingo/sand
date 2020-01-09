package idmanager

import (
	"context"
	"fmt"

	"github.com/Scalingo/go-etcd-lock/lock"
	"github.com/Scalingo/sand/config"
	"github.com/Scalingo/sand/etcd"
	"github.com/Scalingo/sand/store"
	"github.com/pkg/errors"
)

type Manager interface {
	Lock(context.Context) error
	Generate(context.Context) (int, error)
	Unlock(context.Context) error
}

type manager struct {
	store  store.Store
	config *config.Config
	lock   lock.Lock
	field  string
	name   string
	prefix string
}

func New(c *config.Config, s store.Store, name string, field string, prefix string) Manager {
	return &manager{config: c, store: s, field: field, name: name, prefix: prefix}
}

func (m *manager) Lock(ctx context.Context) error {
	client, err := etcd.NewClient()
	if err != nil {
		return errors.Wrapf(err, "fail to get etcd client")
	}
	l, err := lock.NewEtcdLocker(client).WaitAcquire(fmt.Sprintf("/%s-idgen", m.name), 300)
	if err != nil {
		return errors.Wrapf(err, "fail to get etcd lock")
	}
	m.lock = l
	return nil
}

func (m *manager) Unlock(ctx context.Context) error {
	if m.lock == nil {
		return errors.New("not locked")
	}
	return m.lock.Release()
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
