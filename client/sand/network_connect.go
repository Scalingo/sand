package sand

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"runtime"
	"strings"
	"time"

	"github.com/Scalingo/go-utils/logger"
	"github.com/Scalingo/sand/api/params"
	"github.com/pkg/errors"
)

type sandConn struct {
	closed chan struct{}
	net.Conn
}

func (c sandConn) Close() error {
	close(c.closed)
	return c.Conn.Close()
}

func (c sandConn) dumpStackIfCloseForgotten(ctx context.Context, buffer []byte) {
	log := logger.Get(ctx)
	timer := time.NewTimer(24 * time.Hour)
	defer timer.Stop()

	log.Debug("watching connection close")
	select {
	case <-timer.C:
		log.Error("unclosed SAND connnection")
		fmt.Println(string(buffer))
	case <-c.closed:
	}
}

func (c *client) NetworkConnect(ctx context.Context, id string, opts params.NetworkConnect) (net.Conn, error) {
	req, err := http.NewRequest("CONNECT", fmt.Sprintf("%s/networks/%s?ip=%s&port=%s", c.url, id, opts.IP, opts.Port), nil)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to create http request")
	}
	req = req.WithContext(ctx)

	url, err := url.Parse(c.url)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to parse URL '%s'", c.url)
	}
	dial, err := net.Dial("tcp", url.Host)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to connect to %s", url.Host)
	}

	var conn *httputil.ClientConn
	if url.Scheme == "https" {
		host := strings.Split(url.Host, ":")[0]
		config := copyTLSConfig(c.tlsConfig)
		config.ServerName = host
		tlsConn := tls.Client(dial, config)
		conn = httputil.NewClientConn(tlsConn, nil)
	} else {
		conn = httputil.NewClientConn(dial, nil)
	}

	res, err := conn.Do(req)
	if err != httputil.ErrPersistEOF && err != nil {
		conn.Close()
		return nil, errors.Wrapf(err, "fail to execute CONNECT /networks/%s", id)
	}
	if res.StatusCode != 200 {
		conn.Close()
		return nil, errors.Errorf("invalid return code %v", res.StatusCode)
	}

	socket, _ := conn.Hijack()

	sandConn := sandConn{Conn: socket, closed: make(chan struct{})}

	buffer := make([]byte, 128*1024)
	n := runtime.Stack(buffer, false)
	go sandConn.dumpStackIfCloseForgotten(ctx, buffer[:n])
	return sandConn, nil
}

// We can't copy a tls.Config with a simple assignment (i.e. `config := *tls.Config) as go vet
// returns the error: "assignment copies lock value to config: crypto/tls.Config contains sync.Once
// contains sync.Mutex"
func copyTLSConfig(c *tls.Config) *tls.Config {
	return &tls.Config{
		Certificates:             c.Certificates,
		NameToCertificate:        c.NameToCertificate,
		GetCertificate:           c.GetCertificate,
		RootCAs:                  c.RootCAs,
		NextProtos:               c.NextProtos,
		ServerName:               c.ServerName,
		ClientAuth:               c.ClientAuth,
		ClientCAs:                c.ClientCAs,
		InsecureSkipVerify:       c.InsecureSkipVerify,
		CipherSuites:             c.CipherSuites,
		PreferServerCipherSuites: c.PreferServerCipherSuites,
		SessionTicketsDisabled:   c.SessionTicketsDisabled,
		SessionTicketKey:         c.SessionTicketKey,
		ClientSessionCache:       c.ClientSessionCache,
		MinVersion:               c.MinVersion,
		MaxVersion:               c.MaxVersion,
		CurvePreferences:         c.CurvePreferences,
	}
}
