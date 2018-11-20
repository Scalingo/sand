package sdk

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/Scalingo/go-handlers"
	"github.com/sirupsen/logrus"
)

const activatePath = "/Plugin.Activate"

// Handler is the base to create plugin handlers.
// It initializes connections and sockets to listen to.
type Handler struct {
	mux *handlers.Router
}

// NewHandler creates a new Handler with an http mux.
func NewHandler(logger logrus.FieldLogger, manifest string) Handler {
	mux := handlers.NewRouter(logger)
	mux.Use(handlers.ErrorMiddleware)

	mux.HandleFunc(activatePath, func(w http.ResponseWriter, r *http.Request, p map[string]string) error {
		w.Header().Set("Content-Type", DefaultContentTypeV1_1)
		fmt.Fprintln(w, manifest)
		return nil
	})

	return Handler{mux: mux}
}

// Serve sets up the handler to serve requests on the passed in listener
func (h Handler) Serve(l net.Listener) error {
	server := http.Server{
		Addr:    l.Addr().String(),
		Handler: h.mux,
	}
	return server.Serve(l)
}

func (h Handler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	h.mux.ServeHTTP(res, req)
}

// ServeTCP makes the handler to listen for request in a given TCP address.
// It also writes the spec file in the right directory for docker to read.
// Due to constrains for running Docker in Docker on Windows, data-root directory
// of docker daemon must be provided. To get default directory, use
// WindowsDefaultDaemonRootDir() function. On Unix, this parameter is ignored.
func (h Handler) ServeTCP(pluginName, addr, daemonDir string, tlsConfig *tls.Config) error {
	l, spec, err := newTCPListener(addr, pluginName, daemonDir, tlsConfig)
	if err != nil {
		return err
	}
	if spec != "" {
		defer os.Remove(spec)
	}
	return h.Serve(l)
}

// ServeUnix makes the handler to listen for requests in a unix socket.
// It also creates the socket file in the right directory for docker to read.
func (h Handler) ServeUnix(addr string, gid int) error {
	l, spec, err := newUnixListener(addr, gid)
	if err != nil {
		return err
	}
	if spec != "" {
		defer os.Remove(spec)
	}
	return h.Serve(l)
}

// ServeWindows makes the handler to listen for request in a Windows named pipe.
// It also creates the spec file in the right directory for docker to read.
// Due to constrains for running Docker in Docker on Windows, data-root directory
// of docker daemon must be provided. To get default directory, use
// WindowsDefaultDaemonRootDir() function. On Unix, this parameter is ignored.
func (h Handler) ServeWindows(addr, pluginName, daemonDir string, pipeConfig *WindowsPipeConfig) error {
	l, spec, err := newWindowsListener(addr, pluginName, daemonDir, pipeConfig)
	if err != nil {
		return err
	}
	if spec != "" {
		defer os.Remove(spec)
	}
	return h.Serve(l)
}

// HandleFunc registers a function to handle a request path with.
func (h Handler) HandleFunc(path string, fn handlers.HandlerFunc) {
	h.mux.HandleFunc(path, fn)
}
