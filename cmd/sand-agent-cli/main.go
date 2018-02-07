package main

import (
	"fmt"
	"os"

	"github.com/Scalingo/sand/client/sand"
	"github.com/urfave/cli"
)

type App struct {
	config Config
	cli    *cli.App
}

type Config struct {
	ApiURL   string
	CertFile string
	KeyFile  string
	CaFile   string
}

func main() {
	app := &App{
		config: Config{},
		cli:    cli.NewApp(),
	}
	app.cli.Flags = []cli.Flag{
		cli.StringFlag{Name: "api-url", Value: "http://localhost:9999", Usage: "when requests will be sent", EnvVar: "SAND_API_URL"},
		cli.StringFlag{Name: "cert-file", Usage: "identify HTTPS client using this SSL certificate file", EnvVar: "SAND_CERT_FILE"},
		cli.StringFlag{Name: "key-file", Usage: "identify HTTPS client using this SSL key file", EnvVar: "SAND_KEY_FILE"},
		cli.StringFlag{Name: "ca-file", Usage: "verify certificates of HTTPS-enabled servers using this CA bundle", EnvVar: "SAND_CA_FILE"},
	}
	app.cli.Before = func(c *cli.Context) error {
		app.config.ApiURL = c.GlobalString("api-url")
		app.config.CertFile = c.GlobalString("cert-file")
		app.config.KeyFile = c.GlobalString("key-file")
		app.config.CaFile = c.GlobalString("ca-file")
		return nil
	}
	app.cli.Commands = cli.Commands{
		{
			Name:   "network-list",
			Action: app.NetworksList,
		}, {
			Name:   "network-create",
			Action: app.NetworkCreate,
			Flags: []cli.Flag{
				cli.StringFlag{Name: "name", Usage: "name of the network to create"},
			},
		}, {
			Name:   "network-delete",
			Action: app.NetworkDelete,
			Flags: []cli.Flag{
				cli.StringFlag{Name: "network,n", Usage: "ID of the network to delete"},
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
	return sand.NewClient(opts...)
}
