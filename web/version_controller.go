package web

import (
	"encoding/json"
	"net/http"

	"github.com/Scalingo/go-utils/logger"
	"github.com/Scalingo/sand/config"
)

type VersionController struct {
	Config *config.Config
}

func NewVersionController(c *config.Config) VersionController {
	return VersionController{Config: c}
}

func (c VersionController) Show(w http.ResponseWriter, r *http.Request, params map[string]string) error {
	log := logger.Get(r.Context())
	w.WriteHeader(200)
	err := json.NewEncoder(w).Encode(map[string]string{"version": c.Config.Version})
	if err != nil {
		log.WithError(err).Error("fail to encode JSON")
	}
	return nil
}
