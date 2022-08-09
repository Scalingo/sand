package endpoint

import (
	"context"

	"github.com/pkg/errors"

	"github.com/Scalingo/go-utils/logger"
	"github.com/Scalingo/sand/api/types"
)

var (
	ErrActivated = errors.New("endpoint is still active")
)

type DeleteOpts struct {
	ForceDeactivation bool
}

func (r *repository) Delete(ctx context.Context, n types.Network, e types.Endpoint, opts DeleteOpts) error {
	log := logger.Get(ctx)
	log.Info("Delete endpoint")

	var err error

	if opts.ForceDeactivation {
		e, err = r.Deactivate(ctx, n, e)
		if err != nil {
			return errors.Wrapf(err, "fail to deactivate endpoint")
		}
	}

	if e.Active {
		return ErrActivated
	}

	err = r.store.Delete(ctx, e.StorageKey())
	if err != nil {
		return errors.Wrapf(err, "fail to delete endpoint storage key")
	}

	err = r.store.Delete(ctx, e.NetworkStorageKey())
	if err != nil {
		return errors.Wrapf(err, "fail to delete endpoint storage key")
	}

	log.Info("Endpoint deleted")
	return nil
}
