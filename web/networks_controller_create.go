package web

import (
	"encoding/json"
	"net/http"

	"github.com/Scalingo/go-internal-tools/logger"
	"github.com/Scalingo/sand/api/httpresp"
	"github.com/Scalingo/sand/api/params"

	"github.com/pkg/errors"
)

func (c NetworksController) Create(w http.ResponseWriter, r *http.Request, p map[string]string) error {
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()
	log := logger.Get(ctx)

	var cnp params.NetworkCreate
	err := json.NewDecoder(r.Body).Decode(&cnp)
	if err != nil {
		return errors.Wrap(err, "invalid JSON")
	}

	network, err := c.NetworkRepository.Create(ctx, cnp)
	if err != nil {
		return errors.Wrapf(err, "fail to create network '%v'", cnp.Name)
	}

	w.WriteHeader(201)
	err = json.NewEncoder(w).Encode(&httpresp.NetworkCreate{
		Network: network,
	})
	if err != nil {
		log.WithError(err).Error("fail to encode JSON")
	}
	return nil
}
