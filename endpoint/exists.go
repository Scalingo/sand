package endpoint

import (
	"context"
	"fmt"

	"github.com/Scalingo/networking-agent/api/types"
	"github.com/Scalingo/networking-agent/store"
	"github.com/pkg/errors"
)

func (r *repository) Exists(ctx context.Context, n types.Network, nspath string) (types.Endpoint, bool, error) {
	var endpoints []types.Endpoint
	err := r.store.Get(ctx, fmt.Sprintf("%s/%s/%s/", types.EndpointStoragePrefix, r.config.PublicHostname, n.ID), true, &endpoints)
	if err == store.ErrNotFound {
		return types.Endpoint{}, false, nil
	}
	if err != nil {
		return types.Endpoint{}, false, errors.Wrapf(err, "fail to get network %s endpoints", n)
	}

	for _, endpoint := range endpoints {
		if endpoint.TargetNetnsPath == nspath {
			return endpoint, true, nil
		}
	}

	return types.Endpoint{}, false, nil
}
