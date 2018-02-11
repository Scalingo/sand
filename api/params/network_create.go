package params

import (
	"github.com/Scalingo/sand/api/types"
)

type NetworkCreate struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Type    types.NetworkType `json:"type"`
	IPRange string            `json:"ip_range"`
}
