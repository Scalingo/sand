package types

import (
	"fmt"
	"time"
)

const (
	EndpointStoragePrefix        = "/node-endpoints"
	NetworkEndpointStoragePrefix = "/network-endpoints"
)

type Endpoint struct {
	ID              string    `json:"id"`
	NetworkID       string    `json:"network_id"`
	Hostname        string    `json:"hostname"`
	HostIP          string    `json:"host_ip"`
	CreatedAt       time.Time `json:"created_at"`
	TargetNetnsPath string    `json:"target_netns_path"`
	OverlayVethName string    `json:"overlay_veth_name"`
	OverlayVethMAC  string    `json:"overlay_veth_mac"`
	TargetVethName  string    `json:"target_veth_name"`
	TargetVethMAC   string    `json:"target_veth_mac"`
	TargetVethIP    string    `json:"target_veth_ip"`
	Active          bool      `json:"active"`
}

func (e Endpoint) String() string {
	return fmt.Sprintf("Endpoint[%s|%s|Network(%s)|Active(%v)]", e.ID, e.TargetNetnsPath, e.NetworkID, e.Active)
}

func (e Endpoint) StorageKey() string {
	return fmt.Sprintf("%s/%s/%s", EndpointStoragePrefix, e.Hostname, e.ID)
}

func (e Endpoint) NetworkStorageKey() string {
	return fmt.Sprintf("%s/%s/%s", NetworkEndpointStoragePrefix, e.NetworkID, e.ID)
}
