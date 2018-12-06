package web

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"gopkg.in/errgo.v1"

	"github.com/Scalingo/go-internal-tools/logger"
	"github.com/Scalingo/sand/api/params"
	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/client/sand"
	"github.com/Scalingo/sand/netutils"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (c NetworksController) Connect(w http.ResponseWriter, r *http.Request, urlparams map[string]string) error {
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()
	log := logger.Get(ctx).WithField("network_id", urlparams["id"])

	ip := r.URL.Query().Get("ip")
	port := r.URL.Query().Get("port")
	log = log.WithFields(logrus.Fields{
		"ip":   ip,
		"port": port,
	})
	ctx = logger.ToCtx(ctx, log)
	if ip == "" || port == "" {
		w.WriteHeader(400)
		return errors.New("IP and port are mandatory")
	}

	network, ok, err := c.NetworkRepository.Exists(ctx, urlparams["id"])
	if err != nil {
		return errors.Wrapf(err, "fail to query store")
	} else if !ok {
		w.WriteHeader(404)
		return errors.New("network not found")
	}

	endpoints, err := c.EndpointRepository.List(ctx, map[string]string{"network_id": network.ID})
	if err != nil {
		return errors.Wrapf(err, "fail to list endpoints for network %v", network)
	}

	activeEndpoints := []types.Endpoint{}
	var localEndpoint types.Endpoint
	for _, e := range endpoints {
		if e.Active {
			activeEndpoints = append(activeEndpoints, e)
			if e.HostIP == c.Config.PublicIP {
				localEndpoint = e
			}
		}
	}

	if len(activeEndpoints) == 0 {
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]string{"error": "no active endpoint in network " + network.ID})
		return nil
	}

	log.Info("hijacking http connection")
	h := w.(http.Hijacker)
	socket, _, err := h.Hijack()
	if err != nil {
		return errors.Wrapf(err, "fail to hijack http connection")
	}
	defer socket.Close()

	fmt.Fprintf(socket, "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\n\r\n")

	if localEndpoint.ID != "" {
		err := netutils.ForwardConnection(ctx, socket, localEndpoint.TargetNetnsPath, ip, port)
		if err != nil {
			return errors.Wrapf(err, "fail to hijack and forward connection to %v", localEndpoint)
		}
		return nil
	}

	endpoint := activeEndpoints[0]
	options := []sand.Opt{}
	scheme := "http"
	if c.Config.IsHttpTLSEnabled() {
		scheme = "https"

		config, err := sand.TlsConfig(c.Config.HttpTLSCA, c.Config.HttpTLSCert, c.Config.HttpTLSKey)
		if err != nil {
			return errgo.Notef(err, "fail to generate TLS configuration")
		}
		options = append(options, sand.WithTlsConfig(config))
	}
	url := fmt.Sprintf("%s://%s:%s", scheme, endpoint.Hostname, c.Config.HttpPort)
	options = append(options, sand.WithURL(url))
	client := sand.NewClient(options...)
	dstConn, err := client.NetworkConnect(ctx, network.ID, params.NetworkConnect{IP: ip, Port: port})
	if err != nil {
		return errors.Wrapf(err, "fail to connect sand %v", url)
	}

	wg := &sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		io.Copy(dstConn, socket)
	}()

	go func() {
		defer wg.Done()
		defer dstConn.Close()
		io.Copy(socket, dstConn)
	}()

	wg.Wait()

	return nil
}
