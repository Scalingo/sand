package web

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"

	"github.com/Scalingo/go-internal-tools/logger"
	"github.com/docker/docker/pkg/reexec"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (c NetworksController) Connect(w http.ResponseWriter, r *http.Request, params map[string]string) error {
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()
	log := logger.Get(ctx).WithField("network_id", params["id"])

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

	network, ok, err := c.NetworkRepository.Exists(ctx, params["id"])
	if err != nil {
		return errors.Wrapf(err, "fail to query store")
	} else if !ok {
		w.WriteHeader(404)
		return errors.New("network not found")
	}

	log.Info("hijacking http connection")
	h := w.(http.Hijacker)
	socket, _, err := h.Hijack()
	if err != nil {
		return errors.Wrapf(err, "fail to hijack http connection")
	}
	defer socket.Close()

	fmt.Fprintf(socket, "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\n\r\n")

	tcpSocket := socket.(*net.TCPConn)
	socketFile, err := tcpSocket.File()
	if err != nil {
		return errors.Wrapf(err, "fail to get file from tcp connection")
	}
	defer socketFile.Close()

	cmd := &exec.Cmd{
		Path:       reexec.Self(),
		Args:       append([]string{"sc-netns-pipe-socket"}, network.NSHandlePath, ip, port),
		Stderr:     os.Stderr,
		Stdout:     os.Stdout,
		ExtraFiles: []*os.File{socketFile},
	}

	err = cmd.Run()
	if err != nil {
		return errors.Wrapf(err, "fail to pipe socket to %s %s:%s", network.NSHandlePath, ip, port)
	}

	return nil
}
