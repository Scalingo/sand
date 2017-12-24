package web

import (
	"encoding/json"
	"net/http"

	"github.com/Scalingo/go-internal-tools/logger"
	"github.com/Scalingo/networking-agent/api/params"
	"github.com/Scalingo/networking-agent/api/types"
	"github.com/Scalingo/networking-agent/network"
	"github.com/pkg/errors"
)

type NetworkCreateRes struct {
	Network types.Network `json:"network"`
}

func (c NetworksController) Create(w http.ResponseWriter, r *http.Request, p map[string]string) error {
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()
	log := logger.Get(ctx)

	var cnp params.CreateNetworkParams
	err := json.NewDecoder(r.Body).Decode(&cnp)
	if err != nil {
		return errors.Wrap(err, "invalid JSON")
	}

	netRepo := network.NewRepository(c.Config, c.Store, c.Listener)
	network, err := netRepo.Create(ctx, cnp)
	if err != nil {
		return errors.Wrapf(err, "fail to create network '%v'", cnp.Name)
	}

	w.WriteHeader(201)
	err = json.NewEncoder(w).Encode(&NetworkCreateRes{
		Network: network,
	})
	if err != nil {
		log.WithError(err).Error("fail to encode JSON")
	}
	return nil
}
