package network

import (
	"context"

	"github.com/Scalingo/go-utils/errors/v3"
	"github.com/Scalingo/sand/api/types"
)

func (r repository) List(ctx context.Context) ([]types.Network, error) {
	var networks []types.Network

	err := r.store.Get(ctx, "/network/", true, &networks)
	if err != nil {
		return nil, errors.Wrapf(ctx, err, "query store")
	}

	return networks, nil
}
