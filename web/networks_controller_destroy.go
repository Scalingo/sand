package web

import (
	"net/http"

	"github.com/Scalingo/go-utils/errors/v3"
	"github.com/Scalingo/go-utils/logger"
)

func (c NetworksController) Destroy(w http.ResponseWriter, r *http.Request, p map[string]string) error {
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()
	log := logger.Get(ctx)

	log = log.WithField("network_id", p["id"])
	ctx = logger.ToCtx(ctx, log)

	n, ok, err := c.NetworkRepository.Exists(ctx, p["id"])
	if err != nil {
		return errors.Wrapf(ctx, err, "know if network '%s' exists", p["id"])
	}
	if !ok {
		w.WriteHeader(404)
		return nil
	}

	log = log.WithField("network_name", n.Name)
	ctx = logger.ToCtx(ctx, log)

	endpoints, err := c.EndpointRepository.List(ctx, map[string]string{
		"network_id": n.ID,
	})
	if err != nil {
		return errors.Wrapf(ctx, err, "get network %s endpoints", n)
	}

	if len(endpoints) > 0 {
		w.WriteHeader(http.StatusBadRequest)
		return errors.Errorf(ctx, "delete network %s, %d endpoints are still present.", n, len(endpoints))
	}

	err = c.NetworkRepository.Deactivate(ctx, n)
	if err != nil {
		return errors.Wrapf(ctx, err, "deactivate network %s", n)
	}

	err = c.NetworkRepository.Delete(ctx, n, c.IPAllocator)
	if err != nil {
		return errors.Wrapf(ctx, err, "delete network %s", n)
	}

	w.WriteHeader(204)
	return nil
}
