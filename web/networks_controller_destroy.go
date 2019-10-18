package web

import (
	"net/http"

	"github.com/Scalingo/go-utils/logger"
	"github.com/pkg/errors"
)

func (c NetworksController) Destroy(w http.ResponseWriter, r *http.Request, p map[string]string) error {
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()
	log := logger.Get(ctx)

	log = log.WithField("network_id", p["id"])
	ctx = logger.ToCtx(ctx, log)

	n, ok, err := c.NetworkRepository.Exists(ctx, p["id"])
	if err != nil {
		return errors.Wrapf(err, "fail to know if network '%s' exists", p["id"])
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
		return errors.Wrapf(err, "fail to get network %s endpoints", n)
	}

	if len(endpoints) > 0 {
		w.WriteHeader(http.StatusBadRequest)
		return errors.Errorf("fail to delete network %s, %d endpoints are still present.", n, len(endpoints))
	}

	err = c.NetworkRepository.Deactivate(ctx, n)
	if err != nil {
		return errors.Wrapf(err, "fail to deactivate network %s", n)
	}

	err = c.NetworkRepository.Delete(ctx, n, c.IPAllocator)
	if err != nil {
		return errors.Wrapf(err, "fail to delete network %s", n)
	}

	w.WriteHeader(204)
	return nil
}
