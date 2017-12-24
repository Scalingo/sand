package web

import (
	"encoding/json"
	"net/http"

	"gopkg.in/errgo.v1"

	"github.com/Scalingo/go-internal-tools/logger"
	"github.com/Scalingo/networking-agent/api/params"
	"github.com/Scalingo/networking-agent/endpoint"
	"github.com/Scalingo/networking-agent/network"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (c EndpointsController) Create(w http.ResponseWriter, r *http.Request, p map[string]string) error {
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()

	var params params.CreateEndpointParams
	err := json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		w.WriteHeader(404)
		return errors.Wrap(err, "invalid JSON")
	}

	log := logger.Get(ctx).WithFields(logrus.Fields{
		"target_netns": params.NSHandlePath,
		"network_id":   params.NetworkID,
	})

	repo := network.NewRepository(c.Config, c.Store, c.Listener)
	network, ok, err := repo.Exists(ctx, params.NetworkID)
	if err != nil {
		return errors.Wrapf(err, "fail to get network %v", params.NetworkID)
	}
	if !ok {
		w.WriteHeader(404)
		return errors.New("not found")
	}

	err = repo.Ensure(ctx, network)
	if err != nil {
		return errors.Wrapf(err, "fail to ensure network %s", network)
	}

	log = log.WithField("network_name", network.Name)
	log.Info("creating endpoint")
	ctx = logger.ToCtx(ctx, log)

	erepo := endpoint.NewRepository(c.Config, c.Store)
	endpoint, err := erepo.Create(ctx, network, params)
	if err != nil {
		return errgo.Notef(err, "fail to create endpoint")
	}

	log.Info("endpoint created")
	err = json.NewEncoder(w).Encode(&endpoint)
	if err != nil {
		log.WithError(err).Error("fail to encode JSON")
	}

	return nil
}
