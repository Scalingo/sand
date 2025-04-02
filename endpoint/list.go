package endpoint

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/store"
)

func (r *repository) List(ctx context.Context, filters map[string]string) ([]types.Endpoint, error) {
	var endpoints []types.Endpoint
	var key string

	networkID := filters["network_id"]
	hostname := filters["hostname"]

	if networkID == "" {
		// if hostname empty -> all the endpoints
		key = fmt.Sprintf("%s/%s/", types.EndpointStoragePrefix, hostname)
	} else if networkID != "" && hostname == "" {
		key = fmt.Sprintf("%s/%s/", types.NetworkEndpointStoragePrefix, networkID)
	} else {
		key = fmt.Sprintf("%s/%s/%s/", types.EndpointStoragePrefix, r.config.GetPeerHostname(), networkID)
	}

	err := r.store.Get(ctx, key, true, &endpoints)
	if err == store.ErrNotFound {
		return []types.Endpoint{}, nil
	}
	if err != nil {
		return nil, errors.Wrapf(err, "fail to get endpoints")
	}

	return endpoints, nil
}
