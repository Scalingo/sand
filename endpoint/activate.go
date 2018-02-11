package endpoint

import (
	"context"

	"github.com/Scalingo/sand/api/params"
	"github.com/Scalingo/sand/api/types"
	"github.com/pkg/errors"
)

func (r *repository) Activate(ctx context.Context, n types.Network, endpoint types.Endpoint, params params.EndpointActivate) (types.Endpoint, error) {
	var err error

	if params.NSHandlePath == "" {
		return endpoint, errors.New("ns handle path can't be empty")
	}
	endpoint.TargetNetnsPath = params.NSHandlePath

	m := r.managers.Get(n.Type)
	endpoint, err = m.EnsureEndpoint(ctx, n, endpoint, params)
	if err != nil {
		return endpoint, errors.Wrapf(err, "fail to ensure '%v' network endpoint", n.Type)
	}

	endpoint.Active = true

	err = r.store.Set(ctx, endpoint.StorageKey(), &endpoint)
	if err != nil {
		return endpoint, errors.Wrapf(err, "fail to save endpoint %s in store", endpoint)
	}

	err = r.store.Set(ctx, endpoint.NetworkStorageKey(), &endpoint)
	if err != nil {
		return endpoint, errors.Wrapf(err, "fail to save endpoint %s in store network", endpoint)
	}

	return endpoint, nil
}
