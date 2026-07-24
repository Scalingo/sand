package web

import (
	"encoding/json"
	"net/http"

	"github.com/Scalingo/go-utils/errors/v3"
	"github.com/Scalingo/go-utils/logger"
	"github.com/Scalingo/sand/api/httpresp"
)

func (c NetworksController) Show(w http.ResponseWriter, r *http.Request, params map[string]string) error {
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()
	log := logger.Get(ctx)

	network, ok, err := c.NetworkRepository.Exists(ctx, params["id"])
	if err != nil {
		return errors.Wrapf(ctx, err, "query store")
	} else if !ok {
		w.WriteHeader(404)
		return errors.New(ctx, "network not found")
	}
	res := httpresp.NetworkShow{
		Network: network,
	}

	w.WriteHeader(200)
	err = json.NewEncoder(w).Encode(&res)
	if err != nil {
		log.WithError(err).Error("Failed to encode JSON")
	}
	return nil
}
