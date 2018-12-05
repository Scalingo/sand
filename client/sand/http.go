package sand

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"

	"github.com/Scalingo/sand/api/params"
	"github.com/pkg/errors"
)

func (c *client) rawDialer(ctx context.Context, sandNetworkID, network, address string) (net.Conn, error) {
	if network != "tcp" {
		return nil, errors.New("only TCP connections are supported")
	}

	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid address: %s", address)
	}

	return c.NetworkConnect(ctx, sandNetworkID, params.NetworkConnect{
		IP:   host,
		Port: port,
	})
}

type HTTPRoundTripperOpts struct {
	TLSConfig *tls.Config
}

func (c *client) NewHTTPRoundTripper(ctx context.Context, id string, opts HTTPRoundTripperOpts) http.RoundTripper {
	return &http.Transport{
		TLSClientConfig: opts.TLSConfig,
		Dial: func(n, a string) (net.Conn, error) {
			return c.rawDialer(ctx, id, n, a)
		},
		DialContext: func(ctx context.Context, n, a string) (net.Conn, error) {
			return c.rawDialer(ctx, id, n, a)
		},
	}
}
