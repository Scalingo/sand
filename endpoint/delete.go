package endpoint

import (
	"context"

	"github.com/Scalingo/sand/api/types"
	"github.com/pkg/errors"
)

func (r *repository) Delete(ctx context.Context, id string) error {
	endpoint := types.Endpoint{
		Hostname: r.config.PublicHostname,
		ID:       id,
	}
	err := r.store.Get(ctx, endpoint.StorageKey(), false, &endpoint)
	if err != nil {
		return errors.Wrapf(err, "fail to get network endpoint")
	}

	err = r.store.Delete(ctx, endpoint.StorageKey())
	if err != nil {
		return errors.Wrapf(err, "fail to delete endpoint storage key")
	}

	err = r.store.Delete(ctx, endpoint.NetworkStorageKey())
	if err != nil {
		return errors.Wrapf(err, "fail to delete endpoint storage key")
	}

	return nil
}
