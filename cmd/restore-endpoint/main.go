package main

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Scalingo/go-utils/logger"
	"github.com/Scalingo/go-utils/logger/plugins/rollbarplugin"
	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/config"
	"github.com/Scalingo/sand/endpoint"
	"github.com/Scalingo/sand/network/netmanager"
	"github.com/Scalingo/sand/network/overlay"
	"github.com/Scalingo/sand/store"
)

type EndpointRecovery struct {
	ID               string `json:"id"`
	NetworkID        string `json:"network_id"`
	Hostname         string `json:"hostname"`
	HostIP           string `json:"host_ip"`
	TargetNetnsPath  string `json:"target_netns_path"`
	OverlayVethName  string `json:"overlay_veth_name"`
	OverlayVethMAC   string `json:"overlay_veth_mac"`
	TargetVethName   string `json:"target_veth_name"`
	TargetVethMAC    string `json:"target_veth_mac"`
	TargetVethIP     string `json:"target_veth_ip"`
	Active           bool   `json:"active"`
	DockerNetworkID  string `json:"docker_network_id"`
	DockerEndpointID string `json:"docker_endpoint_id"`
}

func main() {
	rollbarplugin.Register()
	log := logrus.FieldLogger(logger.Default())
	ctx := logger.ToCtx(context.Background(), log)

	c, err := config.Build()
	if err != nil {
		log.WithError(err).Error("fail to generate initial config")
		os.Exit(-1)
	}

	dataStore := store.New(c)
	endpointsWatcher, err := store.NewWatcher(ctx, c, store.WithPrefix(types.NetworkEndpointStoragePrefix))
	if err != nil {
		log.WithError(err).Error("fail to initialize store watcher")
	}
	peerListener := overlay.NewNetworkEndpointListener(ctx, c, endpointsWatcher, dataStore)
	managers := netmanager.NewManagerMap()
	managers.Set(types.OverlayNetworkType, overlay.NewManager(c, peerListener))

	endpointRepository := endpoint.NewRepository(c, dataStore, managers)

	recoveryData := []EndpointRecovery{}
	out, err := os.ReadFile(os.Args[1])
	if err != nil {
		log.WithError(err).Error("fail to read file")
		return
	}
	err = json.Unmarshal(out, &recoveryData)
	if err != nil {
		log.WithError(err).Error("fail to unmarshal json")
	}

	for _, endpoint := range recoveryData {
		log.Info("restoring endpoint: ", endpoint)
		sandEndpoint := types.Endpoint{
			ID:              endpoint.ID,
			NetworkID:       endpoint.NetworkID,
			Hostname:        endpoint.Hostname,
			HostIP:          endpoint.HostIP,
			CreatedAt:       time.Now(),
			TargetNetnsPath: endpoint.TargetNetnsPath,
			OverlayVethName: endpoint.OverlayVethName,
			OverlayVethMAC:  endpoint.OverlayVethMAC,
			TargetVethName:  endpoint.TargetVethName,
			TargetVethMAC:   endpoint.TargetVethMAC,
			TargetVethIP:    endpoint.TargetVethIP,
			Active:          true,
		}

		err = endpointRepository.Save(ctx, sandEndpoint)
		if err != nil {
			log.WithError(err).Error("fail to save endpoint")
		}
	}
	log.Info("done")
}
