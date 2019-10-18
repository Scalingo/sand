package web

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/Scalingo/go-utils/logger"
	"github.com/Scalingo/sand/api/httpresp"
	"github.com/Scalingo/sand/api/params"
	"github.com/Scalingo/sand/ipallocator"
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
		"network_id": params.NetworkID,
	})

	if params.Activate {
		log = logger.Get(ctx).WithFields(logrus.Fields{
			"activate":     params.Activate,
			"target_netns": params.ActivateParams.NSHandlePath,
		})

		if params.ActivateParams.NSHandlePath == "" {
			w.WriteHeader(400)
			return errors.New("missing ns_handle_path")
		}
		if _, err := os.Stat(params.ActivateParams.NSHandlePath); err != nil {
			w.WriteHeader(400)
			return errors.Errorf("ns_handle_path '%s' is invalid: %v", params.ActivateParams.NSHandlePath, err)
		}
	}

	network, ok, err := c.NetworkRepository.Exists(ctx, params.NetworkID)
	if err != nil {
		return errors.Wrapf(err, "fail to get network %v", params.NetworkID)
	}
	if !ok {
		w.WriteHeader(404)
		return errors.New("not found")
	}

	allocatedIP, err := c.IPAllocator.AllocateIP(ctx, params.NetworkID, ipallocator.AllocateIPOpts{
		Address: params.IPv4Address,
	})
	if err != nil {
		return errors.Wrapf(err, "fail to allocate IP in pool ip=%v network=%v", params.IPv4Address, network)
	}
	params.IPv4Address = allocatedIP

	err = c.NetworkRepository.Ensure(ctx, network)
	if err != nil {
		return errors.Wrapf(err, "fail to ensure network %s", network)
	}

	log = log.WithField("network_name", network.Name)
	log.Info("creating endpoint")
	ctx = logger.ToCtx(ctx, log)

	params.ActivateParams.SetAddr = true
	params.ActivateParams.MoveVeth = true

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
