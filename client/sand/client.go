package sand

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/Scalingo/sand/api/httpresp"
	"github.com/Scalingo/sand/api/params"
	"github.com/Scalingo/sand/api/types"
	apptls "github.com/Scalingo/sand/utils/tls"
	"github.com/pkg/errors"

	"crypto/tls"
)

type Client interface {
	Version(context.Context) (string, error)
	NetworksList(context.Context) ([]types.Network, error)
	NetworkCreate(context.Context, params.NetworkCreate) (types.Network, error)
	NetworkShow(context.Context, string) (types.Network, error)
	NetworkConnect(context.Context, string, params.NetworkConnect) (net.Conn, error)
	NetworkDelete(context.Context, string) error
	EndpointCreate(context.Context, params.EndpointCreate) (types.Endpoint, error)
	EndpointsList(context.Context, params.EndpointsList) ([]types.Endpoint, error)
	EndpointDelete(context.Context, string) error
	NewHTTPRoundTripper(ctx context.Context, id string, opts HTTPRoundTripperOpts) http.RoundTripper
}

type httpClient struct {
	*http.Client
}

func (c httpClient) Do(r *http.Request) (*http.Response, error) {
	r.Header.Set("User-Agent", "SAND Client")
	if r.Header.Get("Accept") == "" {
		r.Header.Set("Accept", "application/json")
	}
	if r.Header.Get("Content-Type") == "" {
		r.Header.Set("Content-Type", "application/json")
	}
	res, err := c.Client.Do(r)
	if err != nil {
		return nil, err
	}
	if res.StatusCode == 404 {
		err := res.Body.Close()
		if err != nil {
			return nil, errors.Wrapf(err, "fail to close HTTP body")
		}
		return nil, httpresp.Error{Error_: "Not found"}
	}
	return res, nil
}

type client struct {
	url        string
	timeout    time.Duration
	httpClient *httpClient
	tlsConfig  *tls.Config
}

var (
	_ Client = (*client)(nil)
)

type Opt func(c *client)

func NewClient(opts ...Opt) *client {
	c := &client{
		url:     "http://localhost:9999",
		timeout: 30 * time.Second,
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.httpClient == nil {
		c.httpClient = &httpClient{
			Client: &http.Client{
				Timeout: c.timeout,
			},
		}
	}

	if c.tlsConfig != nil {
		c.httpClient.Transport = &http.Transport{
			TLSClientConfig: c.tlsConfig,
		}
	}
	return c
}

func WithURL(url string) Opt {
	return func(c *client) {
		c.url = url
	}
}

func WithHttpClient(hc *http.Client) Opt {
	return func(c *client) {
		c.httpClient = &httpClient{
			Client: hc,
		}
	}
}

func WithTimeout(t time.Duration) Opt {
	return func(c *client) {
		c.timeout = t
	}
}

func WithTlsConfig(config *tls.Config) Opt {
	return func(c *client) {
		c.tlsConfig = config
	}
}

func TlsConfig(ca, cert, key string) (*tls.Config, error) {
	return apptls.NewConfig(ca, cert, key, false)
}
