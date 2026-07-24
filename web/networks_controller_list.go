package web

import (
	"encoding/json"
	"net/http"

	"github.com/Scalingo/go-utils/errors/v3"
	"github.com/Scalingo/go-utils/logger"
	"github.com/Scalingo/sand/api/httpresp"
	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/store"
)

func (c NetworksController) List(w http.ResponseWriter, r *http.Request, params map[string]string) error {
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()
	log := logger.Get(ctx)

	networks, err := c.NetworkRepository.List(ctx)
	if errors.Is(err, store.ErrNotFound) {
		networks = []types.Network{}
	} else if err != nil {
		return errors.Wrapf(ctx, err, "query store")
	}
	res := httpresp.NetworksList{
		Networks: networks,
	}

	w.WriteHeader(200)
	err = json.NewEncoder(w).Encode(&res)
	if err != nil {
		log.WithError(err).Error("encode JSON")
	}
	return nil
}
