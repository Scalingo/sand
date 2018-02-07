package sand

import (
	"context"
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
	NetworksList(context.Context) ([]types.Network, error)
	NetworkCreate(context.Context, params.NetworkCreate) (types.Network, error)
	NetworkDelete(context.Context, string) error
	EndpointCreate(context.Context, params.EndpointCreate) (types.Endpoint, error)
	EndpointsList(context.Context, params.EndpointsList) ([]types.Endpoint, error)
	EndpointDelete(context.Context, string) error
}

type httpClient struct {
	*http.Client
}

func (c httpClient) Do(r *http.Request) (*http.Response, error) {
	r.Header.Set("User-Agent", "sand v0.1.0")
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
	httpClient *httpClient
	tlsConfig  *tls.Config
}

var (
	_ Client = (*client)(nil)
)

type Opt func(c *client)

func NewClient(opts ...Opt) (*client, error) {
	c := &client{
		url: "http://localhost:9999",
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.httpClient == nil {
		c.httpClient = &httpClient{
			Client: &http.Client{
				Timeout: 30 * time.Second,
			},
		}
	}

	if c.tlsConfig != nil {
		c.httpClient.Transport = &http.Transport{
			TLSClientConfig: c.tlsConfig,
		}
	}
	return c, nil
}

func WithURL(url string) Opt {
	return func(c *client) {
		c.url = url
	}
}

func WithHttpClient(hc *http.Client) Opt {
	return func(c *client) {
		c.httpClient = &httpClient{hc}
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
