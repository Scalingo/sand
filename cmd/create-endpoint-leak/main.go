package main

import (
	"context"
	"os"

	"github.com/gofrs/uuid/v5"

	"github.com/Scalingo/go-utils/logger"
	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/config"
	"github.com/Scalingo/sand/endpoint"
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

	_, found, err := networkRepository.Exists(ctx, networkID)
	if err != nil {
		log.WithError(err).Error("check if network exists")
	}

	if !found {
		log.Error("Network not found")
		os.Exit(1)
	}

	networkEndpoints, err := endpointRepository.List(ctx, map[string]string{"network_id": networkID})
	if err != nil {
		log.WithError(err).Error("list network endpoints")
	}

	for _, endpoint := range networkEndpoints {
		oldID := endpoint.ID
		endpoint.ID = uuid.Must(uuid.NewV4()).String()

		err = dataStore.Set(ctx, endpoint.StorageKey(), &endpoint)
		if err != nil {
			log.WithError(err).Error("Fail to store endpoint")
		}

		err = dataStore.Set(ctx, endpoint.NetworkStorageKey(), &endpoint)
		if err != nil {
			log.WithError(err).Error("Fail to store network endpoint")
		}

		log.Infof("Duplicated %s => %s", oldID, endpoint.ID)
	}
}
