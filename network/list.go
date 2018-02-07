package network

import (
	"context"

	"github.com/Scalingo/sand/api/types"
	"github.com/pkg/errors"
)

func (r repository) List(ctx context.Context) ([]types.Network, error) {
	var networks []types.Network

	err := r.store.Get(ctx, "/network/", true, &networks)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to query store")
	}

	return networks, nil
}
