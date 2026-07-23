package main

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/Scalingo/go-utils/errors/v3"
	"github.com/Scalingo/sand/api/params"
)

func (a *App) NetworkCreate(ctx context.Context, c *cli.Command) error {
	client, err := a.sandClient(ctx, c)
	if err != nil {
		return errors.Wrap(ctx, err, "create sand client")
	}
	network, err := client.NetworkCreate(ctx, params.NetworkCreate{
		Name:    c.String("name"),
		IPRange: c.String("ip-range"),
	})
	if err != nil {
		return errors.Wrap(ctx, err, "create network")
	}
	fmt.Println("New network created:")
	fmt.Printf("* id=%s name=%s type=%s ip-range=%s, vni=%d\n", network.ID, network.Name, network.Type, network.IPRange, network.VxLANVNI)
	return nil
}

func (a *App) NetworkShow(ctx context.Context, c *cli.Command) error {
	client, err := a.sandClient(ctx, c)
	if err != nil {
		return errors.Wrap(ctx, err, "create sand client")
	}

	network, err := client.NetworkShow(ctx, c.String("network"))
	if err != nil {
		return errors.Wrap(ctx, err, "show network")
	}

	fmt.Printf("[%s] %s (%s VNI: %d)\n", network.ID, network.Name, network.Type, network.VxLANVNI)
	return nil
}

func (a *App) NetworksList(ctx context.Context, c *cli.Command) error {
	client, err := a.sandClient(ctx, c)
	if err != nil {
		return errors.Wrap(ctx, err, "create sand client")
	}
	networks, err := client.NetworksList(ctx)
	if err != nil {
		return errors.Wrap(ctx, err, "list networks")
	}
	if len(networks) == 0 {
		fmt.Println("No existing network")
		return nil
	}
	fmt.Println("List of networks:")
	for _, network := range networks {
		fmt.Printf("* [%s] %s (%s VNI: %d)\n", network.ID, network.Name, network.Type, network.VxLANVNI)
	}
	return nil
}

func (a *App) NetworkDelete(ctx context.Context, c *cli.Command) error {
	client, err := a.sandClient(ctx, c)
	if err != nil {
		return errors.Wrap(ctx, err, "create sand client")
	}

	if c.Bool("recursive") {
		endpoints, err := client.EndpointsList(ctx, params.EndpointsList{
			NetworkID: c.String("network"),
		})
		if err != nil {
			return errors.Wrap(ctx, err, "list endpoints for network")
		}
		for _, endpoint := range endpoints {
			err := client.EndpointDelete(ctx, endpoint.ID)
			if err != nil {
				return errors.Wrap(ctx, err, "delete endpoint")
			}
			fmt.Printf("Endpoint '%s' deleted.\n", endpoint.ID)
		}
	}

	err = client.NetworkDelete(ctx, c.String("network"))
	if err != nil {
		return errors.Wrap(ctx, err, "delete network")
	}

	fmt.Printf("Network %s has been deleted\n", c.String("network"))
	return nil
}
