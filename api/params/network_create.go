package params

import (
	"github.com/sirupsen/logrus"

	"github.com/Scalingo/sand/api/types"
)

type NetworkCreate struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Type    types.NetworkType `json:"type"`
	IPRange string            `json:"ip_range"`
	Gateway string            `json:"gateway"`
}

func (nc NetworkCreate) LogFields() logrus.Fields {
	logFields := logrus.Fields{}
	if nc.ID != "" {
		logFields["id"] = nc.ID
	}
	if nc.Name != "" {
		logFields["name"] = nc.Name
	}
	if nc.Type != "" {
		logFields["type"] = nc.Type
	}
	if nc.IPRange != "" {
		logFields["ip_range"] = nc.IPRange
	}
	if nc.Gateway != "" {
		logFields["gateway"] = nc.Gateway
	}
	return logFields
}
