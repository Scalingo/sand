package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"path/filepath"

	"github.com/moby/moby/pkg/reexec"
	"github.com/sirupsen/logrus"

	"github.com/Scalingo/go-etcd-lock/v5/lock"
	"github.com/Scalingo/go-handlers"
	dockeripam "github.com/Scalingo/go-plugins-helpers/ipam"
	dockernetwork "github.com/Scalingo/go-plugins-helpers/network"
	dockersdk "github.com/Scalingo/go-plugins-helpers/sdk"
	"github.com/Scalingo/go-utils/cronsetup"
	"github.com/Scalingo/go-utils/errors/v3"
	"github.com/Scalingo/go-utils/graceful"
	"github.com/Scalingo/go-utils/logger"
	"github.com/Scalingo/go-utils/logger/plugins/rollbarplugin"
	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/config"
	"github.com/Scalingo/sand/endpoint"
	"github.com/Scalingo/sand/etcd"
	"github.com/Scalingo/sand/integrations/docker"
	"github.com/Scalingo/sand/ipallocator"
	"github.com/Scalingo/sand/network"
	"github.com/Scalingo/sand/network/netmanager"
	"github.com/Scalingo/sand/network/overlay"
	"github.com/Scalingo/sand/node"
	"github.com/Scalingo/sand/store"
	apptls "github.com/Scalingo/sand/utils/tls"
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

	c, err := config.Build(ctx)
	if err != nil {
		log.WithError(err).Error("Failed to generate initial config")
		os.Exit(-1)
	}

	err = c.CreateDirectories()
	if err != nil {
		log.WithError(err).Error("Failed to create runtime directories")
		os.Exit(-1)
	}

	dataStore := store.New(c)
	endpointsWatcher, err := store.NewWatcher(ctx, c, store.WithPrefix(types.NetworkEndpointStoragePrefix))
	if err != nil {
		log.WithError(err).Error("Failed to initialize store watcher")
	}
	peerListener := overlay.NewNetworkEndpointListener(ctx, c, endpointsWatcher, dataStore)

	managers := netmanager.NewManagerMap()
	managers.Set(types.OverlayNetworkType, overlay.NewManager(c, peerListener))

	etcdClient, err := etcd.NewClient(ctx)
	if err != nil {
		log.WithError(err).Error("Failed to initialize etcd client")
		os.Exit(-1)
	}

	locker := lock.NewEtcdLocker(etcdClient)
	ipAllocator := ipallocator.New(c, dataStore, locker)

	endpointRepository := endpoint.NewRepository(c, dataStore, managers)
	networkRepository := network.NewRepository(c, dataStore, managers)

	err = node.EnsureNetworkEndpoints(ctx, c, networkRepository, endpointRepository)
	if err != nil {
		log.WithError(err).Error("Failed to ensure existing networks")
		os.Exit(-1)
	}
	stopEndpointEnsureCron, err := setupEndpointEnsureCron(ctx, c, networkRepository, endpointRepository)
	if err != nil {
		log.WithError(err).Error("Failed to setup endpoint ensure cron")
		os.Exit(-1)
	}
	defer stopEndpointEnsureCron()

	vctrl := web.NewVersionController(c)
	nctrl := web.NewNetworksController(c, networkRepository, endpointRepository, ipAllocator)
	ectrl := web.NewEndpointsController(c, networkRepository, endpointRepository, ipAllocator)

	sandRouter := handlers.NewRouter(log)
	sandRouter.Use(handlers.ErrorMiddleware)
	sandRouter.HandleFunc("/version", vctrl.Show).Methods("GET")
	sandRouter.HandleFunc("/node/ensure-network-endpoints", nctrl.EnsureNetworkEndpoints).Methods("POST")
	sandRouter.HandleFunc("/networks", nctrl.List).Methods("GET")
	sandRouter.HandleFunc("/networks", nctrl.Create).Methods("POST")
	sandRouter.HandleFunc("/networks/{id}", nctrl.Show).Methods("GET")
	sandRouter.HandleFunc("/networks/{id}", nctrl.Destroy).Methods("DELETE")
	sandRouter.HandleFunc("/networks/{id}", nctrl.Connect).Methods("CONNECT")
	sandRouter.HandleFunc("/endpoints", ectrl.Create).Methods("POST")
	sandRouter.HandleFunc("/endpoints", ectrl.List).Methods("GET")
	sandRouter.HandleFunc("/endpoints/{id}", ectrl.Destroy).Methods("DELETE")

	log.WithField("port", c.HTTPPort).Info("Listening")
	serviceEndpoint := fmt.Sprintf(":%d", c.HTTPPort)

	// We can only have one graceful service per process since graceful 1.2.0
	numServers := 1
	if c.EnableDockerPlugin {
		numServers++
	}
	if c.PprofEnabled {
		numServers++
	}
	gracefulService := graceful.NewService(graceful.WithNumServers(numServers))

	var tlsConfig *tls.Config
	if c.IsHttpTLSEnabled() {
		tlsConfig, err = apptls.NewConfig(ctx, c.HTTPTLSCA, c.HTTPTLSCert, c.HTTPTLSKey, true)
		if err != nil {
			log.WithError(err).Error("Failed to create tls configuration")
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
			log.WithError(err).Error("Failed to write plugin spec file on disk")
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
			log.WithError(err).Error("Failed to initialize docker plugin listener")
			os.Exit(-1)
		}
	}

	if c.PprofEnabled {
		log.WithField("port", c.PprofPort).Info("Enabling pprof server")
		err = gracefulService.ListenAndServe(ctx, "tcp", fmt.Sprintf("localhost:%d", c.PprofPort), newPprofMux())
		if err != nil {
			log.WithError(err).Error("Failed to initialize pprof listener")
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
		log.WithError(err).Error("Failed to listen and serve")
		os.Exit(-1)
	}
	log.Info("HTTP API stopped")
	log.Info("Stop watching etcd changes")
	endpointsWatcher.Close(ctx)
	log.Info("All APIs stopped, shutting down..")
}

func newPprofMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	mux.Handle("/debug/pprof/allocs", pprof.Handler("allocs"))
	mux.Handle("/debug/pprof/block", pprof.Handler("block"))
	mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	mux.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))
	mux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))

	return mux
}

func setupEndpointEnsureCron(ctx context.Context, c *config.Config, repo network.Repository, erepo endpoint.Repository) (func(), error) {
	return cronsetup.Setup(ctx, cronsetup.SetupOpts{
		WithoutTelemetry: true,
		Jobs: []cronsetup.Job{
			{
				Name:   "ensure-network-endpoints",
				Rhythm: "@every " + c.EndpointEnsureInterval.String(),
				Func: func(ctx context.Context) error {
					err := node.EnsureNetworkEndpoints(ctx, c, repo, erepo)
					if err != nil {
						return errors.Wrap(ctx, err, "ensure network endpoints")
					}
					return nil
				},
			},
		},
	})
}
