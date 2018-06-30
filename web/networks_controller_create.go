package web

import (
	"encoding/json"
	"net/http"

	"github.com/Scalingo/go-internal-tools/logger"
	"github.com/Scalingo/sand/api/httpresp"
	"github.com/Scalingo/sand/api/params"
	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/ipallocator"
	"github.com/Scalingo/sand/netutils"

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

	if cnp.IPRange == "" || cnp.Gateway == "" {
		cnp.IPRange = types.DefaultIPRange
		cnp.Gateway = types.DefaultGateway
	}
	if cnp.IPRange != "" && cnp.Gateway == "" {
		cnp.Gateway, err = netutils.DefaultGateway(cnp.IPRange)
		if err != nil {
			return errors.Wrapf(err, "fail to get default gateway for iprange=%v", cnp.IPRange)
		}
	}

	network, err := c.NetworkRepository.Create(ctx, cnp)
	if err != nil {
		return errors.Wrapf(err, "fail to create network '%v'", cnp.Name)
	}

	_, err = c.IPAllocator.AllocateIP(ctx, network.ID, ipallocator.AllocateIPOpts{
		Address:      cnp.Gateway,
		AddressRange: network.IPRange,
	})
	if err != nil {
		return errors.Wrapf(err, "fail to initialize IP pool for network '%v'", network.ID)
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
