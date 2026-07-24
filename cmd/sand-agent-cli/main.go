package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/Scalingo/go-utils/errors/v3"
	"github.com/Scalingo/sand/client/sand"
)

var (
	// Version matches the version of the CLI It is updated dynamically by the
	// compiler when building a release
	Version = "v0.6-dev"
)

type App struct {
	config Config
	cli    *cli.Command
}

type Config struct {
	Version  string
	Timeout  time.Duration
	ApiURL   string
	CertFile string
	KeyFile  string
	CaFile   string
}

func main() {
	app := &App{
		config: Config{
			Version: Version,
		},
		cli: &cli.Command{},
	}
	app.cli.Flags = []cli.Flag{
		&cli.StringFlag{Name: "api-url", Value: "http://localhost:9999", Usage: "when requests will be sent", Sources: cli.EnvVars("SAND_API_URL")},
		&cli.StringFlag{Name: "cert-file", Usage: "identify HTTPS client using this SSL certificate file", Sources: cli.EnvVars("SAND_CERT_FILE")},
		&cli.StringFlag{Name: "key-file", Usage: "identify HTTPS client using this SSL key file", Sources: cli.EnvVars("SAND_KEY_FILE")},
		&cli.StringFlag{Name: "ca-file", Usage: "verify certificates of HTTPS-enabled servers using this CA bundle", Sources: cli.EnvVars("SAND_CA_FILE")},
		&cli.DurationFlag{Name: "timeout", Usage: "timeout for HTTP(S) requests made to SAND", Value: 30 * time.Second, Sources: cli.EnvVars("SAND_TIMEOUT")},
	}
	app.cli.Before = func(ctx context.Context, c *cli.Command) (context.Context, error) {
		app.config.ApiURL = c.String("api-url")
		app.config.CertFile = c.String("cert-file")
		app.config.KeyFile = c.String("key-file")
		app.config.CaFile = c.String("ca-file")
		app.config.Timeout = c.Duration("timeout")
		return ctx, nil
	}
	app.cli.Commands = []*cli.Command{
		{
			Name:   "version",
			Action: app.Version,
		},
		{
			Name:   "node-ensure-network-endpoints",
			Action: app.NodeEnsureNetworkEndpoints,
		},
		{
			Name:   "network-create",
			Action: app.NetworkCreate,
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "name", Usage: "name of the network to create"},
				&cli.StringFlag{Name: "ip-range", Usage: "IP Range from which endpoint IP will be allocated from"},
			},
		}, {
			Name:   "network-show",
			Action: app.NetworkShow,
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "network", Aliases: []string{"n"}, Usage: "ID of the network to display"},
			},
		}, {
			Name:   "network-list",
			Action: app.NetworksList,
		}, {
			Name:   "network-delete",
			Action: app.NetworkDelete,
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "network", Aliases: []string{"n"}, Usage: "ID of the network to delete"},
				&cli.BoolFlag{Name: "recursive", Aliases: []string{"r"}, Usage: "Also delete the endpoints"},
			},
		}, {
			Name:   "network-connect",
			Action: app.NetworkConnect,
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "network", Aliases: []string{"n"}, Usage: "ID of the network to connect to"},
				&cli.StringFlag{Name: "ip", Usage: "IP to reach in the network"},
				&cli.StringFlag{Name: "port", Usage: "Port to reach in the network"},
			},
		}, {
			Name:      "curl",
			Action:    app.Curl,
			Arguments: cli.AnyArguments,
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "network", Aliases: []string{"n"}, Usage: "ID of the network to connect to"},
				&cli.StringFlag{Name: "method", Aliases: []string{"X"}, Usage: "HTTP method to user", Value: "GET"},
				&cli.StringSliceFlag{Name: "header", Aliases: []string{"H"}, Usage: "HTTP header"},
				&cli.BoolFlag{Name: "insecure", Aliases: []string{"k"}, Usage: "By default, every SSL connection curl makes is verified to be secure. This option allows curl to proceed and operate even for server connections otherwise considered insecure."},
			},
		}, {
			Name:   "endpoint-list",
			Action: app.EndpointsList,
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "network", Aliases: []string{"n"}, Usage: "network id to use"},
				&cli.StringFlag{Name: "hostname", Value: "", Usage: "get endpoint of specific hostname, default is self, 'all' to get all endpoints"},
			},
		}, {
			Name:   "endpoint-create",
			Action: app.EndpointCreate,
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "network", Aliases: []string{"n"}, Usage: "network id to use"},
				&cli.StringFlag{Name: "ns", Usage: "path to the namespace file handle"},
				&cli.StringFlag{Name: "ip", Usage: "use a precise IP instead of a generated one (optional)"},
			},
		}, {
			Name:   "endpoint-delete",
			Action: app.EndpointDelete,
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "endpoint", Aliases: []string{"e"}, Usage: "ID of the endpoint to delete"},
			},
		},
	}
	err := app.cli.Run(context.Background(), os.Args)
	if err != nil {
		fmt.Println("An error occured:", err)
	}
}

func (a *App) sandClient(ctx context.Context, _ *cli.Command) (sand.Client, error) {
	opts := []sand.Opt{
		sand.WithURL(a.config.ApiURL),
		sand.WithTimeout(a.config.Timeout),
	}
	if a.config.CaFile != "" && a.config.CertFile != "" && a.config.KeyFile != "" {
		config, err := sand.TlsConfig(
			ctx, a.config.CaFile, a.config.CertFile, a.config.KeyFile,
		)
		if err != nil {
			return nil, errors.Wrap(ctx, err, "generate TLS configuration")
		}
		opts = append(opts, sand.WithTlsConfig(config))
	}
	return sand.NewClient(opts...), nil
}
