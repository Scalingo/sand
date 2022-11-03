package web

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sync"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"gopkg.in/errgo.v1"

	"github.com/Scalingo/go-utils/logger"
	"github.com/Scalingo/sand/api/params"
	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/client/sand"
	"github.com/Scalingo/sand/netutils"
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

	log.Infof("hijacking http connection and forward to %v", localEndpoint.Hostname)
	h := w.(http.Hijacker)
	socket, _, err := h.Hijack()
	if err != nil {
		return errors.Wrapf(err, "fail to hijack http connection")
	}
	fmt.Fprintf(socket, "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\n\r\n")

	// If this is the destination node, forward the connection in the right namespace
	// Otherwise the connection will be forwarded to another SAND agent which has
	// the network namespace available on it.
	if localEndpoint.ID != "" {
		err := netutils.ForwardConnection(ctx, socket, localEndpoint.TargetNetnsPath, ip, port)
		if operr, ok := errors.Cause(err).(*net.OpError); ok {
			if syscallerr, ok := operr.Err.(*os.SyscallError); ok && syscallerr.Err == unix.ECONNREFUSED {
				// It happens that the target from the network connection (ip:port) is not
				// actually bound in the network namespace, in this case a standard
				// connection refused error is sent, this should not be an error, the
				// connection to the SAND client should just be stopped normally
				log.WithError(err).Infof("local endpoint %v not binding port", localEndpoint)
			} else if syscallerr, ok := operr.Err.(*os.SyscallError); ok && syscallerr.Err == unix.EHOSTUNREACH {
				// It's also possible that the targeted IP is not reachable anymore and it leads
				// to a no route to host error. This error is not related to sand itself
				log.WithError(err).Infof("local endpoint %v no route to host", localEndpoint)
			} else {
				return errors.Wrapf(err, "network connection error when forwarding to %v", localEndpoint)
			}
		} else if err != nil {
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
			socket.Close()
			return errgo.Notef(err, "fail to generate TLS configuration")
		}
		options = append(options, sand.WithTlsConfig(config))
	}
	url := fmt.Sprintf("%s://%s:%d", scheme, endpoint.HostIP, c.Config.HttpPort)
	options = append(options, sand.WithURL(url))
	client := sand.NewClient(options...)

	log.Infof("forwarding connection to %v", url)
	dstConn, err := client.NetworkConnect(ctx, network.ID, params.NetworkConnect{IP: ip, Port: port})
	if err != nil {
		socket.Close()
		return errors.Wrapf(err, "fail to connect sand %v", url)
	}

	// The two following conditions should never be wrong, but who knows, the two only
	// types are *net.TCPConn or *tls.Conn, both fit this interface
	src, ok := socket.(netutils.Conn)
	if !ok {
		socket.Close()
		return errors.Wrapf(err, "src socket does not implement netutils.Conn")
	}
	dst, ok := dstConn.(netutils.Conn)
	if !ok {
		socket.Close()
		return errors.Wrapf(err, "dst socket does not implement netutils.Conn")
	}

	wg := &sync.WaitGroup{}
	wg.Add(2)

	// Here're we're forwarding connection, necessarily, one side or another will
	// close the connection at some point, this should not be detected as an
	// error.  Either the client should detect an error or the destination, but
	// SAND here is just acting as a pipe.
	go func() {
		defer wg.Done()
		defer src.Close()
		defer dst.CloseWrite()
		_, err := io.Copy(dst, src)
		if err != nil && err != io.EOF {
			log.WithError(err).Info("fail to copy data from src socket to next sand agent")
			return
		}
		log.Info("end of connection from client to next sand agent")
	}()

	go func() {
		defer wg.Done()
		defer src.CloseWrite()
		defer dst.Close()
		_, err := io.Copy(src, dst)
		if err != nil && err != io.EOF {
			log.WithError(err).Info("fail to copy data next sand agent to src")
			return
		}
		log.Info("end of connection from next sand agent to src")
	}()

	wg.Wait()

	return nil
}
