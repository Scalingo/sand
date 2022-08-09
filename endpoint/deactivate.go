package endpoint

import (
	"context"

	"github.com/pkg/errors"

	"github.com/Scalingo/go-utils/logger"
	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/network/netmanager"
)

func (r *repository) Deactivate(ctx context.Context, n types.Network, e types.Endpoint) (types.Endpoint, error) {
	log := logger.Get(ctx)
	log.Info("Deactivate endpoint")

	if !e.Active {
		log.Info("Endpoint is not active")
		return e, nil
	}

	err := r.managers.Get(n.Type).DeleteEndpoint(ctx, n, e)
	if err != nil && err != netmanager.EndpointAlreadyDisabledErr {
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

	log.Info("Endpoint deactivated")
	return e, nil
}
