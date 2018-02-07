package web

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/Scalingo/go-internal-tools/logger"
	"github.com/Scalingo/sand/api/httpresp"
	"github.com/Scalingo/sand/api/params"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (c EndpointsController) Create(w http.ResponseWriter, r *http.Request, p map[string]string) error {
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()

	var params params.EndpointCreate
	err := json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		w.WriteHeader(400)
		return errors.Wrap(err, "invalid JSON")
	}

	log := logger.Get(ctx).WithFields(logrus.Fields{
		"target_netns": params.NSHandlePath,
		"network_id":   params.NetworkID,
	})

	if params.NSHandlePath == "" {
		w.WriteHeader(400)
		return errors.New("missing ns_handle_path")
	}
	if _, err := os.Stat(params.NSHandlePath); err != nil {
		w.WriteHeader(400)
		return errors.Errorf("ns_handle_path '%s' is invalid: %v", params.NSHandlePath, err)
	}

	network, ok, err := c.NetworkRepository.Exists(ctx, params.NetworkID)
	if err != nil {
		return errors.Wrapf(err, "fail to get network %v", params.NetworkID)
	}
	if !ok {
		w.WriteHeader(404)
		return errors.New("not found")
	}

	err = c.NetworkRepository.Ensure(ctx, network)
	if err != nil {
		return errors.Wrapf(err, "fail to ensure network %s", network)
	}

	log = log.WithField("network_name", network.Name)
	log.Info("creating endpoint")
	ctx = logger.ToCtx(ctx, log)

	endpoint, err := c.EndpointRepository.Create(ctx, network, params)
	if err != nil {
		return errors.Wrapf(err, "fail to create endpoint")
	}

	log.Info("endpoint created")
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(&httpresp.EndpointCreate{
		Endpoint: endpoint,
	})
	if err != nil {
		log.WithError(err).Error("fail to encode JSON")
	}

	return nil
}
