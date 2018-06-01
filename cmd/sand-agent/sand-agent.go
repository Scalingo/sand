package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/Scalingo/go-etcd-lock/lock"
	"github.com/Scalingo/go-handlers"
	"github.com/Scalingo/go-internal-tools/logger"
	dockeripam "github.com/Scalingo/go-plugins-helpers/ipam"
	dockernetwork "github.com/Scalingo/go-plugins-helpers/network"
	dockersdk "github.com/Scalingo/go-plugins-helpers/sdk"
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
	log := logger.Default()
	log.SetLevel(logrus.InfoLevel)
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

	etcdClient, err := etcd.NewClient(c)
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

	nctrl := web.NewNetworksController(c, networkRepository, endpointRepository)
	ectrl := web.NewEndpointsController(c, networkRepository, endpointRepository)

	r := handlers.NewRouter(log)
	r.Use(handlers.ErrorMiddleware)
	r.HandleFunc("/networks", nctrl.List).Methods("GET")
	r.HandleFunc("/networks", nctrl.Create).Methods("POST")
	r.HandleFunc("/networks/{id}", nctrl.Destroy).Methods("DELETE")
	r.HandleFunc("/endpoints", ectrl.Create).Methods("POST")
	r.HandleFunc("/endpoints", ectrl.List).Methods("GET")
	r.HandleFunc("/endpoints/{id}", ectrl.Destroy).Methods("DELETE")

	log.WithField("port", c.HttpPort).Info("Listening")
	serviceEndpoint := fmt.Sprintf(":%d", c.HttpPort)

	wg := &sync.WaitGroup{}

	if c.EnableDockerPlugin {
		log.Info("enabling docker plugin")
		dockerRepository := docker.NewRepository(c, store)
		plugin := docker.NewDockerPlugin(
			c, networkRepository, endpointRepository, dockerRepository, ipAllocator,
		)
		manifest := `{"Implements": ["NetworkDriver", "IpamDriver"]}`
		handler := dockersdk.NewHandler(log, manifest)
		dockernetwork.ConfigureHandler(handler, plugin.DockerNetworkPlugin)
		dockeripam.ConfigureHandler(handler, plugin.DockerIPAMPlugin)

		var listener net.Listener
		dockerPluginEndpoint := fmt.Sprintf(":%d", c.DockerPluginHttpPort)
		if c.HttpTLSCA != "" {
			listener, err = tlsListener(c, dockerPluginEndpoint)
		} else {
			listener, err = net.Listen("tcp", dockerPluginEndpoint)
		}
		if err != nil {
			log.WithError(err).Error("fail to intialize listener")
			os.Exit(-1)
		}

		err = docker.WritePluginSpecsOnDisk(ctx, c)
		if err != nil {
			log.WithError(err).Error("fail to write plugin spec file on disk")
			os.Exit(-1)
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			err := handler.Serve(listener)
			if err != nil && !strings.Contains(err.Error(), "use of closed") {
				log.WithError(err).Error("error after serving HTTP")
			}
			log.Info("docker plugin stopped")
		}()
	}

	var listener net.Listener
	if c.HttpTLSCA != "" {
		listener, err = tlsListener(c, serviceEndpoint)
	} else {
		listener, err = net.Listen("tcp", serviceEndpoint)
	}
	if err != nil {
		log.WithError(err).Error("fail to intialize listener")
		os.Exit(-1)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := http.Serve(listener, r)
		if err != nil && !strings.Contains(err.Error(), "use of closed") {
			log.WithError(err).Error("error after serving HTTP")
		}
		log.Info("http API stopped")
	}()

	sigs := make(chan os.Signal)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
	s := <-sigs
	log.WithField("signal", s).Info("signal catched shuting down")

	err = listener.Close()
	if err != nil {
		log.WithError(err).Error("fail to close listener")
	}

	wg.Wait()
}

func tlsListener(c *config.Config, serviceEndpoint string) (net.Listener, error) {
	config, err := apptls.NewConfig(c.HttpTLSCA, c.HttpTLSCert, c.HttpTLSKey, true)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to create tls configuration")
	}

	listener, err := tls.Listen("tcp", serviceEndpoint, config)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to create tls listener")
	}

	return listener, nil
}

func ensureNetworks(ctx context.Context, c *config.Config, repo network.Repository, erepo endpoint.Repository) error {
	log := logger.Get(ctx)
	ctx = logger.ToCtx(ctx, log)

	log.Info("ensure networks on node")

	endpoints, err := erepo.List(ctx, map[string]string{"hostname": c.PublicHostname})
	if err == store.ErrNotFound {
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "fail to list endpoints of %v", c.PublicHostname)
	}

	for _, endpoint := range endpoints {
		if !endpoint.Active {
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
		})
		if err != nil {
			log.WithError(err).Error("fail to ensure endpoint")
			continue
		}
	}
	return nil
}
