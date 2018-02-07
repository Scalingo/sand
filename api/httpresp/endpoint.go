package httpresp

import (
	"github.com/Scalingo/sand/api/types"
)

type EndpointCreate struct {
	Endpoint types.Endpoint `json:"endpoint"`
}

type EndpointsList struct {
	Endpoints []types.Endpoint `json:"endpoints"`
}
