package endpoint

import (
	"context"

	"github.com/Scalingo/sand/api/types"
	"github.com/pkg/errors"
)

var (
	ErrNotActive = errors.New("endpoint is not active")
)

func (r *repository) Deactivate(ctx context.Context, n types.Network, e types.Endpoint) (types.Endpoint, error) {
	if !e.Active {
		return e, ErrNotActive
	}

	err := r.managers.Get(n.Type).DeleteEndpoint(ctx, n, e)
	if err != nil {
		return e, errors.Wrapf(err, "fail to delete endpoint from overlay network")
	}

	e.Active = false
	e.TargetNetnsPath = ""

	err = r.store.Set(ctx, e.StorageKey(), &e)
	if err != nil {
		return e, errors.Wrapf(err, "fail to save endpoint %s in store", e)
	}

	err = r.store.Set(ctx, e.NetworkStorageKey(), &e)
	if err != nil {
		return e, errors.Wrapf(err, "fail to save endpoint %s in store network", e)
	}

	return e, nil
}
