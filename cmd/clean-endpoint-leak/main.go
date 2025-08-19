package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Scalingo/go-utils/logger"
	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/config"
	"github.com/Scalingo/sand/endpoint"
	"github.com/Scalingo/sand/integrations/docker"
	"github.com/Scalingo/sand/network"
	"github.com/Scalingo/sand/network/netmanager"
	"github.com/Scalingo/sand/network/overlay"
	"github.com/Scalingo/sand/store"
)

func main() {
	log := logger.Default()
	ctx := logger.ToCtx(context.Background(), log)
	if len(os.Args) != 2 {
		log.Error("Invalid usage")
		os.Exit(1)
	}

	networkID := os.Args[1]

	ctx, log = logger.WithFieldToCtx(ctx, "network", networkID)
	log.Info("Cleaning network")

	c, err := config.Build()
	if err != nil {
		log.WithError(err).Error("fail to generate initial config")
		os.Exit(-1)
	}
	dataStore := store.New(c)

	managers := netmanager.NewManagerMap()
	endpointsWatcher, err := store.NewWatcher(ctx, c, store.WithPrefix(types.NetworkEndpointStoragePrefix))
	if err != nil {
		log.WithError(err).Error("fail to initialize store watcher")
	}
	peerListener := overlay.NewNetworkEndpointListener(ctx, c, endpointsWatcher, dataStore)
	managers.Set(types.OverlayNetworkType, overlay.NewManager(c, peerListener))
	endpointRepository := endpoint.NewRepository(c, dataStore, managers)
	networkRepository := network.NewRepository(c, dataStore, managers)
	dockerRepository := docker.NewRepository(c, dataStore)

	_, found, err := networkRepository.Exists(ctx, networkID)
	if err != nil {
		log.WithError(err).Error("check if network exists")
	}

	if !found {
		log.Error("Network not found")
		os.Exit(1)
	}

	dockerEndpoints, err := dockerRepository.ListEndpoints(ctx)
	if err != nil {
		log.WithError(err).Error("list docker endpoints")
	}

	networkEndpoints, err := endpointRepository.List(ctx, map[string]string{"network_id": networkID})
	if err != nil {
		log.WithError(err).Error("list network endpoints")
	}

	idx := make(map[string]types.Endpoint)

	seen := make(map[string]bool)

	for _, endpoint := range networkEndpoints {
		seen[endpoint.ID] = false
		idx[endpoint.ID] = endpoint
	}

	for _, endpoint := range dockerEndpoints {
		_, found := seen[endpoint.SandEndpointID]
		if !found {
			continue
		}

		seen[endpoint.SandEndpointID] = true
	}

	log.Info("Orphan endpoints are:")

	for endpointID, seen := range seen {
		if !idx[endpointID].Active {
			continue
		}
		if seen {
			continue
		}
		log.Info(endpointID)
	}

	fmt.Println("Continue with fix ? (y/N)")
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}

	response = strings.ToLower(strings.TrimSpace(response))

	if response != "y" {
		os.Exit(0)
	}
	log.Info("Start claning")

	for endpointID, seen := range seen {
		endpoint := idx[endpointID]
		if !endpoint.Active {
			continue
		}
		if seen {
			continue
		}

		ctx, log := logger.WithFieldToCtx(ctx, "endpoint", endpointID)
		log.Info("Changing IP to 192.168.254.254/32 and mac to de:ad:be:ef:ca:fe")
		endpoint.TargetVethIP = "192.168.254.254/32"
		endpoint.TargetVethMAC = "de:ad:be:ef:ca:fe"

		err = dataStore.Set(ctx, endpoint.StorageKey(), &endpoint)
		if err != nil {
			log.WithError(err).Error("Fail to store endpoint")
		}

		err = dataStore.Set(ctx, endpoint.NetworkStorageKey(), &endpoint)
		if err != nil {
			log.WithError(err).Error("Fail to store network endpoint")
		}

		log.Info("Deleting endpoint")

		err = dataStore.Delete(ctx, endpoint.StorageKey())
		if err != nil {
			log.WithError(err).Error("delete endpoint storage")
		}

		err = dataStore.Delete(ctx, endpoint.NetworkStorageKey())
		if err != nil {
			log.WithError(err).Error("delete network endpoint storage")
		}
	}
}
