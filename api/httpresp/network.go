package httpresp

import (
	"github.com/Scalingo/sand/api/types"
)

type NetworkCreate struct {
	Network types.Network `json:"network"`
}

type NetworkShow struct {
	Network types.Network `json:"network"`
}

type NetworksList struct {
	Networks []types.Network `json:"networks"`
}
