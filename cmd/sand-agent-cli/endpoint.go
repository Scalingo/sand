package main

import (
	"context"
	"fmt"

	"github.com/Scalingo/sand/api/params"
	"github.com/Scalingo/sand/api/types"
	"github.com/urfave/cli"
)

type CliEndpoint types.Endpoint

func (e CliEndpoint) String() string {
	if e.Active {
		return fmt.Sprintf("* [ACTIVE]  ID=%s networkID=%s hostname=%s IP=%s NS=%s", e.ID, e.NetworkID, e.Hostname, e.TargetVethIP, e.TargetNetnsPath)
	}
	return fmt.Sprintf("* [PASSIVE] ID=%s networkID=%s hostname=%s IP=%s", e.ID, e.NetworkID, e.Hostname, e.TargetVethIP)
}

func (a *App) EndpointCreate(c *cli.Context) error {
	client, err := a.sandClient(c)
	if err != nil {
		return err
	}
	endpoint, err := client.EndpointCreate(context.Background(), params.EndpointCreate{
		NetworkID:   c.String("network"),
		IPv4Address: c.String("ip"),
		Activate:    true,
		ActivateParams: params.EndpointActivate{
			NSHandlePath: c.String("ns"),
		},
	})
	if err != nil {
		return err
	}
	fmt.Println("New endpoint created:")
	fmt.Println(CliEndpoint(endpoint))
	return nil
}

func (a *App) EndpointsList(c *cli.Context) error {
	client, err := a.sandClient(c)
	if err != nil {
		return err
	}

	var hostname string
	if c.String("hostname") == "all" {
		hostname = ""
	} else {
		hostname = c.String("hostname")
	}

	endpoints, err := client.EndpointsList(context.Background(), params.EndpointsList{
		NetworkID: c.String("network"),
		Hostname:  hostname,
	})
	if err != nil {
		return err
	}
	fmt.Println("List of endpoints:")
	for _, endpoint := range endpoints {
		fmt.Println(CliEndpoint(endpoint))
	}
	return nil
}

func (a *App) EndpointDelete(c *cli.Context) error {
	client, err := a.sandClient(c)
	if err != nil {
		return err
	}

	err = client.EndpointDelete(context.Background(), c.String("endpoint"))
	if err != nil {
		return err
	}

	fmt.Printf("Endpoint '%s' deleted.\n", c.String("endpoint"))
	return nil
}
