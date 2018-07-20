package sand

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/Scalingo/sand/api/params"
	"github.com/pkg/errors"
)

func (c *client) NetworkConnect(ctx context.Context, id string, opts params.NetworkConnect) (net.Conn, error) {
	req, err := http.NewRequest("CONNECT", fmt.Sprintf("%s/networks/%s?ip=%s&port=%s", c.url, id, opts.IP, opts.Port), nil)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to create http request")
	}
	req = req.WithContext(ctx)

	url, err := url.Parse(c.url)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to parse URL", c.url)
	}
	dial, err := net.Dial("tcp", url.Host)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to connect to %s", url.Host)
	}

	var conn *httputil.ClientConn
	if url.Scheme == "https" {
		host := strings.Split(url.Host, ":")[0]
		config := *c.tlsConfig
		config.ServerName = host
		tls_conn := tls.Client(dial, &config)
		conn = httputil.NewClientConn(tls_conn, nil)
	} else {
		conn = httputil.NewClientConn(dial, nil)
	}

	res, err := conn.Do(req)
	if err != httputil.ErrPersistEOF && err != nil {
		return nil, errors.Wrapf(err, "fail to execute CONNECT /networks/%s", id)
	}
	if res.StatusCode != 200 {
		return nil, errors.Errorf("invalid return code %v", res.StatusCode)
	}

	socket, _ := conn.Hijack()
	return socket, nil
}
