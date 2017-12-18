package web

import (
	"encoding/json"
	"net/http"

	"github.com/Scalingo/go-internal-tools/logger"
	"github.com/Scalingo/networking-agent/api/types"
	"github.com/Scalingo/networking-agent/config"
	"github.com/Scalingo/networking-agent/store"
	"github.com/pkg/errors"
)

type NetworksController struct {
	Config *config.Config
	Store  store.Store
}

func NewNetworksController(c *config.Config) NetworksController {
	return NetworksController{Config: c, Store: store.New(c)}
}

type NetworkList struct {
	Networks []types.Network `json:"networks"`
}

func (c NetworksController) List(w http.ResponseWriter, r *http.Request, params map[string]string) error {
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()
	log := logger.Get(ctx)

	var res NetworkList
	err := c.Store.Get(ctx, "/network/", true, &res.Networks)
	if err != nil {
		return errors.Wrapf(err, "fail to query store")
	}

	w.WriteHeader(200)
	err = json.NewEncoder(w).Encode(&res)
	if err != nil {
		log.WithError(err).Error("fail to encode JSON")
	}
	return nil
}
