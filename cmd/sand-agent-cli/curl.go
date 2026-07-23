package main

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/Scalingo/go-utils/errors/v3"
	"github.com/Scalingo/sand/client/sand"
)

func (a *App) Curl(ctx context.Context, c *cli.Command) error {
	client, err := a.sandClient(ctx, c)
	if err != nil {
		return errors.Wrap(ctx, err, "create sand client")
	}

	if c.String("network") == "" {
		return errors.New(ctx, "network flag is mandatory")
	}

	if c.NArg() == 0 {
		return errors.New(ctx, "URL is mandatory")
	}

	tlsConfig := tls.Config{}
	if c.Bool("insecure") {
		tlsConfig.InsecureSkipVerify = true
	}

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
		Transport: client.NewHTTPRoundTripper(ctx, c.String("network"), sand.HTTPRoundTripperOpts{
			TLSConfig: &tlsConfig,
		}),
	}

	var body io.Reader
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		body = os.Stdin
	}

	req, err := http.NewRequest(c.String("method"), c.Args().Get(0), body)
	if err != nil {
		return errors.Wrap(ctx, err, "fail to build request")
	}

	for _, header := range c.StringSlice("header") {
		headerSplit := strings.SplitN(header, ":", 2)
		req.Header.Add(strings.Trim(headerSplit[0], " "), strings.Trim(headerSplit[1], " "))
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return errors.Wrap(ctx, err, "fail to make HTTP request")
	}
	_, err = io.Copy(os.Stdout, resp.Body)
	if err != nil {
		return errors.Wrap(ctx, err, "fail to copy response body")
	}
	return nil
}
