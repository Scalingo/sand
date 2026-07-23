package main

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/Scalingo/go-utils/errors/v3"
	"github.com/Scalingo/sand/api/params"
	"github.com/Scalingo/sand/api/types"
)

type CliEndpoint types.Endpoint

func (e CliEndpoint) String() string {
	if e.Active {
		return fmt.Sprintf("* [ACTIVE]  ID=%s networkID=%s hostname=%s IP=%s NS=%s", e.ID, e.NetworkID, e.Hostname, e.TargetVethIP, e.TargetNetnsPath)
	}
	return fmt.Sprintf("* [PASSIVE] ID=%s networkID=%s hostname=%s IP=%s", e.ID, e.NetworkID, e.Hostname, e.TargetVethIP)
}

func (a *App) EndpointCreate(ctx context.Context, c *cli.Command) error {
	client, err := a.sandClient(ctx, c)
	if err != nil {
		return errors.Wrap(ctx, err, "create sand client")
	}
	endpoint, err := client.EndpointCreate(ctx, params.EndpointCreate{
		NetworkID:   c.String("network"),
		IPv4Address: c.String("ip"),
		Activate:    true,
		ActivateParams: params.EndpointActivate{
			NSHandlePath: c.String("ns"),
		},
	})
	if err != nil {
		return errors.Wrap(ctx, err, "create endpoint")
	}
	fmt.Println("New endpoint created:")
	fmt.Println(CliEndpoint(endpoint))
	return nil
}

func (a *App) EndpointsList(ctx context.Context, c *cli.Command) error {
	client, err := a.sandClient(ctx, c)
	if err != nil {
		return errors.Wrap(ctx, err, "create sand client")
	}

	var hostname string
	if c.String("hostname") == "all" {
		hostname = ""
	} else {
		hostname = c.String("hostname")
	}

	endpoints, err := client.EndpointsList(ctx, params.EndpointsList{
		NetworkID: c.String("network"),
		Hostname:  hostname,
	})
	if err != nil {
		return errors.Wrap(ctx, err, "list endpoints")
	}
	fmt.Println("List of endpoints:")
	for _, endpoint := range endpoints {
		fmt.Println(CliEndpoint(endpoint))
	}
	return nil
}

func (a *App) EndpointDelete(ctx context.Context, c *cli.Command) error {
	client, err := a.sandClient(ctx, c)
	if err != nil {
		return errors.Wrap(ctx, err, "create sand client")
	}

	err = client.EndpointDelete(ctx, c.String("endpoint"))
	if err != nil {
		return errors.Wrap(ctx, err, "delete endpoint")
	}

	fmt.Printf("Endpoint '%s' deleted.\n", c.String("endpoint"))
	return nil
}
