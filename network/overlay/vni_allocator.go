package overlay

import (
	"context"

	"github.com/Scalingo/networking-agent/config"
	"github.com/Scalingo/networking-agent/idmanager"
	"github.com/Scalingo/networking-agent/store"
)

const (
	IDManagerName         = "vni"
	VxLANField            = "vxlan_vni"
	StoreCollectionPrefix = "/network/"
)

func NewVNIGenerator(ctx context.Context, config *config.Config, store store.Store) idmanager.Manager {
	return idmanager.New(config, store, IDManagerName, VxLANField, StoreCollectionPrefix)
}
