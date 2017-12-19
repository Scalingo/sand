package types

import (
	"fmt"
	"time"
)

type NetworkType string

const (
	OverlayNetworkType NetworkType = "overlay"
)

type Network struct {
	ID           string      `json:"id"`
	CreatedAt    time.Time   `json:"created_at"`
	Name         string      `json:"name"`
	Type         NetworkType `json:"type"`
	NSHandlePath string      `json:"ns_handle_path"`
	VxLANVNI     int         `json:"vxlan_vni"`
}

func (n Network) StorageKey() string {
	return fmt.Sprintf("/network/%s", n.ID)
}

func (n Network) String() string {
	return fmt.Sprintf("Network<%s|%s>", n.ID, n.Name)
}
