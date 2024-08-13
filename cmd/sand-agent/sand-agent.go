package main

import (
	"context"
	"crypto/tls"
	"fmt"
	apptls "github.com/Scalingo/sand/utils/tls"
	"github.com/moby/moby/pkg/reexec"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"

	"github.com/Scalingo/go-etcd-lock/v5/lock"
	"github.com/Scalingo/go-handlers"
	dockeripam "github.com/Scalingo/go-plugins-helpers/ipam"
	dockernetwork "github.com/Scalingo/go-plugins-helpers/network"
	dockersdk "github.com/Scalingo/go-plugins-helpers/sdk"
	"github.com/Scalingo/go-utils/graceful"
	"github.com/Scalingo/go-utils/logger"
	"github.com/Scalingo/go-utils/logger/plugins/rollbarplugin"
	"github.com/Scalingo/sand/api/params"
	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/config"
	"github.com/Scalingo/sand/endpoint"
	"github.com/Scalingo/sand/etcd"
	"github.com/Scalingo/sand/integrations/docker"
	"github.com/Scalingo/sand/ipallocator"
	"github.com/Scalingo/sand/network"
	"github.com/Scalingo/sand/network/netmanager"
	"github.com/Scalingo/sand/network/overlay"
	"github.com/Scalingo/sand/store"
	"github.com/Scalingo/sand/web"
)

func main() {
	rollbarplugin.Register()
	log := logrus.FieldLogger(logger.Default())
	ctx := logger.ToCtx(context.Background(), log)

	// If reexec to create network namespace
	if filepath.Base(os.Args[0]) != "sand-agent" {
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

	dataStore := store.New(c)
	endpointsWatcher, err := store.NewWatcher(ctx, c, store.WithPrefix(types.NetworkEndpointStoragePrefix))
	if err != nil {
		log.WithError(err).Error("fail to initialize store watcher")
	}
	peerListener := overlay.NewNetworkEndpointListener(ctx, c, endpointsWatcher, dataStore)

	managers := netmanager.NewManagerMap()
	managers.Set(types.OverlayNetworkType, overlay.NewManager(c, peerListener))

	etcdClient, err := etcd.NewClient()
	if err != nil {
		log.WithError(err).Error("fail to initialize etcd client")
		os.Exit(-1)
	}

	locker := lock.NewEtcdLocker(etcdClient)
	ipAllocator := ipallocator.New(c, dataStore, locker)

	endpointRepository := endpoint.NewRepository(c, dataStore, managers)
	networkRepository := network.NewRepository(c, dataStore, managers)

	err = ensureNetworks(ctx, c, networkRepository, endpointRepository)
	if err != nil {
		log.WithError(err).Error("fail to ensure existing networks")
		os.Exit(-1)
	}

	vctrl := web.NewVersionController(c)
	nctrl := web.NewNetworksController(c, networkRepository, endpointRepository, ipAllocator)
	ectrl := web.NewEndpointsController(c, networkRepository, endpointRepository, ipAllocator)

	sandRouter := handlers.NewRouter(log)
	sandRouter.Use(handlers.ErrorMiddleware)
	sandRouter.HandleFunc("/version", vctrl.Show).Methods("GET")
	sandRouter.HandleFunc("/networks", nctrl.List).Methods("GET")
	sandRouter.HandleFunc("/networks", nctrl.Create).Methods("POST")
	sandRouter.HandleFunc("/networks/{id}", nctrl.Show).Methods("GET")
	sandRouter.HandleFunc("/networks/{id}", nctrl.Destroy).Methods("DELETE")
	sandRouter.HandleFunc("/networks/{id}", nctrl.Connect).Methods("CONNECT")
	sandRouter.HandleFunc("/endpoints", ectrl.Create).Methods("POST")
	sandRouter.HandleFunc("/endpoints", ectrl.List).Methods("GET")
	sandRouter.HandleFunc("/endpoints/{id}", ectrl.Destroy).Methods("DELETE")

	log.WithField("port", c.HttpPort).Info("Listening")
	serviceEndpoint := fmt.Sprintf(":%d", c.HttpPort)

	// We can only have one graceful service per process
	numServers := 1
	if c.EnableDockerPlugin {
		numServers++
	}
	gracefulService := graceful.NewService(graceful.WithNumServers(numServers))

	var tlsConfig *tls.Config
	if c.IsHttpTLSEnabled() {
		tlsConfig, err = apptls.NewConfig(c.HttpTLSCA, c.HttpTLSCert, c.HttpTLSKey, true)
		if err != nil {
			log.WithError(err).Error("fail to create tls configuration")
			os.Exit(-1)
		}
	}

	if c.EnableDockerPlugin {
		log.WithField("port", c.DockerPluginHttpPort).Info("Enabling docker plugin")
		dockerRepository := docker.NewRepository(c, dataStore)
		plugin := docker.NewDockerPlugin(
			c, networkRepository, endpointRepository, dockerRepository, ipAllocator,
		)
		manifest := `{"Implements": ["NetworkDriver", "IpamDriver"]}`
		dockerPluginRouter := dockersdk.NewHandler(log, manifest)
		dockernetwork.ConfigureHandler(dockerPluginRouter, plugin.DockerNetworkPlugin)
		dockeripam.ConfigureHandler(dockerPluginRouter, plugin.DockerIPAMPlugin)

		err = docker.WritePluginSpecsOnDisk(ctx, c)
		if err != nil {
			log.WithError(err).Error("fail to write plugin spec file on disk")
			os.Exit(-1)
		}

		dockerPluginEndpoint := fmt.Sprintf(":%d", c.DockerPluginHttpPort)

		logDocker := log.WithField("service", "docker-plugin")
		ctxDocker := logger.ToCtx(ctx, logDocker)

		if c.IsHttpTLSEnabled() {
			err = gracefulService.ListenAndServeTLS(ctxDocker, "tcp", dockerPluginEndpoint, dockerPluginRouter, tlsConfig)
		} else {
			err = gracefulService.ListenAndServe(ctxDocker, "tcp", dockerPluginEndpoint, dockerPluginRouter)
		}
		if err != nil {
			log.WithError(err).Error("fail to initialize docker plugin listener")
			os.Exit(-1)
		}
	}

	logHandler := log.WithField("service", "sand-api")
	ctxHandler := logger.ToCtx(ctx, logHandler)

	if c.IsHttpTLSEnabled() {
		err = gracefulService.ListenAndServeTLS(ctxHandler, "tcp", serviceEndpoint, sandRouter, tlsConfig)
	} else {
		err = gracefulService.ListenAndServe(ctxHandler, "tcp", serviceEndpoint, sandRouter)
	}
	if err != nil {
		log.WithError(err).Error("fail to listen and serve")
		os.Exit(-1)
	}
	log.Info("HTTP API stopped")
	log.Info("Stop watching etcd changes")
	endpointsWatcher.Close()
	log.Info("All APIs stopped, shutting down..")
}

func ensureNetworks(ctx context.Context, c *config.Config, repo network.Repository, erepo endpoint.Repository) error {
	log := logger.Get(ctx)
	ctx = logger.ToCtx(ctx, log)

	log.Info("Ensure networks on node")

	endpoints, err := erepo.List(ctx, map[string]string{"hostname": c.PublicHostname})
	if err == store.ErrNotFound {
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "fail to list endpoints of %v", c.PublicHostname)
	}

	for _, endpoint := range endpoints {
		log = log.WithField("endpoint_id", endpoint.ID)
		ctx = logger.ToCtx(ctx, log)
		if !endpoint.Active {
			log.Debug("skip inactive endpoint")
			continue
		}
		log = log.WithFields(logrus.Fields{
			"network_id":          endpoint.NetworkID,
			"endpoint_id":         endpoint.ID,
			"endpoint_netns_path": endpoint.TargetNetnsPath,
		})
		log.Info("restoring endpoint")

		network, ok, err := repo.Exists(ctx, endpoint.NetworkID)
		if err != nil {
			return errors.Wrapf(err, "fail to get network")
		}
		if !ok {
			log.WithError(errors.Errorf("network not found for %v", endpoint))
			continue
		}

		log.Info("ensuring network")
		err = repo.Ensure(ctx, network)
		if err != nil {
			log.WithError(err).Error("fail to ensure network")
			continue
		}

		endpoint, err = erepo.Activate(ctx, network, endpoint, params.EndpointActivate{
			NSHandlePath: endpoint.TargetNetnsPath,
			SetAddr:      true,
			MoveVeth:     true,
		})
		if err != nil {
			// if we can't activate the endpoint because the netns path doesn't exist anymore, we
			// just deactivate it. Otherwise we raise an error.
			if os.IsNotExist(errors.Cause(err)) {
				endpoint, err = erepo.Deactivate(ctx, network, endpoint)
				if err != nil {
					log.WithError(err).Error("fail to deactivate endpoint")
					continue
				}
			} else {
				log.WithError(err).Error("fail to ensure endpoint")
				continue
			}
		}
	}
	return nil
}
