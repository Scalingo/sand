package main

import (
	"fmt"
	"os"
	"time"

	"github.com/Scalingo/sand/client/sand"
	"github.com/urfave/cli"
)

var (
	// Version matches the version of the CLI It is updated dynamically by the
	// compiler when building a release
	Version = "v0.6-dev"
)

type App struct {
	config Config
	cli    *cli.App
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
		cli: cli.NewApp(),
	}
	app.cli.Flags = []cli.Flag{
		cli.StringFlag{Name: "api-url", Value: "http://localhost:9999", Usage: "when requests will be sent", EnvVar: "SAND_API_URL"},
		cli.StringFlag{Name: "cert-file", Usage: "identify HTTPS client using this SSL certificate file", EnvVar: "SAND_CERT_FILE"},
		cli.StringFlag{Name: "key-file", Usage: "identify HTTPS client using this SSL key file", EnvVar: "SAND_KEY_FILE"},
		cli.StringFlag{Name: "ca-file", Usage: "verify certificates of HTTPS-enabled servers using this CA bundle", EnvVar: "SAND_CA_FILE"},
		cli.DurationFlag{Name: "timeout", Usage: "timeout for HTTP(S) requests made to SAND", Value: 30 * time.Second, EnvVar: "SAND_TIMEOUT"},
	}
	app.cli.Before = func(c *cli.Context) error {
		app.config.ApiURL = c.GlobalString("api-url")
		app.config.CertFile = c.GlobalString("cert-file")
		app.config.KeyFile = c.GlobalString("key-file")
		app.config.CaFile = c.GlobalString("ca-file")
		app.config.Timeout = c.GlobalDuration("timeout")
		return nil
	}
	app.cli.Commands = cli.Commands{
		{
			Name:   "version",
			Action: app.Version,
		},
		{
			Name:   "network-create",
			Action: app.NetworkCreate,
			Flags: []cli.Flag{
				cli.StringFlag{Name: "name", Usage: "name of the network to create"},
			},
		}, {
			Name:   "network-show",
			Action: app.NetworkShow,
			Flags: []cli.Flag{
				cli.StringFlag{Name: "network,n", Usage: "ID of the network to display"},
			},
		}, {
			Name:   "network-list",
			Action: app.NetworksList,
		}, {
			Name:   "network-delete",
			Action: app.NetworkDelete,
			Flags: []cli.Flag{
				cli.StringFlag{Name: "network,n", Usage: "ID of the network to delete"},
				cli.BoolFlag{Name: "recursive,r", Usage: "Also delete the endpoints"},
			},
		}, {
			Name:   "network-connect",
			Action: app.NetworkConnect,
			Flags: []cli.Flag{
				cli.StringFlag{Name: "network,n", Usage: "ID of the network to connect to"},
				cli.StringFlag{Name: "ip", Usage: "IP to reach in the network"},
				cli.StringFlag{Name: "port", Usage: "Port to reach in the network"},
			},
		}, {
			Name:   "curl",
			Action: app.Curl,
			Flags: []cli.Flag{
				cli.StringFlag{Name: "network,n", Usage: "ID of the network to connect to"},
				cli.StringFlag{Name: "method,X", Usage: "HTTP method to user", Value: "GET"},
				cli.StringSliceFlag{Name: "header,H", Usage: "HTTP header"},
				cli.BoolFlag{Name: "insecure,k", Usage: "By default, every SSL connection curl makes is verified to be secure. This option allows curl to proceed and operate even for server connections otherwise considered insecure."},
			},
		}, {
			Name:   "endpoint-list",
			Action: app.EndpointsList,
			Flags: []cli.Flag{
				cli.StringFlag{Name: "network,n", Usage: "network id to use"},
				cli.StringFlag{Name: "hostname", Value: "", Usage: "get endpoint of specific hostname, default is self, 'all' to get all endpoints"},
			},
		}, {
			Name:   "endpoint-create",
			Action: app.EndpointCreate,
			Flags: []cli.Flag{
				cli.StringFlag{Name: "network,n", Usage: "network id to use"},
				cli.StringFlag{Name: "ns", Usage: "path to the namespace file handle"},
				cli.StringFlag{Name: "ip", Usage: "use a precise IP instead of a generated one (optional)"},
			},
		}, {
			Name:   "endpoint-delete",
			Action: app.EndpointDelete,
			Flags: []cli.Flag{
				cli.StringFlag{Name: "endpoint,e", Usage: "ID of the endpoint to delete"},
			},
		},
	}
	err := app.cli.Run(os.Args)
	if err != nil {
		fmt.Println("An error occured:", err)
	}
}

func (a *App) sandClient(c *cli.Context) (sand.Client, error) {
	opts := []sand.Opt{
		sand.WithURL(a.config.ApiURL),
		sand.WithTimeout(a.config.Timeout),
	}
	if a.config.CaFile != "" && a.config.CertFile != "" && a.config.KeyFile != "" {
		config, err := sand.TlsConfig(
			a.config.CaFile, a.config.CertFile, a.config.KeyFile,
		)
		if err != nil {
			return nil, err
		}
		opts = append(opts, sand.WithTlsConfig(config))
	}
	return sand.NewClient(opts...), nil
}
