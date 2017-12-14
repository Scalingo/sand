package params

import (
	"github.com/Scalingo/networking-agent/api/types"
)

type CreateNetworkParams struct {
	Name string            `json:"name"`
	Type types.NetworkType `json:"type"`
}
