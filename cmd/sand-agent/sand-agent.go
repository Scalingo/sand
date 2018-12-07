package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/Scalingo/go-etcd-lock/lock"
	"github.com/Scalingo/go-handlers"
	"github.com/Scalingo/go-internal-tools/logger"
	dockeripam "github.com/Scalingo/go-plugins-helpers/ipam"
	dockernetwork "github.com/Scalingo/go-plugins-helpers/network"
	dockersdk "github.com/Scalingo/go-plugins-helpers/sdk"
	"github.com/Scalingo/go-utils/graceful"
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
	apptls "github.com/Scalingo/sand/utils/tls"
	"github.com/Scalingo/sand/web"
	"github.com/docker/docker/pkg/reexec"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func main() {
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

	store := store.New(c)
	peerListener := overlay.NewNetworkEndpointListener(ctx, c, store)

	managers := netmanager.NewManagerMap()
	managers.Set(types.OverlayNetworkType, overlay.NewManager(c, peerListener))

	etcdClient, err := etcd.NewClient()
	if err != nil {
		log.WithError(err).Error("fail to initialize etcd client")
		os.Exit(-1)
	}
	locker := lock.NewEtcdLocker(etcdClient)
	ipAllocator := ipallocator.New(c, store, locker)

	endpointRepository := endpoint.NewRepository(c, store, managers)
	networkRepository := network.NewRepository(c, store, managers)

	err = ensureNetworks(ctx, c, networkRepository, endpointRepository)
	if err != nil {
		log.WithError(err).Error("fail to ensure existing networks")
		os.Exit(-1)
	}

	vctrl := web.NewVersionController(c)
	nctrl := web.NewNetworksController(c, networkRepository, endpointRepository, ipAllocator)
	ectrl := web.NewEndpointsController(c, networkRepository, endpointRepository, ipAllocator)

	r := handlers.NewRouter(log)
	r.Use(handlers.ErrorMiddleware)
	r.HandleFunc("/version", vctrl.Show).Methods("GET")
	r.HandleFunc("/networks", nctrl.List).Methods("GET")
	r.HandleFunc("/networks", nctrl.Create).Methods("POST")
	r.HandleFunc("/networks/{id}", nctrl.Show).Methods("GET")
	r.HandleFunc("/networks/{id}", nctrl.Destroy).Methods("DELETE")
	r.HandleFunc("/networks/{id}", nctrl.Connect).Methods("CONNECT")
	r.HandleFunc("/endpoints", ectrl.Create).Methods("POST")
	r.HandleFunc("/endpoints", ectrl.List).Methods("GET")
	r.HandleFunc("/endpoints/{id}", ectrl.Destroy).Methods("DELETE")

	log.WithField("port", c.HttpPort).Info("Listening")
	serviceEndpoint := fmt.Sprintf(":%d", c.HttpPort)

	wg := &sync.WaitGroup{}

	if c.EnableDockerPlugin {
		log.WithField("port", c.DockerPluginHttpPort).Info("Enabling docker plugin")
		dockerRepository := docker.NewRepository(c, store)
		plugin := docker.NewDockerPlugin(
			c, networkRepository, endpointRepository, dockerRepository, ipAllocator,
		)
		manifest := `{"Implements": ["NetworkDriver", "IpamDriver"]}`
		handler := dockersdk.NewHandler(log, manifest)
		dockernetwork.ConfigureHandler(handler, plugin.DockerNetworkPlugin)
		dockeripam.ConfigureHandler(handler, plugin.DockerIPAMPlugin)

		err = docker.WritePluginSpecsOnDisk(ctx, c)
		if err != nil {
			log.WithError(err).Error("fail to write plugin spec file on disk")
			os.Exit(-1)
		}

		dockerPluginEndpoint := fmt.Sprintf(":%d", c.DockerPluginHttpPort)

		logDocker := log.WithField("service", "docker-plugin")
		ctxDocker := logger.ToCtx(ctx, logDocker)

		wg.Add(1)
		go func() {
			var err error

			defer wg.Done()
			if c.IsHttpTLSEnabled() {
				err = tlsListener(ctxDocker, c, dockerPluginEndpoint, handler)
			} else {
				gracefulService := graceful.NewService()
				err = gracefulService.ListenAndServe(ctxDocker, "tcp", dockerPluginEndpoint, handler)
			}
			if err != nil {
				log.WithError(err).Error("fail to intialize docker plugin listener")
				os.Exit(-1)
			}
			log.Info("docker plugin stopped")
		}()
	}

	logHandler := log.WithField("service", "sand-api")
	ctxHandler := logger.ToCtx(ctx, logHandler)

	if c.IsHttpTLSEnabled() {
		err = tlsListener(ctxHandler, c, serviceEndpoint, r)
	} else {
		gracefulService := graceful.NewService()
		err = gracefulService.ListenAndServe(ctxHandler, "tcp", serviceEndpoint, r)
	}
	if err != nil {
		log.WithError(err).Error("fail to listen and serve")
		os.Exit(-1)
	}
	log.Info("http API stopped")
	wg.Wait()
	log.Info("all APIs stopped, shutting down..")
}

func tlsListener(ctx context.Context, c *config.Config, serviceEndpoint string, handler http.Handler) error {
	config, err := apptls.NewConfig(c.HttpTLSCA, c.HttpTLSCert, c.HttpTLSKey, true)
	if err != nil {
		return errors.Wrapf(err, "fail to create tls configuration")
	}

	gracefulService := graceful.NewService()
	return gracefulService.ListenAndServeTLS(ctx, "tcp", serviceEndpoint, handler, config)
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
			"endpoint_id": endpoint.ID, "endpoint_netns_path": endpoint.TargetNetnsPath,
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
