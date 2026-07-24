package web

import (
	"net/http"

	"github.com/Scalingo/go-utils/errors/v3"
	"github.com/Scalingo/sand/node"
)

func (c NetworksController) EnsureNetworkEndpoints(w http.ResponseWriter, r *http.Request, params map[string]string) error {
	w.Header().Set("Content-Type", "application/json")

	err := node.EnsureNetworkEndpoints(r.Context(), c.Config, c.NetworkRepository, c.EndpointRepository)
	if err != nil {
		return errors.Wrap(r.Context(), err, "ensure network endpoints")
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}
