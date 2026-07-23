package web

import (
	"net/http"

	"github.com/pkg/errors"

	"github.com/Scalingo/sand/node"
)

func (c NetworksController) EnsureNetworkEndpoints(w http.ResponseWriter, r *http.Request, params map[string]string) error {
	w.Header().Set("Content-Type", "application/json")

	err := node.EnsureNetworkEndpoints(r.Context(), c.Config, c.NetworkRepository, c.EndpointRepository, c.MetricsExporter)
	if err != nil {
		return errors.Wrap(err, "ensure network endpoints")
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}
