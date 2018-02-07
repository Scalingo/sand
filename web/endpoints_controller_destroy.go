package web

import (
	"net/http"

	"github.com/Scalingo/go-internal-tools/logger"
	"github.com/pkg/errors"
)

func (c EndpointsController) Destroy(w http.ResponseWriter, r *http.Request, p map[string]string) error {
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()
	log := logger.Get(ctx)

	endpoint, ok, err := c.EndpointRepository.Exists(ctx, p["id"])
	if err != nil {
		return errors.Wrapf(err, "fail to get endpoint %v", p["id"])
	}
	if !ok {
		w.WriteHeader(404)
		return nil
	}

	log = log.WithField("endpoint_id", endpoint.ID)
	ctx = logger.ToCtx(ctx, log)

	network, ok, err := c.NetworkRepository.Exists(ctx, endpoint.NetworkID)
	if err != nil {
		return errors.Wrapf(err, "fail to get network %v", endpoint.NetworkID)
	}
	if !ok {
		return errors.Wrapf(err, "endpoint %v has unreferenced network ID %v", endpoint, endpoint.NetworkID)
	}

	log = log.WithField("network_id", network.ID)
	ctx = logger.ToCtx(ctx, log)

	err = c.NetworkRepository.DeleteEndpoint(ctx, network, endpoint)
	if err != nil {
		return errors.Wrapf(err, "fail to delete endpoint %v of network %v", network, endpoint)
	}

	err = c.EndpointRepository.Delete(ctx, p["id"])
	if err != nil {
		return errors.Wrapf(err, "fail to destroy endpoint")
	}

	w.WriteHeader(204)
	return nil
}
