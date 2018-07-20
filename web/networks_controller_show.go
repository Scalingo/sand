package web

import (
	"encoding/json"
	"net/http"

	"github.com/Scalingo/go-internal-tools/logger"
	"github.com/Scalingo/sand/api/httpresp"
	"github.com/pkg/errors"
)

func (c NetworksController) Show(w http.ResponseWriter, r *http.Request, params map[string]string) error {
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()
	log := logger.Get(ctx)

	network, ok, err := c.NetworkRepository.Exists(ctx, params["id"])
	if err != nil {
		return errors.Wrapf(err, "fail to query store")
	} else if !ok {
		w.WriteHeader(404)
		return errors.New("network not found")
	}
	res := httpresp.NetworkShow{
		Network: network,
	}

	w.WriteHeader(200)
	err = json.NewEncoder(w).Encode(&res)
	if err != nil {
		log.WithError(err).Error("fail to encode JSON")
	}
	return nil
}
