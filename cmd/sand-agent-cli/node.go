package main

import (
	"context"
	"fmt"

	"github.com/urfave/cli"
)

func (a *App) NodeEnsureNetworkEndpoints(c *cli.Context) error {
	client, err := a.sandClient(c)
	if err != nil {
		return err
	}

	err = client.NodeEnsureNetworkEndpoints(context.Background())
	if err != nil {
		return err
	}

	fmt.Println("Network endpoints ensure triggered")
	return nil
}
