package web

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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
	log := logger.Get(r.Context())

	f, err := os.Open(c.Config.NetnsPath)
	if err != nil {
		return errors.Wrapf(err, "fail to open netns directory")
	}

	filenames, err := f.Readdirnames(-1)
	if err != nil {
		return errors.Wrapf(err, "fail to list netns handlers in %v", c.Config.NetnsPath)
	}

	log.Debugf("%v namespace handlers are present")
	res := NetworkList{Networks: []types.Network{}}
	for _, n := range filenames {
		ns := filepath.Base(n)
		if strings.HasPrefix(ns, c.Config.NetnsPrefix) {
			name := strings.TrimPrefix(ns, c.Config.NetnsPrefix)
			res.Networks = append(res.Networks, types.Network{NSHandlePath: n, Name: name, Type: types.OverlayNetworkType})
		}
	}

	w.WriteHeader(200)
	err = json.NewEncoder(w).Encode(&res)
	if err != nil {
		log.WithError(err).Error("fail to encode JSON")
	}
	return nil
}
