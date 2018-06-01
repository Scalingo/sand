package types

import (
	"fmt"
	"time"
)

type NetworkType string

const (
	OverlayNetworkType NetworkType = "overlay"
	DefaultIPRange                 = "10.0.0.0/24"
	DefaultGateway                 = "10.0.0.1/24"
)

type Network struct {
	ID           string      `json:"id"`
	CreatedAt    time.Time   `json:"created_at"`
	Name         string      `json:"name"`
	Type         NetworkType `json:"type"`
	NSHandlePath string      `json:"ns_handle_path"`
	VxLANVNI     int         `json:"vxlan_vni"`
	IPRange      string      `json:"ip_range"`
	Gateway      string      `json:"gateway"`
}

func (n Network) StorageKey() string {
	return fmt.Sprintf("/network/%s", n.ID)
}

func (n Network) EndpointsStorageKey(hostname string) string {
	if len(hostname) == 0 {
		return fmt.Sprintf("%s/%s", NetworkEndpointStoragePrefix, n.ID)
	}
	return fmt.Sprintf("%s/%s/%s", EndpointStoragePrefix, hostname, n.ID)
}

func (n Network) String() string {
	return fmt.Sprintf("Network[%s|%s]", n.ID, n.Name)
}
