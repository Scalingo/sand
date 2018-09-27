package main

import (
	"context"
	"fmt"

	"github.com/Scalingo/sand/api/params"
	"github.com/urfave/cli"
)

func (a *App) NetworkCreate(c *cli.Context) error {
	client, err := a.sandClient(c)
	if err != nil {
		return err
	}
	network, err := client.NetworkCreate(context.Background(), params.NetworkCreate{
		Name:    c.String("name"),
		IPRange: c.String("ip-range"),
	})
	if err != nil {
		return err
	}
	fmt.Println("New network created:")
	fmt.Printf("* id=%s name=%s type=%s ip-range=%s, vni=%d\n", network.ID, network.Name, network.Type, network.IPRange, network.VxLANVNI)
	return nil
}

func (a *App) NetworkShow(c *cli.Context) error {
	client, err := a.sandClient(c)
	if err != nil {
		return err
	}

	network, err := client.NetworkShow(context.Background(), c.String("network"))
	if err != nil {
		return err
	}

	fmt.Printf("[%s] %s (%s VNI: %d)\n", network.ID, network.Name, network.Type, network.VxLANVNI)
	return nil
}

func (a *App) NetworksList(c *cli.Context) error {
	client, err := a.sandClient(c)
	if err != nil {
		return err
	}
	networks, err := client.NetworksList(context.Background())
	if err != nil {
		return err
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

func (a *App) NetworkDelete(c *cli.Context) error {
	client, err := a.sandClient(c)
	if err != nil {
		return err
	}

	err = client.NetworkDelete(context.Background(), c.String("network"))
	if err != nil {
		return err
	}

	fmt.Printf("Network %s has been deleted\n", c.String("network"))
	return nil
}
