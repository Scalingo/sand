package web

import (
	"net/http"

	"github.com/Scalingo/go-utils/errors/v3"
	"github.com/Scalingo/go-utils/logger"
	"github.com/Scalingo/sand/endpoint"
)

func (c EndpointsController) Destroy(w http.ResponseWriter, r *http.Request, p map[string]string) error {
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()
	log := logger.Get(ctx)

	e, ok, err := c.EndpointRepository.Exists(ctx, p["id"])
	if err != nil {
		return errors.Wrapf(ctx, err, "get endpoint %v", p["id"])
	}
	if !ok {
		w.WriteHeader(404)
		return nil
	}

	log = log.WithField("endpoint_id", e.ID)
	ctx = logger.ToCtx(ctx, log)

	network, ok, err := c.NetworkRepository.Exists(ctx, e.NetworkID)
	if err != nil {
		return errors.Wrapf(ctx, err, "get network %v", e.NetworkID)
	}
	if !ok {
		return errors.Wrapf(ctx, err, "endpoint %v has unreferenced network ID %v", e, e.NetworkID)
	}

	log = log.WithField("network_id", network.ID)
	ctx = logger.ToCtx(ctx, log)

	err = c.EndpointRepository.Delete(ctx, network, e, endpoint.DeleteOpts{
		ForceDeactivation: true,
	})
	if err != nil {
		return errors.Wrapf(ctx, err, "destroy endpoint")
	}

	w.WriteHeader(204)
	return nil
}
