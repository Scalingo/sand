package web

import (
	"encoding/json"
	"net/http"

	"github.com/Scalingo/go-utils/errors/v3"
	"github.com/Scalingo/go-utils/logger"
	"github.com/Scalingo/sand/api/httpresp"
	"github.com/Scalingo/sand/api/params"
	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/ipallocator"
	"github.com/Scalingo/sand/netutils"
)

func (c NetworksController) Create(w http.ResponseWriter, r *http.Request, p map[string]string) error {
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()
	log := logger.Get(ctx)

	var cnp params.NetworkCreate
	err := json.NewDecoder(r.Body).Decode(&cnp)
	if err != nil {
		return errors.Wrap(ctx, err, "invalid JSON")
	}

	if cnp.IPRange != "" && cnp.Gateway == "" {
		cnp.Gateway, err = netutils.DefaultGateway(cnp.IPRange)
		if err != nil {
			return errors.Wrapf(ctx, err, "get default gateway for iprange=%v", cnp.IPRange)
		}
	} else if cnp.IPRange == "" && cnp.Gateway == "" {
		cnp.IPRange = types.DefaultIPRange
		cnp.Gateway = types.DefaultGateway
	}

	network, err := c.NetworkRepository.Create(ctx, cnp)
	if err != nil {
		return errors.Wrapf(ctx, err, "create network '%v'", cnp.Name)
	}

	_, err = c.IPAllocator.AllocateIP(ctx, network.ID, ipallocator.AllocateIPOpts{
		Address:      cnp.Gateway,
		AddressRange: network.IPRange,
	})
	if err != nil {
		return errors.Wrapf(ctx, err, "initialize IP pool for network '%v'", network.ID)
	}

	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(&httpresp.NetworkCreate{
		Network: network,
	})
	if err != nil {
		log.WithError(err).Error("Failed to encode JSON")
	}
	return nil
}
