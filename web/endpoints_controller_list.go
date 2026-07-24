package web

import (
	"encoding/json"
	"net/http"

	"github.com/Scalingo/go-utils/errors/v3"
	"github.com/Scalingo/go-utils/logger"
	"github.com/Scalingo/sand/api/httpresp"
)

func (c EndpointsController) List(w http.ResponseWriter, r *http.Request, p map[string]string) error {
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()
	log := logger.Get(ctx)

	filter := map[string]string{
		"network_id": r.URL.Query().Get("network_id"),
		"hostname":   r.URL.Query().Get("hostname"),
	}
	endpoints, err := c.EndpointRepository.List(ctx, filter)
	if err != nil {
		return errors.Wrapf(ctx, err, "list endpoints with filters %v", filter)
	}

	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(&httpresp.EndpointsList{
		Endpoints: endpoints,
	})
	if err != nil {
		log.WithError(err).Error("encode JSON")
	}

	return nil
}
