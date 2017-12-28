package overlay

import (
	"context"

	"github.com/Scalingo/sand/config"
	"github.com/Scalingo/sand/idmanager"
	"github.com/Scalingo/sand/store"
)

const (
	IDManagerName         = "vni"
	VxLANField            = "vxlan_vni"
	StoreCollectionPrefix = "/network/"
)

func NewVNIGenerator(ctx context.Context, config *config.Config, store store.Store) idmanager.Manager {
	return idmanager.New(config, store, IDManagerName, VxLANField, StoreCollectionPrefix)
}
