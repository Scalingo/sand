package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Scalingo/go-handlers"
	"github.com/Scalingo/go-internal-tools/logger"
	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/config"
	"github.com/Scalingo/sand/endpoint"
	"github.com/Scalingo/sand/network"
	"github.com/Scalingo/sand/network/overlay"
	"github.com/Scalingo/sand/store"
	"github.com/Scalingo/sand/web"
	"github.com/docker/docker/pkg/reexec"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func main() {
	log := logger.Default()
	log.SetLevel(logrus.DebugLevel)
	ctx := logger.ToCtx(context.Background(), log)

	// If reexec to create network namespace
	if filepath.Base(os.Args[0]) != "sand" {
		log.WithField("args", os.Args).Info("reexec")
	}
	ok := reexec.Init()
	if ok {
		log.WithField("args", os.Args).Info("reexec done")
		return
	}

	c, err := config.Build()
	if err != nil {
		log.WithError(err).Error("fail to generate initial config")
		os.Exit(-1)
	}

	err = c.CreateDirectories()
	if err != nil {
		log.WithError(err).Error("fail to create runtime directories")
		os.Exit(-1)
	}

	store := store.New(c)
	peerListener := overlay.NewNetworkEndpointListener(c, store)

	err = ensureNetworks(ctx, c, peerListener)
	if err != nil {
		log.WithError(err).Error("fail to ensure existing networks")
		os.Exit(-1)
	}

	r := handlers.NewRouter(log)
	r.Use(handlers.ErrorMiddleware)

	nctrl := web.NewNetworksController(c, peerListener)
	ectrl := web.NewEndpointsController(c, peerListener)

	r.HandleFunc("/networks", nctrl.List).Methods("GET")
	r.HandleFunc("/networks", nctrl.Create).Methods("POST")
	r.HandleFunc("/networks/{id}", nctrl.Destroy).Methods("DELETE")
	r.HandleFunc("/endpoints", ectrl.Create).Methods("POST")

	log.WithField("port", c.HttpPort).Info("Listening")
	http.ListenAndServe(fmt.Sprintf(":%d", c.HttpPort), r)
}

func ensureNetworks(ctx context.Context, c *config.Config, listener overlay.NetworkEndpointListener) error {
	log := logger.Get(ctx)
	ctx = logger.ToCtx(ctx, log)

	log.Info("ensure networks on node")

	s := store.New(c)
	var networks []types.Network
	err := s.Get(ctx, fmt.Sprintf("/nodes/%s/networks/", c.PublicHostname), true, &networks)
	if err == store.ErrNotFound {
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "fail to get existing networks on %v", c.PublicHostname)
	}

	repo := network.NewRepository(c, s, listener)
	erepo := endpoint.NewRepository(c, s)

	for _, network := range networks {
		log = log.WithField("network_id", network.ID)

		err = s.Get(ctx, network.StorageKey(), false, &network)
		if err != nil {
			log.WithError(err).Error("fail to get network details")
			continue
		}

		log = log.WithField("network_name", network.Name)
		ctx = logger.ToCtx(ctx, log)

		log.Info("ensuring network is setup")
		err = repo.Ensure(ctx, network)
		if err != nil {
			log.WithError(err).Error("fail to ensure network")
			continue
		}

		var endpoints []types.Endpoint
		err = s.Get(ctx, network.EndpointsStorageKey(c.PublicHostname), true, &endpoints)
		if err == store.ErrNotFound {
			continue
		}
		if err != nil {
			log.WithError(err).Error("fail to list network endpoints")
			continue
		}

		log.Info("insuring network endpoints are setup")
		for _, endpoint := range endpoints {
			log = log.WithFields(logrus.Fields{
				"endpoint_id": endpoint.ID, "endpoint_netns_path": endpoint.TargetNetnsPath,
			})
			log.Info("restoring endpoint")
			ctx = logger.ToCtx(ctx, log)
			endpoint, err = erepo.Ensure(ctx, network, endpoint)
			if err != nil {
				log.WithError(err).Error("fail to ensure endpoint")
				continue
			}
		}
	}
	return nil
}
