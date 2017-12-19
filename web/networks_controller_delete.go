package web

import (
	"net/http"

	"github.com/Scalingo/go-internal-tools/logger"
	"github.com/Scalingo/networking-agent/network"
	"github.com/pkg/errors"
)

func (c NetworksController) Destroy(w http.ResponseWriter, r *http.Request, p map[string]string) error {
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()
	log := logger.Get(ctx)

	log = log.WithField("network_id", p["id"])
	ctx = logger.ToCtx(ctx, log)

	repo := network.NewRepository(c.Config, c.Store)

	n, ok, err := repo.Exists(ctx, p["id"])
	if err != nil {
		return errors.Wrapf(err, "fail to know if network '%s' exists", p["id"])
	}
	if !ok {
		w.WriteHeader(404)
		return nil
	}

	log = log.WithField("network_name", n.Name)
	ctx = logger.ToCtx(ctx, log)

	err = repo.Delete(ctx, n)
	if err != nil {
		return errors.Wrapf(err, "fail to delete network %s", n)
	}

	w.WriteHeader(204)
	return nil
}
