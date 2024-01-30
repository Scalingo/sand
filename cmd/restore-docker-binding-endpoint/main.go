package main

import (
	"context"
	"encoding/json"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/Scalingo/go-utils/logger"
	"github.com/Scalingo/go-utils/logger/plugins/rollbarplugin"
	"github.com/Scalingo/sand/config"
	"github.com/Scalingo/sand/integrations/docker"
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
		endpoint := docker.DockerPluginEndpoint{
			DockerPluginNetwork: docker.DockerPluginNetwork{
				DockerNetworkID: endpoint.DockerNetworkID,
				SandNetworkID:   endpoint.NetworkID,
			},
			DockerEndpointID: endpoint.DockerEndpointID,
			SandEndpointID:   endpoint.ID,
		}

		dockerRepository := docker.NewRepository(c, dataStore)
		err = dockerRepository.SaveEndpoint(ctx, endpoint)
		if err != nil {
			log.WithError(err).Error("fail to save endpoint")
		}
	}
	log.Info("done")
}
