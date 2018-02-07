package endpoint

import (
	"context"

	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/store"
	"github.com/pkg/errors"
)

func (r *repository) Exists(ctx context.Context, id string) (types.Endpoint, bool, error) {
	endpoint := types.Endpoint{
		Hostname: r.config.PublicHostname,
		ID:       id,
	}
	err := r.store.Get(ctx, endpoint.StorageKey(), false, &endpoint)
	if err == store.ErrNotFound {
		return endpoint, false, nil
	}
	if err != nil {
		return endpoint, false, errors.Wrapf(err, "fail to get endpoint %s", id)
	}

	return endpoint, true, nil
}
